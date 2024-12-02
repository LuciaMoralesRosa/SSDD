// package cltraft
package main

import (
	"fmt"
	"log"
	"net/rpc"
	"raft/internal/raft"
	"strconv"
	"time"
)

const numeroEnvios = 4

func main() {
	misPuertos := []string{":31110", ":31111", ":31112", ":31113", ":31114",
		":31115", ":31116", ":31117", ":31118", ":31119"}

	idLider := 1 // Inicialmente suponemos un líder
	envio := 0
	for envio < numeroEnvios {
		lider := misPuertos[idLider]
		// Creamos la conexion tcp con el lider
		cliente, err := rpc.Dial("tcp", lider)
		if err != nil {
			log.Fatal("Conexion:", err)
		}
		defer cliente.Close()

		// Creacion de la operacion
		operacion := raft.TipoOperacion{Operacion: "escribir", Clave: "x",
			Valor: "escritura_" + strconv.Itoa(envio)}

		// Realizar la operacion
		var reply raft.ResultadoRemoto
		err = cliente.Call("NodoRaft.SometerOperacionRaft", operacion, &reply)
		if err != nil {
			log.Fatal("rpc :", err)
		}
		fmt.Println(reply)

		// Esperar y enviar siguiente operacion
		time.Sleep(1 * time.Second)
		envio++
	}
}
