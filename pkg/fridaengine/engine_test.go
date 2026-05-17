package fridaengine

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestNewEngine(t *testing.T) {
	t.Run("with nil logger and nil lister", func(t *testing.T) {
		e := NewEngine(nil, nil)
		if e == nil {
			t.Fatal("Engine should not be nil")
		}
		if e.logger == nil {
			t.Fatal("logger should default to slog.Default()")
		}
		if e.lister == nil {
			t.Fatal("lister should default to FridaDeviceLister")
		}
		if e.manager == nil {
			t.Fatal("SessionManager should not be nil")
		}
		if e.ActiveSessions() != 0 {
			t.Fatal("ActiveSessions should be 0 on creation")
		}
	})

	t.Run("with explicit logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		e := NewEngine(nil, logger)
		if e.logger != logger {
			t.Error("should use provided logger")
		}
	})
}

func TestNewEngineWithDefaults(t *testing.T) {
	e, err := NewEngineWithDefaults()
	if err != nil {
		t.Fatalf("NewEngineWithDefaults should not error: %v", err)
	}
	if e == nil {
		t.Fatal("Engine should not be nil")
	}
}

func TestEngineListDevicesDelegation(t *testing.T) {
	e := NewEngine(nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	devices, err := e.ListDevices(ctx)
	if err != nil {
		t.Logf("ListDevices returned error (expected if no devices): %v", err)
	}
	_ = devices
}

func TestHookSessionID(t *testing.T) {
	hs := newHookSession("test-001", "dev-1", "com.example", nil, context.Background(), slog.Default())
	if hs.ID() != "test-001" {
		t.Errorf("ID = %q, want %q", hs.ID(), "test-001")
	}
}

func TestHookSessionTarget(t *testing.T) {
	hs := newHookSession("test-001", "dev-1", "com.example", nil, context.Background(), slog.Default())
	if hs.Target() != "com.example" {
		t.Errorf("Target = %q, want %q", hs.Target(), "com.example")
	}
}

func TestHookSessionInitialState(t *testing.T) {
	hs := newHookSession("test-001", "dev-1", "com.example", nil, context.Background(), slog.Default())
	if hs.State() != SessionStateCreated {
		t.Errorf("initial state = %v, want %v", hs.State(), SessionStateCreated)
	}
}

func TestHookSessionMessagesChannel(t *testing.T) {
	hs := newHookSession("test-001", "dev-1", "com.example", nil, context.Background(), slog.Default())

	msgs := hs.Messages()
	if msgs == nil {
		t.Fatal("Messages() should return a non-nil channel")
	}

	select {
	case _, ok := <-msgs:
		if ok {
			t.Error("newly created session should not have messages in channel")
		}
	default:
	}
}

func TestHookSessionCreateScriptInvalidState(t *testing.T) {
	hs := newHookSession("test-001", "dev-1", "com.example", nil, context.Background(), slog.Default())

	hs.setState(SessionStateReady)
	err := hs.CreateScript("console.log('test');")
	if err == nil {
		t.Error("CreateScript should fail in Ready state")
	}
	if err.Error() != "fridaengine: cannot create script in state ready" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHookSessionDetachIdempotent(t *testing.T) {
	hs := newHookSession("test-001", "dev-1", "com.example", nil, context.Background(), slog.Default())

	hs.setState(SessionStateDetached)

	err := hs.Detach()
	if err != nil {
		t.Errorf("Detach on already-detached session should not error: %v", err)
	}
}

func TestHookSessionStateConcurrent(t *testing.T) {
	hs := newHookSession("test-001", "dev-1", "com.example", nil, context.Background(), slog.Default())

	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			hs.State()
		}
		done <- struct{}{}
	}()

	for i := 0; i < 100; i++ {
		hs.setState(SessionStateReady)
		hs.setState(SessionStateCreated)
	}

	<-done
}

func TestEngineAttachDeviceNotFound(t *testing.T) {
	e := NewEngine(nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := e.Attach(ctx, "nonexistent-device-id", "com.example")
	if err == nil {
		t.Error("Attach to nonexistent device should return error")
	}
	_, ok := err.(*DeviceError)
	if !ok {
		t.Errorf("expected DeviceError, got %T: %v", err, err)
	}
}

func TestEngineAttachContextTimeout(t *testing.T) {
	e := NewEngine(nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(1 * time.Millisecond)

	_, err := e.Attach(ctx, "some-device", "com.example")
	if err == nil {
		t.Error("Attach with expired context should return error")
	}
}

func TestEngineEnumerateProcesses(t *testing.T) {
	e := NewEngine(nil, nil)
	defer e.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	devices, err := e.ListDevices(ctx)
	if err != nil {
		t.Skipf("no devices available: %v", err)
	}
	if len(devices) == 0 {
		t.Skip("no devices available")
	}

	for _, dev := range devices {
		procs, err := e.EnumerateProcesses(ctx, dev.ID)
		if err != nil {
			t.Logf("skipping %s (%s): %v", dev.ID, dev.Name, err)
			continue
		}
		if len(procs) == 0 {
			t.Error("expected at least one process")
		}
		t.Logf("found %d processes on %s (%s)", len(procs), dev.Name, dev.ID)
		return
	}
	t.Skip("no reachable device found for process enumeration")
}

func TestEngineEnumerateProcessesDeviceNotFound(t *testing.T) {
	e := NewEngine(nil, nil)
	defer e.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := e.EnumerateProcesses(ctx, "nonexistent-device-id")
	if err == nil {
		t.Error("expected error for nonexistent device")
	}
}

func TestEngineEnumerateApplications(t *testing.T) {
	e := NewEngine(nil, nil)
	defer e.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	devices, err := e.ListDevices(ctx)
	if err != nil {
		t.Skipf("no devices available: %v", err)
	}
	if len(devices) == 0 {
		t.Skip("no devices available")
	}

	for _, dev := range devices {
		apps, err := e.EnumerateApplications(ctx, dev.ID)
		if err != nil {
			t.Logf("skipping %s (%s): %v", dev.ID, dev.Name, err)
			continue
		}
		if len(apps) == 0 {
			t.Error("expected at least one application")
		}
		t.Logf("found %d applications on %s (%s)", len(apps), dev.Name, dev.ID)
		return
	}
	t.Skip("no reachable device found for application enumeration")
}

func TestHookSessionDetachWithStateTransition(t *testing.T) {
	hs := newHookSession("test-101", "dev-1", "com.example", nil, context.Background(), slog.Default())

	hs.setState(SessionStateCreated)
	if hs.State() != SessionStateCreated {
		t.Error("expected created state")
	}

	hs.setState(SessionStateDetached)
	if hs.State() != SessionStateDetached {
		t.Error("expected detached state")
	}

	err := hs.Detach()
	if err != nil {
		t.Errorf("Detach on detached session should be idempotent: %v", err)
	}
}

func TestHookSessionCreateScriptDetachedState(t *testing.T) {
	hs := newHookSession("test-102", "dev-1", "com.example", nil, context.Background(), slog.Default())
	hs.setState(SessionStateDetached)

	err := hs.CreateScript("console.log('test');")
	if err == nil {
		t.Error("CreateScript should fail in Detached state")
	}
}

func TestSessionManagerDetachAllEmpty(t *testing.T) {
	mgr := newSessionManager(nil, nil)

	err := mgr.DetachAll()
	if err != nil {
		t.Errorf("DetachAll on empty manager should not error: %v", err)
	}
	if mgr.Count() != 0 {
		t.Errorf("Count should still be 0, got %d", mgr.Count())
	}
}
