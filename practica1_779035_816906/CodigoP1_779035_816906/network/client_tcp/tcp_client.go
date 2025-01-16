package main

import (
	"fmt"
	"log"
	"net"
	"time"
	"os"
)

func measureTCPDial(serverAddr string) {
	// Measure time for unsuccessful connection (server not running)
	fmt.Println("Attempting to connect to server (expecting failure)...")
	start := time.Now()
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
	} else {
		fmt.Println("Connected successfully")
		conn.Close()
	}
	duration := time.Since(start)
	fmt.Printf("Failed Dial took: %v\n", duration)

	// Wait for server to start
	fmt.Println("\nPlease start the server, then press Enter.")
	fmt.Scanln()

	// Measure time for successful connection (server running)
	fmt.Println("Attempting to connect to server again (expecting success)...")
	start = time.Now()
	conn, err = net.Dial("tcp", serverAddr)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	duration = time.Since(start)
	fmt.Printf("Dial (with server running) took: %v\n", duration)
	conn.Close()
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go endpoint")
		return
	}
	serverAddr := os.Args[1]
	measureTCPDial(serverAddr)
}

