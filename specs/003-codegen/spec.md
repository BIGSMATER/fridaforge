# Feature Specification: 声明式代码生成器

**Feature Branch**: `003-codegen`
**Created**: 2026-05-18
**Status**: Draft
**Input**: User description: "声明式代码生成器 — 读取 YAML Hook 规格文件，自动生成可执行的 Frida JavaScript 脚本"

## Clarifications

### Session 2026-05-18

- Q: 两个 HookTarget 条目完全相同（class + method + signature + type 均相同）时，生成器应如何行为？ → A: 原样生成，重复代码段。validate 时输出 warning 提醒用户存在重复配置。
- Q: 内嵌模板文件（.tmpl）存在语法错误时，Generator 创建应如何行为？ → A: NewGenerator() 返回 error，fail-fast（模板是二进制内嵌的，编译错误 = 代码 bug）。
- Q: method_signature 含泛型（如 `java.util.List<java.lang.String>, int`），逗号在泛型括号内导致切分歧义，如何处理？ → A: M3 整串原样插入，不分割参数。签名分割逻辑推迟到 M5。

### Session 2026-06-01

- Q: 真机测试发现 Frida 17 移除了全局 `Java` 对象和 `Module.findExportByName(name, sig)` API，生成的脚本无法直接运行。如何处理？ → A: 新增 `--frida-version` CLI flag（默认 16）。Frida 16 模式保持原有 `Java.perform()` 直接生成；Frida 17 模式在 Java hooks 前插入 `frida-compile` 引导注释，Native 模板使用实例方法 `module.findExportByName()`。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 从 YAML 规格生成可执行 Frida 脚本 (Priority: P1)

逆向工程师编写了一份 YAML Hook 规格文件并通过了 `spec validate` 校验。现在需要将该规格转换为可注入 Frida 的可执行 JavaScript 脚本。用户运行 `fridaforge spec generate hooks.yaml`，系统读取 YAML，先校验合法性，校验通过后输出完整的 Frida JavaScript 代码。

**Why this priority**: 代码生成是 FridaForge 从"配置平台"到"脚本工程化平台"的关键转折。没有代码生成，用户仍需手写 Frida JS 脚本，FridaForge 只是配置校验器。这是 M3 的核心价值交付。

**Independent Test**: 提供一个通过校验的合法 YAML 文件（包含单个 overload Hook），运行 `fridaforge spec generate`，验证输出是否为语法正确的 JavaScript 且包含 `Java.perform()` + `Java.use()` + `.implementation`。

**Acceptance Scenarios**:

1. **Given** 一个包含单个 overload Hook 目标的合法 YAML，**When** 运行 `fridaforge spec generate`，**Then** 输出包含 `Java.perform()` 包装的 JavaScript 代码，其中包含 `Java.use()` 和 `.implementation` 方法替换，退出码为 0。
2. **Given** 一个包含多个 Hook 目标的合法 YAML，**When** 生成脚本，**Then** 所有 Hook 目标均出现在输出脚本中，Java hooks 共享一个 `Java.perform()` 包装。
3. **Given** 一个未通过 validate 校验的 YAML 文件，**When** 运行 `fridaforge spec generate`，**Then** 系统输出校验错误信息，退出码非 0（不生成脚本）。
4. **Given** 用户指定 `-o output.js` 参数，**When** 生成脚本，**Then** 内容写入 `output.js` 文件，stdout 无输出。

---

### User Story 2 - 按 Hook 类型生成正确的 Frida API 代码 (Priority: P1)

不同的 Hook 类型（overload、override、native）对应完全不同的 Frida JavaScript API 模式。代码生成器必须为每种类型生成语法正确、语义准确的代码。

**Why this priority**: 模板正确性是代码生成器的核心质量指标。如果生成的代码语法错误或使用了错误的 API，用户无法使用。P1 因为它是 US1（生成脚本）的子维度——US1 生成脚本，US2 保证生成的是正确的脚本。

**Independent Test**: 对每种 Hook 类型分别提供单独的 YAML，验证输出中包含该类型对应的 Frida API 调用（如 overload 包含 `this.xxx()` 原方法调用，native 包含 `Interceptor.attach()`）。

**Acceptance Scenarios**:

1. **Given** HookType 为 `overload`，**When** 生成脚本，**Then** 生成的代码保留 `this.xxx()` 原方法调用，向调用者返回原方法返回值，通过 `send()` 发送参数和返回值。
2. **Given** HookType 为 `override`，**When** 生成脚本，**Then** 生成的代码完全替换方法实现，通过 `send()` 发送拦截信息，不自动调用原方法（用户自行决定返回值）。
3. **Given** HookType 为 `native`，**When** 生成脚本，**Then** 生成的代码使用 `Process.findModuleByName()` + `Module.findExportByName()` 定位函数，使用 `Interceptor.attach()` 附加 onEnter/onLeave 回调。
4. **Given** 生成任意类型的脚本，**When** 用户将脚本注入目标进程，**Then** Frida 能成功解析并执行脚本（无语法错误）。

---

### User Story 3 - 处理方法参数签名歧义 (Priority: P2)

当目标类中存在多个同名方法（重载）时，仅靠方法名无法确定 Hook 目标。用户可以在 YAML 中提供 `method_signature` 字段（如 `"java.lang.String, int"`），生成器据此使用 Frida 的 `.overload('签名')` 精确匹配。

**Why this priority**: 参数签名是 M1 明确推迟到 M3 的能力。实际应用中重载方法极为常见（如 `Log.d()/Log.e()/Log.i()`），缺少签名支持将导致 Hook 命中错误的方法。

**Independent Test**: 提供一个 YAML 文件，其中包含两条同名但签名不同的 Hook 声明，验证生成的两个代码段分别使用了不同的 `.overload('签名')` 调用。

**Acceptance Scenarios**:

1. **Given** HookTarget 提供了 `method_signature: "java.lang.String, int"`，**When** 生成脚本，**Then** 代码中包含 `.overload('java.lang.String', 'int')` 的精确匹配。
2. **Given** HookTarget 未提供 `method_signature`（为空），**When** 生成脚本，**Then** 代码中使用 `.overload()`（无参数）匹配第一个找到的重载方法。
3. **Given** `method_signature` 包含多参数和数组类型（如 `"byte[], int"`），**When** 生成脚本，**Then** 签名正确转换为 Frida overload 参数格式。

---

### User Story 4 - Native 函数 Hook (Priority: P2)

某些逆向场景需要 Hook Native 层（.so）的函数而非 Java 方法。用户通过指定 `hook_type: native`、`module_name` 和 `method_name`（导出函数名）来声明 Native Hook 目标。

**Why this priority**: Native Hook 是完整的逆向工具箱必要组成部分。许多应用的加解密、反调试逻辑在 Native 层实现。但 Native Hook 声明字段与 Java Hook 不同（需要 module_name 而非 class_name），需要独立的校验和生成逻辑。

**Independent Test**: 提供一个 `hook_type: native` 的 YAML（包含 module_name 和 method_name），验证生成代码包含 `Interceptor.attach()` 且正确处理了模块未找到的情况。

**Acceptance Scenarios**:

1. **Given** HookType 为 `native` 且提供了 `module_name` 和 `method_name`，**When** 生成脚本，**Then** 代码包含 `Process.findModuleByName()` 检查模块存在性，随后 `Interceptor.attach()` 附加回调。
2. **Given** Native Hook 的 `method_name` 指向不存在的导出函数，**When** 生成脚本，**Then** 代码中包含 `if (targetAddr === null)` 检查并输出 `console.log` 错误提示（生成安全代码，不移除检查）。
3. **Given** Native Hook 的 `module_name` 为空，**When** 校验（validate），**Then** 报告 validation 错误（native 类型必须提供 module_name）。

---

### Edge Cases

- 空的 hooks 列表（HookSpec.Hooks 长度为 0）——generate 之前先 validate，validate 拒绝空 hooks 列表。
- 模板渲染时包含 YAML 特殊字符（如上标/引号）的类名/方法名应正确转义。
- method_signature 中包含空格、泛型符号（如 `java.util.List<java.lang.String>`）——当前视为字符串原样插入，不做解析。复杂的泛型签名由用户自行处理。
- Native Hook 与 Java Hook 混合在同一个 spec 中——Native 的 `Interceptor.attach()` 不应被套入 `Java.perform()` 包装内。Combined 脚本将 Java hooks 和 Native hooks 分开组织。
- HookTarget 中同时存在 `class_name` 和 `module_name` 但 hook_type 为 native——class_name 应被忽略（模板不使用它）。
- 生成的目标脚本体积过大（上百个 Hook）时不应超时或 OOM。
- 两个完全相同的 HookTarget（同 class + method + signature + type）——生成器原样生成重复代码段；validate 阶段输出 warning（不阻止生成）。
- 内嵌模板 .tmpl 文件包含语法错误——Generator 初始化失败，`NewGenerator()` 返回 error（fail-fast，不在运行时静默降级）。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 提供 `fridaforge spec generate <文件>` CLI 子命令，读取 YAML Hook 规格文件并输出可执行 Frida JavaScript 脚本。
- **FR-002**: 系统 MUST 支持 `overload`、`override`、`native` 三种 HookType，每种类型生成对应的 Frida API 代码模板。
- **FR-003**: `overload` 类型 MUST 保留原方法调用 (`this.xxx()`)，向调用者返回原返回值，并通过 `send()` 发送参数/返回值。
- **FR-004**: `override` 类型 MUST 完全替换方法实现，通过 `send()` 发送拦截信息。
- **FR-005**: `native` 类型 MUST 使用 `Process.findModuleByName()` + `Module.findExportByName()` + `Interceptor.attach()`，生成安全的模块存在性检查代码。
- **FR-006**: 所有 Java Hook（overload/override）MUST 被单个 `Java.perform()` 回调包装。
- **FR-007**: Native Hook MUST 不被 `Java.perform()` 包装，与 Java Hook 在 Combined 输出中分开组织。
- **FR-008**: 当 `method_signature` 非空时，MUST 将该签名整串原样渲染到 Frida API 的 `.overload('签名整串')` 中，不做分割预处理。
- **FR-009**: 当 `method_signature` 为空时，MUST 使用 `.overload()`（无参数）匹配第一个重载方法。
- **FR-010**: 生成的脚本 MUST 是语法正确的 JavaScript。
- **FR-011**: 系统 MUST 先校验 spec 再生成——校验失败时返回错误，退出码非 0。
- **FR-012**: 支持 `-o, --output` 标志将生成脚本写入文件；未指定时输出到 stdout。
- **FR-013**: 支持 `-t, --target` 标志按 `className.methodName` 过滤，仅生成匹配的 Hook 脚本段。
- **FR-014**: `HookTarget` 结构体 MUST 新增 `MethodSignature string` 和 `ModuleName string` 字段（均为可选，YAML 中使用 omitempty tag）。
- **FR-015**: 将已有 `HookTypeReplace` 常量和字符串值 `"replace"` 重命名为 `HookTypeOverride` 和 `"override"`。新增 `HookTypeNative` 常量及 `"native"` 字符串值。升级为 Breaking Change（0.x.y 阶段允许）。
- **FR-016**: 校验器 MUST 识别三种合法 HookType (`overload`/`override`/`native`)，对 `native` 类型校验 `module_name` 非空。
- **FR-017**: 校验器 MUST 检测重复 Hook 目标（class_name + method_name + method_signature + hook_type 完全相同），输出 warning 但不阻止生成。
- **FR-018**: Generator 初始化 MUST 在模板编译失败时返回 error，不静默降级（fail-fast 策略）。

### Key Entities

- **Generator**: 代码生成器的入口。接收 `*spec.HookSpec`，返回 `*GenerateOutput`。内部持有预编译的 Go 模板（text/template），无状态，每次 `Generate()` 调用独立运行。
- **GenerateOutput**: 生成结果，包含 `Combined`（完整可执行脚本字符串）和 `Scripts []GeneratedScript`（各 Hook 独立代码段）。
- **GeneratedScript**: 单个 Hook 目标的生成结果，包含原始 `HookTarget` 引用和生成的 `JSCode` 字符串。
- **RenderContext**: 模板渲染的数据上下文，包含 `AppPackage`、`ClassName`、`MethodName`、`HookType`、`MethodSignature`、`ModuleName` 及预处理后的 `MethodSignatureParams []string`。
- **TemplateError**: 模板编译/渲染阶段发生的错误（实现 error 接口，包装底层模板错误）。
- **GenerateError**: 生成过程中发生的错误，如无法识别的 HookType。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 用户能在 2 秒内完成单个 Hook 规格文件的代码生成（从 YAML 读入到 JS 输出）。
- **SC-002**: 生成脚本在 Frida 中加载成功率为 100%（无语法错误），通过至少 3 种 Hook 类型 + 1 个混合场景（Java+Native）的真机验证。
- **SC-003**: 包含 100 个 Hook 目标的 YAML 文件生成时间不超过 3 秒。
- **SC-004**: 生成代码覆盖 3 种模板路径（overload/override/native），通过 table-driven 测试覆盖所有分支。
- **SC-005**: `fridaforge spec generate --help` 输出的帮助信息能让新手独立完成第一次代码生成。

## Assumptions

- 用户传入的 YAML 文件已通过 `spec validate` 校验（generate 内部也会校验，但规范假设输入合法）。
- 生成的脚本假设 frida-server 已运行在目标设备上，且目标应用使用标准 Dalvik/ART 运行时。
- `method_signature` 字段使用 Frida overload 格式：逗号分隔的类型名，如 `"java.lang.String, int"`。M3 整串原样插入模板，不做参数分割。语法校验推迟到 M5。
- Native Hook 假设目标 .so 在脚本执行时已被目标进程加载——`Process.findModuleByName()` 负责运行时检查。
- M1 旧的 YAML 文件中 `hook_type: replace` 不再支持。用户需要迁移到 `hook_type: override`。0.x.y 阶段允许 Breaking Change。
- 模板文件使用 Go `embed.FS` 内嵌到二进制中，不依赖外部文件（codegen 是纯文本生成，不依赖 CGO 或 frida-go）。

## Dependencies

- M1 产物: `pkg/spec/` 的 `HookSpec`、`HookTarget` 结构体（需修改——新增字段，重命名常量）
- M1 产物: `pkg/config/` 的 `LoadSpec()` 和 `Validate()`（需修改 Validate 校验逻辑）
- M1 产物: `cmd/fridaforge/` 的 CLI 骨架（新增子命令，复用已有 `spec` 命令树）
- 无新增 Go 外部依赖——`text/template`、`embed`、`strings` 均为标准库。

## Out of Scope

- 不生成 Hook 后处理逻辑（如解密结果输出、自定义返回值修改）——用户需自行修改生成脚本。
- 不处理 Android Framework 类的特殊 Hook（如 Binder 拦截需要额外上下文信息）。
- 不提供交互式脚本编辑器——生成器是单向的 YAML → JS 转换。
- `method_signature` 的语法校验（格式校验推迟到 M5+，当前信任用户输入符合 Frida 格式）。
- 不处理 Spawn 模式注入（仅生成 Attach 模式的脚本结构）。
