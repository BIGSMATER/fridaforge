package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bigsmater/fridaforge/pkg/spec"
)

func TestLoadSpec(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fridaforge-loader-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("valid YAML", func(t *testing.T) {
		content := `app_package: com.example.app
hooks:
  - class_name: com.example.MainActivity
    method_name: onCreate
    hook_type: overload
  - class_name: com.example.Utils
    method_name: encrypt
    hook_type: replace
`
		path := filepath.Join(tmpDir, "valid.yaml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("写入测试文件失败: %v", err)
		}

		s, err := LoadSpec(path)
		if err != nil {
			t.Fatalf("LoadSpec() unexpected error: %v", err)
		}
		if s.AppPackage != "com.example.app" {
			t.Errorf("AppPackage = %q, want %q", s.AppPackage, "com.example.app")
		}
		if len(s.Hooks) != 2 {
			t.Fatalf("len(Hooks) = %d, want 2", len(s.Hooks))
		}
		if s.Hooks[0].ClassName != "com.example.MainActivity" {
			t.Errorf("Hooks[0].ClassName = %q", s.Hooks[0].ClassName)
		}
		if s.Hooks[0].MethodName != "onCreate" {
			t.Errorf("Hooks[0].MethodName = %q", s.Hooks[0].MethodName)
		}
		if s.Hooks[0].HookType != spec.HookTypeOverload {
			t.Errorf("Hooks[0].HookType = %q", s.Hooks[0].HookType)
		}
		if s.Hooks[1].ClassName != "com.example.Utils" {
			t.Errorf("Hooks[1].ClassName = %q", s.Hooks[1].ClassName)
		}
		if s.Hooks[1].MethodName != "encrypt" {
			t.Errorf("Hooks[1].MethodName = %q", s.Hooks[1].MethodName)
		}
		if s.Hooks[1].HookType != spec.HookTypeReplace {
			t.Errorf("Hooks[1].HookType = %q", s.Hooks[1].HookType)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadSpec(filepath.Join(tmpDir, "nonexistent.yaml"))
		if err == nil {
			t.Error("LoadSpec() expected error for nonexistent file")
		}
	})

	t.Run("single hook YAML", func(t *testing.T) {
		content := `app_package: com.example.single
hooks:
  - class_name: com.example.Foo
    method_name: bar
    hook_type: replace
`
		path := filepath.Join(tmpDir, "single.yaml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("写入测试文件失败: %v", err)
		}

		s, err := LoadSpec(path)
		if err != nil {
			t.Fatalf("LoadSpec() unexpected error: %v", err)
		}
		if s.AppPackage != "com.example.single" {
			t.Errorf("AppPackage = %q", s.AppPackage)
		}
		if len(s.Hooks) != 1 {
			t.Fatalf("len(Hooks) = %d, want 1", len(s.Hooks))
		}
	})

	t.Run("invalid YAML syntax", func(t *testing.T) {
		content := `app_package: com.example.app
hooks:
  - class_name: broken
    method_name
`
		path := filepath.Join(tmpDir, "invalid.yaml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("写入测试文件失败: %v", err)
		}

		_, err := LoadSpec(path)
		if err == nil {
			t.Error("LoadSpec() expected error for invalid YAML")
		}
	})
}
