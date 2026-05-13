package fridaengine

import (
	"testing"
	"time"
)

func TestSessionStateString(t *testing.T) {
	tests := []struct {
		name  string
		state SessionState
		want  string
	}{
		{"created", SessionStateCreated, "created"},
		{"ready", SessionStateReady, "ready"},
		{"detached", SessionStateDetached, "detached"},
		{"unknown", SessionState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSessionStateValues(t *testing.T) {
	if int(SessionStateCreated) != 0 {
		t.Error("SessionStateCreated should be 0")
	}
	if int(SessionStateReady) != 1 {
		t.Error("SessionStateReady should be 1")
	}
	if int(SessionStateDetached) != 2 {
		t.Error("SessionStateDetached should be 2")
	}
}

func TestHookMessageFields(t *testing.T) {
	ts := time.Now()
	msg := HookMessage{
		Type:      "log",
		Payload:   `{"msg":"hello"}`,
		Timestamp: ts,
	}

	if msg.Type != "log" {
		t.Errorf("Type = %q, want %q", msg.Type, "log")
	}
	if msg.Payload != `{"msg":"hello"}` {
		t.Errorf("Payload = %q", msg.Payload)
	}
	if !msg.Timestamp.Equal(ts) {
		t.Error("Timestamp mismatch")
	}
}

func TestProcessInfoFields(t *testing.T) {
	tests := []struct {
		name string
		pi   ProcessInfo
	}{
		{"system_server", ProcessInfo{PID: 1234, Name: "system_server"}},
		{"launcher", ProcessInfo{PID: 5678, Name: "com.android.launcher"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pi.PID <= 0 {
				t.Error("PID should be positive")
			}
			if tt.pi.Name == "" {
				t.Error("Name should not be empty")
			}
		})
	}
}
