# M1 学习笔记：Go CLI 骨架与声明式配置解析

> Milestone: M1 | 状态: 教学完成，待编码
> 三轨并行：Go 语言 / Android 逆向 / AI 编程范式

---

## 一、Go 语言轨道

### 1.1 `package` 与 `import` — Go 的代码组织

Go 的代码组织单元是 **package**。每个 `.go` 文件第一行声明自己属于哪个 package。同一个目录下的所有 `.go` 文件必须属于同一个 package。

```go
// pkg/spec/types.go
package spec  // ← 包名和目录名一致（Go 惯例）

import (
    "fmt"                  // 标准库
    "gopkg.in/yaml.v3"     // 第三方库
)
```

**关键规则：大写开头 = 导出（public），小写开头 = 包内私有。**

Go 没有 `public`/`private` 关键字——全靠首字母大小写控制可见性。这是 Go 最独特的语言设计之一。

```go
type HookSpec struct {     // 导出：其他包可以用 spec.HookSpec
    AppPackage string      // 导出字段（大写）
    hooks      []HookTarget // 私有字段（小写），其他包看不见
}
```

### 1.2 `struct` + tag — Go 的数据建模

Go 没有类（class），用 `struct` 组织数据。**struct tag**（结构体标签）是附加在字段上的元数据，用反引号 `` ` `` 包裹：

```go
type HookSpec struct {
    AppPackage string       `yaml:"app_package"`  // ← struct tag
    Hooks      []HookTarget `yaml:"hooks"`
}
```

`yaml:"app_package"` 告诉 yaml.v3：**「YAML 文件里的 `app_package` 键 → 映射到这个 Go 字段」**。没有 tag 的话，yaml.v3 会按字段名小写匹配——`AppPackage` 变成 `apppackage`，对不上 YAML 里的 `app_package`，导致解析为默认零值。

**常见 tag 类型**：`json:"name"`（JSON 序列化）、`yaml:"name"`（YAML 序列化）、`xml:"name"`（XML 序列化）。

### 1.3 `if err != nil` — Go 的错误哲学

Go 没有 try-catch。函数返回 `(result, error)` 元组，**每次调用都必须显式检查错误**：

```go
data, err := os.ReadFile(path)
if err != nil {
    return nil, fmt.Errorf("无法读取文件 %s: %w", path, err)
}
```

**`%w` 动词**（Go 1.13+）：将原始 error 包装进新 error 中，形成错误链。外部可以通过 `errors.Is()` / `errors.As()` 逐层解包，判断错误来源。宪法 2.3 强制要求使用 `%w` 包装上下文。

### 1.4 `slice` 与 `map` — Go 的集合类型

```go
// slice — 动态数组（底层是数组 + 长度 + 容量）
hooks := []HookTarget{
    {ClassName: "com.example.Foo", MethodName: "bar", HookType: "overload"},
    {ClassName: "com.example.Baz", MethodName: "qux", HookType: "replace"},
}

// map — 无序键值对
validTypes := map[string]bool{
    "overload": true,
    "replace":  true,
}
```

**slice vs 数组**：`[3]int` 是固定长度数组（类型的一部分），`[]int` 是 slice（可变长度）。Go 中 99% 的场景用 slice。

### 1.5 Table-Driven Tests — Go 社区的测试标准

宪法 2.5 强制要求。核心理念：**把输入和期望输出放在一个表（slice）里，循环执行**：

```go
func TestValidate(t *testing.T) {
    tests := []struct {        // 匿名 struct 做测试用例
        name    string         // 用例名
        spec    HookSpec       // 输入
        wantErr bool           // 期望是否有错
        errMsg  string         // 期望的错误消息片段
    }{
        {"合法输入", validSpec, false, ""},
        {"空 app_package", emptyPkgSpec, true, "app_package"},
        {"非法 hook_type", badTypeSpec, true, "hook_type"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {   // t.Run 让每个用例独立运行
            err := Validate(&tt.spec)
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

`t.Run()` 创建子测试——每个用例独立运行，失败不会影响其他用例。用 `go test -v` 可以看到每个子测试的结果。

### 1.6 `cobra.Command` — CLI 命令树

Cobra 是 Go 生态的事实标准 CLI 框架（被 kubectl、hugo、GitHub CLI 使用）。用**树形结构**组织命令：

```go
// 根命令
var rootCmd = &cobra.Command{
    Use:   "fridaforge",
    Short: "声明式 Frida 脚本工程化平台",
    PersistentPreRun: func(cmd *cobra.Command, args []string) {
        // 每个子命令执行前都会先走这里
        checkEthicalDisclaimer()
    },
}

// 子命令
var specCmd = &cobra.Command{
    Use:   "spec",
    Short: "Hook 规格文件操作",
}

var validateCmd = &cobra.Command{
    Use:   "validate [文件]",
    Short: "校验 Hook 规格 YAML 文件",
    Args:  cobra.ExactArgs(1),  // 必须有且只有一个参数
    RunE: func(cmd *cobra.Command, args []string) error {
        return runValidate(args[0])
    },
}

func init() {
    rootCmd.AddCommand(specCmd)      // spec 挂在 root 下
    specCmd.AddCommand(validateCmd)  // validate 挂在 spec 下
}
```

**关键概念**：
- `RunE` vs `Run`：`RunE` 返回 error，cobra 自动打印；`Run` 不返回 error
- `PersistentPreRun`：对所有子命令生效的前置钩子
- `Args`：参数校验器，`cobra.ExactArgs(1)` 表示严格一个参数
- cobra 自动生成 `--help` 输出，无需手动编写

### 1.7 `yaml.v3` 反序列化

```go
func LoadSpec(path string) (*spec.HookSpec, error) {
    data, err := os.ReadFile(path)       // 读文件全部内容
    if err != nil {
        return nil, fmt.Errorf("读取文件失败: %w", err)
    }

    var s spec.HookSpec
    err = yaml.Unmarshal(data, &s)       // 反序列化：字节 → 结构体
    if err != nil {
        return nil, fmt.Errorf("YAML 解析失败: %w", err)
    }

    return &s, nil  // 返回指针，避免大结构体拷贝
}
```

`yaml.Unmarshal(data, &s)` 中 `&s` 是取地址——Go 中修改外部变量**必须传指针**，否则修改的是副本。yaml.v3 根据 struct tag 做字段映射。

### 1.8 接口（interface）与方法接收者 — Go 的多态

#### 1.8.0 接口到底解决什么问题 —— 用 FridaForge 自己的代码讲

先不用管"鸭子类型"、"隐式满足"这些术语。只看我们项目里的一个真实问题：

**M1 的需求**：`fridaforge device list` 要显示设备列表。

**矛盾**：真实设备列表需要调用 Frida（`frida-core`）才能获取。但 M1 明确规定**不能碰 Frida**——那是 M2 的事。

**不用接口的做法**——把数据硬编码在 CLI 命令里：

```go
// cmd/fridaforge/device.go
func runDeviceList(...) {
    devices := []Device{
        {ID: "emulator-5554", Name: "Android Emulator", ConnectType: "emulator"},
    }
    // 输出...
}
```

M2 时要改成真实 Frida 调用，就要**找到这个函数、改代码、重新测试**。

**用接口的做法**——写一个"合同"，谁签了合同谁负责提供设备数据：

```go
// 1. 定义合同（pkg/device/manager.go）
type DeviceLister interface {
    ListDevices(ctx context.Context) ([]Device, error)
}

// 2. M1 签合同——返回硬编码数据
type StubDeviceLister struct{}
func (s *StubDeviceLister) ListDevices(ctx context.Context) ([]Device, error) {
    return []Device{{...}}, nil
}

// 3. 用户代码只看合同，不关心谁签的（cmd/fridaforge/device.go）
func runDeviceList(lister DeviceLister) {
    devices, _ := lister.ListDevices(ctx)
    // 输出...
}
```

**M2 时**：新写一个实现，M1 的代码**一行不改**：

```go
// M2 新文件 pkg/frida/lister.go
type RealDeviceLister struct{}
func (r *RealDeviceLister) ListDevices(ctx context.Context) ([]Device, error) {
    return frida.EnumerateDevices()  // 真正调 Frida
}
```

**接口的本质一句话**：**一份合同，多个签约方。调用者只看合同，不管谁签约。**

---

**比喻**：

```
合同（DeviceLister）
────────────────────────
签了这份合同的人，必须能
"ListDevices，返回设备列表"
────────────────────────

M1 签约方（StubDeviceLister）      M2 签约方（RealDeviceLister）
ListDevices → 返回硬编码假数据      ListDevices → 通过 Frida 获取真数据
```

你订外卖时只关心"饭会送到"，不关心外卖员是走路还是骑电动车。合同 = "送餐"，签约方 = "走路/骑电动车"——你这边点餐的流程不变。

---

**什么时候需要接口？什么时候不需要？**

本项目里两个对比：

| 功能 | 是否用接口 | 原因 |
|------|-----------|------|
| `device list` | ✅ 用了 `DeviceLister` | M1 桩、M2 真实 Frida——有两个实现 |
| `spec validate` | ❌ 没用接口 | M1 到 M7 都读本地文件——只有一个实现 |

只有一个实现时，不需要接口。接口的价值在于**"同一行为有多个实现需要互相替换"**。

---

#### 1.8.1 核心概念：隐式满足（Structural Typing）

Go 的接口是**隐式满足**的——这是它与 Java、C#、TypeScript 最大的区别之一。

**对比理解**：

```java
// Java：显式声明
interface DeviceLister {
    List<Device> listDevices();
}
class StubDeviceLister implements DeviceLister {  // ← 必须写 implements
    @Override
    List<Device> listDevices() { ... }
}
```

```go
// Go：隐式满足 —— 没有 implements 关键字
type DeviceLister interface {
    ListDevices(ctx context.Context) ([]Device, error)
}

type StubDeviceLister struct{}

// StubDeviceLister 有和接口一模一样的方法，就自动实现了该接口
func (s *StubDeviceLister) ListDevices(ctx context.Context) ([]Device, error) {
    return []Device{...}, nil
}
```

**没有 `implements`、没有 `@Override`、没有 `extends`**。Go 的思路是：**「如果你的方法集合跟我的接口要求完全匹配，那你就实现了这个接口」**——不需要你显式声明。

这就是所谓的 **duck typing（鸭子类型）**："如果它走起来像鸭子、叫起来像鸭子，那它就是鸭子。" 在 Go 里："如果它有接口要求的全部方法，那它就实现了该接口。"

#### 1.8.2 隐式满足是怎么工作的？

编译时，Go 编译器做**结构类型检查**：

```go
func PrintDevices(lister DeviceLister) {  // 参数类型是接口
    devices, _ := lister.ListDevices(...) // 只关心接口的方法
    fmt.Println(devices)
}

func main() {
    stub := &StubDeviceLister{}           // 具体类型
    PrintDevices(stub)                    // ✅ 编译器自动检查：
                                          // StubDeviceLister 有 ListDevices 方法吗？
                                          // 有 → 自动当做 DeviceLister 传进去
}
```

编译器在这里做的检查等价于：**「`StubDeviceLister` 的方法集合中，是否包含了 `DeviceLister` 接口要求的所有方法？」** 如果没有，编译报错——不是在运行时才暴露。

#### 1.8.3 没有 `implements` 那会不会不小心实现错了？

这是对 Go 新手来说最大的疑问。答案是：**编译器会在你传值的地方检查**。

```go
// 编译时强制检查的一个技巧：
var _ DeviceLister = (*StubDeviceLister)(nil)
// 这行代码：把 nil 指针转为 *StubDeviceLister，然后赋值给 DeviceLister 变量
// 如果 StubDeviceLister 没有实现接口 → 编译报错
// `_` 是 Go 的空白标识符（忽略这个值），仅用于编译时类型检查
```

这行代码不产生任何运行时成本，只是编译时验证。很多 Go 开源项目（如 Docker、Kubernetes）都用这个模式来确保类型实现了接口。

#### 1.8.4 为什么 Go 要这样设计？

Java/C# 的 `implements` 是**声明式**的：「我声明我实现了这个接口」。Go 的隐式满足是**结构式**的：「编译器检查你是否有这些方法」。

优势：
1. **松耦合**：定义接口的包不需要知道有哪些实现——完全解耦。你在 `pkg/device/` 定义接口，M2 的人可以在另一个包 `pkg/frida/` 写实现，它们之间没有 import 依赖
2. **方便测试**：在生产代码里传真实实现，测试里传桩——只要满足相同的接口就可以，无需修改生产代码
3. **"小接口"哲学**：Go 鼓励定义只有 1-3 个方法的微型接口，隐式满足让这种设计零负担

#### 1.8.5 示例：M1 中的接口使用模式

```go
// 1. 定义接口（在 pkg/device/manager.go）
type DeviceLister interface {
    ListDevices(ctx context.Context) ([]Device, error)
}

// 2. 桩实现（同一个文件，方便测试）
type StubDeviceLister struct{}

func (s *StubDeviceLister) ListDevices(ctx context.Context) ([]Device, error) {
    return []Device{
        {ID: "emulator-5554", Name: "Android Emulator 5554", ConnectType: "emulator"},
    }, nil
}

// 3. 使用接口（在 cmd/fridaforge/device.go）
//    构造函数接收接口，不关心具体是哪个实现
func NewDeviceListCmd(lister DeviceLister) *cobra.Command {
    return &cobra.Command{
        Use: "list",
        RunE: func(cmd *cobra.Command, args []string) error {
            devices, err := lister.ListDevices(context.Background())
            // ... 处理输出
        },
    }
}
```

**关键设计模式**：依赖注入。`NewDeviceListCmd` 接收 `DeviceLister` 接口而不是具体的 `StubDeviceLister`。这样：
- M1：传入 `&StubDeviceLister{}`，返回桩数据
- M2：传入 `&FridaDeviceLister{}`，连接真实 Frida
- 测试：传入自定义 mock，测试不同场景
- **完全不需要修改 `NewDeviceListCmd` 函数的代码**

#### 1.8.6 方法接收者 — Go 的 `this`

```go
// (s *StubDeviceLister) 是方法接收者。Go 没有 this 关键字。
func (s *StubDeviceLister) ListDevices(ctx context.Context) ([]Device, error) {
    // s 就是方法所属的实例
}
```

你可以把接收者理解为其他语言的 `this`，但 Go 要求你**显式声明**——写在哪里、叫什么名字，完全由你决定。

**指针接收者 vs 值接收者**：

```go
func (s *StubDeviceLister) Foo() { }  // 指针接收者：可以修改 s 的字段
func (s StubDeviceLister) Foo() { }   // 值接收者：操作的是副本，不能修改原对象
```

#### 1.8.7 关键陷阱：混用指针/值接收者时的接口实现

Go **允许**在同个类型上混用指针接收者和值接收者：

```go
type Foo struct{ val int }

func (f *Foo) Set(v int) { f.val = v }    // 指针接收者
func (f Foo) Get() int    { return f.val } // 值接收者 — 混用，编译通过
```

但是！**指针类型和值类型的方法集不同，这会影响接口实现**：

Go 语言规定：
- 类型 `T`（值）的方法集 = **只有值接收者的方法**
- 类型 `*T`（指针）的方法集 = **全部方法（指针接收者 + 值接收者）**

```go
type MyInterface interface {
    Set(int)
    Get() int
}

// Foo 上混用了指针和值接收者
func (f *Foo) Set(v int) { f.val = v }    // 指针接收者
func (f Foo) Get() int    { return f.val } // 值接收者

var v MyInterface
v = &Foo{}  // ✅ 编译通过：*Foo 有全部方法
v = Foo{}   // ❌ 编译错误：Foo（值）没有 Set 方法——因为 Set 需要指针接收者
```

**结论不是"Go 不允许混用"**——Go 允许。但混用会导致只有指针类型 `*T` 能实现接口，值类型 `T` 不行，容易踩坑。

**最佳实践**：统一用一种。99% 的情况下用**指针接收者**：
- 需要修改接收者 → 必须用指针
- 接收者很大 → 指针避免拷贝
- 接收者是 `nil` 也有意义（桩等）→ 指针
- 接收者很小且绝不变异 → 值接收者也可以（如 `time.Time` 这样的值类型）

`StubDeviceLister` 示例（统一指针接收者）：

```go
type StubDeviceLister struct{}

// 全部用指针接收者，一致
func (s *StubDeviceLister) ListDevices(ctx context.Context) ([]Device, error) {
    return []Device{
        {ID: "emulator-5554", Name: "Android Emulator", ConnectType: "emulator"},
    }, nil
}
```

### 1.9 Go 项目标准布局

```
cmd/fridaforge/   # 可执行文件入口（每个子目录 = 一个二进制文件）
pkg/config/       # 可复用的库代码（按领域划分）
pkg/spec/         # 数据类型定义
pkg/device/       # 设备抽象层
```

- `cmd/`：一个目录一个 `main.go`，编译产出一个二进制
- `pkg/`：可以被外部项目 import 的公共库代码
- 测试文件 `*_test.go` 与源码同目录（Go 惯例）

### 1.10 Go 模块系统 — go.mod / go.sum / indirect

Go 的依赖管理通过模块（module）完成。核心文件是 `go.mod`：

```go
module github.com/bigsmater/fridaforge   // ← 模块根路径

go 1.25.2                                 // ← Go 版本要求

require (
    github.com/spf13/cobra v1.10.2        // ← 直接依赖
    gopkg.in/yaml.v3 v3.0.1 // indirect   // ← 间接依赖
)
```

**`go.mod` vs `go.sum`**：

| 文件 | 作用 | 类比 |
|------|------|------|
| `go.mod` | 声明需要哪些依赖和版本范围 | `package.json` |
| `go.sum` | 锁定每个依赖的精确哈希值（防篡改） | `package-lock.json` |

Go 没有独立的 lockfile 格式——`go.sum` 记录的是每个依赖的 SHA256 哈希，用于完整性校验。如果远程仓库的 `go.sum` 和本地不一致，`go mod tidy` 会报错——防止供应链攻击。

**`// indirect` 标记**：

这是一个 Go 新手上路最常见的困惑。`// indirect` 表示这个依赖**还没有被任何 `.go` 文件直接 import**。它是通过 `go get` 手动添加的，或者被另一个依赖间接需要。

**关键行为**：一旦你在代码里写了 `import "gopkg.in/yaml.v3"`，然后跑 `go mod tidy`，`// indirect` 标记会自动消失。这体现了 Go 的"用进废退"哲学——你不用它，它就不算第一公民。

**GOPROXY — Go 模块代理**：

Go 默认从 `proxy.golang.org` 下载依赖，但国内经常超时。解决方法：

```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

`direct` 是回退机制：如果代理没有这个模块（比如私有仓库），就直连源仓库。常用的国内 Go 代理：
- `goproxy.cn` — 七牛维护，七牛自己跑
- `goproxy.io` — 其他社区维护

### 1.11 golangci-lint — Go 的代码质量工具链

Go 有三层代码检查，逐层深入：

| 工具 | 检查内容 | 用途 | 宪法要求 |
|------|---------|------|---------|
| `gofmt` | 代码格式（空格、缩进、换行） | 统一风格 | 强制 (2.1) |
| `go vet` | 可疑代码（无法到达的代码、错误格式字符串） | 静态分析 | 强制 (2.1) |
| `golangci-lint` | 代码质量（未处理 error、未使用变量、拼写） | 深度检查 | 强制 (2.1) |

**`.golangci.yml` 配置结构**：

```yaml
linters:
  enable:              # 显式启用的 linter
    - errcheck         # 检查忽略的 error
    - gofmt            # 格式检查
    - staticcheck      # 深度静态分析
    - misspell         # 拼写检查

issues:
  exclude-use-default: false  # 不跳过默认规则（更严格）
  max-issues-per-linter: 0    # 0 表示不设上限

run:
  timeout: 3m          # 超时 3 分钟
  tests: true           # 测试代码也检查
```

**`errcheck`** 对应宪法 2.3 "永远不要忽略 error 返回值"。如果你写了 `fn()` 而没处理返回值中的 error，lint 会报。

**`staticcheck`** 比 `go vet` 更深。举个例子：
- `go vet` 只检查"这行代码能不能被执行到"
- `staticcheck` 会检查"这个字符串拼接可以优化为 `strings.Builder`"

### 1.12 Makefile 在 Go 项目中的角色

Go 自带 `go build`/`go test`，为什么还需要 Makefile？

**原因**：把多个 Go 命令组合成一个有记忆的口令。没有 Makefile 时，你要记住：
```bash
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out
```

有了 Makefile 就是 `make cover`。

**Makefile 的基本语法**：

```makefile
.PHONY: build test    # 声明这是动作名，不是文件名

build:                # target 名
	go build -o fridaforge ./cmd/fridaforge/  # 缩进必须是 TAB，不能用空格

test:
	go test -v ./...   # -v = verbose，会显示每个 t.Run 子测试结果

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out   # 打印每个函数覆盖率%
```

两个重要细节：
1. **`.PHONY`**：如果恰好存在一个叫 `build` 的文件，make 会认为已经完成了。`.PHONY` 告诉 make "这不是文件，是每次都要执行的动作"。
2. **缩进必须是 TAB**：Makefile 的历史包袱。用空格缩进会报 `*** missing separator. Stop.`

**`go test -v` 的 `-v`**：没有它，测试输出只有 `PASS`/`FAIL`。加上 `-v`，你会看到每个 `t.Run("xxx", ...)` 的子测试名和结果，这就是学习笔记 1.5 讲的 table-driven test 中的子测试。

### 1.13 Go 枚举模式 — `type X string` + const

Go **没有** `enum` 关键字。社区的惯用做法是用一个**自定义 string 类型**配合 `const` 来模拟枚举：

```go
type HookType string   // 1. 定义新类型，底层是 string

const (
    HookTypeOverload HookType = "overload"   // 2. const 列值
    HookTypeReplace  HookType = "replace"
)
```

**为什么不能直接用 `const` 字符串？**

```go
// 如果这样写：
const Overload = "overload"
const Replace  = "replace"

func validate(ht string) { ... }

validate(Overload)  // ✅
validate("hack")     // ✅ 也编译通过！运行时才发现不对
```

有了自定义类型 `HookType`，编译期就有类型检查：

```go
func validate(ht HookType) { ... }

validate(HookTypeOverload)  // ✅
validate("hack")             // ❌ 编译报错：cannot use "hack" (string) as HookType
```

**核心收益**：把 runtime bug → compile-time error。这是 Go 类型系统的一个重要设计哲学：**宁可编译器多报错，也不要运行时崩溃。**

字符串比较也是类型安全的：
```go
if h.HookType == HookTypeOverload { ... }  // ✅
if h.HookType == "overload" { ... }        // ❌ 编译报错
```

### 1.14 Go error 接口深入 — 自定义错误类型

Go 的 `error` 接口只有一个方法：

```go
type error interface {
    Error() string   // 就这一个方法
}
```

**任何有 `Error() string` 方法的类型，就是一个 error。** 这跟 Java 的 `implements Exception` 完全不同——不需要继承、不需要注册。

**自定义错误类型**：

```go
// 定义一个可以包含多个字段错误的错误类型
type ValidationError struct {
    Errors []FieldError
}

// 实现 error 接口
func (e *ValidationError) Error() string {
    // 把所有字段错误的描述拼接成一个字符串
    var b strings.Builder
    for _, fe := range e.Errors {
        b.WriteString(fe.Error())
    }
    return b.String()
}
```

现在 `*ValidationError` 就是一个 `error`，可以在任何接受 `error` 的地方使用：

```go
func Validate(s *HookSpec) error {    // 返回类型是 error 接口
    var ve ValidationError
    // ... 收集错误
    return &ve                        // 实际返回的是 *ValidationError
                                      // 因为它实现了 error 接口，编译通过
}
```

消费端可以用**类型断言**来判断具体是哪种错误：

```go
if validationErr, ok := err.(*ValidationError); ok {
    // 这是校验错误，可以访问 validationErr.Errors 列表
}
```

**`strings.Builder` — 高效字符串拼接**：

在 `Error()` 方法里没有用 `s += "..."` 而是用了 `strings.Builder`。原因是 Go 的字符串是**不可变的**（immutable）——每次 `+=` 都会在内存里分配一份全新字符串并拷贝所有内容。10 个错误字段就是 10 次重新分配 + 拷贝。

`strings.Builder` 内部用 `[]byte` 缓冲，只在最后 `b.String()` 时一次性创建字符串，减少 CPU 和 GC 压力。

### 1.15 cobra 进阶 — 钩子、静默模式、标记文件

**`PersistentPreRun` — 全局前置钩子**：

```go
var rootCmd = &cobra.Command{
    PersistentPreRun: func(cmd *cobra.Command, args []string) {
        checkEthicalDisclaimer()   // 每个子命令执行前都走这里
    },
}
```

`Persistent` 表示**传播**——根命令的 `PersistentPreRun` 对所有子命令生效。无论执行 `fridaforge spec validate` 还是 `fridaforge device list`，都先跑这个钩子。对比：`PreRun` 只对当前命令生效，不传播。

**`SilenceErrors` + `SilenceUsage`**：

cobra 的默认行为：`RunE` 返回 error → cobra 内部打印一次错误 + 打印 usage + `Execute()` 返回 error → `main()` 再打印一次。结果错误重复三次。

```go
SilenceErrors: true,  // cobra 不打印错误，交给 main() 统一处理
SilenceUsage:  true,  // 错误时不显示 Usage（太啰嗦）
```

**标记文件模式（Marker File）**：

```go
markerFile := filepath.Join(homeDir, ".fridaforge", "agreed")
if _, err := os.Stat(markerFile); err == nil {
    return nil  // 文件存在 → 用户已同意 → 跳过
}
// ... 显示伦理声明，要求输入 AGREE
os.MkdirAll(filepath.Dir(markerFile), 0700)
os.WriteFile(markerFile, []byte{}, 0600)
```

这是一个经典的"只执行一次"模式。很多 Unix 工具都用：
- Git：`~/.gitconfig` 存在 → 已初始化
- npm：`~/.npmrc` 存在 → 已配置
- FridaForge：`~/.fridaforge/agreed` 存在 → 已同意

`os.Stat()` 返回文件信息；文件不存在时返回 error。这里只关心存在性，所以用 `_` 忽略返回的 `FileInfo`。

**Unix 文件权限**：`0700`（目录，仅所有者）和 `0600`（文件，仅所有者读写）。对包含隐私的配置文件（"我同意使用安全测试工具"），应该限制权限。

### 1.16 白盒测试 vs 黑盒测试 — `package spec` vs `package spec_test`

测试文件 `types_test.go` 声明是 `package spec`，不是 `package spec_test`。两种写法的区别：

| 声明 | 俗称 | 能访问什么 | 什么时候用 |
|------|------|-----------|-----------|
| `package spec` | 白盒测试 | 同包所有符号——导出和未导出的 | 需要测试内部类型、访问未导出字段 |
| `package spec_test` | 黑盒测试 | 只能访问导出（大写开头）的符号 | 验证公共 API 行为、不依赖内部实现 |

本项目用白盒测试，因为 `TestValidationError` 要直接构造 `FieldError{Path: ..., Line: ...}`——这些字段是小写开头（未导出）的，黑盒测试看不见它们。

**`t.Fatalf` vs `t.Errorf`**：

```go
if len(s.Hooks) != 1 {
    t.Fatalf(...)   // 立即终止当前测试用例（不改了，后面访问 s.Hooks[0] 会 panic）
}
if s.Hooks[0].ClassName != ... {
    t.Errorf(...)   // 记录错误，继续跑后面的检查——一次看到所有不符
}
```

**`%q` 格式动词**：自动给字符串加双引号。`""` 用 `%s` 打印是空白，用 `%q` 打印是 `""`——方便区分空字符串和空白输出。

### 1.17 逃逸分析 — 为什么 `return &ve` 安全

```go
func Validate(s *spec.HookSpec) error {
    var ve ValidationError       // ve 是局部变量，按说应该在栈上
    // ...
    return &ve                   // 返回局部变量的地址！在 C 里是野指针
}
```

Go 编译器做**逃逸分析（escape analysis）**——它检测到 `ve` 的地址通过 `return` "逃出"了函数作用域，就自动把 `ve` 分配到**堆**上而不是栈上。GC 管理它，不会悬空。

可以用编译命令看到逃逸分析的结果：
```bash
go build -gcflags="-m" ./pkg/config/
# 输出：moved to heap: ve
```

**nil 接口陷阱**：

```go
// 这样写有问题：
var ve *spec.ValidationError = nil
var err error = ve       // err 不是 nil！！它"装"了一个类型信息 *ValidationError
err == nil               // false

// 正确写法：
var ve *spec.ValidationError = nil
if hasErrors {
    return ve             // 返回有类型的 nil——调用方 err != nil！
}
return nil                // 返回无类型的 nil——调用方 err == nil
```

**nil 指针 ≠ nil 接口**。nil 接口 = 既无类型信息也无值。nil 指针赋值给接口后 = 有类型信息（`*ValidationError`）但值为 nil——所以接口本身不是 nil。

### 1.18 `init()` — Go 的自动初始化函数

```go
func init() {
    rootCmd.AddCommand(specCmd)
    specCmd.AddCommand(specValidateCmd)
}
```

`init()` 是 Go 的特殊函数：**在 `main()` 执行前自动调用，不需要手动触发**。一个 package 可以有多个 `init()`（多个文件各一个），执行顺序按文件名字母序。

用途：注册、初始化全局状态、构建 cobra 命令树。用在 FridaForge 里把子命令挂到根命令上。

### 1.19 `yaml.Unmarshal` — 为什么必须传指针

```go
var s spec.HookSpec
yaml.Unmarshal(data, &s)   // ✅ &s = 传递 s 的地址
yaml.Unmarshal(data, s)    // ❌ 传了 s 的副本，解析结果写到副本里，函数退出后消失
```

函数签名是 `func Unmarshal(in []byte, out interface{}) error`——第二个参数是 `interface{}`（空接口，接受任何类型）。但内部实现需要**写**到 `out` 指向的变量里，所以必须传指针。**在 Go 里，你想让一个函数修改你的变量，就传它的指针。**

---

## 二、逆向/底层轨道

### 2.1 YAML 字段的逆向语义

| YAML 字段 | 在 Android/ART 运行时中的含义 |
|-----------|------------------------------|
| `app_package` | AndroidManifest.xml 中的 `package` 属性。ART 用它来隔离不同应用的**类加载器**（ClassLoader）——同一个包名 = 同一个沙箱 |
| `class_name` | Dalvik 字节码中的**类全限定名**。Java 源码里是 `com.example.Foo`，在 DEX 文件中被编码为 `Lcom/example/Foo;`（Smali 语法） |
| `method_name` | ART 方法签名的一部分。M1 只记录简单方法名（如 `onCreate`），M3 将加入完整的**参数签名**（如 `(ILjava/lang/String;)V`）以区分重载方法 |

**为什么需要全限定类名**：Android 应用使用 Dalvik/ART 虚拟机，类由「包名 + 类名」唯一标识。Frida 的 `Java.use("com.example.Foo")` 接收的就是这种全限定名——不带 `.class` 后缀，用 `.` 而非 `/`。

### 2.2 Hook 类型的逆向差异

| 类型 | 对应 Frida 模式 | 行为 | 典型用途 |
|------|----------------|------|---------|
| `overload` | `onEnter`/`onLeave` 回调 | 在**原方法前后**插入代码，原方法仍执行 | 监控参数和返回值（不改行为）——如记录加密函数的输入输出 |
| `replace` | `.implementation = function()` | **完全替换**原方法实现，原方法不再执行 | 修改行为——如让 root 检测函数永远返回 `false` |

**Frida 代码对应**：

```javascript
// overload 模式
Java.perform(function() {
    var Target = Java.use("com.example.Target");
    Target.method.overload().implementation = function() {
        console.log("Enter method");  // 前插
        var ret = this.method();      // 调用原方法
        console.log("Leave method");  // 后插
        return ret;
    };
});

// replace 模式
Java.perform(function() {
    var Target = Java.use("com.example.Check");
    Target.isRooted.implementation = function() {
        return false;  // 原方法完全不执行
    };
});
```

M1 只记录 hook 类型声明，不生成 Frida 脚本（脚本生成在 M3）。

### 2.3 Frida 架构与设备通信原理

#### 2.3.1 Frida 的宏观架构

Frida 不是单进程工具，它是一个**三端分离的系统**：

```
┌─────────────────────┐      ┌─────────────────┐      ┌──────────────────┐
│  开发端 (你的电脑)    │      │   USB / 网络      │      │ 目标设备 (手机)    │
│                     │      │                  │      │                  │
│  frida CLI          │──────│  ADB / TCP       │──────│  frida-server    │
│  frida-ps           │      │  27042 端口      │      │  (后台守护进程)    │
│  frida-trace        │      │                  │      │                  │
│  FridaForge (M2+)   │      │                  │      │  目标 App (进程)  │
│                     │      │                  │      │   ↑ 注入 JS      │
└─────────────────────┘      └─────────────────┘      └──────────────────┘
        ↑                                                        ↑
  你的代码在这端                                          frida-agent (GumJS)
 (通过 frida-core API 或                                    引擎跑在 App 进程里
  frida CLI 工具)
```

**三个角色**：

| 角色 | 位置 | 组件 | 职责 |
|------|------|------|------|
| **开发端** | 你的电脑 | `frida-core` 库、frida CLI 工具 | 连接设备、编译/注入 JS 脚本 |
| **传输层** | USB / 网络 | ADB、TCP | 在两端之间传输数据和控制命令 |
| **目标端** | Android 设备 | `frida-server` + `frida-agent` | 接收指令、注入 JavaScript 到目标进程 |

#### 2.3.2 frida-core 是什么？

`frida-core` 是 Frida 的**核心引擎层**（C 语言编写），运行在开发端。它负责：

1. **设备枚举**：发现 USB 和网络连接的设备
2. **会话管理**：创建、维护、销毁与目标进程的连接
3. **脚本生命周期**：编译 JavaScript → 注入到目标进程 → 接收返回值
4. **通信管道**：封装底层传输协议（USB/网络）

M2 时可以通过两种方式在 Go 中调用 frida-core：
- **`frida-go` 绑定**：Go 对 `frida-core` 的 C 语言绑定的封装（相当于 Go 直接调用 `frida-core` 的 C API）
- **调用 `frida-ls-devices` 命令**：用 `os/exec` 执行 frida CLI 工具并解析输出（字符串解析，不需要 C 绑定）

#### 2.3.3 设备发现（Device Enumeration）

`frida-core` 内部扫描设备的机制：

```c
// frida-core 内部的伪代码逻辑
FridaDeviceList *devices = frida_device_manager_enumerate_devices(manager);
// frida-core 做了三件事：
// 1. 调用 ADB: adb devices → 获取 USB 和模拟器设备
// 2. 扫描本机 TCP 端口 27042 → 发现网络设备
// 3. 合并结果，为每个设备打上类型标签（usb / network / emulator）
```

**USB 设备**：frida 通过 ADB（Android Debug Bridge）检测 USB 连接的设备。ADB 是 Android SDK 的工具，用 USB 线连接后，运行 `adb devices` 会列出所有已认证的设备。

**网络设备**：当 Android 设备上运行了 `frida-server -l 0.0.0.0:27042`，它会监听该 TCP 端口。frida-core 扫描局域网内开放 `27042` 端口的主机来发现网络设备。典型场景：WiFi 连接的设备、云手机。

**模拟器**：模拟器通过 ADB 检测（模拟器内置 `adbd` 服务），frida-core 将 ADB 类型为 "emulator" 的设备标记为模拟器。运行 `adb devices` 时模拟器显示为 `emulator-5554`。

#### 2.3.4 frida-server 的作用

`frida-server` 是运行在 Android 设备上的**守护进程**（通常是一个名为 `frida-server-xx.x.x-android-arm64` 的二进制文件）。它的工作：

```
                    frida-server
                        │
         ┌──────────────┼──────────────┐
         │              │              │
    接收开发端的指令   注入到目标进程   返回 Hook 结果
    (监听 27042)    (ptrace attach)    (通过管道送回)
```

启动方式（在 Android shell 里）：
```bash
adb push frida-server /data/local/tmp/     # 推到设备
adb shell chmod +x /data/local/tmp/frida-server
adb shell /data/local/tmp/frida-server &   # 后台运行
```

启动后，frida-server 会：
1. 创建 TCP 监听 socket（默认 27042）
2. 等待开发端的连接
3. 收到 Hook 请求后，用 Linux `ptrace` 系统调用附加到目标进程
4. 在目标进程内加载 `frida-agent`（GumJS 引擎）
5. GumJS 接收 JavaScript 脚本并执行（`Java.perform`、`Interceptor.attach` 等）

#### 2.3.5 从 `frida device list` 到真正的 Hook（M2 路线图）

M1 的 `device list` 只是 CLI 桩，实际数据流在 M2 会变成：

```
fridaforge device list
    ↓
pkg/device/DeviceLister.ListDevices(ctx)
    ↓
pkg/frida/FridaDeviceLister (M2 实现)
    ↓
frida-core API (通过 frida-go 绑定 或 os/exec 调 frida-ls-devices)
    ↓
ADB 扫描 + TCP 端口扫描
    ↓
设备列表返回
```

M1 用 `StubDeviceLister` 跳过这个完整链路，让 CLI 骨架先搭起来——这是标准的接口驱动设计。

#### 2.3.6 M1 需要知道这些吗？

不需要。M1 的 `StubDeviceLister` 只返回硬编码设备，不调用任何 Frida API。但是理解这个架构能让你明白：
- 为什么 `DeviceLister` 接口现在就要定义——它在 M2 会接入 frida-core
- 为什么 `device list` 的 stub 返回"Frida 服务不可达"错误——因为真正的 frida-server 可能没运行
- 为什么设备列表显示设备 ID/名称/类型——这些正是 `frida_device_get_id()` / `frida_device_get_name()` / `frida_device_get_dtype()` 返回的信息

---

## 三、AI 编程范式轨道

### 3.1 SpecKit 工作流 — M1 完整回顾

M1 走完了 SpecKit 的 4 个阶段（还差 `implement`）：

```
/speckit.specify   # spec.md        — 定义用户要什么（3 个 User Story）
        ↓
/speckit.clarify   # spec.md 增强    — 消除 5 个歧义点
        ↓
/speckit.plan      # plan.md +       — 技术架构、宪法检查、数据模型、契约
                     research.md
                     data-model.md
                     contracts/
                     quickstart.md
        ↓
/speckit.analyze   # 发现 8 个问题  — 交叉验证 spec/plan/tasks 一致性
        ↓
/speckit.tasks     # tasks.md       — 26 个可执行 Task
        ↓
/speckit.implement # 下一步          — 逐个 Task 写代码
```

### 3.2 Spec Coding vs Vibe Coding

这是 M0 就讲过的核心区分。用一个例子说明：

| 模式 | 说"我"要什么 | 说"你怎么"实现 |
|------|-------------|---------------|
| **Spec Coding** ✅ | 「文件不存在时给出清晰的错误信息」 | 留空——给 `/speckit.plan` 决定 |
| **Vibe Coding** ❌ | 跳过 | 「用 if err != nil { fmt.Println(...) }」——越界指导 |

SpecKit 强制执行这种分离：`spec.md` 只管 **WHAT**（想要什么），`plan.md`/`research.md` 只管 **HOW**（怎么实现）。混在一起 = 返回 Vibe Coding。

### 3.3 本阶段的 AI 认知突破

**突破点**：理解了 `/speckit.analyze` 的价值。

初次看感觉多余——spec 和 plan 不都已经仔细写了吗？但实际分析后发现了 8 个问题，其中 1 个是 CRITICAL（接口命名违反宪法），3 个 MEDIUM（术语不一致、缺少性能测试、数据模型歧义）。

教训：**即使每个阶段都觉得"已经写得很清楚了"，三个文档放在一起还是会互相矛盾。** 人脑不擅长做这种大规模交叉对比，但 AI 擅长——这就是 `/speckit.analyze` 存在的理由。

---

## 四、本阶段 Go 知识点汇总

| 知识点 | 对应 tasks.md | 首次出现 |
|--------|-------------|---------|
| `package` / `import` | T005, T006 | M1 |
| `struct` + tag | T005, T018 | M1 |
| `if err != nil` + `%w` | T006, T012 | M1 |
| `slice` / `map` | T009, T013 | M1 |
| Table-driven test `t.Run` | T009-T011, T016-T017, T021 | M1 |
| `cobra.Command` 树形注册 | T007, T008, T014, T015, T020 | M1 |
| `yaml.Unmarshal` | T012 | M1 |
| Interface 概念（合同 vs 实现、何时用/不用） | T019 | M1 |
| `os.ReadFile` / `os.Exit` | T012, T014, T020 | M1 |
| `fmt.Errorf` / `fmt.Fprintf` | T006, T014, T020 | M1 |
| go.mod / go.sum / `// indirect` | T002 | M1 |
| GOPROXY 国内镜像 | T002 | M1 |
| golangci-lint 配置 | T003 | M1 |
| Makefile + `.PHONY` | T004 | M1 |
| Go 项目标准布局 `cmd/` `pkg/` | T001 | M1 |
| 白盒/黑盒测试 + `t.Fatalf` vs `t.Errorf` + `%q` | T009 | M1 |
| 逃逸分析 + nil 接口陷阱 | T013 | M1 |
| `init()` 自动初始化函数 | T014 | M1 |
| `yaml.Unmarshal` 指针参数 | T012 | M1 |

---

> **宪法 6.1 确认**：本学习笔记同等覆盖了 Go 语法特性（第一章）、Android/Frida 逆向原理（第二章）、SpecKit 工作流与 AI 编程认知突破（第三章）。教学完成，可以进入 `/speckit.implement`。
