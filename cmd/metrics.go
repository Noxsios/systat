package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Display detailed system metrics using gopsutil",
	Long: `Display detailed system metrics using github.com/shirou/gopsutil.
Provides real-time information about:
  - Host: hostname, uptime, platform info
  - CPU: model, cores, frequency, usage
  - Memory: total, used, available, swap`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.FromContext(cmd.Context())
		logger.Debug("gathering system metrics")

		metrics := make(map[string]interface{})

		hostInfo, err := host.Info()
		if err != nil {
			return fmt.Errorf("failed to get host info: %w", err)
		}
		metrics["host"] = hostInfo

		cpuInfo, err := cpu.Info()
		if err != nil {
			return fmt.Errorf("failed to get CPU info: %w", err)
		}
		metrics["cpu"] = cpuInfo

		memInfo, err := mem.VirtualMemory()
		if err != nil {
			return fmt.Errorf("failed to get memory info: %w", err)
		}
		metrics["memory"] = memInfo

		var b []byte
		if outputJSON {
			b, err = json.MarshalIndent(metrics, "", "  ")
		} else {
			b, err = yaml.Marshal(metrics)
		}
		if err != nil {
			return fmt.Errorf("failed to marshal metrics: %w", err)
		}

		style := "catppuccin-latte"
		if lipgloss.HasDarkBackground() {
			style = "catppuccin-frappe"
		}

		return quick.Highlight(os.Stdout, string(b), "yaml", "terminal256", style)
	},
}

func init() {
	rootCmd.AddCommand(metricsCmd)
}
