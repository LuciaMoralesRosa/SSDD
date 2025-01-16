/*
* AUTOR: Rafael Tolosana Calasanz y Unai Arronategui
* 		 Lizer Bernad (779035) y Lucia Morales (816906)
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: septiembre de 2024
* FICHERO: server_secuencial.go
* DESCRIPCIÓN: contiene la funcionalidad esencial de un servidor secuencial que
* 			   atiende una unica peticion a la vez
 */
package main

import (
	"encoding/gob"
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
	// Para asegurar que la conexión se cierre al terminar la goroutine
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
}

func main() {
	args := os.Args
	if len(args) != 2 {
		log.Println("Error: endpoint missing: go run server_secuencial.go ip:port")
		os.Exit(1)
	}
	endpoint := args[1]

	// Creacion del listener con la direccion proporcionada
	listener, err := net.Listen("tcp", endpoint)
	com.CheckError(err)

	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	log.Println("***** Listening for new connection in endpoint ", endpoint)

	for {
		// Aceptar nuevas conexiones
		conn, err := listener.Accept()

		// Para que el servidor no se cierre de forma abrupta y pueda seguir
		// aceptando peticiones
		if err != nil {
			log.Println("Error accepting connection: ", err)
			continue
		}
		com.CheckError(err)

		processRequest(conn)
	}
}
