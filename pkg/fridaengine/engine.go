package fridaengine

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bigsmater/fridaforge/pkg/device"
	"github.com/frida/frida-go/frida"
)

// Engine 是 FridaForge 并发调度引擎的入口。
// 持有设备发现接口和 SessionManager，提供 Attach 和进程枚举的顶层 API。
type Engine struct {
	lister  device.DeviceLister
	manager *SessionManager
	logger  *slog.Logger
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
	mgr := frida.NewDeviceManager()
	return &Engine{
		lister:  lister,
		manager: newSessionManager(mgr, logger),
		logger:  logger,
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
// 委托给 SessionManager 执行并发控制。
func (e *Engine) Attach(ctx context.Context, deviceID string, target string) (*HookSession, error) {
	return e.manager.Attach(ctx, deviceID, target)
}

// Close 关闭引擎，清理所有活跃 Session。
// 支持 defer 模式——委托给 SessionManager.DetachAll()。
func (e *Engine) Close() error {
	return e.manager.DetachAll()
}

// ActiveSessions 返回当前活跃 Session 数量。
func (e *Engine) ActiveSessions() int {
	return e.manager.Count()
}

// findDevice 根据 deviceID 查找对应的 frida 设备接口。
func (e *Engine) findDevice(ctx context.Context, deviceID string) (frida.DeviceInt, error) {
	fridaDevices, err := e.manager.mgr.EnumerateDevices()
	if err != nil {
		return nil, NewDeviceError("enumerate", deviceID, err)
	}
	for _, d := range fridaDevices {
		if d.ID() == deviceID {
			return d, nil
		}
	}
	return nil, NewDeviceError("find", deviceID, fmt.Errorf("device not found"))
}

// EnumerateProcesses 枚举指定设备上的运行进程。
func (e *Engine) EnumerateProcesses(ctx context.Context, deviceID string) ([]ProcessInfo, error) {
	dev, err := e.findDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	type result struct {
		processes []ProcessInfo
		err       error
	}
	done := make(chan result, 1)

	go func() {
		fridaProcs, ferr := dev.EnumerateProcesses(frida.ScopeFull)
		if ferr != nil {
			done <- result{err: ferr}
			return
		}
		procs := make([]ProcessInfo, len(fridaProcs))
		for i, p := range fridaProcs {
			procs[i] = ProcessInfo{PID: p.PID(), Name: p.Name()}
		}
		done <- result{processes: procs}
	}()

	select {
	case <-ctx.Done():
		return nil, NewDeviceError("enumerate_processes", deviceID, ctx.Err())
	case r := <-done:
		if r.err != nil {
			return nil, NewDeviceError("enumerate_processes", deviceID, r.err)
		}
		e.logger.Info("processes enumerated", "deviceID", deviceID, "count", len(r.processes))
		return r.processes, nil
	}
}

// EnumerateApplications 枚举指定设备上的已安装应用。
func (e *Engine) EnumerateApplications(ctx context.Context, deviceID string) ([]ProcessInfo, error) {
	dev, err := e.findDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	type result struct {
		apps []ProcessInfo
		err  error
	}
	done := make(chan result, 1)

	go func() {
		fridaApps, ferr := dev.EnumerateApplications("", frida.ScopeFull)
		if ferr != nil {
			done <- result{err: ferr}
			return
		}
		apps := make([]ProcessInfo, len(fridaApps))
		for i, a := range fridaApps {
			apps[i] = ProcessInfo{PID: a.PID(), Name: a.Name()}
		}
		done <- result{apps: apps}
	}()

	select {
	case <-ctx.Done():
		return nil, NewDeviceError("enumerate_applications", deviceID, ctx.Err())
	case r := <-done:
		if r.err != nil {
			return nil, NewDeviceError("enumerate_applications", deviceID, r.err)
		}
		e.logger.Info("applications enumerated", "deviceID", deviceID, "count", len(r.apps))
		return r.apps, nil
	}
}
