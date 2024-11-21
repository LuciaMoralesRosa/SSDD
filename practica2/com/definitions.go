/*
* AUTOR: Lucia Morales Rosa (816906) y Lizer Bernad Ferrando (779035)
* Modificacion del fichero "defintions.go" proporcionado en la practica 1 de
* 		Rafael Tolosana y Unai Arronategui
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: octubre de 2024
* FICHERO: definitions.go
* DESCRIPCIÓN: contiene las definiciones funciones necesarias para la práctica 2
* y su correcto despliegue
 */
package com

import (
	"fmt"
	"os"

	"golang.org/x/exp/rand"
)

var primeraVez = true

type MensajeBarrera struct {
	Id    int
	Listo bool
}

func CheckError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func ValorAleatorio() int {
	min := 100
	max := 400
	// Inicializar la semilla con la hora actual
	numeroRango := min + rand.Intn(max-min+1)
	return numeroRango

}

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

	// Abrir S(o crear) el archivo en modo append, con permisos de escritura y lectura
	fichero, err := os.OpenFile("Depuracion.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	CheckError(err)

	// Asegurarse de cerrar el archivo al salir de la función
	defer fichero.Close()

	// Escribir el texto en el archivo
	_, err = fichero.WriteString(textoDepuracion + "\n")
	//fmt.Println(textoDepuracion)
	CheckError(err)
}
