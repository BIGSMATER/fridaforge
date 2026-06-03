package mcpserver

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/bigsmater/fridaforge/pkg/codegen"
)

func createTestGenerator(t *testing.T) *codegen.Generator {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	gen, err := codegen.NewGenerator(logger)
	if err != nil {
		t.Fatalf("创建 codegen.Generator 失败: %v", err)
	}
	return gen
}

// ---------- T009: spec_generate handler 测试 ----------

func TestGenerateHandler_Success(t *testing.T) {
	gen := createTestGenerator(t)
	srv := &Server{generator: gen, logger: slog.New(slog.DiscardHandler)}

	tests := []struct {
		name  string
		input GenerateInput
		want  string
	}{
		{
			name: "overload 类型",
			input: GenerateInput{
				AppPackage: "com.example.app",
				ClassName:  "com.example.Test",
				MethodName: "hello",
				HookType:   "overload",
			},
			want: "Java.perform",
		},
		{
			name: "override 类型",
			input: GenerateInput{
				AppPackage: "com.example.app",
				ClassName:  "com.example.Test",
				MethodName: "hello",
				HookType:   "override",
			},
			want: "Java.perform",
		},
		{
			name: "native 类型",
			input: GenerateInput{
				AppPackage: "com.example.app",
				ClassName:  "com.example.Test",
				MethodName: "hello",
				HookType:   "native",
				ModuleName: "libtest.so",
			},
			want: "Interceptor.attach",
		},
		{
			name: "带签名 overload",
			input: GenerateInput{
				AppPackage: "com.example.app",
				ClassName:  "com.example.Test",
				MethodName: "hello",
				HookType:   "overload",
				Signature:  "java.lang.String,int",
			},
			want: "overload",
		},
		{
			name: "Frida 17 版本",
			input: GenerateInput{
				AppPackage:   "com.example.app",
				ClassName:    "com.example.Test",
				MethodName:   "hello",
				HookType:     "overload",
				FridaVersion: "17",
			},
			want: "Java.perform",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, output, err := srv.generateHandler(context.Background(), nil, tt.input)
			if err != nil {
				t.Fatalf("generateHandler() 错误: %v", err)
			}
			if !strings.Contains(output.Script, tt.want) {
				t.Errorf("脚本中缺少 %q: %s", tt.want, output.Script)
			}
		})
	}
}

func TestGenerateHandler_ValidationError(t *testing.T) {
	gen := createTestGenerator(t)
	srv := &Server{generator: gen, logger: slog.New(slog.DiscardHandler)}

	tests := []struct {
		name  string
		input GenerateInput
	}{
		{"空 app_package", GenerateInput{ClassName: "Test", MethodName: "m", HookType: "overload"}},
		{"空 class_name", GenerateInput{AppPackage: "pkg", MethodName: "m", HookType: "overload"}},
		{"空 method_name", GenerateInput{AppPackage: "pkg", ClassName: "Test", HookType: "overload"}},
		{"invalid hook_type", GenerateInput{AppPackage: "pkg", ClassName: "T", MethodName: "m", HookType: "bad"}},
		{"native 缺 module_name", GenerateInput{AppPackage: "pkg", ClassName: "T", MethodName: "m", HookType: "native"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := srv.generateHandler(context.Background(), nil, tt.input)
			if err == nil {
				t.Fatal("期望校验错误，实际 nil")
			}
		})
	}
}

// ---------- T010: spec_validate handler 测试 ----------

func TestValidateHandler_Success(t *testing.T) {
	srv := &Server{logger: slog.New(slog.DiscardHandler)}
	input := ValidateInput{
		AppPackage: "com.example.app",
		ClassName:  "com.example.Test",
		MethodName: "hello",
		HookType:   "overload",
	}

	_, output, err := srv.validateHandler(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("validateHandler() 错误: %v", err)
	}
	if !output.Valid {
		t.Errorf("合法参数应通过校验，got Valid=%v", output.Valid)
	}
	if len(output.Errors) != 0 {
		t.Errorf("合法参数应无错误，got %d errors", len(output.Errors))
	}
}

func TestValidateHandler_SingleError(t *testing.T) {
	srv := &Server{logger: slog.New(slog.DiscardHandler)}

	tests := []struct {
		name     string
		input    ValidateInput
		wantPath string
	}{
		{"空 app_package", ValidateInput{ClassName: "T", MethodName: "m", HookType: "overload"}, "app_package"},
		{"空 class_name", ValidateInput{AppPackage: "pkg", MethodName: "m", HookType: "overload"}, "hooks[0].class_name"},
		{"空 method_name", ValidateInput{AppPackage: "pkg", ClassName: "T", HookType: "overload"}, "hooks[0].method_name"},
		{"invalid hook_type", ValidateInput{AppPackage: "pkg", ClassName: "T", MethodName: "m", HookType: "bad"}, "hooks[0].hook_type"},
		{"native 缺 module_name", ValidateInput{AppPackage: "pkg", ClassName: "T", MethodName: "m", HookType: "native"}, "hooks[0].module_name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, output, err := srv.validateHandler(context.Background(), nil, tt.input)
			if err != nil {
				t.Fatalf("validateHandler() 错误: %v", err)
			}
			if output.Valid {
				t.Fatalf("非法参数应不通过，got Valid=true")
			}
			if len(output.Errors) == 0 {
				t.Fatal("期望至少一个错误，got 0")
			}
			if output.Errors[0].Path != tt.wantPath {
				t.Errorf("错误路径 = %q, 期望 %q", output.Errors[0].Path, tt.wantPath)
			}
		})
	}
}

func TestValidateHandler_MultipleErrors(t *testing.T) {
	// 同时两个错误：空 app_package + 空 class_name → 应返回全部
	srv := &Server{logger: slog.New(slog.DiscardHandler)}
	input := ValidateInput{HookType: "overload"} // 全部必填字段为空

	_, output, err := srv.validateHandler(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("validateHandler() 错误: %v", err)
	}
	if len(output.Errors) < 2 {
		t.Errorf("comprehensive 模式应返回 ≥2 个错误，实际 %d", len(output.Errors))
	}
}
