package rpctimeout

import (
	"fmt"
	"net/rpc"
	"strings"
	"time"
)

type HostPort string // Con la forma, "host:puerto", con host mediante DNS o IP

// MakeHostPort crea un nuevo HostPort combinando el nombre del host y el puerto
// en un único string con formato "host:puerto".
func MakeHostPort(host, port string) HostPort {
	return  HostPort(host + port)
}

// Host devuelve el nombre del host de un HostPort, extrayendo la parte anterior
// a los dos puntos (":") del formato "host:puerto".
func (hp HostPort) Host() string {
	return string(hp[:strings.Index(string(hp), ":")])
}

// Port devuelve el puerto de un HostPort, extrayendo la parte posterior
// a los dos puntos (":") del formato "host:puerto".
func (hp HostPort) Port() string {
	return string(hp[strings.Index(string(hp), ":") + 1:])
}

// CallTimeout realiza una llamada remota (RPC) a un método de servicio especificado en
// el servidor identificado por el HostPort, con un tiempo de espera (timeout) definido.
// Devuelve un error si la conexión TCP falla o si se excede el tiempo de espera.
func (hp HostPort) CallTimeout(serviceMethod string, args interface{},
							reply interface{}, timeout time.Duration) error {

	client, err := rpc.Dial("tcp", string(hp))

	if err != nil {
		// fmt.printf("Error dialing endpoint: %v ", err)
		return err  // Devuelve error de conexion TCP
	}

	defer client.Close() // AL FINAL, cerrar la conexion remota  tcp 

	done := client.Go(serviceMethod, args, reply, make(chan *rpc.Call, 1)).Done
	
	select {
		case call := <-done:
			return call.Error
		case <-time.After(timeout):
			return fmt.Errorf(
						  "Timeout in CallTimeout with method: %s, args: %v\n",
						  serviceMethod,
						  args)
	}
}

// StringArrayToHostPortArray convierte un array de strings en un array de HostPort,
// asignando cada string a un nuevo elemento HostPort.
func StringArrayToHostPortArray(stringArray []string) (result []HostPort) {

	for _ , s := range stringArray {
		result = append(result, HostPort(s))
	}

	return
}

// HostPortArrayToString convierte un array de HostPort en un único string,
// uniendo cada elemento HostPort con un espacio como separador.
// Array de HostPort end points a un solo string CON ESPACIO DE SEPARACION
func HostPortArrayToString(hostPortArray []HostPort) (result string) {
	for _, hostPort := range hostPortArray {
		result = result + " " + string(hostPort)
	}

	return
}