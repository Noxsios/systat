package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/zcalusic/sysinfo"
	"gopkg.in/yaml.v3"
)

var sysinfoCmd = &cobra.Command{
	Use:   "sysinfo",
	Short: "Display system information using sysinfo",
	Long: `Display detailed system information using github.com/zcalusic/sysinfo.
This includes OS, kernel, product, board, chassis, BIOS, CPU, memory, and system information.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.FromContext(cmd.Context())
		logger.Debug("gathering system information")

		var si sysinfo.SysInfo
		si.GetSysInfo()

		var b []byte
		var err error
		if outputJSON {
			b, err = json.MarshalIndent(si, "", "  ")
		} else {
			b, err = yaml.Marshal(si)
		}
		if err != nil {
			return fmt.Errorf("failed to marshal system info: %w", err)
		}

		style := "catppuccin-latte"
		if lipgloss.HasDarkBackground() {
			style = "catppuccin-frappe"
		}

		return quick.Highlight(os.Stdout, string(b), "yaml", "terminal256", style)
	},
}

func init() {
	rootCmd.AddCommand(sysinfoCmd)
}
