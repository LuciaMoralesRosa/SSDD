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
    "ms"
    "sync"
	"github.com/DistributedClocks/GoVector/govec"
)

type Request struct{
    Clock   int
    Pid     int
}

type Reply struct{}

type RASharedDB struct {
	nodes		int	// Numero de nodos
    OurSeqNum   int
    HigSeqNum   int
    OutRepCnt   int
    ReqCS       boolean
    RepDefd     bool[]
    ms          *MessageSystem
    done        chan bool
    chrep       chan bool
	
	// mutex para proteger concurrencia sobre las variables
    Mutex       sync.Mutex
}


func New(me int, usersFile string) (*RASharedDB) {
    messageTypes := []Message{Request, Reply}
    msgs = ms.New(me, usersFile string, messageTypes)
	nodes = contarLineas(usersFile)
    ra := RASharedDB{nodes, 0, 0, 0, false, []int{}, &msgs, make(chan bool),
		make(chan bool), &sync.Mutex{}}
    
	
    return &ra
}

//Pre: Verdad
//Post: Realiza  el  PreProtocol  para el  algoritmo de
//      Ricart-Agrawala Generalizado
func (ra *RASharedDB) PreProtocol(){
	for(int i = 0; ; i++){
		ms.Send(i, Request{me})
	}
    
}

//Pre: Verdad
//Post: Realiza  el  PostProtocol  para el  algoritmo de
//      Ricart-Agrawala Generalizado
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
