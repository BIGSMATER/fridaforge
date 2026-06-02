# Phase 1a — Data Model: MCP Server 集成

**Feature**: 004-mcpserver | **Date**: 2026-06-02

---

## 实体概览

```
┌──────────────┐    1:N    ┌────────────────┐
│  MCPServer   │──────────→│ ToolDefinition │
│  (go-sdk)    │           │  (go-sdk Tool) │
└──────────────┘           └────────────────┘
       │
       │ 依赖
       ▼
┌──────────────┐    0:1    ┌────────────────┐
│ HookSpec     │←─────────│ CodeGenerator  │
│  (pkg/spec)  │  注入     │  (pkg/codegen) │
└──────────────┘           └────────────────┘
       │
       ▼
┌──────────────────┐
│ ValidationResult │  (pkg/spec.ValidationError)
└──────────────────┘
```

---

## 1. MCPServer（复用 go-sdk）

**类型**: `mcp.Server`（来自 `github.com/modelcontextprotocol/go-sdk/mcp`）

**职责**:
- 管理 MCP 协议生命周期（initialize → tools/list → tools/call → 断开）
- 注册 4 个 Tool handler
- 通过 `StdioTransport` 处理 JSON-RPC 通信

**构造函数**:
```go
server := mcp.NewServer(
    &mcp.Implementation{Name: "fridaforge", Version: "0.1.0"},
    nil,
)
```

**启动**:
```go
server.Run(ctx, &mcp.StdioTransport{})
```

**不需要自定义包装**：go-sdk 的 `mcp.Server` 已经封装了协议握手、错误处理、session 管理。FridaForge 只需注册 Tool handler 并调用 `Run()`。

---

## 2. ToolDefinition（go-sdk 原生）

**类型**: `mcp.Tool`

**字段与约束**:

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `Name` | `string` | 必填，唯一，snake_case | Tool 标识。例: `spec_generate` |
| `Description` | `string` | 必填，中文描述+示例 | AI 仅凭此字段理解用法。需足够详细 |
| `InputSchema` | `*jsonschema.Schema` | nil（auto-generated） | 由 go-sdk 从泛型 handler 参数自动生成 |

**4 个 Tool 定义**:

| Name | Description | Handler Input | Handler Output |
|------|-------------|---------------|----------------|
| `spec_generate` | 根据 Hook 参数生成完整 Frida JavaScript 脚本 | `GenerateInput` | `*mcp.CallToolResult` (TextContent) |
| `spec_validate` | 校验 Hook 参数结构合法性，返回所有字段错误 | `ValidateInput` | `ValidateOutput` (结构化 JSON) |
| `device_list` | 枚举当前已连接/可用的 Frida 调试设备 | 无参数 | `DeviceListOutput` |
| `process_list` | 枚举指定设备上的运行进程列表 | `ProcessListInput` | `ProcessListOutput` |

---

## 3. HookSpec（复用 pkg/spec）

**来源**: `github.com/bigsmater/fridaforge/pkg/spec`

**复用完整性**: ✅ 所有字段均适用

```
HookSpec
├── AppPackage  string         // 目标应用包名
└── Hooks       []HookTarget   // Hook 目标列表
```

```
HookTarget
├── ClassName       string    // 全限定类名
├── MethodName      string    // 目标方法名
├── HookType        HookType  // overload|override|native
├── MethodSignature string    // JNI 方法签名（可选）
└── ModuleName      string    // .so 模块名（native 必填）
```

**MCP 工具中复用方式**:
- `spec_validate`: 将 MCP 输入参数组装为 `HookSpec` + `HookTarget`，调用 `config.Validate()`
- `spec_generate`: 将 MCP 输入参数组装为 `HookSpec`，调用 `codegen.Generator.Generate()`

---

## 4. ValidationResult（复用 pkg/spec）

**来源**: `github.com/bigsmater/fridaforge/pkg/spec`

```
ValidationError
├── Errors   []FieldError     // 校验错误列表
└── Warnings []FieldError     // 校验警告列表

FieldError
├── Path    string  // 字段路径，如 "hooks[0].class_name"
├── Message string  // 错误描述
└── Line    int     // YAML 行号（MCP 场景恒为 0）
```

**MCP 适配**: `config.Validate()` 返回 `*spec.ValidationError`，MCP handler 直接将其转换为 Tool 响应结构。

---

## 5. MCP 工具输入/输出类型（新增）

### 5.1 spec_generate

```go
// GenerateInput 定义 spec_generate Tool 的输入参数。
type GenerateInput struct {
    AppPackage      string `json:"app_package" jsonschema_description:"目标应用包名，如 com.example.app（必填）"`
    ClassName       string `json:"class_name" jsonschema_description:"目标类全限定名，如 com.example.MainActivity（必填）"`
    MethodName      string `json:"method_name" jsonschema_description:"目标方法名，如 hello（必填）"`
    HookType        string `json:"hook_type" jsonschema_description:"Hook 类型：overload（前后插入）/ override（替换）/ native（Native层）（必填）"`
    MethodSignature string `json:"method_signature,omitempty" jsonschema_description:"JNI 方法签名，如 (Ljava/lang/String;)V。Native 类型时建议提供以精确定位重载方法"`
    ModuleName      string `json:"module_name,omitempty" jsonschema_description:".so 模块名，如 libnative-lib.so。仅 native 类型时必填"`
}
```

**返回值**: `*mcp.CallToolResult` 含 `TextContent` 文本内容（完整的 Frida JS 脚本）。

### 5.2 spec_validate

```go
// ValidateInput 定义 spec_validate Tool 的输入参数。
type ValidateInput struct {
    AppPackage      string `json:"app_package" jsonschema_description:"目标应用包名（必填）"`
    ClassName       string `json:"class_name" jsonschema_description:"目标类全限定名（overload/override 时必填）"`
    MethodName      string `json:"method_name" jsonschema_description:"目标方法名（必填）"`
    HookType        string `json:"hook_type" jsonschema_description:"Hook 类型：overload/override/native（必填）"`
    MethodSignature string `json:"method_signature,omitempty" jsonschema_description:"JNI 方法签名（可选）"`
    ModuleName      string `json:"module_name,omitempty" jsonschema_description:".so 模块名（native 时必填）"`
}

// ValidateOutput 定义 spec_validate Tool 的结构化返回值。
type ValidateOutput struct {
    Valid    bool                    `json:"valid"`
    Errors   []ValidationFieldError  `json:"errors,omitempty"`
    Warnings []ValidationFieldError  `json:"warnings,omitempty"`
}

// ValidationFieldError 表示单个字段的校验结果。
type ValidationFieldError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}
```

### 5.3 device_list

```go
// DeviceListOutput 定义 device_list Tool 的结构化返回值。
type DeviceListOutput struct {
    Devices []DeviceListItem `json:"devices"`
}

// DeviceListItem 表示一个设备条目。
type DeviceListItem struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    ConnectType string `json:"connect_type"` // "usb" | "network" | "emulator"
}
```

**输入**: 无参数（空 struct `any` 或专用空 struct）。

### 5.4 process_list

```go
// ProcessListInput 定义 process_list Tool 的输入参数。
type ProcessListInput struct {
    DeviceID string `json:"device_id" jsonschema_description:"目标设备 ID，如 emulator-5554（必填）"`
}

// ProcessListOutput 定义 process_list Tool 的结构化返回值。
type ProcessListOutput struct {
    Processes []ProcessListItem `json:"processes"`
}

// ProcessListItem 表示一个进程条目。
type ProcessListItem struct {
    PID  int    `json:"pid"`
    Name string `json:"name"`
}
```

---

## 6. 模拟数据模型（YAML 配置）

**文件位置**: `~/.fridaforge/mock_devices.yaml`（默认）或通过 `--mock-config` flag 指定。

```yaml
# mock_devices.yaml — 模拟设备与进程数据
devices:
  - id: "emulator-5554"
    name: "Android Emulator 5554"
    connect_type: "emulator"
    processes:
      - pid: 1234
        name: "com.example.test"
      - pid: 5678
        name: "com.android.settings"
  - id: "R5CT1234ABCD"
    name: "Samsung Galaxy S21"
    connect_type: "usb"
    processes:
      - pid: 1001
        name: "com.example.bank"
```

**加载逻辑**: MCP 服务启动时读取 YAML，构建 `StubDeviceLister` 和 `StubProcessLister` 的内存数据。

---

## 7. ProcessLister 接口（新增）

```go
// ProcessLister 定义进程枚举接口。
// M4 使用 YAML 配置的桩实现，后续里程碑对接真实 Frida。
type ProcessLister interface {
    ListProcesses(ctx context.Context, deviceID string) ([]ProcessListItem, error)
}
```

**编译时检查**: `var _ ProcessLister = (*StubProcessLister)(nil)`（类似 pkg/device 中的模式）
