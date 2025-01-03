package cmd

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type viewMode int

const (
	dashboardView viewMode = iota
	networkDetailView
)

type focusedTable int

const (
	cpuTableFocus focusedTable = iota
	diskTableFocus
	netTableFocus
)

type statusCheck struct {
	name   string
	status bool
}

type model struct {
	cpuPercents    []float64
	loadAvg        *load.AvgStat
	memory         *mem.VirtualMemoryStat
	swap           *mem.SwapMemoryStat
	diskStats      map[string]disk.IOCountersStat
	diskPartitions []disk.PartitionStat
	diskUsage      map[string]*disk.UsageStat
	netStats       map[string]psnet.IOCountersStat
	statusChecks   []statusCheck
	k8sClient      *kubernetes.Clientset
	namespaces     []corev1.Namespace
	width          int
	height         int
	lastUpdate     time.Time
	diskTable      table.Model
	cpuTable       table.Model
	memTable       table.Model
	netTable       table.Model
	statusTable    table.Model
	k8sTable       table.Model
	focusedTable   focusedTable
	currentView    viewMode
	selectedIface  string
}

type tickMsg time.Time

type dnsCheckMsg struct {
	host   string
	status bool
}

type pingCheckMsg struct {
	host   string
	status bool
}

type statsUpdateMsg struct {
	cpuPercents    []float64
	loadAvg        *load.AvgStat
	memory         *mem.VirtualMemoryStat
	swap           *mem.SwapMemoryStat
	diskStats      map[string]disk.IOCountersStat
	diskPartitions []disk.PartitionStat
	diskUsage      map[string]*disk.UsageStat
	netStats       map[string]psnet.IOCountersStat
	namespaces     []corev1.Namespace
}

func initialModel() model {
	tableStyle := table.DefaultStyles()
	tableStyle.Header = tableStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)

	tableStyle.Selected = tableStyle.Selected.
		Foreground(lipgloss.Color("#a6d189")).
		Bold(true)

	m := model{
		diskUsage:      make(map[string]*disk.UsageStat),
		netStats:       make(map[string]psnet.IOCountersStat),
		diskStats:      make(map[string]disk.IOCountersStat),
		lastUpdate:     time.Now(),
		cpuPercents:    make([]float64, 0),
		diskPartitions: make([]disk.PartitionStat, 0),
		statusChecks: []statusCheck{
			{name: "runtime.uds.dev", status: false},
			{name: "keycloak.admin.uds.dev", status: false},
			{name: "ping 10.0.0.1", status: false},
		},
		focusedTable:   cpuTableFocus,
		currentView:    dashboardView,
	}

	// Initialize k8s client
	home := homedir.HomeDir()
	if home != "" {
		kubeconfig := filepath.Join(home, ".kube", "config")
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err == nil {
			m.k8sClient, _ = kubernetes.NewForConfig(config)
		}
	}

	m.diskTable = table.New(
		table.WithColumns([]table.Column{
			{Title: "Disk(d)", Width: 20},
			{Title: "Mount(m)", Width: 20},
			{Title: "Used(u)", Width: 15},
			{Title: "Total", Width: 15},
			{Title: "Used%", Width: 10},
		}),
		table.WithStyles(tableStyle),
		table.WithHeight(6),
	)

	m.cpuTable = table.New(
		table.WithColumns([]table.Column{
			{Title: "Core(c)", Width: 10},
			{Title: "Usage(u)", Width: 10},
		}),
		table.WithStyles(tableStyle),
		table.WithHeight(6),
		table.WithFocused(true),
	)

	m.memTable = table.New(
		table.WithColumns([]table.Column{
			{Title: "Type(t)", Width: 10},
			{Title: "Used(u)", Width: 15},
			{Title: "Total(t)", Width: 15},
			{Title: "Used%(p)", Width: 10},
		}),
		table.WithStyles(tableStyle),
	)

	m.netTable = table.New(
		table.WithColumns([]table.Column{
			{Title: "Iface(i)", Width: 15},
			{Title: "IPv4(4)", Width: 20},
			{Title: "RX(r)", Width: 20},
			{Title: "TX(t)", Width: 20},
		}),
		table.WithStyles(tableStyle),
		table.WithHeight(6),
	)

	m.statusTable = table.New(
		table.WithColumns([]table.Column{
			{Title: "Service", Width: 30},
			{Title: "Status", Width: 10},
		}),
		table.WithStyles(tableStyle),
		table.WithHeight(4),
	)

	m.k8sTable = table.New(
		table.WithColumns([]table.Column{
			{Title: "Namespace", Width: 30},
			{Title: "Status", Width: 10},
			{Title: "Age", Width: 15},
		}),
		table.WithStyles(tableStyle),
		table.WithHeight(6),
	)

	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		checkDNSCmd("runtime.uds.dev"),
		checkDNSCmd("keycloak.admin.uds.dev"),
		checkPingCmd("10.0.0.1"),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func checkDNSCmd(host string) tea.Cmd {
	return func() tea.Msg {
		_, err := net.LookupHost(host)
		return dnsCheckMsg{host: host, status: err == nil}
	}
}

func checkPingCmd(host string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("ping", "-c", "1", "-W", "1", host)
		return pingCheckMsg{host: host, status: cmd.Run() == nil}
	}
}

func (m *model) updateStats() tea.Cmd {
	return func() tea.Msg {
		var wg sync.WaitGroup
		var mu sync.Mutex
		msg := statsUpdateMsg{
			diskUsage: make(map[string]*disk.UsageStat),
			diskStats: make(map[string]disk.IOCountersStat),
			netStats:  make(map[string]psnet.IOCountersStat),
		}

		// CPU stats
		wg.Add(1)
		go func() {
			defer wg.Done()
			if percents, err := cpu.Percent(0, true); err == nil {
				mu.Lock()
				msg.cpuPercents = percents
				mu.Unlock()
			}
		}()

		// Load average
		wg.Add(1)
		go func() {
			defer wg.Done()
			if loadAvg, err := load.Avg(); err == nil {
				mu.Lock()
				msg.loadAvg = loadAvg
				mu.Unlock()
			}
		}()

		// Memory stats
		wg.Add(1)
		go func() {
			defer wg.Done()
			if vmem, err := mem.VirtualMemory(); err == nil {
				mu.Lock()
				msg.memory = vmem
				mu.Unlock()
			}
		}()

		// Swap stats
		wg.Add(1)
		go func() {
			defer wg.Done()
			if swap, err := mem.SwapMemory(); err == nil {
				mu.Lock()
				msg.swap = swap
				mu.Unlock()
			}
		}()

		// Disk IO stats
		wg.Add(1)
		go func() {
			defer wg.Done()
			if iostats, err := disk.IOCounters(); err == nil {
				mu.Lock()
				msg.diskStats = iostats
				mu.Unlock()
			}
		}()

		// Disk partitions and usage
		wg.Add(1)
		go func() {
			defer wg.Done()
			if partitions, err := disk.Partitions(false); err == nil {
				mu.Lock()
				msg.diskPartitions = partitions
				mu.Unlock()

				var usageWg sync.WaitGroup
				for _, partition := range partitions {
					usageWg.Add(1)
					go func(p disk.PartitionStat) {
						defer usageWg.Done()
						if usage, err := disk.Usage(p.Mountpoint); err == nil {
							mu.Lock()
							msg.diskUsage[p.Mountpoint] = usage
							mu.Unlock()
						}
					}(partition)
				}
				usageWg.Wait()
			}
		}()

		// Network stats
		wg.Add(1)
		go func() {
			defer wg.Done()
			if iostats, err := psnet.IOCounters(false); err == nil {
				netStats := make(map[string]psnet.IOCountersStat)
				for _, stat := range iostats {
					netStats[stat.Name] = stat
				}
				mu.Lock()
				msg.netStats = netStats
				mu.Unlock()
			}
		}()

		// K8s stats
		if m.k8sClient != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if namespaces, err := m.k8sClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{}); err == nil {
					mu.Lock()
					msg.namespaces = namespaces.Items
					mu.Unlock()
				}
			}()
		}

		wg.Wait()
		return msg
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.currentView == networkDetailView {
				m.currentView = dashboardView
				return m, nil
			}
		case "enter":
			if m.focusedTable == netTableFocus && m.currentView == dashboardView {
				selectedRow := m.netTable.SelectedRow()
				if len(selectedRow) > 0 {
					m.selectedIface = selectedRow[0]
					m.currentView = networkDetailView
					return m, nil
				}
			}
		case "tab":
			if m.currentView == dashboardView {
				m.focusedTable = (m.focusedTable + 1) % 3

				switch m.focusedTable {
				case cpuTableFocus:
					m.cpuTable.Focus()
					m.diskTable.Blur()
					m.netTable.Blur()
				case diskTableFocus:
					m.diskTable.Focus()
					m.cpuTable.Blur()
					m.netTable.Blur()
				case netTableFocus:
					m.netTable.Focus()
					m.cpuTable.Blur()
					m.diskTable.Blur()
				}
			}
			return m, nil
		case "up", "down", "pageup", "pagedown", "home", "end":
			if m.currentView == dashboardView {
				var cmd tea.Cmd
				switch m.focusedTable {
				case cpuTableFocus:
					m.cpuTable, cmd = m.cpuTable.Update(msg)
				case diskTableFocus:
					m.diskTable, cmd = m.diskTable.Update(msg)
				case netTableFocus:
					m.netTable, cmd = m.netTable.Update(msg)
				}
				return m, cmd
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.lastUpdate = time.Time(msg)
		return m, tea.Batch(
			m.updateStats(),
			tickCmd(),
			checkDNSCmd("runtime.uds.dev"),
			checkDNSCmd("keycloak.admin.uds.dev"),
			checkPingCmd("10.0.0.1"),
		)

	case dnsCheckMsg:
		for i := range m.statusChecks {
			if m.statusChecks[i].name == msg.host {
				m.statusChecks[i].status = msg.status
				break
			}
		}
		m.updateTables()

	case pingCheckMsg:
		for i := range m.statusChecks {
			if m.statusChecks[i].name == "ping "+msg.host {
				m.statusChecks[i].status = msg.status
				break
			}
		}
		m.updateTables()

	case statsUpdateMsg:
		if len(msg.cpuPercents) > 0 {
			m.cpuPercents = msg.cpuPercents
		}
		if msg.loadAvg != nil {
			m.loadAvg = msg.loadAvg
		}
		if msg.memory != nil {
			m.memory = msg.memory
		}
		if msg.swap != nil {
			m.swap = msg.swap
		}
		if len(msg.diskStats) > 0 {
			m.diskStats = msg.diskStats
		}
		if len(msg.diskPartitions) > 0 {
			m.diskPartitions = msg.diskPartitions
		}
		if len(msg.diskUsage) > 0 {
			m.diskUsage = msg.diskUsage
		}
		if len(msg.netStats) > 0 {
			m.netStats = msg.netStats
		}
		if len(msg.namespaces) > 0 {
			m.namespaces = msg.namespaces
		}
		m.updateTables()
		return m, nil
	}

	return m, nil
}

func (m *model) updateTables() {
	var cpuRows []table.Row
	for i, percent := range m.cpuPercents {
		cpuRows = append(cpuRows, table.Row{
			fmt.Sprintf("%d", i),
			fmt.Sprintf("%.1f%%", percent),
		})
	}
	m.cpuTable.SetRows(cpuRows)

	var memRows []table.Row
	if m.memory != nil {
		memRows = append(memRows, table.Row{
			"RAM",
			humanize.Bytes(m.memory.Used),
			humanize.Bytes(m.memory.Total),
			fmt.Sprintf("%.1f%%", m.memory.UsedPercent),
		})
	}
	if m.swap != nil {
		memRows = append(memRows, table.Row{
			"Swap",
			humanize.Bytes(m.swap.Used),
			humanize.Bytes(m.swap.Total),
			fmt.Sprintf("%.1f%%", m.swap.UsedPercent),
		})
	}
	m.memTable.SetRows(memRows)
	m.memTable.SetHeight(len(memRows))

	var diskRows []table.Row
	for _, partition := range m.diskPartitions {
		if usage, ok := m.diskUsage[partition.Mountpoint]; ok {
			diskRows = append(diskRows, table.Row{
				partition.Device,
				partition.Mountpoint,
				humanize.Bytes(usage.Used),
				humanize.Bytes(usage.Total),
				fmt.Sprintf("%.1f%%", usage.UsedPercent),
			})
		}
	}
	sort.Slice(diskRows, func(i, j int) bool {
		iPercent := strings.TrimSuffix(diskRows[i][4], "%")
		jPercent := strings.TrimSuffix(diskRows[j][4], "%")
		var iVal, jVal float64
		fmt.Sscanf(iPercent, "%f", &iVal)
		fmt.Sscanf(jPercent, "%f", &jVal)
		return iVal > jVal
	})
	m.diskTable.SetRows(diskRows)

	var netRows []table.Row
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if stats, ok := m.netStats[iface.Name]; ok {
			addrs, _ := iface.Addrs()
			var ipv4s []string
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
					ipv4s = append(ipv4s, ipnet.IP.String())
				}
			}
			netRows = append(netRows, table.Row{
				stats.Name,
				strings.Join(ipv4s, ", "),
				humanize.Bytes(uint64(stats.BytesRecv)),
				humanize.Bytes(uint64(stats.BytesSent)),
			})
		}
	}
	m.netTable.SetRows(netRows)

	var statusRows []table.Row
	for _, check := range m.statusChecks {
		statusRows = append(statusRows, table.Row{
			check.name,
			getStatusSymbol(check.status),
		})
	}
	m.statusTable.SetRows(statusRows)

	if m.k8sClient != nil {
		var k8sRows []table.Row
		for _, ns := range m.namespaces {
			status := "🟢"
			if ns.Status.Phase != "Active" {
				status = "🔴"
			}
			k8sRows = append(k8sRows, table.Row{
				ns.Name,
				status,
				humanize.Time(ns.CreationTimestamp.Time),
			})
		}
		m.k8sTable.SetRows(k8sRows)
	}
}

func getStatusSymbol(ok bool) string {
	if ok {
		return "🟢"
	}
	return "🔴"
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.currentView == networkDetailView {
		return m.networkDetailView()
	}

	availWidth := m.width
	minColumnWidth := 85
	useVerticalLayout := availWidth < minColumnWidth*2

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7287fd")).
		Padding(0, 0)

	if useVerticalLayout {
		style = style.Width(availWidth - 2)
	} else {
		style = style.Width(availWidth/2 - 2)
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8caaee")).
		Bold(true)

	// Status section at the top
	statusSection := style.Copy().Width(availWidth - 2).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			headerStyle.Render("Status"),
			m.statusTable.View(),
		),
	)

	var cpuSection string
	if m.loadAvg != nil {
		cpuSection = style.Copy().Width(availWidth/3 - 2).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				headerStyle.Render(fmt.Sprintf("CPU %s", m.getFocusIndicator(cpuTableFocus))),
				m.cpuTable.View(),
				"",
				"",
				"",
				fmt.Sprintf("Load: %.2f %.2f %.2f",
					m.loadAvg.Load1,
					m.loadAvg.Load5,
					m.loadAvg.Load15),
			),
		)
	} else {
		cpuSection = style.Copy().Width(availWidth/3 - 2).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				headerStyle.Render(fmt.Sprintf("CPU %s", m.getFocusIndicator(cpuTableFocus))),
				m.cpuTable.View(),
				"",
				"",
				"",
				"Load: N/A",
			),
		)
	}

	diskSection := style.Copy().Width(2*availWidth/3 - 2).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			headerStyle.Render(fmt.Sprintf("Disks %s", m.getFocusIndicator(diskTableFocus))),
			m.diskTable.View(),
		),
	)

	memSection := style.Copy().Width(2*availWidth/3 - 2).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			headerStyle.Render("Memory"),
			m.memTable.View(),
		),
	)

	rightStack := lipgloss.JoinVertical(lipgloss.Left, diskSection, memSection)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, cpuSection, rightStack)

	netSection := style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			headerStyle.Render(fmt.Sprintf("Network %s", m.getFocusIndicator(netTableFocus))),
			m.netTable.View(),
		),
	)

	var k8sSection string
	if m.k8sClient != nil {
		k8sSection = style.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				headerStyle.Render("Kubernetes"),
				m.k8sTable.View(),
			),
		)
	}

	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, netSection, k8sSection)
	finalLayout := lipgloss.JoinVertical(lipgloss.Left,
		statusSection,
		topRow,
		bottomRow,
	)

	return lipgloss.NewStyle().
		MaxWidth(m.width).
		MaxHeight(m.height).
		Render(finalLayout)
}

func (m model) networkDetailView() string {
	if stats, ok := m.netStats[m.selectedIface]; ok {
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7287fd")).
			Padding(1, 2).
			Width(m.width - 4)

		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8caaee")).
			Bold(true)

		content := []string{
			headerStyle.Render(fmt.Sprintf("Interface: %s", m.selectedIface)),
			"",
			fmt.Sprintf("RX Bytes:     %s", humanize.Bytes(stats.BytesRecv)),
			fmt.Sprintf("RX Packets:   %d", stats.PacketsRecv),
			fmt.Sprintf("RX Errors:    %d", stats.Errin),
			fmt.Sprintf("RX Dropped:   %d", stats.Dropin),
			"",
			fmt.Sprintf("TX Bytes:     %s", humanize.Bytes(stats.BytesSent)),
			fmt.Sprintf("TX Packets:   %d", stats.PacketsSent),
			fmt.Sprintf("TX Errors:    %d", stats.Errout),
			fmt.Sprintf("TX Dropped:   %d", stats.Dropout),
			"",
			"Press ESC to return",
		}

		return style.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			content...,
		))
	}

	return "Interface not found"
}

func (m model) getFocusIndicator(t focusedTable) string {
	if m.focusedTable == t {
		return "●"
	}
	return ""
}

var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Aliases: []string{"dash"},
	Short:   "Interactive system dashboard",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(initialModel(),
			tea.WithAltScreen(),
			tea.WithMouseCellMotion())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running program: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}
