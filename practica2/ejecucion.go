package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"os/exec"
	"practica2/com"
	"strconv"
	"sync"
	"time"
)

const nProcesos = 6
const puerto = ":31110"
const tiempoDomir = 6

func lanzarProcesos(tipo string, direccion string, id int, usuario string) {
	dir := " /home/" + usuario + "/practica2/cmd/" + tipo + "/main "
	com.Depuracion("Ejecucion - Lanzar Procesos: Ejecutando... ssh" + direccion + dir + strconv.Itoa(id))
	err := exec.Command("ssh", direccion, dir, strconv.Itoa(id), "> output.log 2> error.log").Run()
	com.Depuracion("El comando se ha ejecutado correctamente")
	com.CheckError(err)
}

func iniciarProcesos(usuario string) {
	id := 1
	for i := 2; i <= 4; i++ {
		direccion := fmt.Sprintf("%s@%s%d", usuario, "192.168.3.", i)
		go lanzarProcesos("lector", direccion, id, usuario)
		id++
		go lanzarProcesos("escritor", direccion, id, usuario)
		id++
	}
}

func esperarProcesos(wg *sync.WaitGroup) {
	defer wg.Done()
	listener, err := net.Listen("tcp", puerto)
	com.CheckError(err)
	defer listener.Close()

	i := 1
	for i <= nProcesos {
		conn, err := listener.Accept()
		com.CheckError(err)
		defer conn.Close()

		var mensaje com.MensajeBarrera
		decoder := gob.NewDecoder(conn)
		err = decoder.Decode(&mensaje)
		com.CheckError(err)

		if mensaje.Listo {
			i++
		}
	}
}

func trabajar() {
	fichero, err := os.Open("ms/usuariosBarrera.txt")
	com.CheckError(err)
	defer fichero.Close()

	escaner := bufio.NewScanner(fichero)

	for escaner.Scan() {
		linea := escaner.Text()

		conn, err := net.Dial("tcp", linea)
		com.CheckError(err)
		defer conn.Close()

		mensaje := com.MensajeBarrera{
			Id:    0,
			Listo: false,
		}
		encoder := gob.NewEncoder(conn)
		err = encoder.Encode(mensaje)
		com.CheckError(err)
	}
}

func main() {
	com.Depuracion("Ejecucion - Creando fichero depuracion")

	if len(os.Args) != 2 {
		com.Depuracion("Ejecucion - n parametros incorrecto")
		fmt.Println("Numero de parametros incorrecto")
		os.Exit(1)
	}
	usuario := os.Args[1]

	var wg sync.WaitGroup
	wg.Add(1)
	com.Depuracion("Ejecucion - lanzando espera procesos")
	go esperarProcesos(&wg)

	com.Depuracion("Ejecucion - Iniciando procesos")
	iniciarProcesos(usuario)
	wg.Wait()

	trabajar()
	time.Sleep(tiempoDomir * time.Second)
}
