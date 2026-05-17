package fridaengine

import (
	"errors"
	"fmt"
	"testing"
)

func TestDeviceError(t *testing.T) {
	tests := []struct {
		name    string
		op      string
		id      string
		err     error
		wantErr string
	}{
		{
			name:    "with device ID",
			op:      "enumerate",
			id:      "emulator-5554",
			err:     errors.New("connection refused"),
			wantErr: "fridaengine: device enumerate: emulator-5554: connection refused",
		},
		{
			name:    "without device ID",
			op:      "get_device",
			id:      "",
			err:     errors.New("not found"),
			wantErr: "fridaengine: device get_device: not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			de := NewDeviceError(tt.op, tt.id, tt.err)
			if de.Error() != tt.wantErr {
				t.Errorf("Error() = %q, want %q", de.Error(), tt.wantErr)
			}
			if !errors.Is(de, tt.err) {
				t.Error("errors.Is should unwrap to original error")
			}
		})
	}
}

func TestSessionError(t *testing.T) {
	tests := []struct {
		name    string
		op      string
		target  string
		err     error
		wantErr string
	}{
		{
			name:    "attach error",
			op:      "attach",
			target:  "com.example.app",
			err:     errors.New("process not found"),
			wantErr: "fridaengine: session attach (com.example.app): process not found",
		},
		{
			name:    "detach error",
			op:      "detach",
			target:  "12345",
			err:     errors.New("already detached"),
			wantErr: "fridaengine: session detach (12345): already detached",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := NewSessionError(tt.op, tt.target, tt.err)
			if se.Error() != tt.wantErr {
				t.Errorf("Error() = %q, want %q", se.Error(), tt.wantErr)
			}
			if !errors.Is(se, tt.err) {
				t.Error("errors.Is should unwrap to original error")
			}
		})
	}
}

func TestScriptError(t *testing.T) {
	tests := []struct {
		name    string
		op      string
		err     error
		wantErr string
	}{
		{
			name:    "create error",
			op:      "create",
			err:     errors.New("syntax error"),
			wantErr: "fridaengine: script create: syntax error",
		},
		{
			name:    "load error",
			op:      "load",
			err:     errors.New("runtime error"),
			wantErr: "fridaengine: script load: runtime error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := NewScriptError(tt.op, tt.err)
			if se.Error() != tt.wantErr {
				t.Errorf("Error() = %q, want %q", se.Error(), tt.wantErr)
			}
			if !errors.Is(se, tt.err) {
				t.Error("errors.Is should unwrap to original error")
			}
		})
	}
}

func TestDeviceErrorUnwrapNil(t *testing.T) {
	de := &DeviceError{Op: "test", Err: nil}
	if de.Unwrap() != nil {
		t.Error("Unwrap of nil error should return nil")
	}
}

func TestErrorWrappingChain(t *testing.T) {
	root := errors.New("root cause")
	se := NewSessionError("attach", "app", root)
	de := NewDeviceError("enumerate", "dev1", se)

	if !errors.Is(de, root) {
		t.Error("errors.Is should traverse full error chain")
	}

	var sessionErr *SessionError
	if !errors.As(de, &sessionErr) {
		t.Error("errors.As should find SessionError in chain")
	}
	if sessionErr.Target != "app" {
		t.Errorf("Target = %q, want %q", sessionErr.Target, "app")
	}
}

func TestErrorIsWithSelf(t *testing.T) {
	de := NewDeviceError("op", "id", fmt.Errorf("wrapped"))

	if !errors.Is(de, de) {
		t.Error("errors.Is with self should return true")
	}
}
