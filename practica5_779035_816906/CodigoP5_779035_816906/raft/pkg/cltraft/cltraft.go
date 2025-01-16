package main

import (
	"fmt"
	"raft/internal/comun/check"
	"raft/internal/comun/rpctimeout"
	"raft/internal/raft"
	"strconv"
	"time"
)

const (
	NODOS = 3
)

func main() {
	fmt.Println("Se va a ejecutar el cliente")

	server := "raft-service.default.svc.cluster.local:6000"

	var dirs []string

	for i := 0; i < NODOS; i++ {
		// raft-i.raft-service.default.svc.cluster.local:6000
		nuevoNodo := "raft-" + strconv.Itoa(i) + "." + server
		dirs = append(dirs, nuevoNodo)
	}

	var nodos []rpctimeout.HostPort

	for _, endPoint := range dirs {
		nodos = append(nodos, rpctimeout.HostPort(endPoint))
	}

	// Esperar unos segundos
	time.Sleep(7 * time.Second)

	op1 := raft.TipoOperacion{"escribir", "escritura1", "1"}
	op2 := raft.TipoOperacion{"leer", "escritura1", ""}

	var respuesta raft.ResultadoRemoto

	for respuesta.IdLider == -1 {
		// Asumimos que el lider es el nodo 0
		err := nodos[0].CallTimeout("NodoRaft.SometerOperacionRaft", op1,
			&respuesta, 3 * time.Second)
		check.CheckError(err, "SometerOperacion")
	}

	if !respuesta.EsLider {
		// Someter la operacion correctamente al nodo lider
		err := nodos[respuesta.IdLider].CallTimeout("NodoRaft.SometerOperacionRaft",
			op1, &respuesta, 3 * time.Second)
		check.CheckError(err, "SometerOperacion")
	}

	fmt.Printf("En la primera operacion se ha devuelto el valor: %s\n",
		respuesta.ValorADevolver)
	
	time.Sleep(1 * time.Second)

	// Someter una segunda operacion de lectura
	err := nodos[respuesta.IdLider].CallTimeout("NodoRaft.SometerOperacionRaft",
		op2, &respuesta, 3 * time.Second)
	check.CheckError(err, "SometerOperacion")

	fmt.Printf("En la segunda operacion se ha devuelto el valor: %s\n",
		respuesta.ValorADevolver)

}
