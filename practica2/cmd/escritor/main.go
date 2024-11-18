package main

import (
	"practica2/com"
	"practica2/ra"
	"strconv"
	"sync"
)

const endpointBarrera = "192.168.3.1:31110"
const puerto = ":31111"
const segundos = 5
const maxPeticiones = 100

func main() {
	com.LimpiarTodosLosPuertos()
	id := com.ObtenerArgumentos()

	// Inicializacion de ra
	ra := ra.New(id, "usuarios.txt", "escribir")

	var wg sync.WaitGroup
	wg.Add(1)
	go com.Esperar(&wg, puerto)

	com.EstoyListo(id, endpointBarrera)
	wg.Wait() // Esperar a que todos esten listos

	go com.Final(5)

	// Escribir
	for i := 1; i < maxPeticiones; i++ {
		ra.PreProtocol()
		textoAEscribir := "Soy " + strconv.Itoa(id) + " y esta es la vez " +
			"numero " + strconv.Itoa(i) + " que escribo"
		ra.Fichero.Escribir(textoAEscribir)
		ra.EnviarActualizar(textoAEscribir)
		ra.PostProtocol()
	}

	for {
	}

}
