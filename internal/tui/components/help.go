package components

import "charm.land/lipgloss/v2"

var (
	helpBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(1, 2).
			Foreground(lipgloss.Color("#F9FAFB"))

	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Width(16)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))
)

type HelpEntry struct {
	Key  string
	Desc string
}

func RenderHelp(width, height int) string {
	entries := []HelpEntry{
		{"1/2/3", "Switch to tab"},
		{"Tab/Shift+Tab", "Cycle tabs"},
		{"r", "Refresh data"},
		{"?", "Toggle help"},
		{"q / Ctrl+C", "Quit"},
		{"", ""},
		{"Nodes View", ""},
		{"↑/↓ / j/k", "Navigate nodes"},
		{"", ""},
		{"Jobs View", ""},
		{"↑/↓ / j/k", "Navigate jobs"},
		{"a", "Toggle all/my jobs"},
		{"d", "Cancel selected job"},
		{"y/n", "Confirm/deny cancel"},
		{"", ""},
		{"Submit View", ""},
		{"↑/↓", "Navigate fields"},
		{"◀/▶", "Cycle presets"},
		{"0-9 / : / .", "Type to override value"},
		{"Enter", "Confirm input / submit"},
		{"Esc", "Cancel custom input"},
		{"Ctrl+F", "Browse remote files"},
		{"Space", "Toggle node in overlay"},
	}

	title := helpTitleStyle.Render("Keybindings")
	lines := title + "\n\n"

	for _, e := range entries {
		if e.Key == "" && e.Desc == "" {
			lines += "\n"
		} else if e.Desc == "" {
			lines += helpTitleStyle.Render(e.Key) + "\n"
		} else {
			lines += helpKeyStyle.Render(e.Key) + helpDescStyle.Render(e.Desc) + "\n"
		}
	}

	box := helpBoxStyle.Render(lines)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
