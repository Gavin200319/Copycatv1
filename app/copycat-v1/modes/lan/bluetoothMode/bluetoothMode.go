package copycatv1

import(
	"fmt"
	"bufio"
	"os"
	"os/exec"
	"time"
	"strings"
)
// ----------------------
// Bluetooth Mode with automatic pairing and passkey confirmation
// ----------------------
func BluetoothMode(reader *bufio.Reader) {
	fmt.Println("⚠️ Make sure the target device is in pairing mode, discoverable, and has agent running.")

	// Automatically start OBEX server on receiver side (if this is the receiver)
	go func() {
		os.MkdirAll("ReceivedFiles", 0755)
		cmd := exec.Command("obexpushd", "-B", "-o", "ReceivedFiles")
		cmd.Start() // runs in background
	}()

	// Scan for nearby devices
	fmt.Println("Scanning for nearby Bluetooth devices for 10 seconds...")
	exec.Command("bluetoothctl", "scan", "on").Run()
	time.Sleep(10 * time.Second)
	exec.Command("bluetoothctl", "scan", "off").Run()

	// List discovered devices
	out, err := exec.Command("bluetoothctl", "devices").Output()
	if err != nil {
		fmt.Println("Error listing devices:", err)
		return
	}

	lines := strings.Split(string(out), "\n")
	filteredDevices := []struct {
		Name string
		MAC  string
	}{}

	// Filter devices: only laptops/phones, skip audio/headset
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}
		mac := parts[1]
		name := parts[2]
		lower := strings.ToLower(name)
		if strings.Contains(lower, "headset") || strings.Contains(lower, "speaker") || strings.Contains(lower, "audio") {
			continue
		}
		filteredDevices = append(filteredDevices, struct {
			Name string
			MAC  string
		}{Name: name, MAC: mac})
	}

	if len(filteredDevices) == 0 {
		fmt.Println("No nearby devices found. Make sure the target device is discoverable and running agent.")
		return
	}

	// Show filtered devices
	fmt.Println("\nNearby devices available for pairing:")
	for i, d := range filteredDevices {
		fmt.Printf("%d. %s (%s)\n", i+1, d.Name, d.MAC)
	}

	fmt.Print("Select device number to pair: ")
	var num int
	fmt.Scanln(&num)
	if num < 1 || num > len(filteredDevices) {
		fmt.Println("Invalid choice")
		return
	}

	target := filteredDevices[num-1]
	fmt.Printf("Pairing with %s (%s)...\n", target.Name, target.MAC)

	// Wait until the target device is pairable and pair
	fmt.Println("⏳ Waiting for device to be ready for pairing...")
	for {
		info, _ := exec.Command("bluetoothctl", "info", target.MAC).Output()
		if strings.Contains(string(info), "Paired: yes") {
			fmt.Println("✅ Already paired with", target.Name)
			break
		}

		// Try pairing repeatedly until the agent confirms the passkey
		pairCmd := exec.Command("bluetoothctl", "pair", target.MAC)
		pairOutput, _ := pairCmd.CombinedOutput()
		outputStr := string(pairOutput)
		if strings.Contains(outputStr, "Failed") {
			time.Sleep(2 * time.Second)
			continue
		}
		if strings.Contains(outputStr, "Pairing successful") {
			break
		}
		time.Sleep(2 * time.Second)
	}

	// Trust the device
	exec.Command("bluetoothctl", "trust", target.MAC).Run()
	fmt.Println("✅ Successfully paired and trusted:", target.Name)

	// Send file
	fmt.Print("Enter file path to send: ")
	filePath, _ := reader.ReadString('\n')
	filePath = strings.TrimSpace(filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Println("File does not exist.")
		return
	}

	cmd := exec.Command("bluetooth-sendto", "--device="+target.MAC, filePath)
	err = cmd.Start()
	if err != nil {
		fmt.Println("Error launching Bluetooth send:", err)
		return
	}

	fmt.Println("📤 Bluetooth file transfer started. Complete the transfer on the receiving device.")
}
