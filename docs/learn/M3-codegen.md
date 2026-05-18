# M3 学习笔记：Go 模板引擎与声明式代码生成

> Milestone: M3 | 状态: 进行中
> 三轨并行：Go 语言 / Android 逆向 / AI 编程范式
> 前置：本阶段假设学员已掌握 M1-M2 内容（package/struct/error/cobra/concurrency）

---

## 一、Go 语言轨道：模板引擎与资源嵌入

M3 引入 Go 的三个标准库概念：`text/template`（模板引擎）、`embed.FS`（编译时文件内嵌）、`strings.Builder`（高效字符串拼接）。组合使用它们可以构建出**零外部依赖的代码生成器**。

### 1.1 text/template — 数据驱动的文本生成

`text/template` 是 Go 标准库的模板引擎，核心思想是：**定义文本模板 + 注入数据 → 输出渲染文本**。

```go
package main

import (
    "os"
    "strings"
    "text/template"
)

type Person struct {
    Name string
    Age  int
}

func main() {
    tmplStr := "Hello, my name is {{.Name}}. I am {{.Age}} years old."
    tmpl, _ := template.New("greeting").Parse(tmplStr)

    data := Person{Name: "BIGSMATER", Age: 28}
    var buf strings.Builder
    tmpl.Execute(&buf, data)
    os.Stdout.WriteString(buf.String())
    // 输出: Hello, my name is BIGSMATER. I am 28 years old.
}
```

**核心语法**：

| 语法 | 含义 | 示例 |
|------|------|------|
| `{{.Field}}` | 注入字段值 | `{{.ClassName}}` |
| `{{.Nested.Field}}` | 注入嵌套字段 | 无（M3 用扁平 RenderContext） |
| `{{if .Cond}}...{{end}}` | 条件渲染 | 判断 MethodSignature 是否为空 |
| `{{range .Items}}...{{end}}` | 循环渲染 | 无（M3 signature 不分割） |

**与 `html/template` 的区别**：`html/template` 会自动 HTML 转义——它会将 `<` 转成 `&lt;`。生成 JavaScript 代码时这是致命的（JavaScript 中 `<` 是正常运算符）。**生成 JS 必须用 `text/template`**。

> ⚠️ 项目实际代码 → 见下方 §1.4

### 1.2 embed.FS — 编译时文件内嵌

`embed` 包允许在编译时把文件"嵌入"到 Go 二进制中。生成的程序不依赖外部文件。

```go
package main

import (
    "embed"
    "fmt"
)

//go:embed data/*.txt
var staticFS embed.FS

func main() {
    data, _ := staticFS.ReadFile("data/hello.txt")
    fmt.Println(string(data))

    entries, _ := staticFS.ReadDir("data")
    for _, e := range entries {
        fmt.Println(e.Name())
    }
}
```

**关键规则**：
- `//go:embed` 必须是紧邻 `var` 声明的注释
- 路径是相对于 **源文件所在目录** 的
- 支持 glob 模式：`templates/*.js.tmpl` 匹配所有 `.js.tmpl` 文件
- 内嵌发生在编译期——运行时无法修改内嵌文件

**M3 中的用法**：3 个 `.js.tmpl` 模板文件在编译时嵌入，运行时通过 `template.ParseFS()` 直接编译。

### 1.3 strings.Builder — 高效字符串拼接

在循环中拼接字符串时，用 `+` 每次都会分配新内存。`strings.Builder` 是官方推荐的高效方案。

```go
package main

import (
    "fmt"
    "strings"
)

func main() {
    var b strings.Builder

    b.WriteString("Hello")
    b.WriteString(", ")
    b.WriteString("World!")
    // 内部用 []byte 缓冲，避免重复分配

    result := b.String()    // 一次性输出
    fmt.Println(result)
    fmt.Println("长度:", b.Len())
}
```

**性能**：`strings.Builder` 在拼接 >10 次时比 `+` 快 10-100 倍。

**M3 中的用法**：每个 Hook 的模板渲染结果写入同一个 `strings.Builder`，一次性返回完整 Combined 脚本。

### 1.4 项目实际代码： templates.go 中的模板编译与渲染

```go
// 项目代码片段 (pkg/codegen/templates.go) — 讲解见 Phase 4
```

---

## 二、Android 逆向轨道：Frida JS API 全景

M3 生成的是 Frida JavaScript 脚本。理解 3 种 Hook 模式对应的 Frida API 是正确生成代码的前提。

### 2.1 Java.perform — JVM 上下文入口

```javascript
Java.perform(function() {
    // 此回调在目标应用的 JVM 线程中执行
    // 所有 Java.use() / Java.choose() 调用必须在此内部
});
```

`Java.perform()` 确保回调在 Dalvik/ART 虚拟机上下文中执行——Java bridge 不可在外部使用。

### 2.2 Java.use() + .implementation — 方法替换

```javascript
Java.perform(function() {
    var Crypto = Java.use("com.example.Crypto");

    // overload 模式 — 保留原方法调用
    Crypto.encrypt.overload().implementation = function(input) {
        console.log("[+] encrypt called with: " + input);
        var result = this.encrypt(input);   // ← 调用原方法
        console.log("[+] encrypt result: " + result);
        return result;
    };

    // override 模式 — 完全替换
    Crypto.getKey.implementation = function() {
        console.log("[+] getKey hooked (override)");
        return "static-key-override";       // ← 返回假值
    };
});
```

**关键区别**：
- `overload`：`this.method(args)` 调用原方法，保留原行为，只记录日志
- `override`：不调用原方法，自定义逻辑和返回值

### 2.3 overload() 与签名匹配

```javascript
// 无签名 → 匹配第一个同名方法
Target.method.overload().implementation = function() { ... }

// 有签名 → 精确匹配参数类型
Target.method.overload('java.lang.String', 'int').implementation = function(s, n) { ... }
```

M3 的 `method_signature` 字段直接渲染为 `.overload('用户写的整串')`。

### 2.4 Interceptor.attach — Native 层 Hook

```javascript
var module = Process.findModuleByName("libnative-lib.so");
if (module !== null) {
    var addr = Module.findExportByName("libnative-lib.so", "open");
    if (addr !== null) {
        Interceptor.attach(addr, {
            onEnter: function(args) {
                console.log("[+] open() called, fd param: " + args[0].toInt32());
            },
            onLeave: function(retval) {
                console.log("[+] open() returned: " + retval.toInt32());
            }
        });
    }
}
```

### 2.5 send() — 向宿主机发送消息

所有 Hook 回调都应该用 `send()` 将数据发回 frida-server，再由 FridaForge engine 的 channel 接收。

```javascript
send(JSON.stringify({
    event: "enter",
    class: "com.example.Crypto",
    method: "encrypt",
    timestamp: Date.now()
}));
```

### 2.6 项目实际代码：生成的 JS 脚本示例

```javascript
// 项目生成的完整脚本示例 (来自 quickstart.md 测试场景) — 讲解见 Phase 5
```

---

## 三、AI 编程范式轨道：代码生成器设计哲学

### 3.1 声明式 vs 命令式 — 为什么需要代码生成

| 声明式 (YAML) | 命令式 (Frida JS) |
|---------------|-------------------|
| 描述 **What** | 实现 **How** |
| `hook_type: overload` | `this.encrypt(input)` + `send()` |
| 用户可读、可审计 | Frida 可执行 |
| 10 行 YAML | 100 行 JS |

代码生成器是两者的"桥"：**Spec → Template → Script**。

### 3.2 模板即契约

在 SpecKit 工作流中，模板文件（`.js.tmpl`）承担了"契约"角色：
- **Spec** 定义 "What"（用户故事和需求）
- **Plan** 定义 "How"（架构和包结构）
- **Template** 是 "How" 的具象化——每种 HookType 对应一个模板
- **Tasks** 拆解实现步骤

模板一旦写好，任何 HookTarget 都能生成正确的代码——这就是声明式的力量。

### 3.3 SpecKit 工作流回顾：M3 全流程

```
/speckit.specify  → spec.md      (4 User Stories, 18 FR)
/speckit.clarify  → 3 个边界问题澄清
/speckit.plan     → plan.md + research.md + data-model.md + contracts/
/speckit.tasks    → 36 tasks, 7 phases
/speckit.analyze  → 8 findings, 全修复
/speckit.implement → 当前 ← 教学-编码交替
```

### 3.4 项目实际代码：codegen 架构设计

```go
// 项目架构示意图 — 讲解见 Phase 5
```
