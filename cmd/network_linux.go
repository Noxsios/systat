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
		attrs := link.Attrs()
		iface := map[string]interface{}{
			"name":          attrs.Name,
			"hardware_addr": attrs.HardwareAddr.String(),
			"flags":         attrs.Flags.String(),
			"mtu":          attrs.MTU,
			"type":         link.Type(),
			"state":        attrs.OperState.String(),
			"index":        attrs.Index,
		}

		// Get IP addresses for this interface
		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			logger.Warn("failed to get addresses",
				"interface", attrs.Name,
				"error", err)
			iface["addresses"] = []string{"error: " + err.Error()}
		} else {
			addrList := make([]string, len(addrs))
			for i, addr := range addrs {
				addrList[i] = addr.IPNet.String()
			}
			iface["addresses"] = addrList
		}

		// Get interface statistics
		stats := attrs.Statistics
		if stats != nil {
			iface["statistics"] = map[string]uint64{
				"rx_packets": stats.RxPackets,
				"tx_packets": stats.TxPackets,
				"rx_bytes":   stats.RxBytes,
				"tx_bytes":   stats.TxBytes,
				"rx_errors":  stats.RxErrors,
				"tx_errors":  stats.TxErrors,
				"rx_dropped": stats.RxDropped,
				"tx_dropped": stats.TxDropped,
			}
		}

		interfaces = append(interfaces, iface)
	}
	info["interfaces"] = interfaces

	// Get routing table
	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		logger.Warn("failed to get routing table", "error", err)
	} else {
		routeList := make([]map[string]interface{}, len(routes))
		for i, route := range routes {
			r := map[string]interface{}{
				"dst":      route.Dst,
				"src":      route.Src,
				"gateway":  route.Gw,
				"protocol": route.Protocol,
				"scope":    route.Scope,
				"table":    route.Table,
			}
			if route.LinkIndex > 0 {
				if link, err := netlink.LinkByIndex(route.LinkIndex); err == nil {
					r["interface"] = link.Attrs().Name
				}
			}
			routeList[i] = r
		}
		info["routes"] = routeList
	}

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
