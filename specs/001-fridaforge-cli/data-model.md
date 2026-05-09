# 数据模型: FridaForge CLI 命令行工具

**功能**: 001-fridaforge-cli
**日期**: 2026-05-09

## 核心实体

### HookSpec（Hook 规格文件）

YAML 规格文件的顶层表示。一个文件 = 一个目标应用。

| 字段 | 类型 | 必填 | YAML 键 | 描述 |
|------|------|------|---------|------|
| AppPackage | string | 是 | `app_package` | Android 应用包名（如 `com.example.app`） |
| Hooks | []HookTarget | 是（≥1） | `hooks` | 该应用的 Hook 目标列表 |

**校验规则**:
- `app_package` 不能为空
- `hooks` 必须包含至少一个条目
- 文件必须是合法的 UTF-8 编码 YAML

**Go 表示**:
```go
type HookSpec struct {
    AppPackage string       `yaml:"app_package"`
    Hooks      []HookTarget `yaml:"hooks"`
}
```

---

### HookTarget（Hook 目标）

单条 Hook 声明，针对特定类的特定方法。

| 字段 | 类型 | 必填 | YAML 键 | 描述 |
|------|------|------|---------|------|
| ClassName | string | 是 | `class_name` | Dalvik 类全限定名（如 `com.example.MainActivity`） |
| MethodName | string | 是 | `method_name` | 要 Hook 的方法名（如 `onCreate`）。M1 不包含参数签名。 |
| HookType | HookType | 是 | `hook_type` | Hook 类型：`overload` 或 `replace` |

**校验规则**:
- 三个字段均不能为空
- `hook_type` 必须是以下之一：`overload`、`replace`
- HookTarget 中的未知 YAML 键应产生警告
- 重复的 `class_name` + `method_name` 组合应产生警告

**Go 表示**:
```go
type HookTarget struct {
    ClassName  string   `yaml:"class_name"`
    MethodName string   `yaml:"method_name"`
    HookType   HookType `yaml:"hook_type"`
}

type HookType string

const (
    HookTypeOverload HookType = "overload"
    HookTypeReplace  HookType = "replace"
)
```

---

### Device（设备）

表示已连接的 Frida 可用设备。M1 中由桩 DeviceManager 返回。

| 字段 | 类型 | 描述 |
|------|------|------|
| ID | string | 设备唯一标识符 |
| Name | string | 人类可读的设备名称 |
| ConnectType | string | 连接类型：`"usb"`、`"network"`、`"emulator"` |

**Go 表示**:
```go
type Device struct {
    ID          string
    Name        string
    ConnectType string
}
```

---

### ValidationError（校验错误）

规格校验失败时返回的聚合校验结果。

| 字段 | 类型 | 描述 |
|------|------|------|
| Errors | []FieldError | 字段级校验错误列表 |

**Go 表示**:
```go
type ValidationError struct {
    Errors []FieldError
}

func (e *ValidationError) Error() string { /* 渲染所有字段错误 */ }

type FieldError struct {
    Path    string // 如 "hooks[0].class_name"
    Message string // 如 "不能为空"
    Line    int    // YAML 行号（如果有的话）
}
```

---

### YAML 结构（完整示例）

```yaml
# 单应用规格文件
app_package: com.example.targetapp
hooks:
  - class_name: com.example.MainActivity
    method_name: onCreate
    hook_type: overload

  - class_name: com.example.crypto.AES
    method_name: encrypt
    hook_type: replace
```

## 实体关系

```
HookSpec ──1:N──> HookTarget
Device（独立实体，M1 中与 HookSpec 无关）
ValidationError ──1:N──> FieldError（组合关系）
```

## 状态转换

M1 无可变状态转换。实体均为不可变值类型：
- `HookSpec` 和 `HookTarget` 从 YAML 一次性解析，视为只读。
- `Device` 实例由 `DeviceManager.ListDevices()` 返回，为瞬时快照。
