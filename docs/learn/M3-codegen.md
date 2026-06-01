# M3 学习笔记：Go 模板引擎与声明式代码生成

> Milestone: M3 | 状态: 进行中 (Phase 1/7)
> 三轨并行：Go 语言 / Android 逆向 / AI 编程范式
> 前置：本阶段假设学员已掌握 M1-M2 内容（package/struct/error/cobra/concurrency）

---

## 一、Go 语言轨道：模板引擎与资源嵌入

M3 引入 Go 的三个标准库概念：`text/template`（模板引擎）、`embed.FS`（编译时文件内嵌）、`strings.Builder`（高效字符串拼接）。组合使用它们可以构建出**零外部依赖的代码生成器**。

### 1.1 text/template — 数据驱动的文本生成

`text/template` 是 Go 标准库的模板引擎，核心思想：**定义静态文本模板 + 注入动态数据 → 输出渲染文本**。

```go
package main

import (
    "os"
    "text/template"
)

type Person struct {
    Name string
    Age  int
}

func main() {
    // 定义模板 —— 注意 {{ }} 内的 .Name .Age 是数据字段
    tmplStr := "Hello, my name is {{.Name}}. I am {{.Age}} years old."
    tmpl, _ := template.New("greeting").Parse(tmplStr)

    // 注入数据 —— Execute 把渲染结果写到 io.Writer
    tmpl.Execute(os.Stdout, Person{Name: "BIGSMATER", Age: 28})
}
// 输出: Hello, my name is BIGSMATER. I am 28 years old.
```

**核心语法**：

| 语法 | 含义 | M3 中的用途 |
|------|------|------------|
| `{{.Field}}` | 注入字段值 | `{{.ClassName}}` → 类全限定名 |
| `{{if .Cond}}...{{else}}...{{end}}` | 条件渲染 | 判断 `MethodSignature` 是否为空，决定用 `.overload('sig')` 还是 `.overload()` |
| `{{range .Items}}...{{end}}` | 循环渲染 | 无（M3 不做签名分割，整串插入） |

**`text/template` vs `html/template`**：后者会自动 HTML 转义（`<` → `&lt;`）。生成 JavaScript 代码必须用 `text/template`，否则 JS 语法会被破坏。

```go
// 危险示范 — html/template 会毁掉 JS
// 模板: "if (a < b)"
// html/template 输出: "if (a &lt; b)"  ← 语法错误！
```

> 📝 项目实际代码：§1.9 展示了 `templates.go` 中 `//go:embed` + `template.ParseFS()` + `ExecuteTemplate()` 的完整调用链

### 1.2 embed.FS — 编译时文件内嵌

`embed` 包允许在**编译时**把文件目录"嵌入"到 Go 二进制中。生成的程序不需要任何外部文件。

```go
package main

import (
    "embed"
    "fmt"
)

//go:embed data/*.txt
var staticFS embed.FS

func main() {
    // 从内嵌 FS 读取文件 —— 跟 os.ReadFile 一样的 API
    data, _ := staticFS.ReadFile("data/hello.txt")
    fmt.Println(string(data))

    // 遍历内嵌目录
    entries, _ := staticFS.ReadDir("data")
    for _, e := range entries {
        fmt.Println(e.Name())
    }
}
```

**关键规则**：
- `//go:embed` 必须是紧邻 `var` 声明的注释，中间不能有空行
- 路径相对源文件所在目录
- 支持 glob：`templates/*.js.tmpl` 匹配所有 .js.tmpl 文件
- 内嵌发生在编译期 —— 运行时**不可修改**

**可以内嵌的东西**：
- 单个文件：`//go:embed hello.txt`
- 整个目录：`//go:embed static/*`
- 不支持 `..` 父目录引用
- 不支持指向源文件目录之外的路径

> 📝 项目实际代码：§1.9 展示了完整的 embed.FS + ParseFS 组合

### 1.3 strings.Builder — 高效字符串拼接

常规字符串拼接的问题：

```go
// 每次 + 都分配新内存 —— 拼接 N 次分配 N 次
s := ""
for i := 0; i < 1000; i++ {
    s += "hello"   // 每次循环：分配新内存 + 拷贝旧内容 + 追加新内容
}
```

`strings.Builder` 内部用 `[]byte` 缓冲，避免重复分配：

```go
var b strings.Builder
for i := 0; i < 1000; i++ {
    b.WriteString("hello")  // 追加到内部 buffer，不分配
}
result := b.String()        // 一次性转为 string
```

**性能对比**：拼接 >10 次时 Builder 比 `+` 快 10-100 倍。

**四个核心方法**：
- `WriteString(s)` — 追加字符串
- `WriteByte(c)` — 追加单个字节
- `Len()` — 当前内容长度
- `String()` — 输出最终结果

### 1.4 类型分发 — switch 语句

校验器需要根据 `HookType` 对不同字段做不同校验。Go 的 `switch` 不需要 `break`（自动隐含），多 case 用逗号并列。

```go
func validate(name string) {
    switch name {
    case "a", "b":       // 多个 case 标签用逗号分隔
        fmt.Println("a or b")
    case "c":
        fmt.Println("c")
    default:              // 无匹配时执行
        fmt.Println("unknown")
    }
}
```

> ⚠️ 项目实际代码：

```go
// pkg/config/validator.go — 你写的代码
switch h.HookType {
case spec.HookTypeOverload, spec.HookTypeOverride:
    if h.ClassName == "" {
        ve.Add(prefix+".class_name", "不能为空", 0)
    }
case spec.HookTypeNative:
    if h.ModuleName == "" {
        ve.Add(prefix+".module_name", "不能为空（native Hook 需要 module_name）", 0)
    }
default:
    ve.Add(prefix+".hook_type",
        fmt.Sprintf("不支持的值 %q，有效的 Hook 类型: overload, override, native", h.HookType),
        0)
}
```

这就实现了"Java 类型要求 ClassName，Native 类型要求 ModuleName"的差异化校验——同一个 `for i, h := range s.Hooks` 循环，三种类型走三条不同的校验路径。

### 1.5 map[string]int — O(1) 哈希去重

重复 Hook 检测的核心数据结构：

```go
// 独立示例
func hasDuplicates(items []string) int {
    seen := make(map[string]int)  // key → 首次出现的索引
    for i, item := range items {
        if prevIdx, exists := seen[item]; exists {
            fmt.Printf("重复: %s (首次出现在索引 %d, 当前索引 %d)\n", item, prevIdx, i)
            return prevIdx
        }
        seen[item] = i
    }
    return -1
}
```

`map[string]int` 的查找是 O(1)（哈希表），比二层循环的 O(n²) 快得多。100 个 Hook 的重复检测：HashMap 方式 100 次比较，二层循环 100×99/2 = 4950 次比较。

> ⚠️ 项目实际代码：

```go
// pkg/config/validator.go — 你写的代码
seen := make(map[string]int)
for i, h := range s.Hooks {
    dupKey := fmt.Sprintf("%s|%s|%s|%s",
        h.ClassName, h.MethodName, h.MethodSignature, h.HookType)
    if prevIdx, exists := seen[dupKey]; exists {
        ve.AddWarning(prefix+".hook_type",
            fmt.Sprintf("与 hooks[%d] 重复 ...", prevIdx), 0)
    } else {
        seen[dupKey] = i
    }
}
```

四个字段用 `|` 拼接成复合 key——只要有一个字段不同，key 就不同（不会误判为重复）。

### 1.6 struct tag omitempty — 可选字段

```go
type HookTarget struct {
    ClassName       string   `yaml:"class_name"`
    MethodSignature string   `yaml:"method_signature,omitempty"`  // ← omitempty
    ModuleName      string   `yaml:"module_name,omitempty"`        // ← omitempty
}
```

`omitempty` 的含义：
- **序列化时**：字段值为零值（空字符串 `""`、0、nil）→ 不输出
- **反序列化时**：YAML 中没有这个字段 → 字段保留零值（空字符串）

所以 YAML 可以写：

```yaml
hooks:
  - class_name: com.example.Foo
    method_name: bar
    hook_type: overload
    # method_signature 和 module_name 都是可选的，不写也行
```

### 1.7 ValidationError 的 Warnings 字段

原来的 `ValidationError` 只有 `Errors []FieldError`。Phase 2 新增了 `Warnings []FieldError`，因为"重复 Hook"不应阻止生成（只是提醒），但需要通过某种方式通知调用者。

```go
// 关键设计决策：Warnings 不是 Errors
type ValidationError struct {
    Errors   []FieldError  // 阻止生成
    Warnings []FieldError  // 不阻止生成，但应展示
}

// Validate 返回逻辑：
// - HasErrors() → 返回 error（拒绝）
// - HasWarnings() 但 !HasErrors() → 返回 error（但 Error() 字符串为空）
//   → 调用者判断 err != nil 然后检查 ve.HasWarnings()
```

**为什么返回 error 而不是 nil？** 调用者需要知道"有 warning"这个事实。Go 的 `error` 接口是最自然的传递方式——即使没有 Errors，一个携带 Warnings 的 ValidationError 也是合法的 error 值。

### 1.8 Go 的 error 接口与自定义错误类型

Phase 3 创建了 `pkg/codegen/types.go` 和 `pkg/codegen/errors.go`。核心 Go 概念：**error 接口 + Unwrap 错误链**。

#### 1.8.1 Go 的 error 接口 — 最简单也最强大的接口

Go 的 `error` 接口只有一个方法：

```go
// 标准库的 error 接口 — 完整定义
type error interface {
    Error() string
}
```

任何实现了 `Error() string` 的类型都是合法的 error。这意味着你可以创建**携带结构化信息**的错误：

```go
// ❌ 字符串错误 — 机器无法提取信息
fmt.Errorf("template parse failed: %v", err)

// ✅ 结构化错误 — errors.Is / errors.As 可精确判断
type TemplateError struct {
    Op   string  // "parse" 还是 "render"?
    Name string  // 哪个模板出错了?
    Err  error   // 原始错误是什么?
}
```

**对比其他语言**：Java/C# 用 `try/catch` + 异常类继承；Go 用返回值传递错误 + 接口统一。Go 的方式更显式——你无法"忘记处理"一个错误，因为它就是一个普通的返回值。

#### 1.8.2 Unwrap() — 错误链的穿透

Go 1.13 引入 `Unwrap() error` 约定——如果错误实现了这个方法，`errors.Is/As` 就能穿透错误链：

```go
func (e *TemplateError) Unwrap() error {
    return e.Err
}

// 使用方：
var te *TemplateError
if errors.As(err, &te) {
    fmt.Println("模板操作失败:", te.Op)
    fmt.Println("底层原因:", te.Unwrap())
}
```

**类比快递包裹**：
```
外包装: "codegen: parse template 'overload.js.tmpl': ..."
  │ .Unwrap()
  ▼
内层:   "template: overload.js.tmpl:3: unexpected error in operand"
  │ .Unwrap()
  ▼
最内层: *text/template 的原始错误对象
```

#### 1.8.3 项目实际代码

```go
// pkg/codegen/types.go — 你写的代码
type GenerateOutput struct {
    Combined string            // 完整可执行 Frida JS 脚本
    Scripts  []GeneratedScript // 各 Hook 独立代码段
}

type GeneratedScript struct {
    HookTarget spec.HookTarget // 原始 Hook 目标
    JSCode     string          // 生成的 JS 代码段
}

type RenderContext struct {
    AppPackage      string // 父级 HookSpec.AppPackage
    ClassName       string // 目标类名 (Java hooks)
    MethodName      string // 目标方法/函数名
    HookType        string // "overload" / "override" / "native"
    MethodSignature string // 参数签名整串 (空→无签名)
    ModuleName      string // Native .so 模块名
}
```

**RenderContext 设计原则**：扁平结构体——所有字段都是标量。`text/template` 的 `{{.Field}}` 适合扁平对象，嵌套增加模板复杂度。所以 `HookTarget` 的字段被"展开"成 `RenderContext` 的一级字段，外加来自父级 `HookSpec` 的 `AppPackage`。

```go
// pkg/codegen/errors.go — 你写的代码
type TemplateError struct {
    Op   string // "parse" (编译) 或 "render" (渲染)
    Name string // 模板文件名 (如 "overload.js.tmpl")
    Err  error  // 底层 text/template 错误
}

type GenerateError struct {
    Op  string // "generate"
    Err error  // 底层错误
}
```

**什么时候用哪个？**
- `TemplateError`：`NewGenerator()` 编译模板失败——开发期 bug（fail-fast）
- `GenerateError`：`Generate()` 运行时非法状态——比如 nil HookSpec

两者都实现 `Error()` + `Unwrap()` — 跟 M2 的 `DeviceError/SessionError/ScriptError` 一样的三明治模式。

> 📝 项目实际代码：§1.9 展示了 strings.Builder 在 renderTemplate() 中的使用

### 1.9 templates.go 项目代码 — embed + Parse + Execute 三件套

Phase 4 的核心是 `pkg/codegen/templates.go`——这是 `//go:embed`、`template.ParseFS`、`template.ExecuteTemplate` 三者首次在项目中组合使用。

```go
// pkg/codegen/templates.go — 你写的代码

//go:embed templates/*.js.tmpl
var templateFS embed.FS

type Generator struct {
    tmpl   *template.Template
    logger *slog.Logger
}

func NewGenerator(logger *slog.Logger) (*Generator, error) {
    // 一次 ParseFS 编译所有模板 — 不用逐个 Parse()
    tmpl, err := template.ParseFS(templateFS, "templates/*.js.tmpl")
    if err != nil {
        return nil, &TemplateError{Op: "parse", Err: err}  // fail-fast
    }
    return &Generator{tmpl: tmpl, logger: logger}, nil
}

func (g *Generator) renderTemplate(ctx RenderContext) (string, error) {
    // 按 HookType 选择模板文件
    var name string
    switch ctx.HookType {
    case "overload": name = "overload.js.tmpl"
    case "override": name = "override.js.tmpl"
    case "native":   name = "native.js.tmpl"
    default:
        return "", &TemplateError{Op: "render", Err: fmt.Errorf("unknown hook type")}
    }

    // strings.Builder 作为 io.Writer — 模板直接写进去
    var buf strings.Builder
    if err := g.tmpl.ExecuteTemplate(&buf, name, ctx); err != nil {
        return "", &TemplateError{Op: "render", Name: name, Err: err}
    }
    return buf.String(), nil
}
```

**设计要点**：
1. `//go:embed templates/*.js.tmpl` — 编译时 3 个模板文件嵌入二进制
2. `template.ParseFS(fs, pattern)` — 一次调用编译所有匹配的模板，不需逐个 `Parse()`
3. `ExecuteTemplate(&buf, name, ctx)` — 按模板名选择渲染，数据注入 RenderContext
4. `strings.Builder` 实现了 `io.Writer` 接口 — 模板自然输出到缓冲区
5. `switch ctx.HookType` — 类型分发选择模板文件，跟 Phase 2 validator 一样的模式
6. 错误全用自定义 `TemplateError` 包装（带 Op/Name/Err ）— fail-fast

---

## 二、Android 逆向轨道：Frida JS API 全景

M3 生成的是 Frida JavaScript 脚本。理解三种 Hook 模式对应的 Frida API 是正确生成代码的前提。以下每个 API 都是生成脚本的"语法组件"。

### 2.1 Java.perform — JVM 上下文入口

```javascript
Java.perform(function() {
    // 此回调在目标应用的 JVM 线程内执行
    // 所有 Java.use() / Java.choose() 必须在此内部
    var System = Java.use("java.lang.System");
    console.log(System.currentTimeMillis());
});
```

**为什么必须 `Java.perform()`？** Frida 的 Java bridge 需要 JNI 环境（JNIEnv 指针），而这个环境只在 ART 虚拟机线程中有效。`Java.perform()` 确保回调执行时 JNI 环境已就绪。

**M3 中**：所有 Java hook（overload/override）的代码被套在单个 `Java.perform()` 包装内，减少 bridge 上下文切换开销。

### 2.2 Java.use() + .implementation — Java 方法 Hook

```javascript
Java.perform(function() {
    var Crypto = Java.use("com.example.Crypto");

    // overload 模式 — 拦截，但保留原方法行为
    Crypto.encrypt.overload().implementation = function(input) {
        console.log("[+] encrypt called with: " + input);
        var result = this.encrypt(input);      // ← 调用原方法
        console.log("[+] encrypt returned: " + result);
        return result;                          // ← 返回原结果
    };
});
```

**.implementation 做了什么？** Frida 在 ART 层面修改方法的入口地址，把原方法的第一条指令改为"跳转到你的函数"。你的函数返回后，Frida 负责恢复调用者的状态。

**关键区别：overload vs override**：

| | overload | override |
|--|----------|----------|
| 是否调用原方法 | `this.method(args)` ✅ | ❌ |
| 原方法行为 | 完全保留（只加日志） | 完全替换 |
| 返回值 | 原返回值 | 自定义 |

### 2.3 overload() — 方法签名匹配

Java 允许重载（同名多方法），Frida 用 `.overload()` 区分：

```javascript
// 无参数 → 匹配第一个找到的同名方法
TargetClass.method.overload().implementation = function() { ... }

// 有签名 → 精确匹配参数类型列表
TargetClass.method.overload('java.lang.String', 'int').implementation = function(s, n) { ... }

// 多参数
TargetClass.method.overload('java.lang.String', 'byte[]', 'int').implementation = function(s, b, n) { ... }
```

**M3 的处理**：`method_signature` 字段整串原样注入。用户负责写 Frida 合法格式，系统不校验。

### 2.4 Interceptor.attach — Native 层函数 Hook

当目标函数在 .so 中（非 Java 方法），用 `Interceptor.attach()`：

```javascript
// 1. 找到 .so 模块
var module = Process.findModuleByName("libnative-lib.so");
if (module === null) {
    console.log("[-] Module not found");
    return;
}

// 2. 找到导出函数的地址
var addr = Module.findExportByName("libnative-lib.so", "open");
if (addr === null) {
    console.log("[-] Export not found");
    return;
}

// 3. 附加 Hook
Interceptor.attach(addr, {
    onEnter: function(args) {
        // args[0] 是第一个参数 (ARM64: x0, ARM32: r0)
        console.log("[+] open() called, fd=" + args[0].toInt32());
    },
    onLeave: function(retval) {
        // retval 是返回值
        console.log("[+] open() returned: " + retval.toInt32());
    }
});
```

**关键规则**：
- `args` 是 `NativePointer` 数组，需要用 `.toInt32()`, `.readCString()` 等方法读取
- `onLeave` 中修改 `retval.replace(newValue)` 可替换返回值
- 每次调用 `Interceptor.attach()` 都会执行一次你的回调（高频！）

### 2.5 send() — 向宿主机发送结构化消息

Hook 执行在目标进程内。要把数据传回你的开发机，用 Frida 的 `send()` API：

```javascript
send(JSON.stringify({
    event: "method_enter",
    class: "com.example.Crypto",
    method: "encrypt",
    args: ["plaintext data"],
    timestamp: Date.now()
}));
```

消息格式约定：
- `event`: 事件类型（enter / leave / override / native_enter / native_leave）
- 包含 class + method 标识来源
- 包含 timestamp 方便时序分析

M2 的 `HookSession.Messages()` channel 就是接收这些 `send()` 消息的。

### 2.6 Phase 2 补充：HookType 三态设计

Phase 2 完成了 `replace → override` 重命名并新增 `native`：

| 类型 | YAML 值 | Frida API | class_name | module_name |
|------|---------|-----------|-----------|-------------|
| overload | `"overload"` | `Java.use().overload(sig).implementation` + `this.method()` | 必填 | 不需要 |
| override | `"override"` | `Java.use().overload(sig).implementation` (不调用原方法) | 必填 | 不需要 |
| native | `"native"` | `Interceptor.attach()` | 不需要 | 必填 |

**设计考量**：为什么 native 用 `module_name` 而不是 `class_name`？

Frida 的 Native Hook 需要两个关键信息定位目标函数：
1. `.so` 文件名（`Process.findModuleByName("libc.so")`）
2. 导出函数名（`Module.findExportByName("libc.so", "open")`）

Android 的 Java 类有包名体系（`com.example.app`），Native 库只有文件名（`libnative-lib.so`）。二者不能混用同一个字段。Phase 2 的 validator 通过 `switch h.HookType` 判断：
- Java 方向（overload/override）：`class_name` 必填
- Native 方向：`module_name` 必填

> 📝 补充于 Phase 5: 项目实际生成脚本完整示例 — overload + override + native 混合

---

## 三、AI 编程范式轨道：代码生成器设计哲学

### 3.1 声明式 vs 命令式 — 为什么需要代码生成器

FridaForge 解决的核心问题：**让人写声明（YAML），让机器写命令（JS）**。

```
声明式 (YAML)                           命令式 (Frida JS)
──────────────────────────────────────  ─────────────────────────────────
hook_type: overload                     send({event: "enter", ...})
class_name: com.example.Crypto    →     var result = this.encrypt(input);
method_name: encrypt                    send({event: "leave", ...})
                                        return result;
```

一条 YAML 可能膨胀为 10 行 JS。10 条 YAML = 100 行 JS。手写 JS 容易出错（忘写 `send()`, 忘 `Java.perform`, 签名打错），而代码生成器保证每个 Hook 都遵循同一模式。

### 3.2 模板即契约

在 SpecKit 工作流中，模板（.js.tmpl）承担了"契约"角色：

```
spec.md          → 定义 WHAT（用户故事、功能需求）
plan.md          → 定义 HOW（架构、包结构）
data-model.md    → 定义实体（HookTarget → JS 代码的映射关系）
templates/*.tmpl → 定义 OUTPUT（How 的具象化——每种 HookType 一段模板）
tasks.md         → 定义 ORDER（逐步实现）
```

模板一旦写好，任何合法的 HookSpec 都能生成正确的 JS——这就是**声明式的力量**——你不需要每次验证"这段 JS 写对了吗"，YAML 就是正确答案的保证。

### 3.3 M3 在 SpecKit 全流程中的位置

```
/speckit.specify  → spec.md      (4 User Stories, 18 FR)       ✅
/speckit.clarify  → 3 个边界问题澄清                            ✅
/speckit.plan     → plan + research + data-model + contracts    ✅
/speckit.tasks    → 36 tasks, 7 phases                          ✅
/speckit.analyze  → 8 findings, 全修复                           ✅
/speckit.implement → Phase 1 (当前) ──→ Phase 7                 ⬅️ 进行中
```

> 📝 补充于 Phase 5: 项目架构实际代码 — `pkg/codegen/generator.go` 的 `Generate()` 方法展示 Spec→Template→Script 完整链条

### 3.4 Phase 2 补充：Semantic Versioning 与 Breaking Change

FridaForge 遵循 `0.x.y` 版本号（宪法 §5.3），这一阶段**允许 Breaking Change**。

Phase 2 做了一个 Breaking Change：删除了旧的 `hook_type: "replace"`，重命名为 `"override"`。

```
修改前                            修改后
───────                          ───────
HookTypeReplace = "replace"      HookTypeOverride = "override"
validator 接受 replace            validator 不再接受 replace
```

**为什么不等 1.0 再改？** `0.x.y` 版本意味着"API 不稳定"。实际上 Frida 社区的术语也是 `overload`（保留原调用）和 `override`（完全替换）。`replace` 这个词歧义太大——它无法区分"替换但保留原行为"和"完全替换"。

**实践中怎么处理？** 测试文件中的 YAML 夹具从 `hook_type: replace` 改为 `hook_type: override`。`spec/types_test.go` 中的常量测试从 `HookTypeReplace` 改为 `HookTypeOverride`。

在 SpecKit 工作流中，这种 Breaking Change 必须在 **specify 阶段**就声明（FR-015），在 **clarify 阶段**确认策略（硬 Breaking），在 **implement 阶段**干净利落地执行——不留别名，不搞"deprecated 过渡期"，因为项目还没有外部用户。这是 `0.x.y` 阶段的特权。
