# M2 回顾：Frida 并发调度引擎

> 本文件记录 M2 执行过程的文件职责、设计决策和经验教训，供 M3+ 参考。

---

## 一、文件清单与设计原理

### `pkg/fridaengine/errors.go` — 三层错误类型

**实现**：`DeviceError` / `SessionError` / `ScriptError`，各含 `Error()` + `Unwrap()`。

**为什么三层？** 对应 Frida 调用链的三个层级——设备操作失败、会话失败、脚本失败。调用者可以 `errors.As` 判断是哪一层出问题。为什么不用单一 `EngineError` + 错误码？Go 的惯用方式是类型化错误——`Unwrap()` 支持 `errors.Is` 链式解包，错误码做不到这点。

### `pkg/fridaengine/session.go` — HookSession 状态机 + 消息类型

**实现**：`SessionState` (iota 枚举: Created→Ready→Detached)、`HookMessage`、`ProcessInfo`、`HookSession` 结构体（含 RWMutex、msgCh、onRemove 回调）。

**为什么用 iota 而不用 M1 的 string enum？** `SessionState` 只在 Go 内部流转，不需要 YAML 序列化。int 枚举比 string 比较更快（`==` 是单条 CPU 指令 vs 逐字节比较）。

**为什么 RWMutex 分离读写？** `State()` 被高频查询（监控状态），`setState()` 低频写入。读锁允许多个 goroutine 同时读——如果读写都用 Mutex 则所有 State() 串行化。

**为什么 onRemove 回调？** 直接 `session.Detach()` 时通知 SessionManager 从 sessions map 删除——否则 ActiveSessions() 返回过期数据。最初遗漏了这个机制（analyze 阶段发现）。

### `pkg/fridaengine/device.go` — FridaDeviceLister

**实现**：实现 M1 的 `DeviceLister` 接口。包装 `frida.DeviceManager`，goroutine+select 接入 context，过滤 Local 设备，DeviceType→ConnectType 映射。含编译时接口检查 `var _ DeviceLister = (*FridaDeviceLister)(nil)`。

**为什么单独一个包？** 接口在 `pkg/device/`（纯数据模型），实现在 `pkg/fridaengine/`（依赖 frida-go CGO）。分离避免核心数据结构被 CGO 污染。

**为什么 goroutine+select？** `EnumerateDevices()` 是 CGO 同步调用——不内置 context 支持。goroutine 中执行阻塞调用，select 同时监听 ctx.Done() 和完成信号，实现超时控制。

**映射关系纠正**：最初以为 frida-go 有 4 种 DeviceType（Local/Remote/USB/Network），实际只有 3 种（Local/Remote/USB）。Remote → ConnectTypeNetwork，USB → ConnectTypeUSB。

### `pkg/fridaengine/script.go` — HookScript 包装

**实现**：包装 `frida.Script`，`Load()` / `Unload()` / `onMessage()`。内部类型——不直接暴露给调用者，通过 HookSession 间接使用。

**为什么 internal？** 脚本没有独立生命周期——永远依附于 Session。暴露独立的 HookScript API 会导致调用者困惑"脚本什么时候释放"。

**为什么 onMessage 用 `chan<-` 单向 channel？** 宪法 2.4 要求 channel 方向明确。回调只写不读，签名上就表达出来。

### `pkg/fridaengine/engine.go` — Engine 入口

**实现**：`NewEngine` / `NewEngineWithDefaults`、`ListDevices`（委托 DeviceLister）、`Attach`（委托 SessionManager）、`Close`（委托 DetachAll）、`EnumerateProcesses` / `EnumerateApplications`（goroutine+select 包装 + ScopeFull）、`ActiveSessions`。

**为什么 Engine 是一层薄代理？** 单一职责——Engine 是"门面"，不自己做 Attach 逻辑。最初 Engine 自己管理 `frida.DeviceManager` 和 Attach 流程，Phase 5 重构为委托 SessionManager。这样 Close() 可以统一遍历清理。

**为什么用私有 `findDevice()` helper？** 三个方法（Attach/EnumerateProcesses/EnumerateApplications）都需要根据 deviceID 找 frida 设备。抽出来避免三个地方各自写一遍 EnumerateDevices 循环。

### `pkg/fridaengine/manager.go` — SessionManager 并发调度

**实现**：Mutex 保护 sessions map、软上限 64（超限 Warn 不拒绝）、Attach（goroutine+select + onRemove 回调设置）、DetachAll（先拷贝 map → goroutine 并发 Detach → 错误收集）。

**为什么先拷贝 map 再遍历？** Go map 不是并发安全的——遍历时插入直接 fatal panic。在锁保护下拷贝一份 slice，释放锁后再遍历拷贝。

**为什么错误收集用独立 Mutex？** 每个 goroutine 的 `append(errs, err)` 不是线程安全的。用一个独立 `mu sync.Mutex` 保护 errs slice，避免和 sessions map 的锁冲突。

**为什么不用 errgroup？** errgroup 默认"一失败全取消"——不符合 spec 的"独立运行，聚合所有错误"策略。标准库 WaitGroup + Mutex 足够灵活。

**WaitGroup 教训**：最初给 struct 加了 `wg sync.WaitGroup` 追踪活跃 session，但 `Attach` 的 `wg.Add(1)` 和直接 `Detach` 的 `wg.Done()` 不对称 → `negative WaitGroup counter` panic。去掉后纯用 sessions map 计数，由 onRemove 回调维护。

### `pkg/fridaengine/integration_test.go` — 集成测试骨架

**实现**：`//go:build integration` 条件编译，完整生命周期测试（ListDevices→Attach→CreateScript→收消息→Detach→幂等 Detach）。

**为什么用 build tag 而不是 t.Skip？** t.Skip 仍会编译测试代码，如果 CI 环境缺 frida-core-devkit 会编译失败。build tag 从编译阶段排除。

---

## 二、经验教训

### 2.1 有效模式

**1. 即讲即写（M1 验证，M2 强化）**  
每个 Phase 先讲概念（context 讲了一整个章），再写代码（立刻在 device.go 里看到 goroutine+select）。学员说不懂 channel → 展开讲 → 懂了 → 继续。对比"全部讲完再全部写"效率高得多。

**2. 项目代码作为教学材料主体**  
教学文档 §1.3 讲到 context.Done() 时直接引用 device.go 的 `select { case <-ctx.Done(): ... }`。学员看完讲解立刻在项目代码里看到实例——理解当场锚定。

**3. 两次 analyze 各抓不同层次的问题**  
- 第一次（实现前）：发现 spec/plan/tasks 不一致（Manager vs SessionManager 命名）  
- 第二次（实现后）：发现 ActiveSessions 不准、默认超时缺失、nil session panic

第一次抓设计矛盾，第二次抓实现偏离。

**4. 接口分离的现实价值**  
M1 的 `DeviceLister` 接口 + `StubDeviceLister` 在 M2 直接收获红利——新写一个 `FridaDeviceLister` 实现接口，CLI 代码一行不改。

**5. CGO 依赖的隔离**  
frida-core-devkit 387MB、需要版本匹配、/tmp 被清理——这些痛苦只影响 `pkg/fridaengine/`。M1 的 `pkg/device/` / `pkg/spec/` / `pkg/config/` 完全不受影响。

### 2.2 避坑清单

**1. 不要忘记 Phase 教学文档更新**  
Phase 3/5/6 都忘了立刻补教学文档——用户提醒才补。正确做法：代码写完 → 立刻在对应章节追加项目实际代码示例 → 然后 commit。

**2. CGO 环境变量要用绝对路径**  
最初 Makefile 用 `FRIDA_DEVKIT = .devkit`（相对路径）。`go build` 的 CGO 编译器工作目录不在项目根，找不到文件。改为 `$(CURDIR)/.devkit` 绝对路径。

**3. WaitGroup 的 Add/Done 不对称是致命 bug**  
SessionManager 的 `wg.Add(1)` 在 Attach 中，`wg.Done()` 在 DetachAll 中。但用户可能直接 `session.Detach()`——Done 不会被调用 → `negative WaitGroup counter` panic。教训：WaitGroup 配对必须严格在同一生命周期作用域。

**4. frida-go 的 DeviceType 不是文档写的那样**  
最初以为有 4 种：Local/Remote/USB/Network。实际只有 3 种：Local/Remote/USB。Network 或 socket 设备是 Remote type 的子类。直接看源码（`types.go`）而非靠文档猜测。

**5. frida-core-devkit 版本匹配**  
frida CLI 是 16.5.2，但 frida-go v1.0.2 需要 devkit 17.9.8。版本不一致 → 编译错误 `could not determine what C.FridaBundleFormat refers to`。devkit 必须和 frida-go release 匹配，和本地 frida CLI 版本无关。

**6. 不要一次推完所有 Task**  
M1 犯过——M2 严格按 Phase 停顿等确认。每个 Phase 结束 → 展示完成状态 → 用户说继续 → 才到下一 Phase。这避免了"方向偏了才发现"的浪费。

### 2.3 Go 学习曲线观察

| 概念 | 学员初始状态 | 教学方式 | 理解拐点 |
|------|-----------|---------|---------|
| context | "没概念" | 从 goroutine 泄漏讲起 → 树模型 → 构造函数 → Done/Err → 项目代码 | 看到 `goroutine+select` 在 device.go 的实际用法 |
| channel | "没学过" | 从共享内存问题讲起 → 管道模型 → 缓冲 vs 无缓冲 → range → 方向 → 项目数据流图 | 问"为什么缓冲 64？"——理解了生产消费速度差 |
| 锁 | "为什么要用到锁" | data race 现场模拟 → 锁修复 → RWMutex 为什么 → 为什么不用 channel 替代 | 看到并发 Detach → 重复关 channel panic 的例子 |
| WaitGroup | 基本理解 | 独立示例 → 项目中为什么没用到（session goroutine 在 frida C 线程里） | 理解了"不是我启动的 goroutine 我不管 WaitGroup" |

### 2.4 AI 协作模式确认

M2 验证了 M1 的模式：**SpecKit 规划先行 → 实现阶段讲解编码交替**。关键不是 AI 写了多少代码，而是 AI 在学员困惑时即时展开讲解的能力——学员说"我没学过 channel"，立刻停下手头代码，用独立示例+项目代码双重锚定讲清楚，再继续。

M3 应继续此模式，但增加一条：**教学文档更新不应等学员提醒**——每个 Phase 完成就补充。
