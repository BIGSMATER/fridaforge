package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/bigsmater/fridaforge/pkg/device"
)

func init() {
	rootCmd.AddCommand(deviceCmd)
	deviceCmd.AddCommand(deviceListCmd)
}

var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "设备管理",
	Long:  "管理已连接的 Frida 目标设备。",
}

var deviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出已连接的设备",
	Long:  "列出当前所有已连接的 Frida 可用设备。",
	RunE:  runDeviceList,
}

func runDeviceList(cmd *cobra.Command, args []string) error {
	lister := &device.StubDeviceLister{}
	devices, err := lister.ListDevices(context.Background())
	if err != nil {
		return fmt.Errorf("Frida 服务不可达: %w", err)
	}

	if len(devices) == 0 {
		fmt.Println("未发现已连接的设备。")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE")
	for _, d := range devices {
		fmt.Fprintf(w, "%s\t%s\t%s\n", d.ID, d.Name, d.ConnectType)
	}
	w.Flush()
	return nil
}
