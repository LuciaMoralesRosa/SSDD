package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

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

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run main.go <endpoints_file> <line_number>")
		return
	}

	endpointsFile := os.Args[1]
	lineNumber, err := strconv.Atoi(os.Args[2])
	if err != nil || lineNumber < 1 {
		fmt.Println("Invalid line number")
		return
	}

	// Read endpoints from file
	endpoints, err := readEndpoints(endpointsFile)
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

	fmt.Println("Listening on", localEndpoint)

	// Barrier synchronization
	var mu sync.Mutex
	quitChannel := make(chan bool)
	receivedMap := make(map[string]bool)
	barrierChan := make(chan bool)

	// Start accepting connections
	go func() {
		for {
			select {
			case <-quitChannel:
				fmt.Println("Stopping the listener...")
				break
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
			go func(ep string) {
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
	fmt.Println("Waiting for all the processes to reach the barrier")
	listener.Close()
}
