# Phase 0 — Research: MCP Server 集成

**Feature**: 004-mcpserver | **Date**: 2026-06-02

---

## 1. modelcontextprotocol/go-sdk API

**版本**: v1.6.1（最新稳定版，发布于 2026-05-22）

### 核心类型

| 类型 | 用途 |
|------|------|
| `mcp.Server` | MCP 服务端入口，持有 Tool/Prompt/Resource 注册表 |
| `mcp.Tool` | 工具定义：`Name`, `Description`, `InputSchema` |
| `mcp.CallToolRequest` | 工具调用请求，含 `Params.Name`, `Params.Arguments` |
| `mcp.CallToolResult` | 工具调用返回值，含 `Content []Content`, `IsError`, `Meta` |
| `mcp.Implementation` | 服务身份标识：`Name`, `Version` |
| `mcp.StdioTransport` | stdio 传输层实现 |
| `mcp.CommandTransport` | 客户端侧子进程启动传输（AI 工具启动 FridaForge 时用） |

### 服务端创建模式

```go
server := mcp.NewServer(&mcp.Implementation{Name: "fridaforge", Version: "0.1.0"}, nil)
```

### 添加 Tool 的两种方式

**方式 A — 泛型 AddTool（推荐）**：自动从 struct tag 生成 JSON Schema，自动校验输入参数。

```go
type SpecGenerateInput struct {
    AppPackage string `json:"app_package" jsonschema_description:"目标应用包名"`
    ClassName  string `json:"class_name" jsonschema_description:"目标类全限定名"`
    // ...
}

mcp.AddTool(server, &mcp.Tool{
    Name:        "spec_generate",
    Description: "...",
}, func(ctx context.Context, req *mcp.CallToolRequest, input SpecGenerateInput) (
    *mcp.CallToolResult, SpecGenerateOutput, error,
) {
    return nil, output, nil
})
```

> **Decision**: 使用泛型 `mcp.AddTool`。每条 Tool 定义独立的 Input/Output struct，利用 `jsonschema_description` tag 提供参数描述。go-sdk 自动处理类型校验和 JSON Schema 生成。

**方式 B — server.AddTool + 原始 handler**：手动解析参数，自行生成 schema。仅在需要完全控制参数解析逻辑时使用。本功能不采用。

### Tool handler 返回值模式

```go
func handler(ctx, req, input) (*mcp.CallToolResult, Output, error)
```

- 第一个返回值 `*mcp.CallToolResult`: 额外元数据（可 nil）
- 第二个返回值 `Output`: 会序列化为结构化 JSON（如果非 nil）
- 第三个返回值 `error`: Go error → SDK 自动转为 MCP error

**内容返回策略**：对于脚本生成（spec_generate），返回 `&mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: script}}}` + nil output；对于结构化数据（device_list），返回 output struct。

### 错误处理

```go
result := &mcp.CallToolResult{}
result.SetError(fmt.Errorf("validation failed: %v", err))
return result, nil, nil
```

### 编译时依赖

```
github.com/modelcontextprotocol/go-sdk v1.6.1
└── github.com/google/jsonschema-go v1.2.2 (indirect)  // 自动引入
```

**go-sdk 自身依赖很少**，不会引入重量级框架。stdin/stdout transport 仅依赖 `os.Stdin`/`os.Stdout`。

---

## 2. stdio 传输模式

### go-sdk 中的实现

```go
// 服务端（FridaForge 进程内）
server.Run(context.Background(), &mcp.StdioTransport{})

// 客户端（AI 工具侧 — 由 opencode 等工具处理）
transport := &mcp.CommandTransport{Command: exec.Command("fridaforge", "mcp")}
session, err := client.Connect(ctx, transport, nil)
```

### 管道分工

```
stdin  ────→ FridaForge 进程（读取 JSON-RPC 请求）
stdout ←──── FridaForge 进程（输出 JSON-RPC 响应）
stderr ←──── FridaForge 进程（输出 slog 日志）← 应用日志通道
```

**关键约束**:
- `os.Stdout` 不得输出任何非 JSON-RPC 内容，否则协议解析失败
- 所有应用日志（slog）必须定向到 `os.Stderr`
- `fmt.Println` / `fmt.Printf` 在 stdio transport 模式中**禁止**使用

### 生命周期

```
AI 工具启动 fridaforge mcp → 子进程 fork
  → server.Run() 阻塞在 stdin 读取
  → 收到 initialize 请求 → 自动握手
  → 循环处理 tools/list, tools/call
  → stdin EOF（客户端退出）→ server.Run() 返回
  → 进程退出
```

go-sdk 的 `server.Run()` 自动完成 `initialize`/`initialized` 握手。

---

## 3. slog 结构化日志最佳实践

### Logger 配置

```go
import "log/slog"

func newMCPLogger() *slog.Logger {
    h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelInfo,
        AddSource: false,  // stdio 模式不需要调用栈信息
    })
    return slog.New(h)
}
```

### 日志级别使用约定

| Level | 用途 |
|-------|------|
| `Debug` | 参数详情、中间状态（生产关闭） |
| `Info` | 每次 tool 调用记录、会话建立/关闭 |
| `Warn` | 校验警告（非阻塞性）、非关键异常 |
| `Error` | 工具调用失败、不可恢复错误 |

### 工具调用日志格式（FR-007）

```go
logger.Info("tool called",
    "tool", toolName,
    "sessionID", sessionID,
    "params", paramsHash, // 不记录完整参数，避免日志膨胀
)
```

### go-sdk 与 slog 集成

go-sdk 提供 `mcp.NewLoggingHandler(serverSession, nil)` 将 MCP logging 能力与 slog 对接，允许客户端通过 `logging/setLevel` 动态调整日志级别。本功能暂不使用此特性（初期仅本地日志到 stderr），但架构预留接口。

---

## 4. JSON Schema 生成

### go-sdk 的自动生成机制

泛型 `mcp.AddTool` 利用 Go 1.18+ 泛型 + 反射，根据 Input/Output struct 的字段类型和 tag 自动生成 JSON Schema：

```go
type SpecValidateInput struct {
    AppPackage string       `json:"app_package" jsonschema_description:"目标应用包名（必填）"`
    ClassName  string       `json:"class_name" jsonschema_description:"目标类全限定名"`
    MethodName string       `json:"method_name" jsonschema_description:"目标方法名"`
    HookType   string       `json:"hook_type" jsonschema_description:"Hook 类型: overload/override/native"`
    // ...
}
```

**字段到 JSON Schema 的映射**:
- `string` → `{"type": "string"}`
- `int` → `{"type": "integer"}`
- `[]T` → `{"type": "array", "items": {...}}`
- `json:"name,omitempty"` → 不在 required 列表中
- `jsonschema_description:"..."` → `"description": "..."`
- 无 `omitempty` 的字段 → 自动加入 `required` 列表

### 不需要手动定义 schema

Tool 定义中 `InputSchema` 字段留 `nil`，SDK 从 handler 的泛型参数自动推导。go-sdk 在 tools/list 响应中自动填充 schema。

### 复杂嵌套类型

对于 HookTarget 列表等嵌套结构，直接用 `[]spec.HookTarget` 或定义中间类型，go-sdk 递归生成 schema。

> **Decision**: 使用 struct tag 驱动 JSON Schema 生成，不手写 schema。利用 go-sdk 自动校验和 schema 生成能力，减少维护成本。

---

## 5. 现有包复用分析

### 直接复用的包

| 包 | 复用内容 | 接口 |
|----|---------|------|
| `pkg/spec` | `HookSpec`, `HookTarget`, `HookType`, `ValidationError`, `FieldError` | 直接导入 |
| `pkg/config` | `LoadSpec()`, `Validate()` | 直接调用 |
| `pkg/codegen` | `Generator`, `NewGenerator()`, `Generate()` | 直接调用 |
| `pkg/device` | `DeviceLister` interface, `StubDeviceLister`, `Device` struct | 接口注入 |

### 需要适配的接口

```go
// DeviceLister 接口满足 MCP 设备枚举需求
type DeviceLister interface {
    ListDevices(ctx context.Context) ([]Device, error)
}
```

MCP 的 process_list 工具需要额外接口（当前 device 包无进程枚举接口）。有两种方案：

1. **新增 ProcessLister 接口**（推荐）：在 `pkg/mcpserver/` 内部定义小接口，初始用 YAML stub 实现
2. ~~复用 fridaengine.Engine.EnumerateProcesses()~~：违反"不依赖 fridaengine"约束

> **Decision**: 在 `pkg/mcpserver/types.go` 中定义 `ProcessLister` 接口，初始用 YAML 配置的 `StubProcessLister` 实现。真机对接时替换为 fridaengine 适配器。

---

## 6. 设计决策总结

| 决策点 | 选择 | 理由 |
|--------|------|------|
| MCP 库 | `modelcontextprotocol/go-sdk` v1.6.1 | 官方维护，泛型 AddTool 自动生成 schema，stdlib transport |
| Transport | `StdioTransport` | spec 明确要求 stdio，AI 工具作为父进程启动子进程 |
| Tool 注册 | 泛型 `mcp.AddTool` | 自动 schema 生成 + 参数校验，减少样板代码 |
| Schema 定义 | struct tag（`jsonschema_description`） | go-sdk 原生支持，无需手写 JSON Schema |
| 日志 | `slog.NewTextHandler(os.Stderr)` | 管道隔离——stdout=JSON-RPC，stderr=日志 |
| 设备枚举 | `DeviceLister` 接口（已有） | pkg/device 已定义，直接注入 StubDeviceLister |
| 进程枚举 | 新增 `ProcessLister` 接口 | pkg/device 暂无进程枚举抽象，需新建 |
| 模拟数据 | YAML 配置 + struct 定义 | 与 spec 一致，文件可编辑、可版本管理 |
| 错误报告 | comprehensive（所有字段错误一次返回） | 减少 AI 修正往返次数 |
| 并发模型 | 单 goroutine 顺序处理 | stdio transport 本质上是串行的（单管道），无需并发处理 |
