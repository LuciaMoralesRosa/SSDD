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
	com.Depuracion("Lector - Se han limpiado los puertos")

	id, err := strconv.Atoi(os.Args[1])
	com.CheckError(err)
	com.Depuracion("Lector - Se han obtenido los argumentos")

	rand.Seed(time.Now().UnixNano())
	valorAleatorio := com.ValorAleatorio()

	// Inicializacion de ra
	fmt.Println("Depurando: Estoy enviando el valor id " + strconv.Itoa(id))
	ra := ra.New(id, fichero, "Leer")

	//var wg sync.WaitGroup
	//wg.Add(1)
	//go com.Esperar(&wg, puerto)

	//com.EstoyListo(id, endpointBarrera)
	//wg.Wait() // Esperar a que todos esten listos

	//go com.Final(5)

	time.Sleep(6 * time.Second)

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
	for {
		fmt.Println("Depurando: voy a entrar al for")

	}
}
