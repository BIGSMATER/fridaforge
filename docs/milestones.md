# FridaForge — 全局 Milestone 计划

> 本文档记录项目从 M0 到 M7 的完整里程碑路线图。每个 Milestone 的执行严格遵循 SpecKit 工作流。

## 当前状态：M0 完成 → 准备进入 M1

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

## M1：CLI 骨架与声明式配置解析 ← 当前阶段

| 维度 | 内容 |
|------|------|
| **SpecKit 流程** | `/speckit.specify` → `clarify` → `plan` → `tasks` → `analyze` → `implement` |
| **Go 知识** | **基础语法：** `package`/`import`、`struct` + JSON tag、`func` 与方法接收者、`if err != nil`、`slice`/`map`、`fmt`、`os.Args`；**CLI 框架：** `cobra.Command` 树形命令注册、`viper` 配置绑定、`yaml.v3` 反序列化 |
| **逆向知识** | YAML Spec 的逆向语义：`className` → Dalvik 类全限定名、`methodName` → ART 方法签名格式、`hookType` → `overload`/`replace` 差异 |
| **目标产出** | `cmd/fridaforge/` (CLI 入口完整), `pkg/config/`, `pkg/spec/` |

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
  ├─ 1. 导师讲解本阶段 Go + 逆向知识点 → docs/learn/M[x]-*.md
  │
  ├─ 2. /speckit.specify  → 用户定义功能需求 (spec.md)
  ├─ 3. /speckit.clarify  → AI 找出边界漏洞和模糊点
  ├─ 4. /speckit.plan     → 输出技术架构与接口契约 (plan.md + contracts/)
  ├─ 5. /speckit.tasks    → 拆解为 Checkbox 任务清单 (tasks.md)
  ├─ 6. /speckit.analyze  → 交叉验证 spec/plan/tasks 一致性
  └─ 7. /speckit.implement → 用户确认后逐项编写代码 (每 Task 一 Commit)
```
