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

var primeraVez = true

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
	Depuracion("Definitions - LimpiarTodosLosPuertos: Inicio")
	for i := puertoInicial; i <= puertoFinal; i++ {
		puerto := ":" + strconv.Itoa(i)
		cmd := exec.Command("lsof", "-i", puerto)
		salida, err := cmd.CombinedOutput()
		CheckError(err)
		Depuracion("Definitions - LimpiarTodosLosPuertos: He obtenido la salida " + string(salida))

		lineas := strings.Split(string(salida), "\n")
		if len(lineas) > 1 {
			campos := strings.Fields(lineas[1])
			if len(campos) > 1 {
				pid := campos[1]

				matar := exec.Command("kill", "-9", pid)
				err = matar.Run()
				CheckError(err)

				Depuracion("Definitions - LimpiarTodosLosPuertos: Se ha matado al proceso " + pid + " que usaba el puerto " + puerto)
				fmt.Println("Se ha matado al proceso ", pid, " que usaba el puerto ", puerto)
			}
			Depuracion("Definitions - LimpiarTodosLosPuertos: No se ha encontrado ningun proceso usando el puerto " + puerto)
			fmt.Println("No se ha encontrado ningun proceso usando el puerto ", puerto)
		}
		Depuracion("Definitions - LimpiarTodosLosPuertos: No se ha encontrado ningun proceso usando el puerto " + puerto)
		fmt.Println("No se ha encontrado ningun proceso usando el puerto ", puerto)
	}
}

func ObtenerArgumentos() int {
	if len(os.Args) < 2 {
		fmt.Println("Numero de parametros incorrecto")
		Depuracion("Definitions - ObtenerArgumentos: Numero de argumentos invalido")
		os.Exit(1)
	}

	id, err := strconv.Atoi(os.Args[1])
	CheckError(err)

	return id
}

func EstoyListo(id int, endpoint string) {
	Depuracion("Definitions - EstoyListo: inicio")
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

	Depuracion("Definitions - EstoyListo: final")
}

func Esperar(wg *sync.WaitGroup, endpoint string) {
	Depuracion("Definitions - Esperar: inicio")

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
			Depuracion("Definitions - EstoyListo: nodo trabajando")
			break
		}
	}
	Depuracion("Definitions - EstoyListo: final")
}

func Final(segundos time.Duration) {
	Depuracion("Definitions - Final: enviando final")
	temporizador := time.NewTimer(segundos * time.Second)
	<-temporizador.C
	os.Exit(0)
}

func Depuracion(textoDepuracion string) {
	if primeraVez {
		// Comprobar si el archivo existe
		if _, err := os.Stat("Depuracion.txt"); os.IsNotExist(err) {
			fmt.Println("El archivo no existe.")
		} else {
			// Intentar borrar el archivo
			err := os.Remove("Depuracion.txt")
			CheckError(err)
		}
		primeraVez = false
	}

	// Abrir S(o crear) el archivo en modo append, con permisos de escritura y lectura
	fichero, err := os.OpenFile("Depuracion.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	CheckError(err)

	// Asegurarse de cerrar el archivo al salir de la función
	defer fichero.Close()

	// Escribir el texto en el archivo
	_, err = fichero.WriteString(textoDepuracion + "\n")
	CheckError(err)
}
