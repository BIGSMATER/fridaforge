package fridaengine

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/bigsmater/fridaforge/pkg/device"
	"github.com/frida/frida-go/frida"
)

func TestNewFridaDeviceLister(t *testing.T) {
	t.Run("with nil logger", func(t *testing.T) {
		l := NewFridaDeviceLister(nil)
		if l == nil {
			t.Fatal("NewFridaDeviceLister should not return nil")
		}
		if l.logger == nil {
			t.Fatal("logger should not be nil (defaulted to slog.Default)")
		}
		if l.mgr == nil {
			t.Fatal("DeviceManager should not be nil")
		}
	})

	t.Run("with explicit logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		l := NewFridaDeviceLister(logger)
		if l.logger != logger {
			t.Error("should use the provided logger")
		}
	})
}

func TestFridaDeviceTypeToConnectType(t *testing.T) {
	tests := []struct {
		name       string
		deviceType frida.DeviceType
		want       device.ConnectType
		wantEmpty  bool
	}{
		{
			name:       "local device filtered",
			deviceType: frida.DeviceTypeLocal,
			want:       "",
			wantEmpty:  true,
		},
		{
			name:       "remote device maps to network",
			deviceType: frida.DeviceTypeRemote,
			want:       device.ConnectTypeNetwork,
		},
		{
			name:       "usb device maps to usb",
			deviceType: frida.DeviceTypeUsb,
			want:       device.ConnectTypeUSB,
		},
		{
			name:       "invalid device type",
			deviceType: frida.DeviceType(99),
			want:       "",
			wantEmpty:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fridaDeviceTypeToConnectType(tt.deviceType)
			if tt.wantEmpty && got != "" {
				t.Errorf("expected empty string, got %q", got)
			}
			if !tt.wantEmpty && got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestFridaDeviceListerInterface(t *testing.T) {
	var lister device.DeviceLister = NewFridaDeviceLister(nil)
	if lister == nil {
		t.Fatal("FridaDeviceLister should satisfy DeviceLister interface")
	}
}

func TestFridaDeviceListerListDevicesContextCancel(t *testing.T) {
	lister := NewFridaDeviceLister(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	devices, err := lister.ListDevices(ctx)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
	if devices != nil {
		t.Error("expected nil devices on error")
	}
}
