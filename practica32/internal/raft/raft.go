/*
* AUTOR: Rafael Tolosana Calasanz y Unai Arronategui
* Lucia Morales Rosa (816906) y Lizer Bernad Ferrando (779035)
* ASIGNATURA: 30221 Sistemas Distribuidos del Grado en Ingeniería Informática
*			Escuela de Ingeniería y Arquitectura - Universidad de Zaragoza
* FECHA: noviembre de 2024
* FICHERO: raft.go
* DESCRIPCIÓN: implementacion del raft sin fallos
 */

package raft

//
// API
// ===
// Este es el API que vuestra implementación debe exportar
//
// nodoRaft = NuevoNodo(...)
//   Crear un nuevo servidor del grupo de elección.
//
// nodoRaft.Para()
//   Solicitar la parado de un servidor
//
// nodo.ObtenerEstado() (yo, mandato, esLider)
//   Solicitar a un nodo de elección por "yo", su mandato en curso,
//   y si piensa que es el msmo el lider
//
// nodoRaft.SometerOperacion(operacion interface()) (indice, mandato, esLider)

// type AplicaOperacion

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
	"raft/internal/comun/rpctimeout"
)

const (
	// Constante para fijar valor entero no inicializado
	IntNOINICIALIZADO = -1

	//  false deshabilita por completo los logs de depuracion
	// Aseguraros de poner kEnableDebugLogs a false antes de la entrega
	kEnableDebugLogs = true

	// Poner a true para logear a stdout en lugar de a fichero
	kLogToStdout = false

	// Cambiar esto para salida de logs en un directorio diferente
	kLogOutputDir = "./logs_raft/"
)

const (
	Lider     = "lider"
	Candidato = "candidato"
	Seguidor  = "seguidor"
)

type TipoOperacion struct {
	Operacion string // La operaciones posibles son "leer" y "escribir"
	Clave     string
	Valor     string // en el caso de la lectura Valor = ""
}

// A medida que el nodo Raft conoce las operaciones de las  entradas de registro
// comprometidas, envía un AplicaOperacion, con cada una de ellas, al canal
// "canalAplicar" (funcion NuevoNodo) de la maquina de estados
type AplicaOperacion struct {
	Indice    int // en la entrada de registro
	Operacion TipoOperacion
}

// Tipo de dato Go que representa un solo nodo (réplica) de raft
type NodoRaft struct {
	Mux sync.Mutex // Mutex para proteger acceso a estado compartido

	// Host:Port de todos los nodos (réplicas) Raft, en mismo orden
	Nodos   	[]rpctimeout.HostPort
	Yo      	int 		// indice de este nodos en campo array "nodos"
	IdLider 	int
	NumeroVotos int 		// votos recibidos si es candidato
	Estado 		string  	//rol del nodo
	Latido 		chan bool  	//el nodo recibe el heartbeat
	SoySeguidor chan bool 	// el nodo vuelve a ser follower
	SoyLider 	chan bool 	// el nodo es o sigue siendo líder
	 
	Logger 		*log.Logger
	
	//Estado del nodo
	CurrentTerm	int 		// último mandato visto por el servidor
	VotedFor 	int 		// candidato que ha recibido el voto de este nodo
	CommitIndex int 		// índice de la Entry comprometida más alta
}

type Entry struct {
	Indice int
	Mandato int
	Operacion TipoOperacion
}

// Creacion de un nuevo nodo de eleccion
//
// Tabla de <Direccion IP:puerto> de cada nodo incluido a si mismo.
//
// <Direccion IP:puerto> de este nodo esta en nodos[yo]
//
// Todos los arrays nodos[] de los nodos tienen el mismo orden

// canalAplicar es un canal donde, en la practica 5, se recogerán las
// operaciones a aplicar a la máquina de estados. Se puede asumir que
// este canal se consumira de forma continúa.
//
// NuevoNodo() debe devolver resultado rápido, por lo que se deberían
// poner en marcha Gorutinas para trabajos de larga duracion
func NuevoNodo(nodos []rpctimeout.HostPort, yo int,
	canalAplicarOperacion chan AplicaOperacion) *NodoRaft {
	nr := &NodoRaft{}
	nr.Nodos = nodos
	nr.Yo = yo
	nr.IdLider = -1
	nr.NumeroVotos = 0
	nr.Estado = Seguidor
	nr.Latido = make(chan bool)
	nr.SoySeguidor = make(chan bool)
	nr.SoyLider = make(chan bool)
	nr.CurrentTerm = 0
	nr.VotedFor = -1
	nr.CommitIndex = 0

	if kEnableDebugLogs {
		nombreNodo := nodos[yo].Host() + "_" + nodos[yo].Port()
		logPrefix := fmt.Sprintf("%s", nombreNodo)

		fmt.Println("LogPrefix: ", logPrefix)

		if kLogToStdout {
			nr.Logger = log.New(os.Stdout, nombreNodo+" -->> ",
				log.Lmicroseconds|log.Lshortfile)
		} else {
			err := os.MkdirAll(kLogOutputDir, os.ModePerm)
			if err != nil {
				panic(err.Error())
			}
			logOutputFile, err := os.OpenFile(fmt.Sprintf("%s/%s.txt",
				kLogOutputDir, logPrefix), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				panic(err.Error())
			}
			nr.Logger = log.New(logOutputFile,
				logPrefix+" -> ", log.Lmicroseconds|log.Lshortfile)
		}
		nr.Logger.Println("logger initialized")
	} else {
		nr.Logger = log.New(ioutil.Discard, "", 0)
	}

	// Añadir codigo de inicialización
	go raftProtocol(nr)

	return nr
}

// Metodo Para() utilizado cuando no se necesita mas al nodo
//
// Quizas interesante desactivar la salida de depuracion
// de este nodo
func (nr *NodoRaft) para() {
	go func() { time.Sleep(5 * time.Millisecond); os.Exit(0) }()
}

// Devuelve "yo", mandato en curso y si este nodo cree ser lider
//
// Primer valor devuelto es el indice de este  nodo Raft el el conjunto de nodos
// la operacion si consigue comprometerse.
// El segundo valor es el mandato en curso
// El tercer valor es true si el nodo cree ser el lider
// Cuarto valor es el lider, es el indice del líder si no es él
func (nr *NodoRaft) obtenerEstado() (int, int, bool, int) {

	var yo int = nr.Yo
	var mandato int = nr.CurrentTerm
	var esLider bool
	var idLider int = nr.IdLider
	// Vuestro codigo aqui

	if nr.Yo == nr.IdLider {
		esLider = true
	} else {
		esLider = false
	}

	return yo, mandato, esLider, idLider
}

// El servicio que utilice Raft (base de datos clave/valor, por ejemplo)
// Quiere buscar un acuerdo de posicion en registro para siguiente operacion
// solicitada por cliente.

// Si el nodo no es el lider, devolver falso
// Sino, comenzar la operacion de consenso sobre la operacion y devolver en
// cuanto se consiga
//
// No hay garantia que esta operacion consiga comprometerse en una entrada de
// de registro, dado que el lider puede fallar y la entrada ser reemplazada
// en el futuro.
// Primer valor devuelto es el indice del registro donde se va a colocar
// la operacion si consigue comprometerse.
// El segundo valor es el mandato en curso
// El tercer valor es true si el nodo cree ser el lider
// Cuarto valor es el lider, es el indice del líder si no es él
func (nr *NodoRaft) someterOperacion(operacion TipoOperacion) (int, int,
	bool, int, string) {
	indice := -1
	mandato := -1
	esLider := nr.Yo == nr.IdLider
	idLider := -1
	valorADevolver := ""

	// Vuestro codigo aqui
	if esLider {
		indice = nr.CommitIndex
		mandato = nr.CurrentTerm
		entry := Entry{indice, mandato, operacion}
		nr.Logger.Printf("(%d, %d, %s, %s, %s)", entry.Indice, entry.Mandato, entry.Operacion.Operacion, entry.Operacion.Clave, entry.Operacion.Valor)
		
		var results Results
		confirmados := 0
		for i:=0; i<len(nr.Nodos); i++ {
			if i!= nr.Yo {
				nr.Nodos[i].CallTimeout("NodoRaft.AppendEntries", ArgAppendEntries{mandato, nr.Yo, entry, indice}, &results, 20*time.Millisecond)
			}
			if results.Success {
				confirmados++
			}
		}
		if confirmados > len(nr.Nodos)/2 {
			nr.CommitIndex++
		}
		idLider = nr.Yo
	}

	return indice, mandato, esLider, idLider, valorADevolver
}

// -----------------------------------------------------------------------
// LLAMADAS RPC al API
//
// Si no tenemos argumentos o respuesta estructura vacia (tamaño cero)
type Vacio struct{}

func (nr *NodoRaft) ParaNodo(args Vacio, reply *Vacio) error {
	defer nr.para()
	return nil
}

type EstadoParcial struct {
	Mandato int
	EsLider bool
	IdLider int
}

type EstadoRemoto struct {
	IdNodo int
	EstadoParcial
}

func (nr *NodoRaft) ObtenerEstadoNodo(args Vacio, reply *EstadoRemoto) error {
	reply.IdNodo, reply.Mandato, reply.EsLider, reply.IdLider = nr.obtenerEstado()
	return nil
}

type ResultadoRemoto struct {
	ValorADevolver string
	IndiceRegistro int
	EstadoParcial
}

func (nr *NodoRaft) SometerOperacionRaft(operacion TipoOperacion,
	reply *ResultadoRemoto) error {
	reply.IndiceRegistro, reply.Mandato, reply.EsLider,
		reply.IdLider, reply.ValorADevolver = nr.someterOperacion(operacion)
	return nil
}

// -----------------------------------------------------------------------
// LLAMADAS RPC protocolo RAFT
//
// Structura de ejemplo de argumentos de RPC PedirVoto.
//
// Recordar
// -----------
// Nombres de campos deben comenzar con letra mayuscula !
type ArgsPeticionVoto struct {
	// Vuestros datos aqui
	Term         int
	CandidateId  int
}

// Structura de ejemplo de respuesta de RPC PedirVoto,
//
// Recordar
// -----------
// Nombres de campos deben comenzar con letra mayuscula !
type RespuestaPeticionVoto struct {
	// Vuestros datos aqui
	Term        int
	VoteGranted bool
}

// Metodo para RPC PedirVoto
func (nr *NodoRaft) PedirVoto(peticion *ArgsPeticionVoto,
	reply *RespuestaPeticionVoto) error {
	
	if peticion.Term < nr.CurrentTerm {
		reply.Term = nr.CurrentTerm
		reply.VoteGranted = false
	} else if peticion.Term == nr.CurrentTerm && peticion.CandidateId != nr.VotedFor {
		reply.Term = nr.CurrentTerm
		reply.VoteGranted = false
	} else if peticion.Term > nr.CurrentTerm {
		nr.CurrentTerm = peticion.Term
		nr.VotedFor = peticion.CandidateId
		reply.Term = nr.CurrentTerm
		reply.VoteGranted = true

		if nr.Estado == Candidato || nr.Estado == Lider {
			nr.SoySeguidor <- true
		}
	}
	return nil
}

type ArgAppendEntries struct {
	Term     int
	LeaderId int
	Entries  Entry
	LeaderCommit int
}

type Results struct {
	Term    int  // Mandato actual
	Success bool // Si el seguidor tiene PrevLogIndex y Prevlogterm devuelve true
}

// Metodo de tratamiento de llamadas RPC AppendEntries
func (nr *NodoRaft) AppendEntries(args *ArgAppendEntries,
	results *Results) error {
	
	if args.Entries == (Entry{}){

		if args.Term < nr.CurrentTerm {
			results.Term = nr.CurrentTerm
			} else if args.Term == nr.CurrentTerm {
		nr.IdLider = args.LeaderId
		results.Term = nr.CurrentTerm
		nr.Latido <- true
		} else {
			nr.IdLider = args.LeaderId
			nr.CurrentTerm = args.Term
			results.Term = nr.CurrentTerm
			if nr.Estado == Lider {
			// Si era líder o candidato vuelvo a ser follower
			nr.SoySeguidor <- true
			} else {
				// Si era follower continúa el protocolo de manera normal, y si
				// era candidato recibe heartbeat del nuevo líder
				nr.Latido <- true
			}
		}
	} else {
		// Se introduce nueva entrada en el log
		nr.Logger.Printf("(%d, %d, %s, %s, %s)", args.Entries.Indice,
			args.Entries.Mandato, args.Entries.Operacion.Operacion,
			args.Entries.Operacion.Clave, args.Entries.Operacion.Valor)
		results.Success = true
	}
	return nil
}

// ----- Metodos/Funciones a utilizar como clientes
//
//

// Ejemplo de código enviarPeticionVoto
//
// nodo int -- indice del servidor destino en nr.nodos[]
//
// args *RequestVoteArgs -- argumentos par la llamada RPC
//
// reply *RequestVoteReply -- respuesta RPC
//
// Los tipos de argumentos y respuesta pasados a CallTimeout deben ser
// los mismos que los argumentos declarados en el metodo de tratamiento
// de la llamada (incluido si son punteros
//
// Si en la llamada RPC, la respuesta llega en un intervalo de tiempo,
// la funcion devuelve true, sino devuelve false
//
// la llamada RPC deberia tener un timout adecuado.
//
// Un resultado falso podria ser causado por una replica caida,
// un servidor vivo que no es alcanzable (por problemas de red ?),
// una petición perdida, o una respuesta perdida
//
// Para problemas con funcionamiento de RPC, comprobar que la primera letra
// del nombre  todo los campos de la estructura (y sus subestructuras)
// pasadas como parametros en las llamadas RPC es una mayuscula,
// Y que la estructura de recuperacion de resultado sea un puntero a estructura
// y no la estructura misma.
func (nr *NodoRaft) enviarPeticionVoto(nodo int, args *ArgsPeticionVoto,
	reply *RespuestaPeticionVoto) bool {
	
	err := nr.Nodos[nodo].CallTimeout("NodoRaft.PedirVoto", args, reply, 20*time.Millisecond)
	if err != nil {
		return false
	} else {
		if reply.Term > nr.CurrentTerm {
		// Si pido el voto a un nodo con mayor mandato, dejo de ser
		// candidato y vuelvo a ser follower
		nr.CurrentTerm = reply.Term
		nr.SoySeguidor <- true
		} else if reply.VoteGranted {
			// Si me dan el voto compruebo si tengo mayoría simple, en cuyo caso
			// me convierto en líder
			nr.NumeroVotos++
			if nr.NumeroVotos > len(nr.Nodos)/2 {
				nr.SoyLider <- true
			}
		}
		return true
	}
 
}

// nodo int  índice del servidor destino en nr.nodos[]
//
// args *ArgAppendEntries  argumentos para la llamada RPC
//
// results *Result  respuesta RPC
//
// Si en la llamada RPC, la respuesta llega en un intervalo de tiempo,
// la funcion devuelve true, si no devuelve false. La llamada RPC deberia
// tener un timout adecuado
//
// Un resultado falso podria ser causado por una réplica caída, un servidor vivo
// que no es alcanzable (por problemas de red), una petición perdida, o una
// respuesta perdida
func (nr *NodoRaft) enviarHeartbeat(nodo int, args *ArgAppendEntries, results *Results) bool {
	err := nr.Nodos[nodo].CallTimeout("NodoRaft.AppendEntries", args, results, 20*time.Millisecond)
	if err != nil {
		return false
	} else {
		if results.Term > nr.CurrentTerm {
			// Si he enviado heartbeat a un nodo con mayor mandato dejo de ser
			// líder, actualizo mi mandato y vuelvo a ser follower
			nr.CurrentTerm = results.Term
			nr.IdLider = -1
			nr.SoySeguidor <- true
		}
		return true
	}
}

func raftProtocol(nr *NodoRaft) {
	for {
		for nr.Estado == Seguidor {
			select {
			case <-nr.Latido:
				//Si recibe heartbeat continúo siendo follower
				nr.Estado = Seguidor
			case <-time.After(getRandomTimeout()):
				// Si expira timeout se inicia el proceso de elección de líder
				nr.IdLider = -1
				nr.Estado = Candidato
			}
		}
		for nr.Estado == Candidato {
			nr.CurrentTerm++
			nr.VotedFor = nr.Yo
			nr.NumeroVotos = 1
			// Se inicia el timer de elección y se envían RequestVote al resto de nodos
			timer := time.NewTimer(getRandomTimeout())
			requestVotes(nr)
			select {
			case <-nr.Latido:
				//Si recibe heartbeat de nuevo líder vuelve a ser follower
				nr.Estado = Seguidor
			case <-nr.SoySeguidor:
				//Si el nodo descubre que hay otro con mayor mandato vuelve a ser follower
				nr.Estado = Seguidor
			case <-timer.C:
				//Si expira el timeout se inicia una nueva elección
				nr.Estado = Candidato
			case <-nr.SoyLider:
				//Si el nodo recibe mayoría simple de votos pasa a ser el nuevo líder
				nr.Estado = Lider
			}
		}
		for nr.Estado == Lider {
			nr.IdLider = nr.Yo
			timer := time.NewTimer(50 * time.Millisecond)
			sendHeartbeats(nr)
			select {
				case <-nr.SoySeguidor:
					// Si el nodo descubre que hay otro con mayor mandato vuelve a ser follower
					nr.Estado = Seguidor
				case <-timer.C:
					// Se vuelven a enviar heartbeats pasados 50 milisegundos
					nr.Estado = Lider
			}
		}
	}
}

// Se envían peticiones de voto al resto de nodos Raft de manera concurrente
func requestVotes(nr *NodoRaft) {
	var reply RespuestaPeticionVoto
	for i := 0; i < len(nr.Nodos); i++ {
		if i != nr.Yo {
			go nr.enviarPeticionVoto(i, &ArgsPeticionVoto{nr.CurrentTerm, nr.Yo}, &reply)
		}
	}
}
 
// Se envían heartbeats al resto de nodos Raft de manera concurrente mientras el nodo siga siendo líder
func sendHeartbeats(nr *NodoRaft) {
	var results Results
	for i := 0; i < len(nr.Nodos); i++ {
		if i != nr.Yo {
			go nr.enviarHeartbeat(i, &ArgAppendEntries{nr.CurrentTerm, nr.Yo, Entry{}, nr.CommitIndex}, &results)
		}
	}
}

// Devuelve un timeout aleatorio entre 100 y 500 ms
func getRandomTimeout() time.Duration {
	return time.Duration(rand.Intn(400)+100) * time.Millisecond
}