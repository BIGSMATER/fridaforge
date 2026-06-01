package codegen

import (
	"fmt"
	"strings"

	"github.com/bigsmater/fridaforge/pkg/spec"
)

// Generate 从 HookSpec 生成完整的可执行 Frida JavaScript 脚本。
// fridaVersion 为目标 Frida 版本 ("16" 或 "17")，模板据此选择 API 策略。
// spec 为 nil 或 Hooks 为空时返回 *GenerateError。
func (g *Generator) Generate(s *spec.HookSpec, fridaVersion string) (*GenerateOutput, error) {
	if fridaVersion == "" {
		fridaVersion = "16"
	}
	if s == nil || len(s.Hooks) == 0 {
		return nil, &GenerateError{Op: "generate", Err: fmt.Errorf("空的 Hook 列表")}
	}

	var scripts []GeneratedScript
	var javaBlocks []string
	var nativeBlocks []string

	for _, hook := range s.Hooks {
		ctx := RenderContext{
			AppPackage:      s.AppPackage,
			ClassName:       hook.ClassName,
			MethodName:      hook.MethodName,
			HookType:        string(hook.HookType),
			MethodSignature: hook.MethodSignature,
			ModuleName:      hook.ModuleName,
			FridaVersion:    fridaVersion,
		}

		js, err := g.renderTemplate(ctx)
		if err != nil {
			return nil, &GenerateError{Op: "generate", Err: err}
		}

		scripts = append(scripts, GeneratedScript{
			HookTarget: hook,
			JSCode:     js,
		})

		if hook.HookType == spec.HookTypeNative {
			nativeBlocks = append(nativeBlocks, js)
		} else {
			javaBlocks = append(javaBlocks, js)
		}
	}

	// 组装 Combined 输出
	var combined strings.Builder

	if len(javaBlocks) > 0 {
		combined.WriteString("Java.perform(function() {\n")
		for _, block := range javaBlocks {
			combined.WriteString(block)
		}
		combined.WriteString("});\n")
	}

	if len(nativeBlocks) > 0 {
		if len(javaBlocks) > 0 {
			combined.WriteString("\n")
		}
		for _, block := range nativeBlocks {
			combined.WriteString(block)
			combined.WriteString("\n")
		}
	}

	return &GenerateOutput{
		Combined: combined.String(),
		Scripts:  scripts,
	}, nil
}
