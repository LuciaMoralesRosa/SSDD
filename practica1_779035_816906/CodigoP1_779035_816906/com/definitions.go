/*
* AUTOR: Rafael Tolosana Calasanz y Unai Arronategui
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: septiembre de 2022
* FICHERO: definitions.go
* DESCRIPCIÓN: contiene las definiciones de estructuras de datos necesarias para
*			la práctica 1
 */
package com

import (
	"fmt"
	"os"
	"time"
)

type TPInterval struct {
	Min int
	Max int
}

type Request struct {
	Id       int
	Interval TPInterval
}

type TimeCommEvent struct {
	Id int
	T  time.Time
}

type Reply struct {
	Id     int
	Primes []int
}

func CheckError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
