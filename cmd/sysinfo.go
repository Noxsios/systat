package cmd

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/log"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/zcalusic/sysinfo"
)

var sysinfoCmd = &cobra.Command{
	Use:   "sysinfo",
	Short: "Display system information",
	Long: `Display detailed system information using github.com/zcalusic/sysinfo.
Provides information about:
  - OS version and architecture
  - CPU model and features
  - Memory size and configuration
  - Network interfaces and drivers`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.FromContext(cmd.Context())
		return showSysInfo(logger)
	},
}

func showSysInfo(logger *log.Logger) error {
	logger.Debug("gathering system information")

	var si sysinfo.SysInfo
	si.GetSysInfo()

	if rawOutput {
		return showRawSysInfo(&si)
	}

	// OS Information
	fmt.Println(titleStyle.Render("Operating System"))
	columns := []table.Column{
		{Title: "Property", Width: 20},
		{Title: "Value", Width: 50},
	}

	rows := []table.Row{
		{"OS", si.OS.Name + " " + si.OS.Version},
		{"Architecture", si.OS.Architecture},
		{"Kernel", si.Kernel.Release},
		{"Hostname", si.Node.Hostname},
	}

	t := NewTable(columns, rows)
	fmt.Println(tableStyle.Render(t.View()))

	// CPU Information
	fmt.Println(titleStyle.Render("CPU Information"))
	rows = []table.Row{
		{"Vendor", si.CPU.Vendor},
		{"Model", si.CPU.Model},
		{"Cores", fmt.Sprintf("%d", si.CPU.Cores)},
		{"Threads", fmt.Sprintf("%d", si.CPU.Threads)},
		{"Cache", humanize.Bytes(uint64(si.CPU.Cache))},
	}

	t = NewTable(columns, rows)
	fmt.Println(tableStyle.Render(t.View()))

	// Memory Information
	fmt.Println(titleStyle.Render("Memory Information"))
	rows = []table.Row{
		{"Total", humanize.Bytes(uint64(si.Memory.Size))},
	}

	t = NewTable(columns, rows)
	fmt.Println(tableStyle.Render(t.View()))

	return nil
}

func showRawSysInfo(si *sysinfo.SysInfo) error {
	fmt.Println("Operating System:")
	fmt.Printf("  OS: %s %s\n", si.OS.Name, si.OS.Version)
	fmt.Printf("  Architecture: %s\n", si.OS.Architecture)
	fmt.Printf("  Kernel: %s\n", si.Kernel.Release)
	fmt.Printf("  Hostname: %s\n", si.Node.Hostname)
	fmt.Println()

	fmt.Println("CPU Information:")
	fmt.Printf("  Vendor: %s\n", si.CPU.Vendor)
	fmt.Printf("  Model: %s\n", si.CPU.Model)
	fmt.Printf("  Cores: %d\n", si.CPU.Cores)
	fmt.Printf("  Threads: %d\n", si.CPU.Threads)
	fmt.Printf("  Cache: %s\n", humanize.Bytes(uint64(si.CPU.Cache)))
	fmt.Println()

	fmt.Println("Memory Information:")
	fmt.Printf("  Total: %s\n", humanize.Bytes(uint64(si.Memory.Size)))

	return nil
}

func init() {
	rootCmd.AddCommand(sysinfoCmd)
}
