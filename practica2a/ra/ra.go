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
	"gf"
	"ms"
	"practica2a/ms"
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
	respuestaActualizar chan respuestaActualizar
	Operacion           string
	Fichero             *gf.Fichero
	yo                  string
}

type Actualizar struct {
	Pid   int
	Texto string
}

type respuestaActualizar struct{}

const nProcesos = 6

var matrizExclusion = [2][2]bool{
	{false, true}, // un lector no bloauea a otros lectores pero un escritor bloquea a todos
	{true, true},  // Un escritor bloquea tanto a lectores como a escritores
}

func New(me int, usersFile string, operacion string) *RASharedDB {
	messageTypes := []Message{Request, Reply, Actualizar{}, respuestaActualizar{}}
	msgs = ms.New(me, usersFile, messageTypes)

	yo := strconv.Itoa(me)

	// Inicializacion de los relojes vectoriales
	reloj1 := vclock.New()
	reloj1.Set(yo, 0)
	reloj2 := vclock.New()
	reloj2.Set(yo, 0)

	fichero := gf.CrearFichero("fichero_" + yo + ".txt")

	ra := RASharedDB{reloj1, reloj2, 0, false, make([]bool, nProcesos), &msgs,
		make(chan bool), make(chan bool), &sync.Mutex{}, make(chan string),
		make(chan Request), make(chan Reply),
		make(chan respuestaActualizar), operacion, fichero, yo}

	go recibirMensaje(&ra)
	go manejoPeticiones(&ra)
	go manejoRespuestas(&ra)

	return &ra
}

// Pre: Verdad
// Post: Realiza  el  PreProtocol  para el  algoritmo de Ricart-Agrawala
// Generalizado
func (ra *RASharedDB) PreProtocol() {
	// TODO completar
	ra.Mutex.Lock()

	ra.ReqCS = true //

	// Incrementar numero de secuencia y actualizacion si es necesario
	ra.OurSeqNum[ra.yo] = ra.HigSeqNum[ra.yo] + 1
	ra.OurSeqNum.Merge(ra.HigSeqNum)

	ra.Mutex.Unlock()

	ra.OutRepCnt = nProcesos - 1 // Esperamos n-1 respuestas

	for i := 1; i <= nProcesos; i++ {
		if i != ra.ms.Me {
			peticion := Request{
				Pid:       ra.ms.Me,
				Clock:     ra.OurSeqNum,
				Operacion: ra.Operacion,
			}
			ra.ms.Send(i, peticion)
		}
	}

	// Esperamos a obtener todas las respuestas
	<-ra.chrep
}

// Pre: Verdad
// Post: Realiza  el  PostProtocol  para el  algoritmo de Ricart-Agrawala
// Generalizado
func (ra *RASharedDB) PostProtocol() {
	// TODO completar
	ra.Mutex.Lock()
	ra.ReqCS = false
	ra.Mutex.Unlock()

	for i := 1; i <= nProcesos; i++ {
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
	if reloj1.Compare(reloj2, vclock.Descendant) { // Si reloj1 es mas reciente
		return true
	} else if reloj1.Compare(reloj2, vclock.Concurrent) { // Si son iguales
		return yo < otro
	} else { // Si reloj2 es mas reciente
		return false
	}
}

func recibirMensaje(ra *RASharedDB) {
	for {
		mensaje := ra.ms.Receive()
		switch tipoMensaje := mensaje.(type) {
		case Request:
			ra.peticion <- tipoMensaje
		case Reply:
			ra.respuesta <- tipoMensaje
		case Actualizar:
			ra.Actualizar <- tipoMensaje
		case respuestaActualizar:
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
		peticionReloj := peticion.Clock

		// Actualizo el reloj y veo si pospongo la peticion
		ra.Mutex.Lock()
		ra.HigSeqNum[ra.yo] = calcularMax(ra.HigSeqNum[ra.yo],
			peticionReloj[strconv.Itoa(peticion.Pid)])
		ra.HigSeqNum.Merge(peticionReloj) // Se añade el nuevo reloj de la peticion
		posponer := ra.ReqCS && tengoPrioridad(ra.OurSeqNum, peticionReloj,
			ra.ms.Me, peticion.Pid)
		ra.Mutex.Unlock()

		if posponer {
			ra.RepDefd[peticion.Pid-1] = true // Posponemos
		} else {
			ra.Mutex.Lock()
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
		ra.OutRepCnt-- // Decrementar el numero de respuestas esperadas
		if ra.OutRepCnt == 0 {
			ra.chrep <- true // Si se han obtenido todas las respuestas se manda true al canal
		}
	}
}

// Pre: texto es una cadena de texto valida
// Post: Envia un mensaje de actualizacion a los procesos y espera nProcesos-1
// respuestas
func (ra *RASharedDB) EnviarActualizar(texto string) {
	yo := ra.ms.Me
	for i := 1; i < nProcesos; i++ {
		if i != yo {
			ra.ms.Send(i, Actualizar{yo, texto})
		}
	}

	respuestasEsperadas := nProcesos - 1
	for respuestasEsperadas > 0 {
		<-ra.respuestaActualizar
		respuestasEsperadas--
	}
}

func calcularMax(a, b uint64) uint64 {
	if a < b {
		return b
	} else {
		return a
	}
}
