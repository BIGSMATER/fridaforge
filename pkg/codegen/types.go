// Package codegen provides a declarative Frida JavaScript code generator.
// It reads a HookSpec (from pkg/spec) and renders executable Frida JS scripts
// via text/template from embedded template files.
package codegen

import "github.com/bigsmater/fridaforge/pkg/spec"

// GenerateOutput holds the result of a Generate() call.
type GenerateOutput struct {
	// Combined is the complete executable Frida JavaScript script, with all
	// Java hooks wrapped in a single Java.perform() and Native hooks appended after.
	Combined string

	// Scripts holds the individual generated JavaScript snippet for each HookTarget.
	Scripts []GeneratedScript
}

// GeneratedScript pairs a HookTarget with its rendered JavaScript code.
type GeneratedScript struct {
	HookTarget spec.HookTarget
	JSCode     string
}

// RenderContext provides the data injected into a template during rendering.
// Fields are populated by Generator.Generate() before calling renderTemplate().
type RenderContext struct {
	AppPackage      string // parent HookSpec.AppPackage
	ClassName       string // target class name (Java hooks)
	MethodName      string // target method or function name
	HookType        string // "overload" / "override" / "native"
	MethodSignature string // full method signature string (empty → no signature)
	ModuleName      string // .so module name (Native hooks only)
}
