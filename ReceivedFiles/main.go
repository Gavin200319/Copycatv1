package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	startNetworkMode "copycatv1/app/copycat-v1/modes/lan/startNetworkMode"
	bluetoothMode "copycatv1/app/copycat-v1/modes/lan/bluetoothMode"
	internetMode "copycatv1/app/copycat-v1/modes/internet/client"
)


func main() {
	//reading user input
	reader := bufio.NewReader(os.Stdin)

	// displays the menu so that you can choose how devices will connect
	fmt.Println("--- COPYCAT NETWORK SELECTION ---")
	fmt.Println("Choose network type for file transfer:")
	fmt.Println("1. Internet (LAN/Wi-Fi)")
	fmt.Println("2. Wi-Fi Hotspot (offline LAN)")
	fmt.Println("3. Bluetooth (offline, short-range)")
	fmt.Println("4. Internet:cloud mode.)")
	//asking for input
	fmt.Print("Enter choice (1-4): ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)// cleaning input

	// decision making
	switch choice {
	case "1":
		fmt.Println("📶 Starting COPYCAT on Internet (LAN/Wi-Fi)...")
		startNetworkMode.StartNetworkMode(reader)
	case "2":
		fmt.Println("📡 Starting COPYCAT on Wi-Fi Hotspot...")
		startNetworkMode.StartNetworkMode(reader)
	case "3":
		fmt.Println("🔵 Starting COPYCAT Bluetooth mode...")
		bluetoothMode.BluetoothMode(reader)
	case "4":
		fmt.Println("🔵 Starting COPYCAT cloud mode..")
		internetMode.Start(reader)
	default:
		fmt.Println("Invalid choice. Exiting.")
	}
}
