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
// const puerto = ":31111"
// const segundos = 5
const maxPeticiones = 4
const fichero = "usuarios.txt"

func main() {
	com.Depuracion("Escritor - Lanzando al escritor")
	if len(os.Args) < 2 {
		fmt.Println("Numero de argumentos incorrecto")
		os.Exit(1)
	}
	id, err := strconv.Atoi(os.Args[1])
	com.CheckError(err)
	com.Depuracion("Escritor - Se han obtenido los argumentos")

	rand.Seed(time.Now().UnixNano())
	valorAleatorio := com.ValorAleatorio()
	//valorAleatorio := 3 * time.Second

	// Inicializacion de ra
	fmt.Println("Depurando: Estoy enviando el valor id " + strconv.Itoa(id))
	ra := ra.New(id, fichero, "Escribir")

	//var wg sync.WaitGroup
	//wg.Add(1)
	//go com.Esperar(&wg, puerto)

	//com.EstoyListo(id, endpointBarrera)
	//wg.Wait() // Esperar a que todos esten listos

	//go com.Final(5)

	time.Sleep(6 * time.Second)
	// Escribir
	for i := 1; i < maxPeticiones; i++ {
		ra.PreProtocol()
		fmt.Println("Depurando: Escritor en SC")
		textoAEscribir := "Soy " + strconv.Itoa(id) + " y esta es la vez " +
			"numero " + strconv.Itoa(i) + " que escribo\n"
		ra.Fichero.Escribir(textoAEscribir)
		fmt.Println("Depurando: SC - Ha escrito")
		ra.EnviarActualizar(textoAEscribir)
		fmt.Println("Depurando: SC ha enviado Actualizar")
		ra.PostProtocol()
		fmt.Println("Depurando: Escritor fuera de SC")
		time.Sleep(time.Duration(valorAleatorio) * time.Millisecond)
	}

	for {
		fmt.Println("Depurando: voy a entrar al for")
	}

}
