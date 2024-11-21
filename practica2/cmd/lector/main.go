package main

import (
	"fmt"
	"math/rand"
	"os"
	"practica2/com"
	"practica2/ra"
	"strconv"
	"time"
)

// const endpointBarrera = "192.168.3.1:31110"
// const puerto = ":31112"
// const segundos = 5
const maxPeticiones = 8
const fichero = "usuarios.txt"

func main() {
	com.Depuracion("Lector - Lanzando al lector")
	if len(os.Args) < 2 {
		fmt.Println("Numero de argumentos incorrecto")
		os.Exit(1)
	}

	id, err := strconv.Atoi(os.Args[1])
	com.CheckError(err)
	com.Depuracion("Lector - Se han obtenido los argumentos")

	rand.Seed(time.Now().UnixNano())
	valorAleatorio := com.ValorAleatorio()

	// Esperar a todos los procesos
	com.Depuracion("Esperando en la barrera")
	com.Barrera(fichero, id)
	com.Depuracion("He salido de la barrera")

	// Inicializacion de ra
	fmt.Println("Depurando: Estoy enviando el valor id " + strconv.Itoa(id))
	ra := ra.New(id, fichero, "Leer")

	time.Sleep(4 * time.Second)

	// Leer
	for i := 1; i < maxPeticiones; i++ {
		ra.PreProtocol()
		fmt.Println("Depurando: Lector en SC")
		leido := ra.Fichero.Leer()
		fmt.Println("Depurando - Leer: " + leido)
		ra.PostProtocol()
		fmt.Println("Depurando: Lector fuera de SC")
		time.Sleep(time.Duration(valorAleatorio) * time.Millisecond)
	}

	time.Sleep(5 * time.Second)
	fmt.Println("Depurando: voy a entrar al for")
	for {
	}
}
