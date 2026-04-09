package copycatv1

import (
	"net"
	"strings"
	"copycatv1/state"
	
)
// ----------------------
// Listener - UDP
// ----------------------
func StartListener() {
	addr := net.UDPAddr{
		Port: 9999,
		IP:   net.IPv4zero,
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	buf := make([]byte, 1024)

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		msg := string(buf[:n])
		parts := strings.Split(msg, "|")
		if len(parts) != 3 || parts[0] != "HELLO" {
			continue
		}

		name := parts[1]
		port := parts[2]
		ip := remoteAddr.IP.String()

		state.Devices[name] = state.Device{Name: name, IP: ip, Port: port}
	}
}
