# M2 学习笔记：Go 并发模型与 Frida 调度引擎

> Milestone: M2 | 状态: 已完成
> 三轨并行：Go 语言 / Android 逆向 / AI 编程范式
> 前置：本阶段假设学员已掌握 M1 内容（package/import、struct、error 处理、cobra 命令树）

---

## 一、Go 语言轨道：并发编程核心

M2 引入 Go 最重要的语言特性——**并发编程**。Go 的并发模型围绕三个核心原语构建：goroutine（轻量线程）、channel（通信管道）、context（生命周期控制）。

### 1.1 goroutine — Go 的"线程"

goroutine 是 Go 运行时的**用户态轻量线程**。启动一个 goroutine 只需 `go` 关键字：

```go
package main

import (
    "fmt"
    "time"
)

func say(msg string) {
    for i := 0; i < 3; i++ {
        fmt.Println(msg, i)
        time.Sleep(100 * time.Millisecond)
    }
}

func main() {
    go say("goroutine")   // 启动新 goroutine（并发）
    say("main")           // 当前 goroutine 继续执行
    // 输出交织：main 0, goroutine 0, main 1, goroutine 1, ...
}
```

**关键理解**：
- goroutine 不是 OS 线程——Go 运行时把成千上万个 goroutine 多路复用（multiplex）到少量 OS 线程上
- 启动成本极低（~2KB 栈），可以同时跑数万个
- `go f()` **不等待** f() 完成就继续执行下一行——这就是为什么需要 WaitGroup

### 1.2 sync.WaitGroup — 等待 goroutine 完成

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

func worker(id int, wg *sync.WaitGroup) {
    defer wg.Done()          // 任务完成时计数 -1
    fmt.Printf("Worker %d 开始\n", id)
    time.Sleep(500 * time.Millisecond)
    fmt.Printf("Worker %d 完成\n", id)
}

func main() {
    var wg sync.WaitGroup

    for i := 1; i <= 3; i++ {
        wg.Add(1)            // 计数 +1
        go worker(i, &wg)    // 传指针——值拷贝会导致每个 goroutine 持有自己的 WaitGroup 副本
    }

    wg.Wait()                // 阻塞直到计数归零
    fmt.Println("全部完成")
}
```

**M2 中的用法**：`SessionManager` 用 `WaitGroup` 追踪所有活跃的 Attach goroutine。`Close()` 时调用 `Wait()` 等待所有 goroutine 退出后再释放资源。

**宪法 2.4 约束**：禁止用 `time.Sleep` 等待 goroutine——必须用 `sync.WaitGroup` 或 channel。

### 1.3 context.Context — 超时、取消、值传递

#### 1.3.1 为什么需要 context？—— 没有 context 的世界

想象你写了一个函数去连接远程设备：

```go
func connectToDevice(ip string) error {
    conn, err := net.Dial("tcp", ip+":27042")  // 连接 frida-server
    if err != nil {
        return err
    }
    // ...
}
```

**问题来了**：如果 `net.Dial` 因为网络故障永远不返回怎么办？你的程序就永远卡在这里——这叫 **goroutine 泄漏**。你肯定想加个超时：

```go
func connectToDevice(ip string, timeout time.Duration) error { ... }
```

好，超时有了。但接下来还有新需求：
- 用户按了 Ctrl+C → 需要立即取消所有正在进行中的连接
- HTTP 请求的客户端断开 → 后端应该停止处理
- 父任务失败 → 所有子任务都应该停止

**这就是 context 要解决的问题：在 goroutine 之间传播取消信号、超时和请求范围的值。**

#### 1.3.2 核心心智模型：一棵倒挂的树

```
                context.Background()  ← 根（永不取消）
                     │
          ┌──────────┼──────────┐
          │          │          │
     WithTimeout  WithCancel  WithValue
      (30s 超时)   (手动取消)   (携带 userID)
          │          │
     ┌────┴────┐     │
     │         │     │
   goroutine-A  goroutine-B  goroutine-C
```

**关键规则**：
1. **取消向下传播** — 取消父 context → 所有子 context 自动取消
2. **取消不向上传播** — 子 context 取消不影响父 context
3. **context 不可变** — 每次 `With*` 返回新 context，原 context 不变

#### 1.3.3 四个构造函数

**`context.Background()`** — 一切的根，永远不会被取消，没有超时，没有值。main goroutine 应该用这个。

**`context.WithCancel()`** — 手动控制取消：

```go
ctx, cancel := context.WithCancel(context.Background())

go func() {
    select {
    case <-ctx.Done():          // ctx 被取消时，这个 channel 会关闭
        fmt.Println("被取消了")
        return
    }
}()

cancel()  // 我决定取消 —— 所有监听 ctx.Done() 的 goroutine 都会收到信号
```

类比：就像公司广播——老板按一下按钮（`cancel()`），所有人同时听到（`ctx.Done()` channel 关闭）。

```go
// 更真实的例子：启动多个 worker，一个出错全部取消
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

for _, worker := range workers {
    go func(w Worker) {
        err := w.Do(ctx)   // 每个 worker 都监听同一个 ctx
        if err != nil {
            cancel()       // 任何 worker 出错 → 取消所有其他 worker
        }
    }(worker)
}
```

**`context.WithTimeout()`** — 倒计时自动取消：

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()  // 重要！即使超时前完成也要调用 cancel，释放 timer 资源

select {
case <-ctx.Done():
    fmt.Println(ctx.Err())  // context.DeadlineExceeded
case result := <-doWork():
    fmt.Println(result)
}
```

底层实现：`WithTimeout` 等价于 `WithDeadline(time.Now().Add(timeout))`。Go 内部启动一个 timer goroutine，到期后自动调用 cancel。

**为什么总是 `defer cancel()`？** 即使任务在超时前完成，timer 还活着——`defer cancel()` 确保 timer 被停止，避免 goroutine 泄漏。

**`context.WithValue()`** — 携带请求范围的值（⚠️ 慎用）：

```go
type contextKey string
const userIDKey contextKey = "userID"

ctx := context.WithValue(context.Background(), userIDKey, "user-123")

// 在深层函数中取回
func handleRequest(ctx context.Context) {
    userID, ok := ctx.Value(userIDKey).(string)  // 返回 interface{}，需要类型断言
    if !ok {
        return
    }
    fmt.Println("处理用户:", userID)
}
```

context.Value 的类型安全很差，只适合携带请求范围的元数据（trace ID, request ID），**不要把业务数据塞进 context**。

#### 1.3.4 `ctx.Done()` — 核心机制

`ctx.Done()` 返回一个 **只读 channel**（`<-chan struct{}`）。当 context 被取消或超时，这个 channel 会被**关闭**。

```go
// 错误用法 ❌
if ctx.Done() == nil { ... }  // Done() 返回 channel 本身，不是 bool

// 正确用法 ✅
select {
case <-ctx.Done():           // channel 关闭 → 立即收到零值
    return ctx.Err()         // 返回取消原因
default:
    // 还没取消，继续工作
}
```

**为什么是 channel 关闭而不是发送值？** 因为一个 context 可能被数万个 goroutine 监听——关闭 channel 会**同时唤醒所有**监听者（Go channel 的广播语义），而发送值只能唤醒一个。

#### 1.3.5 `ctx.Err()` — 区分取消原因

```go
select {
case <-ctx.Done():
    err := ctx.Err()
    if errors.Is(err, context.Canceled) {
        // 手动 cancel() 触发的
    }
    if errors.Is(err, context.DeadlineExceeded) {
        // 超时触发的
    }
}
```

#### 1.3.6 项目核心模式：goroutine + select 包装阻塞调用

这是 FridaForge `device.go` 中使用的模式——Go 处理 CGO/网络 IO 超时的**标准套路**：

```go
func blockingCallWithContext(ctx context.Context) (result, error) {
    done := make(chan struct{})
    var res resultType
    var callErr error

    go func() {
        defer close(done)
        res, callErr = SomeBlockingCOrNetworkCall()  // CGO 或 网络 IO
    }()

    select {
    case <-ctx.Done():
        return nil, ctx.Err()   // 超时 → 返回错误
    case <-done:
        // 正常完成 → 注意：goroutine 可能还在后台残留
    }
    return res, callErr
}
```

**为什么用 `done` channel 而不是 WaitGroup？** WaitGroup 用于追踪多个 goroutine，而这里只需要等待一个结果——用 channel 更轻量。

**局限性**：如果 `SomeBlockingCall` 是纯 C 调用（没有超时机制），即使 ctx 取消，C 调用仍在执行，直到它自己返回。Go 的 context 取消**不能强行终止 C 函数**——这是 CGO 的固有限制。frida-go 的 `EnumerateDevices()` 内部有超时机制，所以可以接受。

#### 1.3.7 项目实际代码

```go
// pkg/fridaengine/device.go — 你写的代码
func (l *FridaDeviceLister) ListDevices(ctx context.Context) ([]device.Device, error) {
    done := make(chan struct{})
    var fridaDevices []frida.DeviceInt
    var enumerateErr error

    go func() {
        defer close(done)                              // 3. goroutine 完成 → 关闭 done channel
        fridaDevices, enumerateErr = l.mgr.EnumerateDevices()  // 1. CGO 阻塞调用
    }()

    select {
    case <-ctx.Done():                                 // 4a. 调用者取消了 → 返回超时错误
        return nil, fmt.Errorf("device enumerate: %w", ctx.Err())
    case <-done:                                       // 4b. 枚举完成 → 处理结果
    }
    // ...
}
```

**执行流程**：
1. 主 goroutine 启动一个子 goroutine 去执行 `EnumerateDevices()`（CGO 阻塞调用）
2. 主 goroutine 用 `select` 同时等待两个信号：
   - `ctx.Done()`：调用者说"不等了"（超时或手动取消）
   - `done`：子 goroutine 说"我完成了"
3. 无论谁先到，另一个就不等了

**常见错误** ❌：
```go
// 错误：忘记 defer cancel() —— timer 泄漏
ctx, _ := context.WithTimeout(parent, 30*time.Second)

// 错误：把 context 存到 struct 里
type MyStruct struct {
    ctx context.Context  // 不要这样！context 应该作为函数第一个参数传递
}
```

#### 1.3.8 一句话总结

> **context 是 Go 的"取消信号传播树"**。它不杀人（不能强行终止 goroutine），它只是发信号——收到信号的 goroutine 应该自觉返回。真正的取消工作由 goroutine 自己在 `case <-ctx.Done()` 里做。

### 1.4 sync.Mutex — 保护共享数据

#### 1.4.1 为什么需要锁？—— 不加锁的 bug

```go
type HookSession struct {
    state SessionState  // 多个 goroutine 同时读写
}

func (s *HookSession) Detach() error {
    if s.state == SessionStateDetached {  // goroutine A 读
        return nil
    }
    s.state = SessionStateDetached        // goroutine B 也在写
    close(s.msgCh)
}
```

两个 goroutine 同时调 `Detach()`：

```
时间 →  goroutine A                  goroutine B
       读 state → Created
                                     读 state → Created
       写 state → Detached
                                     写 state → Detached
       关闭 channel                  关闭 channel ← PANIC! 重复关闭
```

**这叫 data race**——两个 goroutine 同时访问同一块内存，至少一个是写。结果不可预测。

#### 1.4.2 锁做了什么

```go
func (s *HookSession) Detach() error {
    s.mu.Lock()         // 加锁——B 在这阻塞
    defer s.mu.Unlock()

    if s.state == SessionStateDetached {
        return nil      // B 进来时 state 已经是 Detached，直接返回
    }
    s.state = SessionStateDetached
    close(s.msgCh)
}
```

```
时间 →  goroutine A                  goroutine B
       Lock() 拿到锁
       读 state → Created           Lock() 阻塞等待...
       写 state → Detached
       关闭 channel                  ...还在等
       Unlock()                      Lock() 拿到锁
                                     读 state → Detached
                                     直接 return（幂等）
                                     Unlock()
```

#### 1.4.3 为什么不用 channel 代替锁？

理论上可以用缓冲为 1 的 channel 模拟互斥锁，但：
1. 语义不清——读代码的人要反应"这是当锁用的 channel"
2. 没有 RWMutex 的读写分离——读操作也独占，性能差
3. Go race detector 认识 `sync.Mutex` 可以精确报 race，不认识 channel-as-mutex
4. `defer Unlock()` 模式用 channel 写起来别扭

> **channel 传递数据所有权，锁保护共享状态。两者不矛盾——我们的 Session 同时用了两种：channel 传 Hook 消息，锁保护 state。**

#### 1.4.4 sync.RWMutex — 读写锁

```go
type Session struct {
    mu    sync.RWMutex
    state SessionState
}

func (s *Session) State() SessionState {
    s.mu.RLock()          // 读锁——允许多个 goroutine 同时读
    defer s.mu.RUnlock()
    return s.state
}

func (s *Session) setState(new SessionState) {
    s.mu.Lock()           // 写锁——独占
    defer s.mu.Unlock()
    s.state = new
}
```

**什么时候用 RWMutex 而不是 Mutex？** 读远多于写时。`State()` 可能被 100 个 goroutine 同时查询——`RLock` 允许并发读。只有 `setState` / `Detach` 时用写锁排斥所有操作。

#### 1.4.5 项目中的锁全景

| 锁 | 保护什么 | 为什么需要 |
|----|---------|-----------|
| `HookSession.mu RWMutex` | `state` 字段 | 防止并发 Detach → 重复关闭 channel panic |
| `HookSession.mu RWMutex` | `CreateScript` vs `Detach` | 防止在 CreateScript 中间被 Detach |
| `SessionManager.mu Mutex` | `sessions` map | Go map 并发写直接 panic——不是 race 是 crash |
| `SessionManager.mu Mutex` | 遍历+删除 | 遍历 sessions 时 Attach 插入 → 迭代器失效 |

#### 1.4.6 项目实际代码

```go
// pkg/fridaengine/session.go
type HookSession struct {
    state   SessionState
    mu      sync.RWMutex
}

func (s *HookSession) State() SessionState {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.state
}

func (s *HookSession) Detach() error {
    s.mu.Lock()         // 整个函数在锁保护下
    defer s.mu.Unlock()

    if s.state == SessionStateDetached {
        return nil      // ← 幂等：第二次调用直接返回
    }
    s.state = SessionStateDetached
    close(s.msgCh)
    // ...
}
```

### 1.5 channel — goroutine 间通信

#### 1.5.1 为什么需要 channel？—— 共享内存的噩梦

假设两个 goroutine 要传递数据。不用 channel 的做法：

```go
var sharedData string
var done bool

go func() {
    sharedData = "hello"  // 步骤 1：写数据
    done = true           // 步骤 2：标记完成
}()

for !done {               // 步骤 3：等完成
}                         // ⚡ CPU 空转！一个核心被占满
fmt.Println(sharedData)   // 步骤 4：可能是空的！编译器可能重排指令
```

三个问题：data race、CPU 空转、指令重排。**Go 的答案：channel——等待数据时不消耗 CPU。**

#### 1.5.2 核心心智模型：有类型的管道

```
goroutine A                 goroutine B
    │                           │
    │  ch <- "hello"            │  msg := <-ch
    │  ───────────────►         │
    │         管道                │
```

channel 是**类型安全、goroutine 安全的队列**：`chan int` 只能传 int，多 goroutine 同时读写不需加锁。

#### 1.5.3 基础操作

```go
ch := make(chan string, 1)

ch <- "hello"        // 发送（箭头指向 channel）
msg := <-ch          // 接收（箭头从 channel 出来）

msg, ok := <-ch      // 接收 + 检查 channel 是否已关闭
if !ok { /* channel 已关闭 */ }

close(ch)            // 关闭 — 只有发送方应该关闭！
```

#### 1.5.4 无缓冲 vs 有缓冲

**无缓冲 = 同步握手**：发送方必须等接收方准备好。

```go
ch := make(chan int)  // 无缓冲

go func() {
    v := <-ch         // 等待...
    fmt.Println(v)    // 收到 42 — 发送方同时释放
}()

ch <- 42              // 阻塞，直到 goroutine 准备好接收
```

**有缓冲 = 带等待区的柜台**：

```go
ch := make(chan int, 3)

ch <- 1  // 不阻塞 [1]
ch <- 2  // 不阻塞 [1,2]
ch <- 3  // 不阻塞 [1,2,3]
ch <- 4  // 阻塞！缓冲区满，等有人取走
```

| 缓冲大小 | 适用场景 |
|---------|---------|
| 0 (无缓冲) | 严格同步、每次发送必须被处理 |
| 1 | 简单通知信号 |
| 64 (我们的选择) | 生产快消费慢的缓冲——Hook 回调高频写入 |

#### 1.5.5 `for range` 遍历 channel

```go
ch := make(chan int, 3)
ch <- 1; ch <- 2; ch <- 3
close(ch)             // ⚠️ 必须先 close，否则 range 永远不退出

for v := range ch {   // 自动读到 close
    fmt.Println(v)
}
// 输出 1 2 3 然后退出
```

**陷阱 ❌**：忘记 close → `for range` 永远阻塞 → goroutine 泄漏。

#### 1.5.6 channel 方向 — 编译器帮你防 bug

```go
func producer(ch chan<- int) { ch <- 42 }   // 只能发送
func consumer(ch <-chan int) { v := <-ch }  // 只能接收
```

`chan int` 自动转为单向。反过来不行。**方向声明 = 文档 + 安全检查**。

#### 1.5.7 常见错误

```go
close(ch); ch <- 1     // panic: send on closed channel
var ch chan int; ch <- 1  // nil channel 永远阻塞
ch := make(chan int); ch <- 1  // 无缓冲且无接收方 → 死锁！
```

#### 1.5.8 项目实际代码

```go
// 发送端（frida 回调 → channel）— script.go
func (s *HookScript) onMessage(ch chan<- HookMessage) {
    s.script.On("message", func(msg string) {
        ch <- HookMessage{Type: "message", Payload: msg, Timestamp: time.Now()}
    })
}

// 接收端（暴露给调用者）— session.go
func (s *HookSession) Messages() <-chan HookMessage {
    return s.msgCh
}

// 消费端 — 调用者代码
for msg := range session.Messages() {
    fmt.Println(msg.Payload)
}
```

**完整数据流**：

```
frida 回调 (C 线程)
    │ ch <- HookMessage{...}
    ▼
有缓冲 channel [64]     ← 吸收生产-消费速度差
    │ <-chan HookMessage
    ▼
for msg := range ch {}  ← session.Detach() 关闭 channel → range 自动退出
```

**为什么缓冲 64？** Hook 回调 `send()` 高频触发——消费慢时缓冲区吸收 64 条消息，防止 frida 回调阻塞（宪法 3.3: 回调 < 100ms）。

### 1.6 并发错误收集 — `[]error` + `sync.Mutex`

多个 goroutine 各自执行操作，怎样把它们的错误汇总到一起？

```go
var errs []error
var mu sync.Mutex
var wg sync.WaitGroup

for _, s := range sessions {
    wg.Add(1)
    go func(session *HookSession) {
        defer wg.Done()
        if err := session.Detach(); err != nil {
            mu.Lock()                // 保护 errs slice — 并发 append 不是线程安全的
            errs = append(errs, err)
            mu.Unlock()
        }
    }(s)
}

wg.Wait()
// 现在 errs 包含所有失败的错误
```

**为什么需要 Mutex？** Go 的 slice 不是线程安全的。两个 goroutine 同时 `append` 到同一个 slice，内部指针/长度/容量会混乱 → 数据丢失或 panic。

**项目实际代码** (`pkg/fridaengine/manager.go:119-133`):

```go
func (m *SessionManager) DetachAll() error {
    sessions := /* 从 map 拷贝出来 */
    var errs []error
    var mu sync.Mutex
    var wg sync.WaitGroup

    for _, s := range sessions {
        wg.Add(1)
        go func(session *HookSession) {
            defer wg.Done()
            if err := session.Detach(); err != nil {
                mu.Lock()
                errs = append(errs, err)
                mu.Unlock()
            }
        }(s)
    }
    wg.Wait()
    // ...
}
```

### 1.7 map 并发安全 — 先拷贝再遍历

Go 的 map **不是并发安全的**。一个 goroutine 在遍历 map，另一个 goroutine 在插入——直接 fatal panic。

```go
// ❌ 错误：可能 panic
for id, session := range m.sessions {
    // 与此同时另一个 goroutine 调了 Attach() → m.sessions[id] = newSession
}

// ✅ 正确：先拷贝出来
m.mu.Lock()
copied := make([]*HookSession, 0, len(m.sessions))
for _, s := range m.sessions {
    copied = append(copied, s)
}
m.mu.Unlock()

// 现在安全遍历拷贝
for _, s := range copied {
    s.Detach()
}
```

**项目实际代码** (`pkg/fridaengine/manager.go:109-114`):

```go
func (m *SessionManager) DetachAll() error {
    m.mu.Lock()
    sessions := make([]*HookSession, 0, len(m.sessions))
    for _, s := range m.sessions {
        sessions = append(sessions, s)  // 在锁保护下拷贝
    }
    m.mu.Unlock()
    // 释放锁后再遍历——安全
}
```

### 1.8 清理链模式 — `Engine.Close → DetachAll → Detach`

Go 的清理是责任链：上层关闭 → 中层遍历 → 底层清理。每一层只对自己的资源负责。

```
Engine.Close()
    │
    └─► SessionManager.DetachAll()
            │
            ├─► HookSession.Detach()  (幂等)
            ├─► HookSession.Detach()  (幂等)
            └─► HookSession.Detach()  (幂等)
                    │
                    └─► frida.Session.Detach() (C 资源释放)
```

**项目实际代码** (`pkg/fridaengine/engine.go:70-72`):

```go
func (e *Engine) Close() error {
    return e.manager.DetachAll()  // 委托——Engine 不直接操作 Session
}
```

**设计原则**：最外层 `defer engine.Close()` 确保退出时所有 Session 被清理，即使某个 Session 的 `defer session.Detach()` 被遗漏。

### 1.9 依赖注入 — 接口作为参数

依赖注入在 Go 中**不需要框架**。标准做法：结构体字段存接口类型，构造函数接受接口参数。

```go
type Logger interface {
    Info(msg string, args ...any)
}

type Service struct {
    log Logger       // 依赖接口，不依赖具体实现
}

func NewService(log Logger) *Service {
    return &Service{log: log}
}
```

**M2 中的用法**：`Engine` 接受 `device.DeviceLister`（M1 定义的接口）和 `*slog.Logger`（标准库 log 接口）。测试时注入 stub，生产时注入真实 Frida 实现。

### 1.10 自定义错误类型 — `error` 接口 + `Unwrap()`

Go 的 `error` 接口只有一个方法 `Error() string`。自定义错误类型只需实现它。加上 `Unwrap() error` 方法后，`errors.Is()` 和 `errors.As()` 就能逐层解包错误链。

```go
type DeviceError struct {
    Op  string
    ID  string
    Err error // 包装底层错误
}
func (e *DeviceError) Error() string { return "device " + e.Op + ": " + e.Err.Error() }
func (e *DeviceError) Unwrap() error { return e.Err }
```

**项目实际代码** (`pkg/fridaengine/errors.go`):

```go
// 三层错误类型，对应 Frida 调用链的三个层级
type DeviceError struct  { Op, ID string; Err error }  // 设备枚举/连接失败
type SessionError struct { Op, Target string; Err error } // Attach/Detach 失败
type ScriptError struct  { Op string; Err error }       // 脚本创建/加载失败

// 每个类型都实现 Error() 和 Unwrap()，支持:
// errors.Is(err, rootCause) — 判断错误链中是否包含特定错误
// errors.As(err, &sessionErr) — 从错误链中提取特定类型错误
```

**关键理解**: `Unwrap()` 是 Go 1.13 引入的接口方法，标准库的 `errors.Is` / `errors.As` 依赖它遍历错误链。不实现 `Unwrap()`，错误包装就只是字符串拼接——丢掉了类型信息。

### 1.11 Go enum — `iota` 常量生成器

```go
type SessionState int
const (
    SessionStateCreated  SessionState = iota // 0
    SessionStateReady                        // 1
    SessionStateDetached                     // 2
)
```

`iota` 在 const 块中从 0 开始，每行 +1。Go 没有真正的 enum 关键字，`type X int` + `iota` 是惯用模式，提供编译时类型安全。

**项目实际代码** (`pkg/fridaengine/session.go`):

```go
type SessionState int
const (
    SessionStateCreated  SessionState = iota // Attach 成功，尚未加载脚本
    SessionStateReady                        // 脚本已加载，可收发消息
    SessionStateDetached                     // 已断开
)

func (s SessionState) String() string {
    switch s {
    case SessionStateCreated: return "created"
    case SessionStateReady:   return "ready"
    case SessionStateDetached:return "detached"
    default:                  return "unknown"
    }
}
```

**为什么用 `iota` 而不用 string enum？** M1 的 `HookType` 用 `type HookType string` + `const HookTypeOverload HookType = "overload"`——那是字符串枚举，方便 YAML 反序列化。这里用 int 枚举，因为 `SessionState` 只在 Go 内部流转，不需要字符串序列化，int 比 string 比较更快（== 是单条 CPU 指令）。

### 1.12 Go test 工具链 — 覆盖率、条件编译、静态分析

#### 1.12.1 `//go:build integration` — 条件编译

Go 的 build tag 控制哪些文件参与编译。集成测试需要真机，CI 上不能跑——用 tag 隔离：

```go
//go:build integration
// +build integration       ← 兼容旧版 Go 的语法

package fridaengine

func TestIntegrationFullLifecycle(t *testing.T) {
    e := NewEngine(nil, nil)
    // ... 需要真机的测试 ...
}
```

**使用**：
```bash
go test ./pkg/fridaengine/                # 普通测试（不含 integration）
go test -tags=integration ./pkg/fridaengine/  # 包含 integration 测试
```

**和 `t.Skip` 的区别**：`t.Skip` 仍然会编译测试代码（如果 import 了不存在的包会编译失败），build tag 直接从编译阶段排除。

#### 1.12.2 覆盖率 — `-coverprofile`

```bash
go test -coverprofile=coverage.out ./pkg/fridaengine/  # 生成覆盖率文件
go tool cover -func=coverage.out                        # 按函数查看
go tool cover -html=coverage.out                        # 浏览器可视化
```

**M2 覆盖率**：76.4%。未覆盖的是 frida session 依赖路径（`CreateScript`、`Detach` 完整流程、`script.go`），这些由 `integration` 标签测试覆盖。

**为什么不是 100%？** 核心库代码 100% 覆盖必须依赖真实 frida-server，这不是单元测试能解决的。CI 上也不能假设有 Android 设备。所以分两层：单测 76.4% + 集成测试补全。

#### 1.12.3 go vet — 静态分析

```bash
go vet ./pkg/fridaengine/  # 检查常见错误：unreachable code、printf 参数错误等
```

vet 和 lint 的区别：vet 是 Go 标准库自带，检查运行时正确性问题；lint (golangci-lint) 是第三方，检查代码风格。M2 通过 vet，lint 工具待安装。

---

## 二、Android 逆向轨道：Frida 完整生命周期

### 2.1 Frida 三端架构

```
┌──────────────┐       USB/TCP        ┌───────────────┐
│  开发机 (PC)  │ ←─────────────────→ │  Android 设备  │
│              │                      │               │
│  frida-core  │                      │ frida-server  │
│  (宿主进程)   │                      │ (守护进程)     │
│  frida-agent  │ ── 注入目标进程 ──→  │ 目标 App      │
│  (JS 引擎)    │                      │ (被 Hook 进程) │
└──────────────┘                      └───────────────┘
```

- **frida-core**：C 库，负责设备枚举、进程通信、Session 管理。`frida-go` 通过 CGO 绑定 frida-core
- **frida-server**：运行在 Android 设备上的守护进程，监听 USB/TCP 端口，接收来自 PC 的命令
- **frida-agent**：动态库（.so），注入到目标进程，包含 JS 引擎（Duktape/V8）

### 2.2 完整 Hook 生命周期

用 JavaScript 伪代码展示 Frida 的完整调用链（Go 版通过 frida-go 调用相同 API）：

```javascript
// Step 1: 枚举设备
const devices = DeviceManager.enumerateDevices();
// → [LocalDevice(本机), RemoteDevice(USB-模拟器), RemoteDevice(USB-真机)]

// Step 2: 获取目标设备（过滤 Local）
const device = devices.find(d => d.type !== "local");

// Step 3: Attach 到目标进程（按包名或 PID）
const session = device.attach("com.example.app");
// 等价命令行: frida -U com.example.app

// Step 4: 创建脚本
const script = session.createScript(`
    Java.perform(function() {
        var MainActivity = Java.use("com.example.MainActivity");
        MainActivity.onCreate.implementation = function() {
            console.log("onCreate 被调用了!");
            this.onCreate();
        };
    });
`);

// Step 5: 监听消息
script.on("message", (msg) => {
    console.log("收到消息:", msg.payload);
});

// Step 6: 加载执行
script.load();
// → 目标进程的 onCreate 被 Hook —— 每次调用都会输出日志

// Step 7: 断开
session.detach();
```

**M2 中 frida-go 的对应调用**：

```go
mgr := frida.NewDeviceManager()
devices, _ := mgr.EnumerateDevices()       // Step 1

dev, _ := mgr.FindDeviceByType(frida.DeviceTypeUSB) // Step 2

session, _ := dev.Attach("com.example.app", nil)    // Step 3

script, _ := session.CreateScript(jsSource)         // Step 4
script.On("message", handler)                       // Step 5
script.Load()                                       // Step 6

session.Detach()                                    // Step 7
```

**项目实际代码** (`pkg/fridaengine/device.go`) — 设备枚举的完整实现：

```go
type FridaDeviceLister struct {
    mgr    *frida.DeviceManager
    logger *slog.Logger
}

func (l *FridaDeviceLister) ListDevices(ctx context.Context) ([]device.Device, error) {
    // 1. goroutine 中执行 CGO 调用（阻塞操作）
    done := make(chan struct{})
    var fridaDevices []frida.DeviceInt
    var enumerateErr error
    go func() {
        defer close(done)
        fridaDevices, enumerateErr = l.mgr.EnumerateDevices()
    }()
    // 2. select 等待结果或超时
    select {
    case <-ctx.Done():
        return nil, fmt.Errorf("device enumerate: %w", ctx.Err())
    case <-done:
    }
    if enumerateErr != nil {
        return nil, NewDeviceError("enumerate", "", enumerateErr)
    }
    // 3. 过滤 Local 设备 + 映射类型
    var result []device.Device
    for _, d := range fridaDevices {
        connectType := fridaDeviceTypeToConnectType(d.DeviceType())
        if connectType == "" { // Local 设备 → 跳过
            continue
        }
        result = append(result, device.Device{
            ID: d.ID(), Name: d.Name(), ConnectType: connectType,
        })
    }
    return result, nil
}

// DeviceType 枚举 → M1 ConnectType 映射
func fridaDeviceTypeToConnectType(dt frida.DeviceType) device.ConnectType {
    switch dt {
    case frida.DeviceTypeRemote: return device.ConnectTypeNetwork
    case frida.DeviceTypeUsb:    return device.ConnectTypeUSB
    default:                     return "" // Local 设备过滤
    }
}
```

**Step 3-7 的 Go 版本** — Attach + 脚本注入 + 消息接收 (`pkg/fridaengine/engine.go` + `session.go`):

```go
// Step 3: Attach 到目标进程
session, err := engine.Attach(ctx, devices[0].ID, "com.example.app")
if err != nil {
    return err
}
defer session.Detach()  // ← Step 7 保证执行（defer + 幂等）

// Step 4-6: 创建脚本 → 注册消息 → 加载
err = session.CreateScript(`
    Java.perform(function() {
        var MainActivity = Java.use("com.example.MainActivity");
        MainActivity.onCreate.implementation = function() {
            send("onCreate called!");  // send() → frida → channel
            this.onCreate();
        };
    });
`)
if err != nil {
    return err
}

// Step 5-7: 接收消息（循环读取 channel，直到 Session Detach 关闭 channel）
for msg := range session.Messages() {
    fmt.Printf("[%s] %s\n", msg.Type, msg.Payload)
}
```

**关键设计**：
- `session.Detach()` 关闭 `msgCh` channel → `for range` 循环自动退出
- `Detach()` 幂等（已 Detached 状态下调用直接返回 nil）
- 用 `defer session.Detach()` 确保即使 panic 也会执行清理

### 2.3 frida-server 部署

```bash
# 1. 下载对应架构的 frida-server
# https://github.com/frida/frida/releases
# arm64 设备 → frida-server-*-android-arm64.xz

# 2. 推送到设备
adb push frida-server /data/local/tmp/
adb shell chmod 755 /data/local/tmp/frida-server

# 3. 启动（需要 root）
adb shell su -c /data/local/tmp/frida-server &

# 4. 验证
frida-ls-devices    # 应看到 USB 设备
```

### 2.4 USB vs TCP 网络连接

| 连接方式 | 场景 | 配置 |
|---------|------|------|
| USB | 开发调试（默认） | `adb forward tcp:27042 tcp:27042` |
| TCP/WiFi | 远程测试、无 USB 线 | `adb tcpip 5555` + `adb connect <ip>:5555` |

USB 连接更稳定，延迟更低。TCP 用于远程或不便接线的场景。frida-go 的 `DeviceType` 枚举区分两者。

### 2.5 进程标识：包名 vs PID

Frida 支持两种 Attach 目标指定方式：

- **按包名**：`device.attach("com.example.app")` → 内部先 `ProcessByName()` 解析 PID，再 Attach
- **按 PID**：`device.attach(12345)` → 直接 Attach
- **M2 策略**：优先按包名（符合 HookSpec 的 `app_package` 语义），也支持 PID

### 2.6 Frida Scope — 进程枚举的范围控制

`EnumerateProcesses` 和 `EnumerateApplications` 接受一个 `Scope` 参数，控制返回范围：

```go
type Scope int
const (
    ScopeMinimal Scope = iota  // 仅当前用户可见的进程
    ScopeMetadata              // 返回元数据（图标、版本等）
    ScopeFull                  // 所有进程 + 完整信息
)
```

| Scope | 返回内容 | 性能 | 用途 |
|-------|---------|------|------|
| `ScopeMinimal` | 仅 PID + Name | 最快 | 按名查进程、Attach 前确认 |
| `ScopeFull` | PID + Name + 图标/参数等 | 较慢 | 展示设备完整进程列表 |

**项目实际代码** (`pkg/fridaengine/engine.go:92-114`):

```go
func (e *Engine) EnumerateProcesses(ctx context.Context, deviceID string) ([]ProcessInfo, error) {
    dev, err := e.findDevice(ctx, deviceID)
    // ...

    go func() {
        fridaProcs, ferr := dev.EnumerateProcesses(frida.ScopeFull)  // 完整列表
        // 映射 frida.Process → ProcessInfo
        procs := make([]ProcessInfo, len(fridaProcs))
        for i, p := range fridaProcs {
            procs[i] = ProcessInfo{PID: p.PID(), Name: p.Name()}
        }
        done <- result{processes: procs}
    }()
    // select ctx.Done() / done ...
}
```

**设计选择**：M2 统一用 `ScopeFull`——虽然比 `ScopeMinimal` 慢一点，但用户枚举进程时通常需要完整列表，半截子数据反而困惑。以后可以加参数让用户选择。

**Go 模式**：`findDevice()` 是私有 helper——三个方法（Attach/EnumerateProcesses/EnumerateApplications）都需要根据 deviceID 找 frida 设备，抽出来避免重复代码。

---

## 三、AI 编程轨道：SpecKit 第二次迭代

### 3.1 为什么第二次 SpecKit 感觉不同

M1 时学员第一次接触 SpecKit——一切都是新的，跟着走就行。M2 是第二次完整循环，此时应该形成**肌肉记忆**：

| 能力维度 | M1（第一次） | M2（第二次） |
|---------|-----------|-----------|
| Specify 阶段 | 被动接受 AI 起草 | 主动检查 spec 是否遗漏边界 |
| Clarify 阶段 | 被提问 | 能预判哪些问题会被问到 |
| Plan 阶段 | 读 AI 生成的架构 | 能对照宪法验证设计 |
| Tasks 阶段 | 看任务清单 | 能判断任务粒度是否合适 |
| Analyze 阶段 | 第一次接触交叉验证 | 知道上一轮的教训（接口命名、文件覆盖） |

### 3.2 AI 编程的认知跃迁：从"替代"到"放大"

M1 阶段学员可能还停留在「AI 帮我写代码」的心智模型。M2 的正确心智模型应该是：

**AI 不是代码替代者——AI 是理解放大器**。

具体表现：
- goroutine 的概念看文档可能半懂，但让 AI 画图、写对比示例、解释"goroutine 泄漏长什么样子"——这些才是 AI 的强项
- 写 `SessionManager` 前让 AI 先解释 `errgroup` 和 `WaitGroup + Mutex` 的 tradeoff——你不需要记住所有细节，只需要知道关键差异
- Frida 的生命周期调用链——让 AI 画出时序图，比死记 API 更有效

### 3.3 SpecKit 流程的关键教训

M1 学会的规则 M2 要主动执行（不等 AI 提醒）：

1. **同文件 Task 合并 Commit** — 每个文件必须完整才能 commit
2. **宪法检查两次** — 实现前看设计，实现后看代码
3. **教学编码交替** — 不要看完所有概念再写，每个概念讲完立刻写对应的代码
4. **plan 是合约** — 如果实现时发现 plan 有问题，回头改 plan，不要在代码里打补丁
