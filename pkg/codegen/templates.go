package codegen

import (
	"embed"
	"fmt"
	"log/slog"
	"strings"
	"text/template"
)

//go:embed templates/*.js.tmpl
var templateFS embed.FS

// Generator 持有预编译的 Frida JS 模板集，从 HookSpec 渲染可执行脚本。
type Generator struct {
	tmpl   *template.Template
	logger *slog.Logger
}

// NewGenerator 编译内嵌模板并返回可用的 Generator。
// 编译失败返回 *TemplateError（fail-fast：内嵌模板在 init 期检查，编译错误 = 源码 bug）。
func NewGenerator(logger *slog.Logger) (*Generator, error) {
	if logger == nil {
		logger = slog.Default()
	}
	tmpl, err := template.ParseFS(templateFS, "templates/*.js.tmpl")
	if err != nil {
		return nil, &TemplateError{Op: "parse", Err: err}
	}
	logger.Debug("codegen: 模板编译成功")
	return &Generator{tmpl: tmpl, logger: logger}, nil
}

// renderTemplate 使用给定的 RenderContext 渲染单个模板。
// 根据 HookType 选择对应的 .tmpl 文件，模板中的条件分支配
// method_signature / module_name 的处理。
func (g *Generator) renderTemplate(ctx RenderContext) (string, error) {
	var name string
	switch ctx.HookType {
	case "overload":
		name = "overload.js.tmpl"
	case "override":
		name = "override.js.tmpl"
	case "native":
		name = "native.js.tmpl"
	default:
		return "", &TemplateError{Op: "render", Name: ctx.HookType,
			Err: fmt.Errorf("未知的 Hook 类型: %q", ctx.HookType)}
	}

	var buf strings.Builder
	if err := g.tmpl.ExecuteTemplate(&buf, name, ctx); err != nil {
		return "", &TemplateError{Op: "render", Name: name, Err: err}
	}
	return buf.String(), nil
}
