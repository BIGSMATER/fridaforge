# M4 学习笔记：MCP Server 集成

> Milestone: M4 | 状态: 实现中
> 三轨并行：Go 语言 / Android 逆向 / AI 编程范式

---

## 一、Go 语言轨道

### 1.1 `encoding/json` struct tag — 自定义序列化

Go 的 `encoding/json` 通过 struct tag 控制序列化行为。M4 中利用 `json` + `jsonschema` 双 tag 模式，让 MCP SDK 自动生成 Tool 的 inputSchema：

```go
type GenerateInput struct {
    AppPackage string `json:"app_package" jsonschema:"目标应用包名,required"`
    ClassName  string `json:"class_name" jsonschema:"目标类全限定名,required"`
}
```

tag 写在反引号内，key:value 用空格或逗号分隔。`json:"app_package"` 控制 JSON 字段名，`jsonschema:"描述,required"` 让 SDK 生成 JSON Schema 的 description 和 required。

### 1.2 `log/slog` 结构化日志

Go 1.21 引入的 `slog` 替代传统的 `log` 包，支持键值对结构化输出：

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
logger.Info("tool called", "tool", "spec_generate", "class", "com.example.Test")
// 输出: time=2026-06-02T10:00:00.000 level=INFO msg="tool called" tool=spec_generate class=com.example.Test
```

MCP stdio 的关键约束：**stdout 仅用于 JSON-RPC 协议消息，日志必须走 stderr**。`slog.NewTextHandler(os.Stderr, ...)` 确保日志不污染协议通道。

### 1.3 `interface` 注入模式 — 依赖解耦

Go 的 interface 实现是隐式的（不需要 `implements` 关键字）。通过构造函数注入接口实例实现依赖反转：

```go
type ProcessLister interface {
    ListProcesses(ctx context.Context, deviceID string) ([]ProcessListItem, error)
}

func NewMCPServer(deviceLister device.DeviceLister, processLister ProcessLister, logger *slog.Logger) *Server {
    return &Server{deviceLister: deviceLister, processLister: processLister, logger: logger}
}
```

M4 通过注入 `StubDeviceLister` + `StubProcessLister` 避免依赖 `pkg/fridaengine`（CGO 污染）。

---

## 二、逆向/底层轨道

### 2.1 M4 在 FridaForge 知织体系中的位置

M4 的逆向知识是横向串联，而非纵向深入。它把 M1-M3 的产出物组装成 AI 工具可调用的形式：

$$
\underbrace{\text{M1: YAML 配置解析}}_{\text{spec.HookSpec}} 
\quad+\quad 
\underbrace{\text{M2: Frida 引擎}}_{\text{DeviceLister interface}} 
\quad+\quad 
\underbrace{\text{M3: 代码生成器}}_{\text{codegen.Generator}} 
\quad=\quad 
\underbrace{\text{M4: MCP 服务}}_{\text{AI 可调用平台}}
$$

### 2.2 Frida 知织全链表

| Milestone | 知织单元 | M4 中使用方式 |
|-----------|----------|--------------|
| M1 | HookType (overload/override/native) | `spec_generate` Tool 按类型分发模板 |
| M1 | YAML HookSpec → Go struct | `spec_validate` Tool 校验参数 |
| M2 | DeviceLister interface | `device_list` Tool（Stub 模式） |
| M2 | Frida 生命周期 (enumerate→attach→inject) | 运行时知识储备，M4 通过 interface 解耦 |
| M3 | codegen.Generator.Generate() | `spec_generate` Tool 调用生成 JS 脚本 |
| M3 | 三种 Hook 模板 (.js.tmpl) | 生成器内部使用，M4 只调用接口 |

---

## 三、AI 编程轨道

### 3.1 MCP 协议设计哲学

MCP (Model Context Protocol) 是 LLM 与外部工具之间的标准化接口协议。三个核心原语：

| 原语 | 作用 | 类比 |
|------|------|------|
| **Tool** | LLM 可调用的函数 | REST POST 端点 |
| **Resource** | LLM 可读取的数据 | REST GET 端点 |
| **Prompt** | 预定义的提示模板 | 预设 prompt pattern |

M4 仅实现 **Tool** 原语——这是 FridaForge 的核心需求（让 LLM 调用功能），Resources 和 Prompts 推迟到后续 Milestone。

### 3.2 JSON-RPC 2.0 消息格式

MCP 的底层线格式是 JSON-RPC 2.0。四种消息类型：

```
请求:  {"jsonrpc":"2.0","id":1,"method":"tools/call","params":{...}}
响应:  {"jsonrpc":"2.0","id":1,"result":{...}}
错误:  {"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"..."}}
通知:  {"jsonrpc":"2.0","method":"notifications/initialized"}
```

关键区别：请求有 `id`（需要响应），通知无 `id`（单向）。这种设计让协议在同步请求-响应和异步通知之间清晰切换。

### 3.3 LLM 如何"理解" Tool

LLM 通过 Tool 的三个属性来理解如何调用：

1. **name** (`"spec_generate"`): 唯一标识符，LLM 用字符串匹配选择 Tool
2. **description** (`"生成 Frida Hook 脚本"`): 自然语言描述，LLM 判断何时使用
3. **inputSchema** (JSON Schema): 结构化参数定义，LLM 构造调用参数

一个设计良好的 Tool description 应该让 LLM 在 **zero-shot** 条件下（无示例）就能正确理解使用场景和参数含义。

---

> 状态: 初始版 — 实现阶段将补充项目实际代码示例
