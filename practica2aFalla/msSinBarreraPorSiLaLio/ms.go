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
	fmt.Println("Estoy ANTES de la goroutine de ms")

	go func() {
		// Creacion del Listener TCP que vincula el proceso actual a su IP:Puerto
		// Escucha conexiones entrantes desde otros procesos del sistema
		fmt.Println("Depurando ms.New: Estoy en la goroutine de ms")
		fmt.Printf("Depurando ms.New: Valor de ms.me: %d", ms.me)
		listener, err := net.Listen("tcp", ms.peers[ms.me-1])
		checkError(err)
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
