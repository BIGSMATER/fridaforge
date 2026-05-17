# 接口契约: Engine

**功能**: 002-frida-engine | **日期**: 2026-05-12

## 包: `pkg/fridaengine`

### 类型: `Engine`

```go
// NewEngine 创建引擎实例。
// lister: 设备发现实现（nil 时使用 NewFridaDeviceLister）
// logger: 日志记录器（nil 时使用 slog.Default()）
func NewEngine(lister device.DeviceLister, logger *slog.Logger) *Engine

// NewEngineWithDefaults 使用默认配置创建引擎（真实 Frida 设备发现）
func NewEngineWithDefaults() (*Engine, error)

// ListDevices 枚举已连接设备，过滤 Local 设备。
func (e *Engine) ListDevices(ctx context.Context) ([]device.Device, error)

// Attach 连接到目标进程并返回 HookSession。
// target: 进程名（包名）或 PID 字符串
func (e *Engine) Attach(ctx context.Context, deviceID string, target string) (*HookSession, error)

// EnumerateProcesses 枚举设备上的运行进程。
func (e *Engine) EnumerateProcesses(ctx context.Context, deviceID string) ([]ProcessInfo, error)

// Close 关闭引擎，清理所有活跃 Session。
func (e *Engine) Close() error
```

### 类型: `HookSession`

```go
// ID 返回 Session 唯一标识。
func (s *HookSession) ID() string

// State 返回当前状态（并发安全）。
func (s *HookSession) State() SessionState

// Target 返回 Attach 目标名称。
func (s *HookSession) Target() string

// CreateScript 创建并加载一个 Frida JavaScript 脚本。
// 脚本加载成功后状态从 Created 变为 Ready。
func (s *HookSession) CreateScript(jsSource string) error

// Messages 返回消息通道（只读）。
// 仅在 Ready 状态有效；非 Ready 状态返回 nil。
func (s *HookSession) Messages() <-chan HookMessage

// Detach 断开会话，关闭消息通道，释放资源。
// 幂等操作：多次调用不报错。
func (s *HookSession) Detach() error
```

### 类型: `HookMessage`

```go
type HookMessage struct {
    Type      string    // "log", "info", "error", "send"
    Payload   string    // JSON 字符串
    Timestamp time.Time
}
```

### 类型: `SessionState`

```go
type SessionState int

const (
    SessionStateCreated  SessionState = iota
    SessionStateReady
    SessionStateDetached
)

func (s SessionState) String() string
```

### 类型: `ProcessInfo`

```go
type ProcessInfo struct {
    PID  int
    Name string
}
```

`EnumerateApplications()` 返回相同类型（应用标识符作为 Name），避免为 M2 创建冗余类型。

### 类型: `FridaDeviceLister`

```go
// NewFridaDeviceLister 创建真实 Frida 设备发现器。
func NewFridaDeviceLister(logger *slog.Logger) *FridaDeviceLister

// ListDevices 实现 device.DeviceLister 接口。
func (l *FridaDeviceLister) ListDevices(ctx context.Context) ([]device.Device, error)

// 编译时检查
var _ device.DeviceLister = (*FridaDeviceLister)(nil)
```

### 错误构造函数

```go
func NewDeviceError(op, id string, err error) *DeviceError
func NewSessionError(op, target string, err error) *SessionError
func NewScriptError(op string, err error) *ScriptError
```

## 调用示例

```go
func main() {
    engine, _ := fridaengine.NewEngineWithDefaults()
    defer engine.Close()

    ctx := context.Background()
    devices, _ := engine.ListDevices(ctx)

    session, _ := engine.Attach(ctx, devices[0].ID, "com.example.app")
    defer session.Detach()

    session.CreateScript(`console.log("Hello FridaForge");`)

    for msg := range session.Messages() {
        fmt.Printf("[%s] %s\n", msg.Type, msg.Payload)
    }
}
```

## 并发安全契约

| 操作 | 并发安全 | 说明 |
|------|---------|------|
| Engine.ListDevices | Yes | 读操作，无共享状态修改 |
| Engine.Attach | Yes | SessionManager.Mutex 保护 sessions map |
| Engine.Close | Yes | 遍历+清理，Mutex 保护 |
| HookSession.State() | Yes | RWMutex.RLock |
| HookSession.CreateScript | No | 单 goroutine 调用，内部状态变更 |
| HookSession.Detach | Yes | 幂等，状态变更用 Lock 保护 |
| HookSession.Messages() | Yes | 只读 channel 访问 |
