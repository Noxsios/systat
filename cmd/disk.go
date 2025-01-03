package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/log"
	"github.com/dustin/go-humanize"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/spf13/cobra"
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
			fmt.Print("\033[H\033[2J") // Clear screen in watch mode
		}
		return nil
	},
}

func showDiskInfo(logger *log.Logger) error {
	logger.Debug("gathering disk information")

	if rawOutput {
		return showRawDiskInfo()
	}

	partitions, err := disk.Partitions(false)
	if err != nil {
		return fmt.Errorf("failed to get disk partitions: %w", err)
	}

	fmt.Println(titleStyle.Render("Disk Partitions"))
	columns := []table.Column{
		{Title: "Device", Width: 15},
		{Title: "Mount", Width: 15},
		{Title: "FS Type", Width: 10},
		{Title: "Total", Width: 10},
		{Title: "Used", Width: 10},
		{Title: "Free", Width: 10},
		{Title: "Use%", Width: 8},
	}

	var rows []table.Row
	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}

		rows = append(rows, table.Row{
			partition.Device,
			partition.Mountpoint,
			partition.Fstype,
			humanize.Bytes(usage.Total),
			humanize.Bytes(usage.Used),
			humanize.Bytes(usage.Free),
			fmt.Sprintf("%.1f%%", usage.UsedPercent),
		})
	}

	t := NewTable(columns, rows)
	fmt.Println(tableStyle.Render(t.View()))

	iostats, err := disk.IOCounters()
	if err != nil {
		return fmt.Errorf("failed to get disk IO statistics: %w", err)
	}

	fmt.Println(titleStyle.Render("Disk IO Statistics"))
	columns = []table.Column{
		{Title: "Device", Width: 15},
		{Title: "Read Bytes", Width: 15},
		{Title: "Write Bytes", Width: 15},
		{Title: "Read Count", Width: 12},
		{Title: "Write Count", Width: 12},
		{Title: "Read Time", Width: 12},
		{Title: "Write Time", Width: 12},
	}

	rows = nil
	for name, stat := range iostats {
		rows = append(rows, table.Row{
			name,
			humanize.Bytes(stat.ReadBytes),
			humanize.Bytes(stat.WriteBytes),
			fmt.Sprintf("%d", stat.ReadCount),
			fmt.Sprintf("%d", stat.WriteCount),
			fmt.Sprintf("%dms", stat.ReadTime),
			fmt.Sprintf("%dms", stat.WriteTime),
		})
	}

	t = NewTable(columns, rows)
	fmt.Println(tableStyle.Render(t.View()))

	return nil
}

func showRawDiskInfo() error {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return fmt.Errorf("failed to get disk partitions: %w", err)
	}

	fmt.Println("Disk Partitions:")
	for _, partition := range partitions {
		fmt.Printf("  Device: %s\n", partition.Device)
		fmt.Printf("    Mount Point: %s\n", partition.Mountpoint)
		fmt.Printf("    FS Type: %s\n", partition.Fstype)

		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			fmt.Printf("    Usage: error: %v\n", err)
			continue
		}

		fmt.Printf("    Total: %s\n", humanize.Bytes(usage.Total))
		fmt.Printf("    Used: %s\n", humanize.Bytes(usage.Used))
		fmt.Printf("    Free: %s\n", humanize.Bytes(usage.Free))
		fmt.Printf("    Use%%: %.1f%%\n", usage.UsedPercent)
		fmt.Println()
	}

	iostats, err := disk.IOCounters()
	if err != nil {
		return fmt.Errorf("failed to get disk IO statistics: %w", err)
	}

	fmt.Println("Disk IO Statistics:")
	for name, stat := range iostats {
		fmt.Printf("  Device: %s\n", name)
		fmt.Printf("    Read Bytes: %s\n", humanize.Bytes(stat.ReadBytes))
		fmt.Printf("    Write Bytes: %s\n", humanize.Bytes(stat.WriteBytes))
		fmt.Printf("    Read Count: %d\n", stat.ReadCount)
		fmt.Printf("    Write Count: %d\n", stat.WriteCount)
		fmt.Printf("    Read Time: %dms\n", stat.ReadTime)
		fmt.Printf("    Write Time: %dms\n", stat.WriteTime)
		fmt.Println()
	}

	return nil
}

func init() {
	rootCmd.AddCommand(diskCmd)
}
