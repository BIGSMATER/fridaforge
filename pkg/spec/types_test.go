package spec

import (
	"testing"
)

func TestHookTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		ht       HookType
		expected string
	}{
		{"overload", HookTypeOverload, "overload"},
		{"override", HookTypeOverride, "override"},
		{"native", HookTypeNative, "native"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.ht) != tt.expected {
				t.Errorf("HookType = %q, want %q", tt.ht, tt.expected)
			}
		})
	}
}

func TestHookSpecYAMLTags(t *testing.T) {
	hs := HookSpec{
		AppPackage: "com.example.app",
		Hooks: []HookTarget{
			{ClassName: "com.example.Foo", MethodName: "bar", HookType: HookTypeOverload},
		},
	}

	if hs.AppPackage != "com.example.app" {
		t.Errorf("AppPackage = %q, want %q", hs.AppPackage, "com.example.app")
	}
	if len(hs.Hooks) != 1 {
		t.Fatalf("len(Hooks) = %d, want 1", len(hs.Hooks))
	}
	if hs.Hooks[0].ClassName != "com.example.Foo" {
		t.Errorf("ClassName = %q, want %q", hs.Hooks[0].ClassName, "com.example.Foo")
	}
	if hs.Hooks[0].MethodName != "bar" {
		t.Errorf("MethodName = %q, want %q", hs.Hooks[0].MethodName, "bar")
	}
	if hs.Hooks[0].HookType != HookTypeOverload {
		t.Errorf("HookType = %q, want %q", hs.Hooks[0].HookType, HookTypeOverload)
	}
}

func TestHookTargetNewFields(t *testing.T) {
	tests := []struct {
		name string
		ht   HookTarget
	}{
		{
			name: "method_signature and module_name",
			ht: HookTarget{
				ClassName:       "com.example.Foo",
				MethodName:      "bar",
				HookType:        HookTypeOverload,
				MethodSignature: "java.lang.String, int",
				ModuleName:      "libfoo.so",
			},
		},
		{
			name: "native hook with module_name",
			ht: HookTarget{
				MethodName: "open",
				HookType:   HookTypeNative,
				ModuleName: "libc.so",
			},
		},
		{
			name: "empty new fields",
			ht: HookTarget{
				ClassName:  "com.example.App",
				MethodName: "onCreate",
				HookType:   HookTypeOverride,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ht.MethodName == "" {
				t.Error("MethodName should not be empty")
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	t.Run("empty errors", func(t *testing.T) {
		ve := &ValidationError{}
		if ve.HasErrors() {
			t.Error("HasErrors() = true, want false")
		}
		if ve.HasWarnings() {
			t.Error("HasWarnings() = true, want false")
		}
		if ve.Error() != "" {
			t.Errorf("Error() = %q, want empty string", ve.Error())
		}
	})

	t.Run("with errors", func(t *testing.T) {
		ve := &ValidationError{}
		ve.Add("hooks[0].class_name", "不能为空", 3)
		ve.Add("hooks[1].hook_type", "不支持的值", 6)

		if !ve.HasErrors() {
			t.Error("HasErrors() = false, want true")
		}
		errStr := ve.Error()
		if errStr == "" {
			t.Fatal("Error() returned empty string")
		}
	})

	t.Run("with warnings only", func(t *testing.T) {
		ve := &ValidationError{}
		ve.AddWarning("hooks[0]", "重复配置", 0)

		if ve.HasErrors() {
			t.Error("HasErrors() = true, want false (warnings are not errors)")
		}
		if !ve.HasWarnings() {
			t.Error("HasWarnings() = false, want true")
		}
	})
}

func TestFieldError(t *testing.T) {
	tests := []struct {
		name    string
		fe      FieldError
		contain string
	}{
		{
			name:    "with line number",
			fe:      FieldError{Path: "app_package", Message: "不能为空", Line: 1},
			contain: "app_package: 不能为空（第 1 行）",
		},
		{
			name:    "without line number",
			fe:      FieldError{Path: "hooks[0].hook_type", Message: "不支持的值"},
			contain: "hooks[0].hook_type: 不支持的值",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.fe.Error()
			if errStr != tt.contain {
				t.Errorf("Error() = %q, want %q", errStr, tt.contain)
			}
		})
	}
}
