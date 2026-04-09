package copycatv1

import (
	"fmt"
	"net"
	"os"
	"time"
)
// ----------------------
// Broadcaster - UDP
// ----------------------
func StartBroadcaster() {
	hostname, _ := os.Hostname()
	addr := net.UDPAddr{
		IP:   net.ParseIP("255.255.255.255"),
		Port: 9999,
	}

	conn, err := net.DialUDP("udp", nil, &addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	for {
		message := fmt.Sprintf("HELLO|%s|9000", hostname)
		conn.Write([]byte(message))
		time.Sleep(2 * time.Second)
	}
}
