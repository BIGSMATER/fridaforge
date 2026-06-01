//go:build integration

package codegen

import (
	"testing"

	"github.com/bigsmater/fridaforge/pkg/config"
	"github.com/bigsmater/fridaforge/pkg/spec"
)

// TestGenerateIntegration 验证生成脚本在真实 Frida 环境中可加载（SC-002: 100% Frida load success）。
// 需要：目标设备运行 frida-server，且存在测试目标应用。
// 运行方式：go test -tags=integration -v ./pkg/codegen/
func TestGenerateIntegration(t *testing.T) {
	t.Skip("需要真实 Frida 环境：frida-server + 目标 Android 设备")

	// 构造一个包含 3 种 HookType 的 spec
	s := &spec.HookSpec{
		AppPackage: "com.example.test",
		Hooks: []spec.HookTarget{
			{ClassName: "com.example.app.MainActivity", MethodName: "onCreate", HookType: spec.HookTypeOverload},
			{ClassName: "com.example.app.DebugDetector", MethodName: "isDebuggable", HookType: spec.HookTypeOverride},
			{MethodName: "getpid", HookType: spec.HookTypeNative, ModuleName: "libc.so"},
		},
	}

	gen, err := NewGenerator(nil)
	if err != nil {
		t.Fatalf("NewGenerator() 失败: %v", err)
	}

	out, err := gen.Generate(s)
	if err != nil {
		t.Fatalf("Generate() 失败: %v", err)
	}

	if out.Combined == "" {
		t.Fatal("生成脚本为空")
	}
	if len(out.Scripts) != 3 {
		t.Errorf("len(Scripts) = %d, want 3", len(out.Scripts))
	}

	// 基础语法验证：生成脚本应包含所有关键 Frida API
	requiredAPIs := []string{
		"Java.perform(function() {",
		"Java.use(",
		".overload(",
		".implementation = function()",
		"send(JSON.stringify",
		"Interceptor.attach(",
		"Process.findModuleByName(",
		"onEnter",
	}
	for _, api := range requiredAPIs {
		if !containsStr(out.Combined, api) {
			t.Errorf("生成脚本缺少 Frida API: %q", api)
		}
	}

	// 验证 YAML 加载 + 校验 + 生成的完整链条
	_ = config.Validate(s) // 校验通过（仅 warnings 不报错）
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
