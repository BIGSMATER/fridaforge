# FridaForge — 全局 Milestone 计划

> 本文档记录项目从 M0 到 M7 的完整里程碑路线图。每个 Milestone 的执行严格遵循 SpecKit 工作流。

## 当前状态：M2 完成 → 准备进入 M3

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

## M2：Frida 并发调度引擎（`fridaengine`）

| 维度 | 内容 |
|------|------|
| **Go 知识** | **并发核心：** goroutine、`sync.WaitGroup`、`context.WithTimeout/Cancel`、`sync.Mutex/RWMutex`、channel 生产-消费者模式；**工程设计：** `interface` 抽象 (`DeviceManager`/`SessionManager`)、依赖注入、错误包装 (`%w`) |
| **逆向知识** | Frida 完整生命周期：`enumerate_devices()` → `attach()` → `create_script()`；`frida-server` 部署；USB vs 网络远程管理 |
| **Harness** | 最小 Android App（Hello World 方法），验证 Attach + 调用方法 |

---

## M3：声明式代码生成器（`codegen`）

| 维度 | 内容 |
|------|------|
| **Go 知识** | `text/template` 模板渲染、`embed.FS` 内嵌文件、`strings.Builder`、`go generate`、`os/exec` |
| **逆向知识** | Frida JS API 深度：`Java.perform()`, `Java.use()`, `.implementation =`, `this.xxx()` 原方法调用；Hook 类型模板化 (Override/Overload/Native) |
| **Harness** | 扩展 M2 的测试 App（加 Native 函数），验证生成脚本正确性 |

---

## M4：MCP Server 集成（`mcpserver`）

| 维度 | 内容 |
|------|------|
| **Go 知识** | `net/http.Server`、`encoding/json` 自定义序列化、middleware 链模式、`log/slog` 结构化日志 |
| **AI 范式** | MCP 协议（JSON-RPC 2.0 + Streamable HTTP）、Tool/Resource/Prompt 设计哲学、LLM 如何通过 MCP 调用工具 |
| **Harness** | Claude Desktop 连接 MCP Server，让大模型自动生成 Hook 脚本 |

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
