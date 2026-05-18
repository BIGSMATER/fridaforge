package fridaengine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestNewSessionManager(t *testing.T) {
	mgr := newSessionManager(nil, nil)
	if mgr == nil {
		t.Fatal("SessionManager should not be nil")
	}
	if mgr.logger == nil {
		t.Fatal("logger should default to slog.Default()")
	}
	if mgr.sessions == nil {
		t.Fatal("sessions map should be initialized")
	}
	if mgr.softLimit != defaultSoftLimit {
		t.Errorf("softLimit = %d, want %d", mgr.softLimit, defaultSoftLimit)
	}
	if mgr.Count() != 0 {
		t.Errorf("Count = %d, want 0", mgr.Count())
	}
}

func TestSessionManagerCount(t *testing.T) {
	mgr := newSessionManager(nil, nil)
	if mgr.Count() != 0 {
		t.Errorf("initial Count = %d, want 0", mgr.Count())
	}

	mgr.mu.Lock()
	mgr.sessions["test"] = &HookSession{id: "test"}
	mgr.mu.Unlock()

	if mgr.Count() != 1 {
		t.Errorf("Count = %d, want 1", mgr.Count())
	}
}

func TestSessionManagerWithCustomLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	mgr := newSessionManager(nil, logger)
	if mgr.logger != logger {
		t.Error("should use provided logger")
	}
}

func TestSessionManagerSoftLimit(t *testing.T) {
	mgr := newSessionManager(nil, nil)

	// Fill to soft limit
	for i := 0; i < defaultSoftLimit; i++ {
		mgr.mu.Lock()
		mgr.sessions[fmt.Sprintf("session-%d", i)] = &HookSession{id: fmt.Sprintf("s-%d", i)}
		mgr.mu.Unlock()
	}

	if mgr.Count() != defaultSoftLimit {
		t.Errorf("Count = %d, want %d", mgr.Count(), defaultSoftLimit)
	}
}

func TestSessionManagerDetachAllClearsSessions(t *testing.T) {
	mgr := newSessionManager(nil, nil)

	// Manually add sessions with Detached state
	mgr.mu.Lock()
	for i := 0; i < 3; i++ {
		hs := newHookSession(fmt.Sprintf("s-%d", i), "dev", "target", nil, context.Background(), mgr.logger)
		hs.setState(SessionStateDetached)
		mgr.sessions[hs.id] = hs
	}
	mgr.mu.Unlock()

	err := mgr.DetachAll()
	if err != nil {
		t.Errorf("DetachAll should not error for detached sessions: %v", err)
	}
	if mgr.Count() != 0 {
		t.Errorf("Count after DetachAll = %d, want 0", mgr.Count())
	}
}

func TestEngineClose(t *testing.T) {
	e := NewEngine(nil, nil)
	err := e.Close()
	if err != nil {
		t.Errorf("Close on empty engine should not error: %v", err)
	}
	if e.ActiveSessions() != 0 {
		t.Errorf("ActiveSessions after Close = %d, want 0", e.ActiveSessions())
	}
}

func TestEngineActiveSessions(t *testing.T) {
	e := NewEngine(nil, nil)
	if e.ActiveSessions() != 0 {
		t.Errorf("initial ActiveSessions = %d, want 0", e.ActiveSessions())
	}
}

func TestEngineAttachIntegration(t *testing.T) {
	e := NewEngine(nil, nil)
	defer e.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	devices, err := e.ListDevices(ctx)
	if err != nil {
		t.Skipf("no devices available for integration test: %v", err)
	}
	if len(devices) == 0 {
		t.Skip("no devices available for integration test")
	}

	var session *HookSession
	for _, dev := range devices {
		session, err = e.Attach(ctx, dev.ID, "com.android.systemui")
		if err != nil {
			t.Logf("skipping %s (%s): %v", dev.ID, dev.Name, err)
			continue
		}
		break
	}
	if session == nil {
		t.Skip("no reachable device for attach integration test")
	}

	if session.State() != SessionStateCreated {
		t.Errorf("state = %v, want %v", session.State(), SessionStateCreated)
	}

	if e.ActiveSessions() != 1 {
		t.Errorf("ActiveSessions = %d, want 1", e.ActiveSessions())
	}

	err = session.Detach()
	if err != nil {
		t.Errorf("Detach failed: %v", err)
	}

	if e.ActiveSessions() != 0 {
		t.Errorf("ActiveSessions after Detach = %d, want 0", e.ActiveSessions())
	}
}

func TestSessionOnRemoveCallback(t *testing.T) {
	mgr := newSessionManager(nil, nil)

	hs := newHookSession("test-onremove", "dev-1", "target", nil, context.Background(), mgr.logger)
	hs.onRemove = func(id string) {
		mgr.mu.Lock()
		delete(mgr.sessions, id)
		mgr.mu.Unlock()
	}

	mgr.mu.Lock()
	mgr.sessions[hs.id] = hs
	mgr.mu.Unlock()

	if mgr.Count() != 1 {
		t.Fatalf("Count = %d, want 1", mgr.Count())
	}

	hs.setState(SessionStateCreated)
	err := hs.Detach()
	if err != nil {
		t.Errorf("Detach error: %v", err)
	}

	if mgr.Count() != 0 {
		t.Errorf("Count after Detach with onRemove = %d, want 0", mgr.Count())
	}
}

func TestDefaultAttachTimeout(t *testing.T) {
	mgr := newSessionManager(nil, nil)

	ctx := context.Background()
	_, ok := ctx.Deadline()
	if ok {
		t.Skip("background context has deadline (unexpected)")
	}

	_ = mgr

	ctx2 := context.Background()
	ctx3, cancel := context.WithTimeout(context.Background(), defaultAttachTimeout)
	defer cancel()

	if _, hasDeadline := ctx3.Deadline(); !hasDeadline {
		t.Error("WithTimeout should create deadline")
	}
	_ = ctx2
}
