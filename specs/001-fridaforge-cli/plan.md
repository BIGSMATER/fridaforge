# 实施计划: FridaForge CLI 命令行工具

**分支**: `001-fridaforge-cli` | **日期**: 2026-05-09 | **规格**: [spec.md](./spec.md)
**输入**: 功能规格来自 `specs/001-fridaforge-cli/spec.md`

**说明**: 本文件由 `/speckit.plan` 生成。

## 概述

使用 Go 1.25 构建 FridaForge CLI 骨架。cobra 做命令路由，viper 做配置绑定，yaml.v3 做 YAML 反序列化。CLI 暴露两个子命令：`fridaforge device list`（列出已连接的设备）和 `fridaforge spec validate <文件>`（校验 Hook 规格 YAML 文件）。M1 范围严格限定为纯本地操作：读文件、打印输出、返回退出码——不涉及网络、数据库或实际 Frida 连接。设备管理器基于接口设计，M1 使用桩实现。解析后的规格数据通过 `pkg/spec/` 以 Go 结构体形式暴露给下游消费。

## 技术上下文

**语言/版本**: Go 1.25
**主要依赖**: cobra (github.com/spf13/cobra), viper (github.com/spf13/viper), yaml.v3 (gopkg.in/yaml.v3)
**存储**: 无（仅文件 I/O）
**测试**: Go 标准库 `testing` + table-driven 测试（`*_test.go` 与源码同目录）
**目标平台**: Linux, macOS, Windows（单一二进制文件）
**项目类型**: CLI 工具
**性能目标**: 规格校验 < 2s，设备列表 < 5s（M1 使用桩/模拟实现，实际性能指标推迟到 M2）
**约束**: 纯本地 CLI——M1 无网络调用、无数据库、无实际 Frida 进程交互
**规模/范围**: 单用户、本地工作站；YAML 文件约 O(100) 条 Hook 目标

## 宪法检查

*关卡：Phase 0 研究前必须通过。Phase 1 设计后复查。*

| 关卡 | 章节 | 状态 | 说明 |
|------|------|------|------|
| gofmt 格式化 | 2.1 | ✅ 已计划 | CI 包含 `gofmt -d`；通过 Makefile target 强制执行 |
| go vet 零告警 | 2.1 | ✅ 已计划 | 包含在 CI/Makefile 中 |
| golangci-lint | 2.1 | ✅ 已计划 | 创建 `.golangci.yml`；CI 中运行 `golangci-lint run` |
| 包命名：小写、单数、无下划线 | 2.2 | ✅ 已计划 | `cmd/fridaforge`、`pkg/config`、`pkg/spec`、`pkg/device` |
| 导出命名：MixedCaps | 2.2 | ✅ 已计划 | `NewHookSpec`、`HookTarget`、`ValidateConfig` |
| 错误包装用 `%w` | 2.3 | ✅ 已计划 | 所有错误通过 `fmt.Errorf("...: %w", err)` 包装上下文 |
| 禁止裸 panic | 2.3 | ✅ 已计划 | 仅 `main.go` 中使用 `log.Fatal`；库代码返回 error |
| 并发：context.Context 生命周期 | 2.4 | ⏸️ 不适用 (M1) | M1 无 goroutine；推迟到 M2 |
| 测试：table-driven、`*_test.go` 同目录、覆盖率 ≥ 80% | 2.5 | ✅ 已计划 | 所有导出函数使用 table-driven 测试 |
| 所有导出项必须有 Go doc 注释 | 2.6 | ✅ 已计划 | 所有导出类型/函数必须含 Go doc 注释 |
| 首次运行时输出伦理声明 | 3.4 | ✅ 已计划 | 根命令 `PersistentPreRun` 钩子：检查标记文件，不存在则提示 |
| Frida 操作通过 FridaDeviceManager | 3.1 | ⏸️ 推迟 (M2) | M1 使用 `DeviceManager` 接口加桩；实际 Frida 集成在 M2 |
| SpecKit 工作流：不跳过 clarify/analyze | 5.2 | ✅ 已计划 | M1 遵循完整 SpecKit 周期 |
| 每个 Task 对应一个 Commit | 5.2 | ✅ 已计划 | tasks.md 每个完成的任务对应一个 Git commit |

**关卡结果**: ✅ 通过——全部适用关卡均满足。两个关卡推迟到 M2（并发、Frida 进程隔离），因为 M1 范围无 goroutine 也无实际 Frida 连接。

## 项目结构

### 文档（本功能）

```text
specs/001-fridaforge-cli/
├── plan.md              # 本文件
├── research.md          # Phase 0 产出
├── data-model.md        # Phase 1 产出
├── quickstart.md        # Phase 1 产出
├── contracts/           # Phase 1 产出 — CLI 与 YAML 契约
│   ├── cli-commands.md
│   └── yaml-schema.md
├── checklists/
│   └── requirements.md  # 来自 /speckit.specify
└── tasks.md             # Phase 2 产出 (/speckit.tasks — 此处不生成)
```

### 源代码（仓库根目录）

```text
cmd/
└── fridaforge/
    ├── main.go              # 根 cobra 命令、入口点、伦理声明
    ├── device.go            # `device list` 子命令
    └── spec.go              # `spec validate` 子命令
pkg/
├── config/
│   ├── loader.go            # YAML 文件加载 + viper 集成
│   ├── loader_test.go
│   ├── validator.go         # HookSpec 结构校验
│   └── validator_test.go
├── spec/
│   ├── types.go             # HookSpec、HookTarget、HookType 结构体 + yaml 标签
│   ├── types_test.go
│   ├── errors.go            # 校验错误类型
│   └── errors_test.go
└── device/
    ├── types.go             # Device 结构体
    ├── manager.go           # DeviceManager 接口 + 桩实现
    └── manager_test.go
```

**结构决策**: Go 标准项目布局。`cmd/` 包含唯一的 CLI 二进制入口点。`pkg/` 存放按领域组织的可复用库包：`config`（文件 I/O + 校验）、`spec`（数据模型）、`device`（设备发现抽象）。与 milestones.md 中 M1 交付物直接对应：`cmd/fridaforge/`、`pkg/config/`、`pkg/spec/`。

## 复杂度追踪

> 无宪法违规需要申辩。所有适用关卡均通过或明确推迟到后续里程碑。
