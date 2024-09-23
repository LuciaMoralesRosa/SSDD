package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func measureUDPLatency(serverAddr string) {
	// Resolving the UDP server address
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}

	// Dial to the UDP server
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fmt.Println("Error dialing UDP server:", err)
		return
	}
	defer conn.Close()

	// Sending a character to the server
	message := []byte("A")
	fmt.Println("Sending a character to the UDP server...")
	start := time.Now()
	_, err = conn.Write(message)
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	// Receiving a response from the server
	buffer := make([]byte, 1024)
	_, _, err = conn.ReadFrom(buffer)
	if err != nil {
		fmt.Println("Error receiving message:", err)
		return
	}
	duration := time.Since(start)

	// Output the round-trip time
	fmt.Printf("Round-trip time for UDP message: %v\n", duration)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run client.go <server-address>")
		return
	}
	serverAddr := os.Args[1]
	measureUDPLatency(serverAddr)
}

