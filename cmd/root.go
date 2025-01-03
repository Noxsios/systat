package cmd

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	logLevel string
	// Common flags
	outputJSON   bool
	rawOutput    bool
	watchOutput  bool
)

var rootCmd = &cobra.Command{
	Use:   "systat",
	Short: "A system information and DNS query CLI tool",
	Long: `systat is a CLI tool that provides system information and DNS queries.
It can query DNS information for *.admin.uds.dev and *.uds.dev domains,
display system information, and show detailed system metrics.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		lvl, err := log.ParseLevel(logLevel)
		if err != nil {
			return err
		}

		logger := log.FromContext(cmd.Context())
		logger.SetLevel(lvl)
		return nil
	},
}

func ExecuteContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	// Logging flags
	rootCmd.PersistentFlags().StringVarP(&logLevel, "level", "l", "info", "log level (debug, info, warn, error)")
	
	// Output format flags
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false, "output in JSON format instead of YAML")
	rootCmd.PersistentFlags().BoolVar(&rawOutput, "raw", false, "output without syntax highlighting")
	rootCmd.PersistentFlags().BoolVar(&watchOutput, "watch", false, "continuously watch for changes")
}
