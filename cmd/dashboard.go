package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Display system metrics in an interactive dashboard",
	Long: `Display all system metrics in an interactive dashboard.
Use arrow keys to navigate between sections.
Press 'q' to quit.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(initialModel())
		_, err := p.Run()
		return err
	},
}

type model struct {
	cpuPercents   []float64
	loadAvg       *load.AvgStat
	memory        *mem.VirtualMemoryStat
	swap          *mem.SwapMemoryStat
	diskStats     map[string]disk.IOCountersStat
	diskPartitions []disk.PartitionStat
	diskUsage     map[string]*disk.UsageStat
	netInterfaces []netlink.Link
	netStats      map[string]netlink.LinkStatistics
	width         int
	lastUpdate    time.Time
	diskTable     table.Model
	cpuTable      table.Model
	memTable      table.Model
	netTable      table.Model
}

func initialModel() model {
	// Initialize tables with default styles
	tableStyle := table.DefaultStyles()
	tableStyle.Header = tableStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)

	m := model{
		diskUsage:     make(map[string]*disk.UsageStat),
		netStats:      make(map[string]netlink.LinkStatistics),
		diskStats:     make(map[string]disk.IOCountersStat),
		lastUpdate:    time.Now(),
		cpuPercents:   make([]float64, 0),
		diskPartitions: make([]disk.PartitionStat, 0),
		netInterfaces: make([]netlink.Link, 0),
	}

	// Initialize tables
	m.diskTable = table.New(
		table.WithColumns([]table.Column{
			{Title: "Disk(d)", Width: 20},
			{Title: "Mount(m)", Width: 20},
			{Title: "Used(u)", Width: 15},
			{Title: "Total", Width: 15},
			{Title: "Used%", Width: 10},
		}),
		table.WithHeight(4),
		table.WithStyles(tableStyle),
	)

	m.cpuTable = table.New(
		table.WithColumns([]table.Column{
			{Title: "Core(c)", Width: 10},
			{Title: "Usage(u)", Width: 10},
		}),
		table.WithStyles(tableStyle),
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
			{Title: "RX(r)", Width: 20},
			{Title: "TX(t)", Width: 20},
			{Title: "Total(s)", Width: 20},
		}),
		table.WithStyles(tableStyle),
	)

	return m
}

type tickMsg time.Time

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tickMsg:
		m.lastUpdate = time.Time(msg)
		
		// Update CPU stats
		if percents, err := cpu.Percent(0, true); err == nil {
			m.cpuPercents = percents
		}

		// Update load average
		if loadAvg, err := load.Avg(); err == nil {
			m.loadAvg = loadAvg
		}

		// Update memory stats
		if vmem, err := mem.VirtualMemory(); err == nil {
			m.memory = vmem
		}

		// Update swap stats
		if swap, err := mem.SwapMemory(); err == nil {
			m.swap = swap
		}

		// Update disk stats
		if iostats, err := disk.IOCounters(); err == nil {
			m.diskStats = iostats
		}

		// Update disk partitions
		if partitions, err := disk.Partitions(false); err == nil {
			m.diskPartitions = partitions
			for _, partition := range partitions {
				if usage, err := disk.Usage(partition.Mountpoint); err == nil {
					m.diskUsage[partition.Mountpoint] = usage
				}
			}
		}

		// Update network stats
		if links, err := netlink.LinkList(); err == nil {
			m.netInterfaces = links
			stats := make(map[string]netlink.LinkStatistics)
			for _, link := range links {
				if link.Attrs().Statistics != nil {
					stats[link.Attrs().Name] = *link.Attrs().Statistics
				}
			}
			m.netStats = stats
		}

		// Update tables
		m.updateTables()

		return m, tickCmd()
	}

	return m, nil
}

func (m *model) updateTables() {
	// Update CPU table
	var cpuRows []table.Row
	for i, percent := range m.cpuPercents {
		cpuRows = append(cpuRows, table.Row{
			fmt.Sprintf("%d", i),
			fmt.Sprintf("%.1f%%", percent),
		})
	}
	m.cpuTable.SetRows(cpuRows)

	// Update memory table
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

	// Update disk table
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
	// Sort by usage percentage
	sort.Slice(diskRows, func(i, j int) bool {
		iPercent := strings.TrimSuffix(diskRows[i][4], "%")
		jPercent := strings.TrimSuffix(diskRows[j][4], "%")
		var iVal, jVal float64
		fmt.Sscanf(iPercent, "%f", &iVal)
		fmt.Sscanf(jPercent, "%f", &jVal)
		return iVal > jVal
	})
	m.diskTable.SetRows(diskRows)

	// Update network table
	var netRows []table.Row
	for _, iface := range m.netInterfaces {
		attrs := iface.Attrs()
		if stats, ok := m.netStats[attrs.Name]; ok {
			netRows = append(netRows, table.Row{
				attrs.Name,
				humanize.Bytes(uint64(stats.RxBytes)),
				humanize.Bytes(uint64(stats.TxBytes)),
				humanize.Bytes(uint64(stats.RxBytes + stats.TxBytes)),
			})
		}
	}
	m.netTable.SetRows(netRows)
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7287fd")).
		Padding(0, 1)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8caaee")).
		Bold(true)

	// Create sections with nil checks
	var cpuSection string
	if m.loadAvg != nil {
		cpuSection = style.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				headerStyle.Render("CPU"),
				m.cpuTable.View(),
				fmt.Sprintf("Load: %.2f %.2f %.2f",
					m.loadAvg.Load1,
					m.loadAvg.Load5,
					m.loadAvg.Load15),
			),
		)
	} else {
		cpuSection = style.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				headerStyle.Render("CPU"),
				m.cpuTable.View(),
				"Load: N/A",
			),
		)
	}

	memSection := style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			headerStyle.Render("Memory"),
			m.memTable.View(),
		),
	)

	diskSection := style.Render(
		m.diskTable.View(),
	)

	netSection := style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			headerStyle.Render("Network"),
			m.netTable.View(),
		),
	)

	// Combine sections
	top := lipgloss.JoinHorizontal(lipgloss.Top, cpuSection, memSection)
	bottom := lipgloss.JoinHorizontal(lipgloss.Top, diskSection, netSection)

	return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}
