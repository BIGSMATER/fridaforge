# M4 学习笔记：MCP Server 集成

> Milestone: M4 | 状态: 已完成
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

依赖注入的核心思想：**把"对象需要的东西"从外部传进去，而不是在内部自己创建**。

```go
// ❌ 硬编码依赖：Server 自己决定用什么 DeviceLister
type Server struct {
    lister *StubDeviceLister  // 写死了具体类型
}
func NewServer() *Server {
    return &Server{lister: &StubDeviceLister{}}  // 内部创建
}

// ✅ 依赖注入：调用者决定传什么进来
type Server struct {
    lister DeviceLister  // 依赖接口，不依赖具体类型
}
func NewServer(lister DeviceLister) *Server {  // 参数传进来
    return &Server{lister: lister}
}
```

**三层价值——为什么用依赖注入？**

一是**方便测试**：换一个参数就切换了测试/生产模式，Server 内部代码一行不改。

```go
server := NewMCPServer(stubLister, ...)   // 测试时注入假数据
server := NewMCPServer(fridaLister, ...)  // 生产时注入真实 Frida
```

二是**隔离 CGO 污染**：`pkg/fridaengine` 依赖 CGO（需要 Android NDK），M4 用 interface 切断编译时依赖链——不 import fridaengine 就能编译运行。

三是**独立演进**：设备枚举的实现可以换成 ADB、Docker、模拟器——Server 完全不感知变化。

**Go 的隐式 interface——与 Java 的关键区别**

Go 不需要 `implements` 关键字。只要你有同名同签名的方法，就自动满足 interface：

```go
// pkg/device/manager.go —— 定义 interface（接口方）
type DeviceLister interface {
    ListDevices(ctx context.Context) ([]Device, error)
}

// pkg/fridaengine/device.go —— 实现 interface（实现方，零声明）
type FridaDeviceLister struct { ... }
func (f *FridaDeviceLister) ListDevices(ctx context.Context) ([]device.Device, error) { ... }
// 不需要写 "implements DeviceLister" — 签名一致即自动满足

// pkg/device/manager.go —— 另一个实现（同样零声明）
type StubDeviceLister struct { ... }
func (s *StubDeviceLister) ListDevices(ctx context.Context) ([]device.Device, error) { ... }
```

**M4 的注入全景**：

```
cmd/fridaforge/mcp.go —— 调用者负责创建依赖
  │
  ├── store := LoadMockStore("~/mock_devices.yaml")  ← 创建桩数据
  │
  └── NewMCPServer(
        store.DeviceLister,   ← 注入 DeviceLister interface
        store.ProcessLister,  ← 注入 ProcessLister interface
        slog.New(...),        ← 注入 slog.Logger
      )
```

Server 收到的都是 interface，不知道也不关心这些 interface 背后是真实 Frida 还是 YAML 文件。

**注意 `omitempty` tag**：标记此 tag 的字段为零值时不会序列化到 JSON，保证 Tool 的 inputSchema 不会要求用户填写可选字段——JSON Schema 的 `required` 列表仅含非 omitempty 字段。

### 1.4 go-sdk Tool 注册：泛型 + 反射协同模式

go-sdk 的 `mcp.AddTool` 同时使用了 Go 的两个高级特性：泛型（编译期类型安全）+ 反射（运行时自动生成 JSON Schema）。两者配合让 Tool 注册只需写 struct + handler，其余自动完成。

**泛型基础——同逻辑、不同类型，不复制代码**

```go
// ❌ 每个类型写一遍
func AddInt(a, b int) int           { return a + b }
func AddFloat(a, b float64) float64 { return a + b }

// ✅ 泛型写一次，类型参数化
func Add[T int | float64](a, b T) T { return a + b }
x := Add(1, 2)       // T 推断为 int
y := Add(1.5, 2.3)   // T 推断为 float64
```

`[T int | float64]` 叫类型参数列表。`T` 是占位符，`int | float64` 是类型约束——编译器只允许这两种类型。

**`AddTool[In, Out any]`——两个类型参数，编译期验证签名**

```go
// go-sdk 声明
func AddTool[In, Out any](s *Server, t *Tool, h func(ctx, *Request, In) (*Result, Out, error))
//                                                  │                 │
//                              handler 的第三个参数必须是 In ───────┘                 │
//                              handler 的返回必须是 Out ──────────────────────────────┘

// 调用时——编译器自动推断 In 和 Out：
mcp.AddTool(server, &mcp.Tool{...}, generateHandler)
//  generateHandler 签名: func(ctx, *Request, GenerateInput) (*Result, GenerateOutput, error)
//                                         ^^^^^^^^^^^^                      ^^^^^^^^^^^^^^
//  ⇒ 编译器推断 In = GenerateInput, Out = GenerateOutput

// ❌ 类型不匹配 → 编译错误，不会留到运行时
mcp.AddTool(server, tool, func(ctx, req, string) (...) {...}) // In 推断为 GenerateInput ≠ string
```

泛型把类型错误从**运行时 panic**提前到**编译期报错**。

**反射基础——运行时读取任意类型的结构**

go-sdk 在编译时不知道你定义了什么 struct，但需要在运行时自动生成 JSON Schema。反射就是"程序照镜子看自己"的能力：

```go
func inspectStruct(v any) {
    t := reflect.TypeOf(v)        // 拿"身份证"——类型描述符
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        field.Name                 // → "AppPackage"
        field.Type.String()        // → "string"
        field.Tag.Get("json")      // → "app_package"
        field.Tag.Get("jsonschema") // → "目标应用包名,required"
    }
}
```

`reflect.TypeOf()` 返回一个 `reflect.Type`，通过它可以遍历字段、读取名称、类型、tag——完全不知道具体 struct 是什么也能做到。

**泛型 + 反射的协作流程**

go-sdk 内部做的事情（简化版）：

```go
func AddTool[In, Out any](s *Server, t *Tool, h ToolHandlerFor[In, Out]) {
    // ──── 编译期 ────
    // 泛型保证 In 和 handler 第三个参数类型一致（编译通过 = 类型安全）

    // ──── 运行时 ────
    var in In                                 // In 是 GenerateInput（泛型已推断）
    tType := reflect.TypeOf(in)               // 反射拿身份证 → 知道有几个字段
    schema := map[string]any{"type": "object"}
    
    for i := 0; i < tType.NumField(); i++ {   // 遍历字段
        f := tType.Field(i)
        jsonName := f.Tag.Get("json")         // "app_package"
        desc := parseJSDescription(f.Tag.Get("jsonschema"))  // "目标应用包名", required
        schema["properties"][jsonName] = map[string]any{
            "type": goTypeToJSON(f.Type),     // string → "string"
            "description": desc,
        }
    }

    // 注册到 server，附上生成的 schema
    s.tools[t.Name] = &registeredTool{tool: t, schema: schema, handler: wrap(h)}
}
```

**分工**：泛型管编译期——签名校验、类型安全；反射管运行时——生成 JSON Schema、JSON 反序列化参数。两者配合让注册一个 Tool 只需要写 struct + handler，三行代码。

**项目实际代码——registerTools 中的调用**

```go
// pkg/mcpserver/server.go

func registerTools(server *mcp.Server, logger *slog.Logger) {
    mcp.AddTool(server, &mcp.Tool{
        Name:        "spec_generate",
        Description: "根据 Hook 参数生成完整的 Frida JavaScript 脚本",
}, generateHandler)  // 泛型从 handler 签名推断 In=GenerateInput, Out=GenerateOutput
                         // 反射从 GenerateInput 的 tag 自动生成 JSON Schema
}
```

### 1.5 闭包捕获依赖——Go 的轻量 DI

**问题**：go-sdk 的 `mcp.AddTool` 要求 handler 签名是固定的，不能加额外参数：

```
必须: func(ctx, req, input GenerateInput) (Result, GenerateOutput, error)
不能: func(ctx, req, input, extraArg1, extraArg2) (...)   ← 多一个都不行
```

但 handler 需要访问 `generator` 和 `logger`。怎么传进去？

**答案：闭包**。闭包 = 函数 + 它记住的外层变量。

从零开始理解：

```go
// 第 1 步：普通函数——只用参数
func add(a, b int) int { return a + b }

// 第 2 步：函数内部可以定义函数
func main() {
    x := 10
    inner := func() { fmt.Println(x) }  // inner 用了外层的 x
    inner()  // 输出 10
}

// 第 3 步：把内部函数返回出去——这就是闭包
func makePrinter(msg string) func() {
    return func() { fmt.Println(msg) }   // 记住了创建时的 msg
}

p1 := makePrinter("Hello")   // p1 记住 msg="Hello"
p2 := makePrinter("World")   // p2 记住 msg="World"
p1()  // "Hello"             ← 各记住各的
p2()  // "World"
```

**关键**：闭包记住的是变量本身（地址），不是当时的值：

```go
x := 10
f := func() { fmt.Println(x) }
x = 20
f()  // 输出 20 —— 不是 10。闭包看的是 x 的最新状态
```

**项目实际代码**——在 `registerTools` 中，每个 handler 都是一个闭包，捕获了外层 `s *Server`：

```go
// pkg/mcpserver/server.go

func (s *Server) registerTools(server *mcp.Server) {
//   ↑ s 在外层

    mcp.AddTool(server, &mcp.Tool{Name: "spec_generate", ...},
        // ──────────── 闭包开始 ────────────
        func(ctx context.Context, req *mcp.CallToolRequest, input GenerateInput) (
            *mcp.CallToolResult, GenerateOutput, error) {

            // s 不在参数列表里，但闭包能访问——因为闭包记住了外层作用域中的 s
            s.logger.Info("tool called", "tool", "spec_generate")
            output, err := s.generator.Generate(hookSpec, "16")

            return &mcp.CallToolResult{...}, GenerateOutput{Script: output.Combined}, nil
            // ──────────── 闭包结束 ────────────
        })
}
```

**三层依赖注入对比**：

| 方式 | 写法 | 适用场景 |
|------|------|----------|
| 全局变量 | `var gen *Generator` | 永远别用 |
| 构造函数参数 | `NewMCPServer(gen, logger, ...)` | Server 级别的依赖（整个 Server 共用的） |
| 闭包捕获 | `func(...) { s.gen.Generate() }` | Handler 级别的依赖（通过外层变量间接访问） |

M4 同时用了后两种：构造函数注入 Server 共用依赖，闭包让每个 handler 在固定签名下访问这些依赖。

**一句话**：闭包就是让函数"私自带行李"——go-sdk 不让你加参数（行李不让带上飞机），但闭包把依赖藏进了函数内部。

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

### 3.4 MCP 基础知识与开发

**MCP 是什么**：LLM 和外部工具之间的标准化通信协议。类比 HTTP 是浏览器 ↔ 服务器的协议，MCP 是 AI 助手 ↔ 外部工具的协议。它解决的根本问题：AI 的训练数据是离线快照，无法访问你的本地工具、文件、API——MCP 让 AI 有了"手"。

**三个原语**（MCP 提供的三种能力类型）：

| 原语 | 做什么 | 类比 | M4 实现？ |
|------|--------|---------|-----------|
| **Tool** | LLM 调用你写的函数 | REST POST | ✅ 4 个 |
| **Resource** | LLM 读取数据（文件、数据库） | REST GET | ❌ M4 不需要 |
| **Prompt** | 预定义的提示模板 | 快捷短语 | ❌ M4 不需要 |

Tool 的三要素：

```json
{
  "name": "spec_generate",                     // LLM 通过名字匹配
  "description": "生成 Frida Hook 脚本",        // LLM 通过描述理解用途
  "inputSchema": {                             // LLM 通过 Schema 构造参数
    "type": "object",
    "properties": {
      "class_name": {"type": "string", "description": "目标类名"}
    }
  }
}
```

LLM 看到 Tool 定义 → 名字匹配 → 理解用途 → 构造参数 → 发起调用——全程不需要人类写任何适配代码。

**协议生命周期（4 步）**：

```
Client                          Server
  │── initialize ────────────────►│  (1) 握手：协商版本+能力
  │◄── capabilities + serverInfo ─│
  │── notifications/initialized ──►│  (2) 就绪：客户端确认
  │── tools/call ────────────────►│  (3) 操作：调用 Tool
  │◄── result ────────────────────│
  │── stdin close ───────────────►│  (4) 关闭：优雅退出
```

**JSON-RPC 2.0 线格式**——MCP 的每行消息都符合此格式。四种类型，靠 `id` 匹配请求和响应：

```json
// 请求（有 id → 需要响应）
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"spec_generate",...}}
// 响应（id 与请求匹配）
{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"..."}]}}
// 错误（id 与请求匹配）
{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"invalid params"}}
// 通知（无 id → 不需要响应）
{"jsonrpc":"2.0","method":"notifications/initialized"}
```

`id` 是自增计数器。客户端发 1、2、3，服务端可以按任意顺序回复——靠 id 对应。通知没有 id，客户端不期待响应。

**传输层选择——为什么用 stdio**：

```
opencode 启动 fridaforge 作为子进程
  ├─ stdout ← JSON-RPC 协议消息（一行一条 JSON）
  └─ stderr ← 日志输出（slog）
```

```go
// Go 代码中启动 stdio
transport := &mcp.StdioTransport{}
server.Run(context.Background(), transport)
// 内部自动：读 stdin → 解析 JSON-RPC → 调 handler → 写 stdout
```

为什么不用 HTTP？本地子进程场景下 stdio 更简单：无端口冲突、无防火墙、无需序列化开销。AI 工具直接 fork 子进程就能通信。HTTP 适用于远程 MCP 服务器（共享服务）。

**用 go-sdk 开发一个 MCP Server——最小完整示例**：

```go
// 第 1 步：定义 I/O 类型（带 jsonschema tag，SDK 自动生成 JSON Schema）
type Input struct {
    Name string `json:"name" jsonschema:"要打招呼的人名,required"`
}
type Output struct {
    Greeting string `json:"greeting"`
}

// 第 2 步：实现 handler（普通 Go 函数，SDK 负责 JSON↔Go 转换）
func handler(ctx context.Context, req *mcp.CallToolRequest, input Input) (
    *mcp.CallToolResult, Output, error) {
    return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Hi " + input.Name}}},
        Output{Greeting: "Hi " + input.Name}, nil
}

// 第 3 步：创建 server
server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "1.0"}, nil)

// 第 4 步：注册 Tool（泛型自动推断 In/Out 类型）
mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "打招呼"}, handler)

// 第 5 步：启动 stdio 传输
server.Run(context.Background(), &mcp.StdioTransport{})
```

go-sdk 自动完成：JSON 反序列化参数 → 调 handler → Go 返回值序列化为 JSON → 写入 stdout。你只写 struct + handler。

**opencode 如何连接 MCP Server**——在 `opencode.jsonc` 配置：

```jsonc
{
  "mcp": {
    "fridaforge": {
      "type": "local",                           // 本地子进程
      "command": ["./fridaforge", "mcp"],         // 启动命令
      "enabled": true
    }
  }
}
```

opencode 启动时自动 fork `./fridaforge mcp`，通过 stdin/stdout 建立连接，然后调用 `tools/list` 获取 Tool 列表——全部自动。

### 3.5 opencode MCP 集成内部机制——从 LLM 决策到 Go 函数被调用

**第 1 步：opencode 连接 MCP 服务器**

```
opencode.jsonc
  ├─ fork fridaforge → tools/list → 
  │    注册表: {spec_generate→fridaforge, spec_validate→fridaforge, ...}
  ├─ fork sentry → tools/list →
  │    注册表: {sentry_search→sentry, sentry_get_issue→sentry, ...}
  └─ 连接 context7 → tools/list → ...
```

opencode 维护一个全局注册表：`map[toolName]mcpServerConnection`。启动时对所有配置的 MCP 服务器调 `tools/list`，建立这个映射。

**第 2 步：LLM 输出 function_call**

LLM 的输出不是 JSON，是结构化标记（不同模型格式不同）：

```json
// Anthropic Claude 格式
{"role":"assistant","content":[{"type":"tool_use","name":"spec_generate","input":{...}}]}

// OpenAI 格式
{"choices":[{"message":{"tool_calls":[{"function":{"name":"spec_generate","arguments":"{...}"}}]}}]}
```

**第 3 步：opencode 内部翻译——function_call → MCP 请求**

opencode 内部有一个翻译层（概念代码）：

```go
func handleLLMToolCall(tc LLMToolCall) string {
    // a. 查"这个 tool 属于哪个 MCP server"
    server := globalRegistry[tc.Name]  // spec_generate → fridaforge 连接

    // b. 构造 JSON-RPC 请求
    req := `{"jsonrpc":"2.0","id":4,"method":"tools/call",
             "params":{"name":"` + tc.Name + `","arguments":` + tc.Args + `}}`

    // c. 写入 MCP server 的 stdin
    server.StdinPipe.Write([]byte(req + "\n"))

    // d. 从 MCP server 的 stdout 读回响应
    resp := readLine(server.StdoutPipe)

    // e. 提取结果，返回给 LLM
    return resp.Result.Content[0].Text
}
```

关键：opencode 只是**管道工**——它不知道 Tool 内部做了什么，只管正确路由。把 LLM 的 function_call 翻译成 JSON-RPC，发给对应的 MCP server，把返回结果还给 LLM。

**第 4 步：Go 函数怎么变成 Tool 的——go-sdk 内部"装箱"**

`AddTool` 在运行时做了三层包装：

```go
// go-sdk 内部概念代码
func AddTool[In, Out any](server *Server, tool *Tool, handler ToolHandlerFor[In, Out]) {
    // ── 装箱 1：反射读 struct tag → JSON Schema ──
    var zero In  // 零值 GenerateInput{}
    schema := reflectSchema(reflect.TypeOf(zero))
    // AppPackage `jsonschema:"目标应用包名,required"` → {"type":"string","description":"目标应用包名"}

    // ── 装箱 2：JSON 反序列化 + 序列化 ──
    wrapped := func(ctx context.Context, req *jsonrpc.Request) (any, error) {
        var in In
        json.Unmarshal(req.Params.Arguments, &in)             // JSON → Go struct
        result, out, err := handler(ctx, &CallToolRequest{}, in) // 调用你的函数
        return serialize(result, out, err), nil                // Go → JSON
    }

    // ── 装箱 3：存入路由表 ──
    server.tools["spec_generate"] = &registeredTool{
        tool:    tool,       // name + description
        schema:  schema,     // JSON Schema
        handler: wrapped,    // 装箱后的 handler
    }
}
```

当 `tools/call {"name":"spec_generate","arguments":{...}}` 到达时：
```
stdin → JSON-RPC 路由 → 查 server.tools["spec_generate"]
  → wrappedHandler 执行
    → json.Unmarshal → generateHandler() → json.Marshal
      → 写入 stdout
```

`tools/list` 时 → 遍历 server.tools → 返回所有 Tool 的 name + description + schema。

**完整链路图**：

```
opencode           LLM               opencode            MCP协议            fridaforge
  │ "生成Hook"       │                  │                   │                   │
  │─────────────────►│                  │                   │                   │
  │                  │ LLM 推理:         │                   │                   │
  │                  │ 匹配 spec_generate│                   │                   │
  │                  │ function_call:    │                   │                   │
  │                  │ {name,args}       │                   │                   │
  │                  │──────────────────│                   │                   │
  │                  │                  │ 查注册表→fridaforge│                   │
  │                  │                  │ tools/call        │                   │
  │                  │                  │ {name,arguments}  │                   │
  │                  │                  │──────────────────►│                   │
  │                  │                  │                   │  查 server.tools   │
  │                  │                  │                   │  json.Unmarshal    │
  │                  │                  │                   │  generateHandler() │
  │                  │                  │                   │  json.Marshal      │
  │                  │                  │◄─────────────────│                   │
  │                  │  返回脚本内容      │                   │                   │
  │◄─────────────────│                  │                   │                   │
```

---

> 状态: 实现完成 — 全部 25/25 任务，覆盖率 88.4%。§3.4 新增 MCP 基础知识与开发，§3.5 新增 opencode 集成内部机制（发现 + 调用 + 函数变 Tool）。
