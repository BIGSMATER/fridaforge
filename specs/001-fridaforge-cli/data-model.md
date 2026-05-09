# Data Model: FridaForge CLI 命令行工具

**Feature**: 001-fridaforge-cli
**Date**: 2026-05-09

## Core Entities

### HookSpec

Top-level representation of a YAML spec file. One file = one target application.

| Field | Type | Required | YAML Key | Description |
|-------|------|----------|----------|-------------|
| AppPackage | string | yes | `app_package` | Android application package name (e.g. `com.example.app`) |
| Hooks | []HookTarget | yes (≥1) | `hooks` | List of hook targets for this application |

**Validation rules**:
- `app_package` must not be empty
- `hooks` must contain at least one entry
- File must be valid UTF-8 encoded YAML

**Go representation**:
```go
type HookSpec struct {
    AppPackage string       `yaml:"app_package"`
    Hooks      []HookTarget `yaml:"hooks"`
}
```

---

### HookTarget

A single hook declaration targeting a specific class method.

| Field | Type | Required | YAML Key | Description |
|-------|------|----------|----------|-------------|
| ClassName | string | yes | `class_name` | Fully qualified Dalvik class name (e.g. `com.example.MainActivity`) |
| MethodName | string | yes | `method_name` | Method name to hook (e.g. `onCreate`). M1 does not include parameter signatures. |
| HookType | HookType | yes | `hook_type` | Type of hook: `overload` or `replace` |

**Validation rules**:
- All three fields must be non-empty
- `hook_type` must be one of: `overload`, `replace`
- Unknown YAML keys in HookTarget should generate a warning
- Duplicate `class_name` + `method_name` combinations should generate a warning

**Go representation**:
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

### Device

Represents a connected Frida-capable device. In M1, returned by the stub DeviceManager.

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Unique device identifier |
| Name | string | Human-readable device name |
| ConnectType | string | Connection type: `"usb"`, `"network"`, `"emulator"` |

**Go representation**:
```go
type Device struct {
    ID          string
    Name        string
    ConnectType string
}
```

---

### ValidationError

Aggregated validation result returned when spec validation fails.

| Field | Type | Description |
|-------|------|-------------|
| Errors | []FieldError | List of field-level validation errors |

**Go representation**:
```go
type ValidationError struct {
    Errors []FieldError
}

func (e *ValidationError) Error() string { /* renders all field errors */ }

type FieldError struct {
    Path    string // e.g. "hooks[0].class_name"
    Message string // e.g. "must not be empty"
    Line    int    // YAML line number (if available)
}
```

---

### YAML Schema (Complete Example)

```yaml
# Single app spec file
app_package: com.example.targetapp
hooks:
  - class_name: com.example.MainActivity
    method_name: onCreate
    hook_type: overload

  - class_name: com.example.crypto.AES
    method_name: encrypt
    hook_type: replace
```

## Entity Relationships

```
HookSpec ──1:N──> HookTarget
Device (standalone, no relationship to HookSpec in M1)
ValidationError ──1:N──> FieldError (composition)
```

## State Transitions

M1 has no mutable state transitions. Entities are immutable value types:
- `HookSpec` and `HookTarget` are parsed once from YAML and treated as read-only.
- `Device` instances are returned from `DeviceManager.ListDevices()` and are ephemeral (reflect point-in-time snapshot).
