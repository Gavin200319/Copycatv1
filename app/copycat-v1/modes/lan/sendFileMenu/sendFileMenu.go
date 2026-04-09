package copycatv1

import (
	"fmt"
	"strings"
	"bufio"
	"strconv"
	"copycatv1/state"
	"os"
	"net"
	"io"
	"path/filepath"
)

// ----------------------
// Sender - TCP
// ----------------------
func SendFileMenu(reader *bufio.Reader) {
	if len(state.Devices) == 0 {
		fmt.Println("No devices discovered yet.")
		return
	}

	deviceList := []state.Device{}
	fmt.Println("\nDiscovered Devices:")
	i := 1
	for _, d := range state.Devices {
		fmt.Printf("%d. %s (%s)\n", i, d.Name, d.IP)
		deviceList = append(deviceList, d)
		i++
	}

	fmt.Print("Select device number: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num > len(deviceList) {
		fmt.Println("Invalid choice")
		return
	}

	target := deviceList[num-1]

	fmt.Print("Enter file path to send: ")
	path, _ := reader.ReadString('\n')
	path = strings.TrimSpace(path)

	sendFile(target.IP+":"+target.Port, path)
}

func sendFile(addr, path string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("File error:", err)
		return
	}
	defer file.Close()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("Connection error:", err)
		return
	}
	defer conn.Close()

	filename := filepath.Base(file.Name())

	// ✅ Send filename FIRST
	fmt.Fprintln(conn, filename)

	// ✅ Then send file bytes
	_, err = io.Copy(conn, file)
	if err != nil {
		fmt.Println("File send error:", err)
		return
	}

	fmt.Println("📤 File sent:", filename)
}
