package main

import (
	"fmt"
	"net"
	"os"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Println("Client connected.")
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go endpoint")
		return
	}
	endpoint := os.Args[1]
	listen, err := net.Listen("tcp", endpoint)
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		os.Exit(1)
	}
	defer listen.Close()

	fmt.Println("TCP server is running and waiting for connections...")
	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

