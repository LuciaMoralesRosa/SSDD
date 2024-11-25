package com

/*
* AUTOR: Rafael Tolosana Calasanz y Unai Arronategui
		 Lucia Morales Rosa (816906) y Lizer Bernad Ferrando (779035)
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: octubre de 2024
* FICHERO: barrier.go
* DESCRIPCIÓN: contiene una barrera basada en la entregada para la practica 1
* 				con pequeñas modificaciones
*/

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// readEndpoints lee una lista de endpoints de un fichero, donde cada linea
// representa un endpoint.
//
// Parametros:
// - filename: Ruta hasta el fichero
//
// Return:
// - Un slice con los endpoints leidos.
// - Error si el fichero no puede ser leido.
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

// Maneja una conexión TCP entrante para la sincronización de la barrera.
// Registra el mensaje recibido en un mapa compartido y verifica si se cumple la
// condición de la barrera.
func handleConnection(conn net.Conn, barrierChan chan<- bool,
	received *map[string]bool, mu *sync.Mutex, n int) {
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

// Configura una sincronización distribuida de barrera utilizando comunicación
// TCP. Cada proceso se comunica con los demás para asegurarse de que todos han
// alcanzado la barrera antes de continuar.
func Barrera(endpointsFile string, lineNumber int) {
	if lineNumber < 1 {
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
						time.Sleep(1 * time.Second)
						continue
					}
					_, err = conn.Write([]byte(strconv.Itoa(lineNumber)))
					if err != nil {
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

	// Close
	fmt.Println("All processes have reached the barrier. Closing...")
	listener.Close()
}
