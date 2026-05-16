package fridaengine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/frida/frida-go/frida"
)

const defaultSoftLimit = 64

// SessionManager 管理多个并发 HookSession。
// 提供并发控制、错误聚合和统一清理。
type SessionManager struct {
	mu        sync.Mutex
	sessions  map[string]*HookSession
	logger    *slog.Logger
	softLimit int
	mgr       *frida.DeviceManager
}

// newSessionManager 创建 SessionManager（内部使用）。
func newSessionManager(mgr *frida.DeviceManager, logger *slog.Logger) *SessionManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &SessionManager{
		sessions:  make(map[string]*HookSession),
		mgr:       mgr,
		logger:    logger,
		softLimit: defaultSoftLimit,
	}
}

// Count 返回当前活跃 Session 数量（并发安全）。
func (m *SessionManager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

// Attach 连接到目标进程并返回 HookSession。
// 为每个 Session 启动独立 goroutine 管理生命周期。
// 使用独立错误聚合：单个 Session 失败不影响其他。
func (m *SessionManager) Attach(ctx context.Context, deviceID, target string) (*HookSession, error) {
	m.mu.Lock()
	if len(m.sessions) >= m.softLimit {
		m.logger.Warn("soft limit reached", "current", len(m.sessions), "limit", m.softLimit)
	}
	m.mu.Unlock()

	fridaDevices, err := m.mgr.EnumerateDevices()
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

	m.logger.Info("attaching to process via manager", "deviceID", deviceID, "target", target)

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

	id := fmt.Sprintf("%s-%s", deviceID, target)
	hookSession := newHookSession(id, deviceID, target, fridaSession, ctx, m.logger)

	m.mu.Lock()
	m.sessions[id] = hookSession
	m.mu.Unlock()

	m.logger.Info("session attached via manager", "sessionID", id, "active", m.Count())
	return hookSession, nil
}

// DetachAll 遍历所有 Session 并逐一 Detach，收集错误。
// 使用 WaitGroup 等待所有 Detach goroutine 完成。
func (m *SessionManager) DetachAll() error {
	m.mu.Lock()
	sessions := make([]*HookSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.mu.Unlock()

	m.logger.Info("detaching all sessions", "count", len(sessions))

	var errs []error
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, s := range sessions {
		wg.Add(1)
		go func(session *HookSession) {
			defer wg.Done()
			if err := session.Detach(); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(s)
	}

	wg.Wait()

	m.mu.Lock()
	m.sessions = make(map[string]*HookSession)
	m.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("fridaengine: detach errors: %v", errs)
	}
	return nil
}
