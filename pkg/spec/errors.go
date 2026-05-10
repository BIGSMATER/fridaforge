package spec

import (
	"fmt"
	"strings"
)

// FieldError 表示单个字段级的校验错误。
type FieldError struct {
	Path    string // 字段路径，如 "hooks[0].class_name"
	Message string // 错误描述，如 "不能为空"
	Line    int    // YAML 行号（0 表示未知）
}

// Error 实现 error 接口。
func (e *FieldError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s: %s（第 %d 行）", e.Path, e.Message, e.Line)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// ValidationError 表示一次校验操作的结果，包含零个或多个字段错误。
// 当 Errors 为空时表示校验通过。
type ValidationError struct {
	Errors []FieldError
}

// Error 实现 error 接口，渲染所有字段错误。
func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("配置校验失败:\n")
	for _, fe := range e.Errors {
		b.WriteString("  ")
		b.WriteString(fe.Error())
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// Add 往校验错误中追加一个字段错误。
func (e *ValidationError) Add(path, msg string, line int) {
	e.Errors = append(e.Errors, FieldError{
		Path:    path,
		Message: msg,
		Line:    line,
	})
}

// HasErrors 返回校验是否包含错误。
func (e *ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}
