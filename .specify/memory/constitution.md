# FridaForge — Project Constitution

> 本文档是 FridaForge 项目的最高治理文件。所有 Spec、Plan、Task 及代码实现必须服从本宪法约束。
> 若本宪法与新需求冲突，必须先修订宪法，再继续开发。

---

## 第一章：项目使命与核心价值观

### 1.1 使命陈述
FridaForge 是一个 **声明式 Frida 脚本工程化平台**。它允许安全研究者使用 YAML 文件声明 Hook 目标，
自动生成可执行的 Frida JavaScript 脚本，并管理多设备的并发 Hook 会话。

### 1.2 核心价值观
- **声明优于过程 (Declaration > Process):** 用户描述"要 Hook 什么"，平台决定"怎么 Hook"。
- **安全性第一 (Safety First):** 任何 Hook 操作不得导致目标应用崩溃或数据破坏，必须提供完整的回滚/清理机制。
- **可观测性 (Observability):** 所有 Hook 操作的输入、输出和副作用必须可追溯。
- **学做合一 (Learning by Building):** 本项目同时是 Go 语言和 Android 逆向的实战教学工具。

### 1.3 宪法权威
本宪法优先级高于所有其他项目文档。任何代码、计划或任务如与本宪法冲突，以本宪法为准。

---

## 第二章：Go 语言编码规范

### 2.1 代码格式
- **强制使用 `gofmt` 格式化所有代码。** CI 中必须包含 `gofmt -d` 检查，格式不符合标准则拒绝合并。
- 使用 `go vet` 进行静态分析，零 warning 要求。
- 使用 `golangci-lint` 作为 lint 工具，配置文件为 `.golangci.yml`。

### 2.2 命名约定
- **包名：** 全小写，单数，无下划线（符合 Go 惯例）。如 `fridaengine` 而非 `frida_engine`。
- **导出标识符：** 使用 `MixedCaps`（如 `NewSession`, `HookMethod`）。
- **非导出标识符：** 使用 `mixedCaps`（如 `parseTarget`, `buildScript`）。
- **接口命名：** 单方法接口以 `-er` 结尾（如 `Renderer`, `Executor`）。

### 2.3 错误处理
- 永远不要忽略 `error` 返回值。
- 使用 `fmt.Errorf("description: %w", err)` 包装上下文。
- 禁止裸 `panic`——仅供 `init()` 或不可恢复场景使用。
- 禁止 `_` 忽略 error，除非有明确注释说明原因。

### 2.4 并发规范
- 所有 goroutine 必须通过 `context.Context` 管理生命周期。
- channel 必须明确定义方向（`chan<-` 或 `<-chan`），禁止双向 channel 作为参数传递。
- 禁止在循环中使用 `time.Sleep` 等待 goroutine 完成，必须使用 `sync.WaitGroup` 或 channel。
- 共享状态优先使用 `sync.Mutex`，仅在有明确性能指标时才使用 lock-free 方案。

### 2.5 测试规范
- 每个可导出的 public 函数必须有对应的单元测试（覆盖率目标 ≥ 80%）。
- 测试文件命名：`*_test.go`。
- 使用 table-driven tests 作为首选测试模式。
- Frida 相关测试必须在 Android 模拟器/真机上验证（Harness Engineering）。

### 2.6 文档规范
- 每个可导出的类型/函数必须有 Go doc comment（以函数名开头）。
- 复杂模块必须包含 `README.md` 说明模块职责和使用示例。

---

## 第三章：Frida 注入安全原则

### 3.1 进程隔离（Process Isolation）
- Hook 操作必须通过 `FridaDeviceManager` 统一管理，禁止绕过管理器直接创建 Frida Session。
- 每个 Hook 会话必须有独立的 `context.Context` 超时控制（默认 30s）。

### 3.2 清理保证（Cleanup Guarantee）
- 所有 Hook 注册必须返回 `Cleanup` 函数。
- `Cleanup` 函数必须支持 `defer` 调用模式，确保即使在 panic 场景下也能执行。
- 平台退出时必须遍历所有活跃 Session，执行 `Session.Detach()`。

### 3.3 最小副作用（Minimal Side Effects）
- 禁止 Hook 修改目标进程的原始行为（除拦截回调外）。
- Hook 回调中禁止执行耗时超过 100ms 的操作。
- 禁止在 Hook 回调中再次调用被 Hook 的函数（防止无限递归）。

### 3.4 伦理边界（Ethical Boundaries）
- 本工具仅供授权安全测试使用。
- 所有 CLI 命令首次运行时必须输出 Ethical Disclaimer（伦理声明），要求用户键入 `AGREE` 确认。
- 禁止自动 Hook 非用户明确指定的进程。

---

## 第四章：MCP Server 交互标准

### 4.1 协议合规
- MCP Server 必须实现 JSON-RPC 2.0 规范。
- 必须支持标准 Request/Response/Notification 三种消息类型。
- Error Code 范围必须遵循 JSON-RPC 2.0 和 MCP 规范的约定。

### 4.2 Tool 定义规范
- 每个 Tool 必须有 `name`, `description`, `inputSchema` 三要素。
- `inputSchema` 必须使用 JSON Schema 格式。
- Tool 的 `description` 必须足够详细，让 LLM 能独立理解如何使用。

### 4.3 安全约束
- MCP Server 默认监听 `localhost` 仅（`127.0.0.1`），禁止绑定 `0.0.0.0`。
- 不得通过 MCP 暴露任意代码执行能力（如 `eval` Tool）。
- 所有 Tool 调用必须在日志中记录（时间、Tool 名、参数摘要）。

### 4.4 性能要求
- MCP Server 响应时间必须 < 5s（单一 Tool 调用）。
- 长时间运行的 Hook 操作应使用异步模式（返回 task_id，通过 Tool 查询状态）。

---

## 第五章：工程规范

### 5.1 项目管理
- 使用 Git 进行版本管理，托管至 GitHub。
- 分支策略：`main` (稳定) + `develop` (开发) + `feature/*` (功能分支)。
- Commit Message 格式：`type(scope): description` (遵循 Conventional Commits)。
  例如：`feat(cli): add device list command`

### 5.2 SpecKit 工作流约束
- 所有功能开发必须经过完整 SpecKit 流程：`/speckit.specify → clarify → plan → tasks → analyze → implement`。
- 禁止跳过 `clarify` 和 `analyze` 阶段直接进入 `implement`。
- 每完成一个 Task，Git Commit 一次（最小可回滚单元）。
- 每个 Milestone 完成后必须进行回顾，更新宪法（如需）。

### 5.3 版本管理
- 遵循 Semantic Versioning 2.0.0。
- `0.x.y` 阶段允许 Breaking Changes。
- `1.0.0` 后严格遵循兼容性承诺。

---

## 第六章：学习与教学规范

### 6.1 三轨并行学习制
本项目的每个 Milestone 必须同等覆盖三条学习线：
- **Go 语言轨道：** 语法特性 → 工程模式 → 并发模型（贯穿 M1-M7）
- **逆向/底层轨道：** Frida 原理 → Android 安全机制（集中在 M2/M3/M5/M6）
- **AI 编程轨道：** SpecKit 工作流 → 范式判断力（在 M0-M4 逐步内化）

禁止让 AI 代理直接生成最终答案——必须先交付"理解"，再交付"代码"。

### 6.2 学习文档要求
- 每个 Milestone 进入 `/speckit.implement` 前，必须先产出教学文档至 `docs/learn/M[x]-*.md`，必须包含：
  - 本阶段 Go 知识点（每个概念配独立可运行的迷你代码示例，而非项目正式代码）
  - 本阶段逆向/底层知识点
  - 本阶段 AI 编程认知突破
- 教学文档以代码驱动——用 10-20 行的独立 Go 示例讲解语法，学员看懂后再进入项目代码实现。
- 学员确认理解后，方可执行 `/speckit.implement`。
- SpecKit 各阶段的设计哲学详见独立文档 `docs/reference/speckit-rationale.md`。

---

> 本宪法由项目导师与学员共同维护。修订需经双方讨论并达成共识。

**Version**: 0.2.0 | **Ratified**: 2026-05-08 | **Last Amended**: 2026-05-09
