//go:build integration
// +build integration

package fridaengine

import (
	"context"
	"testing"
	"time"

	"github.com/bigsmater/fridaforge/pkg/device"
)

// TestIntegrationFullLifecycle 验证完整的 Attach → CreateScript → 接收消息 → Detach 流程。
// 需要 frida-server 在 Android 设备上运行。
func TestIntegrationFullLifecycle(t *testing.T) {
	e := NewEngine(nil, nil)
	defer e.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	devices, err := e.ListDevices(ctx)
	if err != nil {
		t.Skipf("no devices available: %v", err)
	}
	if len(devices) == 0 {
		t.Skip("no devices available for integration test")
	}

	var targetDev device.Device
	for _, d := range devices {
		targetDev = d
		break
	}

	t.Logf("using device: %s (%s)", targetDev.Name, targetDev.ID)

	// 测试 Attach
	session, err := e.Attach(ctx, targetDev.ID, "System UI")
	if err != nil {
		t.Logf("could not attach to System UI, trying any process: %v", err)

		procs, err := e.EnumerateProcesses(ctx, targetDev.ID)
		if err != nil {
			t.Skipf("cannot enumerate processes: %v", err)
		}
		if len(procs) == 0 {
			t.Skip("no processes available")
		}

		session, err = e.Attach(ctx, targetDev.ID, procs[0].Name)
		if err != nil {
			t.Fatalf("Attach failed: %v", err)
		}
	}
	defer session.Detach()

	if session.State() != SessionStateCreated {
		t.Errorf("state = %v, want %v", session.State(), SessionStateCreated)
	}

	// 测试 CreateScript
	err = session.CreateScript(`console.log("FridaForge integration test");`)
	if err != nil {
		t.Fatalf("CreateScript failed: %v", err)
	}

	if session.State() != SessionStateReady {
		t.Errorf("state = %v, want %v", session.State(), SessionStateReady)
	}

	// 测试消息接收
	msgs := session.Messages()
	select {
	case msg, ok := <-msgs:
		if ok {
			t.Logf("received message: %s", msg.Payload)
		}
	case <-time.After(3 * time.Second):
		t.Log("no message received within 3s (expected for console.log)")
	}

	// 测试 Detach
	err = session.Detach()
	if err != nil {
		t.Errorf("Detach failed: %v", err)
	}
	if session.State() != SessionStateDetached {
		t.Errorf("state = %v, want %v", session.State(), SessionStateDetached)
	}

	// 幂等 Detach
	err = session.Detach()
	if err != nil {
		t.Errorf("idempotent Detach should not error: %v", err)
	}
}
