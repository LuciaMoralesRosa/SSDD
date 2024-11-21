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
	// Para asegurar que no se realicen varias operaciones concurrentes 
}

func CrearFichero(nombreFichero string) *Fichero {
	com.Depuracion("GestorFichero - CrearFichero: Estoy creando el fichero para leer o escribir")
	_, err := os.Create(nombreFichero)
	com.CheckError(err)
	fichero := Fichero{nombreFichero: nombreFichero, Mutex: sync.Mutex{}}
	com.Depuracion("GestorFichero - CrearFichero: El fichero se ha creado correctamente")
	return &fichero
}

// Pre: true
// Post:
func (fichero *Fichero) Leer() string {
	com.Depuracion("GestorFichero - Leer: Se va a leer de fichero")
	fichero.Mutex.Lock()
	f, err := os.Open(fichero.nombreFichero)
	com.CheckError(err)
	defer f.Close()

	escaner := bufio.NewScanner(f)
	contenidoFichero := ""

	for escaner.Scan() {
		contenidoFichero += escaner.Text()
	}

	com.Depuracion("GestorFichero - Leer: Se ha leido el texto: " + contenidoFichero)

	err = escaner.Err()
	com.CheckError(err)
	fichero.Mutex.Unlock()

	return contenidoFichero
}

func (fichero *Fichero) Escribir(texto string) {
	com.Depuracion("GestorFichero - Escribir: Se va a escribir")
	fichero.Mutex.Lock()
	f, err := os.OpenFile(fichero.nombreFichero, os.O_WRONLY|os.O_APPEND, 0666)
	com.CheckError(err)

	os.ReadFile()

	defer f.Close()
	_, err = f.WriteString(texto)
	com.CheckError(err)

	fichero.Mutex.Unlock()
	com.Depuracion("GestorFichero - Escribir: Se ha escrito correctamente")
}
