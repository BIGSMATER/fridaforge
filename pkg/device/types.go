package device

// ConnectType 表示设备连接类型。
type ConnectType string

const (
	ConnectTypeUSB       ConnectType = "usb"
	ConnectTypeNetwork   ConnectType = "network"
	ConnectTypeEmulator  ConnectType = "emulator"
)

// Device 表示一台已连接的 Frida 可用设备。
type Device struct {
	ID          string
	Name        string
	ConnectType ConnectType
}
