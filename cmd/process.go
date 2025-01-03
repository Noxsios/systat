package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/log"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cobra"
)

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Display process information",
	Long: `Display detailed process information using github.com/shirou/gopsutil.
Provides information about:
  - Process ID and parent ID
  - Process name and command line
  - CPU and memory usage
  - Creation time and running time`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.FromContext(cmd.Context())

		for {
			if err := showProcessInfo(logger); err != nil {
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

func showProcessInfo(logger *log.Logger) error {
	logger.Debug("gathering process information")

	if rawOutput {
		return showRawProcessInfo()
	}

	processes, err := process.Processes()
	if err != nil {
		return fmt.Errorf("failed to get process list: %w", err)
	}

	// Sort processes by CPU usage
	sort.Slice(processes, func(i, j int) bool {
		cpu1, _ := processes[i].CPUPercent()
		cpu2, _ := processes[j].CPUPercent()
		return cpu1 > cpu2
	})

	fmt.Println(titleStyle.Render("Top Processes by CPU Usage"))

	columns := []table.Column{
		{Title: "PID", Width: 8},
		{Title: "Name", Width: 20},
		{Title: "CPU%", Width: 8},
		{Title: "Memory%", Width: 8},
		{Title: "Status", Width: 10},
		{Title: "User", Width: 12},
		{Title: "Command", Width: 40},
	}

	var rows []table.Row
	for _, p := range processes[:20] { // Show top 20 processes
		pid := p.Pid

		name, err := p.Name()
		if err != nil {
			name = "unknown"
		}

		cpuPercent, err := p.CPUPercent()
		if err != nil {
			cpuPercent = 0
		}

		memPercent, err := p.MemoryPercent()
		if err != nil {
			memPercent = 0
		}

		status, err := p.Status()
		if err != nil {
			status = []string{"unknown"}
		}

		username, err := p.Username()
		if err != nil {
			username = "unknown"
		}

		cmdline, err := p.Cmdline()
		if err != nil {
			cmdline = "unknown"
		}
		if len(cmdline) > 40 {
			cmdline = cmdline[:37] + "..."
		}

		rows = append(rows, table.Row{
			fmt.Sprintf("%d", pid),
			name,
			fmt.Sprintf("%.1f", cpuPercent),
			fmt.Sprintf("%.1f", memPercent),
			status[0],
			username,
			cmdline,
		})
	}

	t := NewTable(columns, rows)
	fmt.Println(tableStyle.Render(t.View()))

	return nil
}

func showRawProcessInfo() error {
	processes, err := process.Processes()
	if err != nil {
		return fmt.Errorf("failed to get process list: %w", err)
	}

	// Sort processes by CPU usage
	sort.Slice(processes, func(i, j int) bool {
		cpu1, _ := processes[i].CPUPercent()
		cpu2, _ := processes[j].CPUPercent()
		return cpu1 > cpu2
	})

	fmt.Println("Top Processes by CPU Usage:")
	for _, p := range processes[:20] { // Show top 20 processes
		pid := p.Pid

		name, err := p.Name()
		if err != nil {
			name = "unknown"
		}

		cpuPercent, err := p.CPUPercent()
		if err != nil {
			cpuPercent = 0
		}

		memPercent, err := p.MemoryPercent()
		if err != nil {
			memPercent = 0
		}

		status, err := p.Status()
		if err != nil {
			status = []string{"unknown"}
		}

		username, err := p.Username()
		if err != nil {
			username = "unknown"
		}

		cmdline, err := p.Cmdline()
		if err != nil {
			cmdline = "unknown"
		}

		fmt.Printf("PID: %d\n", pid)
		fmt.Printf("  Name: %s\n", name)
		fmt.Printf("  CPU%%: %.1f\n", cpuPercent)
		fmt.Printf("  Memory%%: %.1f\n", memPercent)
		fmt.Printf("  Status: %s\n", status[0])
		fmt.Printf("  User: %s\n", username)
		fmt.Printf("  Command: %s\n", cmdline)
		fmt.Println()
	}

	return nil
}

func init() {
	rootCmd.AddCommand(processCmd)
}
