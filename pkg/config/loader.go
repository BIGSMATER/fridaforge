package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/bigsmater/fridaforge/pkg/spec"
)

// LoadSpec 从 YAML 文件路径加载并解析 Hook 规格。
// 返回 *spec.HookSpec 和可能的 I/O 或解析错误。
func LoadSpec(path string) (*spec.HookSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取文件 %s: %w", path, err)
	}

	var s spec.HookSpec
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("YAML 解析失败: %w", err)
	}

	return &s, nil
}
