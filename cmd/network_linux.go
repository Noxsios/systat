//go:build linux

package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
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
			fmt.Print("\033[H\033[2J") // Clear screen in watch mode
		}
		return nil
	},
}

func showNetworkInfo(logger *log.Logger) error {
	logger.Debug("gathering network information")

	// Get all network interfaces
	links, err := netlink.LinkList()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces: %w", err)
	}

	if rawOutput {
		return showRawNetworkInfo(links)
	}

	// Print interfaces table
	fmt.Println(titleStyle.Render("Network Interfaces"))
	
	interfaceColumns := []table.Column{
		{Title: "Name", Width: 10},
		{Title: "Type", Width: 8},
		{Title: "State", Width: 8},
		{Title: "MAC", Width: 17},
		{Title: "MTU", Width: 5},
		{Title: "Addresses", Width: 40},
	}

	var interfaceRows []table.Row
	for _, link := range links {
		attrs := link.Attrs()
		
		// Get IP addresses
		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		addrStrs := make([]string, 0, len(addrs))
		if err != nil {
			logger.Warn("failed to get addresses",
				"interface", attrs.Name,
				"error", err)
			addrStrs = append(addrStrs, "error")
		} else {
			for _, addr := range addrs {
				addrStrs = append(addrStrs, addr.IPNet.String())
			}
		}

		interfaceRows = append(interfaceRows, table.Row{
			attrs.Name,
			link.Type(),
			attrs.OperState.String(),
			attrs.HardwareAddr.String(),
			fmt.Sprintf("%d", attrs.MTU),
			strings.Join(addrStrs, ", "),
		})
	}

	interfaceTable := table.New(
		table.WithColumns(interfaceColumns),
		table.WithRows(interfaceRows),
		table.WithHeight(len(interfaceRows)),
		table.WithFocused(false),
	)
	
	fmt.Println(tableStyle.Render(interfaceTable.View()))

	// Get and print routing table
	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		logger.Warn("failed to get routing table", "error", err)
		return nil
	}

	fmt.Println(titleStyle.Render("Routing Table"))

	routeColumns := []table.Column{
		{Title: "Destination", Width: 20},
		{Title: "Gateway", Width: 20},
		{Title: "Interface", Width: 10},
		{Title: "Protocol", Width: 10},
		{Title: "Scope", Width: 10},
	}

	var routeRows []table.Row
	for _, route := range routes {
		dst := "default"
		if route.Dst != nil {
			dst = route.Dst.String()
		}

		gw := "none"
		if route.Gw != nil {
			gw = route.Gw.String()
		}

		iface := "unknown"
		if route.LinkIndex > 0 {
			if link, err := netlink.LinkByIndex(route.LinkIndex); err == nil {
				iface = link.Attrs().Name
			}
		}

		routeRows = append(routeRows, table.Row{
			dst,
			gw,
			iface,
			strconv.Itoa(route.Protocol),
			strconv.Itoa(int(route.Scope)),
		})
	}

	routeTable := table.New(
		table.WithColumns(routeColumns),
		table.WithRows(routeRows),
		table.WithHeight(len(routeRows)),
		table.WithFocused(false),
	)

	fmt.Println(tableStyle.Render(routeTable.View()))
	return nil
}

func showRawNetworkInfo(links []netlink.Link) error {
	for _, link := range links {
		attrs := link.Attrs()
		fmt.Printf("Interface: %s\n", attrs.Name)
		fmt.Printf("  Type: %s\n", link.Type())
		fmt.Printf("  State: %s\n", attrs.OperState)
		fmt.Printf("  MAC: %s\n", attrs.HardwareAddr)
		fmt.Printf("  MTU: %d\n", attrs.MTU)
		
		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			fmt.Printf("  Addresses: error: %v\n", err)
		} else {
			fmt.Printf("  Addresses:\n")
			for _, addr := range addrs {
				fmt.Printf("    - %s\n", addr.IPNet)
			}
		}
		fmt.Println()
	}

	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("failed to get routing table: %w", err)
	}

	fmt.Println("Routing Table:")
	for _, route := range routes {
		dst := "default"
		if route.Dst != nil {
			dst = route.Dst.String()
		}

		gw := "none"
		if route.Gw != nil {
			gw = route.Gw.String()
		}

		iface := "unknown"
		if route.LinkIndex > 0 {
			if link, err := netlink.LinkByIndex(route.LinkIndex); err == nil {
				iface = link.Attrs().Name
			}
		}

		fmt.Printf("  Destination: %s\n", dst)
		fmt.Printf("    Gateway: %s\n", gw)
		fmt.Printf("    Interface: %s\n", iface)
		fmt.Printf("    Protocol: %s\n", route.Protocol)
		fmt.Printf("    Scope: %s\n", route.Scope)
		fmt.Println()
	}

	return nil
}

func init() {
	rootCmd.AddCommand(networkCmd)
}
