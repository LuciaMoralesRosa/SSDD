/*
* AUTORES: Lizer Bernad (779035) y Lucia Morales (816906)
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: octubre de 2024
* FICHERO: gestor.go
* DESCRIPCIÓN: Implementación de un sistema de gestor de escritores y lectores
*/

package ra

import (
	"fmt"
	"io/ioutil"
)

type Gestor struct {}

// Maneja los errores del sistema, mostrandolo por pantalla y terminando la ejecucion
func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func leerFichero(fichero string) string {
	contenidoFichero, err := ioutil.ReadFile(fichero)
	checkError(err)
	return string(contenidoFichero)
}

func crearFichero(fichero string) {
	_, err := os.Create(fichero)
	checkError(err)
}

func escribirFichero(fichero string, texto string) {
	contenidoFichero, err := ioutil.ReadFile(fichero)
	if os.IsNotExist(err) {
		crearFichero(fichero)
		contenidoFichero = ""
	}
	else {
		checkError(err)
	}
	err := ioutil.WriteFile(fichero, []byte(contenidoFichero + texto + "\n"), 0644)
	checkError(err)
}
