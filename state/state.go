package state

type Device struct {
	Name string
	IP   string
	Port string
}

// ✅ MUST be capital D
var Devices = make(map[string]Device)