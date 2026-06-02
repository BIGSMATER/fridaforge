package mcpserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/bigsmater/fridaforge/pkg/device"
)

// MockDeviceEntry 表示 YAML 配置文件中的单个模拟设备
type MockDeviceEntry struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	ConnectType string `yaml:"connect_type"`
}

// MockProcessEntry 表示 YAML 配置文件中的单个模拟进程
type MockProcessEntry struct {
	DeviceID    string             `yaml:"device_id"`
	Processes   []ProcessListItem  `yaml:"processes"`
}

// MockStoreConfig 是模拟数据 YAML 文件的顶层结构
type MockStoreConfig struct {
	Devices   []MockDeviceEntry  `yaml:"devices"`
	Processes []MockProcessEntry `yaml:"processes"`
}

// MockStore 持有模拟模式的桩数据
type MockStore struct {
	DeviceLister  device.DeviceLister
	ProcessLister ProcessLister
}

// mockDeviceLister 实现 device.DeviceLister，返回可配置的设备列表
type mockDeviceLister struct {
	devices []device.Device
}

func (m *mockDeviceLister) ListDevices(ctx context.Context) ([]device.Device, error) {
	return m.devices, nil
}

// defaultMockConfig 返回内嵌的默认模拟数据（文件不存在时使用）
func defaultMockConfig() MockStoreConfig {
	return MockStoreConfig{
		Devices: []MockDeviceEntry{
			{ID: "emulator-5554", Name: "Android Emulator (AVD 1)", ConnectType: "emulator"},
			{ID: "emulator-5556", Name: "Android Emulator (AVD 2)", ConnectType: "emulator"},
		},
		Processes: []MockProcessEntry{
			{
				DeviceID: "emulator-5554",
				Processes: []ProcessListItem{
					{PID: 1234, Name: "com.android.systemui"},
					{PID: 5678, Name: "com.example.app"},
					{PID: 9012, Name: "com.android.settings"},
				},
			},
			{
				DeviceID: "emulator-5556",
				Processes: []ProcessListItem{
					{PID: 1111, Name: "com.android.phone"},
					{PID: 2222, Name: "com.google.android.gms"},
				},
			},
		},
	}
}

// mockConfigPath returns the path to the YAML mock data file.
func mockConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".fridaforge", "mock_devices.yaml")
}

// LoadMockStore 创建桩数据。优先加载 YAML 配置文件，
// 如文件不存在则使用内嵌默认值。
func LoadMockStore() (*MockStore, error) {
	cfg := defaultMockConfig()

	data, err := os.ReadFile(mockConfigPath())
	if err == nil {
		if unmarshalErr := yaml.Unmarshal(data, &cfg); unmarshalErr != nil {
			return nil, fmt.Errorf("解析 mock 配置文件失败: %w", unmarshalErr)
		}
	}

	devices := make([]device.Device, 0, len(cfg.Devices))
	for _, d := range cfg.Devices {
		devices = append(devices, device.Device{
			ID:          d.ID,
			Name:        d.Name,
			ConnectType: device.ConnectType(d.ConnectType),
		})
	}

	procMap := make(map[string][]ProcessListItem)
	for _, pe := range cfg.Processes {
		procs := make([]ProcessListItem, len(pe.Processes))
		copy(procs, pe.Processes)
		procMap[pe.DeviceID] = procs
	}

	return &MockStore{
		DeviceLister:  &mockDeviceLister{devices: devices},
		ProcessLister: &StubProcessLister{processesByDevice: procMap},
	}, nil
}
