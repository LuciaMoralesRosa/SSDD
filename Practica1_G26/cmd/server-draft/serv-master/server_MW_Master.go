/*
  - AUTOR: Rafael Tolosana Calasanz y Unai Arronategui
  - Lizer Bernad (779035) y Lucia Morales (816906)
  - ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
  - Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
  - FECHA: septiembre de 2024
  - FICHERO: server_MW_Master.go
  - DESCRIPCIÓN: contiene la funcionalidad de un Master en la arquitectura
    Master-Worker
*/
package main

import (
	"encoding/gob"
	"log"
	"net"
	"os"
	"practica1/com"
)

// enviarTarea establece una conexión con un trabajador remoto y le envía un
// intervalo para que procese.
// PRE: `ip` es la dirección IP:puerto del trabajador, `interval` es el rango de
// números a procesar, e `id` es el identificador de la solicitud.
// POST: Devuelve una lista de números primos encontrados en el intervalo o un
// error en caso de fallo.
func enviarTarea(ip string, interval com.TPInterval, id int) ([]int, error) {
	// Establecer una conexión TCP con la maquina remota
	conn, err := net.Dial("tcp", ip)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Enviar peticion al trabajador
	encoder := gob.NewEncoder(conn)
	request := com.Request{Id: id, Interval: interval}
	err = encoder.Encode(request)
	if err != nil {
		return nil, err
	}

	// Obtenemos la respuesta del trabajador
	var reply com.Reply
	decoder := gob.NewDecoder(conn)
	err = decoder.Decode(&reply) //  receive reply
	if err != nil {
		return nil, err
	}

	return reply.Primes, nil
}

func main() {
	args := os.Args
	if len(args) != 2 {
		log.Println("Error: endpoint missing: go run serv_MW_Master.go ip:port")
		os.Exit(1)
	}
	endpoint := args[1]

	// Creacion del listener con la direccion proporcionada
	listener, err := net.Listen("tcp", endpoint)
	com.CheckError(err)
	defer listener.Close()

	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	log.Println("***** Listening for new connection in endpoint ", endpoint)

	for {
		// Aceptar nuevas conexiones
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		defer conn.Close()

		// Crear una peticion
		var request com.Request
		decoder := gob.NewDecoder(conn)
		err = decoder.Decode(&request)
		com.CheckError(err)

		//Definimos las maquinas trabajadoras
		workers := []string{"192.168.3.3:8081",
			"192.168.3.4:8082"}

		var resultados []int

		// Definimos el tamaño de los intervalos
		tamIntervalo := (request.Interval.Max - request.Interval.Min + 1) /
			len(workers)

		// Repartimos el trabajo entre los trabajadores
		for i, worker := range workers {
			intervaloWorker := com.TPInterval{
				Min: request.Interval.Min + i*tamIntervalo,
				Max: request.Interval.Min + (i+1)*tamIntervalo - 1,
			}

			// Aseguramos que el ultimo trabajador tome el restante
			if i == len(workers)-1 {
				intervaloWorker.Max = request.Interval.Max
			}

			// Envio de tarea al trabajador
			primes, err := enviarTarea(worker, intervaloWorker, request.Id)
			if err != nil {
				log.Println("Error sending task to worker:", worker, err)
				continue
			}

			// Obtencion de los resulrados
			resultados = append(resultados, primes...)
			// Los "..." pasa cada primo como un elemento individual
		}

		// Envio de los resultados al cliente
		reply := com.Reply{Id: request.Id, Primes: resultados}
		encoder := gob.NewEncoder(conn)
		err = encoder.Encode(&reply)
		com.CheckError(err)

	}
}
