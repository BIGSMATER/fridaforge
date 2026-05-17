# 数据模型: Frida 并发调度引擎

**功能**: 002-frida-engine | **日期**: 2026-05-12

## 核心实体

### 1. HookSession (会话)

| 属性 | 类型 | 说明 |
|------|------|------|
| id | string | 唯一标识 (UUID v4) |
| deviceID | string | 关联设备 ID |
| target | string | Attach 目标 (包名或 PID) |
| state | SessionState | 当前状态 (枚举) |
| session | *frida.Session | 底层 frida-go Session (非导出) |
| script | *HookScript | 加载的脚本 (Ready 状态后有值) |
| msgCh | chan HookMessage | 消息通道 (缓冲 64) |
| ctx | context.Context | Session 生命周期 context |
| cancel | context.CancelFunc | 取消函数 |
| mu | sync.RWMutex | 状态读写锁 |
| logger | *slog.Logger | 结构化日志 |

**状态机**:
```
  Attach()          Load()           Detach()
  ────────► Created ──────► Ready ────────► Detached
                 ◄────────────────────────
                  (任何状态可跳转 Detached)
```

**状态枚举**:
```go
type SessionState int
const (
    SessionStateCreated  SessionState = iota // Attach 成功，尚未加载脚本
    SessionStateReady                        // 脚本已加载，可收发消息
    SessionStateDetached                     // 已断开
)
```

### 2. HookScript (脚本)

| 属性 | 类型 | 说明 |
|------|------|------|
| script | *frida.Script | 底层 frida-go Script (非导出) |
| sessionID | string | 关联 Session ID |

**生命周期**: 由 HookSession 管理，不独立创建

### 3. HookMessage (消息)

| 属性 | 类型 | 说明 |
|------|------|------|
| Type | string | 消息类型: "log", "info", "error", "send" |
| Payload | string | 消息载荷 (JSON 字符串) |
| Timestamp | time.Time | 接收时间戳 |

### 4. Engine (引擎)

| 属性 | 类型 | 说明 |
|------|------|------|
| lister | device.DeviceLister | 设备发现接口 (依赖注入) |
| manager | *SessionManager | 会话管理器 |
| logger | *slog.Logger | 日志注入 |

### 5. SessionManager (会话管理器)

| 属性 | 类型 | 说明 |
|------|------|------|
| sessions | map[string]*HookSession | 活跃会话映射 |
| mu | sync.Mutex | 会话映射锁 |
| wg | sync.WaitGroup | goroutine 计数 |
| logger | *slog.Logger | 日志 |
| softLimit | int | 软上限 (64) |

### 6. FridaDeviceLister (设备发现)

| 属性 | 类型 | 说明 |
|------|------|------|
| mgr | *frida.DeviceManager | 底层 frida DeviceManager |
| logger | *slog.Logger | 日志 |

实现 `device.DeviceLister` 接口:
```go
func (l *FridaDeviceLister) ListDevices(ctx context.Context) ([]device.Device, error)
```

## 错误类型

```go
// DeviceError — 设备层错误
type DeviceError struct {
    Op   string // 操作: "enumerate", "get_device"
    ID   string // 设备 ID (可选)
    Err  error  // 底层错误
}

// SessionError — 会话层错误
type SessionError struct {
    Op     string // 操作: "attach", "detach"
    Target string // 目标进程
    Err    error  // 底层错误
}

// ScriptError — 脚本层错误
type ScriptError struct {
    Op   string // 操作: "create", "load"
    Err  error  // 底层错误
}
```

三者均实现 `error` 接口和 `Unwrap() error` 方法，支持 `errors.Is()` / `errors.As()`。

## 关系图

```
Engine ──owns──► DeviceLister (接口)
  │                  ▲
  │                  │ implements
  │           FridaDeviceLister
  │
  └──owns──► SessionManager ──manages──► map[string]*HookSession
                                               │
                                               │ owns
                                          HookScript
                                               │
                                               │ produces
                                          chan HookMessage
```
