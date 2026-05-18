# Implementation Plan: 声明式代码生成器

**Branch**: `003-codegen` | **Date**: 2026-05-18 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/003-codegen/spec.md`

## Summary

构建 `pkg/codegen/` 包，通过 Go `text/template` + `embed.FS` 将 YAML HookSpec 声明转换为可执行的 Frida JavaScript 脚本。支持 3 种 Hook 类型模板（overload/override/native），处理 method_signature 歧义，组合多个 Hook 为完整脚本。同步修改 `pkg/spec/` 数据模型（重命名 replace→override，新增 native 类型 + method_signature/ModuleName 字段）和 `pkg/config/` 校验器。CLI 新增 `fridaforge spec generate` 子命令。

## Technical Context

**Language/Version**: Go 1.25
**Primary Dependencies**: `text/template`, `embed` (标准库，无新增外部依赖)
**Storage**: N/A (纯文本生成，无持久化)
**Testing**: Go 标准 `testing` + table-driven 测试
**Target Platform**: Linux, macOS, Windows（纯 Go，无 CGO）
**Project Type**: library (Go package) + CLI 集成
**Performance Goals**: 单 spec 生成 < 2s, 100 Hook spec < 3s
**Constraints**: 无 CGO 依赖（codegen 不依赖 frida-go）；模板内嵌于二进制
**Scale/Scope**: 单一 YAML 含 O(100) 条 Hook 目标

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| 关卡 | 章节 | 状态 | 说明 |
|------|------|------|------|
| gofmt 格式化 | 2.1 | ✅ | Makefile 已包含 `gofmt -d` |
| go vet 零告警 | 2.1 | ✅ | CI/Makefile 已配置 |
| golangci-lint | 2.1 | ✅ | `.golangci.yml` 已配置 |
| 包命名: 小写、单数、无下划线 | 2.2 | ✅ | `codegen` |
| 导出命名: MixedCaps | 2.2 | ✅ | `NewGenerator`, `Generate`, `GenerateOutput` |
| 接口命名 -er 结尾 | 2.2 | ⏸️ N/A | codegen 无 interface 定义（纯函数式风格，无状态 Generator 结构体） |
| 错误包装 %w | 2.3 | ✅ | `TemplateError`, `GenerateError` 包装底层错误 |
| 禁止裸 panic | 2.3 | ✅ | 仅 `NewGenerator` 用 error 返回 |
| goroutine 通过 context 管理 | 2.4 | ⏸️ N/A | codegen 无并发，单次 Generate() 纯同步 |
| channel 明确定义方向 | 2.4 | ⏸️ N/A | 无 channel |
| WaitGroup 代替 time.Sleep | 2.4 | ⏸️ N/A | 无 goroutine |
| Mutex 保护共享状态 | 2.4 | ⏸️ N/A | Generator 无状态，无共享数据 |
| table-driven 测试 | 2.5 | ✅ | 所有测试使用 table-driven 模式 |
| 覆盖率 ≥ 80% | 2.5 | ✅ | 模板渲染路径覆盖 100% |
| Go doc comment | 2.6 | ✅ | 所有导出类型/函数含 Go doc |
| Hook 清理保证 (defer) | 3.x | ⏸️ N/A | codegen 不操作 Frida session |
| 伦理声明 | 3.4 | ⏸️ N/A | M1 已实现 |
| SpecKit 工作流 | 5.2 | ✅ | /speckit.specify → clarify → plan → tasks → analyze → implement |
| 学习文档要求 | 6.2 | ✅ | 计划产出 `docs/learn/M3-codegen.md` |

**Re-check after Phase 1 design**: ✅ 所有关卡通过。新增 pkg/codegen/ 无并发需求，无 CGO 依赖，不引入任何宪法违规。

## Project Structure

### Documentation (this feature)

```text
specs/003-codegen/
├── plan.md              # 本文件
├── research.md          # Phase 0 产出
├── data-model.md        # Phase 1 产出
├── quickstart.md        # Phase 1 产出
├── contracts/           # Phase 1 产出
│   └── codegen-api.md   # Generator API 契约
├── checklists/
│   └── requirements.md  # Spec Quality Checklist
└── tasks.md             # Phase 2 产出 (via /speckit.tasks)
```

### Source Code (repository root)

```text
pkg/codegen/                        # M3 新增包
├── generator.go                    # Generator 结构体 + Generate()
├── generator_test.go
├── templates.go                     # embed.FS + template.ParseFS + render
├── templates_test.go
├── templates/                       # 模板文件 (embed.FS 内嵌)
│   ├── overload.js.tmpl
│   ├── override.js.tmpl
│   └── native.js.tmpl
├── types.go                        # GenerateOutput, GeneratedScript, RenderContext
├── types_test.go
├── errors.go                       # TemplateError, GenerateError
└── errors_test.go

pkg/spec/                           # M1 已有 (修改)
├── types.go                        # 新增 HookTypeNative, MethodSignature, ModuleName; 重命名 replace→override
└── types_test.go                   # 更新测试

pkg/config/                         # M1 已有 (修改)
├── validator.go                    # 更新校验规则 (3 种 HookType, native 校验 module_name, 重复检测)
└── validator_test.go               # 更新测试

cmd/fridaforge/                     # M1 已有 (修改)
└── spec.go                         # 新增 spec generate 子命令
```

**Structure Decision**: `pkg/codegen/` 作为新独立包，与 `pkg/spec/` 和 `pkg/fridaengine/` 平级。`pkg/spec/` 和 `pkg/config/` 的修改是 M1 已有文件的增量变更。codegen 无 CGO 依赖，不污染 fridaengine 的编译隔离。模板文件使用 `embed.FS` 内嵌，二进制自包含。

## Complexity Tracking

无宪法违规需要申辩。所有适用关卡均通过或明确不适用。

| 说明项 | 处理 |
|--------|------|
| pkg/spec/ 重命名 replace→override | Breaking Change (0.x.y 阶段允许)。仅影响内部常量名 + validator，无外部 API |
| codegen 无 CGO 依赖 | 区别于 fridaengine，纯 Go 编译，无 devkit 依赖 |
| 模板 embed.FS 内嵌 | 首次使用 embed 包，符合宪法"声明优于过程"原则——模板非外部文件 |
