package gf

import (
	"bufio"
	"os"
	"practica2/com"
	"sync"
)

type Fichero struct {
	nombreFichero string
	Mutex         sync.Mutex
}

func CrearFichero(nombreFichero string) *Fichero {
	_, err := os.Create(nombreFichero)
	com.CheckError(err)
	fichero := Fichero{nombreFichero: nombreFichero, Mutex: sync.Mutex{}}
	return &fichero
}

// Pre: true
// Post:
func (fichero *Fichero) Leer() string {
	fichero.Mutex.Lock()
	f, err := os.Open(fichero.nombreFichero)
	com.CheckError(err)
	defer f.Close()

	escaner := bufio.NewScanner(f)
	contenidoFichero := ""

	for escaner.Scan() {
		contenidoFichero += escaner.Text()
	}

	err = escaner.Err()
	com.CheckError(err)
	fichero.Mutex.Unlock()

	return contenidoFichero
}

func (fichero *Fichero) Escribir(texto string) {
	fichero.Mutex.Lock()
	f, err := os.OpenFile(fichero.nombreFichero, os.O_WRONLY|os.O_APPEND, 0666)
	com.CheckError(err)

	defer f.Close()
	_, err = f.WriteString(texto)
	com.CheckError(err)

	fichero.Mutex.Unlock()
}
