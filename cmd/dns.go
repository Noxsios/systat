package cmd

import (
	"fmt"
	"os"

	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/miekg/dns"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	dnsServer    = "10.0.0.1:53"
	adminUDSDev  = ".admin.uds.dev"
	udsDevDomain = ".uds.dev"
)

var dnsCmd = &cobra.Command{
	Use:   "dns [domain]",
	Short: "Query DNS information for a domain",
	Long: `Query DNS information for a domain under *.admin.uds.dev or *.uds.dev.
Example: systat dns keycloak.admin.uds.dev`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.FromContext(cmd.Context())
		domain := args[0]

		logger.Debug("querying DNS", "domain", domain)

		msg := new(dns.Msg)
		msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)

		client := new(dns.Client)
		resp, _, err := client.Exchange(msg, dnsServer)
		if err != nil {
			return fmt.Errorf("DNS query failed: %w", err)
		}

		b, err := yaml.Marshal(resp)
		if err != nil {
			return fmt.Errorf("failed to marshal response: %w", err)
		}

		style := "catppuccin-latte"
		if lipgloss.HasDarkBackground() {
			style = "catppuccin-frappe"
		}

		return quick.Highlight(os.Stdout, string(b), "yaml", "terminal256", style)
	},
}

func init() {
	rootCmd.AddCommand(dnsCmd)
}
