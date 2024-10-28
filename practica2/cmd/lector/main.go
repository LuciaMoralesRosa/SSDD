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

func leer(ra *ra.RASharedDB, ficheroLectura string, gestor *g.Gestor) {
	for i := 0; i < 5; i++ {
		fmt.Println("Depuracion lector.leer iteracion " + strconv.Itoa(i))
		fmt.Println("Depuracion lector.leer entrando a ra.PreProtocol")
		ra.PreProtocol()
		fmt.Println("Depuracion lector.leer entrando a gestor.leerFichero")
		lecturaDeFichero := gestor.LeerFichero(ficheroLectura)
		fmt.Println("He leido esto: " + lecturaDeFichero)
		fmt.Println("Depuracion lector.leer entrando a ra.PostProtocol")
		ra.PostProtocol()
	}
}

func main() {

	/*go run main.go 2 ../../ms/users.txt ../ficheroTexto */
	lineNumber, err := strconv.Atoi(os.Args[1])
	if err != nil || lineNumber < 1 {
		fmt.Println("Invalid line number")
		return
	}

	ficheroUsuarios := os.Args[2]
	procesoEscritor := false
	ficheroLectura := os.Args[3]
	gestor := g.New()

	ra := ra.New(lineNumber, ficheroUsuarios, procesoEscritor, gestor)

	go leer(ra, ficheroLectura, &gestor)
}
