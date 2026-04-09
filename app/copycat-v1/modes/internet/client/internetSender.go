package internet

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

func sendFile(conn net.Conn, targetID, path string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("File error:", err)
		return
	}
	defer file.Close()

	filename := filepath.Base(path)

	fmt.Fprintf(conn, "SEND|%s|%s\n", targetID, filename)

	io.Copy(conn, file)

	fmt.Println("📤 Sent:", filename)
}
