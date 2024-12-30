package gf

/*
* AUTOR: Lucia Morales Rosa (816906) y Lizer Bernad Ferrando (779035)
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: octubre de 2024
* FICHERO: gestorFicheros.go
* DESCRIPCIÓN: Contiene la implementacion del gestro de ficheros para el
* 	algoritmo de Ricart-Agrawala Generalizado
 */

import (
	"os"
	"practica2/com"
)

type Fichero struct {
	nombreFichero string
}

// CrearFichero crea un nuevo archivo con el nombre especificado y devuelve un
// puntero a un objeto Fichero.
func CrearFichero(nombreFichero string) *Fichero {
	com.Depuracion("GestorFichero - CrearFichero: Estoy creando el fichero para leer o escribir")
	_, err := os.Create(nombreFichero)
	com.CheckError(err)
	fichero := Fichero{nombreFichero: nombreFichero}
	com.Depuracion("GestorFichero - CrearFichero: El fichero se ha creado correctamente")
	return &fichero
}

// Leer lee el contenido del archivo gestionado por el objeto Fichero y lo
// devuelve como un string.
func (fichero *Fichero) Leer() string {
	com.Depuracion("GestorFichero - Leer: Se va a leer de fichero")

	// Leer el contenido del fichero
	contenidoBytes, err := os.ReadFile(fichero.nombreFichero)
	com.CheckError(err)

	// Convertir el contenido del fichero a string
	contenidoFichero := string(contenidoBytes)
	com.Depuracion("GestorFichero - Leer: Se ha leido el texto: " + contenidoFichero)

	return contenidoFichero
}

// Escribir escribe el texto proporcionado al final del archivo gestionado por
// el objeto Fichero.
func (fichero *Fichero) Escribir(texto string) {
	com.Depuracion("GestorFichero - Escribir: Se va a escribir")

	// Abrir fichero con permisos y append para escribir al final
	f, err := os.OpenFile(fichero.nombreFichero, os.O_WRONLY|os.O_APPEND, 0666)
	com.CheckError(err)
	defer f.Close()

	// Escribir el texto del escritor
	_, err = f.WriteString(texto)
	com.CheckError(err)

	com.Depuracion("GestorFichero - Escribir: Se ha escrito correctamente")
}
