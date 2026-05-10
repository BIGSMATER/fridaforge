package config

import (
	"testing"

	"github.com/bigsmater/fridaforge/pkg/spec"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    spec.HookSpec
		wantErr bool
		errMsgs []string
	}{
		{
			name: "valid spec",
			spec: spec.HookSpec{
				AppPackage: "com.example.app",
				Hooks: []spec.HookTarget{
					{ClassName: "com.example.MainActivity", MethodName: "onCreate", HookType: spec.HookTypeOverload},
				},
			},
			wantErr: false,
		},
		{
			name: "empty app_package",
			spec: spec.HookSpec{
				AppPackage: "",
				Hooks: []spec.HookTarget{
					{ClassName: "com.example.Foo", MethodName: "bar", HookType: spec.HookTypeOverload},
				},
			},
			wantErr: true,
			errMsgs: []string{"app_package", "不能为空"},
		},
		{
			name: "empty hooks list",
			spec: spec.HookSpec{
				AppPackage: "com.example.app",
				Hooks:      []spec.HookTarget{},
			},
			wantErr: true,
			errMsgs: []string{"hooks", "至少需要"},
		},
		{
			name: "empty class_name",
			spec: spec.HookSpec{
				AppPackage: "com.example.app",
				Hooks: []spec.HookTarget{
					{ClassName: "", MethodName: "onCreate", HookType: spec.HookTypeOverload},
				},
			},
			wantErr: true,
			errMsgs: []string{"hooks[0].class_name", "不能为空"},
		},
		{
			name: "empty method_name",
			spec: spec.HookSpec{
				AppPackage: "com.example.app",
				Hooks: []spec.HookTarget{
					{ClassName: "com.example.Foo", MethodName: "", HookType: spec.HookTypeOverload},
				},
			},
			wantErr: true,
			errMsgs: []string{"hooks[0].method_name", "不能为空"},
		},
		{
			name: "invalid hook_type",
			spec: spec.HookSpec{
				AppPackage: "com.example.app",
				Hooks: []spec.HookTarget{
					{ClassName: "com.example.Foo", MethodName: "bar", HookType: "unknown"},
				},
			},
			wantErr: true,
			errMsgs: []string{"hooks[0].hook_type", "不支持"},
		},
		{
			name: "multiple hooks valid",
			spec: spec.HookSpec{
				AppPackage: "com.example.app",
				Hooks: []spec.HookTarget{
					{ClassName: "com.example.A", MethodName: "foo", HookType: spec.HookTypeOverload},
					{ClassName: "com.example.B", MethodName: "bar", HookType: spec.HookTypeReplace},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple errors reported",
			spec: spec.HookSpec{
				AppPackage: "",
				Hooks: []spec.HookTarget{
					{ClassName: "", MethodName: "bar", HookType: spec.HookTypeOverload},
				},
			},
			wantErr: true,
			errMsgs: []string{"app_package", "hooks[0].class_name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.spec)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && len(tt.errMsgs) > 0 {
				errStr := err.Error()
				for _, msg := range tt.errMsgs {
					if !contains(errStr, msg) {
						t.Errorf("error message missing expected substring %q\nGot: %s", msg, errStr)
					}
				}
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
