package fridaengine

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bigsmater/fridaforge/pkg/device"
	"github.com/frida/frida-go/frida"
)

// FridaDeviceLister 通过 frida-go 实现 device.DeviceLister 接口。
// 过滤 Local 设备，仅返回 Remote (USB) 和 Network 设备。
type FridaDeviceLister struct {
	mgr    *frida.DeviceManager
	logger *slog.Logger
}

// NewFridaDeviceLister 创建 FridaDeviceLister。
// 传入 nil logger 则使用 slog.Default()。
func NewFridaDeviceLister(logger *slog.Logger) *FridaDeviceLister {
	if logger == nil {
		logger = slog.Default()
	}
	return &FridaDeviceLister{
		mgr:    frida.NewDeviceManager(),
		logger: logger,
	}
}

// ListDevices 实现 device.DeviceLister 接口。
// 通过 frida.DeviceManager 枚举已连接设备，过滤 Local 设备。
func (l *FridaDeviceLister) ListDevices(ctx context.Context) ([]device.Device, error) {
	done := make(chan struct{})
	var fridaDevices []frida.DeviceInt
	var enumerateErr error

	go func() {
		defer close(done)
		fridaDevices, enumerateErr = l.mgr.EnumerateDevices()
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("fridaengine: device enumerate: %w", ctx.Err())
	case <-done:
	}

	if enumerateErr != nil {
		l.logger.Error("enumerate devices failed", "error", enumerateErr)
		return nil, NewDeviceError("enumerate", "", enumerateErr)
	}

	var result []device.Device
	for _, d := range fridaDevices {
		dt := d.DeviceType()
		connectType := fridaDeviceTypeToConnectType(dt)
		if connectType == "" {
			l.logger.Debug("filtering local device", "id", d.ID(), "name", d.Name())
			continue
		}

		result = append(result, device.Device{
			ID:          d.ID(),
			Name:        d.Name(),
			ConnectType: connectType,
		})
	}

	l.logger.Info("devices enumerated", "total", len(fridaDevices), "filtered", len(result))
	return result, nil
}

// fridaDeviceTypeToConnectType 将 frida-go DeviceType 转换为 M1 ConnectType。
// 返回空字符串表示应过滤该设备 (Local 设备)。
// frida-go 只有三种 DeviceType: Local(0), Remote(1—TCP连接), Usb(2—USB连接)。
func fridaDeviceTypeToConnectType(dt frida.DeviceType) device.ConnectType {
	switch dt {
	case frida.DeviceTypeRemote:
		return device.ConnectTypeNetwork
	case frida.DeviceTypeUsb:
		return device.ConnectTypeUSB
	default:
		return ""
	}
}

var _ device.DeviceLister = (*FridaDeviceLister)(nil)
