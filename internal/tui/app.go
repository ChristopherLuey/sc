package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/christopherluey/clustertui/internal/cluster"
	"github.com/christopherluey/clustertui/internal/config"
	"github.com/christopherluey/clustertui/internal/tui/components"
	"github.com/christopherluey/clustertui/internal/tui/views/jobs"
	"github.com/christopherluey/clustertui/internal/tui/views/nodes"
	"github.com/christopherluey/clustertui/internal/tui/views/submit"
)

const (
	tabNodes  = 0
	tabJobs   = 1
	tabSubmit = 2
)

// Setup fields for first-run config
const (
	setupHost = iota
	setupUser
	setupIdentity
	setupFieldCount
)

type App struct {
	cfg       *config.Config
	svc       *cluster.Service
	activeTab int
	width     int
	height    int

	nodesView  nodes.Model
	jobsView   jobs.Model
	submitView submit.Model

	connected bool
	showHelp  bool
	err       error
	statusMsg string

	// First-run setup
	firstRun     bool
	setupInputs  [setupFieldCount]textinput.Model
	setupFocused int
	setupSpinner spinner.Model

	// Connecting state
	connecting bool
}

func NewApp(cfg *config.Config, firstRun bool) *App {
	app := &App{
		cfg:      cfg,
		svc:      cluster.New(cfg),
		firstRun: firstRun,
	}

	app.nodesView = nodes.New()
	app.jobsView = jobs.New(cfg.SSH.User)
	app.submitView = submit.New(
		cfg.JobDefaults.Account,
		cfg.Cluster.DefaultPartition,
		cfg.JobDefaults.CPUsPerTask,
		cfg.JobDefaults.Memory,
		cfg.JobDefaults.TimeLimit,
		cfg.SSH.User,
	)

	if firstRun {
		for i := range app.setupInputs {
			app.setupInputs[i] = textinput.New()
			app.setupInputs[i].SetWidth(50)
			app.setupInputs[i].CharLimit = 256
		}
		app.setupInputs[setupHost].Placeholder = "sc.stanford.edu"
		app.setupInputs[setupHost].SetValue(cfg.SSH.Host)
		app.setupInputs[setupUser].Placeholder = "username"
		app.setupInputs[setupIdentity].Placeholder = "~/.ssh/id_ed25519"
		app.setupInputs[setupHost].Focus()

		app.setupSpinner = spinner.New(
			spinner.WithSpinner(spinner.Dot),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))),
		)
	}

	return app
}

func (a *App) Init() tea.Cmd {
	if a.firstRun {
		return a.setupSpinner.Tick
	}
	a.connecting = true
	return tea.Batch(connectCmd(a.svc), tickCmd(a.cfg.Cluster.RefreshInterval))
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		viewH := a.height - 4 // tab bar + status bar
		a.nodesView.SetSize(a.width, viewH)
		a.jobsView.SetSize(a.width, viewH)
		a.submitView.SetSize(a.width, viewH)
		return a, nil

	case ConnectedMsg:
		a.connected = true
		a.connecting = false
		a.statusMsg = ""
		// Reinitialize jobs view with resolved user
		a.jobsView = jobs.New(a.cfg.SSH.User)
		a.jobsView.SetSize(a.width, a.height-4)
		return a, tea.Batch(fetchNodesCmd(a.svc), fetchJobsCmd(a.svc))

	case ConnectErrorMsg:
		a.connecting = false
		a.err = msg.Err
		a.statusMsg = "Connection failed: " + msg.Err.Error()
		return a, nil

	case NodesLoadedMsg:
		a.nodesView.SetNodes(msg.Nodes)
		a.submitView.SetClusterData(msg.Nodes)
		return a, nil

	case JobsLoadedMsg:
		a.jobsView.SetJobs(msg.Jobs)
		return a, nil

	case JobSubmittedMsg:
		a.submitView.SetMessage(fmt.Sprintf("Job %s submitted successfully!", msg.JobID), false)
		return a, fetchJobsCmd(a.svc)

	case JobCancelledMsg:
		a.jobsView.SetMessage(fmt.Sprintf("Job %s cancelled", msg.JobID))
		return a, fetchJobsCmd(a.svc)

	case jobs.CancelConfirmMsg:
		return a, cancelJobCmd(a.svc, msg.JobID)

	case submit.BrowseRequestMsg:
		return a, listDirCmd(a.svc, msg.Path)

	case submit.SubmitRequestMsg:
		return a, submitJobCmd(a.svc, msg.Sub)

	case ErrorMsg:
		a.statusMsg = "Error: " + msg.Err.Error()
		return a, nil

	case TickMsg:
		if a.connected {
			var cmds []tea.Cmd
			switch a.activeTab {
			case tabNodes:
				cmds = append(cmds, fetchNodesCmd(a.svc))
			case tabJobs:
				cmds = append(cmds, fetchJobsCmd(a.svc))
			case tabSubmit:
				// Keep node list fresh for include/exclude selectors
				cmds = append(cmds, fetchNodesCmd(a.svc))
			}
			cmds = append(cmds, tickCmd(a.cfg.Cluster.RefreshInterval))
			return a, tea.Batch(cmds...)
		}
		return a, tickCmd(a.cfg.Cluster.RefreshInterval)

	case tea.KeyPressMsg:
		// First-run setup key handling
		if a.firstRun {
			return a.updateSetup(msg)
		}

		// Help overlay
		if a.showHelp {
			a.showHelp = false
			return a, nil
		}

		// Tab/Shift+Tab always switch tabs
		switch {
		case key.Matches(msg, Keys.NextTab):
			a.submitView.CloseOverlay()
			a.activeTab = (a.activeTab + 1) % 3
			return a, nil
		case key.Matches(msg, Keys.PrevTab):
			a.submitView.CloseOverlay()
			a.activeTab = (a.activeTab + 2) % 3
			return a, nil
		}

		// Other global keys — safe when submit text input is NOT focused
		submitTextFocused := a.activeTab == tabSubmit && a.submitView.TextInputFocused()
		if !submitTextFocused {
			switch {
			case key.Matches(msg, Keys.Quit):
				a.svc.Close()
				return a, tea.Quit
			case key.Matches(msg, Keys.Help):
				a.showHelp = true
				return a, nil
			case key.Matches(msg, Keys.Tab1):
				a.submitView.CloseOverlay()
				a.activeTab = tabNodes
				return a, nil
			case key.Matches(msg, Keys.Tab2):
				a.submitView.CloseOverlay()
				a.activeTab = tabJobs
				return a, nil
			case key.Matches(msg, Keys.Tab3):
				a.submitView.CloseOverlay()
				a.activeTab = tabSubmit
				return a, nil
			case key.Matches(msg, Keys.Refresh):
				if a.connected {
					return a, tea.Batch(fetchNodesCmd(a.svc), fetchJobsCmd(a.svc))
				}
				return a, nil
			}
		} else {
			// Submit text input focused: only ctrl+c quits
			if key.Matches(msg, Keys.Quit) && msg.String() == "ctrl+c" {
				a.svc.Close()
				return a, tea.Quit
			}
		}
	}

	// Route to active view
	var cmd tea.Cmd
	switch a.activeTab {
	case tabNodes:
		a.nodesView, cmd = a.nodesView.Update(msg)
	case tabJobs:
		a.jobsView, cmd = a.jobsView.Update(msg)
	case tabSubmit:
		a.submitView, cmd = a.submitView.Update(msg)
	}
	return a, cmd
}

func (a *App) makeView(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (a *App) View() tea.View {
	if a.firstRun {
		return a.makeView(a.viewSetup())
	}

	if a.showHelp {
		return a.makeView(components.RenderHelp(a.width, a.height))
	}

	tabBar := components.RenderTabBar(a.activeTab, a.width)

	var view, hint string
	if a.connecting {
		s := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
		view = s.Render(fmt.Sprintf("\n  Connecting to %s...", a.cfg.SSH.Host))
	} else if !a.connected && a.err != nil {
		s := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
		view = s.Render(fmt.Sprintf("\n  Connection failed: %v\n\n  Press 'r' to retry or 'q' to quit.", a.err))
	} else {
		switch a.activeTab {
		case tabNodes:
			view = a.nodesView.View()
			hint = a.nodesView.Hint()
		case tabJobs:
			view = a.jobsView.View()
			hint = a.jobsView.Hint()
		case tabSubmit:
			view = a.submitView.View()
			hint = a.submitView.Hint()
		}
	}

	statusBar := components.RenderStatusBar(
		a.connected,
		a.cfg.SSH.User,
		a.cfg.SSH.Host,
		a.cfg.Cluster.DefaultPartition,
		a.width,
		a.statusMsg,
	)

	// Layout: tabbar + view (padded) + hint + statusbar
	viewHeight := a.height - lipgloss.Height(tabBar) - lipgloss.Height(statusBar)
	if hint != "" {
		viewHeight-- // reserve 1 line for hint
	}
	if viewHeight < 0 {
		viewHeight = 0
	}
	paddedView := lipgloss.NewStyle().Height(viewHeight).Render(view)

	var content string
	if hint != "" {
		content = lipgloss.JoinVertical(lipgloss.Left,
			tabBar,
			paddedView,
			hint,
			statusBar,
		)
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left,
			tabBar,
			lipgloss.NewStyle().Height(viewHeight+1).Render(view),
			statusBar,
		)
	}

	return a.makeView(content)
}

// First-run setup

var setupLabels = [setupFieldCount]string{
	"SSH Host",
	"Username",
	"Identity File",
}

func (a *App) updateSetup(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return a, tea.Quit
	case "tab", "down":
		if a.setupFocused < setupFieldCount {
			a.setupInputs[a.setupFocused].Blur()
		}
		a.setupFocused = (a.setupFocused + 1) % (setupFieldCount + 1) // +1 for connect button
		if a.setupFocused < setupFieldCount {
			return a, a.setupInputs[a.setupFocused].Focus()
		}
		return a, nil
	case "shift+tab", "up":
		if a.setupFocused < setupFieldCount {
			a.setupInputs[a.setupFocused].Blur()
		}
		a.setupFocused--
		if a.setupFocused < 0 {
			a.setupFocused = setupFieldCount
		}
		if a.setupFocused < setupFieldCount {
			return a, a.setupInputs[a.setupFocused].Focus()
		}
		return a, nil
	case "enter":
		if a.setupFocused == setupFieldCount {
			// Connect button
			host := a.setupInputs[setupHost].Value()
			user := a.setupInputs[setupUser].Value()
			identity := a.setupInputs[setupIdentity].Value()

			if host == "" {
				host = "sc.stanford.edu"
			}
			if user == "" {
				a.statusMsg = "Username is required"
				return a, nil
			}

			a.cfg.SSH.Host = host
			a.cfg.SSH.User = user
			a.cfg.SSH.IdentityFile = identity

			// Save config
			_ = config.Save(a.cfg, "")

			a.firstRun = false
			a.connecting = true
			a.svc = cluster.New(a.cfg)
			a.jobsView = jobs.New(user)
			a.jobsView.SetSize(a.width, a.height-4)

			return a, tea.Batch(connectCmd(a.svc), tickCmd(a.cfg.Cluster.RefreshInterval))
		}
		// Next field
		if a.setupFocused < setupFieldCount {
			a.setupInputs[a.setupFocused].Blur()
		}
		a.setupFocused = (a.setupFocused + 1) % (setupFieldCount + 1)
		if a.setupFocused < setupFieldCount {
			return a, a.setupInputs[a.setupFocused].Focus()
		}
		return a, nil
	}

	// Update focused input
	if a.setupFocused < setupFieldCount {
		var cmd tea.Cmd
		a.setupInputs[a.setupFocused], cmd = a.setupInputs[a.setupFocused].Update(msg)
		return a, cmd
	}
	return a, nil
}

func (a *App) viewSetup() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED")).
		Render("ClusterTUI Setup")

	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Render("Configure your SSH connection to the cluster")

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F9FAFB")).Width(18)

	var fields string
	for i := 0; i < setupFieldCount; i++ {
		label := labelStyle.Render(setupLabels[i])
		input := a.setupInputs[i].View()
		fields += fmt.Sprintf("  %s %s\n", label, input)
	}

	var button string
	if a.setupFocused == setupFieldCount {
		btnStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FAFB")).
			Background(lipgloss.Color("#7C3AED")).
			Padding(0, 3)
		button = "  " + btnStyle.Render("[ Connect ]")
	} else {
		btnStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Padding(0, 3)
		button = "  " + btnStyle.Render("[ Connect ]")
	}

	var msg string
	if a.statusMsg != "" {
		msg = "\n  " + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Render(a.statusMsg)
	}

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Render("  Tab: navigate  Enter: connect  Ctrl+C: quit")

	content := fmt.Sprintf("\n  %s\n  %s\n\n%s\n%s%s\n\n%s",
		title, subtitle, fields, button, msg, help)

	return lipgloss.Place(a.width, a.height, lipgloss.Left, lipgloss.Center, content)
}
