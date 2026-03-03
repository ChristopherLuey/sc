package components

import "charm.land/lipgloss/v2"

var tabs = []string{"Nodes", "Jobs", "Submit"}

var (
	activeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F9FAFB")).
			Background(lipgloss.Color("#7C3AED")).
			Padding(0, 2)

	inactiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Padding(0, 2)

	barStyle = lipgloss.NewStyle().
			PaddingTop(1).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#6B7280"))
)

func RenderTabBar(activeTab int, width int) string {
	var rendered []string
	for i, t := range tabs {
		if i == activeTab {
			rendered = append(rendered, activeStyle.Render(t))
		} else {
			rendered = append(rendered, inactiveStyle.Render(t))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
	return barStyle.Width(width).Render(row)
}
