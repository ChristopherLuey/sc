package jobs

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/christopherluey/clustertui/internal/slurm"
)

var (
	stateColors = map[string]color.Color{
		"RUNNING":   lipgloss.Color("#10B981"),
		"PENDING":   lipgloss.Color("#F59E0B"),
		"FAILED":    lipgloss.Color("#EF4444"),
		"CANCELLED": lipgloss.Color("#6B7280"),
		"COMPLETED": lipgloss.Color("#6B7280"),
		"TIMEOUT":   lipgloss.Color("#EF4444"),
	}
)

type CancelRequestMsg struct{ JobID string }
type CancelConfirmMsg struct{ JobID string }
type CancelDenyMsg struct{}

type Model struct {
	table       table.Model
	allJobs     []slurm.JobInfo
	filtered    []slurm.JobInfo
	showAllUsers bool
	currentUser string
	width       int
	height      int
	loading     bool
	confirming  string // job ID being confirmed for cancel
	message     string
}

func New(user string) Model {
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

	return Model{table: t, currentUser: user, loading: true}
}

func (m Model) Init() tea.Cmd {
	return nil
}

var (
	keyToggleAll = key.NewBinding(key.WithKeys("a"))
	keyCancel    = key.NewBinding(key.WithKeys("d"))
	keyConfirmY  = key.NewBinding(key.WithKeys("y"))
	keyConfirmN  = key.NewBinding(key.WithKeys("n", "escape"))
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.confirming != "" {
			switch {
			case key.Matches(msg, keyConfirmY):
				jobID := m.confirming
				m.confirming = ""
				return m, func() tea.Msg { return CancelConfirmMsg{JobID: jobID} }
			case key.Matches(msg, keyConfirmN):
				m.confirming = ""
				return m, nil
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, keyToggleAll):
			m.showAllUsers = !m.showAllUsers
			m.applyFilter()
			return m, nil
		case key.Matches(msg, keyCancel):
			if len(m.filtered) > 0 {
				cursor := m.table.Cursor()
				if cursor >= 0 && cursor < len(m.filtered) {
					job := m.filtered[cursor]
					if job.User == m.currentUser {
						m.confirming = job.JobID
					} else {
						m.message = "Can only cancel your own jobs"
					}
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.table.SetWidth(w)
	m.table.SetHeight(h - 4)
	m.table.SetColumns(columns(w))
}

func (m *Model) SetJobs(jobs []slurm.JobInfo) {
	m.allJobs = jobs
	m.loading = false
	m.applyFilter()
}

func (m *Model) SetMessage(msg string) {
	m.message = msg
}

func (m *Model) applyFilter() {
	if m.showAllUsers {
		m.filtered = m.allJobs
	} else {
		m.filtered = nil
		for _, j := range m.allJobs {
			if j.User == m.currentUser {
				m.filtered = append(m.filtered, j)
			}
		}
	}
	rows := make([]table.Row, len(m.filtered))
	for i, j := range m.filtered {
		rows[i] = table.Row{
			j.JobID,
			truncate(j.Name, 20),
			j.User,
			j.Partition,
			j.State,
			j.TimeUsed,
			j.Nodes,
			fmt.Sprintf("%d", j.GPUs),
			j.Reason,
		}
	}
	m.table.SetRows(rows)
}

var hintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))

func (m Model) Hint() string {
	return hintStyle.Render("  ↑/↓: navigate  a: filter  d: cancel  Tab/Shift+Tab: switch tabs  ?: help  q: quit")
}

func (m Model) View() string {
	if m.loading {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Render("  Loading jobs...")
	}

	var header string
	if m.showAllUsers {
		header = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Render("  Showing all jobs  [a] toggle filter")
	} else {
		header = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Render(fmt.Sprintf("  Showing jobs for %s  [a] toggle filter", m.currentUser))
	}

	if m.confirming != "" {
		header += lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).Bold(true).
			Render(fmt.Sprintf("  Cancel job %s? [y/n]", m.confirming))
	} else if m.message != "" {
		header += "  " + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Render(m.message)
	}

	if len(m.filtered) == 0 {
		return header + "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Render("  No jobs found")
	}

	// Apply state colors
	view := m.table.View()
	for _, j := range m.filtered {
		if c, ok := stateColors[j.State]; ok {
			style := lipgloss.NewStyle().Foreground(c)
			view = strings.Replace(view, j.State, style.Render(j.State), 1)
		}
	}

	return header + "\n" + view
}

func columns(width int) []table.Column {
	idW := 10
	nameW := 20
	userW := 10
	partW := 12
	stateW := 10
	timeW := 12
	nodeW := 10
	gpuW := 5
	reasonW := width - idW - nameW - userW - partW - stateW - timeW - nodeW - gpuW - 9
	if reasonW < 10 {
		reasonW = 10
	}

	return []table.Column{
		{Title: "JobID", Width: idW},
		{Title: "Name", Width: nameW},
		{Title: "User", Width: userW},
		{Title: "Partition", Width: partW},
		{Title: "State", Width: stateW},
		{Title: "Time", Width: timeW},
		{Title: "Nodes", Width: nodeW},
		{Title: "GPUs", Width: gpuW},
		{Title: "Reason", Width: reasonW},
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
