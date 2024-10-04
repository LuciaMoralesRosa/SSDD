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
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"practica1/com"
	"strconv"
	"strings"
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

// go run server_MW_Master.go 192.168.3.2:8080 a816906 practica1/cmd/server-draft/serv-worker server_MW_Worker.go puerto
func main() {
	args := os.Args
	if len(args) != 6 {
		log.Println("Error: endpoint missing: go run serv_MW_Master.go ip:port usuario ruta fichero puerto")
		os.Exit(1)
	}
	endpoint := args[1]

	usuario := args[2] // a816906
	ruta := args[3]    // cmd/server-draft/serv-worker
	fichero := args[4] // server_MW_Worker.go
	puerto, err := strconv.Atoi(args[5])
	if err != nil {
		log.Println("Error convirtuendo el puerto a entero:", err)
	}

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
		workers := []string{fmt.Sprintf("192.168.3.3:%d", puerto),
			fmt.Sprintf("192.168.3.3:%d", puerto+1)}

		// Ruta del fichero a ejecutar

		// Ejecución remota de los Workers
		for i, worker := range workers {

			// Creacion de comando con parametros
			workerSinPuerto := strings.Split(worker, ":")[0]
			comando := fmt.Sprintf("ssh %s@%s 'cd /misc/alumnos/sd/sd2425/%s/%s; go run %s %s:%d'", usuario, workerSinPuerto, usuario, ruta, fichero, workerSinPuerto, puerto+i)

			// ssh a816906@192.168.3.3 'cd /misc/alumnos/sd/sd2425/a816906/practica1/cmd/server-draft/serv-worker; go run server_MW_Worker.go 192.168.3.3:8081'

			//Creacion comando cutre
			cmd := exec.Command(comando)
			output, err := cmd.CombinedOutput() // Captura la salida y errores
			if err != nil {
				fmt.Printf("Error ejecutando en %s: %s\n", worker, err)
			}
			fmt.Printf("Salida de %s - (output) ->\n%s\n", worker, output)

		}

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
