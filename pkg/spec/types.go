package spec

// HookType 表示 Hook 的类型：重载（overload）或替换（replace）。
type HookType string

const (
	// HookTypeOverload 表示重载模式：在原方法前后插入代码，保留原方法调用。
	HookTypeOverload HookType = "overload"
	// HookTypeReplace 表示替换模式：完全替换原方法实现。
	HookTypeReplace HookType = "replace"
)

// HookSpec 表示一个 YAML 规格文件的整体结构。
// 一个文件对应一个目标应用。
type HookSpec struct {
	AppPackage string       `yaml:"app_package"`
	Hooks      []HookTarget `yaml:"hooks"`
}

// HookTarget 表示单条 Hook 声明，针对特定类的特定方法。
// 注意：HookTarget 不重复存储 app_package，该值从父级 HookSpec.AppPackage 获取。
// YAML 中 hooks 列表的每条记录仅需声明 class_name、method_name、hook_type。
type HookTarget struct {
	ClassName  string   `yaml:"class_name"`
	MethodName string   `yaml:"method_name"`
	HookType   HookType `yaml:"hook_type"`
}
