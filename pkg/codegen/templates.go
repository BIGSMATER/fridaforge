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

// Generator holds a pre-compiled set of Frida JS templates and renders
// executable JavaScript from HookSpec declarations.
type Generator struct {
	tmpl   *template.Template
	logger *slog.Logger
}

// NewGenerator compiles embedded templates and returns a ready-to-use Generator.
// Returns *TemplateError on compilation failure (fail-fast: embedded templates
// are checked at init, a parse error means a source code bug).
func NewGenerator(logger *slog.Logger) (*Generator, error) {
	if logger == nil {
		logger = slog.Default()
	}
	tmpl, err := template.ParseFS(templateFS, "templates/*.js.tmpl")
	if err != nil {
		return nil, &TemplateError{Op: "parse", Err: err}
	}
	logger.Debug("codegen: template compilation successful")
	return &Generator{tmpl: tmpl, logger: logger}, nil
}

// renderTemplate renders a single template with the given RenderContext.
// It selects the correct .tmpl file based on HookType and handles
// method_signature / module_name through the template conditionals.
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
			Err: fmt.Errorf("unknown hook type: %q", ctx.HookType)}
	}

	var buf strings.Builder
	if err := g.tmpl.ExecuteTemplate(&buf, name, ctx); err != nil {
		return "", &TemplateError{Op: "render", Name: name, Err: err}
	}
	return buf.String(), nil
}
