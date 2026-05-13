package fridaengine

import "fmt"

// DeviceError 表示设备层错误（设备枚举、连接失败）。
type DeviceError struct {
	Op  string
	ID  string
	Err error
}

func (e *DeviceError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("fridaengine: device %s: %s: %v", e.Op, e.ID, e.Err)
	}
	return fmt.Sprintf("fridaengine: device %s: %v", e.Op, e.Err)
}

// Unwrap 返回底层错误，支持 errors.Is / errors.As。
func (e *DeviceError) Unwrap() error {
	return e.Err
}

// NewDeviceError 创建一个 DeviceError。
func NewDeviceError(op, id string, err error) *DeviceError {
	return &DeviceError{Op: op, ID: id, Err: err}
}

// SessionError 表示会话层错误（Attach、Detach 失败）。
type SessionError struct {
	Op     string
	Target string
	Err    error
}

func (e *SessionError) Error() string {
	return fmt.Sprintf("fridaengine: session %s (%s): %v", e.Op, e.Target, e.Err)
}

// Unwrap 返回底层错误。
func (e *SessionError) Unwrap() error {
	return e.Err
}

// NewSessionError 创建一个 SessionError。
func NewSessionError(op, target string, err error) *SessionError {
	return &SessionError{Op: op, Target: target, Err: err}
}

// ScriptError 表示脚本层错误（脚本创建、加载失败）。
type ScriptError struct {
	Op  string
	Err error
}

func (e *ScriptError) Error() string {
	return fmt.Sprintf("fridaengine: script %s: %v", e.Op, e.Err)
}

// Unwrap 返回底层错误。
func (e *ScriptError) Unwrap() error {
	return e.Err
}

// NewScriptError 创建一个 ScriptError。
func NewScriptError(op string, err error) *ScriptError {
	return &ScriptError{Op: op, Err: err}
}
