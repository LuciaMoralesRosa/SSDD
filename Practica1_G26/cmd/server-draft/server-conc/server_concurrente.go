/**
* AUTOR: Rafael Tolosana Calasanz y Unai Arronategui
* 		 Lizer Bernad (779035) y Lucia Morales (816906)
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: septiembre de 2024
* FICHERO: server-draft.go
* DESCRIPCIÓN: contiene la funcionalidad de un servidor concurrente que crea una
*			   goroutine por peticion
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

// processRequest maneja la conexión con un cliente y procesa su solicitud.
// PRE: `conn` es una conexión TCP válida.
// POST: Decodifica la solicitud, encuentra los números primos y devuelve los
// resultados al cliente.
func processRequest(conn net.Conn) {
	// Para asegurar que la conexión se cierre al terminar la goroutine
	defer conn.Close()

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
		log.Println("Error: endpoint missing: go run serv_conc.go ip:port")
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

		// Lanzamos una goroutine para la peticion
		go processRequest(conn)
	}
}
