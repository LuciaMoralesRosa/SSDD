// Escribir vuestro código de funcionalidad Raft en este fichero
//

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

	//"crypto/rand"
	"sync"
	"time"

	//"net/rpc"

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

// MODIFICADO
type LogEntry struct {
	Operacion TipoOperacion
	Term      int
}

// Tipo de dato Go que representa un solo nodo (réplica) de raft
type NodoRaft struct {
	Mux sync.Mutex // Mutex para proteger acceso a estado compartido

	// Host:Port de todos los nodos (réplicas) Raft, en mismo orden
	Nodos   []rpctimeout.HostPort
	Yo      int // indice de este nodos en campo array "nodos"
	IdLider int
	// Utilización opcional de este logger para depuración
	// Cada nodo Raft tiene su propio registro de trazas (logs)
	Logger *log.Logger

	// Vuestros datos aqui
	// mirar figura 2 para descripción del estado que debe mantener un nodo Raft

	//MODIFICADO
	Estado string // Puede ser "Lider", "Seguidor" o "Candidato"

	// Persistent state on all servers
	CurrentTerm int // Latest term server has seen
	VotedFor    int // CandidateId that received vote in current term
	Log         []LogEntry

	// Volatile state on all servers
	CommitIndex int // Index of highest log entry known to be committed
	LastApplied int // Index of highest log entry applied to state machine

	// Volatile state on leaders
	NextIndex  map[int]int
	MatchIndex map[int]int

	// Canales empleados
	//chLatido   chan bool
	chSeguidor chan bool
	chLider    chan bool

	NumVotos int
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

	//MODIFICADO
	nr.Estado = "Seguidor"
	nr.CurrentTerm = 0
	nr.VotedFor = -1
	nr.CommitIndex = -1
	nr.LastApplied = -1
	nr.NextIndex = make(map[int]int)
	nr.MatchIndex = make(map[int]int)
	nr.NumVotos = 0

	// Canales
	//nr.chLatido = make(chan bool)
	nr.chSeguidor = make(chan bool)
	nr.chLider = make(chan bool)
	// FIN MODIFICADO

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
	//MODIFICADO
	go nr.raftProtocol()

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
	var mandato int
	var esLider bool
	var idLider int = nr.IdLider

	// Vuestro codigo aqui
	//MODIFICADO
	mandato = nr.CurrentTerm
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
	EsLider := false
	idLider := -1
	valorADevolver := ""

	// Vuestro codigo aqui
	// MODIFICADO
	valorADevolver = "error"
	idLider = nr.IdLider

	if nr.Yo == nr.IdLider {
		EsLider = true
		//idLider = nr.Yo
		indice = nr.CommitIndex
		mandato = nr.CurrentTerm

		nr.Log = append(nr.Log, LogEntry{Operacion: operacion, Term: nr.CurrentTerm})
		nr.Logger.Printf("Mandato %d, Operacion %v\n", nr.CurrentTerm, operacion)
		fmt.Printf("Soy %d y he recibido una operacion\n", nr.Yo)
		// ALERTA!! - respond after entry applied to state machine
		var results Results
		confirmados := 1
		for i := 0; i < len(nr.Nodos); i++ {
			if i != nr.Yo {
				args := ArgAppendEntries{
					Term:         mandato,
					LeaderId:     nr.Yo,
					PrevLogIndex: indice,
					PrevLogTerm:  nr.CurrentTerm,
					Entries:      []LogEntry{{operacion, nr.CurrentTerm}},
					LeaderCommit: -1,
				}
				nr.Nodos[i].CallTimeout("NodoRaft.AppendEntries", &args, &results, 33*time.Millisecond)
				if results.Success {
					confirmados++
					// Actualizar nextIndex
					nr.NextIndex[i] = nr.NextIndex[i] + len(args.Entries)
					nr.MatchIndex[i] = nr.NextIndex[i] - 1
				} else {
					// Si pertenezco a un mandato anterior, actualizo el mandato y dejo de ser líder
					if nr.CurrentTerm < results.Term {
						nr.CurrentTerm = results.Term
						nr.chSeguidor <- true
					}
				}
			}
		}
		if confirmados > len(nr.Nodos)/2 {
			nr.CommitIndex++
			valorADevolver = "ok"
			fmt.Printf("Operación comprometida\n")
		}
	}

	return indice, mandato, EsLider, idLider, valorADevolver
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
	// MODIFICADO
	Term         int
	CandidateId  int
	LastLogIndex int //index of candidate’s last log entry
	LastLogTerm  int //term of candidate’s last log entry
}

// Structura de ejemplo de respuesta de RPC PedirVoto,
//
// Recordar
// -----------
// Nombres de campos deben comenzar con letra mayuscula !
type RespuestaPeticionVoto struct {
	// Vuestros datos aqui
	// MODIFICADO
	Term        int
	VoteGranted bool
}

// Metodo para RPC PedirVoto
func (nr *NodoRaft) PedirVoto(peticion *ArgsPeticionVoto,
	reply *RespuestaPeticionVoto) error {
	// Vuestro codigo aqui
	// MODIFICADO
	nr.Mux.Lock()
	lastLogIndex, lastLogTerm := nr.lastLogIndexAndTerm()
	nr.Mux.Unlock()
	nr.Logger.Printf("El nodo %d me pide su voto: [Current term: %d, Log index/term: (%d, %d)]",
		peticion.CandidateId,
		peticion.Term,
		peticion.LastLogIndex,
		peticion.LastLogTerm)
	nr.Logger.Printf("Mi estado: [Current term: %d, VotedFor: %d, Log index/term: (%d, %d)]",
		nr.CurrentTerm,
		nr.VotedFor,
		lastLogIndex,
		lastLogTerm)

	// If RPC request or response contains term T > currentTerm, set currentTerm = T, convert to follower
	if peticion.Term > nr.CurrentTerm {
		nr.chSeguidor <- true
		nr.CurrentTerm = peticion.Term
		nr.VotedFor = -1
	}

	if peticion.Term == nr.CurrentTerm &&
		(nr.VotedFor == -1 || nr.VotedFor == peticion.CandidateId) &&
		(peticion.LastLogTerm > lastLogTerm || (peticion.LastLogTerm == lastLogTerm && peticion.LastLogIndex >= lastLogIndex)) {
		reply.VoteGranted = true
		nr.VotedFor = peticion.CandidateId
		// Reset contador elección
	} else {
		reply.VoteGranted = false
	}
	reply.Term = nr.CurrentTerm
	return nil
}

type ArgAppendEntries struct {
	// Vuestros datos aqui
	// MODIFICADO
	Term     int
	LeaderId int

	PrevLogIndex int //index of log entry immediately preceding new ones
	PrevLogTerm  int //term of prevLogIndex entry
	Entries      []LogEntry
	LeaderCommit int
}

type Results struct {
	// Vuestros datos aqui
	// MODIFICADO
	Term    int  // Mandato actual
	Success bool // Si el seguidor tiene PrevLogIndex y Prevlogterm devuelve true
}

// Metodo de tratamiento de llamadas RPC AppendEntries
func (nr *NodoRaft) AppendEntries(args *ArgAppendEntries,
	results *Results) error {
	// Completar....
	// MODIFICADO
	nr.Logger.Printf("Soy %d y he recibido un AppendEntries de %d con mandato %d y entradas %v\n",
		nr.Yo, args.LeaderId, args.Term, args.Entries)
	// Si mi mandato es inferior al de la petición del líder
	if nr.CurrentTerm < args.Term {
		nr.chSeguidor <- true
		nr.CurrentTerm = args.Term
		//nr.VotedFor = -1 <- Esto para generalizar en una función
	}

	results.Success = false
	if nr.CurrentTerm == args.Term {
		nr.chSeguidor <- true

		nr.Logger.Printf("Los mandatos son iguales\n")

		// Si no hemos registrado ninguna entrada o
		// si nos quedan entradas por registrar en este mandato
		if args.PrevLogIndex == -1 ||
			(args.PrevLogIndex < len(nr.Log) && args.PrevLogTerm == nr.Log[args.PrevLogIndex].Term) {
			results.Success = true

			nr.Logger.Printf("El resultado va a ser exitoso\n")

			// Hallar un punto de inserción
			logInsertIndex := args.PrevLogIndex + 1
			newEntriesIndex := 0
			// Al final del bucle,
			// - logInsertIndex apunta al final del log o al índice en el que se deja de coincidir con alguna entrada del líder
			// - newEntriesIndex apunta al final de Entries o a un índice donde el mandato no coincide con la entrada correspondiente
			for {
				if logInsertIndex >= len(nr.Log) || newEntriesIndex >= len(args.Entries) {
					break
				}
				if nr.Log[logInsertIndex].Term != args.Entries[newEntriesIndex].Term {
					break
				}
				logInsertIndex++
				newEntriesIndex++
			}

			if newEntriesIndex < len(args.Entries) {
				nr.Logger.Printf("Insertando entradas %v desde el índice %d\n", args.Entries[newEntriesIndex:], logInsertIndex)
				nr.Log = append(nr.Log[:logInsertIndex], args.Entries[newEntriesIndex:]...)
				//nr.Logger.Printf("Log is now: %v", nr.Log)
			}

			// Set commit index
			if args.LeaderCommit > nr.CommitIndex {
				nr.CommitIndex = min(args.LeaderCommit, len(nr.Log)-1)
				//nr.newCommitReadyChan <- struct{}{}
			}
		}
	}

	results.Term = nr.CurrentTerm

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

	// Completar....
	// MODIFICADO

	nr.Mux.Lock()
	defer nr.Mux.Unlock()
	err := nr.Nodos[nodo].CallTimeout("NodoRaft.PedirVoto", args, reply, 33*time.Millisecond)
	if err != nil {
		return false
	} else {
		if reply.Term > nr.CurrentTerm {
			// Si el que me responde tiene mandato mayor, actualizado mi mandato al suyo y paso a ser seguidor
			nr.CurrentTerm = reply.Term
			nr.VotedFor = -1
			nr.chSeguidor <- true
		} else if reply.VoteGranted {
			nr.NumVotos++
			// Comprobamos si tenemos mayoría
			if nr.NumVotos > len(nr.Nodos)/2 {
				for i := 0; i < len(nr.Nodos); i++ {
					if i != nr.Yo {
						nr.NextIndex[i] = len(nr.Log)
						nr.MatchIndex[i] = -1
					}
				}
				nr.chLider <- true
			}

		}
	}

	return true
}

func (nr *NodoRaft) empezarEleccion() {
	nr.Logger.Printf("Soy %d y empiezo elección", nr.Yo)
	nr.CurrentTerm++
	nr.VotedFor = nr.Yo
	nr.Mux.Lock()
	lastLogIndex, lastLogTerm := nr.lastLogIndexAndTerm()
	nr.Mux.Unlock()
	nr.NumVotos = 1
	peticion := ArgsPeticionVoto{
		nr.CurrentTerm,
		nr.Yo,
		lastLogIndex,
		lastLogTerm,
	}
	var respuesta RespuestaPeticionVoto
	for i := 0; i < len(nr.Nodos); i++ {
		if i != nr.Yo {
			go nr.enviarPeticionVoto(i, &peticion, &respuesta)
		}
	}
}

func (nr *NodoRaft) enviarLatido(nodo int, nextIndex int, args *ArgAppendEntries,
	results *Results) {
	err := nr.Nodos[nodo].CallTimeout("NodoRaft.AppendEntries", args, results, 33*time.Millisecond)
	if err != nil {
		return
	} else {
		if results.Term > nr.CurrentTerm {
			// Si he enviado heartbeat a un nodo con mayor mandato dejo de ser
			// líder, actualizo mi mandato y vuelvo a ser follower
			nr.CurrentTerm = results.Term
			nr.VotedFor = -1
			nr.chSeguidor <- true
			return
		}

		if results.Success {
			nr.Mux.Lock()
			nr.NextIndex[nodo] = nextIndex + len(args.Entries)
			nr.MatchIndex[nodo] = nr.NextIndex[nodo] - 1
			nr.Mux.Unlock()
			// ALERTA <- Bucle por cada nueva entrada comprometida para actualizar el commit index
		}
	}
}

func (nr *NodoRaft) enviarLatidos() {
	nr.Logger.Printf("Soy %d y envío latidos", nr.Yo)
	var results Results
	for i := 0; i < len(nr.Nodos); i++ {
		if i != nr.Yo {
			ni := nr.NextIndex[i]
			prevLogIndex := ni - 1
			prevLogTerm := -1
			if prevLogIndex >= 0 {
				prevLogTerm = nr.Log[prevLogIndex].Term
			}
			entries := nr.Log[ni:]
			args := ArgAppendEntries{
				Term:         nr.CurrentTerm,
				LeaderId:     nr.Yo,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  prevLogTerm,
				Entries:      entries,
				LeaderCommit: nr.CommitIndex,
			}
			go nr.enviarLatido(i, ni, &args, &results)
		}
	}
}

func (nr *NodoRaft) raftProtocol() {
	for {
		for nr.Estado == "Seguidor" {
			select {
			case <-nr.chSeguidor:
				nr.Logger.Printf("Soy %d, soy seguidor y me ordenan seguir siéndolo.\n", nr.Yo)
			case <-time.After(getRandomTimeout()):
				// Si expira timeout se inicia el proceso de elección del líder
				nr.IdLider = -1
				nr.Estado = "Candidato"
				nr.Logger.Printf("Soy %d y ha expirado el timeout. Así que me presento como candidato.\n", nr.Yo)
			}
		}
		for nr.Estado == "Candidato" {
			// Se inicia el timer de elección y se envían RequestVote al resto de nodos
			timer := time.NewTimer(getRandomTimeout())
			nr.empezarEleccion()
			select {
			case <-nr.chSeguidor:
				// Latido
				nr.Estado = "Seguidor"
			case <-timer.C:
				// Si expira el timeout se inicia una nueva elección
				nr.Estado = "Candidato"
			case <-nr.chLider:
				// Si el nodo recibe mayoría simple de votos pasa a ser el nuevo líder
				nr.Estado = "Lider"
				//fmt.Println("Soy ", nr.Yo, " y ahora soy líder. Este es mi mandato ", nr.CurrentTerm)
				nr.Logger.Printf("Soy %d y ahora soy líder. Este es mi mandato %d\n", nr.Yo, nr.CurrentTerm)
			}
		}
		for nr.Estado == "Lider" {
			nr.IdLider = nr.Yo
			// La frecuencia de látidos no debe ser superior a 20 veces por segundo
			timer := time.NewTimer(50 * time.Millisecond)
			go nr.enviarLatidos()
			select {
			case <-nr.chSeguidor:
				// Si el nodo descubre que hay otro con mayor mandato vuelve a ser follower
				nr.Estado = "Seguidor"
				fmt.Println("Soy ", nr.Yo, ", era líder y ahora me vuelvo seguidor")
			case <-timer.C:
				// Se vuelven a enviar latidos pasados 50 milisegundos
				nr.Estado = "Lider"
				fmt.Println("Soy ", nr.Yo, " y renuevo cargo como líder")
			}
		}
	}
}

// Devuelve un timeout aleatorio entre 150 y 450 ms
func getRandomTimeout() time.Duration {
	return time.Duration(rand.Intn(300)+150) * time.Millisecond
}

// lastLogIndexAndTerm returns the last log index and the last log entry's term
// (or -1 if there's no log) for this server
func (nr *NodoRaft) lastLogIndexAndTerm() (int, int) {
	if len(nr.Log) > 0 {
		lastIndex := len(nr.Log) - 1
		return lastIndex, nr.Log[lastIndex].Term
	} else {
		return -1, -1
	}
}

// Hallar mínimo entre dos enteros
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
