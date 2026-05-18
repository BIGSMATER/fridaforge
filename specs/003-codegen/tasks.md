# Tasks: 声明式代码生成器

**Input**: Design documents from `specs/003-codegen/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: 宪法 2.5 强制要求覆盖率 ≥ 80%，测试为必需项。所有导出函数必须有 table-driven 测试。

**Organization**: 按依赖关系分阶段，文件修改合并到同一 Phase。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行（不同文件，无依赖）
- **[Story]**: 关联的用户故事（US1-US4）
- 包含精确文件路径

---

## Phase 1: 教学准备 + 环境搭建

**目标**: 宪法 §6.2 教学文档初始版（三轨并行），创建 `pkg/codegen/` 包目录和模板文件

**教学步骤**:
1. 📖 讲 `text/template` 语法 + `embed.FS` 概念 + Frida JS API (`Java.perform/Java.use/Interceptor.attach/send`)
2. ✍️ 创建 `docs/learn/M3-codegen.md` 初始版（独立迷你示例 10-20 行，三轨齐全）
3. 💻 创建模板文件

- [ ] T001 [P] 创建 `docs/learn/M3-codegen.md` 教学文档初始版 — 宪法 §6.2: (1) Go 轨道: text/template 语法、embed.FS 内嵌、strings.Builder; (2) 逆向轨道: Java.use/Java.perform/overload/implementation/send + Interceptor.attach 全貌; (3) AI 轨道: 声明式代码生成器设计哲学 (Spec → Template → Script 的桥)
- [ ] T002 [P] 创建 `pkg/codegen/` 和 `pkg/codegen/templates/` 目录
- [ ] T003 [P] 创建 overload 模板文件 `pkg/codegen/templates/overload.js.tmpl` — Java.perform + Java.use + .overload(sig).implementation + this.xxx() 原方法调用 + send()
- [ ] T004 [P] 创建 override 模板文件 `pkg/codegen/templates/override.js.tmpl` — Java.use + .overload(sig).implementation (完全替换，无原方法调用) + send()
- [ ] T005 [P] 创建 native 模板文件 `pkg/codegen/templates/native.js.tmpl` — Process.findModuleByName + Module.findExportByName + Interceptor.attach + 错误检查 + send()

---

## Phase 2: 数据模型变更（阻塞所有后续阶段）

**目标**: 修改 `pkg/spec/` 和 `pkg/config/` — codegen 的前置依赖

**⚠️ CRITICAL**: 此阶段未完成前，codegen 不得启动

- [ ] T006 [P] 更新 `pkg/spec/types.go` — 重命名 `HookTypeReplace` → `HookTypeOverride` (值 `"replace"` → `"override"`)，新增 `HookTypeNative`，`HookTarget` 新增 `MethodSignature string` 和 `ModuleName string` 字段 (yaml omitempty tag)
- [ ] T007 [P] 更新 `pkg/spec/types_test.go` — 覆盖 3 种 HookType 常量 + HookTarget 新字段的 YAML 反序列化
- [ ] T008 更新 `pkg/config/validator.go` — 校验 hook_type 识别 3 种值 (overload/override/native)，native 类型要求 module_name 非空，新增重复 Hook 检测 (class+method+sig+type 相同 → ValidationError.Warnings)
- [ ] T009 更新 `pkg/config/validator_test.go` — table-driven 覆盖: 3 种 hook_type 合法校验、native 缺 module_name 报错、重复 Hook 产生 warning

**Checkpoint**: spec/config 数据模型升级完成，codegen 可以依赖新类型

---

## Phase 3: Codegen 基础类型

**目标**: 定义 `GenerateOutput`、`GeneratedScript`、`RenderContext`、`TemplateError`、`GenerateError`

- [ ] T010 [P] 创建 `pkg/codegen/types.go` — `GenerateOutput` 结构体 (Combined string + Scripts []GeneratedScript)，`GeneratedScript` 结构体 (HookTarget + JSCode)，`RenderContext` 结构体 (AppPackage, ClassName, MethodName, HookType, MethodSignature, ModuleName)
- [ ] T011 [P] 创建 `pkg/codegen/errors.go` — `TemplateError` (Op, Name, Err)，`GenerateError` (Op, Err)，均实现 `Error()` 和 `Unwrap()`
- [ ] T012 [P] 创建 `pkg/codegen/types_test.go` — table-driven: GenerateOutput/Scripts 字段赋值，RenderContext 各字段默认值
- [ ] T013 [P] 创建 `pkg/codegen/errors_test.go` — table-driven: TemplateError.Error() 包含 Op/Name/Err，Unwrap() 正确，GenerateError 同理

**Checkpoint**: codegen 基础类型就绪

---

## Phase 4: 模板引擎 (US1 + US2 + US3 + US4 的模板层面)

**目标**: embed.FS 内嵌 + template.ParseFS 编译 + render 函数 — 覆盖 3 种 Hook 类型的模板渲染

**教学**: 📖 讲 `template.ParseFS` 编译 + `ExecuteTemplate` 执行 + `strings.Builder` 高效字符串拼接 → 📝 补充学习文档 §1 项目代码示例

**独立测试**: 调用 `renderTemplate("overload.js.tmpl", renderCtx)` 返回包含 `Java.use()` 的 JS 字符串；native 模板返回 `Interceptor.attach()`

- [ ] T014 [US2] 创建 `pkg/codegen/templates.go` — `//go:embed templates/*.js.tmpl` 内嵌模板，`NewGenerator(logger)` 调用 `template.ParseFS()` 编译，编译失败返回 `*TemplateError` (fail-fast)。`renderTemplate(name, ctx)` 内部方法使用 `strings.Builder` + `template.ExecuteTemplate()` 渲染单段 JS
- [ ] T015 [US3] 在 `pkg/codegen/templates.go` 的 `renderTemplate()` 中处理 method_signature: 非空时注入 `{{.MethodSignature}}` 到 overload 调用，空时输出 `.overload()`
- [ ] T016 [US4] 在 `pkg/codegen/templates.go` 的 `renderTemplate()` 中加入 native 类型分发: 当 HookType=="native" 时使用 native.js.tmpl 模板
- [ ] T017 [US2] 创建 `pkg/codegen/templates_test.go` — table-driven: (1) NewGenerator 模板编译成功，(2) 3 种 HookType 各渲染为合法 JS 代码段，(3) method_signature 空/非空分支， (4) native 模板含 findModuleByName 检查

**Checkpoint**: 模板引擎可独立使用 — 给定 HookTarget + RenderContext 可生成对应 JS 代码段

---

## Phase 5: 生成器 (US1 + US3 + US4 的组装层面)

**目标**: `Generator.Generate()` — 遍历 HookSpec.Hooks，按类型分发模板，组装 Combined 输出 (Java.perform 包装 + Native hooks)

**教学**: 📖 讲 `template.Execute` 数据注入 + `GenerateOutput` 组装逻辑 + 模板-代码分离设计 → 📝 补充学习文档 §2 项目代码示例

**独立测试**: 给定一个 3 Hook 的 HookSpec (覆盖 3 种类型)，`Generate()` 返回输出包含 Java.perform() + 2 个 Java hook + 1 个 Native hook

- [ ] T018 [US1] 创建 `pkg/codegen/generator.go` — `NewGenerator()` 工厂（延迟到 NewGenerator 时才编译模板），`Generate(spec)` 方法：遍历 hooks，对每个 HookTarget 构建 RenderContext，调用 renderTemplate()，区分 Java/Native 组装 Combined
- [ ] T019 [US1] 在 `pkg/codegen/generator.go` 的 `Generate()` 中实现 Java hooks 包裹在单个 `Java.perform(function() { ... })` 内，Native hooks 裸放在其后。无 Java hooks 时跳过 Java.perform() 包装
- [ ] T020 [US3] 在 `pkg/codegen/generator.go` 中为 RenderContext 填充 MethodSignature 字段 (直接取自 HookTarget.MethodSignature)
- [ ] T021 [US4] 在 `pkg/codegen/generator.go` 中为 Native RenderContext 忽略 ClassName/AppPackage，使用 ModuleName
- [ ] T022 [US1] 创建 `pkg/codegen/generator_test.go` — table-driven: (1) 单 overload Hook 生成完整脚本，(2) 多 Hook 混合 (overload+override+native) Combined 结构验证，(3) 空 hooks → 错误，(4) Generate() 输出合法 JS (含 `Java.perform`)
- [ ] T023 [US4] 在 `pkg/codegen/generator_test.go` 追加 pure-native Hook 测试 — 验证跳过 Java.perform()

**Checkpoint**: 生成器完整可用 — HookSpec → 合法 Frida JS 脚本

---

## Phase 6: CLI 集成 (US1 CLI 层面)

**目标**: `fridaforge spec generate` 子命令

**教学**: 📖 讲 cobra 子命令注册 + `os.WriteFile` vs `fmt.Println` 输出策略 → 📝 补充学习文档 §3 项目代码示例

**独立测试**: `fridaforge spec generate valid.yaml` 输出 JS 到 stdout，`-o out.js` 写入文件，无效 YAML 报错

- [ ] T024 [US1] 在 `cmd/fridaforge/spec.go` 新增 `specGenerateCmd` cobra 子命令: Use="generate <文件>"，Short="生成 Frida JS Hook 脚本"，Args=cobra.ExactArgs(1)，RunE 实现: LoadSpec → Validate → NewGenerator → Generate → 输出
- [ ] T025 [US1] 在 `cmd/fridaforge/spec.go` 添加 `-o, --output` flag (string) 和 `-t, --target` flag (string) — -o 控制文件输出 vs stdout，-t 按 `className.methodName` 精确匹配过滤 (不含 signature)
- [ ] T026 [US1] 在 `cmd/fridaforge/spec.go` 的 RunE 中实现 `-o` 输出逻辑: 指定时 `os.WriteFile`，未指定时 `fmt.Println(Combined)`

**Checkpoint**: CLI `fridaforge spec generate` 端到端可用

---

## Phase 7: 打磨与交叉关注

**目标**: 代码质量验证，文档更新

- [ ] T027 运行 `gofmt -d` 并修复所有格式问题，验证所有导出类型/函数含 Go doc comment（宪法 §2.6）
- [ ] T028 运行 `go vet ./...` 并修复所有警告
- [ ] T029 运行 `golangci-lint run ./...` 并修复所有问题
- [ ] T030 运行 `go test -coverprofile=coverage.out ./pkg/codegen/ ./pkg/spec/ ./pkg/config/` — 覆盖率 ≥ 80%，追加未完覆盖路径
- [ ] T031 运行 `go test -v ./pkg/codegen/` — 验证全部测试通过
- [ ] T032 运行 `go test -bench=. ./pkg/codegen/` — 验证 SC-001 (<2s 单 spec) 和 SC-003 (<3s 100 hooks) 性能指标
- [ ] T033 [P] 创建 `pkg/codegen/integration_test.go` — 使用 `//go:build integration` 标签，真机 Frida 加载生成脚本验证 SC-002 (100% Frida load success)
- [ ] T034 [P] 更新 `docs/learn/M3-codegen.md` — 各 Phase 已追加项目代码示例，补全最终版本并标记"已完成"
- [ ] T035 [P] 更新 `docs/milestones.md` — 标记 M3 完成状态，更新实际产出物清单
- [ ] T036 [P] 更新 `AGENTS.md` — 标记 M3 完成状态


---

## 依赖与执行顺序

### 阶段依赖

```
Phase 1 (环境)
  └─► Phase 2 (数据模型变更)
        └─► Phase 3 (基础类型) ──► Phase 4 (模板引擎) ──► Phase 5 (生成器) ──► Phase 6 (CLI) ──► Phase 7 (打磨)
```

### 各阶段内部顺序

- Phase 1: T001 教学文档初始版 (独立，可最早启动)
- Phase 2: T006+T007 可并行 → T008 → T009 (validator 测试依赖 validator 实现)
- Phase 3: T010+T011 可并行 → T012+T013 可并行
- Phase 4: T014 → T015+T016 可并行 (不同逻辑分支) → T017
- Phase 5: T018 → T019+T020+T021 可并行 → T022+T023 可并行
- Phase 6: T024 → T025+T026 可并行

### 可并行任务汇总

| Phase | 可并行任务 |
|-------|-----------|
| Phase 1 | T001, T002, T003, T004, T005 |
| Phase 2 | T006+T007 |
| Phase 3 | T010+T011, T012+T013 |
| Phase 4 | T015+T016 |
| Phase 5 | T019+T020+T021, T022+T023 |
| Phase 7 | T034+T035+T036 |

---

## 实施策略

### MVP 优先 (Phase 1-5)

1. 完成 Phase 1: 环境准备 → 目录 + 3 个模板文件
2. 完成 Phase 2: 数据模型变更 → spec/config 升级
3. 完成 Phase 3: 基础类型 → types + errors
4. 完成 Phase 4: 模板引擎 → embed + render 可用
5. 完成 Phase 5: 生成器 → `Generator.Generate()` 可用
6. **停一下并验证**: 在 Go 测试和手动 YAML 上验证生成正确性

### 渐进交付

1. Phase 1: 教学文档 + 环境 → 学习 doc 初始版 + 3 个模板就绪
2. +Phase 2: 数据模型 → spec/config 升级完成
3. +Phase 3: 基础类型 → codegen 类型定义就绪
4. +Phase 4: 模板引擎 → 模板渲染可独立测试 (+ 📝 补充学习 doc §1)
5. +Phase 5: 生成器 → 核心功能可用 Minimal Viable Codegen (+ 📝 补充学习 doc §2)
6. +Phase 6: CLI → 端到端 `fridaforge spec generate` 可用 (+ 📝 补充学习 doc §3)
7. Phase 7: 打磨 → 学习 doc 最终版 + 就绪进入 M4

---

## 备注

- [P] 任务 = 不同文件，无依赖
- [Story] 标签将任务追溯到具体用户故事
- 测试: 宪法 2.5 要求覆盖率 ≥ 80%，测试为必需项
- 同一文件中的多个 Task 必须合并为一个 Commit
- 不同 package 的 Task 应分开 Commit
- Phase 1-6 每完成一个 Phase 做一次 Commit
- 宪法合规: 每个 Phase 实施前检查 `.specify/memory/constitution.md`
