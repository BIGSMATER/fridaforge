package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:           "fridaforge",
	Short:         "声明式 Frida 脚本工程化平台",
	Version:       version,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := checkEthicalDisclaimer(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

// checkEthicalDisclaimer 检查用户是否已接受伦理声明。
// 若 ~/.fridaforge/agreed 标记文件存在则跳过；
// 否则打印声明并要求用户输入 AGREE。
func checkEthicalDisclaimer() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("无法获取用户目录: %w", err)
	}
	markerFile := filepath.Join(homeDir, ".fridaforge", "agreed")
	if _, err := os.Stat(markerFile); err == nil {
		return nil
	}

	fmt.Println(strings.Repeat("═", 60))
	fmt.Println("  FridaForge — 伦理使用声明")
	fmt.Println()
	fmt.Println("  本工具仅供授权安全测试和教育目的使用。")
	fmt.Println()
	fmt.Println("  使用 FridaForge 前，你必须获得目标应用和设备")
	fmt.Println("  所有者的明确许可。")
	fmt.Println()
	fmt.Println("  未经授权的使用可能违反适用法律。")
	fmt.Println(strings.Repeat("═", 60))
	fmt.Println()
	fmt.Print("  输入 'AGREE' 表示接受并继续: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("读取输入失败: %w", err)
	}
	input = strings.TrimSpace(input)
	if input != "AGREE" {
		return fmt.Errorf("必须输入 AGREE 才能使用 FridaForge")
	}

	if err := os.MkdirAll(filepath.Dir(markerFile), 0700); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	if err := os.WriteFile(markerFile, []byte{}, 0600); err != nil {
		return fmt.Errorf("写入标记文件失败: %w", err)
	}
	return nil
}
