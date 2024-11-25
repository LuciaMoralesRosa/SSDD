/*
* AUTOR: Rafael Tolosana Calasanz
* 		 Lucia Morales Rosa (816906) y Lizer Bernad Ferrando (779035)
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: octubre de 2024
* FICHERO: ra.go
* DESCRIPCIÓN: Implementación del algoritmo de Ricart-Agrawala Generalizado en Go
 */
package ra

import (
	"practica2/com"
	"practica2/gf"
	"practica2/ms"
	"strconv"
	"sync"

	"github.com/DistributedClocks/GoVector/govec/vclock"
)

const nProcesos = 6 // Numero de procesos en ejecucion

type Request struct {
	Clock     vclock.VClock // Reloj vectorial
	Pid       int           // Identificador del proceso
	Operacion string        // Tipo de operacion: "Leer" o "Escribir"
}

type Reply struct{}

type ActualizarFichero struct {
	Pid   int    // Identificador del proceso
	Texto string // Texto a escribir en los ficheros
}

type AckActualizar struct{}

type RASharedDB struct {
	OurSeqNum vclock.VClock // Reloj vectorial
	HigSeqNum vclock.VClock // Reloj vectorial
	OutRepCnt int           // Respuestas recibidas
	ReqCS     bool          // Solicita acceso a SC
	RepDefd   []bool        // Inidica si hay peticion de SC pendiente
	ms        *ms.MessageSystem
	done      chan bool  // Canal para indicar la finalizacion
	chrep     chan bool  // Canal para notificar recepcion de respuestas
	Mutex     sync.Mutex // mutex para proteger concurrencia sobre las variables

	// TODO: completar
	peticion      chan Request       // Canal de peticiones
	respuesta     chan Reply         // Canal de respuestas
	ackActualizar chan AckActualizar // Canal de acks de actualizacion
	Operacion     string             // Leer o Escribir
	Fichero       *gf.Fichero        // Objeto fichero del gestor de ficheros
	yo            string             // Identificador de proceso
}

var matrizExclusion = [2][2]bool{
	{false, true}, // Lector - Lector -> no bloquear SC
	{true, true},  // Escritor - x -> bloquear acceso a SC
}

// Inicializa el RA
func New(me int, usersFile string, op string) *RASharedDB {
	com.Depuracion("RA - New: inicio")

	messageTypes := []ms.Message{Request{}, Reply{}, ActualizarFichero{},
		AckActualizar{}}
	msgs := ms.New(me, usersFile, messageTypes)

	yo := strconv.Itoa(me)

	// Creacion del fichero para lectura y escritura
	fichero := gf.CrearFichero("fichero_" + yo + ".txt")

	// Inicializacion de los relojes vectoriales
	reloj1 := vclock.New()
	reloj1.Set(yo, 0)
	reloj2 := vclock.New()
	reloj2.Set(yo, 0)

	// Inicializacion de la estructura RASharedDB
	ra := RASharedDB{reloj1, reloj2, 0, false, make([]bool, nProcesos), &msgs,
		make(chan bool), make(chan bool), sync.Mutex{}, make(chan Request),
		make(chan Reply), make(chan AckActualizar), op, fichero, yo}

	// Lanzar goroutinas
	com.Depuracion("RA - New: lanzando goroutinas")
	go recibirMensaje(&ra)
	go manejoPeticionesSC(&ra)
	go manejoRespuestas(&ra)

	return &ra
}

// Pre: Verdad
// Post: Realiza el PreProtocol del algoritmo de Ricart-Agrawala Generalizado
func (ra *RASharedDB) PreProtocol() {
	com.Depuracion("RA - Preprotocol: inicio")

	// Mutex para evitar condiciones de carrera con otras goroutinas
	ra.Mutex.Lock()

	// Indico que quiero entrar a la SC
	ra.ReqCS = true

	// Incrementar numero de secuencia y actualizacion
	ra.OurSeqNum[ra.yo] = ra.HigSeqNum[ra.yo] + 1
	// Actualizar los valores de relojes mayores conocidos de otros procesos
	ra.OurSeqNum.Merge(ra.HigSeqNum)

	ra.Mutex.Unlock()

	// Indicar que se esperan n-1 respuestas
	ra.OutRepCnt = nProcesos - 1
	for i := 1; i <= nProcesos; i++ {
		if i != ra.ms.Me {
			peticion := Request{
				Pid:       ra.ms.Me,
				Clock:     ra.OurSeqNum,
				Operacion: ra.Operacion,
			}
			com.Depuracion("RA - Preprotocol: Enviando peticion a " +
				strconv.Itoa(i))
			ra.ms.Send(i, peticion)
		}
	}

	// Esperamos a que llegue un true, ie, obtener todas las respuestas
	<-ra.chrep
	com.Depuracion("RA - Preprotocol: final")
	com.Depuracion("Acceso a seccion critica con reloj = " + ra.OurSeqNum.ReturnVCString())
}

// Pre: Verdad
// Post: Realiza  el  PostProtocol  para el  algoritmo de Ricart-Agrawala
// Generalizado
func (ra *RASharedDB) PostProtocol() {
	com.Depuracion("RA - Posprotocol: inicio")

	// Mutex para evitar condiciones de carrera con otras goroutinas
	ra.Mutex.Lock()
	// Indico que ya no quiero tener acceso a la SC
	ra.ReqCS = false
	ra.Mutex.Unlock()

	// Notificar liberacion de la SC
	for i := 1; i <= nProcesos; i++ {
		com.Depuracion("RA - Posprotocol: avisando de liberacion de SC")
		if ra.RepDefd[i-1] {
			ra.Mutex.Lock()
			ra.RepDefd[i-1] = false // Se indica que se responde
			// Envio de respuesta notificiando que se ha liberado SC
			ra.ms.Send(i, Reply{})
			ra.Mutex.Unlock()
		}
	}
}

// Pre: True
// Post: Funcion para la finalizacion del Ricart-Agrawala
func (ra *RASharedDB) Stop() {
	ra.ms.Stop()
	ra.done <- true
}

// Pre: True
// Post: Devuelve true si el proceso con pid "yo" tiene prioridad sobre el
// el proceso con pid "otro"
func tengoPrioridad(reloj1 vclock.VClock, reloj2 vclock.VClock, yo int, otro int) bool {
	// Compara si reloj1 es un ancestro de reloj2 (reloj1 tiene prioridad)
	if reloj1.Compare(reloj2, vclock.Descendant) {
		// Si reloj1 es un ancestro de reloj2, el proceso "yo" tiene prioridad
		return true
	} else if reloj1.Compare(reloj2, vclock.Concurrent) { // Si son iguales
		// Si no pueden determinar un orden claro, se comparan los pid
		return yo < otro
	} else {
		// Si reloj1 no es un ancestro de reloj2, ni los relojes son
		// concurrentes, se asume que reloj2 tiene prioridad
		return false
	}
}

// Escucha y procesa los mensajes recibidos por el proceso. Dependiendo del tipo
// de mensaje recibido, lo procesa de distintas formas.
func recibirMensaje(ra *RASharedDB) {
	for {
		mensaje := ra.ms.Receive()
		switch tipoMensaje := mensaje.(type) {
		case Request:
			com.Depuracion("RA - recibirMensaje: Peticion recibida")
			ra.peticion <- tipoMensaje
		case Reply:
			com.Depuracion("RA - recibirMensaje: Respuesta recibida")
			ra.respuesta <- tipoMensaje
		case ActualizarFichero:
			com.Depuracion("RA - recibirMensaje: Actualizar recibido")
			ra.Fichero.Escribir(tipoMensaje.Texto)
			com.Depuracion("RA - recibirMensaje: Texto escrito en fichero")
			ra.ms.Send(tipoMensaje.Pid, AckActualizar{})
			com.Depuracion("RA - recibirMensaje: Mensaje de actualizacion enviado")
		case AckActualizar:
			com.Depuracion("RA - recibirMensaje: Ack recibido")
			ra.ackActualizar <- tipoMensaje
		}
	}
}

// Pre: Verdad
// Post: Cuando llega una peticion:
//   - Si el proceso tiene prioridad envia la peticion.
//   - Si no tiene prioridad o no pide acceso a SC se pospone la peticion
func manejoPeticionesSC(ra *RASharedDB) {
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

		com.Depuracion("Actualizacion de reloj \n" + ra.OurSeqNum.ReturnVCString())

		// Veo si pospongo la peticion
		soyEscritor := boolToInt(ra.Operacion == "Escribir")
		esEscritor := boolToInt(peticion.Operacion == "Escribir")

		posponer := ra.ReqCS && tengoPrioridad(ra.OurSeqNum, peticionReloj,
			ra.ms.Me, peticion.Pid) && matrizExclusion[soyEscritor][esEscritor]
		ra.Mutex.Unlock()

		if posponer {
			com.Depuracion("RA - manejoPeticiones: Peticion pospuesta")
			ra.RepDefd[peticion.Pid-1] = true // Posponemos
		} else {
			com.Depuracion("RA - manejoPeticiones: Peticion no pospuesta")
			ra.Mutex.Lock()
			com.Depuracion("RA - manejoPeticiones: Enviar mensaje peticion")
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
			com.Depuracion("RA - manejoRespuestas: Todas las respuestas recibidas")
			ra.chrep <- true // Notificar al canal
		}
	}
}

// Pre: texto es una cadena de texto valida
// Post: Envia un mensaje de actualizacion a los procesos y espera nProcesos-1
// respuestas
func (ra *RASharedDB) AvisarActualizar(texto string) {
	com.Depuracion("RA - AvisarActualizar: se va a enviar actualizar")

	// Notificar a todos los procesos de que deben actualizar su fichero
	yo := ra.ms.Me
	for i := 1; i <= nProcesos; i++ {
		if i != yo {
			com.Depuracion("RA - AvisarActualizar: enviando actualizar a " +
				strconv.Itoa(i))
			ra.ms.Send(i, ActualizarFichero{yo, texto})
		}
	}

	// Esperar a que todos confirmen que han actualizado su fichero
	respuestasEsperadas := nProcesos - 1
	com.Depuracion("RA - AvisarActualizar: Respuestas esperadas de ack = " +
		strconv.Itoa(respuestasEsperadas))
	for respuestasEsperadas > 0 {
		<-ra.ackActualizar
		respuestasEsperadas--
	}

	com.Depuracion("RA - AvisarActualizar: se han recibido todos los acks -> " +
		"todos los ficheros han sido actualizados")
}

// Calcula el maximo de los dos parametros
func calcularMax(a, b uint64) uint64 {
	if a < b {
		return b
	} else {
		return a
	}
}

// Convierte true en 1 y false en 0
func boolToInt(condicion bool) int {
	if condicion {
		return 1
	} else {
		return 0
	}
}
