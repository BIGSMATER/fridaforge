package config

import (
	"fmt"

	"github.com/bigsmater/fridaforge/pkg/spec"
)

// Validate 校验 HookSpec 的结构合法性。
// 返回 nil 表示校验通过；返回 *spec.ValidationError 表示存在一个或多个字段错误。
func Validate(s *spec.HookSpec) error {
	var ve spec.ValidationError

	if s.AppPackage == "" {
		ve.Add("app_package", "不能为空", 0)
	}
	if len(s.Hooks) == 0 {
		ve.Add("hooks", "至少需要一个 Hook 目标", 0)
	}

	for i, h := range s.Hooks {
		prefix := fmt.Sprintf("hooks[%d]", i)

		if h.ClassName == "" {
			ve.Add(prefix+".class_name", "不能为空", 0)
		}
		if h.MethodName == "" {
			ve.Add(prefix+".method_name", "不能为空", 0)
		}
		if h.HookType != spec.HookTypeOverload && h.HookType != spec.HookTypeReplace {
			ve.Add(prefix+".hook_type",
				fmt.Sprintf("不支持的值 %q，有效的 Hook 类型: overload, replace", h.HookType),
				0)
		}
	}

	if ve.HasErrors() {
		return &ve
	}
	return nil
}
