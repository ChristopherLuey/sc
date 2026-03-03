package nodes

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/christopherluey/sc/internal/slurm"
)

var (
	stateColors = map[string]color.Color{
		"IDLE":      lipgloss.Color("#10B981"),
		"MIXED":     lipgloss.Color("#F59E0B"),
		"ALLOCATED": lipgloss.Color("#EF4444"),
		"DOWN":      lipgloss.Color("#6B7280"),
		"DRAIN":     lipgloss.Color("#6B7280"),
		"DRAINED":   lipgloss.Color("#6B7280"),
		"DRAINING":  lipgloss.Color("#F59E0B"),
	}
)

type Model struct {
	table   table.Model
	nodes   []slurm.NodeInfo
	width   int
	height  int
	loading bool
}

func New() Model {
	cols := columns(80)
	t := table.New(
		table.WithColumns(cols),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#6B7280"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#F9FAFB")).
		Background(lipgloss.Color("#7C3AED"))
	t.SetStyles(s)

	return Model{table: t, loading: true}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.table.SetWidth(w)
	m.table.SetHeight(h - 2)
	m.table.SetColumns(columns(w))
}

func (m *Model) SetNodes(nodes []slurm.NodeInfo) {
	m.nodes = nodes
	m.loading = false
	rows := make([]table.Row, len(nodes))
	for i, n := range nodes {
		rows[i] = table.Row{
			n.Name,
			n.State,
			strings.Join(n.Partitions, ","),
			fmt.Sprintf("%d/%d", n.CPUsAlloc, n.CPUsTotal),
			fmt.Sprintf("%d/%dG", n.MemAlloc/1024, n.MemTotal/1024),
			gpuString(n),
		}
	}
	m.table.SetRows(rows)
}

var hintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))

func (m Model) Hint() string {
	return hintStyle.Render("  ↑/↓: navigate  Tab/Shift+Tab: switch tabs  r: refresh  ?: help  q: quit")
}

func (m Model) View() string {
	if m.loading {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Render("  Loading nodes...")
	}
	if len(m.nodes) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Render("  No nodes found")
	}

	// Apply state colors to rows
	view := m.table.View()
	for _, n := range m.nodes {
		if c, ok := stateColors[n.State]; ok {
			style := lipgloss.NewStyle().Foreground(c)
			view = strings.Replace(view, n.State, style.Render(n.State), 1)
		}
	}
	return view
}

func gpuString(n slurm.NodeInfo) string {
	if n.GPUsTotal == 0 {
		return "-"
	}
	if n.GPUType != "" {
		return fmt.Sprintf("%s %d/%d", n.GPUType, n.GPUsAlloc, n.GPUsTotal)
	}
	return fmt.Sprintf("%d/%d", n.GPUsAlloc, n.GPUsTotal)
}

func columns(width int) []table.Column {
	// Distribute width: Node(15) State(12) Partition(20) CPUs(10) Mem(12) GPUs(rest)
	nameW := 15
	stateW := 12
	partW := 20
	cpuW := 10
	memW := 12
	gpuW := width - nameW - stateW - partW - cpuW - memW - 6
	if gpuW < 15 {
		gpuW = 15
	}

	return []table.Column{
		{Title: "Node", Width: nameW},
		{Title: "State", Width: stateW},
		{Title: "Partition", Width: partW},
		{Title: "CPUs", Width: cpuW},
		{Title: "Memory", Width: memW},
		{Title: "GPUs", Width: gpuW},
	}
}
