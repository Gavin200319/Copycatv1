package copycatv1

import (
	"fmt"
	"copycatv1/state"
)
// ----------------------
// Show Devices
// ----------------------
func ShowDevices() {
	if len(state.Devices) == 0 {
		fmt.Println("No devices discovered yet.")
		return
	}
	fmt.Println("\nDiscovered Devices:")
	for _, d := range state.Devices {
		fmt.Printf("- %s (%s:%s)\n", d.Name, d.IP, d.Port)
	}
}
