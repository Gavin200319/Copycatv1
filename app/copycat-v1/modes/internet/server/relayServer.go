package main

import (
	"fmt"
	"net"

)

func main() {
	// IMPORTANT: force IPv4
	listener, err := net.Listen("tcp4", "0.0.0.0:9999")
	if err != nil {
		fmt.Println("⚠️ Port 9999 already in use - relay server may already be running")
		fmt.Println("If you need to restart, kill the existing process first")
		return
	}
	defer listener.Close()

	fmt.Println("🌍 Relay server running on port 9999...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn)
	}
}
