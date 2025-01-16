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

	almacen := make(map[string]string)

	// obtener entero de indice de este nodo
	yo := strings.ReplaceAll(os.Args[1], "raft-", "") // raft-i -> i
	me, err := strconv.Atoi(yo)
	check.CheckError(err, "Main, mal numero entero de indice de nodo:")

	var nodos []rpctimeout.HostPort
	// Resto de argumento son los end points como strings
	// De todas la replicas-> pasarlos a HostPort
	for _, endPoint := range os.Args[2:] {
		nodos = append(nodos, rpctimeout.HostPort(endPoint))
	}

	canalAplicarOp := make(chan raft.AplicaOperacion, 1000)

	// Parte Servidor
	nr := raft.NuevoNodo(nodos, me, canalAplicarOp)
	rpc.Register(nr)

	go aplicarOperacion(almacen, canalAplicarOp)

	fmt.Println("Replica escucha en :", me, " de ", os.Args[2:])

	l, err := net.Listen("tcp", os.Args[2:][me])
	check.CheckError(err, "Main listen error:")

	for {
		rpc.Accept(l)
	}
}

func aplicarOperacion(almacen map[string]string, canalConexion chan raft.AplicaOperacion) {
	for {
		op := <-canalConexion
		fmt.Printf("Soy el SERVIDOR y me ha llegado una operacion de tipo %s\n", op.Operacion.Operacion)
		if op.Operacion.Operacion == "leer" {
			op.Operacion.Valor = almacen[op.Operacion.Clave]
		} else if op.Operacion.Operacion == "escribir" {
			almacen[op.Operacion.Clave] = op.Operacion.Valor
			//op.Operacion.Valor = "escrito"
		}
		fmt.Printf("Soy el SERVIDOR y voy a responder a mi nodo por el canal aplicarOp\n")
		canalConexion <- op
	}
}
