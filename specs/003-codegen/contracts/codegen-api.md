# 接口契约: Codegen Generator

**功能**: 003-codegen | **日期**: 2026-05-18

## 包: `pkg/codegen`

### 类型: `Generator`

```go
// NewGenerator 从内嵌模板创建代码生成器。
// 模板文件由 //go:embed 内嵌，在 NewGenerator 时编译。
// 编译失败返回 *TemplateError。
// logger 为 nil 时使用 slog.Default()。
func NewGenerator(logger *slog.Logger) (*Generator, error)

// Generate 从 HookSpec 生成完整的 Frida JavaScript 脚本。
// fridaVersion 为目标 Frida 版本 ("16" 或 "17")，模板据此选择 API 策略。
// 返回 *GenerateOutput，包含组合后的完整脚本和各 Hook 独立代码段。
// spec 为 nil 或 hooks 为空时返回 *GenerateError。
func (g *Generator) Generate(spec *spec.HookSpec, fridaVersion string) (*GenerateOutput, error)
```

### 类型: `GenerateOutput`

```go
type GenerateOutput struct {
    Combined string            // 完整可执行 JavaScript 脚本
    Scripts  []GeneratedScript // 各 Hook 独立生成结果
}
```

### 类型: `GeneratedScript`

```go
type GeneratedScript struct {
    HookTarget spec.HookTarget // 原始 Hook 目标
    JSCode     string          // 生成的 JavaScript 代码段
}
```

### 类型: `RenderContext`

```go
// RenderContext 为模板渲染提供数据上下文。
// 非导出类型，仅内部使用。
type RenderContext struct {
    AppPackage      string // 父级 HookSpec.AppPackage
    ClassName       string // 目标类名 (Java hooks)
    MethodName      string // 目标方法/函数名
    HookType        string // "overload" / "override" / "native"
    MethodSignature string // 参数签名整串 (空→无签名)
    ModuleName      string // Native .so 模块名
    FridaVersion    string // 目标 Frida 版本 ("16" 或 "17")
}
```

### 错误构造函数

```go
func NewTemplateError(op, name string, err error) *TemplateError
func NewGenerateError(op string, err error) *GenerateError
```

### 错误类型

```go
type TemplateError struct { Op, Name string; Err error }
func (e *TemplateError) Error() string
func (e *TemplateError) Unwrap() error

type GenerateError struct { Op string; Err error }
func (e *GenerateError) Error() string
func (e *GenerateError) Unwrap() error
```

## 调用示例

```go
package main

import (
    "fmt"
    "os"

    "github.com/bigsmater/fridaforge/pkg/codegen"
    "github.com/bigsmater/fridaforge/pkg/config"
)

func main() {
    spec, err := config.LoadSpec("hooks.yaml")
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    if err := config.Validate(spec); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    gen, err := codegen.NewGenerator(nil)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    output, err := gen.Generate(spec)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    fmt.Println(output.Combined)
}
```

## 模板文件契约

内嵌模板由 `//go:embed templates/*.js.tmpl` 注入，文件名 = 模板名:

| 模板文件 | HookType | 使用的 Frida API |
|----------|----------|-----------------|
| `overload.js.tmpl` | overload | `Java.use().class.method.overload(sig).implementation` + `this.method()` 原调用 |
| `override.js.tmpl` | override | `Java.use().class.method.overload(sig).implementation` (无原调用) |
| `native.js.tmpl` | native | `Process.findModuleByName()` + `module.findExportByName()` + `Interceptor.attach()` |

所有 Java 模板产生单段代码（不含 `Java.perform()` wrapper）——`Generate()` 负责组装 wrapper。

## 线程安全

`Generator.Generate()` 是并发安全的——无共享可变状态，纯函数风格。多个 goroutine 可同时调用 `Generate()`。
