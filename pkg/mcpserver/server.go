package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/bigsmater/fridaforge/pkg/codegen"
	"github.com/bigsmater/fridaforge/pkg/config"
	"github.com/bigsmater/fridaforge/pkg/device"
	"github.com/bigsmater/fridaforge/pkg/spec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewMCPServer 创建并配置 MCP Server，注册全部 4 个 Tool。
// generator 用于 spec_generate Tool 调用代码生成器。
func NewMCPServer(generator *codegen.Generator, deviceLister device.DeviceLister, processLister ProcessLister, logger *slog.Logger) *mcp.Server {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	srv := &Server{
		generator:     generator,
		deviceLister:  deviceLister,
		processLister: processLister,
		logger:        logger,
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "fridaforge",
		Version: "0.4.0",
	}, &mcp.ServerOptions{
		Logger: logger,
	})

	srv.registerTools(server)
	return server
}

// registerTools 注册所有 4 个 MCP Tool。
// 使用闭包捕获 Server 的依赖，handler 签名仍符合 go-sdk 的泛型约束。
func (s *Server) registerTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "spec_generate",
		Description: "根据 Hook 参数生成完整的 Frida JavaScript 脚本，支持 overload、override、native 三种 Hook 类型",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GenerateInput) (*mcp.CallToolResult, GenerateOutput, error) {
		return s.generateHandler(ctx, req, input)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "spec_validate",
		Description: "校验 Hook 参数是否合法，返回所有字段级错误和警告信息（comprehensive 模式）",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ValidateInput) (*mcp.CallToolResult, ValidateOutput, error) {
		return s.validateHandler(ctx, req, input)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "device_list",
		Description: "枚举当前连接的调试设备列表",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, DeviceListOutput, error) {
		return s.deviceListHandler(ctx, req, input)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "process_list",
		Description: "枚举指定设备上运行的进程列表",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ProcessListInput) (*mcp.CallToolResult, ProcessListOutput, error) {
		return s.processListHandler(ctx, req, input)
	})
}

// ---------- spec_generate handler ----------

func (s *Server) generateHandler(ctx context.Context, req *mcp.CallToolRequest, input GenerateInput) (*mcp.CallToolResult, GenerateOutput, error) {
	s.logger.Info("tool called", "tool", "spec_generate", "class", input.ClassName, "method", input.MethodName, "type", input.HookType)

	hookSpec := &spec.HookSpec{
		AppPackage: input.AppPackage,
		Hooks: []spec.HookTarget{{
			ClassName:       input.ClassName,
			MethodName:      input.MethodName,
			HookType:        spec.HookType(input.HookType),
			MethodSignature: input.Signature,
			ModuleName:      input.ModuleName,
		}},
	}

	if err := config.Validate(hookSpec); err != nil {
		s.logger.Error("spec_generate 校验失败", "error", err)
		return nil, GenerateOutput{}, fmt.Errorf("参数校验失败: %w", err)
	}

	output, err := s.generator.Generate(hookSpec, "16")
	if err != nil {
		s.logger.Error("spec_generate 生成失败", "error", err)
		return nil, GenerateOutput{}, fmt.Errorf("脚本生成失败: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output.Combined},
		},
	}, GenerateOutput{Script: output.Combined}, nil
}

// ---------- spec_validate handler ----------

func (s *Server) validateHandler(ctx context.Context, req *mcp.CallToolRequest, input ValidateInput) (*mcp.CallToolResult, ValidateOutput, error) {
	s.logger.Info("tool called", "tool", "spec_validate", "class", input.ClassName, "method", input.MethodName)

	hookSpec := &spec.HookSpec{
		AppPackage: input.AppPackage,
		Hooks: []spec.HookTarget{{
			ClassName:       input.ClassName,
			MethodName:      input.MethodName,
			HookType:        spec.HookType(input.HookType),
			MethodSignature: input.Signature,
			ModuleName:      input.ModuleName,
		}},
	}

	err := config.Validate(hookSpec)
	if err == nil {
		return nil, ValidateOutput{Valid: true, Errors: nil, Warnings: nil}, nil
	}

	ve, ok := err.(*spec.ValidationError)
	if !ok {
		return nil, ValidateOutput{}, err
	}

	output := ValidateOutput{Valid: !ve.HasErrors()}

	for _, e := range ve.Errors {
		output.Errors = append(output.Errors, ValidationFieldError{
			Path:    e.Path,
			Message: e.Message,
		})
	}
	for _, w := range ve.Warnings {
		output.Warnings = append(output.Warnings, ValidationFieldError{
			Path:    w.Path,
			Message: w.Message,
		})
	}

	return nil, output, nil
}

// ---------- device_list stub ----------

func (s *Server) deviceListHandler(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, DeviceListOutput, error) {
	// Phase 4 实现
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "// device_list stub: 将在 Phase 4 实现"},
		},
	}, DeviceListOutput{}, nil
}

// ---------- process_list stub ----------

func (s *Server) processListHandler(ctx context.Context, req *mcp.CallToolRequest, input ProcessListInput) (*mcp.CallToolResult, ProcessListOutput, error) {
	// Phase 4 实现
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "// process_list stub: 将在 Phase 4 实现"},
		},
	}, ProcessListOutput{}, nil
}
