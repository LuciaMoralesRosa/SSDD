package main

/*
* AUTOR: Lucia Morales Rosa (816906) y Lizer Bernad Ferrando (779035)
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: octubre de 2024
* FICHERO: main.go
* DESCRIPCIÓN: Contiene la implementacion del escritor para el algoritmo de
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
	com.Depuracion("Escritor - Lanzando al escritor")
	if len(os.Args) < 2 {
		fmt.Println("Numero de argumentos incorrecto: main.go <numProceso>")
		os.Exit(1)
	}
	id, err := strconv.Atoi(os.Args[1])
	com.CheckError(err)
	com.Depuracion("Escritor - Se han obtenido los argumentos")

	rand.Seed(time.Now().UnixNano())
	valorAleatorio := com.ValorAleatorio()

	// Esperar a todos los procesos
	com.Depuracion("Escritor - Esperando en la barrera")
	com.Barrera(fichero, id)
	com.Depuracion("Escritor - He salido de la barrera")

	// Inicializacion de ra
	ra := ra.New(id, fichero, "Escribir")

	time.Sleep(4 * time.Second)

	// Escribir
	for i := 1; i < maxPeticiones; i++ {
		ra.PreProtocol()
		com.Depuracion("Escritor - Estoy en SC")
		textoAEscribir := "Soy " + strconv.Itoa(id) + ", escritura numero " +
			strconv.Itoa(i) + ".\n"
		ra.Fichero.Escribir(textoAEscribir)
		fmt.Println("Escritor - He escrito")
		ra.EnviarActualizar(textoAEscribir)
		fmt.Println("Escritor - He enviado Actualizar")
		ra.PostProtocol()
		fmt.Println("Escritor - He salido de SC")
		time.Sleep(time.Duration(valorAleatorio) * time.Millisecond)
	}

	fmt.Println("Escritor - Final ejecucion, voy a entrar al for")
	com.Depuracion("Escritor - Final ejecucion, voy a entrar al for")
	for {
	}

}
