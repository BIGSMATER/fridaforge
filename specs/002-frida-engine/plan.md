# Implementation Plan: Frida 并发调度引擎

**Branch**: `002-frida-engine` | **Date**: 2026-05-12 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/002-frida-engine/spec.md`

## Summary

构建 `pkg/fridaengine/` 包，通过 `frida-go` 官方 Go 绑定与 frida-core 通信，实现设备枚举、进程 Attach、脚本注入和并发 Session 管理。将 M1 的 `StubDeviceLister` 升级为真实 Frida 实现 (`FridaDeviceLister`)，引入 Go 并发模型 (goroutine, sync.WaitGroup, context.Context, sync.Mutex/RWMutex, channel) 作为核心调度机制。

## Technical Context

**Language/Version**: Go 1.25
**Primary Dependencies**: frida-go (`github.com/frida/frida-go/frida`), cobra, yaml.v3
**Storage**: N/A (无持久化存储)
**Testing**: Go 标准 `testing` + table-driven 测试 + `//go:build integration` tag 隔离真机测试
**Target Platform**: Linux, macOS (frida-go 支持 Windows，但 M2 优先 Linux/macOS)
**Project Type**: library (Go package) + CLI 集成
**Performance Goals**: 设备枚举 < 5s, Attach+脚本加载 < 30s, 5 并发 Session 单核 CPU
**Constraints**: 依赖 frida-core-devkit 编译; 需要 frida-server 运行在目标设备
**Scale/Scope**: 单一开发机, 最多 64 并发 Session (软上限), 1-5 台 Android 设备

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| 关卡 | 章节 | 状态 | 说明 |
|------|------|------|------|
| gofmt 格式化 | 2.1 | ✅ | Makefile 已包含 `gofmt -d` |
| go vet 零告警 | 2.1 | ✅ | CI/Makefile 已配置 |
| golangci-lint | 2.1 | ✅ | `.golangci.yml` 已配置 |
| 包命名: 小写、单数、无下划线 | 2.2 | ✅ | `fridaengine` (非 `frida_engine`) |
| 接口命名 -er 结尾 | 2.2 | ✅ | `DeviceLister` (已有); 新增函数风格命名 |
| 错误包装 %w | 2.3 | ✅ | 全部使用 `fmt.Errorf("...: %w", err)` |
| 禁止裸 panic | 2.3 | ✅ | 仅 `NewEngineWithDefaults` 可能用 `log.Fatal` |
| goroutine 通过 context 管理 | 2.4 | ✅ | 所有 goroutine 接受 context.Context |
| channel 明确定义方向 | 2.4 | ✅ | `chan<-` / `<-chan` |
| WaitGroup 代替 time.Sleep | 2.4 | ✅ | SessionManager 使用 WaitGroup |
| Mutex 保护共享状态 | 2.4 | ✅ | SessionManager.sessions + HookSession.state |
| table-driven 测试 | 2.5 | ✅ | 所有测试使用 table-driven 模式 |
| 覆盖率 ≥ 80% | 2.5 | ✅ | 目标 100% 导出函数覆盖 |
| Go doc comment | 2.6 | ✅ | 所有导出类型/函数含 Go doc |
| Cleanup 保证 (defer) | 3.2 | ✅ | Session.Detach() 支持 defer + 幂等 |
| Hook 回调 ≤ 100ms | 3.3 | ✅ | 消息通过有缓冲 channel 异步传递 |
| 伦理声明 | 3.4 | ✅ | M1 已实现，M2 不新增 CLI 命令 |
| Frida 操作通过统一管理器 | 3.1 | ✅ | Engine → SessionManager → HookSession |
| SpecKit 工作流 | 5.2 | ✅ | /speckit.specify → clarify → plan → tasks → analyze → implement |

**Re-check after Phase 1 design**: ✅ 所有关卡通过

## Project Structure

### Documentation (this feature)

```text
specs/002-frida-engine/
├── plan.md              # 本文件
├── research.md          # Phase 0 产出
├── data-model.md        # Phase 1 产出
├── quickstart.md        # Phase 1 产出
├── contracts/           # Phase 1 产出
│   └── engine-api.md    # Engine API 契约
├── checklists/
│   └── requirements.md  # Spec Quality Checklist
└── tasks.md             # Phase 2 产出 (未生成)
```

### Source Code (repository root)

```text
pkg/fridaengine/              # M2 新增包
├── engine.go                 # Engine 结构体 + 工厂函数
├── engine_test.go
├── device.go                 # FridaDeviceLister (实现 DeviceLister)
├── device_test.go
├── session.go                # HookSession + SessionState + HookMessage
├── session_test.go
├── script.go                 # HookScript 包装
├── manager.go                # SessionManager (并发调度)
├── manager_test.go
├── errors.go                 # DeviceError, SessionError, ScriptError
└── errors_test.go

pkg/device/                   # M1 已有 (不修改)
├── types.go                  # Device, ConnectType
├── manager.go                # DeviceLister 接口 + StubDeviceLister
└── *_test.go
```

**Structure Decision**: M1 的 `pkg/device/` 保持不变。M2 新增 `pkg/fridaengine/` 包实现真实 Frida 集成。两个包通过 `DeviceLister` 接口解耦：`fridaengine.FridaDeviceLister` 实现 `device.DeviceLister`，依赖方向正确 (fridaengine → device)。

## Complexity Tracking

无宪法违规需要申辩。所有适用关卡均通过。

| 说明项 | 处理 |
|--------|------|
| frida-go 引入 CGO 依赖 | 这是 frida 集成的本质要求，无法避免。通过 `//go:build integration` 隔离真机测试 |
| pkg/fridaengine 是纯新增 | 不修改 M1 任何现有文件，仅新增包 |
