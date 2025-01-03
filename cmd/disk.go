package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var diskCmd = &cobra.Command{
	Use:   "disk",
	Short: "Display disk usage and IO statistics",
	Long: `Display detailed disk information using github.com/shirou/gopsutil.
Provides information about:
  - Partitions and mount points
  - Disk usage statistics
  - IO counters and statistics`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.FromContext(cmd.Context())

		for {
			if err := showDiskInfo(logger); err != nil {
				return err
			}

			if !watchOutput {
				break
			}
			time.Sleep(2 * time.Second)
		}
		return nil
	},
}

func showDiskInfo(logger *log.Logger) error {
	logger.Debug("gathering disk information")

	info := make(map[string]interface{})

	partitions, err := disk.Partitions(true)
	if err != nil {
		return fmt.Errorf("failed to get disk partitions: %w", err)
	}
	info["partitions"] = partitions

	// Get usage for each partition
	var usageStats []interface{}
	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			logger.Debug("failed to get usage for partition",
				"mountpoint", partition.Mountpoint,
				"error", err)
			continue
		}
		usageStats = append(usageStats, usage)
	}
	info["usage"] = usageStats

	// Get IO counters
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return fmt.Errorf("failed to get IO counters: %w", err)
	}
	info["io_counters"] = ioCounters

	var b []byte

	if outputJSON {
		b, err = json.MarshalIndent(info, "", "  ")
	} else {
		b, err = yaml.Marshal(info)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal disk info: %w", err)
	}

	if rawOutput {
		fmt.Println(string(b))
		return nil
	}

	style := "catppuccin-latte"
	if lipgloss.HasDarkBackground() {
		style = "catppuccin-frappe"
	}

	format := "yaml"
	if outputJSON {
		format = "json"
	}

	return quick.Highlight(os.Stdout, string(b), format, "terminal256", style)
}

func init() {
	rootCmd.AddCommand(diskCmd)
}
