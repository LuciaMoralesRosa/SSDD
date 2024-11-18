/*
* AUTOR: Rafael Tolosana Calasanz
* Autores: Lizer Bernad (779035) y Lucia Morales (816906)
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: septiembre de 2021
* FICHERO: ms.go
* DESCRIPCIÓN: Implementación de un sistema de mensajería asíncrono, insipirado en el Modelo Actor
 */
package ms

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

type Message interface{}

type MessageSystem struct {
	mbox  chan Message // Canal de mensajes
	peers []string     // Lista de direcciones IP de los nodos
	done  chan bool    // Controla la finalizacion del sistema
	me    int          // ID del proceso actual
}

const (
	MAXMESSAGES = 10000
)

// Maneja los errores del sistema, mostrandolo por pantalla y terminando la ejecucion
func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

//------------------------CODIGO BARRERA -------------------------------------//

func readEndpoints(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var endpoints []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			endpoints = append(endpoints, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return endpoints, nil
}

func handleConnection(conn net.Conn, barrierChan chan<- bool, received *map[string]bool, mu *sync.Mutex, n int) {
	defer conn.Close()
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading from connection:", err)
		return
	}
	msg := string(buf)
	mu.Lock()
	(*received)[msg] = true
	fmt.Println("Received ", len(*received), " elements")
	if len(*received) == n-1 {
		barrierChan <- true
	}
	mu.Unlock()
}

func todosNodosConectados(me int, peers string, ficheroUsuarios string) net.Listener {
	fmt.Println("Depurando ms.TodosNodosConectados: Estoy en la funcion")
	// Read endpoints from file
	endpoints, err := readEndpoints(ficheroUsuarios)
	if err != nil {
		fmt.Println("Error reading endpoints:", err)
		os.Exit(1)
	}

	n := len(endpoints)

	// Get the endpoint for this process
	listener, err := net.Listen("tcp", peers)
	if err != nil {
		fmt.Println("Error creating listener:", err)
		os.Exit(1)
	}
	fmt.Println("Depurando ms.TodosNodosConectados: Se ha creado el listener del proceso " + peers)

	// Esperar a que todos los procesos se inicialicen
	var mu sync.Mutex
	barrierChan := make(chan bool)
	receivedMap := make(map[string]bool)
	quitChannel := make(chan bool)
	var wg sync.WaitGroup // Waiting Group for the process' routines

	// Start accepting connections
	go func() {
		fmt.Println("Depurando ms.TodosNodosConectados: Se ha lanzado la goroutine para aceptar peticiones")
		stop := false
		for !stop {
			select {
			case <-quitChannel:
				fmt.Println("Stopping the listener...")
				stop = true
			default:
				conn, err := listener.Accept()
				if err != nil {
					fmt.Println("Error accepting connection:", err)
					continue
				}
				fmt.Println("Depurando ms.TodosNodosConectados: Voy a lanzar la goroutine de handleConnection")
				go handleConnection(conn, barrierChan, &receivedMap, &mu, n)
			}
		}
	}()

	// Notify other processes
	for i, ep := range endpoints {
		fmt.Println("Depurando ms.TodosNodosConectados: Estoy notificando a los procesos que estoy en la barrera")
		if i+1 != me {
			wg.Add(1) // Add one routine to the waiting list
			go func(ep string) {
				defer wg.Done() // Substract one routine from the waiting list
				for {
					conn, err := net.Dial("tcp", ep)
					if err != nil {
						fmt.Println("Error connecting to", ep, ":", err)
						time.Sleep(1 * time.Second)
						continue
					}
					_, err = conn.Write([]byte(strconv.Itoa(me)))
					if err != nil {
						fmt.Println("Error sending message:", err)
						conn.Close()
						continue
					}
					conn.Close()
					break
				}
			}(ep)
		}
	}

	// Wait for all processes to reach the barrier
	fmt.Println("Waiting for all the processes to reach the barrier")
	<-barrierChan

	// Stop the Listening loop
	close(quitChannel)

	// Wait for the process routines to finish
	wg.Wait()
	return listener
}

func (ms *MessageSystem) Me() int {
	fmt.Println("Depuracion ms.me: soy " + strconv.Itoa(ms.me))
	return ms.me
}

// Lee un fichero que contiene las direcciones IP de los nodos y las guarda en "lines"
func parsePeers(path string) (lines []string) {
	file, err := os.Open(path)
	checkError(err)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	fmt.Println("Depuracion ms.parsePeers: lines[0]=" + lines[0] + ", lines[1]=" + lines[1])
	return lines
}

// Pre: pid en {1..n}, el conjunto de procesos del SD
// Post: envía el mensaje msg a pid
func (ms *MessageSystem) Send(pid int, msg Message) {
	//Envía un mensaje a un proceso pid con TCP. Se conecta al
	//proceso usando su dirección IP y puerto, y serializa el mensaje con gob
	//para enviarlo a través de la conexión.
	conn, err := net.Dial("tcp", ms.peers[pid-1])
	checkError(err)
	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(&msg)
	//Cierra la conexion
	fmt.Println("Depuracion ms.Send: Realizando Send al proceso " + ms.peers[pid-1])
	conn.Close()
}

// Pre: True
// Post: el mensaje msg de algún Proceso P_j se retira del mailbox y lo devuelve
//
//	Si mailbox vacío, Receive bloquea hasta que llegue algún mensaje
func (ms *MessageSystem) Receive() (msg Message) {
	msg = <-ms.mbox
	return msg
}

//	messageTypes es un slice con tipos de mensajes que los procesos se pueden intercambiar a través de este ms
//
// Hay que registrar un mensaje antes de poder utilizar (enviar o recibir y escribir)
// Notar que se utiliza en la función New
func Register(messageTypes []Message) {
	for _, msgTp := range messageTypes {
		gob.Register(msgTp)
	}
}

// Pre: whoIam es el pid del proceso que inicializa este ms
//
//	usersFile es la ruta a un fichero de texto que en cada línea contiene IP:puerto de cada participante
//	messageTypes es un slice con todos los tipos de mensajes que los procesos se pueden intercambiar a través de este ms
func New(whoIam int, usersFile string, messageTypes []Message) (ms MessageSystem) {
	// Guarda el pid del proceso que ha inicializado el gestor de mensajes
	ms.me = whoIam
	// Obtiene todos los nodos del sistema
	ms.peers = parsePeers(usersFile)
	// Creacion de los canales para el paso de mensajes y finalizacion
	ms.mbox = make(chan Message, MAXMESSAGES)
	ms.done = make(chan bool)
	// Registra el tipo de mensajes (Request, Reply y Escribir)
	Register(messageTypes)

	go func() {
		// Creacion del Listener TCP que vincula el proceso actual a su IP:Puerto
		// Escucha conexiones entrantes desde otros procesos del sistema
		fmt.Println("Depurando ms.New: Estoy en la goroutine de ms")
		//listener, err := net.Listen("tcp", ms.peers[ms.me-1])
		listener := todosNodosConectados(ms.me, ms.peers[ms.me-1], usersFile)
		//checkError(err)
		fmt.Println("Depuracion ms.New despues de hacer el net.Listen")
		fmt.Println("Process listening at " + ms.peers[ms.me-1])
		defer close(ms.mbox)
		for {
			select {
			case <-ms.done:
				// Si llega un mensaje al canal, se termina la ejecucion
				return
			default:
				// Cuando otro proceso se conecta, acepta la conexion
				conn, err := listener.Accept()
				checkError(err)
				// Decodifica el mensaje entrante y se guarda en en msg
				decoder := gob.NewDecoder(conn)
				var msg Message
				err = decoder.Decode(&msg)
				// Se cierra la conexion
				conn.Close()

				// El mensaje se inserta en el canal mailbox
				// (Sera consumido en la funcion Receive())
				ms.mbox <- msg
				fmt.Println("Depuracion ms.New: se ha insertado nuevo mensaje en ms.mbox")
			}
		}
	}()
	fmt.Println("Depuracion ms.New final de la funcion y devolviendo ms")
	return ms
}

// Pre: True
// Post: termina la ejecución de este ms
func (ms *MessageSystem) Stop() {
	fmt.Println("Depuracion ms.Stop: Terminando ejecucion ms.done <- true")
	ms.done <- true
}
