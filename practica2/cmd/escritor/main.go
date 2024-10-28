package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	g "practica2/gestor"
	"practica2/ra"
	"strconv"
	"sync"
	"time"
)

// Maneja los errores del sistema, mostrandolo por pantalla y terminando la ejecucion
func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

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

func todosNodosConectados(lineNumber int, ficheroUsuarios string) {
	// Read endpoints from file
	endpoints, err := readEndpoints(ficheroUsuarios)
	if err != nil {
		fmt.Println("Error reading endpoints:", err)
		os.Exit(1)
	}

	n := len(endpoints)

	if lineNumber > n {
		fmt.Printf("Line number %d out of range\n", lineNumber)
		os.Exit(1)
	}

	// Get the endpoint for this process
	localEndpoint := endpoints[lineNumber-1]
	listener, err := net.Listen("tcp", localEndpoint)
	if err != nil {
		fmt.Println("Error creating listener:", err)
		os.Exit(1)
	}

	// Esperar a que todos los procesos se inicialicen
	var mu sync.Mutex
	barrierChan := make(chan bool)
	receivedMap := make(map[string]bool)
	quitChannel := make(chan bool)
	var wg sync.WaitGroup // Waiting Group for the process' routines

	// Start accepting connections
	go func() {
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
				go handleConnection(conn, barrierChan, &receivedMap, &mu, n)
			}
		}
	}()

	// Notify other processes
	for i, ep := range endpoints {
		if i+1 != lineNumber {
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
					_, err = conn.Write([]byte(strconv.Itoa(lineNumber)))
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
}

func escribir(ra *ra.RASharedDB, ficheroEscritura string, textoEscritura string, gestor *g.Gestor) {
	for i := 0; i < 5; i++ {
		ra.PreProtocol()
		gestor.EscribirFichero(ficheroEscritura, textoEscritura)
		ra.EscribirTexto(ficheroEscritura, textoEscritura+" "+strconv.Itoa(i))
		ra.PostProtocol()
	}
}

func main() {
	/*go run main.go 1 ../../ms/users.txt ../ficheroTexto.txt heEscritoEstoooYEY*/

	lineNumber, err := strconv.Atoi(os.Args[1])
	if err != nil || lineNumber < 1 {
		fmt.Println("Invalid line number")
		return
	}

	ficheroUsuarios := os.Args[2]
	procesoEscritor := true
	ficheroEscritura := os.Args[3]
	textoEscritura := os.Args[4]

	todosNodosConectados(lineNumber, ficheroUsuarios)
	gestor := g.New()

	ra := ra.New(lineNumber, ficheroUsuarios, procesoEscritor, gestor)

	go escribir(ra, ficheroEscritura, textoEscritura, &gestor)
}
