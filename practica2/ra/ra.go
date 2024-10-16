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
    "ms"
    "sync"
	"practica2/ms"
	"practica2/gestor"
	"github.com/DistributedClocks/GoVector/govec"
)

type Request struct{
    Clock   []byte // Marca temporal vectorial
    Pid     int // PID del proceso solicitante
	PeticionEscritura bool // Indica si es peticion de escritura (true) o lectura (false)
	NumeroSecuencia	int // Numero de secuencia
}

type Reply struct{
	Clock	[]byte // Marca temporal vectorial
}

type Escribir struct {
	Texto	string // Contenido a escribir
	Clock	[]byte // Marca temporal vectorial
}

type RASharedDB struct {
	nodes			int	// Numero de nodos
    OurSeqNum   	int // Numero de secuencia actual
    HigSeqNum   	int // Numero de secuencia mas alto observado
    OutRepCnt   	int // Respuestas externas pendientes
    ReqCS       	boolean // Indica si el proceso esta solicitando acceso a SC
    RepDefd     	bool[] //
    ms          	*MessageSystem // Gestor de mensajes
    done        	chan bool // Canal para el manejo de la terminacion
    chrep       	chan bool // Canal para el manejo de respuestas

	// Para lectores y escritores
	procesoEscritor	bool // Indica si el proceso es escritor (true) o lector (false)
	gestor			*Gestor // Gestor de operaciones de lectura y escritura
	// Para relojes vectoriales
	logger			*govec.GoLog // Manejo de relojes vectoriales y trazabilidad distribuida
	// mutex para proteger concurrencia sobre las variables
    Mutex       sync.Mutex
}


//[0][0] -> lector lector
//[0][1] -> lector escritor
//[1][0] -> escritor lector
//[1][1] -> escritor escritor

var matrizExclusion = [2][2] bool {
	{false, true}, // un lector no bloauea a otros lectores pero un escritor bloquea a todos
	{true, true} // Un escritor bloquea tanto a lectores como a escritores
}

// Inicializa una nueva instancia de RASharedBD
func New(me int, usersFile string, esEscritor bool, gestor *Gestor, logger *govec.GoLog) (*RASharedDB) {
    messageTypes := []Message{Request, Reply, Escribir} // Tipos de mensajes
    msgs = ms.New(me, usersFile string, messageTypes) 
	nodes = contarLineas(usersFile)
	// Inicializa la estructura
    ra := RASharedDB{nodes, 0, 0, 0, false, []int{}, &msgs, make(chan bool),
		make(chan bool), esEscritor, gestor, logger, &sync.Mutex{}}
    
	go recibirMensaje
    return &ra
}

// Gestiona los mensajes entrantes. Constantemente esta esperando mensajes entrantes.
func recibirMensaje(){
	for {
		msg := ms.Receive()
		switch x := msg.(type) {
			case Request:
				// Ha llegado una peticion (tiene Clock, pid solicitante, escritura?)
				ra.peticionRecibida(x)
			case Reply:
			case Escribir:
		}
	}
}

func peticionRecibida(mensaje Request){
	ra.Mutex.Lock() // Vamos a bloquear
	ra.HigSeqNum = Max(ra.HigSeqNum, mensaje.NumeroSecuencia)

	// Se pospone la peticion si: 
	//	1 - Mi numero de secuencia es menor al numero de secuencia del mensaje
	//	2 - Los numero de secuencia son iguales pero mi pid es menor
	posponerPeticion := ra.ReqCS && (ra.OurSeqNum < mensaje.NumeroSecuencia || 
		(ra.OurSeqNum == mensaje.NumeroSecuencia && ra.ms.Me() < mensaje.pid))
	
	ra.Mutex.Unlock()

	// Si se pospone la peticion y soy un proceso escritor o la peticion es de escritura
	if posponerPeticion && (matrizExclusion[ra.procesoEscritor][mensaje.PeticionEscritura]){
		ra.Mutex.Lock()
		// Indicamos que se ha pospuesto un mensaje del proceso pid
		ra.RepDefd[mensaje.pid - 1] = true 
		ra.Mutex.Unlock()
	}
	else{
		// Si no se pospone o es un proceso de lectura y una peticion de lectura
		envioPreparado := ra.logger.PrepareSend("Enviando respuesta", "Enviar respuesta", govec.GetDefaultLogOptions())
	}
	ms.Send(mesnaje.pid, Reply{})
}


//Pre: Verdad
//Post: Realiza  el  PreProtocol  para el  algoritmo de
//      Ricart-Agrawala Generalizado
// Proceso que isgue in proceso cuando quiere acceder a la seccion critica
func (ra *RASharedDB) PreProtocol(){
	for(int i = 0; ; i++){
		ms.Send(i, Request{me})
	}
    
}

//Pre: Verdad
//Post: Realiza  el  PostProtocol  para el  algoritmo de
//      Ricart-Agrawala Generalizado
// El proceso libera la seccion critica y notifica a todos los procesos
func (ra *RASharedDB) PostProtocol(){

}

func (ra *RASharedDB) Stop(){
    ra.ms.Stop()
    ra.done <- true
}

func contarLineas(ruta string) (int) {
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

	return lineas, nil
}
