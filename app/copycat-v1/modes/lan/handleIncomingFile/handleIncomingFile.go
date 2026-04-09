package copycatv1

import (
	"fmt"
	"os"
	"net"
	"io"
	"bufio"
	"strings"
)

func HandleIncomingFile(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// ✅ Read filename safely (supports spaces)
	filename, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Filename read error:", err)
		return
	}
	filename = strings.TrimSpace(filename)

	file, err := os.Create("ReceivedFiles/" + filename)
	if err != nil {
		fmt.Println("File creation error:", err)
		return
	}
	defer file.Close()

	// ✅ Copy file data correctly
	_, err = io.Copy(file, reader)
	if err != nil {
		fmt.Println("File receive error:", err)
		return
	}

	fmt.Println("✅ Received file:", "ReceivedFiles/"+filename)
}
