package rpctimeout

import (
	"fmt"
	"net/rpc"
	"strings"
	"time"
)

type HostPort string // Con la forma, "host:puerto", con host mediante DNS o IP

func MakeHostPort(host, port string) HostPort {
	return  HostPort(host + port)
}

func (hp HostPort) Host() string {
	return string(hp[:strings.Index(string(hp), ":")])
}

func (hp HostPort) Port() string {
	return string(hp[strings.Index(string(hp), ":") + 1:])
}


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


func StringArrayToHostPortArray(stringArray []string) (result []HostPort) {

	for _ , s := range stringArray {
		result = append(result, HostPort(s))
	}

	return
}

// Array de HostPort end points a un solo string CON ESPACIO DE SEPARACION
func HostPortArrayToString(hostPortArray []HostPort) (result string) {
	for _, hostPort := range hostPortArray {
		result = result + " " + string(hostPort)
	}

	return
}