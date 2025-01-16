package com

/*
* AUTOR: Lucia Morales Rosa (816906) y Lizer Bernad Ferrando (779035)
* Modificacion del fichero "defintions.go" proporcionado en la practica 1 de
* 		Rafael Tolosana y Unai Arronategui
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: octubre de 2024
* FICHERO: definitions.go
* DESCRIPCIÓN: contiene las funciones auxiliares necesarias para la práctica 2
 */

import (
	"fmt"
	"os"

	"golang.org/x/exp/rand"
)

// Variable para evaluar si es la primera vez en la ejecucion que se llama a
// Depuracion para reescribir el fichero Depuracion.txt o crearlo
var primeraVez = true
var primeraVezLog = true
var openLogfile *os.File

// CheckError maneja errores críticos. Si err no es nulo, imprime el error en
// stderr y termina el programa
func CheckError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

// ValorAleatorio genera un número entero aleatorio entre 100 y 400
func ValorAleatorio() int {
	min := 100
	max := 400
	// Inicializar la semilla con la hora actual
	numeroRango := min + rand.Intn(max-min+1)
	return numeroRango

}

// Depuracion escribe el texto de depuración en un archivo llamado
// "Depuracion.txt". Si es la primera ejecución, verifica y elimina el archivo
// si ya existe.
func Depuracion(textoDepuracion string) {
	if primeraVez {
		// Comprobar si el archivo existe
		if _, err := os.Stat("Depuracion.txt"); os.IsNotExist(err) {
			fmt.Println("El archivo no existe.")
		} else {
			// Intentar borrar el archivo
			err := os.Remove("Depuracion.txt")
			CheckError(err)
		}
		primeraVez = false
	}

	// Abrir el archivo en modo append, con permisos de escritura y lectura
	f, err := os.OpenFile("Depuracion.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	CheckError(err)

	// Asegurarse de cerrar el archivo al salir de la función
	defer f.Close()

	// Escribir el texto de depuracion en el archivo
	_, err = f.WriteString(textoDepuracion + "\n")
	CheckError(err)

	// Escribir el texto de depuracion por pantalla
	//fmt.Println(textoDepuracion)
}
