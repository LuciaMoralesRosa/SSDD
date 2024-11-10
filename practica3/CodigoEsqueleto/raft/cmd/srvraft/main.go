package main

import (
	//"errors"
	//"fmt"
	//"log"
	"net"
	"net/rpc"
	"os"
	"raft/internal/comun/check"
	"raft/internal/comun/rpctimeout"
	"raft/internal/raft"
	"strconv"
	"time"
	//"time"
)

func main() {
	// obtener entero de indice de este nodo
	me, err := strconv.Atoi(os.Args[1])
	check.CheckError(err, "Main, mal numero entero de indice de nodo:")

	var nodos []rpctimeout.HostPort
	// Resto de argumento son los end points como strings
	// De todas la replicas-> pasarlos a HostPort
	for _, endPoint := range os.Args[2:] {
		nodos = append(nodos, rpctimeout.HostPort(endPoint))
	}

	// Parte Servidor
	nr := raft.NuevoNodo(nodos, me, make(chan raft.AplicaOperacion, 1000))
	rpc.Register(nr)

	//fmt.Println("Replica escucha en :", me, " de ", os.Args[2:])

	time.Sleep(2 * time.Second)
	l, err := net.Listen("tcp", os.Args[2:][me])
	check.CheckError(err, "Main listen error:")

	rpc.Accept(l)
}
