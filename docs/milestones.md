# FridaForge — 全局 Milestone 计划

> 本文档记录项目从 M0 到 M7 的完整里程碑路线图。每个 Milestone 的执行严格遵循 SpecKit 工作流。

## 当前状态：M4 完成 → 准备进入 M5

---

## M0：项目初始化与宪法确立 ✅

| 维度 | 内容 |
|------|------|
| **Go 知识** | Go 项目目录结构约定、`go mod init`、`.gitignore` 设计 |
| **逆向知识** | Frida 宏观架构：GumJS 引擎 → Interceptor → Stalker 调用栈 |
| **AI 范式** | 首遇 SpecKit 工作流，理解 Spec Coding vs Vibe Coding 本质差异 |
| **产出物** | `constitution.md` (6章), `go.mod`, `.gitignore`, `README.md`, `speckit-rationale.md` |
| **已提交** | commit `1ba38b4` |

---

## M1：CLI 骨架与声明式配置解析 ✅

| 维度 | 内容 |
|------|------|
| **SpecKit 流程** | `/speckit.specify` → `clarify` → `plan` → `tasks` → `analyze` → `implement`（全部完成） |
| **Go 知识** | **基础语法：** `package`/`import`、`struct` + tag、`func` 与方法接收者、`if err != nil` + `%w`、`slice`/`map`、`fmt`、`os.Args`；**CLI 框架：** `cobra.Command` 树形命令注册、`yaml.v3` 反序列化；**工程设计：** `interface` 抽象、`context.Context`、`text/tabwriter`、`init()` 自动初始化、逃逸分析、nil 接口陷阱；**工具链：** `go.mod`/`go.sum`/GOPROXY、golangci-lint、Makefile |
| **逆向知识** | YAML Spec 的逆向语义：`className` → Dalvik 类全限定名、`methodName` → ART 方法签名格式、`hookType` → `overload`/`replace` 差异；Frida 三端架构（开发端/传输层/目标端）、frida-core/frida-server/frida-agent 分工 |
| **目标产出** | `cmd/fridaforge/` (CLI 入口完整), `pkg/config/`, `pkg/spec/`, `pkg/device/`；14 个 Go 源文件；覆盖率 100%；教学文档 1146 行 |
> 注：`viper` 原计划使用，M1 评估后认为 `os.ReadFile` + `yaml.Unmarshal` 足够——viper 的核心价值在多来源配置合并，M1 无需此能力。推迟到 M2+。

---

## M2：Frida 并发调度引擎（`fridaengine`）✅

| 维度 | 内容 |
|------|------|
| **Go 知识** | **并发核心：** goroutine、`sync.WaitGroup`、`context.WithTimeout/Cancel`、`sync.Mutex/RWMutex`、channel 生产-消费者模式；**工程设计：** `interface` 抽象 (`DeviceManager`/`SessionManager`)、依赖注入、错误包装 (`%w`) |
| **逆向知识** | Frida 完整生命周期：`enumerate_devices()` → `attach()` → `create_script()`；`frida-server` 部署；USB vs 网络远程管理；Scope 枚举 (ScopeMinimal/Full) |
| **Harness** | 最小 Android App（Hello World 方法），验证 Attach + 调用方法（真机测试通过） |
| **产出物** | `pkg/fridaengine/` 12 个源文件 (6 生产 + 6 测试含 `integration_test.go`); 38 tests; 覆盖率 76.4% (CGO 路径由 integration tag 补全); 教学文档 ~1140 行; Makefile `make devkit`; M2 回顾文档 |

---

## M3：声明式代码生成器（`codegen`）✅

| 维度 | 内容 |
|------|------|
| **Go 知识** | `text/template` 模板渲染、`embed.FS` 内嵌文件、`strings.Builder`、`go generate`、`os/exec` |
| **逆向知识** | Frida JS API 深度：`Java.perform()`, `Java.use()`, `.implementation =`, `this.xxx()` 原方法调用；Hook 类型模板化 (Override/Overload/Native) |
| **Harness** | 扩展 M2 的测试 App（加 Native 函数），验证生成脚本正确性 |
| **产出物** | `pkg/codegen/` 7 个源文件 (generator.go, templates.go, types.go, errors.go + 4 test + 1 integration test); `pkg/spec/types.go` 新增 native/override + 2 字段; `pkg/config/validator.go` 升级校验; `cmd/fridaforge/spec.go` 新增 generate 子命令; 3 个模板文件 (.js.tmpl); 教学文档 .md + .html; 30 tests; 覆盖率 97% |
| **已提交** | commit `260489c` — 最终 analyze 修复完成 |
| **经验教训** | 见下方 ↓ |

**M3 经验教训：**

1. **HTML 教学文档效果优于纯 Markdown** — `<details>` 折叠、`<aside>` 标注框、`<dl>` 定义列表等语义元素在大纲式学习中优势明显。宪法 §6.2 已正式确立两者同等地位。

2. **学习文档应按知识点组织，非按 Phase** — 最初的教学文档以 "Phase 2 知识点" 为标题，学员反馈后重构为以知识点为主的平铺结构。Phase 是实现里程碑，知识点是认知单元——二者不应混淆。

3. **代码注释语种须与项目一致** — codegen 包最初用了英文注释，但项目既有代码全为中文。跨文件维护两种语言注释增加认知成本，统一为中文。

4. **模板正确性不可仅靠单元测试** — `override.js.tmpl` 当初与 `overload.js.tmpl` 行为完全相同（都调用了 `this.method()`），但单元测试只检查子串存在性（"hooked (override)"），未检查语义正确性。bug 在 `/speckit.analyze` 交叉验证阶段才暴露。

5. **SpecKit 中途新需求必须先更新 spec + tasks** — 实现阶段发现 Frida 17 API 不兼容，直接写了 `--frida-version` 功能但没有先更新 spec/tasks，导致文档与代码暂时不一致。正确流程：clarify → 更新 spec → 追加 tasks → 再写代码。

6. **集成测试须在真机验证** — Frida 17 的 `Java` 全局移除和 `Module.findExportByName` API 变更，在纯单元测试中完全不可见。真机集成测试（SC-002）暴露了这些运行时问题，直接导致了 `--frida-version` 功能的诞生。

7. **`/speckit.implement` 是 Build Mode，`/speckit.analyze` 及之前阶段是 Plan Mode** — 实现阶段可以修改文件（构建代码）；规划和分析阶段只读（输出报告）。Mode 切换有明确边界，避免在"分析"时误改源码。

---

## M4：MCP Server 集成（`mcpserver`）✅

| 维度 | 内容 |
|------|------|
| **Go 知识** | 泛型 (`[In, Out any]`)、反射 (`reflect.TypeOf` + struct tag)、闭包捕获依赖（轻量 DI）、`log/slog` 结构化日志、interface 隐式实现 + 注入、`encoding/json` 自定义序列化 |
| **AI 范式** | MCP 协议全流程（JSON-RPC 2.0 + stdio Transport + Tool/Resource/Prompt 三原语）、LLM 如何通过 Tool 定义理解并调用外部功能、opencode MCP 集成内部机制（注册表 → 翻译层 → 管道通信） |
| **Harness** | opencode 本地 MCP 连接 FridaForge MCP Server，让大模型自动生成 Hook 脚本 |
| **产出物** | `pkg/mcpserver/` 7 个源文件 (4 生产 + 3 测试)、`cmd/fridaforge/mcp.go` 新子命令、25 tasks 全部完成；28 tests + 3 benchmarks；覆盖率 88.4%；教学文档 539 行 (.md) + 416 行 (.html)；全链路 MCP 协议握手 + tools/list + tools/call 验证通过 |
| **已提交** | 共计 14 commits on `004-mcpserver` |

**M4 经验教训：**

1. **闭包捕获依赖是 Go 中框架约束下的标准 DI 模式** — go-sdk 锁死 handler 签名后，Server 的依赖无法通过参数传入。闭包（匿名函数捕获外层变量）是 Go 解决"框架要固定签名、你的代码需要额外参数"的惯用方案。学员从"完全不懂闭包"到"理解轻量 DI"，梯度教学（普通函数 → 函数内部函数 → 返回闭包 → 捕获变量变化 → 项目代码）有效。

2. **泛型 + 反射是 Go 框架设计的"编译期安全 + 运行时灵活"组合拳** — 泛型保证编译期类型安全（handler 签名不匹配直接编译报错），反射在运行时自动生成 JSON Schema（框架作者不需要知道你定义了什么 struct）。这种模式在 Go 第三方库中越来越常见。学员对反射的理解需要从"照镜子"的类比开始，再深入到 `reflect.TypeOf` → `Field.Tag.Get`。

3. **MCP 协议的教学需要分层：概念 → 协议 → 实现** — 第一层讲"MCP 是 AI 和工具的翻译官"（概念），第二层讲 JSON-RPC 2.0 消息格式和生命周期（协议），第三层讲 go-sdk 怎么装箱 Go 函数变成 Tool（实现）。学员在第三层的"装箱"理解上困惑最多，需要把 `AddTool` 内部的"反射读 tag → JSON Schema"和"JSON 反序列化 → 调 handler → 序列化返回"拆成独立步骤。

4. **文档交叉验证不可跳过——替换编辑极易丢失内容** — M4 的 spec.md 在一次编辑中丢失了"操作日志"Q&A 和"MCP 协议版本"edge case。原因是用 `edit` 工具做替换时，`oldString` 不完全匹配导致静默失败，后续编辑又无意中覆盖了之前的行。教训：每次修改多行文档后必须 `grep` 验证关键内容未丢失；`replaceAll` 比逐行替换更安全但需确认变换范围。

5. **覆盖率高不等于行为正确——集成测试暴露语义偏差** — 单元测试覆盖率 92.4%，但集成测试（tools/call 全链路）暴露了两个问题：(a) `spec_validate` 在参数全空时 `isError=true`（预期是结构化输出），原因是 go-sdk 的 isError 设置逻辑与预期不同；(b) `process_list` 对不存在的设备返回空列表而非错误（与 spec US4-AS2 不一致）。这些在纯单元测试中完全不可见。

6. **benchmark 不是"跑一下就行"——goroutine 泄漏会累积副作用** — `BenchmarkServerStartup` 初版用 `go server.Run()` 启动 server 后未等待退出，导致每次迭代泄漏一个 goroutine，产生 400+ ERROR 日志行的级联错误。Go benchmark 框架会按 `b.N` 放大每次迭代的代价，必须在迭代内完成完整生命周期（start → use → clean up）。修复：加 `sync.WaitGroup` 在每轮迭代内等待 server goroutine 退出。

7. **教学文档章节命名应以知识点为准，非 Phase** — 此教训在 M3 已提出，M4 进一步验证。Phase 2 的 §1.4 最初命名为"项目实际代码示例 (Phase 2)"，后被重构为"go-sdk Tool 注册：泛型 + 反射协同模式"——这个标题本身就是知识点。代码示例应嵌入相关知识点章节，而非集中在一个"Phase N 项目代码" dump。

8. **外部 SDK 的默认行为需要实际跑过才知道** — go-sdk 的 `StdioTransport` 在 stdin close 时立即返回 EOF error——这与预期"等待所有请求完成"不同。实际测试中 server 在收到最后一个请求后直接退出，导致客户端读不到响应。解决方案是不在测试中立即 close stdin，而是用 sleep/cat 保持管道开启。——此经验对任何依赖外部 SDK 的场景都适用。

---

## M5：靶机 1 — 证书锁定对抗 Harness

| 维度 | 内容 |
|------|------|
| **Go 知识** | `crypto/tls` TLS 连接、`crypto/x509` 证书解析、`tls.Config` 自定义、`net/http` RoundTripper |
| **逆向知识** | SSL Pinning 全体系：OkHttp `CertificatePinner.check()`、`TrustManager.checkServerTrusted()`、`WebViewClient.onReceivedSslError()`、Native SSL_verify、Network Security Config |
| **产出** | 靶机 APK + FridaForge spec + 自动脱绑脚本 |

---

## M6：靶机 2 — Hybrid App WebView 逆向 Harness

| 维度 | 内容 |
|------|------|
| **Go 知识** | WebSocket 通信、channel pipeline 数据处理、`bufio.Scanner` |
| **逆向知识** | WebView 逆向全景：`addJavascriptInterface()` Bridge、`shouldInterceptRequest()` Hook、`evaluateJavascript()` 注入、Chrome DevTools Protocol 联动、`@JavascriptInterface` 注解 |
| **产出** | Hybrid App 靶机 + FridaForge spec + WebView Hook 脚本 |

---

## M7：发布打磨与开源

| 维度 | 内容 |
|------|------|
| **Go 知识** | `goreleaser` 跨平台编译发布、`go test -race` 竞态检测、`pprof` 性能分析、GitHub Actions CI/CD |
| **逆向知识** | 复盘总结整个知识体系 |
| **产出** | CHANGELOG、`docs/learn/` 合集、GitHub Release、公开文章素材 |

---

## SpecKit 工作流标准执行模板（每个 Milestone 重复）

```
M[x] 启动
  │
  ├─ 阶段 A：SpecKit 规划（固定顺序，严格执行）
  │   ├─ 1. /speckit.specify  → 用户定义功能需求 (spec.md)
  │   ├─ 2. /speckit.clarify  → AI 找出边界漏洞和模糊点
  │   ├─ 3. /speckit.plan     → 技术架构与接口契约 (plan.md + research.md + data-model.md + contracts/)
  │   ├─ 4. /speckit.tasks    → 拆解为 Task 清单 (tasks.md)
  │   └─ 5. /speckit.analyze  → 交叉验证 spec/plan/tasks 一致性（实现前）
  │
  ├─ 阶段 B：教学准备（实现前）
  │   └─ 6. 产出教学文档初始版 docs/learn/M[x]-*.md
  │      （用独立迷你代码示例讲解核心概念，三轨齐全）
  │
  └─ 阶段 C：实现（每个 Phase = 讲解 → 编码 → 补充学习文档 → Commit）
      └─ 7. /speckit.implement
          ├─ Phase N 讲解（该 Phase 涉及的新概念，独立示例先行）
          ├─ Phase N 编码（按 tasks.md 逐 Task 执行）
          ├─ Phase N 补充教学文档（在对应章节追加项目真实代码示例）
          ├─ Commit（逻辑相关的 1-3 个 Task 可合并，同文件 Task 必须合并）
          └─ 学员确认 → 继续下一 Phase
      │
      └─ 8. Milestone 收尾
          ├─ /speckit.analyze（实现后再次交叉验证，检查代码 vs 文档一致性）
          ├─ 更新 milestones.md 本阶段实际产出物
          ├─ 更新教学文档状态为"已完成"
          ├─ Review & commit
          └─ 合并到主分支
```
