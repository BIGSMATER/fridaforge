package device

import (
	"testing"
)

func TestDevice(t *testing.T) {
	tests := []struct {
		name     string
		device   Device
		wantID   string
		wantName string
		wantType ConnectType
	}{
		{
			name:     "emulator device",
			device:   Device{ID: "emulator-5554", Name: "Android Emulator", ConnectType: ConnectTypeEmulator},
			wantID:   "emulator-5554",
			wantName: "Android Emulator",
			wantType: ConnectTypeEmulator,
		},
		{
			name:     "usb device",
			device:   Device{ID: "R5CT1234ABCD", Name: "Samsung Galaxy S21", ConnectType: ConnectTypeUSB},
			wantID:   "R5CT1234ABCD",
			wantName: "Samsung Galaxy S21",
			wantType: ConnectTypeUSB,
		},
		{
			name:     "network device",
			device:   Device{ID: "192.168.1.100:27042", Name: "Remote Device", ConnectType: ConnectTypeNetwork},
			wantID:   "192.168.1.100:27042",
			wantName: "Remote Device",
			wantType: ConnectTypeNetwork,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.device.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", tt.device.ID, tt.wantID)
			}
			if tt.device.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", tt.device.Name, tt.wantName)
			}
			if tt.device.ConnectType != tt.wantType {
				t.Errorf("ConnectType = %q, want %q", tt.device.ConnectType, tt.wantType)
			}
		})
	}
}
