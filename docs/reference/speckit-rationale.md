# SpecKit 各阶段设计哲学

> 本文档是宪法 6.2 节引用的补充学习材料。不建议在写代码时逐字参考，
> 而是作为"为什么 SpecKit 要这样设计"的认知基底，帮助你建立 AI 编程的判断力。

---

## 0. 前置概念：AI 编程范式对比

### Vibe Coding（感觉编码）
**定义：** 不给 AI 任何结构化上下文，凭一句模糊需求直接生成代码。

**致命缺陷：**
- 不可复现——同一句话第二次得到的代码完全不同
- 不可审查——你不知道 AI 为什么生成那段代码
- 不可扩展——功能一复杂，AI 开始脑补、相互矛盾

**结论：** Vibe Coding 适合"写个一次性脚本"，不适合"构建一个开源工程"。

### Spec Coding（规范驱动开发 — SpecKit 的范式）
**定义：** 先写形式化的需求文档（Spec），再让 AI 按文档生成代码。

**核心理念：** "Specification as executable contract"——规范本身就是可执行的契约。

**优势：**
- 可复现——Spec 不变，代码方向不变
- 可追溯——Spec/Plan/Tasks 三级文档链完整记录决策过程
- 可验证——Tasks 是 checklist，做没做完一目了然

**与 SpecKit 的关系：** SpecKit 不是发明了 Spec Coding，而是把"生成 Spec/Plan/Tasks 文档"这件事工程化、模板化、可复用了。没有 SpecKit 你也能手写这些文档，但 SpecKit 提供了标准模板和检查流程，让 AI 的生成质量更高、一致性更强。

### TDD（测试驱动开发）
**定义：** 先写测试用例，再写最少代码让测试通过。

**本项目中的应用：** 不作为全局范式，但在 M3（codegen）和 M4（mcpserver）阶段，对核心函数强制使用 table-driven tests。

### Harness Engineering（测试脚手架工程）
**定义：** 为被测系统搭建模拟环境，让工具在"类生产"条件下持续接受验证。

**本项目中的应用：** M5（CertPinning 靶机）和 M6（Hybrid App 靶机）。没有真实靶机，FridaForge 的正确性无法验证。

---

## 1. `/speckit.constitution` — 为什么"先有宪法"？

**它在做什么：** 为整个项目建立不可动摇的治理规则——代码风格、安全边界、架构约束。

**为什么必须有：** AI 生成的代码默认没有"记忆"。你今天说"用 Unix 风格的错误处理"，AI 明天可能忘掉。Constitution 作为项目级的持久化上下文，每次被读取，约束 AI 的输出边界。

**如果不做会怎样：** 你会在 M2 看到 goroutine 裸开（没有 context），在 M3 看到代码没有 go doc comment，在 M5 看到 Frida Hook 没有 Cleanup——因为 AI 不知道你的项目有这些"潜规则"。

**FridaForge 示例：** 宪法第 4.3 条规定"MCP Server 默认监听 localhost"。以后每次 `/speckit.implement`，AI 都会遵守这个约束，不会脑补成 `0.0.0.0:8080`。

---

## 2. `/speckit.specify` — 为什么"先定义 What，禁止聊 How"？

**它在做什么：** 用自然语言描述功能需求、用户故事、验收条件。严格禁止在 Spec 中写技术实现细节。

**为什么必须有：** 如果你在定义需求时同时夹带技术方案（如"用 goroutine pool 管理 Frida 设备"），你就把技术选型锁死了。后续如果发现更好的方案（比如用 actor model），已经改不动——因为需求里早写死了。

**Spec 隔离 What（需求不变），Plan 负责 How（方案可变）。**

**如果不做会怎样：** "Vibe Coding"式陷阱：你对 AI 说"用 goroutine 管理多个设备"，AI 会直接写代码——但你没告诉它异常处理策略、超时策略、设备发现策略。AI 会自行脑补，结果偏差巨大。

**FridaForge 示例：** M2 的 Spec 会写："系统应支持同时连接最多 4 个 Android 设备，每个设备可运行多个 Hook 会话，会话间互不干扰。"——这里没有出现 `sync.WaitGroup` 或 `channel`，因为那是 Plan 阶段的事。

---

## 3. `/speckit.clarify` — 为什么需要"AI 来审问我"？

**它在做什么：** AI 阅读你的 Spec，逐项找出模糊点、矛盾点、遗漏点，用结构化问卷的方式向你提问。

**为什么必须有：** 人类在写需求时天然会遗漏边界情况。例如你写"Hook 目标 App 的 `encrypt` 方法"，但你忘了说：① 如果 App 有重载（overload）怎么办？② 如果 `encrypt` 在 Native 层怎么办？③ 如果 App 进程不存在怎么办？

Clarify 是**需求的质量门**——在投入工程之前补齐盲区。

**如果不做会怎样：** 实现到一半才发现 Spec 有重大遗漏 → 回头改 Spec → Plan 也要改 → Tasks 也要改 → 大量返工。这就是"Vibe Coding 永远在做返工"的根本原因。

---

## 4. `/speckit.plan` — 为什么先定接口契约再写代码？

**它在做什么：** 定义技术架构——Go 包结构、interface 接口定义、数据流向、第三方库选型。产出 `contracts/` 目录（接口契约）。

**为什么必须有：** 接口契约是模块并行开发的通信协议。如果你不先定义 `DeviceManager` 的 interface 就直接写实现，那么写 CLI 的人（或同一批 AI）不知道该怎么调用设备管理器。定义了 interface 后，两个模块可以并行开发——调用方只看 interface，不关心实现。

**如果不做会怎样：** 模块间耦合失控——CLI 直接操作 Frida 的内部 session 对象，导致 CLI 和 Frida engine 无法解耦。以后换一个 Frida 版本，所有模块都要改。

---

## 5. `/speckit.tasks` — 为什么拆成 Checkbox 清单？

**它在做什么：** 把 Plan 中的架构拆解为**有依赖关系**的可执行任务清单。每个 Task 是原子化的——"改一个文件，加一个功能，跑一个测试"。

**为什么必须有：**
- **可追踪：** 项目进度用 Checkbox 数量衡量（"完成 12/15"），不是凭感觉说"做得差不多了"
- **可回滚：** 如果 Task 3 的代码写错了，你只需要 revert Task 3 的 commit，不影响 Task 1-2
- **防遗漏：** "定义 error types"、"写单元测试"、"更新 go doc" 这些容易被忽略的边缘任务，都被显式列出来

**如果不做会怎样：** AI 一口气生成 500 行代码——你不知道哪些是对的你留下了，哪些是错的你改了，也分不清这是第几个功能点。最后变成一锅粥。

---

## 6. `/speckit.analyze` — 需要 AI "自我审查"什么？

**它在做什么：** 交叉比对 Spec（需求）、Plan（架构）、Tasks（任务）三份文档，检查一致性。

**为什么必须有：** 这是 Spec Coding 的质量保险。你在迭代修改 Spec/Plan/Tasks 的过程中，三份文档可能产生漂移。Analyze 是最后一道检查——确认"我们即将实现的东西，确实是用户要求的东西"。

**FridaForge 示例：** Spec 要求支持 3 种 Hook 类型（Overload、Override、Native），Plan 只设计了 2 种代码模板，Tasks 也只拆了 2 个。Analyze 会报："Spec Native Hook 缺少 Plan 和 Tasks 覆盖。"

---

## 7. `/speckit.implement` — 为什么"到了一切就绪才能写代码"？

**它在做什么：** 按 Tasks 清单顺序，逐项生成代码、写测试、跑验证。每完成一个 Task，做一次 Git Commit。

**为什么放在最后：** 前置的 5 个阶段（Constitution → Specify → Clarify → Plan → Tasks → Analyze）本质上是一个**逐步收敛的信息漏斗**——需求从模糊到清晰，架构从抽象到具体，任务从粗到细。到了 Implement 阶段，AI 已拥有完整上下文：需求文档、架构设计、接口契约、任务清单。这时的代码生成质量远高于"一个 prompt 直接出代码"。

**为什么每完成一个 Task 就 Commit：**
- 最小可回滚单元——如果 T05 的代码引入 bug，你可以回退到 T04 而不是全部重来
- 可审查——导师可以逐 commit 审查你的代码
- 学习颗粒度——每个 Task ≈ 一个知识点，你学一个就 Commit 一个

---

## 总结：一张图看懂 SpecKit 工作流

```
Vibe Coding（不推荐）:
  "帮我写个工具" → AI 扔一堆代码 → 跑两下报错 → 修修补补 → 不了了之

Spec-Driven（本项目）:
  Constitution → Specify → Clarify → Plan → Tasks → Analyze → Implement
     ↑              ↑         ↑        ↑        ↑        ↑         ↑
   定规则        定需求    补盲区   定架构   拆任务   查一致性   写代码
     └───────────────── 信息逐步收敛的漏斗 ──────────────────┘
```
