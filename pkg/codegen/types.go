// Package codegen 提供声明式 Frida JavaScript 代码生成能力。
// 读取 HookSpec（来自 pkg/spec），通过内嵌模板渲染可执行的 Frida JS 脚本。
package codegen

import "github.com/bigsmater/fridaforge/pkg/spec"

// GenerateOutput 表示一次 Generate() 调用的完整输出。
type GenerateOutput struct {
	// Combined 是完整的可执行 Frida JavaScript 脚本：
	// 所有 Java hooks 包裹在单个 Java.perform() 内，Native hooks 追加在其后。
	Combined string

	// Scripts 包含每个 HookTarget 对应的独立生成代码段。
	Scripts []GeneratedScript
}

// GeneratedScript 将 HookTarget 与生成的 JavaScript 代码配对。
type GeneratedScript struct {
	HookTarget spec.HookTarget
	JSCode     string
}

// RenderContext 提供模板渲染时注入的数据上下文。
// 由 Generator.Generate() 在调用 renderTemplate() 前填充。
type RenderContext struct {
	AppPackage      string // 父级 HookSpec.AppPackage
	ClassName       string // 目标类全限定名（Java hooks）
	MethodName      string // 目标方法名或函数名
	HookType        string // "overload" / "override" / "native"
	MethodSignature string // 参数签名整串（空 → 无签名匹配）
	ModuleName      string // .so 模块名（仅 Native hooks）
	FridaVersion    string // 目标 Frida 版本 ("16" 或 "17")，模板据此选择 API
}
