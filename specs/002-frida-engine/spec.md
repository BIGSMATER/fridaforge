# Feature Specification: Frida 并发调度引擎

**Feature Branch**: `002-frida-engine`  
**Created**: 2026-05-12  
**Status**: Draft  
**Input**: User description: "构建 Frida 并发调度引擎 (fridaengine)，通过 frida-go 官方绑定实现设备枚举、进程 Attach、脚本注入和并发 Session 管理。替换 M1 的 StubDeviceLister 为真实 Frida 实现，引入 Go 并发模型 (goroutine/sync.WaitGroup/context/channel) 作为核心调度机制。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 枚举已连接的 Frida 设备 (Priority: P1)

安全研究者需要查看通过 USB 或网络连接的所有 Android 设备，以便选择目标进行 Hook 操作。

**Why this priority**: 设备发现是所有 Hook 操作的前提——没有设备列表就无法进行后续任何操作。这是引擎的入口能力。

**Independent Test**: 在一台连接了 Android 设备（模拟器或真机）的开发机上运行设备枚举，验证至少返回一台 Remote/USB 类型设备，不包含本地设备 (Local)。

**Acceptance Scenarios**:

1. **Given** 开发机连接了一台 Android 设备（模拟器或真机），**When** 调用设备枚举功能，**Then** 返回至少包含该设备 ID、名称和连接类型 (USB/Network) 的列表
2. **Given** 开发机没有连接任何 Android 设备，**When** 调用设备枚举功能，**Then** 返回空列表而不报错
3. **Given** 开发机有本地 frida-server (Local 设备)，**When** 调用设备枚举功能，**Then** Local 设备被过滤不返回

---

### User Story 2 - Attach 到目标进程并注入脚本 (Priority: P1)

安全研究者选择一台设备和一个目标应用后，需要 Attach 到该应用的进程并注入 Frida JavaScript 脚本，以拦截方法调用并观察运行时行为。

**Why this priority**: Attach + 脚本注入是 Frida Hook 的核心操作，是引擎存在的根本目的。

**Independent Test**: 对一台运行中的 Android 测试应用执行 Attach 并加载一段最简 Hook 脚本（如 `console.log("Hello FridaForge")`），验证收到脚本消息。

**Acceptance Scenarios**:

1. **Given** 目标应用正在运行，**When** 按包名 Attach 并注入脚本，**Then** 返回 HookSession 对象且状态为"已连接"
2. **Given** 目标应用未运行，**When** 按包名 Attach，**Then** 返回明确错误"进程未找到"
3. **Given** 已 Attach 的 Session，**When** 调用 Detach，**Then** Session 状态变为"已断开"且目标进程恢复正常运行

---

### User Story 3 - 并发管理多个 Hook 会话 (Priority: P2)

安全研究者需要同时对多个应用进行 Hook（例如：追踪跨进程调用链），引擎需支持并发管理多个 Session。

**Why this priority**: 多 Session 并发是引擎的核心架构能力，但 P1 场景（单 Session）即可交付最小可用版本。

**Independent Test**: 同时对 2 个不同应用 Attach，验证两个 Session 独立运行、独立收发消息，单独 Detach 一个不影响另一个。

**Acceptance Scenarios**:

1. **Given** 两个不同目标应用都在运行，**When** 同时对它们 Attach，**Then** 两个 Session 均成功建立且互不干扰
2. **Given** 多个活跃 Session，**When** 引擎收到关闭信号，**Then** 所有 Session 被顺序 Detach，不留下悬挂连接

---

### User Story 4 - 超时保护与异常恢复 (Priority: P2)

引擎必须对所有操作设置超时保护，防止因网络、设备无响应等原因导致无限等待。

**Why this priority**: 超时保护直接影响工具的可用性——没有超时机制，一次网络故障就可能卡死整个引擎。

**Independent Test**: 对一个不存在的设备发起 Attach，验证在 30 秒内返回超时错误而非无限挂起。

**Acceptance Scenarios**:

1. **Given** 设置了 30 秒 Attach 超时，**When** 超时尚未完成连接，**Then** 返回超时错误并释放资源
2. **Given** 一个正在运行的 Session，**When** 设备 USB 断开，**Then** Session 检测到断开并通过 channel 通知调用者

---

### User Story 5 - 枚举设备上运行的进程 (Priority: P3)

安全研究者需要查看目标设备上有哪些进程或应用正在运行，以确认 Hook 目标是否正在运行及其 PID。

**Why this priority**: 进程枚举增强了设备探索能力，但 Hook 操作可以通过包名 Attach（引擎自动解析 PID）而不需要手动枚举。

**Independent Test**: 对一台运行中的 Android 设备枚举进程，验证返回列表中包含 system_server 或 launcher 进程。

**Acceptance Scenarios**:

1. **Given** 一台已连接的 Android 设备，**When** 枚举其进程，**Then** 返回进程列表，每个进程包含名称和 PID
2. **Given** 一个已知包名正在运行，**When** 按名称查找进程，**Then** 精确返回该进程信息

---

### Edge Cases

- 设备在 Attach 过程中突然断开如何处理？（Session 通过 frida-go 的 `On("detached")` 信号感知并报告）
- 多个 goroutine 同时 Attach 同一进程是否允许？（允许——每个 Attach 创建独立 Session，frida 本身的限制是每进程可以多次 Attach）
- Hook 脚本语法错误时的处理？（frida-go 的 `script.Load()` 会返回错误，引擎应包装并返回给调用者）
- 设备列表为空时的 CLI 输出？（返回空列表无报错，CLI 显示友好提示信息）
- Session 不主动 Detach 退出程序时是否泄漏？（引擎注册 `Cleanup` 机制，利用 Go 的 `defer` 或信号捕获确保所有 Session 被 Detach）

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 引擎 MUST 通过 `frida.DeviceManager` 枚举已连接设备，过滤 Local 设备，仅返回 Remote (USB) 和 Network 类型设备
- **FR-002**: 引擎 MUST 实现 M1 定义的 `DeviceLister` 接口，提供 `ListDevices(ctx context.Context) ([]Device, error)` 方法
- **FR-003**: 引擎 MUST 支持按进程名或 PID Attach 目标进程，返回 `HookSession` 对象
- **FR-004**: 引擎 MUST 支持通过 `HookSession.CreateScript(jsSource)` 注入 Frida JavaScript 脚本
- **FR-005**: 引擎 MUST 通过 `context.Context` 管理所有 goroutine 生命周期，支持超时和取消
- **FR-006**: 每个 HookSession MUST 提供 `Detach()` 清理方法，确保支持 `defer` 调用模式
- **FR-007**: 引擎退出时 MUST 遍历所有活跃 Session 并逐一 Detach
- **FR-008**: Hook 消息 MUST 通过有缓冲 channel（容量 64）异步传递给调用者，防止 Hook 回调阻塞
- **FR-009**: 多 Session 并发 MUST 使用独立错误聚合策略：单个 Session 失败不影响其他 Session，最终返回聚合的所有错误
- **FR-010**: 所有错误 MUST 使用 `fmt.Errorf("...: %w", err)` 包装上下文
- **FR-011**: 引擎 MUST 定义三类错误：`DeviceError`（设备枚举/连接失败）、`SessionError`（Attach/Detach 失败）、`ScriptError`（脚本创建/加载失败），每类错误可包装底层 frida-go 错误
- **FR-012**: 默认 Attach 超时为 30 秒，默认 Session 生命周期无上限（由调用者 context 控制）
- **FR-013**: 引擎 MUST 支持枚举设备上的运行进程（`EnumerateProcesses`）和应用（`EnumerateApplications`）
- **FR-014**: Engine 和 SessionManager MUST 接受 `*slog.Logger` 注入用于结构化日志输出，不自行决定日志行为
- **FR-015**: HookSession MUST 通过 `State() SessionState` 方法暴露当前状态，使用 `sync.RWMutex` 保证并发安全
- **FR-016**: SessionManager MUST 维护并发会话软上限 64，超过时记录 `slog.Warn` 但不拒绝新 Attach

### Key Entities

- **Engine**: 引擎入口，持有 `DeviceLister`、`SessionManager` 和 `*slog.Logger`，提供设备发现和 Attach 的顶级 API
- **FridaDeviceLister**: 实现 `DeviceLister` 接口，包装 `frida.DeviceManager`，过滤设备类型
- **HookSession**: 包装 `frida.Session`，关联一个 context 和消息 channel，提供 Detach 清理能力。状态机：Created（Attach 成功后）→ Ready（脚本加载后，可接收消息）→ Detached（已断开）
- **HookScript**: 包装 `frida.Script`，管理脚本生命周期和消息回调
- **SessionManager**: 管理多个并发 HookSession，提供并发控制、错误聚合和统一清理

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 安全研究者能在 5 秒内完成设备枚举并看到已连接设备列表
- **SC-002**: 安全研究者能在 30 秒内完成一次完整的 Attach + 脚本加载流程（从设备选择到收到第一条 Hook 消息）
- **SC-003**: 引擎同时管理 5 个并发 Session 时，CPU 使用率不超过单核（5 为基准测量点，软上限为 64）
- **SC-004**: 引擎在收到退出信号后，2 秒内完成所有 Session 的 Detach 清理
- **SC-005**: Attach 超时后 500ms 内释放所有资源，不发生 goroutine 泄漏

## Assumptions

- 目标开发机已安装 frida-core-devkit（libfrida-core.a + frida-core.h），这是 frida-go 的编译依赖
- 目标 Android 设备已部署并运行 frida-server，且版本与 frida-core devkit 兼容
- M2 仅支持 Attach 已运行的进程，不包含 Spawn 模式（启动应用+Attach 延后到 M5）
- `FridaDeviceLister` 放在 `pkg/fridaengine/` 包内，与 `DeviceLister` 接口所在包 (`pkg/device/`) 分离
- 设备枚举仅返回 Remote (USB) 和 Network 类型设备，过滤 Local 设备
- M1 的 `StubDeviceLister` 保留仅用于测试
- Hook 消息通道缓冲区大小为 64，足以缓冲高频 Hook 回调

## Clarifications

### Session 2026-05-12

- Q: HookSession 应该有哪些显式状态？ → A: 三状态: Created → Ready（脚本加载后）→ Detached
- Q: 引擎应该定义哪几类错误？ → A: 三类: DeviceError + SessionError + ScriptError
- Q: 引擎包采用什么日志/可观测性策略？ → A: `*slog.Logger` 依赖注入
- Q: Session 状态应该如何暴露给调用者？ → A: `Session.State() SessionState` 方法 + RWMutex 保护
- Q: 引擎是否有硬性的并发 Session 数量上限？ → A: 软上限 64，超限 slog.Warn 但允许继续

## Dependencies

- 外部依赖: `github.com/frida/frida-go/frida` — Frida 官方 Go 绑定
- 编译依赖: `frida-core-devkit` (libfrida-core.a + frida-core.h)
- 运行依赖: `frida-server` 在目标 Android 设备上运行
- M1 组件: `pkg/device/` 的 `DeviceLister` 接口和 `Device` 结构体（复用，不修改）
