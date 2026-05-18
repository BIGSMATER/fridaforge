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

> 📝 补充于 Phase 4: 项目实际代码 — `pkg/codegen/templates.go` 中用 `template.ParseFS()` 从内嵌 FS 编译模板

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

> 📝 补充于 Phase 4: 项目实际代码 — `pkg/codegen/templates.go` 中用 `//go:embed templates/*.js.tmpl` 内嵌 3 个模板

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

> 📝 补充于 Phase 4: 项目实际代码 — `pkg/codegen/templates.go` 中用 `strings.Builder` 累积各 Hook 渲染结果

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
