package mcpserver

import (
	"context"
	"log/slog"
	"os"

	"github.com/bigsmater/fridaforge/pkg/device"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewMCPServer 创建并配置 MCP Server，注册全部 4 个 Tool。
// 初期使用 stub handler，后续 Phase 由 tools_spec.go 和 tools_device.go 实现真实逻辑。
// 依赖通过参数注入：deviceLister、processLister 和 logger，
// 实现方无需感知这些依赖的具体来源。
func NewMCPServer(deviceLister device.DeviceLister, processLister ProcessLister, logger *slog.Logger) *mcp.Server {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "fridaforge",
		Version: "0.4.0",
	}, &mcp.ServerOptions{
		Logger: logger,
	})

	registerTools(server, logger)

	return server
}

// registerTools 注册所有 4 个 MCP Tool。
// 当前为 stub 实现，后续 Phase 会替换为真实 handler。
func registerTools(server *mcp.Server, logger *slog.Logger) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "spec_generate",
		Description: "根据 Hook 参数生成完整的 Frida JavaScript 脚本，支持 overload、override、native 三种 Hook 类型",
	}, generateHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "spec_validate",
		Description: "校验 Hook 参数是否合法，返回所有字段级错误和警告信息",
	}, validateHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "device_list",
		Description: "枚举当前连接的调试设备列表",
	}, deviceListHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "process_list",
		Description: "枚举指定设备上运行的进程列表",
	}, processListHandler)
}

// ---------- Stub handlers（Phase 3/4 实现真实逻辑）----------

func generateHandler(ctx context.Context, req *mcp.CallToolRequest, input GenerateInput) (*mcp.CallToolResult, GenerateOutput, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "// generate stub: 将在 Phase 3 实现"},
		},
	}, GenerateOutput{Script: "// stub"}, nil
}

func validateHandler(ctx context.Context, req *mcp.CallToolRequest, input GenerateInput) (*mcp.CallToolResult, ValidateOutput, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "// validate stub: 将在 Phase 3 实现"},
		},
	}, ValidateOutput{Valid: true}, nil
}

func deviceListHandler(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, DeviceListOutput, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "// device_list stub: 将在 Phase 4 实现"},
		},
	}, DeviceListOutput{}, nil
}

func processListHandler(ctx context.Context, req *mcp.CallToolRequest, input ProcessListInput) (*mcp.CallToolResult, ProcessListOutput, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "// process_list stub: 将在 Phase 4 实现"},
		},
	}, ProcessListOutput{}, nil
}
