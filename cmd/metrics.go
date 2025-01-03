package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/log"
	"github.com/dustin/go-humanize"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Display detailed system metrics",
	Long: `Display detailed system metrics using github.com/shirou/gopsutil.
Provides information about:
  - CPU usage and load averages
  - Memory usage (RAM and swap)
  - Host information and uptime`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.FromContext(cmd.Context())

		for {
			if err := showMetrics(logger); err != nil {
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

func showMetrics(logger *log.Logger) error {
	logger.Debug("gathering system metrics")

	if rawOutput {
		return showRawMetrics()
	}

	// CPU Usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return fmt.Errorf("failed to get CPU usage: %w", err)
	}

	fmt.Println(titleStyle.Render("CPU Usage"))
	columns := []table.Column{
		{Title: "CPU", Width: 10},
		{Title: "Usage", Width: 10},
	}

	rows := []table.Row{
		{"Total", fmt.Sprintf("%.1f%%", cpuPercent[0])},
	}

	t := NewTable(columns, rows)
	fmt.Println(tableStyle.Render(t.View()))

	// Load Average
	loadAvg, err := load.Avg()
	if err == nil {
		fmt.Println(titleStyle.Render("Load Average"))
		columns := []table.Column{
			{Title: "Period", Width: 10},
			{Title: "Load", Width: 10},
		}

		rows := []table.Row{
			{"1 min", fmt.Sprintf("%.2f", loadAvg.Load1)},
			{"5 min", fmt.Sprintf("%.2f", loadAvg.Load5)},
			{"15 min", fmt.Sprintf("%.2f", loadAvg.Load15)},
		}

		t = NewTable(columns, rows)
		fmt.Println(tableStyle.Render(t.View()))
	}

	// Memory Usage
	vmem, err := mem.VirtualMemory()
	if err == nil {
		fmt.Println(titleStyle.Render("Memory Usage"))
		columns := []table.Column{
			{Title: "Type", Width: 10},
			{Title: "Value", Width: 15},
		}

		rows := []table.Row{
			{"Total", humanize.Bytes(vmem.Total)},
			{"Used", humanize.Bytes(vmem.Used)},
			{"Free", humanize.Bytes(vmem.Free)},
			{"Used%", fmt.Sprintf("%.1f%%", vmem.UsedPercent)},
			{"Cached", humanize.Bytes(vmem.Cached)},
		}

		t = NewTable(columns, rows)
		fmt.Println(tableStyle.Render(t.View()))
	}

	// Swap Usage
	swap, err := mem.SwapMemory()
	if err == nil {
		fmt.Println(titleStyle.Render("Swap Usage"))
		columns := []table.Column{
			{Title: "Type", Width: 10},
			{Title: "Value", Width: 15},
		}

		rows := []table.Row{
			{"Total", humanize.Bytes(swap.Total)},
			{"Used", humanize.Bytes(swap.Used)},
			{"Free", humanize.Bytes(swap.Free)},
			{"Used%", fmt.Sprintf("%.1f%%", swap.UsedPercent)},
		}

		t = NewTable(columns, rows)
		fmt.Println(tableStyle.Render(t.View()))
	}

	return nil
}

func showRawMetrics() error {
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return fmt.Errorf("failed to get CPU usage: %w", err)
	}
	fmt.Printf("CPU Usage: %.1f%%\n\n", cpuPercent[0])

	loadAvg, err := load.Avg()
	if err != nil {
		fmt.Printf("Load Average: error: %v\n", err)
	} else {
		fmt.Println("Load Average:")
		fmt.Printf("  1 min:  %.2f\n", loadAvg.Load1)
		fmt.Printf("  5 min:  %.2f\n", loadAvg.Load5)
		fmt.Printf("  15 min: %.2f\n", loadAvg.Load15)
		fmt.Println()
	}

	vmem, err := mem.VirtualMemory()
	if err != nil {
		fmt.Printf("Memory Usage: error: %v\n", err)
	} else {
		fmt.Println("Memory Usage:")
		fmt.Printf("  Total:  %s\n", humanize.Bytes(vmem.Total))
		fmt.Printf("  Used:   %s\n", humanize.Bytes(vmem.Used))
		fmt.Printf("  Free:   %s\n", humanize.Bytes(vmem.Free))
		fmt.Printf("  Used%%:  %.1f%%\n", vmem.UsedPercent)
		fmt.Printf("  Cached: %s\n", humanize.Bytes(vmem.Cached))
		fmt.Println()
	}

	swap, err := mem.SwapMemory()
	if err != nil {
		fmt.Printf("Swap Usage: error: %v\n", err)
	} else {
		fmt.Println("Swap Usage:")
		fmt.Printf("  Total: %s\n", humanize.Bytes(swap.Total))
		fmt.Printf("  Used:  %s\n", humanize.Bytes(swap.Used))
		fmt.Printf("  Free:  %s\n", humanize.Bytes(swap.Free))
		fmt.Printf("  Used%%: %.1f%%\n", swap.UsedPercent)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(metricsCmd)
}
