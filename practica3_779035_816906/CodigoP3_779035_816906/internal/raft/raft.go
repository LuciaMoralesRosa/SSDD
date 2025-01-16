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
	"raft/internal/comun/rpctimeout"
	"sync"
	"time"
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
	LIDER     = "lider"     // Estado lider
	CANDIDATO = "candidato" // Estado candidato
	SEGUIDOR  = "seguidor"  // Estado seguidor
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
	Term      int // Mandato
	Operacion TipoOperacion
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
	NumVotos int    // Numero de votos recibidos en la eleccion
	Estado   string // Estado del nodo: Lider, Seguidor o Candidato

	// Canales
	Latido      chan bool // Canal para latidos
	SoySeguidor chan bool // Canal para indicar que es seguidor
	SoyLider    chan bool // Canal para indicar que es lider

	// Estado prersistente en todos los servidores - Actualizar antes de
	// responder a RCPs
	CurrentTerm int          // Ultimo mandato que ha visto
	VotedFor    int          // Candidato por el que ha votado el nodo
	Log         []EntradaLog // Entradas de log

	// Estado volatil en servidores
	CommitIndex int // Indice de la entrada mas alta a ser sometida

	// Empleados para raft con fallos - En este asumimos que no hay
	// Estado volatil en servidores
	LastApplied int // Indice de la mayor entrada de log
	// Estado volatil en liders - Reiniciar tras eleccion
	NextIndex map[int]int // Indice de la siguiente entrada de log a enviar
	// Para cada nodo
	MatchIndex map[int]int // indice de la mayor entrada de log

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

	// Inicializacion del resto de variables de la estructura
	nr.NumVotos = 0
	nr.Estado = SEGUIDOR // Empiezan siendo seguidor hasta que haya eleccion
	nr.CurrentTerm = 0
	nr.VotedFor = -1
	nr.CommitIndex = 0
	// Creacion de canales
	nr.Latido = make(chan bool)
	nr.SoySeguidor = make(chan bool)
	nr.SoyLider = make(chan bool)

	// Otros - Para raft con fallos
	nr.LastApplied = -1
	nr.NextIndex = make(map[int]int)
	nr.MatchIndex = make(map[int]int)

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
				kLogOutputDir, logPrefix), os.O_RDWR|os.O_CREATE|os.O_TRUNC,
				0755)
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
	go protocoloRaft(nr) // Maquina de estados de raft

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
	esLider = nr.Yo == nr.IdLider
	if esLider {
		// Si el proceso al que se ha sometido la operacion es el lider
		// Establecer indice y mandato de la entrada
		indice = nr.CommitIndex
		mandato = nr.CurrentTerm
		// Creacion de la nueva entrada
		entradaLog := EntradaLog{mandato, operacion}
		// Añadir la entrada al log del lider
		nr.Mux.Lock()
		nr.Log = append(nr.Log, entradaLog)
		nr.Mux.Unlock()
		nr.Logger.Printf("(Mandato: %d, Op: %s, Clave: %s, Valor: %s)",
			entradaLog.Term, operacion.Operacion, operacion.Clave,
			operacion.Valor)

		var resultados Results
		// Inicializar los guardados de la entrada en los logs de otros nodos
		exitos := 0

		// Soy el lider asi que envio la nueva entrada a todos los nodos
		for i := 0; i < len(nr.Nodos); i++ {
			if i != nr.Yo {
				// Crear la estructura ArgAppendEntries con la nueva entrada
				argumentos := ArgAppendEntries{
					Term:         mandato,
					LeaderId:     nr.Yo,
					Entries:      entradaLog,
					LeaderCommit: indice,
				}
				nr.Nodos[i].CallTimeout("NodoRaft.AppendEntries", &argumentos,
					&resultados, 33*time.Millisecond)
			}
			// Compruebo si el seguidor ha aceptado la entrada
			if resultados.Success {
				exitos++
			} else {
				// El nodo no ha aceptado la entrada asi que comprueba si debe
				// dejar de ser lider
				if nr.CurrentTerm < resultados.Term {
					// Si pertenezco a un mandato anterior, actualizo el mandato
					nr.CurrentTerm = resultados.Term
					// Y dejo de ser líder
					nr.SoySeguidor <- true
				}
			}
		}

		// Comprometer entrada si se debe
		if exitos > len(nr.Nodos)/2 {
			// Se han recibido mas de la mitad de los exitos
			// Es decir, ha sido replicada en la mayoria de los servidores
			nr.CommitIndex++
			valorADevolver = "Commited"
			// Indicamos con un print que se ha comprometido en la maquina de
			// estados -> mejorar en P4
			fmt.Printf("Operacion comprometida\n")
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
	Term        int
	IdCandidato int
}

// Structura de ejemplo de respuesta de RPC PedirVoto,
//
// Recordar
// -----------
// Nombres de campos deben comenzar con letra mayuscula !
type RespuestaPeticionVoto struct {
	// Vuestros datos aqui
	Term     int
	VotoDado bool // True significa que se ha dado el voto al candidato
}

// args *ArgsPeticionVoto -- argumentos de la peticion
//
// reply *RespuestaPeticionVoto -- respuesta
//
// Determina si el nodo vota o no al que ha realizado la peticion
// Le votara si el mandato recibido es mayor al propio, o si son iguales y
// todavia no ha votado por otro candidado. En este caso el nodo pasara a ser
// seguidor si no lo era ya y actualizara sus campos.
// En caso de recibir un mandato menor, respondera con su propio mandato y no le
// votara
func (nr *NodoRaft) PedirVoto(peticion *ArgsPeticionVoto,
	reply *RespuestaPeticionVoto) error {

	nr.Logger.Printf("El nodo %d me pide su voto: [Current term: %d]",
		peticion.IdCandidato, peticion.Term)
	nr.Logger.Printf("Mi estado: [Current term: %d, VotedFor: %d]",
		nr.CurrentTerm, nr.VotedFor)

	// Comprobar la validez del candidato segun el mandato
	if peticion.Term < nr.CurrentTerm {
		// Si el que pide el voto tiene mandato menor al mio no se lo doy
		reply.Term = nr.CurrentTerm
		reply.VotoDado = false
	} else if peticion.Term == nr.CurrentTerm &&
		peticion.IdCandidato != nr.VotedFor {
		// Ya he votado a otro con el mismo mandato
		reply.Term = nr.CurrentTerm
		reply.VotoDado = false
	} else if peticion.Term > nr.CurrentTerm {
		// Si su mandato es mayor, voto por el candidato
		nr.CurrentTerm = peticion.Term
		nr.VotedFor = peticion.IdCandidato
		reply.Term = peticion.Term
		reply.VotoDado = true
		// Si no era seguidor -> paso a serlo
		if nr.Estado != SEGUIDOR {
			nr.SoySeguidor <- true
		}
	}
	return nil
}

type ArgAppendEntries struct {
	Term         int        // Mandato
	LeaderId     int        // Id del nodo lider
	Entries      EntradaLog // Vacio si es latido
	LeaderCommit int        // Ultimo commit del lider
}

type Results struct {
	Term    int  // Mandato actual
	Success bool // True si el nodo acepta la entrada
}

// entrada EntradaLog entrada recibida en la llamada RPC
//
// La funcion devuelve true si la entrada es un EntradaLog vacio, indicando
// que el mensaje es un latido y false si tiene contenido, indicando que es
// una operacion
func esLatido(entrada EntradaLog) bool {
	// Si es un log vacio es que manejamos un latido
	// Si no es vacio, manejamos una operacion
	return entrada == (EntradaLog{})
}

// args *ArgAppendEntries  argumentos para la llamada RPC
//
// results *Result  respuesta RPC
//
// Metodo de tratamiento de llamadas RPC AppendEntries SIN FALLOS
func (nr *NodoRaft) AppendEntries(args *ArgAppendEntries,
	results *Results) error {

	nr.Logger.Printf("Soy %d y he recibido un AppendEntries de %d con mandato"+
		" %d y entradas %v\n",
		nr.Yo, args.LeaderId, args.Term, args.Entries)

	// Comprobar si la llamada ha sido por una entrada o por un latido
	if esLatido(args.Entries) {
		// Comprobar la validez del lider y actualizar campos
		switch {
		case args.Term > nr.CurrentTerm:
			// El mandato recibido es mayor, asi que es de un lider
			// Actualizo mis datos
			nr.IdLider = args.LeaderId
			nr.CurrentTerm = args.Term
			results.Term = nr.CurrentTerm
			if nr.Estado == LIDER {
				// Si era lider, dejo de serlo
				nr.SoySeguidor <- true
			}
			if nr.Estado == CANDIDATO || nr.Estado == SEGUIDOR {
				// Notifico que he recibido el latido
				nr.Latido <- true
			}
		case args.Term == nr.CurrentTerm:
			// Lo reconozco como lider y notifico el latido
			nr.IdLider = args.LeaderId
			results.Term = nr.CurrentTerm
			nr.Latido <- true
		case args.Term < nr.CurrentTerm:
			// Yo tengo mayor mandato, asi que no reconozco el latido
			results.Term = nr.CurrentTerm
		}
	} else {
		// No es un latido -> Manejo de operacion
		nr.manejarOperacion(args, results)
	}
	return nil
}

// args *ArgAppendEntries  argumentos de la llamada RPC
//
// results *Result  respuesta RPC
//
// En esta primera version, añade la entrada al log
// Habra que modificarlo cuando se tengan en cuenta los errores
func (nr *NodoRaft) manejarOperacion(args *ArgAppendEntries, results *Results) {
	// Crear entrada dado el mandato y la operacion
	entradaLog := EntradaLog{args.Entries.Term, args.Entries.Operacion}

	// Añadir entrada al log del nodo
	nr.Mux.Lock() // Para evitar condiciones de carrera en el Log
	nr.Log = append(nr.Log, entradaLog)
	nr.Mux.Unlock()
	nr.Logger.Printf("(Mandato: %d, Op: %s, Clave: %s, Valor: %s)",
		args.Entries.Term, args.Entries.Operacion.Operacion,
		args.Entries.Operacion.Clave, args.Entries.Operacion.Valor)

	// Indicar que el nodo ha aceptado la entrada correctamente
	results.Success = true
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

	nr.Mux.Lock()
	defer nr.Mux.Unlock()

	// Llamada RPC para pedir el voto del candidato
	err := nr.Nodos[nodo].CallTimeout("NodoRaft.PedirVoto", args, reply,
		33*time.Millisecond)
	if err != nil {
		return false
	} else {
		// Comprobar la respuesta del nodo y mi validez como lider
		if reply.Term > nr.CurrentTerm {
			// Si el que me responde tiene mandato mayor, actualizado mi mandato
			// al suyo y paso a ser seguidor
			nr.SoySeguidor <- true
			nr.CurrentTerm = reply.Term
			nr.VotedFor = -1
		} else if reply.VotoDado {
			nr.NumVotos++ // Me nan dado el voto
			// Si he conseguido mayoria paso a ser lider
			if nr.NumVotos > len(nr.Nodos)/2 {
				nr.SoyLider <- true
			}
			// Mas restricciones en la siguiente practica
		}
		return true
	}

}

// nr *NodoRaft Estructura nr del nodo
//
// Realiza la funcionalidad cuando el nodo es seguidor
func soySeguidor(nr *NodoRaft) {
	select {
	case <-nr.Latido:
		// Llega latido asi que sigo siendo seguidor
		nr.Estado = SEGUIDOR
	case <-time.After(timeoutAleatorio()):
		// Si expira timeout se inicia el proceso de elección de líder
		nr.IdLider = -1
		nr.Estado = CANDIDATO
		nr.Logger.Printf("Soy %d y ha expirado el timeout. Así que me "+
			"presento como candidato.\n", nr.Yo)
	}
}

// nr *NodoRaft Estructura nr del nodo
//
// Realiza la funcionalidad cuando el nodo es candidato
func soyCandidato(nr *NodoRaft) {
	// Se inicia el timer de elección y se empieza una eleccion
	timer := time.NewTimer(timeoutAleatorio())
	comenzarEleccion(nr)
	select {
	case <-nr.SoySeguidor:
		// Recibo un latido o un mensaje de ser seguidor, cambio a seguidor
		nr.Estado = SEGUIDOR
	case <-nr.Latido:
		// Recibo un latido o un mensaje de ser seguidor, cambio a seguidor
		nr.Estado = SEGUIDOR
	case <-nr.SoyLider:
		//Ha recibido mayoria de votos
		nr.Estado = LIDER
	case <-timer.C:
		//Si expira el timeout empieza me pongo como candidato y se empezara una
		// nueva elección
		nr.Estado = CANDIDATO
	}
}

// nr *NodoRaft Estructura nr del nodo
//
// Realiza la funcionalidad cuando el nodo es lider
func soyLider(nr *NodoRaft) {
	nr.IdLider = nr.Yo
	// La frecuencia de látidos no debe ser superior a 20 veces por segundo
	timer := time.NewTimer(50 * time.Millisecond)
	enviarLatidosATodos(nr)
	select {
	case <-timer.C:
		// Se vuelven a enviar los latidos pasados 50 milisegundos
		nr.Estado = LIDER
	case <-nr.SoySeguidor:
		// Hay otro con mandato mayor
		nr.Estado = SEGUIDOR
	}
}

// nr *NodoRaft Estructura nr del nodo
//
// Maquina de estados de raft
func protocoloRaft(nr *NodoRaft) {
	for {
		for nr.Estado == SEGUIDOR {
			soySeguidor(nr)
		}
		for nr.Estado == CANDIDATO {
			soyCandidato(nr)
		}
		for nr.Estado == LIDER {
			soyLider(nr)
		}
	}
}

// nr *NodoRaft Estructura nr del nodo
//
// Inicia una eleccion votandose a si mismo y pidiendo votos al resto de nodos
func comenzarEleccion(nr *NodoRaft) {
	// Soy candidato asi que me presento yo mismo
	nr.Logger.Printf("Soy %d y empiezo elección", nr.Yo)
	nr.CurrentTerm++
	nr.VotedFor = nr.Yo
	nr.NumVotos = 1

	// Creacion de la peticion con mi mandato actual y mi id
	peticionParaLider := ArgsPeticionVoto{
		Term:        nr.CurrentTerm,
		IdCandidato: nr.Yo,
	}
	var respuesta RespuestaPeticionVoto

	// Enviar la peticion a todos los nodos
	for i := 0; i < len(nr.Nodos); i++ {
		if i != nr.Yo {
			go nr.enviarPeticionVoto(i, &peticionParaLider, &respuesta)
		}
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
func (nr *NodoRaft) enviarLatido(nodo int, args *ArgAppendEntries,
	results *Results) bool {
	// Llamada RPC de envio de latido
	err := nr.Nodos[nodo].CallTimeout("NodoRaft.AppendEntries", args, results,
		33*time.Millisecond)
	if err != nil {
		return false
	} else {
		// Comprobar mi validez como lider dado el mandato del nodo
		if results.Term > nr.CurrentTerm {
			// El nodo al que envio el latido tiene mayor mandato
			// Actualizo mi mandato al nuevo
			nr.CurrentTerm = results.Term
			// Como estoy enviando latidos soy lider pero debo dejar de serlo
			nr.IdLider = -1
			nr.SoySeguidor <- true
		}
		return true
	}
}

// nr *NodoRaft Estructura nr del nodo
//
// Envia un latido a todos los nodos menos a si mismo
func enviarLatidosATodos(nr *NodoRaft) {
	nr.Logger.Printf("Soy %d y envío latidos", nr.Yo)

	var resultados Results
	// Enviar un latido a todos los nodos menos a mi mismo
	for i := 0; i < len(nr.Nodos); i++ {
		if i != nr.Yo {
			// Crear el latido con entrada vacia
			argumentos := ArgAppendEntries{
				Term:         nr.CurrentTerm,
				LeaderId:     nr.Yo,
				Entries:      EntradaLog{},
				LeaderCommit: nr.CommitIndex,
			}
			go nr.enviarLatido(i, &argumentos, &resultados)
		}
	}
}

// Devuelve un timeout aleatorio entre 150 y 450 ms
func timeoutAleatorio() time.Duration {
	return time.Duration(rand.Intn(300)+150) * time.Millisecond
}
