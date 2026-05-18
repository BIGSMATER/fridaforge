package config

import (
	"fmt"

	"github.com/bigsmater/fridaforge/pkg/spec"
)

// Validate 校验 HookSpec 的结构合法性。
// 返回 nil 表示校验通过；返回 *spec.ValidationError 表示存在一个或多个字段错误。
// 警告不阻止校验通过，可通过 ValidationError.HasWarnings() 检查。
func Validate(s *spec.HookSpec) error {
	var ve spec.ValidationError

	if s.AppPackage == "" {
		ve.Add("app_package", "不能为空", 0)
	}
	if len(s.Hooks) == 0 {
		ve.Add("hooks", "至少需要一个 Hook 目标", 0)
	}

	seen := make(map[string]int)
	for i, h := range s.Hooks {
		prefix := fmt.Sprintf("hooks[%d]", i)

		switch h.HookType {
		case spec.HookTypeOverload, spec.HookTypeOverride:
			if h.ClassName == "" {
				ve.Add(prefix+".class_name", "不能为空", 0)
			}
		case spec.HookTypeNative:
			if h.ModuleName == "" {
				ve.Add(prefix+".module_name", "不能为空（native Hook 需要 module_name）", 0)
			}
		default:
			ve.Add(prefix+".hook_type",
				fmt.Sprintf("不支持的值 %q，有效的 Hook 类型: overload, override, native", h.HookType),
				0)
		}

		if h.MethodName == "" {
			ve.Add(prefix+".method_name", "不能为空", 0)
		}

		dupKey := fmt.Sprintf("%s|%s|%s|%s", h.ClassName, h.MethodName, h.MethodSignature, h.HookType)
		if prevIdx, exists := seen[dupKey]; exists {
			ve.AddWarning(prefix+".hook_type",
				fmt.Sprintf("与 hooks[%d] 重复 (class_name=%q, method_name=%q, method_signature=%q, hook_type=%q)",
					prevIdx, h.ClassName, h.MethodName, h.MethodSignature, h.HookType),
				0)
		} else {
			seen[dupKey] = i
		}
	}

	if ve.HasErrors() {
		return &ve
	}
	if ve.HasWarnings() {
		return &ve
	}
	return nil
}
