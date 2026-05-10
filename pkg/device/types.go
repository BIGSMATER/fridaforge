package device

// ConnectType 表示设备连接类型。
type ConnectType string

const (
	// ConnectTypeUSB 表示通过 USB 线缆连接。
	ConnectTypeUSB ConnectType = "usb"
	// ConnectTypeNetwork 表示通过网络（TCP）连接。
	ConnectTypeNetwork ConnectType = "network"
	// ConnectTypeEmulator 表示 Android 模拟器设备。
	ConnectTypeEmulator ConnectType = "emulator"
)

// Device 表示一台已连接的 Frida 可用设备。
type Device struct {
	ID          string
	Name        string
	ConnectType ConnectType
}
