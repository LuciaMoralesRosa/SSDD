/*
* AUTOR: Rafael Tolosana Calasanz y Unai Arronategui
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: septiembre de 2022
* FICHERO: server-draft.go
* DESCRIPCIÓN: Contiene la funcionalidad de un servidor concurrente con un pool 
* 			   de goroutines para atender las peticiones
 */
package main

import (
	"encoding/gob"
	"log"
	"os"
	"net"
	"practica1/com"
)

// Tamaño del pool de goroutines
const poolSize = 4

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

// processRequest maneja el procesamiento de una solicitud de un cliente.
// PRE: conn es una conexión TCP establecida.
// POST: Cierra la conexión una vez procesada la solicitud, decodifica la
// solicitud, encuentra los primos en el intervalo solicitado y responde al
// cliente con los resultados.
func processRequest(conn net.Conn){
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

// worker es una función que actúa como un trabajador.
// PRE: `tarea` es un canal de conexiones TCP.
// POST: Procesa cada conexión llamando a `processRequest` para manejar la
// solicitud y cerrar la conexión. El trabajador estará en un bucle esperando
// nuevas tareas (conexiones) en el canal.
func worker(id int, tarea <- chan net.Conn){
	for conn := range tarea {
		log.Println("Trabajador %d - Procesando una nueva conexion\n", id)
		processRequest(conn)
	}
}

// createPool crea un conjunto de trabajadores que procesarán las tareas.
// PRE: poolSize es el número de trabajadores a crear, tarea es un canal donde
// se recibirán conexiones TCP.
// POST: Lanza `poolSize` goroutines que ejecutan la función `worker` para
// procesar conexiones recibidas en el canal `tarea`.
func createPool(poolSize int, tarea chan net.Conn){
	for i := 1; i <= poolSize; i++ {
		go worker(i, tarea)
	}
}

func main() {
	args := os.Args
	if len(args) != 2 {
		log.Println("Error: endpoint missing: go run server.go ip:port")
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
		com.CheckError(err)

		// Enviar la conexion al canal
		tarea <- conn
	}
}
