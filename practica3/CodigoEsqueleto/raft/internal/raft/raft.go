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
	Lider = "lider"
	Candidato = "candidato"
	Seguidor = "seguidor"
)

type TipoOperacion struct {
	Operacion string  // La operaciones posibles son "leer" y "escribir"
	Clave string
	Valor string    // en el caso de la lectura Valor = ""
}


// A medida que el nodo Raft conoce las operaciones de las  entradas de registro
// comprometidas, envía un AplicaOperacion, con cada una de ellas, al canal
// "canalAplicar" (funcion NuevoNodo) de la maquina de estados 
type AplicaOperacion struct {
	Indice int  // en la entrada de registro
	Operacion TipoOperacion
}

type LogEntry struct {
	Operacion TipoOperacion
	Term int
}

// Tipo de dato Go que representa un solo nodo (réplica) de raft
//
type NodoRaft struct {
	Mux   sync.Mutex       // Mutex para proteger acceso a estado compartido

	// Host:Port de todos los nodos (réplicas) Raft, en mismo orden
	Nodos []rpctimeout.HostPort
	Yo    int           // indice de este nodos en campo array "nodos"
	IdLider int			// Indice del lider
	
	// Utilización opcional de este logger para depuración
	// Cada nodo Raft tiene su propio registro de trazas (logs)
	Logger *log.Logger

	// Vuestros datos aqui.
	EstadoNodo		string	// Puede ser Lider, Candidato o Seguidor
	VotosParaLider	int		// Numero de votos que tiene el nodo para ser lider
	
	CurrentTerm		int // Ultimo termino
	VotedFor		int	// Candidato por el que ha votado
	Logs			[]LogEntry
	
	CommitIndex		int	// Indice de la entry comprometida mas alta
	LastApplied		int // Indice de la entrada de registro más alta aplicada a
	// la máquina de estados
	
	// Usamos array de enteros que funcionan igual que un mapa
	// Es decir, la posicion del array equivaldria a la clave del map
	NextIndex		[]int
	MatchIndex		[]int

	soySeguidor		chan bool
	soyLider		chan bool
	Latidos			chan bool // El nodo recibe el latido
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
			nr.Logger = log.New(os.Stdout, nombreNodo + " -->> ",
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
						   logPrefix + " -> ", log.Lmicroseconds|log.Lshortfile)
		}
		nr.Logger.Println("logger initialized")
	} else {
		nr.Logger = log.New(ioutil.Discard, "", 0)
	}

	// Añadir codigo de inicialización
	nr.EstadoNodo = Seguidor	
	nr.Latidos = make(chan bool) 
	nr.soyLider = make(chan bool)
	nr.soySeguidor = make(chan bool)
	nr.VotosParaLider = 0
	nr.CurrentTerm = 0
	nr.VotedFor = -1
	nr.Logs	= []LogEntry{}

	nr.CommitIndex	= -1
	nr.LastApplied	= -1
	nr.NextIndex = make([]int, len(nodos))
	nr.MatchIndex = make([]int, len(nodos))

	// Lanzamiento de la maquina de estados
	go maquinaEstados(nr)
	
	return nr
}


// Metodo Para() utilizado cuando no se necesita mas al nodo
//
// Quizas interesante desactivar la salida de depuracion
// de este nodo
//
func (nr *NodoRaft) para() {
	go func() {time.Sleep(5 * time.Millisecond); os.Exit(0) } ()
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
	esLider = nr.Yo == idLider

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
	valorADevolver = "Error"
	EsLider = nr.Yo == nr.IdLider

	if EsLider {
		indice = nr.CommitIndex
		mandato = nr.CurrentTerm

		nr.Log = append(nr.Log, LogEntry{Operacion: operacion,
										 Term: nr.CurrentTerm})

		var resultadoAE Results
		confirmados := 1

		longitudNodos := len(nr.Nodos)
		for i:=0; i < longitudNodos; i++ {
			// Si soy yo, no hago nada
			if i != nr.Yo {
				argumentos := ArgAppendEntries{Term: mandato,
											   IdLider: nr.Yo,
											   LogIndexPrev: indice,
											   LogTermPrev: nr.CurrentTerm,
											   Entradas: []LogEntry{{operacion,
															nr.CurrentTerm}},
											   CommitLider: -1,
											}
				err := nr.Nodos[i].CallTimeout("NodoRaft.AppendEntries",
											   &argumentos, &resultadoAE,
											   50*time.Millisecond)
				check.CheckError(err, "Error en la llamada AppendEntries")
				if resultadoAE.Exito {
					confirmados++
					nr.NextIndex[i] += len(args.Entries)
					nr.MatchIndex[i] = nr.NextIndex[i] - 1
				} else {
					if mandato < resultadoAE.Term {
						nr.CurrentTerm = resultadoAE.Term
						soySeguidor <- true
					}
				}
			}
		}
		
		if confirmados > longitudNodos/2 {
			nr.CommitIndex++
			valorADevolver = "Comprometido"
		}
	}
	return indice, mandato, EsLider, idLider, valorADevolver
}


// -----------------------------------------------------------------------
// LLAMADAS RPC al API
//
// Si no tenemos argumentos o respuesta estructura vacia (tamaño cero)
type Vacio struct{}

func (nr * NodoRaft) ParaNodo(args Vacio, reply *Vacio) error {
	defer nr.para()
	return nil
}

type EstadoParcial struct {
	Mandato	int
	EsLider bool
	IdLider	int
}

type EstadoRemoto struct {
	IdNodo	int
	EstadoParcial
}

func (nr *NodoRaft) ObtenerEstadoNodo(args Vacio, reply *EstadoRemoto) error {
	reply.IdNodo,reply.Mandato,reply.EsLider,reply.IdLider = nr.obtenerEstado()
	return nil
}

type ResultadoRemoto struct {
	ValorADevolver string
	IndiceRegistro int
	EstadoParcial
}

func (nr *NodoRaft) SometerOperacionRaft(operacion TipoOperacion,
												reply *ResultadoRemoto) error {
	reply.IndiceRegistro,reply.Mandato, reply.EsLider,
			reply.IdLider,reply.ValorADevolver = nr.someterOperacion(operacion)
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
//
type ArgsPeticionVoto struct {
	Term	int
	IdCandidato	int
	LastLogIndexCandidato	int
	LastLogTermCandidato	int
}

// Structura de ejemplo de respuesta de RPC PedirVoto,
//
// Recordar
// -----------
// Nombres de campos deben comenzar con letra mayuscula !
//
//
type RespuestaPeticionVoto struct {
	Term	int
	VotoEmitido	bool // Indica si ha concedido su voto al candidato
}


// Metodo para RPC PedirVoto
//
func (nr *NodoRaft) PedirVoto(peticion *ArgsPeticionVoto,
										reply *RespuestaPeticionVoto) error {
	// Vuestro codigo aqui
	lastLogIndex, lastLogTerm := obtenerLastLog(nr)
	if peticion.Term > nr.CurrentTerm {
		nr.soySeguidor <- true
		// Actualizar el termino
		nr.CurrentTerm = peticion.Term
		nr.VotedFor = -1
	}

	if peticion.Term == nr.CurrentTerm &&
	   (nr.VotedFor == -1 || nr.VotedFor == peticion.IdCandidato) &&
	   (peticion.LastLogTermCandidato > lastLogTerm ||
	   (peticion.LastLogTermCandidato == lastLogTerm &&
	   (peticion.LastLogIndexCandidato >= lastLogIndex))) {
		// Voto por el candidato
		reply.VotoEmitido = true
		nr.VotedFor = peticion.IdCandidato
	} else {
		reply.VotoEmitido = false
	}
	reply.Term = nr.CurrentTerm
	return nil	
}

type ArgAppendEntries struct {
	// Vuestros datos aqui
	Term	int
	IdLider	int
	LogIndexPrev	int
	LogTermPrev		int
	Entradas		[]LogEntry
	CommitLider		int
}

type Results struct {
	// Vuestros datos aqui
	// La salida del AppendEntries es el termino actual y si 
	Term	int	// Termino del lider que envio la solicitud
	Exito 	bool	//true si la entrada fue replicada correctamente
	//SiguienteIndice	int
}


// Metodo de tratamiento de llamadas RPC AppendEntries
func (nr *NodoRaft) AppendEntries(args *ArgAppendEntries,
													  results *Results) error {
	// Completar....
	if nr.CurrentTerm < args.Term {
		// Mi mandato es menor al de la peticion de lider -> soy seguidor y 
		// actualizo el termino
		nr.soySeguidor <- true
		nr.CurrentTerm = args.Term
	}

	// Inicializacion de exito a falso
	results.Exito = false

	if nr.CurrentTerm == args.Term {
		longitudLog := len(nr.Log)
		// Sigo siendo seguidor 
		nr.soySeguidor <- true
		if args.LogIndexPrev == -1 || (args.LogIndexPrev < longitudLog &&
		   args.LogTermPRev == nr.Log[args.LogIndexPrev].Term) {
			args.Exito = true
			indiceInsercion := args.LogIndexPrev + 1
			indiceEntrada := 0

			for{
				if (indiceInsercion >= longitudLog ||
				   indiceEntrada >= len(args.Entradas)) ||
				   (nr.Log[indiceInsercion].Term !=
				   args.Entradas[indiceEntrada].Term) {
					break
				}
				indiceInsercion++
				indiceEntrada++
			}
			if indiceEntrada < len(args.Entradas) {
				// Reemplazar el Log desde indiceInterseccion las entradas de
				// args.Entradas desde indiceEntrada
				nr.Log = append(nr.Log[:indiceInsercion],
								args.Entradas[indiceEntrada:]...)
				
			}

			if args.CommitLider > nr.CommitIndex {
				// Si el commit del lider es mayor que el mio -> 
				if args.CommitLider < (len(nr.Log)-1) {
					nr.CommitIndex = args.CommitLider
				} else {
					nr.CommitIndex = len(nr.Log)-1
				}
				
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
//
func (nr *NodoRaft) enviarPeticionVoto(nodo int, args *ArgsPeticionVoto,
											reply *RespuestaPeticionVoto) bool {
	// Completar....
	nr.Mux.Lock()
	defer nr.Mux.Unlock()

	err := nr.Nodos[nodo].CallTimeout("NodoRaft.PedirVoto", args, reply,
									  50*time.Millisecond)
	check.CheckError(err, "Error en la llamada PedirVoto")
	if reply.Term > nr.CurrentTerm {
		// El que me responde tiene mayor Term -> actualizo y paso a seguidor
		nr.CurrentTerm = reply.Term
		soySeguidor <- true
		nr.VotedFor = -1 // Dejo de votar
	} else if reply.VotoEmitido {
		nr.VotosParaLider++
		
		// Comprobar mayoria de votos hacia mi
		longitudNodos := len(nr.Nodos)
		longitudLog := len(nr.Log)
		if VotosParaLider > longitudNodos/2 {
			for i := 0; i < longitudNodos; i++ {
				if i != nr.Yo {
					nr.NextIndex[i] = longitudLog
					nr.MatchIndex[i] -1
				}
			}
			// Si tengo mayoria paso a ser lider
			nr.SoyLider <- true
		}
	}	
	return true
}



// --------------------------- OTROS METODOS -----------------------------------
func maquinaEstadosLider(nr *NodoRaft) {
	nr.IdLider = nr.Yo
	temporizador := time.NewTimer(50 * time.Millisecond)
	go enviarLatido(nr)
	select {
		case <- nr.SoySeguidor:
			nr.EstadoNodo = Seguidor
		case <-temporizador.C:
			// Se usa el canal del temporizador para saber si han pasado los 50ms
			nr.EstadoNodo = Lider
	}
}

func maquinaEstadosSeguidor(nr *NodoRaft){
	select {
		case<-time.After(generarTiempoAleatorio()):
			nr.Lider = -1
			nr.EstadoNodo = Candidato
		case <- nr.soySeguidor:
			nr.EstadoNodo = Seguidor
	}
}

func maquinaEstadosCandidato(nr *NodoRaft){
	nr.CurrentTerm++ // Se inicia un nuevo termino
	nr.VotosParaLider = 1 // Se propone como lider y se vota
	nr.VotedFor = nr.Yo // Se vota a su mismo para lider
	// Se establece el tiempo para la votacion
	temporizador := time.NewTimer(generarTiempoAleatorio())
	nuevaEleccion(nr)

	select {
		case <- nr.SoySeguidor:
			nr.EstadoNodo = Seguidor // Si le llega un latido es que hay lider
		case <- nr.SoyLider:
			nr.EstadoNodo = Lider
		case <- temporizador.C:
			nr.EstadoNodo = Candidato // Se establece como candidato y se 
									  // repetira la votacion
	}
}

func maquinaEstados(nr *NodoRaft) {
	for {
		switch nr.EstadoNodo {
		case Lider:
			maquinaEstadosLider(nr)
		case Candidato:
			maquinaEstadosCandidato(nr)
		case Seguidor:
			maquinaEstadosSeguidor(nr)
		}
	}
}

func enviarLatido(nr *NodoRaft){
	var resultadoAE Results
	for nodo := 0; nodo < len(nr.Nodos); nodo++ {
		if nodo != nr.Yo {
			nextIndex := nr.NextIndex[i]
			logIndexPrev := nextIndex -1
			if logIndexPrev >= 0 {
				logTermPrev := nr.Log[logIndexPrev].Term
			} else {
				logTermPrev := -1
			}
			Entradas := nr.Log[nextIndex:]
			argumentos := ArgAppendEntries{Term: nr.CurrentTerm,
										   IdLider: nr.Yo,
										   LogIndexPrev: logIndexPrev,
										   LogTermPrev: logTermPrev,
										   Entradas: entradas,
										   CommitLider: nr.CommitIndex,
			}
			err := nr.Nodos[nodo].CallTimeout("NodoRaft.AppendEntries",
											  &argumentos, &resultadoAE,
											  50*Time.Millisecond)
			check.CheckError(err, "Error en la llamada AppendEntries")
			if resultadoAE > nr.CurrentTerm {
				nr.CurrentTerm = resultadoAE.Term
				nr.VotedFor = -1
				nr.SoySeguidor <- true
				return
			}
			if resultadoAE.Exito {
				nr.Mux.Lock()
				nr.NextIndex[nodo] = nextIndex + len(argumentos.Entradas)
				nr.MatchIndex[nodo] = nr.NextIndex[nodo] - 1
				nr.Mux.Unlock()
			}
		}
	}
}

func nuevaEleccion(nr *NodoRaft) {
	lastLogIndex, lastLogTerm := nr.obtenerLastLog()
	argumentos := ArgsPeticionVoto{nr.CurrentTerm, nr.Yo, lastLogIndex,
								   lastLogTerm}
	var resultadoPV RespuestaPeticionVoto
	for nodo := 0; nodo < len(nr.Nodos); nodo++{
		if nodo != nr.Yo {
			go nr.enviarPeticionVoto(nodo, &argumentos, &resultadoPV)
		}
	}
}

func generarTiempoAleatorio() time.Duration {
	return time.Duration(rand.Intn(300)+150 * time.Millisecond)
}

func obtenerLastLog(nr *NodoRaft) (int, int) {
	nr.Mux.Lock()
	longitudLog := len(nr.Log)
	if longitudLog > 0 {
		return (longitudLog - 1, nr.Log[longitudLog - 1].Term)
	} else {
		return -1, -1
	}
	nr.Mux.Unlock()
}
