package fridaengine

import "time"

// SessionState 表示 HookSession 的生命周期状态。
type SessionState int

const (
	// SessionStateCreated 表示 Attach 成功，尚未加载脚本。
	SessionStateCreated SessionState = iota
	// SessionStateReady 表示脚本已加载，可收发消息。
	SessionStateReady
	// SessionStateDetached 表示已断开连接。
	SessionStateDetached
)

// String 返回状态的可读名称。
func (s SessionState) String() string {
	switch s {
	case SessionStateCreated:
		return "created"
	case SessionStateReady:
		return "ready"
	case SessionStateDetached:
		return "detached"
	default:
		return "unknown"
	}
}

// HookMessage 表示从 Frida 脚本收到的消息。
type HookMessage struct {
	Type      string
	Payload   string
	Timestamp time.Time
}

// ProcessInfo 表示设备上的运行进程信息。
type ProcessInfo struct {
	PID  int
	Name string
}
