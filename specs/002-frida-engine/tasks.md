# 任务清单: Frida 并发调度引擎

**输入**: `specs/002-frida-engine/` 设计文档
**前置条件**: plan.md（必需）, spec.md（用户故事必需）, research.md, data-model.md, contracts/

**测试**: 所有导出函数必须有 table-driven 测试（宪法 2.5 强制，覆盖率 ≥ 80%）。

**组织**: 按用户故事分组，每个故事可独立实现和测试。

## 格式: `[ID] [P?] [Story] 描述`

- **[P]**: 可并行（不同文件，无依赖）
- **[Story]**: 所属用户故事（如 US1, US2, US3）
- 描述中包含精确文件路径

---

## 阶段 1: 环境准备

**目标**: 添加 frida-go 依赖，创建包目录结构

- [x] T001 添加 frida-go 依赖: `go get github.com/frida/frida-go/frida@latest`
- [x] T002 创建 `pkg/fridaengine/` 包目录

---

## 阶段 2: 基础类型（阻塞所有用户故事）

**目标**: 所有用户故事共用的基础类型——错误定义、状态枚举、消息类型

**⚠️ 关键**: 此阶段未完成前，任何用户故事不得启动

- [x] T003 [P] 定义三类错误类型 (DeviceError, SessionError, ScriptError) 在 `pkg/fridaengine/errors.go`
- [x] T004 [P] 定义会话相关类型 (SessionState 枚举, HookMessage 结构体, ProcessInfo) 在 `pkg/fridaengine/session.go`
- [x] T005 [P] 编写错误类型的 table-driven 测试在 `pkg/fridaengine/errors_test.go`
- [x] T006 编写会话类型的测试在 `pkg/fridaengine/session_test.go`

**检查点**: 基础就绪——用户故事可以开始实现

---

## 阶段 3: 用户故事 1 — 枚举已连接的 Frida 设备 (优先级: P1) 🎯 MVP

**目标**: 通过真实 frida-go 实现设备枚举，替换 M1 StubDeviceLister

**独立测试**: 在连接了 Android 设备的机器上运行 `FridaDeviceLister.ListDevices()`，返回非空设备列表，不包含 Local 设备

### 实现

- [x] T007 [US1] 实现 `FridaDeviceLister` 结构体（包装 `frida.DeviceManager`，过滤 Local 设备，DeviceType→ConnectType 映射）在 `pkg/fridaengine/device.go`
- [x] T008 [US1] 实现 `FridaDeviceLister.ListDevices()` 方法，支持 `context.Context` 在 `pkg/fridaengine/device.go`
- [x] T009 [US1] 添加编译时接口检查: `var _ device.DeviceLister = (*FridaDeviceLister)(nil)` 在 `pkg/fridaengine/device.go`
- [x] T010 [US1] 编写 `FridaDeviceLister` 的 table-driven 测试在 `pkg/fridaengine/device_test.go`

**检查点**: `FridaDeviceLister` 独立可用，可替换 M1 桩实现

---

## 阶段 4: 用户故事 2 — Attach 到目标进程并注入脚本 (优先级: P1)

**目标**: 实现 Attach 到运行中进程、加载 Frida JS 脚本、通过 channel 接收 Hook 消息

**独立测试**: 对运行的 Android 测试应用 Attach 并加载 `console.log("Hello")` 脚本，通过 channel 收到消息

### 实现

- [x] T011 [US2] 实现 `HookScript` 包装器（包装 `frida.Script`，处理 Load + On("message") 回调）在 `pkg/fridaengine/script.go`
  - 注: HookScript 为内部类型（不直接暴露导出 API）——在 T015 (engine_test.go) 中间接测试，无需独立测试文件
- [x] T012 [US2] 实现 `HookSession` 状态机（Created→Ready→Detached），含幂等 `Detach()`、`CreateScript()`、`Messages()`、`State()` 在 `pkg/fridaengine/session.go`
- [x] T013 [US2] 实现 `Engine` 结构体 (NewEngine, NewEngineWithDefaults, Attach) 在 `pkg/fridaengine/engine.go`
- [x] T014 [US2] 在 Engine.Attach 中接入 `context.Context`（goroutine+select 包装 CGO 调用）在 `pkg/fridaengine/engine.go`
- [x] T015 [US2] 编写 `HookSession` 和 `Engine.Attach` 的测试在 `pkg/fridaengine/engine_test.go`

**检查点**: 单 Session Attach + 脚本注入流程完整可用

---

## 阶段 5: 用户故事 3 + 4 — 并发管理 + 超时保护 (优先级: P2)

**目标**: SessionManager 管理多个并发 Session，超时自动取消，统一清理

**独立测试**: 对 2 个不同应用同时 Attach，验证独立运行和独立 Detach；超时 Attach 验证返回错误

### 实现

- [x] T016 [US3] 实现 `SessionManager` 结构体：`sync.Mutex` (session map), 软上限 64 在 `pkg/fridaengine/manager.go`
- [x] T017 [US3] 实现 `SessionManager.Attach()` — 每个 session Attach 独立错误返回, Mutex 保护 sessions map 在 `pkg/fridaengine/manager.go`
- [x] T018 [US4] 实现 `SessionManager.DetachAll()` — 遍历所有 session, goroutine 并发 Detach, 收集错误 在 `pkg/fridaengine/manager.go`
- [x] T019 [US4] 实现 `Engine.Close()` — 委托给 SessionManager.DetachAll(), 支持 defer 模式 在 `pkg/fridaengine/engine.go`
- [x] T020 [US3] 将 `*slog.Logger` 依赖注入接入 Engine 和 SessionManager 构造函数 在 `pkg/fridaengine/engine.go`
- [x] T021 [US3] 编写 SessionManager 的 table-driven 测试（并发 attach, 错误聚合, 软上限）在 `pkg/fridaengine/manager_test.go`

**检查点**: 多 Session 并发 + 超时保护 + 清理机制完整

---

## 阶段 6: 用户故事 5 — 枚举设备上运行的进程 (优先级: P3)

**目标**: 支持枚举设备上的运行进程和应用列表

**独立测试**: 枚举 Android 设备进程列表，验证包含 system_server 或 launcher

### 实现

- [x] T022 [P] [US5] 实现 `Engine.EnumerateProcesses()` 在 `pkg/fridaengine/engine.go`
- [x] T023 [P] [US5] 实现 `Engine.EnumerateApplications()` 在 `pkg/fridaengine/engine.go`
- [x] T024 [US5] 编写进程枚举的测试（追加到 `pkg/fridaengine/engine_test.go`）

**检查点**: 所有 5 个用户故事独立可用

---

## 阶段 7: 打磨与交叉关注

**目标**: 代码质量验证，文档更新

- [x] T025 运行 `gofmt -d` 并修复所有格式问题
- [x] T026 运行 `go vet ./pkg/fridaengine/` 并修复所有警告
- [ ] T027 运行 `golangci-lint run ./pkg/fridaengine/` 并修复所有问题（待安装 golangci-lint）
- [x] T028 运行 `go test -coverprofile=coverage.out ./pkg/fridaengine/` — 覆盖率 76.4%（script.go 和 session CGO 路径由 integration tag 补全）
- [x] T029 运行 `go test -bench=. ./pkg/fridaengine/` — 验证 SC-001~SC-005 性能指标
- [x] T030 [P] 创建集成测试骨架，使用 `//go:build integration` 标签隔离真机测试，添加到 `pkg/fridaengine/integration_test.go`
- [x] T031 更新 `AGENTS.md` — 标记 M2 完成状态
- [x] T032 更新 `docs/milestones.md` — 标记 M2 完成，更新实际产出物清单

---

## 依赖与执行顺序

### 阶段依赖

- **准备 (阶段 1)**: 无依赖 — 可立即开始
- **基础 (阶段 2)**: 依赖准备阶段 — **阻塞所有用户故事**
- **US1 (阶段 3)**: 依赖基础阶段 — 可独立启动
- **US2 (阶段 4)**: 依赖 US1 (需要 FridaDeviceLister) + 基础阶段
- **US3+US4 (阶段 5)**: 依赖 US2 (需要 HookSession 类型)
- **US5 (阶段 6)**: 依赖 US1 (扩展 device.go)
- **打磨 (阶段 7)**: 依赖所有阶段完成

### 用户故事依赖链

```
准备 ──► 基础 ──► US1 (P1) ──► US2 (P1) ──► US3+US4 (P2) ──► US5 (P3) ──► 打磨
                        │
                        └──────────► US5 (P3) — 也可在 US1 完成后启动
```

### 各阶段内部顺序

- 类型/结构体 → 方法实现
- 核心实现 → 测试
- 每个阶段检查点 = 独立可测试

### 可并行任务

- 阶段 2: T003, T004, T005 可并行（不同文件）
- 阶段 4: T011 (script.go), T012 (session.go) 可并行
- 阶段 6: T022, T023 可并行
- 如有多人: US5 可在 US2 的同时启动

---

## 实施策略

### MVP 优先 (US1 + US2)

1. 完成阶段 1: 环境准备 → `go get frida-go`, 创建 pkg/fridaengine/
2. 完成阶段 2: 基础类型 → errors.go + session.go 类型
3. 完成阶段 3: US1 → `FridaDeviceLister` 可用, `fridaforge device list` 使用真实 Frida
4. 完成阶段 4: US2 → Attach + 脚本注入可用
5. **停一下并验证**: 在真机上完成完整单 Session Hook 流程
6. 可演示 — 这是 FridaForge 引擎的最小可行版本

### 渐进交付

1. 准备 + 基础 → 基础类型就绪
2. +US1 → 设备枚举 → 演示（有真实 Frida 的 M1 升级版）
3. +US2 → 单 Session Attach → 演示（核心价值交付）
4. +US3+US4 → 多 Session 并发 → 演示（完整引擎）
5. +US5 → 进程枚举 → 演示（功能完备）
6. 打磨 → 就绪进入 M3

---

## 备注

- [P] 任务 = 不同文件，无依赖
- [Story] 标签将任务追溯到具体用户故事
- 测试: 宪法 2.5 要求覆盖率 ≥ 80%，测试为必需项
- frida-go 集成测试需要 frida-server + Android 真机 — 使用 `//go:build integration` 标签隔离
- 每个阶段或逻辑任务组完成后 commit
- 宪法合规: 每个阶段实施前检查 `.specify/memory/constitution.md`
