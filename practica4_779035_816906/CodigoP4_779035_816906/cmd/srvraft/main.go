package main

import (
	//"errors"
	//"fmt"
	//"log"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"raft/internal/comun/check"
	"raft/internal/comun/rpctimeout"
	"raft/internal/raft"
	"strconv"
	//"time"
)

func main() {

	// Se van a guardar las Clave-valor
	almacen := make(map[string]string)

	// obtener entero de indice de este nodo
	me, err := strconv.Atoi(os.Args[1])
	check.CheckError(err, "Main, mal numero entero de indice de nodo:")

	var nodos []rpctimeout.HostPort
	// Resto de argumento son los end points como strings
	// De todas la replicas-> pasarlos a HostPort
	for _, endPoint := range os.Args[2:] {
		nodos = append(nodos, rpctimeout.HostPort(endPoint))
	}

	// Creacion del canal para comunicarse con el nodo
	canalAplicarOp := make(chan raft.AplicaOperacion, 1000)

	// Parte Servidor
	nr := raft.NuevoNodo(nodos, me, canalAplicarOp)
	rpc.Register(nr)

	// Go rutina para gestionar las operaciones solicitadas por el cliente
	go aplicarOperacion(almacen, canalAplicarOp)

	fmt.Println("Replica escucha en :", me, " de ", os.Args[2:])

	l, err := net.Listen("tcp", os.Args[2:][me])
	check.CheckError(err, "Main listen error:")

	for {
		rpc.Accept(l)
	}
}

func aplicarOperacion(almacen map[string]string,
	canalConexion chan raft.AplicaOperacion) {
	for {
		// Escuchar envio de operaciones desde el nodo
		op := <-canalConexion
		fmt.Printf("Soy el SERVIDOR y me ha llegado una operacion de tipo "+
			"%s\n", op.Operacion.Operacion)
		// Comportamiento segun la operacion
		if op.Operacion.Operacion == "leer" {
			op.Operacion.Valor = almacen[op.Operacion.Clave]
		} else if op.Operacion.Operacion == "escribir" {
			almacen[op.Operacion.Clave] = op.Operacion.Valor
			//op.Operacion.Valor = "escrito"
		}
		fmt.Printf("Soy el SERVIDOR y voy a responder a mi nodo por el " +
			"canal aplicarOp\n")
		// Respuesta al nodo
		canalConexion <- op
	}
}
