package testintegracionraft1

import (
	"fmt"
	"os"
	"path/filepath"
	"raft/internal/comun/check"
	"strconv"
	"testing"
	"time"

	"raft/internal/comun/rpctimeout"
	"raft/internal/despliegue"
	"raft/internal/raft"
)

/*
Ejecucion normal
go run main.go 0 192.168.3.2:31110 192.168.3.3:31111 192.168.3.4:31112
go run main.go 1 192.168.3.2:31110 192.168.3.3:31111 192.168.3.4:31112
go run main.go 2 192.168.3.2:31110 192.168.3.3:31111 192.168.3.4:31112

Ejecucion de pruebas en otras maquinas por error en 192.168.3.1 y 192.168.3.2
go run main.go 0 192.168.3.4:31110 192.168.3.6:31111 192.168.3.7:31112
go run main.go 1 192.168.3.4:31110 192.168.3.6:31111 192.168.3.7:31112
go run main.go 2 192.168.3.4:31110 192.168.3.6:31111 192.168.3.7:31112
*/

const (
	//hosts
	// Ejecucion normal en maquinas
	MAQUINA1 = "192.168.3.2"
	MAQUINA2 = "192.168.3.3"
	MAQUINA3 = "192.168.3.4"

	// Ejecucion especial en maquinas por error en 192.168.3.1 y 192.168.3.2
	//MAQUINA1 = "192.168.3.4"
	//MAQUINA2 = "192.168.3.6"
	//MAQUINA3 = "192.168.3.7"

	// Ejecucion en local
	//MAQUINA1 = "127.0.0.1"
	//MAQUINA2 = "127.0.0.1"
	//MAQUINA3 = "127.0.0.1"

	//puertos
	PUERTOREPLICA1 = "31110"
	PUERTOREPLICA2 = "31111"
	PUERTOREPLICA3 = "31112"

	//nodos replicas
	REPLICA1 = MAQUINA1 + ":" + PUERTOREPLICA1
	REPLICA2 = MAQUINA2 + ":" + PUERTOREPLICA2
	REPLICA3 = MAQUINA3 + ":" + PUERTOREPLICA3

	//numero de nodos
	numeroNodos = 3

	// paquete main de ejecutables relativos a PATH previo
	EXECREPLICA = "cmd/srvraft/main.go"

	// comandos completo a ejecutar en máquinas remota con ssh. Ejemplo :
	// 				cd $HOME/raft; go run cmd/srvraft/main.go 127.0.0.1:29001

	// Ubicar, en esta constante, nombre de fichero de vuestra clave privada local
	// emparejada con la clave pública en authorized_keys de máquinas remotas

	PRIVKEYFILE = "id_rsa"
)

// PATH de los ejecutables de modulo golang de servicio Raft
// var PATH string = filepath.Join(os.Getenv("HOME"), "tmp", "p3", "raft")
var PATH string = filepath.Join(os.Getenv("HOME"), "practica3")

// go run cmd/srvraft/main.go 0 127.0.0.1:29001 127.0.0.1:29002 127.0.0.1:29003
var EXECREPLICACMD string = "cd " + PATH + "; go run " + EXECREPLICA

// TEST primer rango
func TestPrimerasPruebas(t *testing.T) { // (m *testing.M) {
	// <setup code>
	// Crear canal de resultados de ejecuciones ssh en maquinas remotas
	cfg := makeCfgDespliegue(t,
		numeroNodos,
		[]string{REPLICA1, REPLICA2, REPLICA3},
		[]bool{true, true, true})

	// tear down code
	// eliminar procesos en máquinas remotas
	defer cfg.stop()

	// Run test sequence

	// Test1 : No debería haber ningun primario, si SV no ha recibido aún latidos
	t.Run("T1:soloArranqueYparada",
		func(t *testing.T) { cfg.soloArranqueYparadaTest1(t) })

	// Test2 : No debería haber ningun primario, si SV no ha recibido aún latidos
	t.Run("T2:ElegirPrimerLider",
		func(t *testing.T) { cfg.elegirPrimerLiderTest2(t) })

	// Test3: tenemos el primer primario correcto
	t.Run("T3:FalloAnteriorElegirNuevoLider",
		func(t *testing.T) { cfg.falloAnteriorElegirNuevoLiderTest3(t) })

	// Test4: Tres operaciones comprometidas en configuración estable
	t.Run("T4:tresOperacionesComprometidasEstable",
		func(t *testing.T) { cfg.tresOperacionesComprometidasEstable(t) })
}

// ---------------------------------------------------------------------
//
// Canal de resultados de ejecución de comandos ssh remotos
type canalResultados chan string

func (cr canalResultados) stop() {
	close(cr)

	// Leer las salidas obtenidos de los comandos ssh ejecutados
	for s := range cr {
		fmt.Println(s)
	}
}

// ---------------------------------------------------------------------
// Operativa en configuracion de despliegue y pruebas asociadas
type configDespliegue struct {
	t           *testing.T
	conectados  []bool
	numReplicas int
	nodosRaft   []rpctimeout.HostPort
	cr          canalResultados
}

// Crear una configuracion de despliegue
func makeCfgDespliegue(t *testing.T, n int, nodosraft []string,
	conectados []bool) *configDespliegue {
	cfg := &configDespliegue{}
	cfg.t = t
	cfg.conectados = conectados
	cfg.numReplicas = n
	cfg.nodosRaft = rpctimeout.StringArrayToHostPortArray(nodosraft)
	cfg.cr = make(canalResultados, 2000)

	return cfg
}

func (cfg *configDespliegue) stop() {
	//cfg.stopDistributedProcesses()

	time.Sleep(500 * time.Millisecond)
	cfg.cr.stop()
}

// --------------------------------------------------------------------------
// FUNCIONES DE SUBTESTS

// Se pone en marcha una replica ?? - 3 NODOS RAFT
func (cfg *configDespliegue) soloArranqueYparadaTest1(t *testing.T) {
	//t.Skip("SKIPPED soloArranqueYparadaTest1")

	fmt.Println(t.Name(), ".....................")

	cfg.t = t // Actualizar la estructura de datos de tests para errores
	// Poner en marcha replicas en remoto con un tiempo de espera incluido
	cfg.startDistributedProcesses()
	//time.Sleep(2500 * time.Millisecond)

	// Comprobar estados replicas en todos los nodos
	for i := 0; i < numeroNodos; i++ {
		cfg.comprobarEstadoRemoto(i, 0, false, -1)
	}
	cfg.stopDistributedProcesses()

	fmt.Println(".............", t.Name(), "Superado")
}

// Primer lider en marcha - 3 NODOS RAFT
func (cfg *configDespliegue) elegirPrimerLiderTest2(t *testing.T) {
	//t.Skip("SKIPPED ElegirPrimerLiderTest2")

	fmt.Println(t.Name(), ".....................")
	cfg.startDistributedProcesses()
	//time.Sleep(2500 * time.Millisecond)

	// Comprobar si se ha obtenido un lider
	fmt.Printf("Comprobando si hay un lider\n")
	lider := cfg.pruebaUnLider(numeroNodos)
	fmt.Printf("El lider es el nodo %d\n", lider)

	// Parar réplicas alamcenamiento en remoto
	cfg.stopDistributedProcesses() // Parametros
	fmt.Println(".............", t.Name(), "Superado")
}

// Fallo de un primer lider y reeleccion de uno nuevo - 3 NODOS RAFT
func (cfg *configDespliegue) falloAnteriorElegirNuevoLiderTest3(t *testing.T) {
	//t.Skip("SKIPPED FalloAnteriorElegirNuevoLiderTest3")

	fmt.Println(t.Name(), ".....................")
	cfg.startDistributedProcesses()

	lider := cfg.pruebaUnLider(numeroNodos)
	fmt.Printf("El lider es el nodo %d\n", lider)

	// Desconectar lider
	cfg.pararLider(lider)
	fmt.Printf("Se ha parado el nodo %d\n", lider)

	cfg.activarNodosDesconectados()
	fmt.Printf("Comprobar nuevo lider\n")
	nuevoLider := cfg.pruebaUnLider(numeroNodos)
	fmt.Printf("El nuevo lider es el nodo %d\n", nuevoLider)

	// Parar réplicas almacenamiento en remoto
	cfg.stopDistributedProcesses() //parametros
	fmt.Println(".............", t.Name(), "Superado")
}

// 3 operaciones comprometidas con situacion estable y sin fallos - 3 NODOS RAFT
func (cfg *configDespliegue) tresOperacionesComprometidasEstable(t *testing.T) {
	//t.Skip("SKIPPED tresOperacionesComprometidasEstable")

	fmt.Println(t.Name(), ".....................")
	cfg.startDistributedProcesses()

	lider := cfg.pruebaUnLider(numeroNodos)
	fmt.Printf("El lider es el nodo %d\n", lider)

	// someterOperaciones
	cfg.someterOperacion(lider, 0, "escribir", "", "primeraEscritura")
	cfg.someterOperacion(lider, 1, "leer", "", "")
	cfg.someterOperacion(lider, 2, "escribir", "", "segundaEscritura")

	// Parar réplicas almacenamiento en remoto
	cfg.stopDistributedProcesses() //parametros
	fmt.Println(".............", t.Name(), "Superado")
}

// --------------------------------------------------------------------------
// FUNCIONES DE APOYO
// Comprobar que hay un solo lider
// probar varias veces si se necesitan reelecciones
func (cfg *configDespliegue) pruebaUnLider(numreplicas int) int {
	for iters := 0; iters < 10; iters++ {
		time.Sleep(500 * time.Millisecond)
		mapaLideres := make(map[int][]int)
		for i := 0; i < numreplicas; i++ {
			if cfg.conectados[i] {
				if _, mandato, eslider, _ := cfg.obtenerEstadoRemoto(i); eslider {
					mapaLideres[mandato] = append(mapaLideres[mandato], i)
				}
			}
		}

		ultimoMandatoConLider := -1
		for mandato, lideres := range mapaLideres {
			if len(lideres) > 1 {
				cfg.t.Fatalf("mandato %d tiene %d (>1) lideres",
					mandato, len(lideres))
			}
			if mandato > ultimoMandatoConLider {
				ultimoMandatoConLider = mandato
			}
		}

		if len(mapaLideres) != 0 {
			return mapaLideres[ultimoMandatoConLider][0] // Termina
		}
	}
	cfg.t.Fatalf("un lider esperado, ninguno obtenido")
	return -1 // Termina
}

func (cfg *configDespliegue) obtenerEstadoRemoto(
	indiceNodo int) (int, int, bool, int) {
	var reply raft.EstadoRemoto
	/*err :=*/ cfg.nodosRaft[indiceNodo].CallTimeout("NodoRaft.ObtenerEstadoNodo",
		raft.Vacio{}, &reply, 10*time.Millisecond)
	// check.CheckError(err, "Error en llamada RPC ObtenerEstadoRemoto")

	return reply.IdNodo, reply.Mandato, reply.EsLider, reply.IdLider
}

// start  gestor de vistas; mapa de replicas y maquinas donde ubicarlos;
// y lista clientes (host:puerto)
func (cfg *configDespliegue) startDistributedProcesses() {
	//cfg.t.Log("Before starting following distributed processes: ", cfg.nodosRaft)

	for i, endPoint := range cfg.nodosRaft {
		despliegue.ExecMutipleHosts(EXECREPLICACMD+
			" "+strconv.Itoa(i)+" "+
			rpctimeout.HostPortArrayToString(cfg.nodosRaft),
			[]string{endPoint.Host()}, cfg.cr, PRIVKEYFILE)

		// dar tiempo para se establezcan las replicas
		time.Sleep(50 * time.Millisecond)
	}

	// Tiempo para iniciar todo y elegir un líder
	time.Sleep(5000 * time.Millisecond)
}

func (cfg *configDespliegue) stopDistributedProcesses() {
	var reply raft.Vacio

	for _, endPoint := range cfg.nodosRaft {
		/*err :=*/ endPoint.CallTimeout("NodoRaft.ParaNodo",
			raft.Vacio{}, &reply, 10*time.Millisecond)
		//check.CheckError(err, "Error en llamada RPC Para nodo")
	}
}

// Comprobar estado remoto de un nodo con respecto a un estado prefijado
func (cfg *configDespliegue) comprobarEstadoRemoto(idNodoDeseado int,
	mandatoDeseado int, esLiderDeseado bool, IdLiderDeseado int) {
	idNodo, mandato, esLider, idLider := cfg.obtenerEstadoRemoto(idNodoDeseado)

	cfg.t.Log("Params: ", idNodoDeseado, mandatoDeseado, esLiderDeseado,
		IdLiderDeseado, "\n")
	cfg.t.Log("Estado replica 0: ", idNodo, mandato, esLider, idLider, "\n")

	if idNodo != idNodoDeseado || mandato != mandatoDeseado ||
		esLider != esLiderDeseado || idLider != IdLiderDeseado {
		cfg.t.Fatalf("Estado incorrecto en replica %d en subtest %s",
			idNodoDeseado, cfg.t.Name())
	}

}

// MIS FUNCIONES ---------------------------------------------------------------

func (cfg *configDespliegue) activarNodosDesconectados() {
	for i, endPoint := range cfg.nodosRaft {
		if !cfg.conectados[i] {
			despliegue.ExecMutipleHosts(EXECREPLICACMD+" "+strconv.Itoa(i)+
				rpctimeout.HostPortArrayToString(cfg.nodosRaft),
				[]string{endPoint.Host()}, cfg.cr, PRIVKEYFILE)
			cfg.conectados[i] = true
		}
	}
	time.Sleep(10000 * time.Millisecond)
}

func (cfg *configDespliegue) pararLider(indiceLider int) {
	var reply raft.Vacio
	for i, endPoint := range cfg.nodosRaft {
		if i == indiceLider {
			err := endPoint.CallTimeout("NodoRaft.ParaNodo", raft.Vacio{},
				&reply, 10*time.Millisecond)
			check.CheckError(err, "Error en llamada RPC Para nodo")
			cfg.conectados[i] = false
		}
	}
}

func (cfg *configDespliegue) someterOperacion(idLider int, indice int,
	operacion string, clave string, valor string) {
	var reply raft.ResultadoRemoto
	peticion := raft.TipoOperacion{
		Operacion: operacion,
		Clave:     clave,
		Valor:     valor,
	}
	err := cfg.nodosRaft[idLider].CallTimeout("NodoRaft.SometerOperacionRaft",
		peticion, &reply, 50*time.Millisecond)
	check.CheckError(err, "Error en llamada RPC SometerOperacionRaft")

	if reply.IndiceRegistro != indice || idLider != reply.IdLider {
		cfg.t.Fatalf("Operacion no sometida correctamente en indice %d en "+
			"subtest %s", indice, cfg.t.Name())
	}
}
