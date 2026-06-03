package mcpserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bigsmater/fridaforge/pkg/device"
)

func TestLoadMockStore_DefaultConfig(t *testing.T) {
	// 确保没有 YAML 配置文件干扰（临时修改 HOME）
	t.Setenv("HOME", t.TempDir())

	store, err := LoadMockStore()
	if err != nil {
		t.Fatalf("LoadMockStore() 失败: %v", err)
	}
	if store == nil {
		t.Fatal("LoadMockStore() 返回 nil")
	}
	if store.DeviceLister == nil {
		t.Fatal("DeviceLister 为 nil")
	}
	if store.ProcessLister == nil {
		t.Fatal("ProcessLister 为 nil")
	}
}

func TestLoadMockStore_ParseYAML(t *testing.T) {
	dir := t.TempDir()
	yamlContent := []byte(`
devices:
  - id: "usb-001"
    name: "Pixel 6"
    connect_type: "usb"
processes:
  - device_id: "usb-001"
    processes:
      - pid: 100
        name: "com.test.app"
      - pid: 200
        name: "system_server"
`)
	homeDir := filepath.Join(dir, "home")
	os.MkdirAll(filepath.Join(homeDir, ".fridaforge"), 0755)
	os.WriteFile(filepath.Join(homeDir, ".fridaforge", "mock_devices.yaml"), yamlContent, 0644)
	t.Setenv("HOME", homeDir)

	store, err := LoadMockStore()
	if err != nil {
		t.Fatalf("LoadMockStore() 解析 YAML 失败: %v", err)
	}

	devices, err := store.DeviceLister.ListDevices(t.Context())
	if err != nil {
		t.Fatalf("ListDevices() 失败: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("期望 1 台设备，实际 %d", len(devices))
	}
	if devices[0].ID != "usb-001" {
		t.Errorf("设备 ID = %q，期望 %q", devices[0].ID, "usb-001")
	}
	if devices[0].Name != "Pixel 6" {
		t.Errorf("设备名 = %q，期望 %q", devices[0].Name, "Pixel 6")
	}

	procs, err := store.ProcessLister.ListProcesses(t.Context(), "usb-001")
	if err != nil {
		t.Fatalf("ListProcesses() 失败: %v", err)
	}
	if len(procs) != 2 {
		t.Fatalf("期望 2 个进程，实际 %d", len(procs))
	}

	// 不存在的设备应返回空列表
	procs, err = store.ProcessLister.ListProcesses(t.Context(), "nonexistent")
	if err != nil {
		t.Fatalf("ListProcesses() nonexistent 失败: %v", err)
	}
	if len(procs) != 0 {
		t.Errorf("不存在的设备应返回空列表，实际 %d", len(procs))
	}
}

func TestLoadMockStore_BrokenYAML(t *testing.T) {
	dir := t.TempDir()
	homeDir := filepath.Join(dir, "home")
	os.MkdirAll(filepath.Join(homeDir, ".fridaforge"), 0755)
	os.WriteFile(filepath.Join(homeDir, ".fridaforge", "mock_devices.yaml"), []byte("invalid: [broken"), 0644)
	t.Setenv("HOME", homeDir)

	_, err := LoadMockStore()
	if err == nil {
		t.Fatal("损坏的 YAML 应返回错误")
	}
}

func TestStubProcessLister(t *testing.T) {
	lister := &StubProcessLister{
		processesByDevice: map[string][]ProcessListItem{
			"dev1": {{PID: 100, Name: "app1"}, {PID: 200, Name: "app2"}},
		},
	}

	tests := []struct {
		name     string
		deviceID string
		wantLen  int
	}{
		{"存在的设备", "dev1", 2},
		{"不存在的设备", "dev2", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			procs, err := lister.ListProcesses(t.Context(), tt.deviceID)
			if err != nil {
				t.Fatalf("ListProcesses(%q) 错误: %v", tt.deviceID, err)
			}
			if len(procs) != tt.wantLen {
				t.Errorf("期望 %d 个进程，实际 %d", tt.wantLen, len(procs))
			}
		})
	}
}

func TestMockDeviceLister(t *testing.T) {
	lister := &mockDeviceLister{
		devices: []device.Device{
			{ID: "d1", Name: "Device 1", ConnectType: "usb"},
			{ID: "d2", Name: "Device 2", ConnectType: "emulator"},
		},
	}

	devices, err := lister.ListDevices(t.Context())
	if err != nil {
		t.Fatalf("ListDevices() 失败: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("期望 2 台设备，实际 %d", len(devices))
	}
	if devices[0].ID != "d1" || devices[1].ID != "d2" {
		t.Errorf("设备 ID 不匹配")
	}
}
