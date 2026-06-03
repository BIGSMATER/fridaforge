# Quickstart: 配置 AI 助手使用 FridaForge MCP

**Feature**: 004-mcpserver | **Date**: 2026-06-02

本文档指导用户将 FridaForge MCP 服务配置为 AI 编码助手（opencode / Claude Desktop 等）的外部工具，使 AI 能自动生成 Frida Hook 脚本。

---

## 前提条件

- FridaForge 已安装（`go install` 或源码构建）
- `fridaforge mcp` 命令可用
- AI 助手支持 MCP stdio transport（opencode v2+, Claude Desktop, VS Code Copilot 等）

---

## 1. 验证 FridaForge MCP 服务

```bash
# 构建项目
go build -o fridaforge ./cmd/fridaforge/

# 启动 MCP 服务（会阻塞等待 stdio 输入）
./fridaforge mcp
```

正常情况：进程启动后阻塞，无输出到 stdout（等待 JSON-RPC 握手）。

**检查 stderr 日志**：
```bash
./fridaforge mcp 2> /tmp/mcp.log &
# 在另一个终端
tail -f /tmp/mcp.log
# 应看到: "MCP server started" "waiting for client connection"
```

---

## 2. 配置模拟数据

编辑 `~/.fridaforge/mock_devices.yaml`（首次自动创建默认配置）：

```yaml
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

或使用命令行 flag 指定配置文件：
```bash
./fridaforge mcp --mock-config /path/to/custom_mock_devices.yaml
```

---

## 3. 配置 opencode

在 opencode 项目根目录的 `opencode.json`（或 `~/.config/opencode/opencode.json`）中添加 MCP 服务配置：

```json
{
  "mcpServers": {
    "fridaforge": {
      "command": "/absolute/path/to/fridaforge",
      "args": ["mcp"],
      "env": {}
    }
  }
}
```

**配置说明**:
- `command`: FridaForge 可执行文件的**绝对路径**（opencode 会 fork 子进程）
- `args`: 固定为 `["mcp"]`
- `env`: 可选环境变量（如 `MOCK_DEVICES_PATH`）

**重启 opencode**：配置修改后需要重启 AI 助手/重新加载窗口。

---

## 4. 配置 Claude Desktop

在 Claude Desktop 配置文件（`~/Library/Application Support/Claude/claude_desktop_config.json` macOS 或 `%APPDATA%\Claude\claude_desktop_config.json` Windows）中添加：

```json
{
  "mcpServers": {
    "fridaforge": {
      "command": "/absolute/path/to/fridaforge",
      "args": ["mcp"]
    }
  }
}
```

---

## 5. 验证集成

### 5.1 检查 MCP 服务是否被发现

在 opencode 中输入：
```
列出可用的工具
```

AI 助手应列出 FridaForge 提供的 4 个 Tool：
- `spec_generate` — 生成 Hook 脚本
- `spec_validate` — 校验 Hook 参数
- `device_list` — 枚举设备
- `process_list` — 枚举进程

### 5.2 试运行一个生成任务

在 opencode 中输入：
```
帮我生成一个 Hook 脚本，监控 com.example.test 应用的 com.example.MainActivity 类的 onCreate 方法，类型是 overload
```

AI 助手应：
1. 识别需求，调用 `spec_validate` 校验参数
2. 调用 `spec_generate` 生成脚本
3. 返回完整的 Frida JavaScript 代码

### 5.3 验证设备枚举

```
帮我看看现在有哪些设备可以调试
```

AI 助手应返回模拟数据中的设备列表。

---

## 6. 故障排查

### 6.1 fridaforge mcp 启动失败

**症状**: opencode 提示 MCP 服务连接失败。

**排查**:
```bash
# 手动验证命令可执行
/path/to/fridaforge mcp --help

# 检查是否输出帮助信息
```

### 6.2 JSON-RPC 通信异常

**症状**: AI 助手能看到工具但调用报错。

**排查**: 检查 stderr 日志：
```bash
/path/to/fridaforge mcp 2>&1 | tee /tmp/mcp_debug.log
```

常见原因：
- `os.Stdout` 被污染（应用层 println 输出到 stdout）
- 模拟配置文件格式错误（YAML 解析失败）

### 6.3 工具不可见

**症状**: AI 助手看不到 FridaForge 的 Tool。

**排查**:
1. 确认 `mcpServers` 配置在 AI 助手的配置文件中
2. 确认使用的是绝对路径
3. 重启 AI 助手

---

## 7. 架构概览

```
┌──────────────┐    stdio (fork + pipes)     ┌──────────────────┐
│ AI 助手       │ ←─────────────────────────→ │  fridaforge mcp   │
│ (opencode等)  │  stdin → JSON-RPC request  │  (子进程)         │
│              │  stdout ← JSON-RPC response │                  │
│              │                             │  stderr: slog日志  │
└──────────────┘                             └──────────────────┘
                                                      │
                                                      │ 复用
                                    ┌─────────────────┼─────────────────┐
                                    ▼                 ▼                  ▼
                              pkg/codegen       pkg/config         pkg/device
                              (脚本生成)         (参数校验)          (设备枚举)
```

---

## 8. 安全注意事项

- MCP 服务仅通过 stdio 通信，**不监听网络端口**
- 不暴露任意代码执行能力（无 eval tool）
- 所有 Tool 调用均记录到 stderr 日志
- Hook 脚本生成复用已有的代码生成逻辑，经模板渲染而非字符串拼接
