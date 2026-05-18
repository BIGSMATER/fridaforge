# 数据模型: 声明式代码生成器

**功能**: 003-codegen | **日期**: 2026-05-18

## 核心实体

### 1. Generator (生成器)

| 属性 | 类型 | 说明 |
|------|------|------|
| tmpl | *template.Template | 预编译的模板集 (非导出) |
| logger | *slog.Logger | 结构化日志 |

**生命周期**: `NewGenerator()` 创建 → 多次调用 `Generate()` → 进程退出（无 Close 需要）

**约束**: 无状态，无需 Mutex。`Generate()` 为纯函数。

### 2. GenerateOutput (生成输出)

| 属性 | 类型 | 说明 |
|------|------|------|
| Combined | string | 完整可执行脚本（Java.perform 包装 + Native hooks） |
| Scripts | []GeneratedScript | 各 Hook 独立代码段 |

**Combined 结构规则**:
```
Java.perform(function() {
    // script[0].JSCode  (overload/override hook 1)
    // script[1].JSCode  (overload/override hook 2)
    // ...
});
// script[N].JSCode      (native hook — 不在 Java.perform 内)
```

### 3. GeneratedScript (单个生成脚本)

| 属性 | 类型 | 说明 |
|------|------|------|
| HookTarget | spec.HookTarget | 原始 Hook 目标引用 |
| JSCode | string | 生成的 JavaScript 代码段（不含 Java.perform 包装） |

### 4. RenderContext (模板渲染上下文)

| 属性 | 类型 | 说明 |
|------|------|------|
| AppPackage | string | 父级 HookSpec.AppPackage |
| ClassName | string | 目标类全限定名 (Java hooks) |
| MethodName | string | 目标方法/函数名 |
| HookType | string | "overload" / "override" / "native" |
| MethodSignature | string | 参数签名整串（空 → 不传 signature）。M3 不做分割预处理 (clarify Q3 决议: 整串原样插入) |
| ModuleName | string | Native hook 的 .so 模块名 |

**渲染规则**:
- `MethodSignature` 为空 → 模板中使用 `.overload()` (无参数)
- `MethodSignature` 非空 → 模板中使用 `.overload('{{.MethodSignature}}')` (整串)
- `HookType == "native"` → 忽略 `ClassName` + `AppPackage`，使用 `ModuleName`

## 错误类型

```go
// TemplateError — 模板编译/渲染错误
type TemplateError struct {
    Op   string // 操作: "parse" (编译) 或 "render" (渲染)
    Name string // 模板文件名
    Err  error  // 底层 text/template 错误
}

// GenerateError — 生成过程错误
type GenerateError struct {
    Op  string // 操作: "generate"
    Err error  // 底层错误
}
```

两者均实现 `error` 接口和 `Unwrap() error` 方法。

## 数据关系图

```
spec.HookSpec (M1 已有)
  ├── AppPackage: string
  └── Hooks: []spec.HookTarget
        ├── ClassName: string
        ├── MethodName: string
        ├── HookType: HookType           ← 新增 native
        ├── MethodSignature: string      ← 新增
        └── ModuleName: string           ← 新增

          │
          │ Generator.Generate(spec)
          ▼

      GenerateOutput
        ├── Combined: string          ← 完整 JS 脚本
        └── Scripts: []GeneratedScript
              ├── HookTarget: spec.HookTarget
              └── JSCode: string      ← 单 Hook 代码段

          │
          │ 中间态 (内部)
          ▼
      RenderContext
        └── (HookTarget 字段 + AppPackage)
              │
              │ template.Execute()
              ▼
          JSCode (字符串)
```

## 状态机

无状态机。Generator 是无状态纯函数映射：`HookSpec → GenerateOutput`。
