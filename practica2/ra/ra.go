/*
* AUTOR: Rafael Tolosana Calasanz
* Autores: Lizer Bernad (779035) y Lucia Morales (816906)
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: septiembre de 2021
* FICHERO: ricart-agrawala.go
* DESCRIPCIÓN: Implementación del algoritmo de Ricart-Agrawala Generalizado en Go
 */
package ra

import (
	"bufio"
	"os"
	g "practica2/gestor"
	"practica2/ms"
	"strconv"
	"sync"

	"github.com/DistributedClocks/GoVector/govec"
)

type Request struct {
	Clock             []byte // Marca temporal vectorial
	Pid               int    // PID del proceso solicitante
	PeticionEscritura bool   // Indica si es peticion de escritura (true) o lectura (false)
	NumeroSecuencia   int    // Numero de secuencia
}

type Reply struct {
	Clock []byte // Marca temporal vectorial
}

type Escribir struct {
	Fichero string // fichero donde escribir
	Texto   string // Contenido a escribir
	Clock   []byte // Marca temporal vectorial
}

type RASharedDB struct {
	nodos     int               // Numero de nodos
	OurSeqNum int               // Numero de secuencia actual
	HigSeqNum int               // Numero de secuencia mas alto observado
	OutRepCnt int               // Respuestas externas pendientes
	ReqCS     bool              // Indica si el proceso esta solicitando acceso a SC
	RepDefd   []bool            // Array de pospuestos
	ms        *ms.MessageSystem // Gestor de mensajes
	done      chan bool         // Canal para el manejo de la terminacion
	chrep     chan bool         // Canal para el manejo de respuestas

	// Para lectores y escritores
	procesoEscritor bool // Indica si el proceso es escritor (true) o lector (false)
	// Para relojes vectoriales
	logger *govec.GoLog // Manejo de relojes vectoriales y trazabilidad distribuida
	g      *g.Gestor
	// mutex para proteger concurrencia sobre las variables
	Mutex sync.Mutex
}

//[0][0] -> lector lector
//[0][1] -> lector escritor
//[1][0] -> escritor lector
//[1][1] -> escritor escritor

var matrizExclusion = [2][2]bool{
	{false, true}, // un lector no bloauea a otros lectores pero un escritor bloquea a todos
	{true, true},  // Un escritor bloquea tanto a lectores como a escritores
}

// Inicializa una nueva instancia de RASharedBD
func New(me int, usersFile string, esEscritor bool, g g.Gestor) *RASharedDB {
	// Definicion de los tipos de mensajes que soporta el sistema
	messageTypes := []ms.Message{Request{}, Reply{}, Escribir{}} // Tipos de mensajes
	// Inicializacion del logger
	logger := govec.InitGoVector("Mi proceso: "+strconv.Itoa(me), "LogFile", govec.GetDefaultConfig())

	// Creacion del gestor de mensajes
	msgs := ms.New(me, usersFile, messageTypes)
	nodes := contarLineas(usersFile)
	// Inicializacion de la estructura ra
	ra := RASharedDB{nodes, 0, 0, 0, false, make([]bool, nodes), &msgs, make(chan bool),
		make(chan bool), esEscritor, logger, &g, sync.Mutex{}}

	// goroutina de recepcion de mensajes
	go ra.recibirMensaje()
	return &ra
}

// Gestiona los mensajes entrantes. Constantemente esta esperando mensajes entrantes.
func (ra *RASharedDB) recibirMensaje() {
	for {
		msg := ra.ms.Receive()
		var entrada string
		switch x := msg.(type) {
		case Request:
			ra.logger.UnpackReceive("Recibir respuesta", x.Clock, &entrada,
				govec.GetDefaultLogOptions())
			ra.peticionRecibida(x)
		case Reply:
			ra.logger.UnpackReceive("Recibir respuesta", x.Clock, &entrada,
				govec.GetDefaultLogOptions())
			ra.respuestaRecibida()
		case Escribir:
			ra.logger.UnpackReceive("Recibir escritura: "+x.Texto, x.Clock, &entrada,
				govec.GetDefaultLogOptions())
			ra.escribir(x.Fichero, x.Texto)
		}
	}
}

func (ra *RASharedDB) escribir(fichero string, texto string) {
	ra.g.EscribirFichero(fichero, texto)
}

func (ra *RASharedDB) respuestaRecibida() {
	ra.chrep <- true
}

func (ra *RASharedDB) peticionRecibida(mensaje Request) {
	var posponerPeticion bool
	ra.Mutex.Lock() // Vamos a bloquear
	ra.HigSeqNum = max(ra.HigSeqNum, mensaje.NumeroSecuencia)

	// Se pospone la peticion si:
	//	1 - Mi numero de secuencia es menor al numero de secuencia del mensaje
	//	2 - Los numero de secuencia son iguales pero mi pid es menor
	posponerPeticion = ra.ReqCS && (ra.OurSeqNum < mensaje.NumeroSecuencia ||
		(ra.OurSeqNum == mensaje.NumeroSecuencia && ra.ms.Me() < mensaje.Pid))

	ra.Mutex.Unlock()

	// Si se pospone la peticion y soy un proceso escritor o la peticion es de escritura
	if posponerPeticion && (matrizExclusion[boolToInt(ra.procesoEscritor)][boolToInt(mensaje.PeticionEscritura)]) {
		ra.logger.LogLocalEvent("Posponer", govec.GetDefaultLogOptions())
		ra.Mutex.Lock()
		// Indicamos que se ha pospuesto un mensaje del proceso pid
		ra.RepDefd[mensaje.Pid-1] = true
		ra.Mutex.Unlock()
	} else {
		// Si no se pospone o es un proceso de lectura y una peticion de lectura
		datosEnvio := ra.logger.PrepareSend("Enviar respuesta", "Respuesta", govec.GetDefaultLogOptions())
		ra.ms.Send(mensaje.Pid, Reply{datosEnvio})
	}
}

func (ra *RASharedDB) EscribirTexto(fichero string, texto string) {
	ra.logger.LogLocalEvent("Indicar escritura", govec.GetDefaultLogOptions())
	for i := 1; i <= ra.nodos; i++ {
		if i != ra.ms.Me() {
			datosEnvio := ra.logger.PrepareSend("Enviar escribir", "Escritura", govec.GetDefaultLogOptions())
			ra.ms.Send(i, Escribir{fichero, texto, datosEnvio})
		}
	}
}

// Pre: Verdad
// Post: Realiza  el  PreProtocol  para el  algoritmo de
//
//	Ricart-Agrawala Generalizado
//
// Proceso que isgue in proceso cuando quiere acceder a la seccion critica
func (ra *RASharedDB) PreProtocol() {
	ra.Mutex.Lock()
	ra.ReqCS = true
	ra.OurSeqNum = ra.HigSeqNum + 1
	ra.Mutex.Unlock()
	ra.OutRepCnt = ra.nodos - 1

	// Enviar peticion a todos menos a mi mismo
	for i := 1; i <= ra.nodos; i++ {
		if i != ra.ms.Me() {
			datosEnvio := ra.logger.PrepareSend("Enviar peticion", "Peticion",
				govec.GetDefaultLogOptions())
			ra.ms.Send(i, Request{datosEnvio, ra.ms.Me(), ra.procesoEscritor,
				ra.OurSeqNum})
		}
	}

	for ra.OutRepCnt > 0 {
		<-ra.chrep
		ra.OutRepCnt = ra.OutRepCnt - 1
	}

}

// Pre: Verdad
// Post: Realiza  el  PostProtocol  para el  algoritmo de
//
//	Ricart-Agrawala Generalizado
//
// El proceso libera la seccion critica y notifica a todos los procesos
func (ra *RASharedDB) PostProtocol() {
	ra.Mutex.Lock()
	ra.ReqCS = false
	for i := 1; i <= ra.nodos; i++ {
		if ra.RepDefd[i-1] {
			ra.RepDefd[i-1] = false
			datosEnvio := ra.logger.PrepareSend("Enviar respuesta", "Respuesta",
				govec.GetDefaultLogOptions())
			ra.ms.Send(i, Reply{datosEnvio})
		}
	}
	ra.Mutex.Unlock()
}

func (ra *RASharedDB) Stop() {
	ra.ms.Stop()
	ra.done <- true
}

func contarLineas(ruta string) int {
	archivo, err := os.Open(ruta)
	if err != nil {
		return 0
	}
	defer archivo.Close()

	scanner := bufio.NewScanner(archivo)
	lineas := 0
	for scanner.Scan() {
		lineas++
	}

	if err := scanner.Err(); err != nil {
		return 0
	}

	return lineas
}

func boolToInt(valor bool) int {
	if valor {
		return 1
	} else {
		return 0
	}
}

func max(valor1 int, valor2 int) int {
	if valor1 > valor2 {
		return valor1
	} else {
		return valor2
	}
}
