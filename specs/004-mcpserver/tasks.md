# Tasks: MCP Server 集成

**Input**: Design documents from `specs/004-mcpserver/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Spec SC-006 要求关键路径覆盖率 ≥ 80%，测试为必需项。

**Organization**: 按用户故事分阶段，同文件 Task 合并提交。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行（不同文件，无依赖）
- **[Story]**: 关联的用户故事（US1-US4）
- 包含精确文件路径

---

## Phase 1: 教学准备 + 环境搭建

**目标**: 宪法 §6.2 教学文档初始版（三轨并行），添加 MCP SDK 依赖，创建包目录

**教学步骤**:
1. 📖 讲 MCP 协议（JSON-RPC 2.0 + stdio Transport + Tool/Resource/Prompt 设计哲学）+ Go 知识（`encoding/json` 自定义序列化、`log/slog` 结构化日志、middleware 链模式）
2. ✍️ 创建 `docs/learn/M4-mcp-server.md` 初始版（独立迷你示例 10-20 行，三轨齐全）
3. ✍️ 同步创建 `docs/learn/M4-mcp-server.html`（同等地位，`<details>` 折叠 + `<aside>` 标注 + 打印样式）
4. 💻 搭建项目骨架

- [x] T001 [P] 创建 `docs/learn/M4-mcp-server.md` 教学文档初始版 — 宪法 §6.2: (1) Go 轨道: encoding/json jsonschema tag 自定义序列化、log/slog 结构化日志、接口注入模式; (2) 逆向轨道: M4 逆向知识轻量——串联 M1-M3 已学知识 (Frida 生命周期 + Hook 类型 + YAML Spec) 构建 AI 可调用工作流; (3) AI 轨道: MCP 协议设计哲学 (JSON-RPC 2.0 + Tool/Resource/Prompt 三原语、LLM 如何通过 Tool 描述理解并调用外部能力)
- [x] T002 [P] 创建 `docs/learn/M4-mcp-server.html` — 与 .md 同等地位，`<details>/<summary>` 折叠区域、`<aside>` 标注框、`<dl>` 定义列表、`@media print` 打印样式
- [x] T003 [P] 添加 `github.com/modelcontextprotocol/go-sdk` v1.6.1 到 go.mod（`go get github.com/modelcontextprotocol/go-sdk@v1.6.1`）
- [x] T004 [P] 创建 `pkg/mcpserver/` 目录

---

## Phase 2: 基础设施（阻塞所有用户故事）

**目标**: 核心类型、接口和 Server 骨架——所有用户故事的共享基础

**⚠️ CRITICAL**: 本阶段未完成前，不得启动任何用户故事

- [x] T005 [P] 创建 `pkg/mcpserver/types.go` — 定义 MCP I/O 类型（`GenerateInput`、`ValidateInput`、`ValidateOutput`、`ValidationFieldError`、`DeviceListOutput`、`DeviceListItem`、`ProcessListInput`、`ProcessListOutput`）、`ProcessLister` 接口、`StubProcessLister` stub 实现
- [x] T006 [P] 创建 `pkg/mcpserver/mock_store.go` — 定义 YAML mock 数据类型（`MockDeviceEntry`、`MockProcessEntry`）和加载器 `LoadMockStore()`，读取 `~/.fridaforge/mock_devices.yaml`，构造 `StubDeviceLister` + `StubProcessLister`
- [x] T007 创建 `pkg/mcpserver/server.go` — 实现 `NewMCPServer` 构造函数和 `Run()` 方法，注册 4 个 Tool（handler 先用 stub），启动 stdio transport，`slog` logger 写入 `os.Stderr`（依赖 T005、T006）
- [x] T008 [P] 创建 `pkg/mcpserver/mock_store_test.go` — table-driven 测试：类型序列化/反序列化校验 + YAML mock 数据加载

**Checkpoint**: 基础设施就绪——`NewMCPServer` 编译通过，4 个 Tool 已注册 stub handler。可开始用户故事实现。

---

## Phase 3: US1 & US2 — spec_generate + spec_validate（Priority: P1）🎯 MVP

**目标**: AI 可通过 MCP 生成合法 Frida Hook 脚本并校验 Hook 参数——M4 的核心价值交付

**独立测试**: 启动 `fridaforge mcp`，发送 `spec_generate` 请求（overload 类型，`com.example.Test.hello`），验证返回的 Frida JS 脚本语法正确。发送 `spec_validate` 分别传入合法/非法参数，验证 comprehensive 错误报告。

### US1 + US2 实现

- [x] T009 [US1] 在 `pkg/mcpserver/server.go` 中实现 `spec_generate` handler（作为 `Server` 方法，闭包注入依赖）：从 `GenerateInput` 组装 `spec.HookSpec` + `spec.HookTarget`，调用 `config.Validate()` 校验，调用 `codegen.Generator.Generate()` 生成脚本，传入 `input.FridaVersion`（空时默认 "16"），返回 `*mcp.CallToolResult`（`TextContent` 含 Combined 脚本）
- [x] T010 [US2] 在 `pkg/mcpserver/server.go` 中实现 `spec_validate` handler（作为 `Server` 方法）：从 `ValidateInput` 组装配置，调用 `config.Validate()`，将 `spec.ValidationError`/`spec.FieldError` 转换为 `ValidateOutput`/`ValidationFieldError`（comprehensive——一次返回所有错误）
- [x] T011 [US1] 创建 `pkg/mcpserver/tools_spec_test.go` — table-driven 测试 `spec_generate` handler（overload/override/native 三种类型、错误场景：nil spec、缺 module_name、无效 hook_type）
- [x] T012 [US2] 在 `pkg/mcpserver/tools_spec_test.go` 中补充 table-driven 测试 `spec_validate` handler（合法参数、空 class_name、空 app_package、native 缺 module_name、无效 hook_type、多条错误 comprehensive 返回、重复 Hook warning）

**Checkpoint**: US1 和 US2 功能完整——`spec_generate` 产出真实 Frida JS 脚本，`spec_validate` 返回 comprehensive 字段级错误。MVP 就绪。

---

## Phase 4: US3 & US4 — device_list + process_list（Priority: P2）

**目标**: AI 可枚举已连接设备和运行进程，支撑设备感知工作流

**独立测试**: 调用 `device_list`（无参数），验证返回结构化 JSON 含设备列表。调用 `process_list` 分别传入有效/无效设备 ID，验证进程列表或 device-not-found 错误。

### US3 + US4 实现

- [x] T013 [US3] 在 `pkg/mcpserver/server.go` 中实现 `device_list` handler（作为 `Server` 方法）：调用注入的 `DeviceLister.ListDevices(ctx)`，映射 `[]device.Device` → `[]DeviceListItem`，返回 `DeviceListOutput` 结构化 JSON
- [x] T014 [US4] 在 `pkg/mcpserver/server.go` 中实现 `process_list` handler（作为 `Server` 方法，含设备存在性校验）：先通过 `deviceLister` 校验设备是否存在，不存在返回错误
- [x] T015 [US3] 创建 `pkg/mcpserver/tools_device_test.go` — table-driven 测试 `device_list` handler（非空设备列表、空列表、lister 错误）
- [x] T016 [US4] 在 `pkg/mcpserver/tools_device_test.go` 中补充 table-driven 测试 `process_list` handler（有效设备返回进程列表、无效设备 ID 返回错误、lister 内部错误）

**Checkpoint**: 4 个 Tool 全部功能完整，可独立测试。

---

## Phase 5: CLI 集成

**目标**: 将 MCP Server 作为 `fridaforge mcp` 子命令暴露

- [x] T017 创建 `cmd/fridaforge/mcp.go` — 实现 `mcpCmd` cobra 命令：创建 `NewMCPServer`（使用默认 YAML mock store），调用 `server.Run(context.Background(), &mcp.StdioTransport{})`，启动错误通过 stderr 输出
- [x] T018 在 `cmd/fridaforge/main.go` 中注册 `mcpCmd`：`init()` 中添加 `rootCmd.AddCommand(mcpCmd)`

---

## Phase 6: 打磨与收尾

**目标**: 集成测试、lint 检查和 quickstart 验证

- [x] T019 [P] 创建 `pkg/mcpserver/server_test.go` — 集成测试使用 `InMemoryTransport`：验证 initialize 握手、tools/list 返回 4 个 Tool、每个 Tool 的 tools/call 合法/非法参数、错误传播；包含 disconnect 子测试：关闭 transport 后验证 server 正常退出（FR-008）
- [x] T020 [P] 运行 go vet 和 lint 检查
- [x] T021 运行 quickstart 验证：编译二进制，启动 `fridaforge mcp`，验证 stdout 无噪音，stderr 日志含启动信息，模拟 stdin initialize 请求
- [x] T022 [P] 在 `pkg/mcpserver/server_test.go` 中添加 FR-010 安全审查测试：审计所有 MCP handler 不暴露 eval/exec/文件系统写入等危险操作路径（宪法 §4.3）
- [x] T023 [P] 在 `pkg/mcpserver/tools_spec_test.go` 中添加 SC-002/SC-003 benchmark 测试：spec_generate 单 Hook 耗时 ≤2s，spec_validate 耗时 ≤500ms（`testing.B` 或计时断言）
- [x] T024 [P] 在 `pkg/mcpserver/server_test.go` 中添加 SC-001 benchmark 测试：NewMCPServer() + server.Run() 启动握手耗时 ≤1s
- [x] T025 运行 `go test -coverprofile=coverage.out ./pkg/mcpserver/... && go tool cover -func=coverage.out` 验证关键路径总覆盖率 ≥80%（SC-006 门禁）

---

## 依赖关系与执行顺序

### 阶段依赖

- **Phase 1 (Setup)**: 无依赖——可立即启动
- **Phase 2 (Foundational)**: 依赖 Phase 1 —— ⚠️ 阻塞所有用户故事
- **Phase 3 (US1+US2)**: 依赖 Phase 2 —— P1，核心 MVP
- **Phase 4 (US3+US4)**: 依赖 Phase 2 —— P2，可与 Phase 3 并行
- **Phase 5 (CLI)**: 依赖 Phase 3 + Phase 4（所有 handler 就绪后才能接入 CLI）
- **Phase 6 (Polish)**: 依赖所有前序阶段

### 用户故事依赖

- **US1 (spec_generate)**: 无跨故事依赖。复用 `pkg/codegen` + `pkg/config`（已存在）
- **US2 (spec_validate)**: 无跨故事依赖。复用 `pkg/config`（已存在）
- **US3 (device_list)**: 无跨故事依赖。复用 `pkg/device.DeviceLister`（已存在）
- **US4 (process_list)**: 无跨故事依赖。使用 Phase 2 新增的 `ProcessLister` 接口

Phase 2 完成后，四个用户故事各自独立。

### 阶段内顺序

- Phase 1: T001 ‖ T002 ‖ T003 ‖ T004（所有 [P]，不同文件/操作）
- Phase 2: T005 ‖ T006 → T007 → T008
- Phase 3: T009 + T010 同文件顺序实现 → T011 + T012
- Phase 4: T013 + T014 同文件顺序实现 → T015 + T016
- Phase 5: T017 → T018
- Phase 6: T019 ‖ T020 ‖ T021 ‖ T022 ‖ T023 ‖ T024（[P] 段）→ T025（覆盖率门禁）

---

## 并行机会

### Phase 1
```
T001 (teaching .md) ‖ T002 (teaching .html) ‖ T003 (go get) ‖ T004 (mkdir)
```

### Phase 2
```
T005 (types.go) ‖ T006 (mock_store.go)   →   T007 (server.go)   →   T008 (测试)
```

### Phase 3 (同文件，顺序提交)
```
T009 (spec_generate handler) + T010 (spec_validate handler)  →  T011 + T012 (测试)
```
T009 和 T010 共享 `pkg/mcpserver/tools_spec.go`——按 M1/M2 经验，同文件 Task 应合并 Commit。

### Phase 4 (同文件，顺序提交)
```
T013 (device_list handler) + T014 (process_list handler)  →  T015 + T016 (测试)
```

### 跨阶段并行（多开发者场景）
```
Phase 2 完成后:
  开发者 A: Phase 3 (T009→T010→T011→T012)   ‖
  开发者 B: Phase 4 (T013→T014→T015→T016)

Phase 3+4 完成后:
  开发者 A: Phase 5 (T017→T018)
  开发者 B: Phase 6 (T019 ‖ T020 ‖ T021 ‖ T022 ‖ T023 ‖ T024 → T025)
```

---

## 实施策略

### MVP 优先（仅 US1 + US2）

1. 完成 Phase 1: 环境搭建
2. 完成 Phase 2: 基础设施（⚠️ 阻塞所有故事）
3. 完成 Phase 3: US1 + US2（spec_generate + spec_validate）
4. **停止验证**: `go test ./pkg/mcpserver/... -v` + 手动 stdio 管道测试
5. Demo: `fridaforge mcp` 通过 stdio 接受 `initialize` + `tools/call`
6. ← 这是 M4 最小可交付里程碑

### 增量交付

1. Setup + Foundational → 基础设施就绪（Server 编译通过，所有 handler stub 已注册）
2. 加入 US1+US2 → MVP！（AI 可生成和校验 Hook 脚本）
3. 加入 US3+US4 → 完整功能（AI 可枚举设备和进程）
4. CLI 接入（`fridaforge mcp`）
5. 打磨（集成测试、lint、quickstart 烟雾测试）

---

## 注意事项

- [P] 标记 = 不同文件、无依赖。同文件 Task 按 M1/M2 经验牺牲 [P] 保持一致性。
- [Story] 标签将 Task 映射到具体用户故事，便于追踪。
- Phase 2 完成后每个用户故事可独立测试。
- 所有 handler 通过闭包接收 `*slog.Logger`；`pkg/mcpserver/` 内禁止 `fmt.Println`。
- `server.go` 注册全部 4 个 Tool，handler 实现以 `Server` 方法形式内聚在同一文件（与 tasks.md 原始设计中独立的 tools_spec.go/tools_device.go 不同——闭包依赖注入要求 handler 访问 `s *Server`，拆分文件会增加跨文件复杂度，合并后更内聚）。
- 按逻辑分组提交：Phase 2 合并为一个 Commit，然后每个故事单独 Commit，再 CLI，最后打磨。
- 每个 Checkpoint 处可停下来独立验证故事。
- 每个 Phase 完成后运行 `go test ./pkg/mcpserver/... -cover` 追踪覆盖率向 ≥80% 目标推进。
