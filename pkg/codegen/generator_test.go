package codegen

import (
	"strings"
	"testing"

	"github.com/bigsmater/fridaforge/pkg/spec"
)

func TestGenerate(t *testing.T) {
	g, err := NewGenerator(nil)
	if err != nil {
		t.Fatalf("NewGenerator() unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		spec     spec.HookSpec
		contains []string // Combined 中应包含的子串
		excludes []string // Combined 中不应包含的子串
		wantErr  bool
	}{
		{
			name: "single overload hook",
			spec: spec.HookSpec{
				AppPackage: "com.example.app",
				Hooks: []spec.HookTarget{
					{ClassName: "com.example.app.Crypto", MethodName: "encrypt", HookType: spec.HookTypeOverload, MethodSignature: "java.lang.String"},
				},
			},
			contains: []string{
				"Java.perform(function() {",
				"encrypt",
				"overload('java.lang.String')",
				"this.encrypt.apply",
				"});",
			},
		},
		{
			name: "single override hook",
			spec: spec.HookSpec{
				AppPackage: "com.example.app",
				Hooks: []spec.HookTarget{
					{ClassName: "com.example.app.Foo", MethodName: "bar", HookType: spec.HookTypeOverride},
				},
			},
			contains: []string{
				"Java.perform(function() {",
				"hooked (override)",
				"});",
			},
		},
		{
			name: "single native hook — no Java.perform",
			spec: spec.HookSpec{
				AppPackage: "com.example.app",
				Hooks: []spec.HookTarget{
					{MethodName: "open", HookType: spec.HookTypeNative, ModuleName: "libc.so"},
				},
			},
			contains: []string{
				"Interceptor.attach",
				"Process.findModuleByName",
				"libc.so",
			},
			excludes: []string{
				"Java.perform",
			},
		},
		{
			name: "mixed Java and Native hooks",
			spec: spec.HookSpec{
				AppPackage: "com.example.app",
				Hooks: []spec.HookTarget{
					{ClassName: "com.example.app.A", MethodName: "foo", HookType: spec.HookTypeOverload},
					{MethodName: "getpid", HookType: spec.HookTypeNative, ModuleName: "libc.so"},
					{ClassName: "com.example.app.B", MethodName: "bar", HookType: spec.HookTypeOverride},
				},
			},
			contains: []string{
				"Java.perform(function() {", // 开头
				"});",                       // Java.perform 结束
				"Interceptor.attach",        // Native 在其后
			},
		},
		{
			name: "Combined ends with Native after Java.perform",
			spec: spec.HookSpec{
				AppPackage: "com.example.app",
				Hooks: []spec.HookTarget{
					{ClassName: "com.example.app.X", MethodName: "y", HookType: spec.HookTypeOverride},
					{MethodName: "write", HookType: spec.HookTypeNative, ModuleName: "libc.so"},
				},
			},
			contains: []string{
				"});",                // Java.perform 结束
				"Interceptor.attach", // Native 在其后
			},
		},
		{
			name: "native hook appears after Java.perform() closure",
			spec: spec.HookSpec{
				AppPackage: "com.example.app",
				Hooks: []spec.HookTarget{
					{MethodName: "read", HookType: spec.HookTypeNative, ModuleName: "libc.so"},
					{ClassName: "com.example.app.Z", MethodName: "w", HookType: spec.HookTypeOverload},
				},
			},
			contains: []string{
				"Java.perform(function() {",
				"});",
			},
		},
		{
			name: "method_signature passed through to template",
			spec: spec.HookSpec{
				AppPackage: "com.example.app",
				Hooks: []spec.HookTarget{
					{ClassName: "com.example.app.Crypto", MethodName: "encrypt", HookType: spec.HookTypeOverload, MethodSignature: "byte[], int, java.lang.String"},
				},
			},
			contains: []string{
				"overload('byte[], int, java.lang.String')",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := g.Generate(&tt.spec, "")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Generate() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if out == nil {
				t.Fatal("Generate() returned nil output")
			}
			if out.Combined == "" {
				t.Error("Combined should not be empty")
			}
			if len(out.Scripts) != len(tt.spec.Hooks) {
				t.Errorf("len(Scripts) = %d, want %d", len(out.Scripts), len(tt.spec.Hooks))
			}

			for _, want := range tt.contains {
				if !strings.Contains(out.Combined, want) {
					t.Errorf("Combined missing %q\nCombined:\n%s", want, out.Combined)
				}
			}
			for _, notWant := range tt.excludes {
				if strings.Contains(out.Combined, notWant) {
					t.Errorf("Combined should NOT contain %q\nCombined:\n%s", notWant, out.Combined)
				}
			}
		})
	}

	t.Run("nil spec returns error", func(t *testing.T) {
		_, err := g.Generate(nil, "")
		if err == nil {
			t.Error("Generate(nil) should return error")
		}
	})

	t.Run("empty hooks returns error", func(t *testing.T) {
		_, err := g.Generate(&spec.HookSpec{
			AppPackage: "com.example.app",
			Hooks:      []spec.HookTarget{},
		}, "")
		if err == nil {
			t.Error("Generate() with empty Hooks should return error")
		}
	})

	t.Run("all hooks written to Scripts slice", func(t *testing.T) {
		out, err := g.Generate(&spec.HookSpec{
			AppPackage: "com.example.app",
			Hooks: []spec.HookTarget{
				{ClassName: "com.example.A", MethodName: "a", HookType: spec.HookTypeOverload},
				{ClassName: "com.example.B", MethodName: "b", HookType: spec.HookTypeOverride},
				{MethodName: "c", HookType: spec.HookTypeNative, ModuleName: "libx.so"},
			},
		}, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out.Scripts) != 3 {
			t.Fatalf("len(Scripts) = %d, want 3", len(out.Scripts))
		}
		if out.Scripts[0].JSCode == "" {
			t.Error("Scripts[0].JSCode should not be empty")
		}
		if out.Scripts[1].HookTarget.ClassName != "com.example.B" {
			t.Errorf("Scripts[1].HookTarget.ClassName = %q", out.Scripts[1].HookTarget.ClassName)
		}
		if out.Scripts[2].HookTarget.ModuleName != "libx.so" {
			t.Errorf("Scripts[2].HookTarget.ModuleName = %q", out.Scripts[2].HookTarget.ModuleName)
		}
	})

	t.Run("GenerateOutput Scripts match input Hooks", func(t *testing.T) {
		hooks := []spec.HookTarget{
			{ClassName: "com.example.X", MethodName: "x", HookType: spec.HookTypeOverride},
			{MethodName: "y", HookType: spec.HookTypeNative, ModuleName: "liby.so"},
		}
		out, err := g.Generate(&spec.HookSpec{
			AppPackage: "com.example.app",
			Hooks:      hooks,
		}, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for i, script := range out.Scripts {
			if script.HookTarget.MethodName != hooks[i].MethodName {
				t.Errorf("Scripts[%d].HookTarget.MethodName = %q, want %q",
					i, script.HookTarget.MethodName, hooks[i].MethodName)
			}
		}
	})
}
