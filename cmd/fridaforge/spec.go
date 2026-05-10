package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bigsmater/fridaforge/pkg/config"
)

func init() {
	rootCmd.AddCommand(specCmd)
	specCmd.AddCommand(specValidateCmd)
}

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Hook 规格文件操作",
	Long:  "管理 Frida Hook 规格文件（YAML），支持校验和后续的代码生成。",
}

var specValidateCmd = &cobra.Command{
	Use:   "validate <文件>",
	Short: "校验 Hook 规格 YAML 文件",
	Long:  "加载并校验一个 Hook 规格 YAML 文件的结构合法性。",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecValidate,
}

func runSpecValidate(cmd *cobra.Command, args []string) error {
	path := args[0]

	s, err := config.LoadSpec(path)
	if err != nil {
		return fmt.Errorf("无法加载文件: %w", err)
	}

	if err := config.Validate(s); err != nil {
		return fmt.Errorf("✗ 配置无效: %s\n%v", path, err)
	}

	fmt.Printf("✓ 配置有效: %s\n", path)
	fmt.Printf("  目标应用: %s\n", s.AppPackage)
	fmt.Printf("  Hook 数量: %d\n", len(s.Hooks))
	return nil
}
