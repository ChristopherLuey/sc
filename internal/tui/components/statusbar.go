package components

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

var (
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#6B7280")).
			PaddingBottom(1)

	connStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
	disconnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)
)

func RenderStatusBar(connected bool, user, host, partition string, width int, message string) string {
	var status string
	if connected {
		status = connStyle.Render("● Connected")
	} else {
		status = disconnStyle.Render("○ Disconnected")
	}

	info := fmt.Sprintf(" %s  %s@%s  partition:%s", status, user, host, partition)

	if message != "" {
		info += "  " + message
	}

	return statusStyle.Width(width).Render(info)
}
