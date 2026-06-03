# Implementation Plan: MCP Server 集成

**Branch**: `004-mcpserver` | **Date**: 2026-06-02 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-mcpserver/spec.md`

## Summary

将 FridaForge 的 Hook 脚本生成和设备管理能力通过 MCP (Model Context Protocol) 协议暴露给 AI 编码工具。使用官方 `modelcontextprotocol/go-sdk`，通过 stdio transport 作为子进程与 AI 助手通信。暴露 4 个 Tool: spec_generate（脚本生成）、spec_validate（参数校验）、device_list（设备枚举）、process_list（进程枚举）。复用现有的 `pkg/codegen`、`pkg/config`、`pkg/device` 包，新增 `pkg/mcpserver` 和 `cmd/fridaforge/mcp.go`。

## Technical Context

**Language/Version**: Go 1.25.2
**Primary Dependencies**: `github.com/modelcontextprotocol/go-sdk` v1.6.1, `github.com/spf13/cobra` v1.10.2
**Storage**: YAML 配置文件（~/.fridaforge/mock_devices.yaml）用于模拟数据
**Testing**: `go test` (table-driven tests), 目标覆盖率 ≥ 80%
**Target Platform**: Linux (开发), macOS (CI), Windows (编译通过)
**Project Type**: CLI + MCP Server 子进程
**Performance Goals**: 启动 ≤ 1s (SC-001), 脚本生成 ≤ 2s (SC-002), 校验 ≤ 500ms (SC-003)
**Constraints**: 单 goroutine 顺序处理（stdio 串行管道）, stdout 仅 JSON-RPC, stderr 仅日志
**Scale/Scope**: 4 个 Tool, 1 个新 package, 1 个新 CLI 子命令, ~500 LOC new code

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| 宪法条款 | 状态 | 合规说明 |
|----------|------|---------|
| §2.1 代码格式 (gofmt, go vet, golangci-lint) | ✅ PASS | 所有新代码使用 gofmt, .golangci.yml 已存在 |
| §2.2 命名约定 (MixedCaps, 小写包名) | ✅ PASS | pkg/mcpserver 小写单数，导出符号使用 MixedCaps |
| §2.3 错误处理 (不忽略 error, fmt.Errorf wrap) | ✅ PASS | go-sdk handler 返回值含 error，不忽略 |
| §2.4 并发规范 (context.Context, WaitGroup) | ✅ PASS | 使用 context.Background() 传递，stdio 模式单 goroutine |
| §2.5 测试规范 (≥80% 覆盖率, table-driven) | ✅ PASS | 所有 Tool handler 可独立测试，模拟 transport |
| §2.6 文档规范 (Go doc comment) | ✅ PASS | 每个导出类型/函数写 doc comment |
| §3.x Frida 注入安全原则 | N/A | M4 不涉及真机 Hook，使用模拟数据 |
| §4.1 协议合规 (JSON-RPC 2.0) | ✅ PASS | go-sdk 内置 JSON-RPC 2.0 实现 |
| §4.2 Tool 定义规范 (name, description, inputSchema) | ✅ PASS | 泛型 AddTool 自动生成三要素 |
| §4.3 安全约束 (无网络监听, 无 eval) | ✅ PASS | stdio 模式无网络绑定，不暴露代码执行 |
| §4.4 性能要求 (<5s 响应) | ✅ PASS | 代码生成 <2s 已验证，校验 <500ms |
| §5.2 SpecKit 工作流 (spec → clarify → plan → tasks → analyze → implement) | ✅ PASS | 当前处于 plan 阶段 |
| §6.1 三轨并行学习制 | ✅ PASS | M4 涉及 Go (MCP SDK, interface), 逆向 (工具链整合), AI (MCP 协议理解) |
| §6.2 学习文档要求 | ⚠️ PLAN | 需在 implement 前产出 M4 教学文档初始版至 docs/learn/M4-*.md 和 .html |

**Gate Result**: ✅ ALL CHECKS PASSED — 可以进入 Phase 2 (tasks)。

**Constitution §4.3 特殊说明**: 宪法 §4.3 要求 "MCP Server 默认监听 localhost"，但 stdio transport 不使用网络连接（父子进程管道通信），天然比 localhost 更安全。此差异基于 spec 中用户明确选择 "stdio" 的决定。

## Project Structure

### Documentation (this feature)

```text
specs/004-mcpserver/
├── plan.md              # This file
├── research.md          # Phase 0: go-sdk API, stdio transport, slog, JSON Schema
├── data-model.md        # Phase 1a: MCPServer, ToolDefinition, I/O types, ProcessLister
├── quickstart.md        # Phase 1c: AI 助手配置指南
├── contracts/           # Phase 1b: Tool 输入/输出契约
│   └── mcp-tools-api.md
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
pkg/mcpserver/               # 新增 package
├── server.go                # NewMCPServer(), Run() — 组装 4 个 Tool 并启动
├── tools_spec.go            # spec_generate + spec_validate handler 实现
├── tools_device.go          # device_list + process_list handler 实现
├── types.go                 # I/O 类型定义 + ProcessLister 接口 + StubProcessLister
├── mock_store.go            # YAML 模拟数据加载
├── server_test.go           # MCPServer 集成测试 (InMemoryTransport)
├── tools_spec_test.go       # spec_generate / spec_validate 单元测试
├── tools_device_test.go     # device_list / process_list 单元测试
└── mock_store_test.go       # 模拟数据加载测试

cmd/fridaforge/mcp.go        # 新增子命令: `fridaforge mcp`
cmd/fridaforge/main.go       # 修改: rootCmd.AddCommand(mcpCmd)

go.mod                        # 修改: 新增 go-sdk 依赖
go.sum                        # 自动更新
```

**Structure Decision**: 遵循项目现有 Go 标准布局——`pkg/` 下建新 `mcpserver` package，`cmd/fridaforge/` 下新增一个命令文件。不引入新的顶层目录。`pkg/mcpserver` 通过接口依赖 `pkg/device.DeviceLister` 和 `pkg/codegen.Generator`，不反向依赖 fridaengine。

## Complexity Tracking

> 无 Constitution Check 违规，此节留空。
