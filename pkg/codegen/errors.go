package codegen

import "fmt"

// TemplateError represents a failure to compile or render an embedded template.
type TemplateError struct {
	Op   string // operation that failed: "parse" (compilation) or "render" (execution)
	Name string // template file name (e.g. "overload.js.tmpl")
	Err  error  // underlying error from text/template
}

// Error implements the error interface.
func (e *TemplateError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("codegen: %s template %q: %v", e.Op, e.Name, e.Err)
	}
	return fmt.Sprintf("codegen: %s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error for use with errors.Is / errors.As.
func (e *TemplateError) Unwrap() error {
	return e.Err
}

// GenerateError represents a failure during the Generate() process.
type GenerateError struct {
	Op  string // operation: "generate"
	Err error  // underlying error
}

// Error implements the error interface.
func (e *GenerateError) Error() string {
	return fmt.Sprintf("codegen: %s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error for use with errors.Is / errors.As.
func (e *GenerateError) Unwrap() error {
	return e.Err
}
