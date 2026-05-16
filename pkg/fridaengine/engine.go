package fridaengine

import (
	"context"
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
