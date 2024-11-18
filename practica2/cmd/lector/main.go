package main

import (
	"fmt"
	"os"
	"practica2/com"
	"practica2/ra"
	"strconv"
	"sync"
)

const endpointBarrera = "192.168.3.1:31110"
const puerto = ":31112"
const segundos = 5
const maxPeticiones = 100

func main() {
	com.Depuracion("Escritor - Lanzando al escritor")
	if len(os.Args) < 2 {
		fmt.Println("Numero de argumentos incorrecto")
		os.Exit(1)
	}
	com.Depuracion("Escritor - Se han limpiado los puertos")

	id, err := strconv.Atoi(os.Args[1])
	com.CheckError(err)
	com.Depuracion("Escritor - Se han obtenido los argumentos")

	// Inicializacion de ra
	ra := ra.New(id, "usuarios.txt", "leer")

	var wg sync.WaitGroup
	wg.Add(1)
	go com.Esperar(&wg, puerto)

	com.EstoyListo(id, endpointBarrera)
	wg.Wait() // Esperar a que todos esten listos

	go com.Final(5)

	// Escribir
	for i := 1; i < maxPeticiones; i++ {
		ra.PreProtocol()
		ra.Fichero.Leer()
		ra.PostProtocol()
	}

	for {

	}
}
