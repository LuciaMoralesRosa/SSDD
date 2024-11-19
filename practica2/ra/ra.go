/*
* AUTOR: Rafael Tolosana Calasanz
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: septiembre de 2021
* FICHERO: ricart-agrawala.go
* DESCRIPCIÓN: Implementación del algoritmo de Ricart-Agrawala Generalizado en Go
 */
package ra

import (
	"fmt"
	"practica2/com"
	"practica2/gf"
	"practica2/ms"
	"strconv"
	"sync"

	"github.com/DistributedClocks/GoVector/govec/vclock"
)

type Request struct {
	Clock     vclock.VClock //Reloj vectorial
	Pid       int
	Operacion string // Tipo de operacion: "read" o "write"
}

type Reply struct{}

type Exclusion struct {
	op1 string
	op2 string
}

type RASharedDB struct {
	OurSeqNum vclock.VClock
	HigSeqNum vclock.VClock
	OutRepCnt int
	ReqCS     bool
	RepDefd   []bool
	ms        *ms.MessageSystem
	done      chan bool
	chrep     chan bool
	Mutex     sync.Mutex // mutex para proteger concurrencia sobre las variables
	// TODO: completar
	peticion            chan Request
	respuesta           chan Reply
	respuestaActualizar chan RespuestaActualizar
	Operacion           string
	Fichero             *gf.Fichero
	yo                  string
}

type Actualizar struct {
	Pid   int
	Texto string
}

type RespuestaActualizar struct{}

const nProcesos = 6

var matrizExclusion = [2][2]bool{
	{false, true}, // un lector no bloauea a otros lectores pero un escritor bloquea a todos
	{true, true},  // Un escritor bloquea tanto a lectores como a escritores
}

func New(me int, usersFile string, op string) *RASharedDB {
	com.Depuracion("RA - New: inicio")

	messageTypes := []ms.Message{Request{}, Reply{}, Actualizar{}, RespuestaActualizar{}}
	com.Depuracion("RA - New: estoy enviando a ms el valor me " + strconv.Itoa(me))
	msgs := ms.New(me, usersFile, messageTypes)

	yo := strconv.Itoa(me)
	fmt.Println("RA - NEW: yo soy " + yo)

	// Inicializacion de los relojes vectoriales
	reloj1 := vclock.New()
	reloj1.Set(yo, 0)
	reloj2 := vclock.New()
	reloj2.Set(yo, 0)

	com.Depuracion("RA - New: creando fichero de lectura y escritura")
	fichero := gf.CrearFichero("fichero_" + yo + ".txt")
	com.Depuracion("RA - New: fichero creado")

	ra := RASharedDB{reloj1, reloj2, 0, false, make([]bool, nProcesos), &msgs,
		make(chan bool), make(chan bool), sync.Mutex{}, make(chan Request),
		make(chan Reply), make(chan RespuestaActualizar), op, fichero, yo}

	com.Depuracion("RA - New: el valor ra.ms.me es " + strconv.Itoa(ra.ms.Me))

	com.Depuracion("RA - New: lanzando goroutinas")
	go recibirMensaje(&ra)
	go manejoPeticiones(&ra)
	go manejoRespuestas(&ra)

	return &ra
}

// REVISADO
// Pre: Verdad
// Post: Realiza  el  PreProtocol  para el  algoritmo de Ricart-Agrawala
// Generalizado
func (ra *RASharedDB) PreProtocol() {
	com.Depuracion("RA - Preprotocol: inicio")
	// TODO completar
	ra.Mutex.Lock()

	ra.ReqCS = true // Indico que quiero entrar a la seccion critica

	// Incrementar numero de secuencia y actualizacion si es necesario
	ra.OurSeqNum[ra.yo] = ra.HigSeqNum[ra.yo] + 1
	ra.OurSeqNum.Merge(ra.HigSeqNum)

	ra.Mutex.Unlock()

	ra.OutRepCnt = nProcesos - 1 // Esperamos n-1 respuestas

	for i := 1; i <= nProcesos; i++ {
		if i != ra.ms.Me {
			com.Depuracion("RA - Preprotocol: Estoy en el if con i = " + strconv.Itoa(i) + " y yo soy " + strconv.Itoa(ra.ms.Me))

			peticion := Request{
				Pid:       ra.ms.Me,
				Clock:     ra.OurSeqNum,
				Operacion: ra.Operacion,
			}
			com.Depuracion("RA - Preprotocol: Enviar peticion de " + peticion.Operacion + " a proceso " + strconv.Itoa(i))
			ra.ms.Send(i, peticion)
		}
	}

	// Esperamos a obtener todas las respuestas
	<-ra.chrep
	com.Depuracion("RA - Preprotocol: final")
}

// Pre: Verdad
// Post: Realiza  el  PostProtocol  para el  algoritmo de Ricart-Agrawala
// Generalizado
func (ra *RASharedDB) PostProtocol() {
	com.Depuracion("RA - Posprotocol: inicio")
	// TODO completar
	ra.Mutex.Lock()
	ra.ReqCS = false // Indico que ya no quiero tener acceso a la seccion critica
	ra.Mutex.Unlock()

	for i := 1; i <= nProcesos; i++ {
		com.Depuracion("RA - Posprotocol: avisando de liberacion de SC")
		if ra.RepDefd[i-1] {
			ra.RepDefd[i-1] = false
			ra.Mutex.Lock()
			ra.ms.Send(i, Reply{})
			ra.Mutex.Unlock()
		}
	}
}

func (ra *RASharedDB) Stop() {
	ra.ms.Stop()
	ra.done <- true
}

// Pre:
// Post: Devuelve true si el proceso con pid "yo" tiene prioridad sobre el
// el proceso con pid "otro"
func tengoPrioridad(reloj1 vclock.VClock, reloj2 vclock.VClock, yo int, otro int) bool {
	com.Depuracion("Mi reloj (pid = " + strconv.Itoa(yo) + ") es: " + reloj1.ReturnVCString() + " \nEl otro (pid = " + strconv.Itoa(otro) + ") reloj es: " + reloj2.ReturnVCString())
	com.Depuracion("RA - tengoPrioridad: viendo si tengo prioridad")
	if reloj1.Compare(reloj2, vclock.Descendant) { // Si reloj1 es mas reciente
		return true
	} else if reloj1.Compare(reloj2, vclock.Concurrent) { // Si son iguales
		return yo < otro
	} else { // Si reloj2 es mas reciente
		return false
	}
}

//prio := ra.OurSeqNum.Compare(clockSol, vclock.Descendant) || ((ra.OurSeqNum.Compare(clockSol, vclock.Equal) ||
//ra.OurSeqNum.Compare(clockSol, vclock.Concurrent)) && procSol > me)

func recibirMensaje(ra *RASharedDB) {
	for {
		mensaje := ra.ms.Receive()
		switch tipoMensaje := mensaje.(type) {
		case Request:
			com.Depuracion("RA - recibirMensaje: se ha recibido peticion")
			ra.peticion <- tipoMensaje
		case Reply:
			com.Depuracion("RA - recibirMensaje: se ha recibido respuesta")
			ra.respuesta <- tipoMensaje
		case Actualizar:
			com.Depuracion("RA - recibirMensaje: se ha recibido actualizar")
			ra.Fichero.Escribir(tipoMensaje.Texto)
			com.Depuracion("RA - recibirMensaje: se ha escrito el texto en el fichero")
			ra.ms.Send(tipoMensaje.Pid, RespuestaActualizar{})
			com.Depuracion("RA - recibirMensaje: se ha enviado mensaje de que todos deben actualizar")
		case RespuestaActualizar:
			com.Depuracion("RA - recibirMensaje: se ha recibido respuestaActualizar")
			ra.respuestaActualizar <- tipoMensaje
		}
	}
}

// Pre: Verdad
// Post: Cuando llega una peticion:
//   - Si el proceso tiene prioridad envia la peticion.
//   - Si no tiene prioridad o no pide acceso a SC se pospone la peticion
func manejoPeticiones(ra *RASharedDB) {
	for {
		// Obtener la peticion
		peticion := <-ra.peticion
		com.Depuracion("RA - manejoPeticiones: se ha recibido peticion")
		peticionReloj := peticion.Clock

		// Actualizo el reloj
		ra.Mutex.Lock()
		ra.HigSeqNum[ra.yo] = calcularMax(ra.HigSeqNum[ra.yo],
			peticionReloj[strconv.Itoa(peticion.Pid)])
		ra.HigSeqNum.Merge(peticionReloj) // Se añade el nuevo reloj de la peticion

		// Veo si pospongo la peticion
		soyEscritor := boolToInt(ra.Operacion == "Escribir")
		esEscritor := boolToInt(peticion.Operacion == "Escribir")

		posponer := ra.ReqCS && tengoPrioridad(ra.OurSeqNum, peticionReloj,
			ra.ms.Me, peticion.Pid) && matrizExclusion[soyEscritor][esEscritor]
		ra.Mutex.Unlock()

		if posponer {
			com.Depuracion("RA - manejoPeticiones: se ha pospuesto la peticion")
			ra.RepDefd[peticion.Pid-1] = true // Posponemos
		} else {
			com.Depuracion("RA - manejoPeticiones: no se ha pospuesto la peticion")
			ra.Mutex.Lock()
			com.Depuracion("RA - manejoPeticiones: se va a enviar mensaje de peticion")
			ra.ms.Send(peticion.Pid, Reply{}) // Hacemos peticion
			ra.Mutex.Unlock()
		}
	}
}

// Pre: Verdad
// Post: Decrementa el numero de respuestas esperadas y avisa al canal si se han
// recibido todas las respuestas esperadas
func manejoRespuestas(ra *RASharedDB) {
	for {
		<-ra.respuesta // Cuando llega una respuesta
		com.Depuracion("RA - manejoRespuestas: se ha recibido respuesta")
		ra.OutRepCnt-- // Decrementar el numero de respuestas esperadas
		if ra.OutRepCnt == 0 {
			com.Depuracion("RA - manejoRespuestas: se han recibido todas las respuestas")
			ra.chrep <- true // Si se han obtenido todas las respuestas se manda true al canal
		}
	}
}

// Pre: texto es una cadena de texto valida
// Post: Envia un mensaje de actualizacion a los procesos y espera nProcesos-1
// respuestas
func (ra *RASharedDB) EnviarActualizar(texto string) {
	com.Depuracion("RA - EnviarActualizar: se va a enviar actualizar")

	yo := ra.ms.Me
	for i := 1; i <= nProcesos; i++ {
		if i != yo {
			com.Depuracion("RA - EnviarActualizar: enviando actualizar a " + strconv.Itoa(i))
			ra.ms.Send(i, Actualizar{yo, texto})
		}
	}

	respuestasEsperadas := nProcesos - 1
	com.Depuracion("RA - EnviarActualizar: Respuestas esperadas = " + strconv.Itoa(respuestasEsperadas))
	for respuestasEsperadas > 0 {
		com.Depuracion("RA - EnviarActualizar: Esperando respuestas esperadas ")
		<-ra.respuestaActualizar
		respuestasEsperadas--
		com.Depuracion("RA - EnviarActualizar: Se ha obtenido respuesta esperada. Quedan : " + strconv.Itoa(respuestasEsperadas))
	}
	com.Depuracion("RA - EnviarActualizar: se han recibido todas las respuestasActualizar")

}

func calcularMax(a, b uint64) uint64 {
	if a < b {
		return b
	} else {
		return a
	}
}

func boolToInt(condicion bool) int {
	if condicion {
		return 1
	} else {
		return 0
	}
}
