package device

import (
	"context"
	"testing"
)

func TestStubDeviceLister(t *testing.T) {
	lister := &StubDeviceLister{}

	devices, err := lister.ListDevices(context.Background())
	if err != nil {
		t.Fatalf("ListDevices() unexpected error: %v", err)
	}

	if len(devices) != 2 {
		t.Fatalf("len(devices) = %d, want 2", len(devices))
	}

	expected := []struct {
		id   string
		name string
		typ  ConnectType
	}{
		{"emulator-5554", "Android Emulator 5554", ConnectTypeEmulator},
		{"R5CT1234ABCD", "Samsung Galaxy S21", ConnectTypeUSB},
	}

	for i, exp := range expected {
		if devices[i].ID != exp.id {
			t.Errorf("devices[%d].ID = %q, want %q", i, devices[i].ID, exp.id)
		}
		if devices[i].Name != exp.name {
			t.Errorf("devices[%d].Name = %q, want %q", i, devices[i].Name, exp.name)
		}
		if devices[i].ConnectType != exp.typ {
			t.Errorf("devices[%d].ConnectType = %q, want %q", i, devices[i].ConnectType, exp.typ)
		}
	}
}

func TestDeviceListerInterface(t *testing.T) {
	var lister DeviceLister = &StubDeviceLister{}
	devices, err := lister.ListDevices(context.Background())
	if err != nil {
		t.Fatalf("ListDevices() via interface unexpected error: %v", err)
	}
	if len(devices) == 0 {
		t.Fatal("expected at least one device")
	}
}
