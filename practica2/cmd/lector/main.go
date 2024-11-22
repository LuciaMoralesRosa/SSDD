package main

/*
* AUTOR: Lucia Morales Rosa (816906) y Lizer Bernad Ferrando (779035)
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: octubre de 2024
* FICHERO: main.go
* DESCRIPCIÓN: Contiene la implementacion del lector para el algoritmo de
	Ricart-Agrawala Generalizado
*/

import (
	"fmt"
	"math/rand"
	"os"
	"practica2/com"
	"practica2/ra"
	"strconv"
	"time"
)

const maxPeticiones = 4
const fichero = "usuarios.txt"

func main() {
	com.Depuracion("Lector - Lanzando al lector", -1)
	if len(os.Args) < 2 {
		fmt.Println("Numero de argumentos incorrecto: main.go <numProceso>")
		os.Exit(1)
	}

	id, err := strconv.Atoi(os.Args[1])
	com.CheckError(err)
	com.Depuracion("Lector - Se han obtenido los argumentos", -1)

	rand.Seed(time.Now().UnixNano())
	valorAleatorio := com.ValorAleatorio()

	// Esperar a todos los procesos
	com.Depuracion("Lector - Esperando en la barrera", -1)
	com.Barrera(fichero, id)
	com.Depuracion("Lector - He salido de la barrera", -1)

	// Inicializacion de ra
	ra := ra.New(id, fichero, "Leer")

	time.Sleep(4 * time.Second)

	// Leer
	for i := 1; i < maxPeticiones; i++ {
		ra.PreProtocol()
		com.Depuracion("Lector - Estoy en SC", -1)
		leido := ra.Fichero.Leer()
		com.Depuracion("Lector - He leido el texto "+leido, -1)
		ra.PostProtocol()
		com.Depuracion("Lector - He salido de la seccion critica", -1)
		time.Sleep(time.Duration(valorAleatorio) * time.Millisecond)
	}

	time.Sleep(5 * time.Second)
	fmt.Println("Lector - Final ejecucion, voy a entrar al for")
	com.Depuracion("Lector - Final ejecucion, voy a entrar al for", -1)
	for {
	}
}
