package codegen

import "fmt"

// TemplateError 表示内嵌模板编译或渲染阶段的失败。
type TemplateError struct {
	Op   string // 失败的操作: "parse"（编译）或 "render"（渲染执行）
	Name string // 模板文件名（如 "overload.js.tmpl"）
	Err  error  // 底层 text/template 的错误
}

// Error 实现 error 接口。
func (e *TemplateError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("codegen: %s 模板 %q: %v", e.Op, e.Name, e.Err)
	}
	return fmt.Sprintf("codegen: %s: %v", e.Op, e.Err)
}

// Unwrap 返回底层错误，供 errors.Is / errors.As 穿透。
func (e *TemplateError) Unwrap() error {
	return e.Err
}

// GenerateError 表示 Generate() 生成过程的失败。
type GenerateError struct {
	Op  string // 操作: "generate"
	Err error  // 底层错误
}

// Error 实现 error 接口。
func (e *GenerateError) Error() string {
	return fmt.Sprintf("codegen: %s: %v", e.Op, e.Err)
}

// Unwrap 返回底层错误，供 errors.Is / errors.As 穿透。
func (e *GenerateError) Unwrap() error {
	return e.Err
}
