package codegen

import (
	"testing"

	"github.com/bigsmater/fridaforge/pkg/spec"
)

func TestGenerateOutput(t *testing.T) {
	t.Run("empty output", func(t *testing.T) {
		out := &GenerateOutput{}
		if out.Combined != "" {
			t.Error("Combined should be empty for zero value")
		}
		if out.Scripts != nil {
			t.Error("Scripts should be nil for zero value")
		}
	})

	t.Run("with scripts", func(t *testing.T) {
		ht := spec.HookTarget{ClassName: "com.example.Foo", MethodName: "bar", HookType: spec.HookTypeOverload}
		out := &GenerateOutput{
			Combined: "Java.perform(function() { /* hooks */ });",
			Scripts: []GeneratedScript{
				{HookTarget: ht, JSCode: "console.log('hello');"},
			},
		}
		if out.Combined == "" {
			t.Error("Combined should not be empty")
		}
		if len(out.Scripts) != 1 {
			t.Errorf("len(Scripts) = %d, want 1", len(out.Scripts))
		}
		if out.Scripts[0].HookTarget.ClassName != "com.example.Foo" {
			t.Errorf("Scripts[0].HookTarget.ClassName = %q", out.Scripts[0].HookTarget.ClassName)
		}
		if out.Scripts[0].JSCode == "" {
			t.Error("Scripts[0].JSCode should not be empty")
		}
	})
}

func TestGeneratedScript(t *testing.T) {
	ht := spec.HookTarget{
		ClassName:       "com.example.app.MainActivity",
		MethodName:      "onCreate",
		HookType:        spec.HookTypeOverride,
		MethodSignature: "android.os.Bundle",
	}
	gs := GeneratedScript{
		HookTarget: ht,
		JSCode:     "var M = Java.use(\"com.example.app.MainActivity\");",
	}

	if gs.HookTarget.MethodName != "onCreate" {
		t.Errorf("MethodName = %q", gs.HookTarget.MethodName)
	}
	if gs.JSCode == "" {
		t.Error("JSCode should not be empty")
	}
}

func TestRenderContext(t *testing.T) {
	tests := []struct {
		name string
		rc   RenderContext
	}{
		{
			name: "full context for overload",
			rc: RenderContext{
				AppPackage:      "com.example.app",
				ClassName:       "com.example.app.Crypto",
				MethodName:      "encrypt",
				HookType:        "overload",
				MethodSignature: "java.lang.String, byte[]",
			},
		},
		{
			name: "native context with module_name",
			rc: RenderContext{
				MethodName: "open",
				HookType:   "native",
				ModuleName: "libc.so",
			},
		},
		{
			name: "override without signature",
			rc: RenderContext{
				AppPackage: "com.example.app",
				ClassName:  "com.example.app.DebugDetector",
				MethodName: "isDebuggable",
				HookType:   "override",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.rc.MethodName == "" {
				t.Error("MethodName should not be empty")
			}
		})
	}
}
