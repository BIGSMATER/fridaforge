package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/bigsmater/fridaforge/pkg/codegen"
	"github.com/bigsmater/fridaforge/pkg/mcpserver"
)

func init() {
	mcpCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {} // 跳过伦理声明——MCP 协议不得在 stdin/stdout 上写入额外内容
	rootCmd.AddCommand(mcpCmd)
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "启动 MCP Server（stdio 传输）",
	Long:  "以 MCP 协议通过 stdin/stdout 与 AI 编码工具通信，暴露 FridaForge 的 Hook 生成、校验、设备枚举、进程枚举能力。",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
		logger.Info("fridaforge MCP Server 启动中")

		gen, err := codegen.NewGenerator(logger)
		if err != nil {
			return err
		}

		store, err := mcpserver.LoadMockStore()
		if err != nil {
			logger.Warn("加载 mock 配置失败，使用内嵌默认值", "error", err)
			store, _ = mcpserver.LoadMockStore()
		}

		server := mcpserver.NewMCPServer(gen, store.DeviceLister, store.ProcessLister, logger)

		transport := &mcp.StdioTransport{}
		logger.Info("MCP Server 已就绪，等待客户端连接")
		if err := server.Run(context.Background(), transport); err != nil {
			logger.Error("MCP Server 退出", "error", err)
			return err
		}
		return nil
	},
}
