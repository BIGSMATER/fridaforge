# 研究笔记: 声明式代码生成器

**功能**: 003-codegen | **日期**: 2026-05-18

## 决策 1: 模板引擎选型 — text/template

**决策**: 使用 Go 标准库 `text/template` 而非其他模板引擎。

**理由**:
- 标准库，零外部依赖（宪法目标：减少依赖）
- 语法简洁，支持 `{{.Field}}` 变量注入和 `{{range}}` 迭代
- `template.ParseFS()` 直接与 `embed.FS` 集成，从二进制内嵌文件编译模板
- M3 模板复杂度低（3 个文件，每个 < 30 行），不需要高级模板特性（partial、inheritance）

**备选方案**:
- `html/template`: 对 JS 代码过度转义 (`<` `>` 会被 HTML 编码)，不适合
- `pongo2` / `quicktemplate`: 第三方依赖，M3 不需要
- 手拼字符串 (`strings.Builder`): 简单但不可维护，模板与逻辑耦合

## 决策 2: method_signature 处理 — 整串原样

**决策**: M3 不做签名分割，整串原样插入到 `.overload('整串')` 中。

**理由**:
- 泛型签名 (`java.util.List<java.lang.String>`) 的逗号分割需要实现完整的签名解析器 → 过度工程
- 用户自己写的签名符合 Frida 格式即可，系统不校验
- 签名解析推迟到 M5（届时需要更精确的代码生成）
- clarify 阶段确认：使用 `RenderContext.MethodSignature` 直接注入模板

**模板中的用法**:
```
.overload('{{.MethodSignature}}')   // 非空时
.overload()                          // 空时
```

## 决策 3: embed.FS 使用模式

**决策**: 使用 `//go:embed templates/*.js.tmpl` 将所有模板嵌入单个 `embed.FS`，在 `NewGenerator()` 时一次性 `template.ParseFS()` 编译。

**理由**:
- `template.ParseFS(fs, "templates/*.js.tmpl")` 一次编译所有模板
- 按名称查找模板渲染: `t.ExecuteTemplate(buf, "overload.js.tmpl", ctx)`
- 编译失败时 `NewGenerator()` 返回 error (fail-fast, clarify 确认)
- P3 阶段讲解 `embed.FS` 概念（Go 轨道）

**模板目录结构**:
```
pkg/codegen/templates/
├── overload.js.tmpl   // Java.perform + Java.use + .overload(sig).implementation + this.xxx()
├── override.js.tmpl   // Java.perform + Java.use + .implementation（完全替换）
└── native.js.tmpl     // Process.findModuleByName + Interceptor.attach
```

## 决策 4: Generator 无状态设计

**决策**: `Generator` 持有预编译的 `*template.Template`，无其他可变状态。`Generate()` 是纯函数（相同输入 → 相同输出）。

**理由**:
- 无需 Mutex 保护（无共享可变状态）
- 可重用（一次初始化，多次 Generate）
- 无需 context.Context（同步操作，无 IO/网络）
- 符合宪法 Simplicity 原则

## 决策 5: Combined 输出结构

**决策**: Combined 脚本按以下顺序组织:
1. `Java.perform(function() { /* 所有 Java hooks */ })` — 仅当有 Java hooks 时
2. 随后裸放 Native hooks（不在 `Java.perform()` 内）

**理由**:
- `Java.perform()` 是 Frida 的 Java bridge 入口，Native hooks 不应在其中
- 一个 spec 中对同一 app 的所有 Java hooks 共享一个 `Java.perform()` 包装（减少 bridge 开销）
- 若无 Java hooks，输出全为 Native `Interceptor.attach()` 代码
- 顺序: Java hooks first (包装在 Java.perform 中), Native hooks after
- clarify 阶段未明确顺序要求，选择先 Java 后 Native（常规 Android hook 实践）

**备选方案**:
- 每个 Hook 独立 `Java.perform()`: 冗余，但隔离性好
- Native hooks 在前: 不影响功能，仅风格差异

## 决策 6: Breaking Change — replace → override

**决策**: 直接删除 `HookTypeReplace` 和字符串值 `"replace"`，不保留别名。

**理由**:
- 0.x.y 阶段允许 Breaking Changes（宪法 5.3）
- 项目尚未发布稳定版，无外部用户依赖 `"replace"` 字符串
- clarify 确认：硬 Breaking Change，不保留别名
- 影响范围: `pkg/spec/types.go` (常量) + `pkg/config/validator.go` (校验) + 测试文件

## 决策 7: 重复 HookTarget 检测策略

**决策**: 校验时检测重复（class + method + signature + type 完全相同），输出 warning 结构但不阻止 generate。

**理由**:
- clarify Q1 确认: Option A — 原样生成 + warning
- 两个相同的 Hook 在 Frida 中安全（重复 attach 同一方法不影响运行）
- Warning 提示用户可能无意重复配置
- 不在生成阶段去重——保留用户原始意图

**实现**: `validate()` 返回 `ValidationError` 时新增 `Warnings []string` 字段。

## 决策 8: 测试策略 — 模板覆盖 + 模板编译错误

**决策**: (1) 单元测试用 table-driven 覆盖 3 种模板路径。(2) 模板编译错误在 `NewGenerator()` 测试中覆盖。

**理由**:
- codegen 无 CGO，单元测试在任意环境运行
- 无 frida-server 依赖（与 M2 不同，M3 不需要集成测试标签）
- 覆盖率目标 ≥ 80%: templates.go + generator.go 为主要覆盖目标
- 模板内容正确性由 table-driven 测试验证（比较字符串输出）

## 已解决的技术未知项

所有 Phase 0 研究项已解决:
- 模板引擎选型 ✅
- method_signature 处理 ✅
- embed.FS 模式 ✅
- Generator 设计 ✅
- Combined 结构 ✅
- Breaking Change 策略 ✅
- 重复检测 ✅
- 测试策略 ✅
