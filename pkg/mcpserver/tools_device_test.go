package mcpserver

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/bigsmater/fridaforge/pkg/device"
)

// ---------- T013: device_list handler 测试 ----------

func TestDeviceListHandler_Success(t *testing.T) {
	lister := &mockDeviceLister{
		devices: []device.Device{
			{ID: "d1", Name: "Device 1", ConnectType: "usb"},
			{ID: "d2", Name: "Device 2", ConnectType: "emulator"},
		},
	}
	srv := &Server{deviceLister: lister, logger: slog.New(slog.DiscardHandler)}

	_, output, err := srv.deviceListHandler(context.Background(), nil, struct{}{})
	if err != nil {
		t.Fatalf("deviceListHandler() 错误: %v", err)
	}
	if len(output.Devices) != 2 {
		t.Fatalf("期望 2 台设备，实际 %d", len(output.Devices))
	}
	if output.Devices[0].ID != "d1" {
		t.Errorf("设备 ID = %q, 期望 %q", output.Devices[0].ID, "d1")
	}
}

func TestDeviceListHandler_Empty(t *testing.T) {
	lister := &mockDeviceLister{}
	srv := &Server{deviceLister: lister, logger: slog.New(slog.DiscardHandler)}

	_, output, err := srv.deviceListHandler(context.Background(), nil, struct{}{})
	if err != nil {
		t.Fatalf("空设备列表不应报错: %v", err)
	}
	if len(output.Devices) != 0 {
		t.Errorf("期望空列表，实际 %d", len(output.Devices))
	}
}

func TestDeviceListHandler_ListerError(t *testing.T) {
	srv := &Server{
		deviceLister: deviceListerFunc(func(ctx context.Context) ([]device.Device, error) {
			return nil, errors.New("connection refused")
		}),
		logger: slog.New(slog.DiscardHandler),
	}

	_, _, err := srv.deviceListHandler(context.Background(), nil, struct{}{})
	if err == nil {
		t.Fatal("期望 lister 错误被传播")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("错误应包含底层错误: %v", err)
	}
}

// deviceListerFunc 将函数转换为 DeviceLister 实现（用于注入错误）
type deviceListerFunc func(ctx context.Context) ([]device.Device, error)

func (f deviceListerFunc) ListDevices(ctx context.Context) ([]device.Device, error) {
	return f(ctx)
}

// ---------- T014: process_list handler 测试 ----------

func TestProcessListHandler_Success(t *testing.T) {
	lister := &StubProcessLister{
		processesByDevice: map[string][]ProcessListItem{
			"dev1": {{PID: 100, Name: "app1"}, {PID: 200, Name: "app2"}},
		},
	}
	srv := &Server{processLister: lister, logger: slog.New(slog.DiscardHandler)}

	_, output, err := srv.processListHandler(context.Background(), nil, ProcessListInput{DeviceID: "dev1"})
	if err != nil {
		t.Fatalf("processListHandler() 错误: %v", err)
	}
	if len(output.Processes) != 2 {
		t.Fatalf("期望 2 个进程，实际 %d", len(output.Processes))
	}
}

func TestProcessListHandler_NoSuchDevice(t *testing.T) {
	lister := &StubProcessLister{processesByDevice: map[string][]ProcessListItem{}}
	srv := &Server{processLister: lister, logger: slog.New(slog.DiscardHandler)}

	_, output, err := srv.processListHandler(context.Background(), nil, ProcessListInput{DeviceID: "nonexistent"})
	if err != nil {
		t.Fatalf("不存在的设备不应报错: %v", err)
	}
	if len(output.Processes) != 0 {
		t.Errorf("期望空列表，实际 %d", len(output.Processes))
	}
}

func TestProcessListHandler_ListerError(t *testing.T) {
	srv := &Server{
		processLister: processListerFunc(func(ctx context.Context, deviceID string) ([]ProcessListItem, error) {
			return nil, errors.New("internal error")
		}),
		logger: slog.New(slog.DiscardHandler),
	}

	_, _, err := srv.processListHandler(context.Background(), nil, ProcessListInput{DeviceID: "dev1"})
	if err == nil {
		t.Fatal("期望 lister 错误被传播")
	}
	if !strings.Contains(err.Error(), "internal error") {
		t.Errorf("错误应包含底层错误: %v", err)
	}
}

// processListerFunc 将函数转换为 ProcessLister 实现（用于注入错误）
type processListerFunc func(ctx context.Context, deviceID string) ([]ProcessListItem, error)

func (f processListerFunc) ListProcesses(ctx context.Context, deviceID string) ([]ProcessListItem, error) {
	return f(ctx, deviceID)
}
