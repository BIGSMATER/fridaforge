package fridaengine

import (
	"time"

	"github.com/frida/frida-go/frida"
)

// HookScript 包装 frida.Script，管理脚本生命周期和消息回调。
// 这是一个内部类型——不直接暴露给调用者，通过 HookSession 间接使用。
type HookScript struct {
	script *frida.Script
}

// Load 加载并执行脚本。
func (s *HookScript) Load() error {
	return s.script.Load()
}

// Unload 卸载脚本。
func (s *HookScript) Unload() error {
	return s.script.Unload()
}

// onMessage 注册消息回调，将 frida 消息字符串转为 HookMessage 发送到 channel。
func (s *HookScript) onMessage(ch chan<- HookMessage) {
	s.script.On("message", func(msg string) {
		ch <- HookMessage{
			Type:      "message",
			Payload:   msg,
			Timestamp: time.Now(),
		}
	})
}

// createScript 使用 frida.Session 创建脚本。
func createScript(session *frida.Session, jsSource string) (*HookScript, error) {
	script, err := session.CreateScript(jsSource)
	if err != nil {
		return nil, &ScriptError{Op: "create", Err: err}
	}
	return &HookScript{script: script}, nil
}
