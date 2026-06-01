package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bigsmater/fridaforge/pkg/codegen"
	"github.com/bigsmater/fridaforge/pkg/config"
)

func init() {
	rootCmd.AddCommand(specCmd)
	specCmd.AddCommand(specValidateCmd)
	specCmd.AddCommand(specGenerateCmd)
}

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Hook 规格文件操作",
	Long:  "管理 Frida Hook 规格文件（YAML），支持校验和代码生成。",
}

var specValidateCmd = &cobra.Command{
	Use:   "validate <文件>",
	Short: "校验 Hook 规格 YAML 文件",
	Long:  "加载并校验一个 Hook 规格 YAML 文件的结构合法性。",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecValidate,
}

var specGenerateCmd = &cobra.Command{
	Use:   "generate <文件>",
	Short: "生成 Frida JS Hook 脚本",
	Long:  "读取 YAML Hook 规格文件，校验后生成可执行的 Frida JavaScript 脚本。",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecGenerate,
}

var (
	generateOutput string
	generateTarget string
)

func init() {
	specGenerateCmd.Flags().StringVarP(&generateOutput, "output", "o", "",
		"输出文件路径 (默认: stdout)")
	specGenerateCmd.Flags().StringVarP(&generateTarget, "target", "t", "",
		"仅生成指定 className.methodName 的 Hook (精确匹配)")
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

func runSpecGenerate(cmd *cobra.Command, args []string) error {
	path := args[0]

	s, err := config.LoadSpec(path)
	if err != nil {
		return fmt.Errorf("无法加载文件: %w", err)
	}

	if err := config.Validate(s); err != nil {
		return fmt.Errorf("✗ 配置无效: %s\n%v", path, err)
	}

	gen, err := codegen.NewGenerator(nil)
	if err != nil {
		return fmt.Errorf("初始化代码生成器失败: %w", err)
	}

	out, err := gen.Generate(s)
	if err != nil {
		return fmt.Errorf("生成脚本失败: %w", err)
	}

	// 输出 Combined 脚本
	if generateTarget != "" {
		var filtered strings.Builder
		for _, script := range out.Scripts {
			targetKey := script.HookTarget.ClassName + "." + script.HookTarget.MethodName
			if targetKey == generateTarget {
				filtered.WriteString(script.JSCode)
				filtered.WriteString("\n")
			}
		}
		if filtered.Len() == 0 {
			return fmt.Errorf("未找到匹配的 Hook: %s", generateTarget)
		}
		out.Combined = filtered.String()
	}

	if generateOutput != "" {
		if err := os.WriteFile(generateOutput, []byte(out.Combined), 0644); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}
		fmt.Printf("✓ 脚本已生成: %s\n", generateOutput)
		fmt.Printf("  目标应用: %s, Hook 数量: %d\n", s.AppPackage, len(out.Scripts))
		return nil
	}

	fmt.Println(strings.TrimSpace(out.Combined))
	return nil
}
