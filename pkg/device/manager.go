package device

import "context"

// DeviceLister 定义设备发现接口。
// M1 使用桩实现（StubDeviceLister），M2 替换为真实的 Frida 实现。
type DeviceLister interface {
	ListDevices(ctx context.Context) ([]Device, error)
}

// StubDeviceLister 是 DeviceLister 的桩实现，返回硬编码设备列表。
// 用于 M1 的独立测试和 CLI 骨架验证。
type StubDeviceLister struct{}

// ListDevices 返回预设的设备列表。
func (s *StubDeviceLister) ListDevices(ctx context.Context) ([]Device, error) {
	return []Device{
		{ID: "emulator-5554", Name: "Android Emulator 5554", ConnectType: ConnectTypeEmulator},
		{ID: "R5CT1234ABCD", Name: "Samsung Galaxy S21", ConnectType: ConnectTypeUSB},
	}, nil
}

// 编译时检查：*StubDeviceLister 实现了 DeviceLister 接口。
var _ DeviceLister = (*StubDeviceLister)(nil)
