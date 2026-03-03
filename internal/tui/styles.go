package tui

import "charm.land/lipgloss/v2"

var (
	// Colors
	colorPrimary   = lipgloss.Color("#7C3AED")
	colorSecondary = lipgloss.Color("#6366F1")
	colorGreen     = lipgloss.Color("#10B981")
	colorYellow    = lipgloss.Color("#F59E0B")
	colorRed       = lipgloss.Color("#EF4444")
	colorGray      = lipgloss.Color("#6B7280")
	colorDim       = lipgloss.Color("#9CA3AF")
	colorWhite     = lipgloss.Color("#F9FAFB")
	colorBg        = lipgloss.Color("#1F2937")
	colorBgDark    = lipgloss.Color("#111827")

	// Tab bar
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorPrimary).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(colorDim).
				Padding(0, 2)

	tabBarStyle = lipgloss.NewStyle().
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorGray)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorGray)

	connectedStyle   = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	disconnectedStyle = lipgloss.NewStyle().Foreground(colorRed).Bold(true)

	// Table
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorWhite).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colorGray)

	tableSelectedStyle = lipgloss.NewStyle().
				Foreground(colorWhite).
				Background(colorPrimary)

	// State colors
	stateIdleStyle    = lipgloss.NewStyle().Foreground(colorGreen)
	stateMixedStyle   = lipgloss.NewStyle().Foreground(colorYellow)
	stateAllocStyle   = lipgloss.NewStyle().Foreground(colorRed)
	stateDownStyle    = lipgloss.NewStyle().Foreground(colorGray)
	stateRunningStyle = lipgloss.NewStyle().Foreground(colorGreen)
	statePendingStyle = lipgloss.NewStyle().Foreground(colorYellow)
	stateFailedStyle  = lipgloss.NewStyle().Foreground(colorRed)

	// Form
	focusedFieldStyle = lipgloss.NewStyle().Foreground(colorPrimary)
	blurredFieldStyle = lipgloss.NewStyle().Foreground(colorDim)
	labelStyle        = lipgloss.NewStyle().Foreground(colorWhite).Width(18)
	buttonStyle       = lipgloss.NewStyle().
				Foreground(colorWhite).
				Background(colorPrimary).
				Padding(0, 3)
	buttonBlurredStyle = lipgloss.NewStyle().
				Foreground(colorDim).
				Padding(0, 3)

	// Help
	helpStyle = lipgloss.NewStyle().Foreground(colorDim)

	// Messages
	successStyle = lipgloss.NewStyle().Foreground(colorGreen)
	errorStyle   = lipgloss.NewStyle().Foreground(colorRed)

	// Title
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
)
