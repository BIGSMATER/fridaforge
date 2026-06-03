# MCP Tools API Contracts

**Feature**: 004-mcpserver | **Date**: 2026-06-02

本文档定义 FridaForge MCP Server 暴露的 4 个 Tool 的完整输入/输出契约，包括 JSON Schema、行为语义和错误模式。

---

## 通用约定

- **协议**: JSON-RPC 2.0 over stdio（stdin/stdout）
- **错误格式**: MCP standard error codes（由 go-sdk 自动处理）
- **类型校验**: 由 go-sdk 在 handler 调用前自动执行（基于 struct 类型推导）
- **业务校验**: 由 handler 内部执行（如 native hook 缺少 module_name）

---

## Tool 1: spec_generate

### 概述

根据 Hook 参数生成完整可执行的 Frida JavaScript Hook 脚本。复用 `pkg/codegen.Generator`。

### 输入 Schema

```json
{
  "type": "object",
  "properties": {
    "app_package": {
      "type": "string",
      "description": "目标应用包名，如 com.example.app（必填）"
    },
    "class_name": {
      "type": "string",
      "description": "目标类全限定名，如 com.example.MainActivity（必填）"
    },
    "method_name": {
      "type": "string",
      "description": "目标方法名，如 hello（必填）"
    },
    "hook_type": {
      "type": "string",
      "enum": ["overload", "override", "native"],
      "description": "Hook 类型：overload（前后插入）/ override（替换）/ native（Native层）（必填）"
    },
    "method_signature": {
      "type": "string",
      "description": "JNI 方法签名，如 (Ljava/lang/String;)V。Native 类型时建议提供以精确定位重载方法"
    },
    "module_name": {
      "type": "string",
      "description": ".so 模块名，如 libnative-lib.so。仅 native 类型时必填"
    }
  },
  "required": ["app_package", "class_name", "method_name", "hook_type"]
}
```

### 返回

**成功**: `CallToolResult` 含 `TextContent`，text 字段为完整 Frida JS 脚本。

```json
{
  "content": [
    {
      "type": "text",
      "text": "Java.perform(function() {\n  var target = Java.use(\"com.example.MainActivity\");\n  ...\n});\n"
    }
  ],
  "isError": false
}
```

**业务校验失败**: `isError: true` + 错误描述。

```json
{
  "content": [
    {
      "type": "text",
      "text": "校验失败: native Hook 必须提供 module_name"
    }
  ],
  "isError": true
}
```

### 行为语义

1. 将输入参数组装为 `spec.HookSpec{AppPackage, Hooks: []HookTarget{...}}`
2. 调用 `config.Validate()` 进行业务校验
3. 校验通过后调用 `codegen.NewGenerator(nil).Generate()`
4. 返回 `GenerateOutput.Combined` 文本内容

### 错误场景

| 场景 | 返回值 |
|------|--------|
| 缺少必填参数 | go-sdk 自动返回类型错误 |
| hook_type 非法值 | go-sdk 自动返回类型错误 |
| native 缺少 module_name | isError=true, 说明缺失字段 |
| 空 class_name | isError=true, 说明缺失字段 |
| 代码生成内部错误 | isError=true, 包含底层错误信息 |

---

## Tool 2: spec_validate

### 概述

校验单个 Hook 参数的结构合法性，一次性返回所有字段错误（comprehensive error）。复用 `pkg/config.Validate`。

### 输入 Schema

```json
{
  "type": "object",
  "properties": {
    "app_package": {
      "type": "string",
      "description": "目标应用包名（必填）"
    },
    "class_name": {
      "type": "string",
      "description": "目标类全限定名（overload/override 时必填）"
    },
    "method_name": {
      "type": "string",
      "description": "目标方法名（必填）"
    },
    "hook_type": {
      "type": "string",
      "enum": ["overload", "override", "native"],
      "description": "Hook 类型（必填）"
    },
    "method_signature": {
      "type": "string",
      "description": "JNI 方法签名（可选）"
    },
    "module_name": {
      "type": "string",
      "description": ".so 模块名（native 时必填）"
    }
  },
  "required": ["app_package", "method_name", "hook_type"]
}
```

### 返回 Schema（结构化 JSON via Output type）

```json
{
  "type": "object",
  "properties": {
    "valid": {
      "type": "boolean",
      "description": "校验是否通过（无 errors 时为 true）"
    },
    "errors": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "field": {"type": "string"},
          "message": {"type": "string"}
        },
        "required": ["field", "message"]
      }
    },
    "warnings": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "field": {"type": "string"},
          "message": {"type": "string"}
        }
      }
    }
  },
  "required": ["valid"]
}
```

### 行为语义

1. 将输入参数组装为 `spec.HookSpec`
2. 调用 `config.Validate()`
3. 将 `ValidationError.Errors` 和 `ValidationError.Warnings` 转换为 JSON 输出

### 示例交互

**请求（参数完整合法）**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "spec_validate",
    "arguments": {
      "app_package": "com.example.test",
      "class_name": "com.example.MainActivity",
      "method_name": "onCreate",
      "hook_type": "overload"
    }
  }
}
```

**响应**:
```json
{
  "valid": true,
  "errors": null,
  "warnings": null
}
```

**请求（多个字段错误）**:
```json
{
  "arguments": {
    "app_package": "",
    "class_name": "",
    "method_name": "test",
    "hook_type": "native"
  }
}
```

**响应（comprehensive — 所有错误一次返回）**:
```json
{
  "valid": false,
  "errors": [
    {"field": "app_package", "message": "不能为空"},
    {"field": "hooks[0].class_name", "message": "不能为空"},
    {"field": "hooks[0].module_name", "message": "不能为空（native Hook 需要 module_name）"}
  ],
  "warnings": null
}
```

### 错误场景

| 场景 | valid | errors 内容 |
|------|-------|-------------|
| 完全正确 | true | [] |
| 空 app_package | false | [app_package: 不能为空] |
| native 缺 module_name | false | [module_name: 不能为空] |
| 无效 hook_type | false | [hook_type: 不支持的值] |
| 空 method_name | false | [method_name: 不能为空] |
| 多个字段有误 | false | 所有字段错误均在列表中 |

---

## Tool 3: device_list

### 概述

枚举当前已连接的 Frida 调试设备列表。使用 `DeviceLister` 接口，初始注入 `StubDeviceLister`（基于 YAML 配置）。

### 输入

无参数（`any` 类型）。

### 返回 Schema（结构化 JSON via Output type）

```json
{
  "type": "object",
  "properties": {
    "devices": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": {"type": "string", "description": "设备唯一标识"},
          "name": {"type": "string", "description": "设备可读名称"},
          "connect_type": {
            "type": "string",
            "enum": ["usb", "network", "emulator"],
            "description": "连接方式"
          }
        },
        "required": ["id", "name", "connect_type"]
      }
    }
  },
  "required": ["devices"]
}
```

### 行为语义

1. 调用注入的 `DeviceLister.ListDevices(ctx)`
2. 将 `[]device.Device` 转换为 `[]DeviceListItem`
3. 返回结构化 JSON

### 示例交互

**请求**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "device_list",
    "arguments": {}
  }
}
```

**响应（模拟模式）**:
```json
{
  "devices": [
    {"id": "emulator-5554", "name": "Android Emulator 5554", "connect_type": "emulator"},
    {"id": "R5CT1234ABCD", "name": "Samsung Galaxy S21", "connect_type": "usb"}
  ]
}
```

**响应（空设备列表）**:
```json
{
  "devices": []
}
```

### 错误场景

| 场景 | 行为 |
|------|------|
| 正常（含空列表） | 返回 devices 数组（可能为空） |
| DeviceLister 内部错误 | isError=true, 包含错误描述 |

---

## Tool 4: process_list

### 概述

枚举指定设备上运行的应用进程列表。使用 `ProcessLister` 接口，初始注入 `StubProcessLister`（基于 YAML 配置）。

### 输入 Schema

```json
{
  "type": "object",
  "properties": {
    "device_id": {
      "type": "string",
      "description": "目标设备 ID，如 emulator-5554（必填）"
    }
  },
  "required": ["device_id"]
}
```

### 返回 Schema（结构化 JSON via Output type）

```json
{
  "type": "object",
  "properties": {
    "processes": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "pid": {"type": "integer", "description": "进程 ID"},
          "name": {"type": "string", "description": "进程名/应用包名"}
        },
        "required": ["pid", "name"]
      }
    }
  },
  "required": ["processes"]
}
```

### 行为语义

1. 调用注入的 `ProcessLister.ListProcesses(ctx, deviceID)`
2. 将结果转换为 `[]ProcessListItem`
3. 返回结构化 JSON

### 示例交互

**请求**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "process_list",
    "arguments": {
      "device_id": "emulator-5554"
    }
  }
}
```

**响应**:
```json
{
  "processes": [
    {"pid": 1234, "name": "com.example.test"},
    {"pid": 5678, "name": "com.android.settings"}
  ]
}
```

### 错误场景

| 场景 | 行为 |
|------|------|
| 正常 | 返回 processes 数组 |
| 设备 ID 不存在 | isError=true, "device not found: xxx" |
| ProcessLister 内部错误 | isError=true, 包含底层错误信息 |
| 缺少 device_id 参数 | go-sdk 自动返回类型错误 |

---

## 错误码参考

go-sdk 自动处理的 JSON-RPC 标准错误码：

| Code | 含义 | 触发场景 |
|------|------|---------|
| -32600 | Invalid Request | 非法 JSON |
| -32601 | Method not found | 调用未注册的 Tool |
| -32602 | Invalid params | go-sdk 类型校验失败 |
| -32603 | Internal error | handler 返回 error |

业务校验错误通过 `isError: true` + text content 返回，不使用 JSON-RPC error。
