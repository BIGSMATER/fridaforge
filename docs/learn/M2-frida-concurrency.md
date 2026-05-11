# M2 学习笔记：Go 并发模型与 Frida 调度引擎

> Milestone: M2 | 状态: 规划完成，待实现
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

`context.Context` 是 Go 中 goroutine 生命周期管理的**标准方式**。它像一棵倒挂的树：根 context 通过 `WithCancel` / `WithTimeout` / `WithDeadline` 派生子 context，取消父节点会级联取消所有子节点。

```go
package main

import (
    "context"
    "fmt"
    "time"
)

func doWork(ctx context.Context) error {
    select {
    case <-time.After(2 * time.Second):
        fmt.Println("工作完成")
        return nil
    case <-ctx.Done():           // ctx 被取消时这个 channel 关闭
        return ctx.Err()         // context.Canceled 或 context.DeadlineExceeded
    }
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()               // 避免资源泄漏——即使任务完成也要调用 cancel

    err := doWork(ctx)
    fmt.Println(err)             // context.DeadlineExceeded —— 1 秒超时 < 2 秒工作
}
```

**M2 中的用法**：
- `Engine.Attach(ctx, deviceID, target)` 接受 ctx，默认 30 秒超时
- `SessionManager` 为每个 Attach goroutine 创建子 context（`ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)`）
- Session 断开时调用 `cancel()`

### 1.4 sync.Mutex — 保护共享数据

多个 goroutine 同时读写同一个变量会导致 **data race**（数据竞争）。`sync.Mutex` 提供互斥锁保护：

```go
package main

import (
    "fmt"
    "sync"
)

type Counter struct {
    mu    sync.Mutex      // 保护 value
    value int
}

func (c *Counter) Inc() {
    c.mu.Lock()           // 获取锁——其他 goroutine 在此阻塞
    c.value++
    c.mu.Unlock()
}

func (c *Counter) Value() int {
    c.mu.Lock()
    defer c.mu.Unlock()   // defer 确保即使 panic 也释放锁
    return c.value
}

func main() {
    var c Counter
    var wg sync.WaitGroup

    for i := 0; i < 1000; i++ {
        wg.Add(1)
        go func() {
            c.Inc()
            wg.Done()
        }()
    }
    wg.Wait()
    fmt.Println(c.Value()) // 1000 — 不加锁的话可能 < 1000
}
```

**`sync.RWMutex`** — 读写锁：`RLock()`（读锁，多个 goroutine 可同时持有）vs `Lock()`（写锁，独占）。

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

**M2 中的用法**：`SessionManager.sessions` 用 `sync.Mutex` 保护（并发 Attach/Detach 时写入），`HookSession.state` 用 `sync.RWMutex` 保护（State() 高频读取，状态变更低频写入）。

### 1.5 channel — goroutine 间通信

Go 的核心理念：**「不要通过共享内存来通信——通过通信来共享内存」**。channel 是 goroutine 之间的类型安全管道。

```go
package main

import "fmt"

func producer(ch chan<- int) {    // chan<- 单向发送 channel
    for i := 1; i <= 5; i++ {
        ch <- i
    }
    close(ch)                      // 关闭 channel 通知消费者"没有了"
}

func consumer(ch <-chan int) {    // <-chan 单向接收 channel
    for v := range ch {            // range 自动读到 channel 关闭
        fmt.Println("收到:", v)
    }
}

func main() {
    ch := make(chan int, 2)        // 有缓冲 channel，容量 2
    go producer(ch)
    consumer(ch)
}
```

**缓冲 vs 无缓冲**：
- `make(chan int)` — 无缓冲：发送方阻塞直到接收方读取（同步）
- `make(chan int, 64)` — 缓冲 64：满之前发送不阻塞（异步）

**宪法 2.4 约束**：channel 参数必须明确方向（`chan<-` 或 `<-chan`），禁止双向 channel 作为参数。

**M2 中的用法**：每个 `HookSession` 创建 `chan HookMessage` (缓冲 64)，frida-go 回调写入，调用者从 `Messages() <-chan HookMessage` 读取。方向明确——回调只写入，调用者只读取。

### 1.6 依赖注入 — 接口作为参数

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
