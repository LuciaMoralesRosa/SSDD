/*
  - AUTOR: Rafael Tolosana Calasanz y Unai Arronategui
  - Lizer Bernad (779035) y Lucia Morales (816906)
  - ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
  - Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
  - FECHA: septiembre de 2024
  - FICHERO: master.go
  - DESCRIPCIÓN: contiene la funcionalidad de un Master en la arquitectura
    Master-Worker con politica Round Robin para la asignacion de trabajadores
*/

package main

import (
	"bufio"
	"encoding/gob"
	"log"
	"net"
	"os"
	"practica1/com"
	"sync"
)

var currentWorkerIndex int = 0
var mu sync.Mutex

func enviarTarea(ip string, interval com.TPInterval, id int) ([]int, error) {
	conn, err := net.Dial("tcp", ip)
	com.CheckError(err)
	defer conn.Close()

	encoder := gob.NewEncoder(conn)
	request := com.Request{Id: id, Interval: interval}
	err = encoder.Encode(request)
	com.CheckError(err)

	var reply com.Reply
	decoder := gob.NewDecoder(conn)
	err = decoder.Decode(&reply)
	com.CheckError(err)

	return reply.Primes, nil
}

func obtenerTrabajadores(fichero string) ([]string, error) {
	file, err := os.Open(fichero)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var trabajadores []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			trabajadores = append(trabajadores, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return trabajadores, nil
}

func main() {
	args := os.Args
	if len(args) != 3 {
		log.Println("Error: go run master.go ip:port maquinas.txt")
		os.Exit(1)
	}

	miMaquina := args[1]

	listener, err := net.Listen("tcp", miMaquina)
	com.CheckError(err)
	defer listener.Close()

	workers, err := obtenerTrabajadores(args[2])
	com.CheckError(err)

	var resultados []int

	for {
		// Aceptacion de las conexiones
		conn, err := listener.Accept()
		com.CheckError(err)

		defer conn.Close()

		// Obtencion de la peticion del cliente
		var peticion com.Request
		decoder := gob.NewDecoder(conn)
		err = decoder.Decode(&peticion)
		com.CheckError(err)

		// Politica Round Robin para seleccionar el worker
		mu.Lock()
		worker := workers[currentWorkerIndex]
		currentWorkerIndex = (currentWorkerIndex + 1) % len(workers)
		mu.Unlock()

		// Enviar la tarea y obtener los resultados
		primos, err := enviarTarea(worker, peticion.Interval, peticion.Id)
		com.CheckError(err)
		resultados := append(resultados, primos...)

		// Enviar resultados al cliente
		respuestaCiente := com.Reply{Id: peticion.Id, Primes: resultados}
		encoder := gob.NewEncoder(conn)
		err = encoder.Encode(&respuestaCiente)
		com.CheckError(err)
	}
}
