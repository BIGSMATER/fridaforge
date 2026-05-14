package fridaengine

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bigsmater/fridaforge/pkg/device"
	"github.com/frida/frida-go/frida"
)

// Engine 是 FridaForge 并发调度引擎的入口。
// 持有设备发现接口和 frida.DeviceManager，提供 Attach 和进程枚举的顶层 API。
type Engine struct {
	lister device.DeviceLister
	mgr    *frida.DeviceManager
	logger *slog.Logger
}

// NewEngine 创建引擎实例。
// lister: 设备发现实现（nil 时使用 NewFridaDeviceLister）。
// logger: 日志记录器（nil 时使用 slog.Default()）。
func NewEngine(lister device.DeviceLister, logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	if lister == nil {
		lister = NewFridaDeviceLister(logger)
	}
	return &Engine{
		lister: lister,
		mgr:    frida.NewDeviceManager(),
		logger: logger,
	}
}

// NewEngineWithDefaults 使用默认配置创建引擎（真实 Frida 设备发现）。
func NewEngineWithDefaults() (*Engine, error) {
	return NewEngine(nil, nil), nil
}

// ListDevices 枚举已连接设备，过滤 Local 设备。
func (e *Engine) ListDevices(ctx context.Context) ([]device.Device, error) {
	return e.lister.ListDevices(ctx)
}

// Attach 连接到目标进程并返回 HookSession。
// deviceID: 目标设备 ID（来自 ListDevices 的结果）。
// target: 进程名（包名）或 PID 字符串。
// 默认 30 秒超时——通过父 ctx 控制。
func (e *Engine) Attach(ctx context.Context, deviceID string, target string) (*HookSession, error) {
	// 1. 查找匹配的 frida 设备
	fridaDevices, err := e.mgr.EnumerateDevices()
	if err != nil {
		return nil, NewDeviceError("enumerate", deviceID, err)
	}

	var targetDevice frida.DeviceInt
	for _, d := range fridaDevices {
		if d.ID() == deviceID {
			targetDevice = d
			break
		}
	}
	if targetDevice == nil {
		return nil, NewDeviceError("find", deviceID, fmt.Errorf("device not found"))
	}

	e.logger.Info("attaching to process", "deviceID", deviceID, "target", target)

	// 2. Attach（通过 goroutine+select 接入 context 超时）
	type attachResult struct {
		session *frida.Session
		err     error
	}
	done := make(chan attachResult, 1)

	go func() {
		session, err := targetDevice.Attach(target, nil)
		done <- attachResult{session, err}
	}()

	var fridaSession *frida.Session
	select {
	case <-ctx.Done():
		return nil, NewSessionError("attach", target, ctx.Err())
	case result := <-done:
		if result.err != nil {
			return nil, NewSessionError("attach", target, result.err)
		}
		fridaSession = result.session
	}

	// 3. 创建 HookSession 包装
	id := fmt.Sprintf("%s-%s-%d", deviceID, target, time.Now().UnixNano())
	hookSession := newHookSession(id, deviceID, target, fridaSession, ctx, e.logger)

	e.logger.Info("attached successfully", "sessionID", id, "target", target)
	return hookSession, nil
}
