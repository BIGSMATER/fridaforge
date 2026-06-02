// Package mcpserver 提供 FridaForge 的 MCP Server 实现。
// 通过 stdin/stdout 与 AI 编码工具通信，暴露 Hook 脚本生成、
// 参数校验、设备枚举、进程枚举四项能力。
package mcpserver

import (
	"context"

	"github.com/bigsmater/fridaforge/pkg/device"
)

// GenerateInput 是 spec_generate Tool 的输入参数
type GenerateInput struct {
	AppPackage string `json:"app_package" jsonschema:"目标应用包名，如 com.example.app,required"`
	ClassName  string `json:"class_name" jsonschema:"目标类全限定名，如 com.example.Test,required"`
	MethodName string `json:"method_name" jsonschema:"目标方法名,required"`
	HookType   string `json:"hook_type" jsonschema:"Hook 类型: overload/override/native,required"`
	Signature  string `json:"signature,omitempty" jsonschema:"方法参数签名，仅 overload 时可选"`
	ModuleName string `json:"module_name,omitempty" jsonschema:"原生 .so 模块名，仅 native 时必填"`
}

// GenerateOutput 是 spec_generate Tool 的输出结果
type GenerateOutput struct {
	Script string `json:"script" jsonschema:"生成的完整 Frida JavaScript 脚本"`
}

// ValidateInput 是 spec_validate Tool 的输入参数（与 GenerateInput 一致）
type ValidateInput struct {
	AppPackage string `json:"app_package" jsonschema:"目标应用包名,required"`
	ClassName  string `json:"class_name" jsonschema:"目标类全限定名,required"`
	MethodName string `json:"method_name" jsonschema:"目标方法名,required"`
	HookType   string `json:"hook_type" jsonschema:"Hook 类型: overload/override/native,required"`
	Signature  string `json:"signature,omitempty" jsonschema:"方法参数签名，仅 overload 时可选"`
	ModuleName string `json:"module_name,omitempty" jsonschema:"原生 .so 模块名，仅 native 时必填"`
}

// ValidationFieldError 表示单个字段的校验错误
type ValidationFieldError struct {
	Path    string `json:"path" jsonschema:"错误字段路径，如 hooks[0].class_name"`
	Message string `json:"message" jsonschema:"错误描述"`
}

// ValidateOutput 是 spec_validate Tool 的输出结果
type ValidateOutput struct {
	Valid    bool                   `json:"valid" jsonschema:"校验是否通过"`
	Errors   []ValidationFieldError `json:"errors" jsonschema:"字段校验错误列表"`
	Warnings []ValidationFieldError `json:"warnings" jsonschema:"警告信息列表，如重复 Hook"`
}

// DeviceListItem 表示设备列表中的单个设备
type DeviceListItem struct {
	ID          string `json:"id" jsonschema:"设备唯一标识"`
	Name        string `json:"name" jsonschema:"设备名称"`
	ConnectType string `json:"connect_type" jsonschema:"连接类型: usb/network/emulator"`
}

// DeviceListOutput 是 device_list Tool 的输出结果
type DeviceListOutput struct {
	Devices []DeviceListItem `json:"devices" jsonschema:"已连接设备列表"`
}

// ProcessListInput 是 process_list Tool 的输入参数
type ProcessListInput struct {
	DeviceID string `json:"device_id" jsonschema:"目标设备标识,required"`
}

// ProcessListItem 表示进程列表中的单个进程
type ProcessListItem struct {
	PID  int    `json:"pid" jsonschema:"进程 ID"`
	Name string `json:"name" jsonschema:"进程名称"`
}

// ProcessListOutput 是 process_list Tool 的输出结果
type ProcessListOutput struct {
	Processes []ProcessListItem `json:"processes" jsonschema:"进程列表"`
}

// ProcessLister 定义进程枚举接口。
// 通过依赖注入解耦，M4 初期注入 StubProcessLister，
// 后续可替换为基于 frida-go 的真实实现。
type ProcessLister interface {
	ListProcesses(ctx context.Context, deviceID string) ([]ProcessListItem, error)
}

// StubProcessLister 是 ProcessLister 的桩实现，用于模拟模式
type StubProcessLister struct {
	processesByDevice map[string][]ProcessListItem
}

// ListProcesses 返回指定设备的模拟进程列表
func (s *StubProcessLister) ListProcesses(ctx context.Context, deviceID string) ([]ProcessListItem, error) {
	procs, ok := s.processesByDevice[deviceID]
	if !ok {
		return nil, nil
	}
	return procs, nil
}

// Server 是 MCP Server 的核心结构体，持有所有依赖
type Server struct {
	deviceLister  device.DeviceLister
	processLister ProcessLister
	// codegen generator 在 server.go 中通过 NewMCPServer 传入
}
