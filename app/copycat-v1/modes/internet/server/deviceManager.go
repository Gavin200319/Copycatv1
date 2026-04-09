package main

import (
	"net"
	"sync"
)

var (
	mu      sync.Mutex
	clients = make(map[string]net.Conn)
)
func registerDevice(id string, conn net.Conn) {
	mu.Lock()
	defer mu.Unlock()
	clients[id] = conn
}

func getDevice(id string) net.Conn {
	mu.Lock()
	defer mu.Unlock()
	return clients[id]
}

func listDevices() map[string]net.Conn {
	mu.Lock()
	defer mu.Unlock()

	copy := make(map[string]net.Conn)
	for k, v := range clients {
		copy[k] = v
	}
	return copy
}

func removeDevice(id string) {
	mu.Lock()
	defer mu.Unlock()
	delete(clients, id)
}
