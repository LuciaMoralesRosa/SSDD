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

const (
	LIDER     = "lider"
	SEGUIDOR  = "seguidor"
	CANDIDATO = "candidato"
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

type EntradaLog struct {
	Operacion TipoOperacion
	Term      int
	Index     int
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

	// Vuestros datos aqui.

	// Estados persistentes en todos los servidores
	CurrentTerm int
	VotedFor    int
	Log         []EntradaLog

	// Estado volatil en todos los servidores
	CommitIndex int
	LastApplied int

	// Estado volatil en lideres
	NextIndex  map[int]int
	MatchIndex map[int]int

	// Canales de comunicacion
	SoySeguidor  chan bool
	SoyLider     chan bool
	SoyCandidato chan bool
	Latido       chan bool

	AplicarOp chan AplicaOperacion //
	Aplicada  chan AplicaOperacion // Indicar a los clientes que su
	// entrada ha sido aplicada a la
	// maquina de estados

	// Otras variables necesarias
	Estado   string
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
	
	time.Sleep(5 * time.Second)
	nr := &NodoRaft{}
	nr.Nodos = nodos
	nr.Yo = yo
	nr.IdLider = -1

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
	nr.CurrentTerm = 0
	nr.VotedFor = -1
	nr.CommitIndex = -1
	nr.LastApplied = -1

	nr.NextIndex = make(map[int]int)
	nr.MatchIndex = make(map[int]int)

	nr.SoySeguidor = make(chan bool)
	nr.SoyLider = make(chan bool)
	nr.SoyCandidato = make(chan bool)
	nr.Latido = make(chan bool)

	nr.AplicarOp = canalAplicarOperacion
	nr.Aplicada = make(chan AplicaOperacion)

	nr.Estado = SEGUIDOR
	nr.NumVotos = 0

	go nr.protocoloRaft()

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
	mandato = nr.CurrentTerm
	esLider = nr.Yo == nr.IdLider

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
	esLider := false
	idLider := -1
	valorADevolver := ""

	// Vuestro codigo aqui
	idLider = nr.IdLider
	esLider = nr.Yo == nr.IdLider

	if esLider {
		nr.Mux.Lock()
		nr.Logger.Printf("Soy lider y me ha llegado una nueva entrada\n")
		mandato = nr.CurrentTerm
		indice = len(nr.Log)

		indiceEntrada := len(nr.Log)
		entrada := EntradaLog{
			Operacion: operacion,
			Term:      mandato,
			Index:     indiceEntrada,
		}
		nr.Logger.Printf("Soy lider y voy a añadir la nueva entrada a mi log\n")
		nr.Log = append(nr.Log, entrada)
		nr.Mux.Unlock()

		nr.Logger.Printf("Soy lider y voy a esperar a la notificacion de " +
			" aplicada por el canal nr.Aplicada\n")
		//valor := <-nr.AplicarOp
		//valorADevolver = valor.Operacion.Operacion
		valor := <-nr.Aplicada
		nr.Logger.Printf("Soy lider y se ha obtenido valor por el canar " +
			"nr.Aplicada\n Se va a responder al cliente\n")
		valorADevolver = valor.Operacion.Valor

		nr.LastApplied = indiceEntrada
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

type EstadoLog struct {
	Term   int
	Indice int
}

func (nr *NodoRaft) ObtenerEstadoLog(args Vacio, reply *EstadoLog) error {
	reply.Indice, reply.Term = nr.obtenerEstadoLog()
	return nil
}

func (nr *NodoRaft) obtenerEstadoLog() (int, int) {
	indice := -1
	mandato := 0

	if len(nr.Log) != 0 {
		indice = nr.CommitIndex
		mandato = nr.Log[indice].Term
	}
	return indice, mandato
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
	CandidatoId  int
	LastLogIndex int
	LastLogTerm  int
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
	// Vuestro codigo aqui
	nr.Mux.Lock()
	lastLogIndex, lastLogTerm := nr.obtenerLastLogs()
	nr.Mux.Unlock()

	nr.Logger.Printf("Nodo %d: Ahora tengo mandato %d, he votado por %d, mi"+
		" log index es %d y mi log term es %d", nr.Yo, nr.CurrentTerm,
		nr.VotedFor, lastLogIndex, lastLogTerm)

	nr.Logger.Printf("Nodo %d: El nodo %d me pide su voto con mandato %d, log "+
		"index %d y log term %d", nr.Yo, peticion.CandidatoId, peticion.Term,
		peticion.LastLogIndex, peticion.LastLogTerm)

	// Si la peticion tiene un termino mayor
	if peticion.Term > nr.CurrentTerm {
		nr.SoySeguidor <- true
		nr.CurrentTerm = peticion.Term
		nr.VotedFor = -1
	}

	// Los terminos deben ser iguales, no se ha debido votar por otro candidato
	// y el log del candidato debe estar tan actualizado como el del nodo
	if peticion.Term == nr.CurrentTerm &&
		(nr.VotedFor == -1 || nr.VotedFor == peticion.CandidatoId) &&
		nr.logCandidatoActualizado(peticion.LastLogTerm, lastLogTerm,
			peticion.LastLogIndex, lastLogIndex) {
		// Entonces votamos por el candidato
		reply.VoteGranted = true
		nr.VotedFor = peticion.CandidatoId
	} else {
		// peticion.Term < nr.CurrentTerm
		reply.VoteGranted = false
	}

	reply.Term = nr.CurrentTerm

	return nil
}

func (nr *NodoRaft) logCandidatoActualizado(logTermPeticion, logTerm,
	logIndexPeticion, logIndex int) bool {
	return logTermPeticion > logTerm ||
		(logTermPeticion == logTerm && logIndexPeticion >= logIndex)
}

func (nr *NodoRaft) obtenerLastLogs() (int, int) {
	if len(nr.Log) > 0 {
		lastIndex := len(nr.Log) - 1
		return lastIndex, nr.Log[lastIndex].Term
	} else {
		return -1, -1
	}
}

type ArgAppendEntries struct {
	// Vuestros datos aqui
	Term         int
	Leader       int
	PrevLogIndex int
	PrevLogTerm  int
	Entradas     []EntradaLog
	Entrada      EntradaLog // Vacio si es latido
	LeaderCommit int
}

type Results struct {
	// Vuestros datos aqui
	Term    int
	Success bool
}

// Metodo de tratamiento de llamadas RPC AppendEntries
// Puede llegar una entrada o un latido
func (nr *NodoRaft) AppendEntries(args *ArgAppendEntries,
	results *Results) error {
	// Completar....
	nr.Logger.Printf("Nodo %d: He recibido un appendEntries de %d con mandato"+
		" %d y entradas %v\n", nr.Yo, args.Leader, args.Term, args.Entradas)

	if nr.CurrentTerm < args.Term {
		// Lo reconozco como lider y me quedo como seguidor
		nr.IdLider = args.Leader
		nr.CurrentTerm = args.Term

		if nr.Estado != LIDER { // Si era seguidor o candidato recibo latido
			nr.Latido <- true
		}
		// Sigo siendo seguidor
		nr.SoySeguidor <- true
		// Actualuizar el commitIndex??
	}

	// Inicializar consistencia a falso
	results.Success = false

	if nr.CurrentTerm == args.Term { // Mismo mandato
		nr.SoySeguidor <- true // Paso a ser o sigo siendo seguidor
		nr.IdLider = args.Leader

		// Si el seguidor es consistente o no hay entradas
		if args.PrevLogIndex == -1 || nr.esConsistente(args.PrevLogIndex,
			args.PrevLogTerm) {
			nr.Logger.Printf("Soy seguidor y respondo al lider que ha tenido" +
				" exito")
			results.Success = true
			go nr.guardarEntradasEnLog(args)
		}
	}

	results.Term = nr.CurrentTerm

	return nil
}

func (nr *NodoRaft) esConsistente(indice int, mandato int) bool {
	return (indice < len(nr.Log) && mandato == nr.Log[indice].Term)
}

func (nr *NodoRaft) guardarEntradasEnLog(args *ArgAppendEntries) {
	nr.Logger.Printf("En guardar entradas con entrada %d. \n",
		args.Entrada.Index)
	if !esLatido(args.Entrada) && len(args.Entradas) != 0 {
		// Añade la nueva entrada y las anteriores si es necesario
		// Indice del log donde empezar a insertar entradas
		indiceLog := args.PrevLogIndex + 1
		// Indica a partir de donde copiar de las entradas enviadas
		indiceEntradas := 0

		for !(posicionEncontrada(indiceLog, indiceEntradas, nr.Log,
			args.Entradas)) {
			indiceLog++
			indiceEntradas++
		}

		// Si el indice de las entradas esta dentro del vector
		if indiceEntradas < len(args.Entradas) {
			nr.Logger.Printf("Nodo %d: Se van a insertar las entradas %v desde"+
				" el indice %d\n", nr.Yo, args.Entradas[indiceEntradas:],
				indiceLog)

			// Conservar las entradas que coinciden con el lider
			entradasCoincidentes := nr.Log[:indiceLog]
			// Obtener nuevas entradas a escribir enviadas por el lider
			nuevasEntradas := args.Entradas[indiceEntradas:]
			nr.Log = append(entradasCoincidentes, nuevasEntradas...)

			nr.Logger.Printf("Nodo %d: Mi nuevo log es %v\n", nr.Yo, nr.Log)
		}
	}

	nr.Logger.Printf("Nodo %d: Soy seguidor y el leaderCommit de args es %d y "+
		" mi CommitIndex es %d\n", nr.Yo, args.LeaderCommit, nr.CommitIndex)

	if args.LeaderCommit > nr.CommitIndex {
		nr.Logger.Printf("Nodo %d: Soy seguidor y voy a aplicar entradas en mi"+
			" maquina\n", nr.Yo)
		nr.CommitIndex = min(args.LeaderCommit, len(nr.Log)-1)
		nr.aplicarEntradasEnMaquinas()
	}
}

func (nr *NodoRaft) aplicarEntradasEnMaquinas() {
	nr.Mux.Lock()
	ultimaAplicada := nr.LastApplied
	var entradas []EntradaLog // Entradas a aplicar

	if nr.CommitIndex > nr.LastApplied { // Se pueden aplicar nuevas entradas
		// Entradas desde la ultima aplicada hasta la ultima comprometida
		entradas = nr.Log[nr.LastApplied+1 : nr.CommitIndex+1]
		nr.Logger.Printf("Nodo %d: Soy seguidor voy a aplicar las entradas "+
			"%v\n", nr.Yo, entradas)
		nr.LastApplied = nr.CommitIndex // Actualizamos ultima entrada aplicada
	}

	// Aplicar todas las entradas
	for entrada := 0; entrada < len(entradas); entrada++ {
		solicitud := AplicaOperacion{
			Operacion: nr.Log[ultimaAplicada+entrada+1].Operacion,
			Indice:    ultimaAplicada + entrada + 1,
		}
		nr.Logger.Printf("Nodo %d: Soy seguidor voy a notificar a mi servidor"+
			"	de la entrada %d\n", nr.Yo, entrada)
		nr.AplicarOp <- solicitud
		//valor := <-nr.AplicarOp
		nr.Logger.Printf("Nodo %d: Soy seguidor he notificado a mi servidor de"+
			"	la entrada %d\n", nr.Yo, entrada)
		nr.Logger.Printf("Nodo %d: Soy seguidor voy a esperar confirmacion del"+
			" servidor para la entraa %d\n", nr.Yo, entrada)
		<-nr.AplicarOp
		nr.Logger.Printf("Nodo %d: Soy seguidor el servidor me ha enviado "+
			"confirmacion para la entrada %d\n", nr.Yo, entrada)
		//nr.Aplicada <- valor.Operacion.Valor
	}
}

func min(v1, v2 int) int {
	if v1 < v2 {
		return v1
	}
	return v2
}

func esLatido(entrada EntradaLog) bool {
	return entrada == EntradaLog{}
}

// Devuelve true si se ha encontrado la posicion critica
func posicionEncontrada(indiceLog int, indiceEntradas int, log []EntradaLog,
	entradas []EntradaLog) bool {
	// Dentro de limites

	if len(entradas) == 0 || len(log) == 0 {
		return true
	}

	if indiceLog >= len(log) || indiceEntradas >= len(entradas) {
		return true
	}

	coinciden := log[indiceLog].Term == entradas[indiceEntradas].Term

	return !(coinciden)

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
	err := nr.Nodos[nodo].CallTimeout("NodoRaft.PedirVoto", args, reply,
		33*time.Millisecond)
	if err != nil {
		return false
	} else {
		nr.Mux.Lock()
		if nr.Estado == CANDIDATO {
			if reply.Term > nr.CurrentTerm {
				// El que responde tiene mandato mayor al del candidato
				nr.CurrentTerm = reply.Term
				nr.VotedFor = -1
				nr.SoySeguidor <- true
			} else if reply.VoteGranted {
				// Si el nodo me ha votado a mi
				nr.NumVotos++
				if nr.NumVotos > len(nr.Nodos)/2 {
					// Se ha conseguido mayoria
					for nodo := 0; nodo < len(nr.Nodos); nodo++ {
						if nodo != nr.Yo {
							//nr.Mux.Lock()
							nr.NextIndex[nodo] = len(nr.Log)
							nr.MatchIndex[nodo] = -1
							//nr.Mux.Unlock()
						}
					}
					nr.SoyLider <- true
				}
			}
		}
		nr.Mux.Unlock()
	}
	return true
}

func (nr *NodoRaft) protocoloRaft() {
	for {
		for nr.Estado == SEGUIDOR {
			nr.protocoloSeguidor()
		}
		for nr.Estado == CANDIDATO {
			nr.protocoloCandidato()
		}
		for nr.Estado == LIDER {
			nr.protocoloLider()
		}
	}
}

func (nr *NodoRaft) protocoloSeguidor() {
	select {
	// Va recibir latidos dentro de un timeout aleatorio. Si no lo recibe:
	case <-time.After(obtenerAleatorio()):
		// Dejara de reconocer al lider
		nr.IdLider = -1
		// Pasara a ser candidato a lider
		nr.Estado = CANDIDATO
	// Si recibe un soySeguidor continua siendo seguidor sin hacer cambios
	case <-nr.SoySeguidor:
		nr.Logger.Printf("Nodo %d: Soy seguidor y sigo siendolo en "+
			"mandato %d \n", nr.Yo, nr.CurrentTerm)
	}
}

func (nr *NodoRaft) protocoloCandidato() {
	// No ha recibido latidos asi que comienza una eleccion
	temporizador := time.NewTimer(obtenerAleatorio()) // Establecer timeout
	nr.comenzarEleccion()

	// Puede ganar la eleccion, perderla o empatar
	select {
	// Ha perdido la eleccion porque ha recibido un latido del nuevo lider
	case <-nr.SoySeguidor:
		nr.Estado = SEGUIDOR
		nr.Logger.Printf("Nodo %d: He perdido la eleccion y paso a ser "+
			"seguidor en el mandato %d \n", nr.Yo, nr.CurrentTerm)
	case <-nr.Latido:
		nr.Estado = SEGUIDOR
		nr.Logger.Printf("Nodo %d: He perdido la eleccion y paso a ser "+
			"seguidor en el mandato %d \n", nr.Yo, nr.CurrentTerm)
	// Ha ganado la eleccion al conseguir mayoria de votos
	case <-nr.SoyLider:
		nr.Estado = LIDER
		nr.Logger.Printf("Nodo %d: He ganado la eleccion y paso a ser "+
			"lider en el mandato %d \n", nr.Yo, nr.CurrentTerm)
	// Vence el timeout: sigue como candidato y empieza una nueva eleccion
	case <-temporizador.C:
		nr.Logger.Printf("Nodo %d: Ha vencido el timeout de eleccion y "+
			"sigo siendo candidato en el mandato %d\n", nr.Yo,
			nr.CurrentTerm)
	}
}

func (nr *NodoRaft) comenzarEleccion() {
	// Actualiza el CurrentTerm y se vota a si mismo
	nr.CurrentTerm++
	nr.VotedFor = nr.Yo
	nr.NumVotos = 1

	nr.Mux.Lock()
	lastLogIndex, lastLogTerm := nr.obtenerLastLogs()
	nr.Mux.Unlock()

	// Va a solicitar votos de todos los nodos
	peticionVoto := ArgsPeticionVoto{
		Term:         nr.CurrentTerm,
		CandidatoId:  nr.Yo,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}

	var respuestaVoto RespuestaPeticionVoto
	for i := 0; i < len(nr.Nodos); i++ {
		if i != nr.Yo {
			go nr.enviarPeticionVoto(i, &peticionVoto, &respuestaVoto)
		}
	}
}

func (nr *NodoRaft) protocoloLider() {
	// Actualiza el nr.IdLider al mio
	nr.IdLider = nr.Yo
	// Va a mandar latidos periodicos cada 50ms
	temporizadorLatidos := time.NewTimer(50 * time.Millisecond)
	go nr.enviarLatidosATodos()

	// Siendo lider puede renovar su cargo o perderlo
	select {
	case <-nr.SoySeguidor:
		nr.Estado = SEGUIDOR
		nr.Logger.Printf("Nodo %d: Era lider y ahora soy seguidor en el "+
			"mandato %d \n", nr.Yo, nr.CurrentTerm)
	case <-temporizadorLatidos.C:
		nr.Logger.Printf("Nodo %d: Soy lider y sigo siendolo en el "+
			"mandato %d \n", nr.Yo, nr.CurrentTerm)
	}
}

// Envia latidos a todos los nodos. Como es un latido la entrada de log es vacia
func (nr *NodoRaft) enviarLatidosATodos() {
	nr.Logger.Printf("Nodo %d: Soy lider y envio latidos en el "+
		"mandato %d \n", nr.Yo, nr.CurrentTerm)

	var prevLogIndex int
	var prevLogTerm int
	var entrada EntradaLog
	var entradas []EntradaLog

	for nodo := 0; nodo < len(nr.Nodos); nodo++ {
		if nodo != nr.Yo {
			// Comprobamos si hay que mandar latido o hay entradas
			nextIndexNodo_i := nr.NextIndex[nodo]
			if len(nr.Log)-1 >= nextIndexNodo_i {
				// Hay entradas que enviar
				entrada = EntradaLog{
					Operacion: nr.Log[nr.NextIndex[nodo]].Operacion,
					Term:      nr.Log[nr.NextIndex[nodo]].Term,
					Index:     nextIndexNodo_i,
				}

				entradas = nr.Log[nextIndexNodo_i:]
				prevLogIndex = nextIndexNodo_i - 1

				if prevLogIndex >= 0 { // No es la primera entrada
					prevLogTerm = nr.Log[prevLogIndex].Term
				} else { // Es la primera entrada
					prevLogTerm = -1
				}

			} else {
				// Enviar latido
				entrada = EntradaLog{}
				entradas = []EntradaLog{}
				if nextIndexNodo_i != 0 {
					prevLogIndex = nextIndexNodo_i - 1
					prevLogTerm = nr.Log[prevLogIndex].Term
				} else {
					prevLogTerm = -1
				}
			}

			// entradas contiene todas las entradas del log del lider
			// posteriores al indice del nodo
			// Index del Lider: 1, 2, 3, 4, 5, 6
			// Index del seguidor: 1, 2, 3
			// Entradas: 4, 5, 6
			argumentos := ArgAppendEntries{
				Term:         nr.CurrentTerm,
				Leader:       nr.Yo,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  prevLogTerm,
				Entradas:     entradas,
				Entrada:      entrada,
				LeaderCommit: nr.CommitIndex,
			}

			go nr.envioDeLatido(nodo, nextIndexNodo_i, &argumentos)

		}
	}
}

func (nr *NodoRaft) envioDeLatido(nodo int, nextIndex int,
	args *ArgAppendEntries) {
	var resultadoLatido Results

	err := nr.Nodos[nodo].CallTimeout("NodoRaft.AppendEntries", args,
		&resultadoLatido, 33*time.Millisecond)
	if err != nil {
		nr.Logger.Printf("Soy lider y la llamada AppendEntries me esta dando" +
			" un error\n")

		// No bloquear en errores
		return
	}

	if resultadoLatido.Success {
		nr.Logger.Printf("Soy lider y resultado latido en termino %d es %t\n",
			resultadoLatido.Term, resultadoLatido.Success)
	}

	if esLatido(args.Entrada) { // Enviado latido
		// Si le llega una respuesta con mandato mayor -> deja de ser lider
		if resultadoLatido.Term > nr.CurrentTerm {
			nr.CurrentTerm = resultadoLatido.Term
			nr.VotedFor = -1
			nr.SoySeguidor <- true
		}
	} else { // Se han enviado entradas
		nr.Logger.Printf("Soy lider y he enviado entradas\n")
		if resultadoLatido.Success {
			// Si la llamada ha tenido exito, ie, el seguidor es consistente
			nr.Mux.Lock()
			// Actualizo el nextIndex del nodo al nuevo
			nr.NextIndex[nodo] = nextIndex + len(args.Entradas)
			nr.MatchIndex[nodo] = nr.NextIndex[nodo] - 1
			nr.Mux.Unlock()

			// Como se han añadido entradas, comprobamos si se puede comprometer
			nr.comprometerEntradas()
		} else {
			nr.Logger.Printf("Soy lider y el appendentries ha fallado\n")
			// El seguidor no era consistente - se reintentara la llamada
			nr.NextIndex[nodo] = nextIndex - 1
		}
	}
}

func (nr *NodoRaft) comprometerEntradas() {
	// CommitIndex -> indice de la entrada comprometida mas alta
	nr.Logger.Printf("Soy lider y voy a ver si tengo entradas que " +
		"comprometer\n")
	fmt.Printf("El valor del commit index del lider al empezar comprometer "+
		"entrada es: %d\n", nr.CommitIndex)

	for entrada := nr.CommitIndex + 1; entrada < len(nr.Log); entrada++ {
		// El lider solo compromete contando replicas las entradas del termino
		// actual
		if nr.Log[entrada].Term == nr.CurrentTerm {
			confirmaciones := 1 // La que ha causado la llamada a este metodo

			// Comprobar cuantos de los nodos tienen un matchIndex mayor o igual
			for nodo := 0; nodo < len(nr.Nodos); nodo++ {
				if nr.MatchIndex[nodo] >= entrada {
					nr.Logger.Printf("Soy lider y tengo %d confirmaciones\n",
						confirmaciones)
					confirmaciones++
				}
			}
			nr.Logger.Printf("Soy lider y para la entrada %d tengo %d "+
				" confirmaciones\n", entrada, confirmaciones)

			// Mirar si hay mayoria -> entonces comprometemos la entrada
			if confirmaciones > len(nr.Nodos)/2 {
				nr.Logger.Printf("Soy lider y he conseguido mayoria para " +
					"aplicar la entrada\n")
				nr.CommitIndex = entrada
				fmt.Printf("El valor del commit index del lider antes de "+
					"enviar la respusta es: %d\n", nr.CommitIndex)
				// Solicitud para aplicar la operacion en la maquina de estados
				aplicarOp := AplicaOperacion{
					Indice:    entrada,
					Operacion: nr.Log[entrada].Operacion,
				}
				nr.Logger.Printf("Soy lider y voy a notificar a mi servidor\n")
				nr.AplicarOp <- aplicarOp
				nr.Logger.Printf("Soy lider y he notificado a mi servidor, " +
					"voy a esperar su respuesta\n")
				valor := <-nr.AplicarOp // La aplico en el lider
				nr.Logger.Printf("Soy lider me ha respondido mi servidor\n Se" +
					" va a pasar notificacion para responder al cliente\n")
				nr.Aplicada <- valor
			}
		}
	}
}

// ------------------------- FUNCIONES AUXILIARES ----------------------------//
func obtenerAleatorio() time.Duration {
	return time.Duration(rand.Intn(300)+150) * time.Millisecond
}
