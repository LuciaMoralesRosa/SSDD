/*
* AUTOR: Lucia Morales Rosa (816906) y Lizer Bernad Ferrando (779035)
* Modificacion del fichero "defintions.go" proporcionado en la practica 1 de
* 		Rafael Tolosana y Unai Arronategui
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: octubre de 2024
* FICHERO: definitions.go
* DESCRIPCIÓN: contiene las definiciones funciones necesarias para la práctica 2
* y su correcto despliegue
 */
package com

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const puertoInicial = 31110
const puertoFinal = 31119

type MensajeBarrera struct {
	Id    int
	Listo bool
}

func CheckError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func LimpiarTodosLosPuertos() {
	for i := puertoInicial; i <= puertoFinal; i++ {
		puerto := ":" + strconv.Itoa(i)
		cmd := exec.Command("lsof", "-i", puerto)
		salida, err := cmd.CombinedOutput()
		CheckError(err)

		lineas := strings.Split(string(salida), "\n")
		if len(lineas) > 1 {
			campos := strings.Fields(lineas[1])
			if len(campos) > 1 {
				pid := campos[1]

				matar := exec.Command("kill", "-9", pid)
				err = matar.Run()
				CheckError(err)
				fmt.Println("Se ha matado al proceso ", pid, " que usaba el puerto ", puerto)
			}
			fmt.Println("No se ha encontrado ningun proceso usando el puerto ", puerto)
		}
		fmt.Println("No se ha encontrado ningun proceso usando el puerto ", puerto)
	}
}

func ObtenerArgumentos() int {
	if len(os.Args) < 2 {
		fmt.Println("Numero de parametros incorrecto")
		os.Exit(1)
	}

	id, err := strconv.Atoi(os.Args[1])
	CheckError(err)

	return id
}

func EstoyListo(id int, endpoint string) {
	conn, err := net.Dial("tcp", endpoint)
	CheckError(err)
	defer conn.Close()

	mensaje := MensajeBarrera{
		Id:    id,
		Listo: true,
	}
	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(mensaje)
	CheckError(err)
}

func Esperar(wg *sync.WaitGroup, endpoint string) {
	defer wg.Done()
	listener, err := net.Listen("tcp", endpoint)
	CheckError(err)
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		CheckError(err)
		defer conn.Close()

		var mensaje MensajeBarrera
		decoder := gob.NewDecoder(conn)
		err = decoder.Decode(&mensaje)
		CheckError(err)

		if !mensaje.Listo {
			break
		}
	}
}

func Final(segundos time.Duration) {
	temporizador := time.NewTimer(segundos * time.Second)
	<-temporizador.C
	os.Exit(0)
}
