package copycatv1

import (
	"net"
	"fmt"
	"os"
	"copycatv1/app/copycat-v1/modes/lan/handleIncomingFile"
)

// ----------------------
// Receiver - TCP
// ----------------------
func StartReceiver() {
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("📥 Receiver listening on port 9000...")

	os.MkdirAll("ReceivedFiles", 0755)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}
		go copycatv1.HandleIncomingFile(conn) // ✅ FIXED (no wrong package call)
	}
}
