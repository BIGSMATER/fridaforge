# 研究笔记: Frida 并发调度引擎

**功能**: 002-frida-engine | **日期**: 2026-05-12

## 决策 1: Frida 设备类型映射

**决策**: 将 frida-go 的 `DeviceType` 映射为 M1 的 `ConnectType`，过滤 Local 设备。

**理由**:
- frida-go `Device.DeviceType()` 返回 `int` 枚举值 (0=Local, 1=Remote, 2=USB, 3=Network)
- M1 `device.ConnectType` 是字符串枚举: `"usb"`, `"network"`, `"emulator"`
- 映射策略: RemoteType(1)=USB, USBType(2)=USB, NetworkType(3)=Network, LocalType(0)=过滤
- `FridaDeviceLister.ListDevices()` 内联执行映射+过滤，对外仅返回 `[]device.Device`

**备选方案**:
- 直接暴露 frida-go Device 类型: 破坏 M1 接口契约，调用者强依赖 frida-go
- 在 Device 结构体新增 DeviceType 整数字段: 数据冗余，ConnectType 已表达需求

## 决策 2: Session 并发模式 — errgroup 风格

**决策**: SessionManager 使用 `sync.WaitGroup` + `sync.Mutex` 实现独立错误聚合，不依赖 `golang.org/x/sync/errgroup`。

**理由**:
- `errgroup` 默认 "一失败全取消" (WithContext 模式)，与 spec 要求的"独立运行，聚合全部错误"冲突
- 用标准库 `sync.WaitGroup` + `sync.Mutex` 可实现更灵活的错误聚合
- 减少外部依赖——M2 已引入 `frida-go` 一个 CGO 依赖，不宜再增
- 实现模式:
  ```go
  type SessionManager struct {
      mu       sync.Mutex
      sessions map[string]*HookSession
      wg       sync.WaitGroup
      logger   *slog.Logger
  }
  ```
  每个 `Attach()` 调用 `wg.Add(1)`，goroutine 内 `defer wg.Done()`

**备选方案**:
- `errgroup.WithContext`: 不符合独立错误策略
- 纯 channel 同步: 过度设计，Session 量级不需要

## 决策 3: Hook 消息 channel 设计

**决策**: 每个 HookSession 暴露 `Messages() <-chan HookMessage` 只读 channel，缓冲 64。

**理由**:
- `chan<-` / `<-chan` 方向明确定义，符合宪法 2.4
- 缓冲 64: 防止 frida-go 回调因调用者处理慢而阻塞 (宪法 3.3: 回调 < 100ms)
- 消息类型 `HookMessage` 包含 Type、Payload、Timestamp，结构化优于裸 string
- Channel 由 `HookSession` 在 Ready 状态时创建，Detach 时关闭

**消息协议**:
```go
type HookMessage struct {
    Type      string    // "log", "info", "error", "send"
    Payload   string    // JSON 字符串，来自 frida send()
    Timestamp time.Time
}
```

**备选方案**:
- 回调函数模式 `script.On("message", fn)`: 回调执行在 frida-go 线程上，违反宪法 3.3
- 无缓冲 channel: 消息积压时阻塞 frida 回调

## 决策 4: 清理策略 — defer + 信号捕获

**决策**: 两层清理机制。(1) Session 级别: `Detach()` 支持 defer 调用，幂等操作 (多次调用不报错)。(2) Engine 级别: `Close()` 遍历所有活跃 Session 逐一 Detach。

**理由**:
- 宪法 3.2 要求 Cleanup 支持 defer 模式，即使在 panic 场景下
- `Detach()` 幂等: 通过状态机保证——若已 Detached，直接返回 nil
- Engine.Close() 作为兜底: 调用者忘记 defer Detach() 时保证资源释放
- 不自动注册 `os.Signal` 处理: Engine 是库包，信号处理属于调用者 (cmd/fridaforge) 职责

**备选方案**:
- 仅 defer Detach: 调用者遗漏时泄漏 C 资源
- 自动信号注册: 库不应自行决定信号行为

## 决策 5: frida-core-devkit 安装方式

**决策**: 在 `docs/learn/M2-frida-concurrency.md` 中提供安装文档，不在 Makefile 中自动化。

**理由**:
- `frida-core-devkit` 需要从 GitHub Releases 下载对应平台/架构的 `.tar.xz`
- 安装路径依赖系统权限 (`/usr/local/include`, `/usr/local/lib`)
- 自动化安装脚本不适合跨平台 (Linux apt vs macOS brew vs Windows)
- frida-go README 已提供标准安装步骤，引用即可

**备选方案**:
- Makefile target 自动下载: 维护成本高，版本锁定困难

## 决策 6: 测试策略 — 桩优先 + Harness 验证

**决策**: M2 测试分两层。(1) 单元测试: `FridaDeviceLister`/`SessionManager` 纯逻辑测试使用桩。(2) 集成测试: 需要真机/模拟器，使用 `go build -tags=integration` 隔离。

**理由**:
- 单元测试不依赖 frida-server: CI 和开发环境通用
- 集成测试用 build tag 隔离: `//go:build integration` 防止 CI 误执行
- 桩设计: `FridaDeviceLister` 可注入 mock `frida.DeviceManager` (需要 Go 接口包装)
- 宪法 2.5 要求覆盖率 ≥ 80%: 纯逻辑覆盖满足，集成测试作为补充

**备选方案**:
- 全部真机测试: CI 不可行
- 无 integration build tag: 裸 `go test ./...` 会失败

## 决策 7: frida-go 版本选择

**决策**: 使用 `github.com/frida/frida-go/frida@latest`，在 go.mod 中锁定 minor version。

**理由**:
- frida-go 主线活跃更新，`@latest` 获取最新稳定版
- frida-core-devkit 版本需与 frida-go 编译时使用的 `frida-core.h` 头文件匹配
- go.sum 锁定具体 commit hash
- 首次引入时 `go get github.com/frida/frida-go/frida@latest`

## 决策 8: 模式切换 (Plan Mode → Build Mode)

本次 Plan 完成后将进入 Build Mode，可以:
- 安装 frida-go 依赖 (`go get`)
- 创建 `pkg/fridaengine/` 源文件
- 运行编译验证

## 已解决的技术未知项

所有 Phase 0 研究项已解决:
- DeviceType 映射 ✅
- 并发模式 ✅
- Channel 协议 ✅
- 清理策略 ✅
- 测试策略 ✅
- 版本选择 ✅
