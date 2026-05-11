# 任务清单: FridaForge CLI 命令行工具

**输入**: `specs/001-fridaforge-cli/` 下的设计文档
**前置**: plan.md（必读）、spec.md（用户故事必读）、research.md、data-model.md、contracts/

**测试**: 根据宪法 2.5（覆盖率 ≥ 80%，table-driven）和用户要求（"单元测试使用 Go 标准库 testing + table-driven tests，测试文件与源码同目录"）包含测试任务。

**组织**: 按用户故事分组，支持各故事独立实现和独立测试。

## 格式: `[ID] [P?] [Story] 描述`

- **[P]**: 可并行执行（不同文件，无依赖关系）
- **[Story]**: 所属用户故事（如 US1、US2、US3）
- 描述中包含具体文件路径

## 路径约定

Go 标准项目布局（参见 plan.md）：
- `cmd/fridaforge/` — CLI 入口点（cobra 命令）
- `pkg/config/` — YAML 加载与校验
- `pkg/spec/` — 数据类型（HookSpec、HookTarget 等）
- `pkg/device/` — 设备管理器接口

---

## Phase 1: 环境搭建（共享基础设施）

**目标**: 项目初始化和基础工具配置

- [x] T001 创建目录结构：`cmd/fridaforge/`、`pkg/config/`、`pkg/spec/`、`pkg/device/`
- [x] T002 用 `go get` 向 go.mod 添加 Go 依赖（cobra、viper、yaml.v3）
- [x] T003 [P] 在仓库根目录创建 `.golangci.yml` lint 配置文件
- [x] T004 [P] 在仓库根目录创建 `Makefile`，包含 target：`build`、`test`、`lint`、`cover`

---

## Phase 2: 基础设施（阻塞性前置）

**目标**: 核心数据类型和 CLI 骨架——任何用户故事开始前必须完成

**⚠️ 关键**: 此阶段完成前，任何用户故事都不得开始

- [x] T005 在 `pkg/spec/types.go` 中创建 HookSpec、HookTarget、HookType 结构体（含 yaml 标签）
- [x] T006 [P] 在 `pkg/spec/errors.go` 中创建 ValidationError、FieldError 错误类型
- [x] T007 在 `cmd/fridaforge/main.go` 中创建根 cobra 命令（含 `--version` 标志）
- [x] T008 在 `cmd/fridaforge/main.go` 中实现伦理声明 PersistentPreRun（标记文件检查、AGREE 提示）

**检查点**: 基础设施就绪——数据类型和 CLI 骨架已存在。可以开始用户故事实现。

---

## Phase 3: 用户故事 1 — 校验 Hook 配置文件 (优先级: P1) 🎯 MVP

**目标**: `fridaforge spec validate <文件>` 校验 HookSpec YAML 文件，报告通过/失败并精确定位错误。

**独立测试**: 创建 `valid.yaml` 和 `invalid.yaml`，分别运行 `fridaforge spec validate`，验证正确的通过/失败反馈和错误信息（对照 contracts/cli-commands.md）。

**覆盖的 FR**: FR-002、FR-003、FR-003a、FR-004、FR-006、FR-007、FR-008、FR-009

### 用户故事 1 的测试 ⚠️

> **注意：先编写测试，确保测试 FAIL 后再实现**

- [x] T009 [P] [US1] 在 `pkg/spec/types_test.go` 中编写 HookSpec/HookTarget/HookType 的 table-driven 单元测试；同步编写 `pkg/spec/errors_test.go` 中 ValidationError/FieldError 的单元测试
- [x] T010 [P] [US1] 在 `pkg/config/validator_test.go` 中编写校验器的 table-driven 单元测试（覆盖全部校验规则和边界情况）
- [x] T011 [P] [US1] 在 `pkg/config/loader_test.go` 中编写加载器的 table-driven 单元测试（文件 I/O、YAML 解析、错误路径、大规模输入 ≥100 hooks 性能测试、非 UTF-8 编码处理）

### 用户故事 1 的实现

- [x] T012 [US1] 在 `pkg/config/loader.go` 中实现 YAML 文件加载函数 `LoadSpec(path string) (*spec.HookSpec, error)`——依赖 T005、T006
- [x] T013 [US1] 在 `pkg/config/validator.go` 中实现 HookSpec 校验函数 `Validate(spec *spec.HookSpec) error`（含结构化字段错误）——依赖 T005、T006
- [x] T014 [US1] 在 `cmd/fridaforge/spec.go` 中实现 `spec validate` cobra 子命令（接收文件参数，调用 loader，调用 validator，按 contracts/cli-commands.md 输出结果）——依赖 T007、T012、T013
- [x] T015 [US1] 在 `cmd/fridaforge/spec.go` 中实现 `spec` 父 cobra 命令（含 `--help`）——依赖 T007

**检查点**: `fridaforge spec validate <文件>` 完整可用。全部 US1 测试通过。

---

## Phase 4: 用户故事 2 — 列出连接的设备 (优先级: P2)

**目标**: `fridaforge device list` 列出已连接的设备（M1 使用桩 DeviceLister），显示设备 ID、名称和连接类型。

**独立测试**: 运行 `fridaforge device list`，桩 DeviceLister 返回预定义设备和空列表，验证输出对照 contracts/cli-commands.md。

**覆盖的 FR**: FR-001、FR-007、FR-008、FR-010

### 用户故事 2 的测试 ⚠️

> **注意：先编写测试，确保测试 FAIL 后再实现**

- [x] T016 [P] [US2] 在 `pkg/device/types_test.go` 中编写 Device 结构体的 table-driven 单元测试
- [x] T017 [P] [US2] 在 `pkg/device/manager_test.go` 中编写 StubDeviceLister 的 table-driven 单元测试

### 用户故事 2 的实现

- [x] T018 [US2] 在 `pkg/device/types.go` 中定义 Device 结构体——依赖 T005（共享模式）
- [x] T019 [US2] 在 `pkg/device/manager.go` 中定义 DeviceLister 接口 + StubDeviceLister 桩实现——依赖 T018
- [x] T020 [US2] 在 `cmd/fridaforge/device.go` 中实现 `device list` cobra 子命令（调用 DeviceLister.ListDevices，按 contracts/cli-commands.md 格式化输出）——依赖 T007、T019

**检查点**: `fridaforge device list` 完整可用（桩实现）。全部 US2 测试通过。

---

## Phase 5: 用户故事 3 — YAML 配置解析为结构化数据 (优先级: P3)

**目标**: `config.LoadSpec()` 成为干净的公共 Go API，为下游模块（M2/M3 的 codegen、engine）返回 `*spec.HookSpec`。验证解析准确性。

**独立测试**: 从测试代码调用 `config.LoadSpec("testdata/valid.yaml")`，验证返回的 `*spec.HookSpec` 字段值与文件内容一致。

**覆盖的 FR**: FR-005

### 用户故事 3 的测试 ⚠️

- [x] T021 [P] [US3] 在 `pkg/config/loader_test.go` 中编写 `config.LoadSpec` 集成测试——解析 YAML 并断言全部字段值

### 用户故事 3 的实现

- [x] T022 [US3] 在 `pkg/config/loader.go` 中将 `config.LoadSpec` 完善为导出的公共 API（含 Go doc 注释）——依赖 T012

**检查点**: `config.LoadSpec` 是已文档化、可测试的公共 API，可供下游消费。

---

## Phase 6: 收尾与横切关注点

**目标**: 质量关卡、文档和最终验证

- [x] T023 运行 `gofmt -d ./...`、`go vet ./...`、`golangci-lint run ./...`——修复全部告警
- [x] T024 运行 `go test -cover ./...` 并验证所有包覆盖率 ≥ 80%
- [x] T025 [P] 为 `pkg/` 下所有导出类型、函数和方法添加 Go doc 注释
- [x] T026 验证 quickstart.md 流程：`go build ./cmd/fridaforge/`、`./fridaforge --help`、`./fridaforge spec validate testdata/valid.yaml`、`./fridaforge device list`

---

## 依赖关系与执行顺序

### 阶段依赖

- **Phase 1（环境搭建）**: 无依赖——可立即开始
- **Phase 2（基础设施）**: 依赖 Phase 1 完成——阻塞所有用户故事
- **Phase 3（用户故事 1）**: 依赖 Phase 2（基础设施）
- **Phase 4（用户故事 2）**: 依赖 Phase 2（基础设施）——独立于 US1
- **Phase 5（用户故事 3）**: 依赖 US1（T012 LoadSpec）——US3 是 US1 加载器的完善
- **Phase 6（收尾）**: 依赖所有需要包含的用户故事完成

### 用户故事依赖

- **用户故事 1 (P1)**: 基础设施完成后即可开始——不依赖其他故事
- **用户故事 2 (P2)**: 基础设施完成后即可开始——独立于 US1，可并行
- **用户故事 3 (P3)**: 依赖 US1 的 LoadSpec 实现（T012）

### 各故事内部顺序

- 测试必须先编写且 FAIL，再实现
- 类型/结构体先于加载器/校验器
- 加载器/校验器先于 cobra 命令
- 故事完成后才能进入下一优先级

### 可并行机会

- Phase 1 中 T003 和 T004 可并行
- Phase 2 中 T005 和 T006 可并行
- US1 测试 T009、T010、T011 可全部并行
- US2 测试 T016、T017 可并行
- Phase 2 完成后，US1 和 US2 整个阶段可并行
- Phase 6 中 T025 可与 T023/T024 并行

---

## 并行示例: 用户故事 1 测试

```bash
# 同时启动全部 US1 测试（不同文件，无依赖关系）:
Task: "在 pkg/spec/types_test.go 中编写 HookSpec/HookTarget/HookType 的 table-driven 单元测试"
Task: "在 pkg/config/validator_test.go 中编写校验器的 table-driven 单元测试"
Task: "在 pkg/config/loader_test.go 中编写加载器的 table-driven 单元测试"
```

## 并行示例: 用户故事 1 + 用户故事 2

```bash
# 基础设施完成后，US1 和 US2 可并行推进:
# 开发者 A:
Task: "在 pkg/config/loader.go 中实现 YAML 文件加载函数"
Task: "在 pkg/config/validator.go 中实现 HookSpec 校验函数"

# 开发者 B（并行）:
Task: "在 pkg/device/types.go 中定义 Device 结构体"
Task: "在 pkg/device/manager.go 中定义 DeviceLister 接口 + 桩实现"
```

---

## 实施策略

### MVP 优先（仅用户故事 1）

1. 完成 Phase 1：环境搭建（T001-T004）
2. 完成 Phase 2：基础设施（T005-T008）
3. 完成 Phase 3：用户故事 1（T009-T015）
4. **停下来验证**: `fridaforge spec validate` 端到端可用
5. 演示：校验真实的 HookSpec YAML 文件

### 增量交付

1. 环境搭建 + 基础设施 → 基础就绪
2. 加上用户故事 1 → `spec validate` 可用 → 演示（MVP！）
3. 加上用户故事 2 → `device list` 可用 → 演示（扩展后的 CLI）
4. 加上用户故事 3 → `LoadSpec()` 公共 API 就绪 → 已准备好供 M2 使用
5. 收尾 → CI 就绪的质量标准

### 多人协作策略

多人开发时：
1. 团队一起完成环境搭建 + 基础设施
2. 基础设施完成后：
   - 开发者 A：用户故事 1（spec validate）
   - 开发者 B：用户故事 2（device list）
3. 开发者 A 接手用户故事 3（增强自己写的 LoadSpec）
4. 团队一起完成收尾

---

## 注意事项

- [P] 任务 = 不同文件，无依赖关系
- [Story] 标签将任务映射到特定用户故事，方便追溯
- 每个用户故事应可独立完成和独立测试
- 实现前先验证测试 FAIL（red-green-refactor）
- 每个任务或逻辑组完成后提交一次 commit（宪法 5.2）
- 可在任意检查点停下来独立验证故事
- M1 桩 DeviceLister 返回硬编码设备；真实 Frida 集成在 M2
- `go.mod` 已存在（Go 1.25.2）；T002 仅添加缺失的依赖
