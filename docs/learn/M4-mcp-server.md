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
