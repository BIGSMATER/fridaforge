//go:build integration

package codegen

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/frida/frida-go/frida"

	"github.com/bigsmater/fridaforge/pkg/spec"
)

// TestGenerateIntegration 验证生成脚本在真实 Frida 环境中可加载（SC-002: 100% Frida load success）。
// 需要：USB 连接的 Android 设备 + 运行中的 frida-server
// 运行方式：CGO_CFLAGS=-I$(pwd)/.devkit CGO_LDFLAGS=-L$(pwd)/.devkit go test -tags=integration -v -run TestGenerateIntegration ./pkg/codegen/
func TestGenerateIntegration(t *testing.T) {
	gen, err := NewGenerator(nil)
	if err != nil {
		t.Fatalf("NewGenerator() 失败: %v", err)
	}

	s := &spec.HookSpec{
		AppPackage: "android",
		Hooks: []spec.HookTarget{
			{
				ClassName:       "java.lang.Runtime",
				MethodName:      "exec",
				HookType:        spec.HookTypeOverload,
				MethodSignature: "java.lang.String",
			},
			{
				ClassName:  "java.lang.System",
				MethodName: "getProperty",
				HookType:   spec.HookTypeOverride,
			},
			{
				MethodName: "open",
				HookType:   spec.HookTypeNative,
				ModuleName: "libc.so",
			},
		},
	}

	out, err := gen.Generate(s)
	if err != nil {
		t.Fatalf("Generate() 失败: %v", err)
	}
	if out.Combined == "" {
		t.Fatal("生成脚本为空")
	}
	t.Logf("生成的脚本 (%d 字节) — 包含 %d 个 Hook", len(out.Combined), len(out.Scripts))

	// 尝试连接设备并注入脚本
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mgr := frida.NewDeviceManager()
	defer mgr.Close()

	devices, err := mgr.EnumerateDevices()
	if err != nil {
		t.Fatalf("枚举设备失败: %v", err)
	}

	attached := false
	for _, d := range devices {
		dt := d.DeviceType()
		t.Logf("设备: %s (ID=%s, Type=%d)", d.Name(), d.ID(), dt)
		if dt == frida.DeviceTypeLocal {
			continue
		}

		session, attachErr := d.Attach("system_server", nil)
		if attachErr != nil {
			t.Logf("  Attach system_server 失败: %v", attachErr)
			continue
		}

		script, createErr := session.CreateScript(out.Combined)
		if createErr != nil {
			session.Detach()
			t.Fatalf("CreateScript 失败 (生成的 JS 有语法错误? SC-002 不通过): %v", createErr)
		}

		msgCh := make(chan string, 1)
		script.On("message", func(msg string) {
			select {
			case msgCh <- msg:
			default:
			}
		})

		if loadErr := script.Load(); loadErr != nil {
			session.Detach()
			t.Fatalf("脚本加载失败 (SC-002 不通过): %v", loadErr)
		}

		t.Logf("✅ 脚本加载成功 — SC-002 通过 (%s / %s)", d.Name(), d.ID())

		// 等待 Hook 消息
		select {
		case msg := <-msgCh:
			t.Logf("✅ 收到 Hook 消息 — 脚本正常运行: %s", msg)
		case <-time.After(3 * time.Second):
			t.Log("⚠️ 3s 内未收到消息 — 脚本已加载但未触发 Hook（不影响语法正确性）")
		case <-ctx.Done():
		}

		script.Unload()
		session.Detach()
		attached = true
		break
	}

	if !attached {
		t.Skip(fmt.Sprintf("未找到可连接的设备 (共 %d 个设备)，SC-002 需真机环境验证", len(devices)))
	}
}
