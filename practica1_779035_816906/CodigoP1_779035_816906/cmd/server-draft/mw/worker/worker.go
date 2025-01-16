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


/*
Para ejecutar:
 - Cliente:
 go run main.go 192.168.3.2:31112

 - Master:
 ./encenderMaquinas.sh
  go run master.go maquinas.txt

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

func processRequest(conn net.Conn) {
	defer conn.Close()

	var peticion com.Request
	decoder := gob.NewDecoder(conn)
	err := decoder.Decode(&peticion)
	com.CheckError(err)

	primos := findPrimes(peticion.Interval)

	respuesta := com.Reply{Id: peticion.Id, Primes: primos}
	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(&respuesta)
	com.CheckError(err)
}

func main() {
	args := os.Args
	if len(args) != 2 {
		log.Println("Error: go run worker.go ip:port")
		os.Exit(1)
	}

	endpoint := args[1]

	listener, err := net.Listen("tcp", endpoint)
	com.CheckError(err)
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		com.CheckError(err)

		go processRequest(conn)
	}
}
