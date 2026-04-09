package copycatv1

import (
	"fmt"
	"strings"
	"bufio"
	showDevices "copycatv1/app/copycat-v1/modes/lan/showDevices"
	sendFileMenu "copycatv1/app/copycat-v1/modes/lan/sendFileMenu"
	
)
// ----------------------
// Menu Loop (LAN / Hotspot)
// ----------------------
func MenuLoop(reader *bufio.Reader) {
	for {
		fmt.Println("\n--- COPYCAT MENU ---")
		fmt.Println("1. Show Devices")
		fmt.Println("2. Send File")
		fmt.Println("3. Exit")
		fmt.Print("Choose: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			showDevices.ShowDevices()
		case "2":
			sendFileMenu.SendFileMenu(reader)
		case "3":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid option. Try again.")
		}
	}
}
