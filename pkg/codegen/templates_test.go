package codegen

import (
	"strings"
	"testing"
)

func TestNewGenerator(t *testing.T) {
	t.Run("successful compilation", func(t *testing.T) {
		g, err := NewGenerator(nil)
		if err != nil {
			t.Fatalf("NewGenerator() unexpected error: %v", err)
		}
		if g == nil {
			t.Fatal("NewGenerator() returned nil Generator")
		}
		if g.tmpl == nil {
			t.Error("Generator.tmpl should not be nil after successful compilation")
		}
	})
}

func TestRenderTemplate(t *testing.T) {
	g, err := NewGenerator(nil)
	if err != nil {
		t.Fatalf("NewGenerator() unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		ctx      RenderContext
		contains []string        // substrings expected in output
		excludes []string        // substrings NOT expected in output
	}{
		{
			name: "overload with signature",
			ctx: RenderContext{
				AppPackage:      "com.example.app",
				ClassName:       "com.example.app.Crypto",
				MethodName:      "encrypt",
				HookType:        "overload",
				MethodSignature: "java.lang.String, byte[]",
			},
			contains: []string{
				"com.example.app.Crypto",
				"encrypt",
				"overload('java.lang.String, byte[]')",
				"this.encrypt.apply",
				"send(JSON.stringify",
			},
		},
		{
			name: "overload without signature",
			ctx: RenderContext{
				AppPackage: "com.example.app",
				ClassName:  "com.example.app.Foo",
				MethodName: "bar",
				HookType:   "overload",
			},
			contains: []string{
				"overload()",
				"this.bar.apply",
			},
			excludes: []string{
				"overload('", // should be overload() not overload('')
			},
		},
		{
			name: "override with signature",
			ctx: RenderContext{
				AppPackage:      "com.example.app",
				ClassName:       "com.example.app.Factory",
				MethodName:      "create",
				HookType:        "override",
				MethodSignature: "int",
			},
			contains: []string{
				"com.example.app.Factory",
				"create",
				"overload('int')",
				"hooked (override)",
				"send(JSON.stringify",
			},
		},
		{
			name: "override without signature",
			ctx: RenderContext{
				AppPackage: "com.example.app",
				ClassName:  "com.example.app.Bar",
				MethodName: "destroy",
				HookType:   "override",
			},
			contains: []string{
				"overload()",
				"hooked (override)",
			},
		},
		{
			name: "native with module",
			ctx: RenderContext{
				MethodName: "open",
				HookType:   "native",
				ModuleName: "libc.so",
			},
			contains: []string{
				"Process.findModuleByName",
				"libc.so",
				"Module.findExportByName",
				"Interceptor.attach",
				"onEnter",
				"onLeave",
				"send(JSON.stringify",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			js, err := g.renderTemplate(tt.ctx)
			if err != nil {
				t.Fatalf("renderTemplate() error: %v", err)
			}
			if js == "" {
				t.Fatal("renderTemplate() returned empty string")
			}

			for _, want := range tt.contains {
				if !strings.Contains(js, want) {
					t.Errorf("output missing %q\nGot:\n%s", want, js)
				}
			}
			for _, notWant := range tt.excludes {
				if strings.Contains(js, notWant) {
					t.Errorf("output should NOT contain %q\nGot:\n%s", notWant, js)
				}
			}
		})
	}

	t.Run("unknown hook type", func(t *testing.T) {
		_, err := g.renderTemplate(RenderContext{
			HookType: "invalid_type",
		})
		if err == nil {
			t.Error("renderTemplate() expected error for unknown hook type")
		}
	})

	t.Run("empty method_signature uses overload with no args", func(t *testing.T) {
		ctx := RenderContext{
			ClassName:  "com.example.X",
			MethodName: "y",
			HookType:   "overload",
		}
		js, err := g.renderTemplate(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(js, "overload()") {
			t.Errorf("expected overload() when signature is empty, got:\n%s", js)
		}
		if strings.Contains(js, "overload('") {
			t.Errorf("should NOT contain overload(' when signature is empty, got:\n%s", js)
		}
	})

	t.Run("native output is not wrapped in Java.perform", func(t *testing.T) {
		ctx := RenderContext{
			MethodName: "getpid",
			HookType:   "native",
			ModuleName: "libc.so",
		}
		js, err := g.renderTemplate(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Native template uses Interceptor.attach, never Java.perform
		if strings.Contains(js, "Java.perform") {
			t.Error("native template should not contain Java.perform()")
		}
	})
}
