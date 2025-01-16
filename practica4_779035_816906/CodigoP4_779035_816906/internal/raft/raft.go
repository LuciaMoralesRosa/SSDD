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
	CurrentTerm int          // Termino actual
	VotedFor    int          // Identificador del proceso votado
	Log         []EntradaLog // Array de entradas del nodo

	// Estado volatil en todos los servidores
	CommitIndex int // Indice de la mayor entrada comprometida conocida
	LastApplied int // Indice de la mayor entrada aplicada

	// Estado volatil en lideres
	// Indice de la entrada siguiente a enviar a cada nodo
	NextIndex map[int]int
	// Indice de la entrada mas alta conocida replicada en cada nodo
	MatchIndex map[int]int

	// Canales de comunicacion
	SoySeguidor chan bool // Notificar estado de seguidor
	SoyLider    chan bool // Notificar estado de lider
	Latido      chan bool // Notificar recepcion de latido

	AplicarOp chan AplicaOperacion // Empleado para la comunicacion con el
	// servidor
	Aplicada chan AplicaOperacion // Empleado por el lider para notificar
	// al cliente que la operacion ha sido comprometida

	// Otras variables necesarias
	Estado   string // LIDER, CANDIDADO o SEGUIDOR
	NumVotos int    // Numero de votos que ha recibido el candidato
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
	nr.Latido = make(chan bool)

	nr.AplicarOp = canalAplicarOperacion
	nr.Aplicada = make(chan AplicaOperacion)

	// Todos comienzan con estado de SEGUIDOR
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
		// Si es lider
		nr.Mux.Lock()
		nr.Logger.Printf("Soy lider y me ha llegado una nueva entrada\n")
		mandato = nr.CurrentTerm
		indice = len(nr.Log)

		// Creacion de la entrada
		entrada := EntradaLog{
			Operacion: operacion,
			Term:      mandato,
			Index:     indice,
		}
		nr.Logger.Printf("Soy lider y voy a añadir la nueva entrada a mi log\n")
		// Añadir la entrada al log del lider
		nr.Log = append(nr.Log, entrada)
		nr.Mux.Unlock()

		nr.Logger.Printf("Soy lider y voy a esperar a la notificacion de " +
			" aplicada por el canal nr.Aplicada\n")
		// Esperar a que la entrada haya sido comprometida
		valor := <-nr.Aplicada
		nr.Logger.Printf("Soy lider y se ha obtenido valor por el canar " +
			"nr.Aplicada\n Se va a responder al cliente\n")
		valorADevolver = valor.Operacion.Valor
		// Actualizar el valor de la ultima entrada comprometida
		nr.LastApplied = indice
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
	Term         int // Mandato del candidato
	CandidatoId  int // Identificador de proceso del candidato
	LastLogIndex int // Indice de la ultima entrada del log del candidato
	LastLogTerm  int // Mandato de la ultima entrada del log del candidato
}

// Structura de ejemplo de respuesta de RPC PedirVoto,
//
// Recordar
// -----------
// Nombres de campos deben comenzar con letra mayuscula !
type RespuestaPeticionVoto struct {
	// Vuestros datos aqui
	Term        int  // Mandato del nodo al que se le solicita el voto
	VoteGranted bool // True si vota al candidato que lo solicita
}

// Metodo para RPC PedirVoto
func (nr *NodoRaft) PedirVoto(peticion *ArgsPeticionVoto,
	reply *RespuestaPeticionVoto) error {
	// Vuestro codigo aqui

	// Obtener el lastLogIndex y lastLogTerm
	nr.Mux.Lock()
	lastLogIndex, lastLogTerm := nr.obtenerLastLogs()
	nr.Mux.Unlock()

	nr.Logger.Printf("Nodo %d: Ahora tengo mandato %d, he votado por %d, mi"+
		" log index es %d y mi log term es %d", nr.Yo, nr.CurrentTerm,
		nr.VotedFor, lastLogIndex, lastLogTerm)

	nr.Logger.Printf("Nodo %d: El nodo %d me pide su voto con mandato %d, log "+
		"index %d y log term %d", nr.Yo, peticion.CandidatoId, peticion.Term,
		peticion.LastLogIndex, peticion.LastLogTerm)

	// Comprobar la validez del candidato

	if peticion.Term > nr.CurrentTerm {
		// Si la peticion tiene un termino mayor
		nr.SoySeguidor <- true         // Sigo siendo seguidor
		nr.CurrentTerm = peticion.Term // Actualizo mi mandato
		nr.VotedFor = -1
	}

	// Los terminos deben ser iguales, no se ha debido votar por otro candidato
	// y el log del candidato debe estar tan actualizado como el del nodo
	if peticion.Term == nr.CurrentTerm &&
		(nr.VotedFor == -1 || nr.VotedFor == peticion.CandidatoId) &&
		nr.logCandidatoActualizado(peticion.LastLogTerm, lastLogTerm,
			peticion.LastLogIndex, lastLogIndex) {
		// El candidato es valido para el lider -> vota por el candidato
		reply.VoteGranted = true
		nr.VotedFor = peticion.CandidatoId
	} else {
		// No es valido para lider
		reply.VoteGranted = false
	}

	// Envio mi mandato al candidato
	reply.Term = nr.CurrentTerm

	return nil
}

// Comprueba si el candidato a lider tiene un log actualizado
func (nr *NodoRaft) logCandidatoActualizado(logTermPeticion, logTerm,
	logIndexPeticion, logIndex int) bool {
	// El termino del candidato es mayor al mio o
	// El termino es igual pero su log esta mas actualizado
	return logTermPeticion > logTerm ||
		(logTermPeticion == logTerm && logIndexPeticion >= logIndex)
}

// Devuelve el lastLogIndex y el lastLogTerm del nodo
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
	Term         int          // Mandato
	Leader       int          // Identificador del proceso lider
	PrevLogIndex int          // Indice de la entrada anterior a la nueva
	PrevLogTerm  int          // Termino de la entrada anterior a la nueva
	Entradas     []EntradaLog // Array de entradas a añadir al log
	Entrada      EntradaLog   // Vacio si es latido
	LeaderCommit int          // CommitIndex del lider
}

type Results struct {
	// Vuestros datos aqui
	Term    int  // Mandato del seguidor
	Success bool // True si es consistente
}

// Metodo de tratamiento de llamadas RPC AppendEntries
// Puede llegar una entrada o un latido
func (nr *NodoRaft) AppendEntries(args *ArgAppendEntries,
	results *Results) error {
	// Completar....
	nr.Logger.Printf("Nodo %d: He recibido un appendEntries de %d con mandato"+
		" %d y entradas %v\n", nr.Yo, args.Leader, args.Term, args.Entradas)

	// Comprobar la validez del lider comparando los mandatos
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

	// Inicializar exito a falso
	results.Success = false

	// Si el seguidor y el lider tienen el mismo mandato
	if nr.CurrentTerm == args.Term {
		// Confirmo que soy seguidor y que es el lider
		nr.SoySeguidor <- true
		nr.IdLider = args.Leader

		// Si no hay entradas o el log es consistente
		if args.PrevLogIndex == -1 || nr.esConsistente(args.PrevLogIndex,
			args.PrevLogTerm) {
			nr.Logger.Printf("Soy seguidor y respondo al lider que ha tenido" +
				" exito")
			results.Success = true
			go nr.guardarEntradasEnLog(args)
		}
	}

	// Respondo al lider con mi mandato
	results.Term = nr.CurrentTerm

	return nil
}

// Verifica si el log del seguidor es consistente con el del lider
func (nr *NodoRaft) esConsistente(indice int, mandato int) bool {
	return (indice < len(nr.Log) && mandato == nr.Log[indice].Term)
}

// Comprueba si hay entradas que guardar en el log y de ser asi las guarda y
// compromete en la maquina de estados si es preciso
func (nr *NodoRaft) guardarEntradasEnLog(args *ArgAppendEntries) {
	nr.Logger.Printf("En guardar entradas con entrada %d. \n",
		args.Entrada.Index)
	if !esLatido(args.Entrada) && len(args.Entradas) != 0 {
		// No es un latido y hay entradas que añadir

		// Indice del log donde empezar a insertar entradas
		indiceLog := args.PrevLogIndex + 1 // Eq a NextIndex[nodo]
		// Indica a partir de donde copiar de las entradas enviadas
		indiceEntradas := 0

		// Establecer la posicion de las nuevas entradas en el log del seguidor
		for !(posicionEncontrada(indiceLog, indiceEntradas, nr.Log,
			args.Entradas)) {
			indiceLog++
			indiceEntradas++
		}

		// Si el indiceEntradas es mayor a la longitud del vector de entradas
		// significa que no hay parte del vector que añadir al log
		if indiceEntradas < len(args.Entradas) {
			// Si es menor -> hay entradas que añadir
			nr.Logger.Printf("Nodo %d: Se van a insertar las entradas %v desde"+
				" el indice %d\n", nr.Yo, args.Entradas[indiceEntradas:],
				indiceLog)

			// Conservar las entradas que coinciden con el lider
			entradasCoincidentes := nr.Log[:indiceLog]

			// Obtener nuevas entradas a escribir enviadas por el lider
			nuevasEntradas := args.Entradas[indiceEntradas:]

			// Unir las entradas que coindicen con las nuevas
			nr.Log = append(entradasCoincidentes, nuevasEntradas...)

			nr.Logger.Printf("Nodo %d: Mi nuevo log es %v\n", nr.Yo, nr.Log)
		}
	}

	nr.Logger.Printf("Nodo %d: Soy seguidor y el leaderCommit de args es %d y "+
		" mi CommitIndex es %d\n", nr.Yo, args.LeaderCommit, nr.CommitIndex)

	// Comprobar si hay entradas que aplicar a la maquina de estados
	if args.LeaderCommit > nr.CommitIndex {
		// Si el CommitIndex del lider es mayor -> hay entradas sin aplicar
		nr.Logger.Printf("Nodo %d: Soy seguidor y voy a aplicar entradas en mi"+
			" maquina\n", nr.Yo)

		// Establecer el nuevo commitIndex del nodo
		// El minimo para que no de un valor mayor a la longitud de su log
		nr.CommitIndex = min(args.LeaderCommit, len(nr.Log)-1)
		nr.aplicarEntradasEnMaquinas()
	}
}

// Aplica las entradas correspondientes en la maquina de estados del nodo
func (nr *NodoRaft) aplicarEntradasEnMaquinas() {
	nr.Mux.Lock()
	ultimaAplicada := nr.LastApplied // Indice de la ultima entrada aplicada
	var entradas []EntradaLog        // Entradas a aplicar

	if nr.CommitIndex > nr.LastApplied {
		// Se pueden aplicar nuevas entradas
		// Entradas desde la ultima aplicada hasta la ultima comprometida
		entradas = nr.Log[nr.LastApplied+1 : nr.CommitIndex+1]
		nr.Logger.Printf("Nodo %d: Soy seguidor voy a aplicar las entradas "+
			"%v\n", nr.Yo, entradas)
		nr.LastApplied = nr.CommitIndex // Actualizamos ultima entrada aplicada
	}

	// Aplicar todas las entradas entre la ultima aplicada y ultima comprometida
	for entrada := 0; entrada < len(entradas); entrada++ {
		// Crear el mensaje para el servidor con la operacion
		mensaje := AplicaOperacion{
			Operacion: nr.Log[ultimaAplicada+entrada+1].Operacion,
			Indice:    ultimaAplicada + entrada + 1,
		}
		nr.Logger.Printf("Nodo %d: Soy seguidor voy a notificar a mi servidor"+
			"	de la entrada %d\n", nr.Yo, entrada)

		// Enviar el mensaje al servidor
		nr.AplicarOp <- mensaje

		nr.Logger.Printf("Nodo %d: Soy seguidor he notificado a mi servidor de"+
			"	la entrada %d\n", nr.Yo, entrada)
		nr.Logger.Printf("Nodo %d: Soy seguidor voy a esperar confirmacion del"+
			" servidor para la entraa %d\n", nr.Yo, entrada)

		// Esperar la respuesta del servidor
		<-nr.AplicarOp
		nr.Logger.Printf("Nodo %d: Soy seguidor el servidor me ha enviado "+
			"confirmacion para la entrada %d\n", nr.Yo, entrada)
	}
}

// Funcion auxiliar para obtener el minimo entre dos valores enteros
func min(v1, v2 int) int {
	if v1 < v2 {
		return v1
	}
	return v2
}

// Devuelve true si la EntradaLog es vacia y false en caso contrario
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
	// Enviar una peticion de voto al nodo
	err := nr.Nodos[nodo].CallTimeout("NodoRaft.PedirVoto", args, reply,
		33*time.Millisecond)
	if err != nil {
		return false
	} else {
		// He obtenido una respuesta del nodo
		if reply.Term > nr.CurrentTerm {
			// El que responde tiene mandato mayor al del candidato
			// Dejo de ser candidato y actualizo mi mandato
			nr.CurrentTerm = reply.Term
			nr.VotedFor = -1
			nr.SoySeguidor <- true
		} else if reply.VoteGranted {
			nr.NumVotos++
			// Compruebo mayoria de votos
			if nr.NumVotos > len(nr.Nodos)/2 {
				// Se ha conseguido mayoria -> soy lider
				for nodo := 0; nodo < len(nr.Nodos); nodo++ {
					if nodo != nr.Yo {
						nr.Mux.Lock()
						// Establecer el nextIndex del nodo a la entrada
						// siguiente a la ultima entrada del log
						nr.NextIndex[nodo] = len(nr.Log)
						// Establecer matchIndex de los nodos a -1
						nr.MatchIndex[nodo] = -1
						nr.Mux.Unlock()
					}
				}
				// Establecer estado de lider
				nr.SoyLider <- true
			}
		}
	}
	return true
}

// Maquina de estados de Raft
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

			// Si el nextIndex es menor a la longitud de mi log -> hay entradas
			if len(nr.Log)-1 >= nextIndexNodo_i {
				// Crear entrada a enviar
				entrada = EntradaLog{
					Operacion: nr.Log[nr.NextIndex[nodo]].Operacion,
					Term:      nr.Log[nr.NextIndex[nodo]].Term,
					Index:     nextIndexNodo_i,
				}

				// Establecer en entradas todas las entradas que le faltan
				entradas = nr.Log[nextIndexNodo_i:]

				// Establecer los valores de prevLogIndex y prevLogTerm
				prevLogIndex = nextIndexNodo_i - 1
				if prevLogIndex >= 0 { // No es la primera entrada
					prevLogTerm = nr.Log[prevLogIndex].Term
				} else { // Es la primera entrada
					prevLogTerm = -1
				}

			} else {
				// Enviar latido
				entrada = EntradaLog{} // Entrada vacia
				entradas = []EntradaLog{}

				// Establecer los valores de prevLogIndex y prevLogTerm
				if nextIndexNodo_i != 0 {
					prevLogIndex = nextIndexNodo_i - 1
					prevLogTerm = nr.Log[prevLogIndex].Term
				} else {
					prevLogTerm = -1
				}
			}

			// Crear ArgAppendEntries a enviar al nodo
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

// Realiza la llamada RPC con el envio del latido o entradas
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
		// Comprobar validez del lider
		if resultadoLatido.Term > nr.CurrentTerm {
			// Si le llega una respuesta con mandato mayor -> deja de ser lider
			nr.CurrentTerm = resultadoLatido.Term
			nr.VotedFor = -1
			nr.SoySeguidor <- true
		}
	} else {
		// Se han enviado entradas
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

// Comprueba si hay nuevas entradas que comprometer y si es asi, lo hace
func (nr *NodoRaft) comprometerEntradas() {
	// CommitIndex -> indice de la entrada comprometida mas alta
	nr.Logger.Printf("Soy lider y voy a ver si tengo entradas que " +
		"comprometer\n")

	for entrada := nr.CommitIndex + 1; entrada < len(nr.Log); entrada++ {
		// comprueba las entradas desde la ultima comprometida hasta el final

		// El lider solo compromete entradas de su mandato actual
		if nr.Log[entrada].Term == nr.CurrentTerm {
			confirmaciones := 1 // La del lider

			// Comprobar cuantos de los nodos tienen un matchIndex mayor o igual
			for nodo := 0; nodo < len(nr.Nodos); nodo++ {
				if nr.MatchIndex[nodo] >= entrada {
					confirmaciones++
					nr.Logger.Printf("Soy lider y tengo %d confirmaciones\n",
						confirmaciones)
				}
			}
			nr.Logger.Printf("Soy lider y para la entrada %d tengo %d "+
				" confirmaciones\n", entrada, confirmaciones)

			// Mirar si hay mayoria -> entonces comprometemos la entrada
			if confirmaciones > len(nr.Nodos)/2 {
				nr.Logger.Printf("Soy lider y he conseguido mayoria para " +
					"aplicar la entrada\n")
				nr.CommitIndex = entrada // Actualizar el commitIndex

				// Mensaje para aplicar la operacion en la maquina de estados
				aplicarOp := AplicaOperacion{
					Indice:    entrada,
					Operacion: nr.Log[entrada].Operacion,
				}

				// Enviar operacion al servidr
				nr.Logger.Printf("Soy lider y voy a notificar a mi servidor\n")
				nr.AplicarOp <- aplicarOp
				nr.Logger.Printf("Soy lider y he notificado a mi servidor, " +
					"voy a esperar su respuesta\n")
				// Espero respuesta del servidor
				valor := <-nr.AplicarOp
				nr.Logger.Printf("Soy lider me ha respondido mi servidor\n Se" +
					" va a pasar notificacion para responder al cliente\n")

				// Notifico para responder al cliente
				nr.Aplicada <- valor
			}
		}
	}
}

// Genera un numero aleatorio
func obtenerAleatorio() time.Duration {
	return time.Duration(rand.Intn(300)+150) * time.Millisecond
}
