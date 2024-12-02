/*
* AUTOR: Rafael Tolosana Calasanz y Unai Arronategui
* Lucia Morales Rosa (816906) y Lizer Bernad Ferrando (779035)
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: noviembre de 2024
* FICHERO: main.go
* DESCRIPCIÓN: servidor de raft para la practica 3
 */
package main

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"raft/internal/comun/check"
	"raft/internal/comun/rpctimeout"
	"raft/internal/raft"
	"strconv"
)

func main() {
// Obtener índice de este nodo
	me, err := strconv.Atoi(os.Args[1])
	check.CheckError(err, "Servidor - Error en parametro introducido:")
	
	var nodos []rpctimeout.HostPort
	// El resto de argumento son los endpoints como strings
	// Cada endpoint se convierte a HostPort
	for _, endPoint := range os.Args[2:] {
		nodos = append(nodos, rpctimeout.HostPort(endPoint))
	}
 	
	// Crear nodo
	nr := raft.NuevoNodo(nodos, me, make(chan raft.AplicaOperacion, 1000))
	rpc.Register(nr)
 	
	fmt.Println("Réplica escucha en:", me, "de", os.Args[2:])
 	
	l, err := net.Listen("tcp", os.Args[2:][me])
	check.CheckError(err, "Servidor - Error en listen:")
 	
	rpc.Accept(l)
}