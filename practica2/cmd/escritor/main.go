package main

import (
	"practica2/gestor"

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

func todosNodosConectados(lineNumber int) int {
	// Read endpoints from file
	endpoints, err := readEndpoints(ficheroUsuarios)
	if err != nil {
		fmt.Println("Error reading endpoints:", err)
		return
	}

	n := len(endpoints)

	if lineNumber > n {
		fmt.Printf("Line number %d out of range\n", lineNumber)
		return
	}

	// Get the endpoint for this process
	localEndpoint := endpoints[lineNumber-1]
	listener, err := net.Listen("tcp", localEndpoint)
	if err != nil {
		fmt.Println("Error creating listener:", err)
		return
	}



	// Esperar a que todos los procesos se inicialicen
	barrierChan := make(chan bool)
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
		if i + 1 != lineNumber {
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
		else {
			pid := ep
		}
	}

	// Wait for all processes to reach the barrier
	fmt.Println("Waiting for all the processes to reach the barrier")
	<-barrierChan

	// Stop the Listening loop
	close(quitChannel)

	// Wait for the process routines to finish
	wg.Wait()
	
	return pid
}

func escribir(ra *ra.RASharedBD, ficheroEscritura string, textoEscritura string){
	for i := 0; i < 5; i ++ {
		ra.PreProtocol()
		gestor.escribirFichero(ficheroEscritura, textoEscritura)
		ra.escribirTexto(fichero, texto + " " + i)
		ra.PostProtocol()
	}
}

func main() {
	args := os.Args()

	lineNumber, err := strconv.Atoi(os.Args[1])
	if err != nil || lineNumber < 1 {
		fmt.Println("Invalid line number")
		return
	}
	
	ficheroUsuarios := args[2]
	procesoEscritor := true
	ficheroEscritura := args[3]
	textoEscritura := args[4]

	pid := todosNodosConectados(lineNumber)
	ra := ra.New(pid, ficheroUsuarios, procesoEscritor)

	go escribir(ra, ficheroEscritura, textoEscritura)
}
