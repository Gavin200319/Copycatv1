package internet

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

func Start(reader *bufio.Reader) {
	// CHANGE THIS if using VPS or LAN IP
	serverAddr := "4.tcp.eu.ngrok.io:17821"

	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("Connection failed:", err)
		return
	}

	fmt.Print("Enter device ID: ")
	id, _ := reader.ReadString('\n')
	id = strings.TrimSpace(id)

	fmt.Fprintf(conn, "REGISTER|%s\n", id)

	go receiveFiles(conn)

	for {
		fmt.Println("\n=== Send Options ===")
		fmt.Println("1. Send File")
		fmt.Println("2. Stream Screen (PrintScreen Video)")
		fmt.Println("3. List Devices")
		fmt.Println("4. Exit")
		fmt.Print("Choose: ")

		choice, _ := reader.ReadString('\n')
		
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			// Send File
			fmt.Print("Target Device ID: ")
			target, _ := reader.ReadString('\n')
			target = strings.TrimSpace(target)

			fmt.Print("File path: ")
			path, _ := reader.ReadString('\n')
			path = strings.TrimSpace(path)

			sendFile(conn, target, path)

		case "2":
			// Stream Screen Video
			fmt.Print("Target Device ID: ")
			target, _ := reader.ReadString('\n')
			target = strings.TrimSpace(target)
			
			fmt.Println("Starting screen stream... Press Ctrl+C to stop")
			streamScreen(conn, target)

		case "3":
			// List Devices
			devices := getDeviceList(conn)

			fmt.Println("\n📱 Devices:")
			for i, id := range devices {
   				 fmt.Println(i, ":", id)
				}

			fmt.Print("Select device index: ")

			var index int
			fmt.Scanln(&index)

			target := devices[index]

			fmt.Println("Selected:", target)
		case "4":
			return

		}
	}

}
func getDeviceList(conn net.Conn) []string {
    conn.Write([]byte("LIST_DEVICES"))

    buf := make([]byte, 4096)
    n, _ := conn.Read(buf)

    data := string(buf[:n])

    if data == "" {
        return []string{}
    }

    return strings.Split(data, "|")
}
