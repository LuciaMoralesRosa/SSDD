package gf

import (
	"fmt"
	"os"
	"sync"
)

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

type Fichero struct {
	nombreFichero string
	Mutex         sync.Mutex
}

func CrearFichero(nombreFichero string) *Fichero {
	_, err := os.Create(nombreFichero)
	checkError(err)
	fichero :

// Pre: true
// Post:
func (fichero *Fichero) Leer() string {
	fichero.Mutex.Lock()
	f, err := os.Open(fichero.nombreFi= Fichero{nombreFichero: nombreFichero, Mutex: sync.Mutex{}}
	return &fichero
}chero)
	CheckError()
}