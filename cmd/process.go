package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Display process information and resource usage",
	Long: `Display detailed process information using github.com/shirou/gopsutil.
Provides information about:
  - Running processes and their states
  - CPU and memory usage per process
  - Process hierarchy and relationships
  - Open files and network connections`,
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
		}
		return nil
	},
}

func showProcessInfo(logger *log.Logger) error {
	logger.Debug("gathering process information")

	processes, err := process.Processes()
	if err != nil {
		return fmt.Errorf("failed to get process list: %w", err)
	}

	var processInfos []map[string]interface{}
	for _, p := range processes {
		info := make(map[string]interface{})

		if name, err := p.Name(); err == nil {
			info["name"] = name
		}
		if cmdline, err := p.Cmdline(); err == nil {
			info["cmdline"] = cmdline
		}
		if status, err := p.Status(); err == nil {
			info["status"] = status
		}
		if createTime, err := p.CreateTime(); err == nil {
			info["create_time"] = time.Unix(createTime/1000, 0)
		}
		if cpuPercent, err := p.CPUPercent(); err == nil {
			info["cpu_percent"] = cpuPercent
		}
		if memInfo, err := p.MemoryInfo(); err == nil {
			info["memory"] = memInfo
		}
		if username, err := p.Username(); err == nil {
			info["username"] = username
		}

		processInfos = append(processInfos, info)
	}

	var b []byte

	if outputJSON {
		b, err = json.MarshalIndent(processInfos, "", "  ")
	} else {
		b, err = yaml.Marshal(processInfos)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal process info: %w", err)
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
	rootCmd.AddCommand(processCmd)
}
