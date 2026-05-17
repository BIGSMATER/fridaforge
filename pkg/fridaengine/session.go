package fridaengine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/frida/frida-go/frida"
)

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

// HookSession 包装 frida.Session，管理 Attach 会话的完整生命周期。
// 状态机: Created → Ready → Detached
type HookSession struct {
	id       string
	deviceID string
	target   string
	state    SessionState
	session  *frida.Session
	script   *HookScript
	msgCh    chan HookMessage
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	logger   *slog.Logger
	onRemove func(id string) // SessionManager 回调：Detach 时从 sessions map 移除
}

// newHookSession 创建 HookSession（内部使用，由 Engine.Attach 调用）。
func newHookSession(id, deviceID, target string, session *frida.Session, parentCtx context.Context, logger *slog.Logger) *HookSession {
	ctx, cancel := context.WithCancel(parentCtx)
	return &HookSession{
		id:       id,
		deviceID: deviceID,
		target:   target,
		state:    SessionStateCreated,
		session:  session,
		msgCh:    make(chan HookMessage, 64),
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger,
	}
}

func (s *HookSession) setState(state SessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

// ID 返回 Session 唯一标识。
func (s *HookSession) ID() string {
	return s.id
}

// State 返回当前状态（并发安全）。
func (s *HookSession) State() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Target 返回 Attach 目标名称。
func (s *HookSession) Target() string {
	return s.target
}

// CreateScript 创建并加载 Frida JavaScript 脚本。
// 脚本加载成功后状态从 Created 变为 Ready。
func (s *HookSession) CreateScript(jsSource string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SessionStateCreated {
		return fmt.Errorf("fridaengine: cannot create script in state %s", s.state)
	}
	if s.session == nil {
		return fmt.Errorf("fridaengine: no active frida session")
	}

	var err error
	s.script, err = createScript(s.session, jsSource)
	if err != nil {
		return NewScriptError("create", err)
	}

	s.script.onMessage(s.msgCh)

	if err := s.script.Load(); err != nil {
		return NewScriptError("load", err)
	}

	s.state = SessionStateReady
	s.logger.Info("script loaded", "sessionID", s.id, "target", s.target)
	return nil
}

// Messages 返回消息通道（只读）。
// 仅在 Ready 状态有效。
func (s *HookSession) Messages() <-chan HookMessage {
	return s.msgCh
}

// Detach 断开会话，关闭消息通道，释放资源。
// 幂等操作：多次调用不报错。
func (s *HookSession) Detach() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == SessionStateDetached {
		return nil
	}

	defer s.cancel()
	defer close(s.msgCh)

	var errs []error

	if s.script != nil {
		if err := s.script.Unload(); err != nil {
			s.logger.Warn("script unload failed", "sessionID", s.id, "error", err)
		}
	}

	if s.session != nil {
		if err := s.session.Detach(); err != nil {
			errs = append(errs, NewSessionError("detach", s.target, err))
		}
	}

	s.state = SessionStateDetached
	s.logger.Info("session detached", "sessionID", s.id, "target", s.target)

	if s.onRemove != nil {
		s.onRemove(s.id)
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
