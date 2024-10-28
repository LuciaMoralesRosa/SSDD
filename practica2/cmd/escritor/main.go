package main

import (
	"fmt"
	"os"
	g "practica2/gestor"
	"practica2/ra"
	"strconv"
)

// Maneja los errores del sistema, mostrandolo por pantalla y terminando la ejecucion
func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func escribir(ra *ra.RASharedDB, ficheroEscritura string, textoEscritura string, gestor *g.Gestor) {
	for i := 0; i < 5; i++ {
		fmt.Println("Depuracion escritor.escribir iteracion " + strconv.Itoa(i))
		fmt.Println("Depuracion escritor.escribir entrando a ra.PreProtocol")
		ra.PreProtocol()
		fmt.Println("Depuracion escritor.escribir entrando a gestor.escribirFichero")
		gestor.EscribirFichero(ficheroEscritura, textoEscritura)
		fmt.Println("Depuracion escritor.escribir entrando a ra.escribirTexto")
		ra.EscribirTexto(ficheroEscritura, textoEscritura+" "+strconv.Itoa(i))
		fmt.Println("Depuracion escritor.escribir entrando a ra.PostProtocol")
		ra.PostProtocol()
	}
}

func main() {
	/*go run main.go 1 ../../ms/users.txt ../ficheroTexto.txt heEscritoEstoooYEY*/

	lineNumber, err := strconv.Atoi(os.Args[1])
	if err != nil || lineNumber < 1 {
		fmt.Println("Invalid line number")
		return
	}

	ficheroUsuarios := os.Args[2]
	procesoEscritor := true
	ficheroEscritura := os.Args[3]
	textoEscritura := os.Args[4]

	//todosNodosConectados(lineNumber, ficheroUsuarios)
	gestor := g.New()

	ra := ra.New(lineNumber, ficheroUsuarios, procesoEscritor, gestor)

	go escribir(ra, ficheroEscritura, textoEscritura, &gestor)
}
