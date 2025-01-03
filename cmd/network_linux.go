//go:build linux

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
	"gopkg.in/yaml.v3"
)

var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Display network interfaces and routing information",
	Long: `Display detailed network information using github.com/vishvananda/netlink.
Provides information about:
  - Network interfaces and their states
  - IP addresses and CIDR ranges
  - Routing table entries
  - Network namespaces`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.FromContext(cmd.Context())

		for {
			if err := showNetworkInfo(logger); err != nil {
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

func showNetworkInfo(logger *log.Logger) error {
	logger.Debug("gathering network information")

	info := make(map[string]interface{})

	// Get all network interfaces
	links, err := netlink.LinkList()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var interfaces []map[string]interface{}
	for _, link := range links {
		iface := make(map[string]interface{})
		iface["name"] = link.Attrs().Name
		iface["hardware_addr"] = link.Attrs().HardwareAddr.String()
		iface["flags"] = link.Attrs().Flags.String()
		iface["mtu"] = link.Attrs().MTU

		// Get IP addresses for this interface
		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			logger.Debug("failed to get addresses for interface",
				"interface", link.Attrs().Name,
				"error", err)
			continue
		}
		iface["addresses"] = addrs

		interfaces = append(interfaces, iface)
	}
	info["interfaces"] = interfaces

	// Get routing table
	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("failed to get routing table: %w", err)
	}
	info["routes"] = routes

	var b []byte

	if outputJSON {
		b, err = json.MarshalIndent(info, "", "  ")
	} else {
		b, err = yaml.Marshal(info)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal network info: %w", err)
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
	rootCmd.AddCommand(networkCmd)
}
