/*
  - AUTOR: Rafael Tolosana Calasanz y Unai Arronategui
  - Lizer Bernad (779035) y Lucia Morales (816906)
  - ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
  - Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
  - FECHA: septiembre de 2024
  - FICHERO: server_MW_Worker.go
  - DESCRIPCIÓN: contiene la funcionalidad de un Worker en laarquitectura
    Master-Worker
*/
package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
	"practica1/com"
)

// PRE: verdad = !foundDivisor
// POST: IsPrime devuelve verdad si n es primo y falso en caso contrario
func isPrime(n int) (foundDivisor bool) {
	foundDivisor = false
	for i := 2; (i < n) && !foundDivisor; i++ {
		foundDivisor = (n%i == 0)
	}
	return !foundDivisor
}

// PRE: interval.A < interval.B
// POST: FindPrimes devuelve todos los números primos comprendidos en el
//
//	intervalo [interval.A, interval.B]
func findPrimes(interval com.TPInterval) (primes []int) {
	for i := interval.Min; i <= interval.Max; i++ {
		if isPrime(i) {
			primes = append(primes, i)
		}
	}
	return primes
}

// processRequest maneja la solicitud de un cliente a través de una conexión TCP.
// PRE: conn es una conexión TCP establecida.
// POST: Cierra la conexión una vez procesada la solicitud, decodifica la
// solicitud, encuentra los primos en el intervalo solicitado y responde al
// cliente con los resultados.
func processRequest(conn net.Conn) {
	// Asegura que la conexión se cierre al final de la goroutine
	defer conn.Close()

	// Declaracion de la variable "request" para almacenar la solicitud y
	// configuracion del decodificador para leer los datos de la conexión "conn"
	var request com.Request
	decoder := gob.NewDecoder(conn)
	err := decoder.Decode(&request)
	com.CheckError(err)

	// Procesar la tarea
	primes := findPrimes(request.Interval)

	// Responder con los resultados
	reply := com.Reply{Id: request.Id, Primes: primes}
	encoder := gob.NewEncoder(conn)
	encoder.Encode(&reply)
	com.CheckError(err)
}

func main() {
	args := os.Args
	if len(args) != 2 {
		log.Println("Error: endpoint missing: go run serv_MW_Worker.go ip:port")
		os.Exit(1)
	}
	endpoint := args[1]

	// Creacion del listener con la direccion proporcionada
	listener, err := net.Listen("tcp", endpoint)
	if err != nil {
		fmt.Println("Error starting TCP worker:", err)
		os.Exit(1)
	}
	defer listener.Close()

	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	log.Println("***** Listening for new connection in endpoint from Master ",
		endpoint)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		// Lanzamos una goroutine para la peticion
		go processRequest(conn)
	}
}
