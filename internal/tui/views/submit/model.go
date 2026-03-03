package submit

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/christopherluey/clustertui/internal/cluster"
	"github.com/christopherluey/clustertui/internal/slurm"
)

type SubmitRequestMsg struct {
	Sub slurm.JobSubmission
}

type BrowseRequestMsg struct{ Path string }

type DirListingMsg struct {
	Path    string
	Entries []cluster.DirEntry
	Err     error
}

type fieldKind int

const (
	kindSelector fieldKind = iota
	kindText
	kindNodeSelect
)

type field struct {
	label       string
	kind        fieldKind
	options     []string // selector options
	sel         int      // selector index
	input       textinput.Model
	chosen      map[string]bool // node select
	customMode  bool            // selector with custom text entry
	customInput textinput.Model // inline text input for custom value
	allowCustom bool            // whether this selector supports custom mode
}

const (
	fAccount = iota
	fPartition
	fJobName
	fGPUCount
	fGPUType
	fCPUs
	fMemory
	fTimeLimit
	fOutput
	fWorkDir
	fScript
	fInclude
	fExclude
	fCount
)

type Model struct {
	fields         [fCount]field
	focused        int
	width, height  int
	availableNodes []string
	inNodeSelect   bool
	nodeSelectIdx  int
	nodeCursor     int
	nodeScroll     int
	message        string
	messageIsErr   bool

	// File browser
	sshUser        string
	inFileBrowser  bool
	browserLoading bool
	browserPath    string
	browserEntries []cluster.DirEntry
	browserCursor  int
	browserScroll  int
	browserErr     string
}

func newCustomInput() textinput.Model {
	ti := textinput.New()
	ti.SetWidth(16)
	ti.CharLimit = 32
	return ti
}

func New(account, partition string, cpus int, memory, timeLimit, sshUser string) Model {
	var m Model
	m.sshUser = sshUser

	// Account selector
	accounts := []string{account}
	m.fields[fAccount] = field{label: "Account", kind: kindSelector, options: accounts, sel: 0}

	// Partition selector
	partitions := []string{"viscam", "viscam-interactive", "svl", "svl-interactive"}
	partIdx := 0
	for i, p := range partitions {
		if p == partition {
			partIdx = i
			break
		}
	}
	m.fields[fPartition] = field{label: "Partition", kind: kindSelector, options: partitions, sel: partIdx}

	// Job Name text
	ti := textinput.New()
	ti.Placeholder = "my-job"
	ti.SetWidth(40)
	ti.CharLimit = 256
	m.fields[fJobName] = field{label: "Job Name", kind: kindText, input: ti}

	// GPU Count selector (with custom)
	gpuOpts := make([]string, 9)
	for i := 0; i <= 8; i++ {
		gpuOpts[i] = strconv.Itoa(i)
	}
	m.fields[fGPUCount] = field{label: "GPU Count", kind: kindSelector, options: gpuOpts, sel: 1, allowCustom: true, customInput: newCustomInput()}

	// GPU Type selector (dynamic, starts with just "any")
	gpuTypes := []string{"any"}
	m.fields[fGPUType] = field{label: "GPU Type", kind: kindSelector, options: gpuTypes, sel: 0}

	// CPUs per Task selector (with custom)
	cpuOpts := []string{"1", "2", "4", "6", "8", "12", "16", "24", "32", "48", "64", "128"}
	cpuIdx := 3 // default "6"
	cs := strconv.Itoa(cpus)
	for i, c := range cpuOpts {
		if c == cs {
			cpuIdx = i
			break
		}
	}
	m.fields[fCPUs] = field{label: "CPUs per Task", kind: kindSelector, options: cpuOpts, sel: cpuIdx, allowCustom: true, customInput: newCustomInput()}

	// Memory selector (with custom)
	memOpts := []string{"8G", "16G", "32G", "48G", "64G", "96G", "128G", "256G", "512G", "1024G"}
	memIdx := 2 // default "32G"
	for i, v := range memOpts {
		if v == memory {
			memIdx = i
			break
		}
	}
	m.fields[fMemory] = field{label: "Memory", kind: kindSelector, options: memOpts, sel: memIdx, allowCustom: true, customInput: newCustomInput()}

	// Time Limit selector (with custom)
	timeOpts := []string{"1:00:00", "4:00:00", "8:00:00", "12:00:00", "24:00:00", "48:00:00", "72:00:00", "120:00:00"}
	timeIdx := 4 // default "24:00:00"
	for i, v := range timeOpts {
		if v == timeLimit {
			timeIdx = i
			break
		}
	}
	m.fields[fTimeLimit] = field{label: "Time Limit", kind: kindSelector, options: timeOpts, sel: timeIdx, allowCustom: true, customInput: newCustomInput()}

	// Output File text
	oi := textinput.New()
	oi.Placeholder = "slurm-%j.out"
	oi.SetWidth(40)
	oi.CharLimit = 256
	m.fields[fOutput] = field{label: "Output File", kind: kindText, input: oi}

	// Working Directory text
	wi := textinput.New()
	if sshUser != "" {
		wi.Placeholder = "/viscam/u/" + sshUser + "/"
	} else {
		wi.Placeholder = "/viscam/u/<user>/"
	}
	wi.SetWidth(40)
	wi.CharLimit = 256
	m.fields[fWorkDir] = field{label: "Working Dir", kind: kindText, input: wi}

	// Script/Command text
	si := textinput.New()
	si.Placeholder = "/path/to/script.sh or shell command"
	si.SetWidth(40)
	si.CharLimit = 256
	m.fields[fScript] = field{label: "Script/Command", kind: kindText, input: si}

	// Node selects
	m.fields[fInclude] = field{label: "Include Nodes", kind: kindNodeSelect, chosen: make(map[string]bool)}
	m.fields[fExclude] = field{label: "Exclude Nodes", kind: kindNodeSelect, chosen: make(map[string]bool)}

	m.focused = fAccount
	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// SetClusterData extracts node names and unique GPU types from live node data.
func (m *Model) SetClusterData(nodes []slurm.NodeInfo) {
	seen := make(map[string]bool, len(nodes))
	names := make([]string, 0, len(nodes))
	gpuSet := make(map[string]bool)

	for _, n := range nodes {
		if !seen[n.Name] {
			seen[n.Name] = true
			names = append(names, n.Name)
		}
		if n.GPUType != "" {
			gpuSet[n.GPUType] = true
		}
	}

	sort.Strings(names)
	m.availableNodes = names

	// Build GPU type options: "any" + sorted unique types
	gpuTypes := []string{"any"}
	sorted := make([]string, 0, len(gpuSet))
	for t := range gpuSet {
		sorted = append(sorted, t)
	}
	sort.Strings(sorted)
	gpuTypes = append(gpuTypes, sorted...)

	// Preserve current selection if possible
	prev := ""
	if m.fields[fGPUType].sel < len(m.fields[fGPUType].options) {
		prev = m.fields[fGPUType].options[m.fields[fGPUType].sel]
	}
	m.fields[fGPUType].options = gpuTypes
	m.fields[fGPUType].sel = 0
	for i, t := range gpuTypes {
		if t == prev {
			m.fields[fGPUType].sel = i
			break
		}
	}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	inputW := w - 22
	if inputW > 60 {
		inputW = 60
	}
	if inputW < 20 {
		inputW = 20
	}
	for i := range m.fields {
		if m.fields[i].kind == kindText {
			m.fields[i].input.SetWidth(inputW)
		}
	}
}

func (m *Model) SetMessage(msg string, isErr bool) {
	m.message = msg
	m.messageIsErr = isErr
}

func (m *Model) Reset() {
	m.message = ""
}

// TextInputFocused returns true when a text input field or custom input is active,
// signaling to the app that single-char keys (q, r, 1-3) should not
// be intercepted as global shortcuts.
func (m Model) TextInputFocused() bool {
	if m.inNodeSelect || m.inFileBrowser {
		return false
	}
	if m.focused >= fCount {
		return false
	}
	if m.fields[m.focused].kind == kindText {
		return true
	}
	if m.fields[m.focused].customMode {
		return true
	}
	return false
}

// CloseOverlay dismisses the node-select overlay if open.
func (m *Model) CloseOverlay() {
	m.inNodeSelect = false
	m.inFileBrowser = false
}

var (
	keyUp    = key.NewBinding(key.WithKeys("up"))
	keyDown  = key.NewBinding(key.WithKeys("down"))
	keyLeft  = key.NewBinding(key.WithKeys("left"))
	keyRight = key.NewBinding(key.WithKeys("right"))
	keyEnter = key.NewBinding(key.WithKeys("enter"))
	keyEsc   = key.NewBinding(key.WithKeys("esc"))
	keySpace = key.NewBinding(key.WithKeys(" "))
	keyBrowse = key.NewBinding(key.WithKeys("ctrl+f"))
	// Overlay also supports j/k
	keyOverlayUp   = key.NewBinding(key.WithKeys("up", "k"))
	keyOverlayDown = key.NewBinding(key.WithKeys("down", "j"))
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case DirListingMsg:
		m.browserLoading = false
		if msg.Err != nil {
			m.browserErr = msg.Err.Error()
			return m, nil
		}
		m.browserPath = msg.Path
		m.browserEntries = msg.Entries
		m.browserCursor = 0
		m.browserScroll = 0
		m.browserErr = ""
		return m, nil

	case tea.KeyPressMsg:
		if m.inFileBrowser {
			return m.updateFileBrowser(msg)
		}
		if m.inNodeSelect {
			return m.updateNodeSelect(msg)
		}
		return m.updateForm(msg)
	}

	// Pass non-key msgs to focused text input or custom input
	if m.focused < fCount {
		f := &m.fields[m.focused]
		if f.customMode {
			var cmd tea.Cmd
			f.customInput, cmd = f.customInput.Update(msg)
			return m, cmd
		}
		if f.kind == kindText {
			var cmd tea.Cmd
			f.input, cmd = f.input.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) updateForm(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	// Handle custom mode input
	if m.focused < fCount && m.fields[m.focused].customMode {
		return m.updateCustomMode(msg)
	}

	switch {
	case key.Matches(msg, keyDown):
		return m.moveFocus(1)
	case key.Matches(msg, keyUp):
		return m.moveFocus(-1)

	case key.Matches(msg, keyLeft):
		if m.focused < fCount && m.fields[m.focused].kind == kindSelector {
			f := &m.fields[m.focused]
			f.sel--
			if f.sel < 0 {
				f.sel = len(f.options) - 1
			}
		}
		return m, nil

	case key.Matches(msg, keyRight):
		if m.focused < fCount && m.fields[m.focused].kind == kindSelector {
			f := &m.fields[m.focused]
			f.sel = (f.sel + 1) % len(f.options)
		}
		return m, nil

	case key.Matches(msg, keyEnter):
		if m.focused == fCount {
			sub, err := m.buildSubmission()
			if err != nil {
				m.message = err.Error()
				m.messageIsErr = true
				return m, nil
			}
			return m, func() tea.Msg { return SubmitRequestMsg{Sub: sub} }
		}
		if m.focused < fCount && m.fields[m.focused].kind == kindNodeSelect {
			m.inNodeSelect = true
			m.nodeSelectIdx = m.focused
			m.nodeCursor = 0
			m.nodeScroll = 0
			return m, nil
		}
		return m.moveFocus(1)
	}

	// Check if typing a digit/colon on a custom-capable selector → enter custom mode
	if m.focused < fCount && m.fields[m.focused].kind == kindSelector && m.fields[m.focused].allowCustom {
		text := tea.KeyPressMsg(msg).String()
		if len(text) == 1 {
			r := rune(text[0])
			if unicode.IsDigit(r) || r == ':' || r == '.' {
				f := &m.fields[m.focused]
				f.customMode = true
				f.customInput.SetValue(string(r))
				f.customInput.CursorEnd()
				return m, f.customInput.Focus()
			}
		}
	}

	// Ctrl+F on Script field opens file browser
	if m.focused == fScript && key.Matches(msg, keyBrowse) {
		m.inFileBrowser = true
		m.browserLoading = true
		m.browserErr = ""
		m.browserCursor = 0
		m.browserScroll = 0
		// Use Working Dir field value, or default home path
		startPath := strings.TrimSpace(m.fields[fWorkDir].input.Value())
		if startPath == "" {
			if m.sshUser != "" {
				startPath = "/viscam/u/" + m.sshUser
			} else {
				startPath = "~"
			}
		}
		m.browserPath = startPath
		return m, func() tea.Msg { return BrowseRequestMsg{Path: startPath} }
	}

	// Forward to text input
	if m.focused < fCount && m.fields[m.focused].kind == kindText {
		var cmd tea.Cmd
		m.fields[m.focused].input, cmd = m.fields[m.focused].input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) updateCustomMode(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	f := &m.fields[m.focused]

	switch {
	case key.Matches(msg, keyEnter), key.Matches(msg, keyUp), key.Matches(msg, keyDown):
		// Confirm custom value
		val := strings.TrimSpace(f.customInput.Value())
		if val != "" {
			m.applyCustomValue(m.focused, val)
		}
		f.customMode = false
		f.customInput.Blur()
		f.customInput.SetValue("")

		if key.Matches(msg, keyUp) {
			return m.moveFocus(-1)
		}
		if key.Matches(msg, keyDown) {
			return m.moveFocus(1)
		}
		return m, nil

	case key.Matches(msg, keyEsc):
		// Cancel custom mode
		f.customMode = false
		f.customInput.Blur()
		f.customInput.SetValue("")
		return m, nil
	}

	// Forward to custom input
	var cmd tea.Cmd
	f.customInput, cmd = f.customInput.Update(msg)
	return m, cmd
}

// applyCustomValue sets the selector to the custom value, adding it as an option if needed.
func (m *Model) applyCustomValue(idx int, val string) {
	f := &m.fields[idx]
	// Check if value already exists in options
	for i, opt := range f.options {
		if opt == val {
			f.sel = i
			return
		}
	}
	// Add as new option and select it
	f.options = append(f.options, val)
	f.sel = len(f.options) - 1
}

func (m Model) updateNodeSelect(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	n := len(m.availableNodes)
	if n == 0 {
		m.inNodeSelect = false
		return m, nil
	}

	switch {
	case key.Matches(msg, keyOverlayDown):
		if m.nodeCursor < n-1 {
			m.nodeCursor++
		}
	case key.Matches(msg, keyOverlayUp):
		if m.nodeCursor > 0 {
			m.nodeCursor--
		}
	case key.Matches(msg, keySpace):
		node := m.availableNodes[m.nodeCursor]
		f := &m.fields[m.nodeSelectIdx]
		f.chosen[node] = !f.chosen[node]
		if !f.chosen[node] {
			delete(f.chosen, node)
		}
	case key.Matches(msg, keyEnter), key.Matches(msg, keyEsc):
		m.inNodeSelect = false
	}

	// Keep cursor visible in scrollable region
	maxVisible := m.overlayHeight() - 4
	if maxVisible < 1 {
		maxVisible = 1
	}
	if m.nodeCursor < m.nodeScroll {
		m.nodeScroll = m.nodeCursor
	}
	if m.nodeCursor >= m.nodeScroll+maxVisible {
		m.nodeScroll = m.nodeCursor - maxVisible + 1
	}

	return m, nil
}

func (m Model) updateFileBrowser(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.browserLoading {
		// Only allow escape while loading
		if key.Matches(msg, keyEsc) {
			m.inFileBrowser = false
			m.browserLoading = false
		}
		return m, nil
	}

	// Total items = ".." entry (if not at root) + entries
	hasParent := m.browserPath != "/"
	totalItems := len(m.browserEntries)
	if hasParent {
		totalItems++
	}

	switch {
	case key.Matches(msg, keyOverlayDown):
		if m.browserCursor < totalItems-1 {
			m.browserCursor++
		}
	case key.Matches(msg, keyOverlayUp):
		if m.browserCursor > 0 {
			m.browserCursor--
		}
	case key.Matches(msg, keyEnter):
		if totalItems == 0 {
			return m, nil
		}
		// Determine which entry was selected
		idx := m.browserCursor
		if hasParent {
			if idx == 0 {
				// Go to parent directory
				parent := path.Dir(m.browserPath)
				m.browserLoading = true
				m.browserPath = parent
				return m, func() tea.Msg { return BrowseRequestMsg{Path: parent} }
			}
			idx-- // adjust for ".." entry
		}
		if idx < len(m.browserEntries) {
			entry := m.browserEntries[idx]
			if entry.IsDir {
				newPath := path.Join(m.browserPath, entry.Name)
				m.browserLoading = true
				m.browserPath = newPath
				return m, func() tea.Msg { return BrowseRequestMsg{Path: newPath} }
			}
			// File selected — populate Script/Command field
			fullPath := path.Join(m.browserPath, entry.Name)
			m.fields[fScript].input.SetValue(fullPath)
			m.inFileBrowser = false
		}
		return m, nil
	case key.Matches(msg, keyEsc):
		m.inFileBrowser = false
		return m, nil
	}

	// Keep cursor visible
	maxVisible := m.browserOverlayHeight() - 4
	if maxVisible < 1 {
		maxVisible = 1
	}
	if m.browserCursor < m.browserScroll {
		m.browserScroll = m.browserCursor
	}
	if m.browserCursor >= m.browserScroll+maxVisible {
		m.browserScroll = m.browserCursor - maxVisible + 1
	}

	return m, nil
}

func (m Model) moveFocus(dir int) (Model, tea.Cmd) {
	total := fCount + 1 // fields + submit button

	// Blur current input
	if m.focused < fCount {
		f := &m.fields[m.focused]
		if f.kind == kindText {
			f.input.Blur()
		}
		if f.customMode {
			f.customMode = false
			f.customInput.Blur()
			f.customInput.SetValue("")
		}
	}

	m.focused += dir
	if m.focused < 0 {
		m.focused = total - 1
	} else if m.focused >= total {
		m.focused = 0
	}

	// Focus new text input
	if m.focused < fCount && m.fields[m.focused].kind == kindText {
		return m, m.fields[m.focused].input.Focus()
	}
	return m, nil
}

// --- View ---

func (m Model) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED"))
	labelStyle := lipgloss.NewStyle().Width(18)
	focusedLabelStyle := lipgloss.NewStyle().Width(18).Foreground(lipgloss.Color("#7C3AED"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	arrowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))
	selValStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F9FAFB"))
	nodeHintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	customStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))

	b.WriteString(titleStyle.Render("  Submit Job"))
	b.WriteString("\n\n")

	for i := 0; i < fCount; i++ {
		f := &m.fields[i]
		isFocused := i == m.focused

		ls := labelStyle
		if isFocused {
			ls = focusedLabelStyle
		}
		label := ls.Render(f.label)

		var val string
		switch f.kind {
		case kindSelector:
			if f.customMode {
				val = customStyle.Render("→ ") + f.customInput.View()
			} else {
				cur := f.options[f.sel]
				if isFocused {
					val = arrowStyle.Render("◀") + " " + selValStyle.Render(cur) + " " + arrowStyle.Render("▶")
				} else {
					val = dimStyle.Render("◀") + " " + dimStyle.Render(cur) + " " + dimStyle.Render("▶")
				}
			}
		case kindText:
			val = f.input.View()
		case kindNodeSelect:
			summary := nodeSummary(f.chosen)
			if isFocused {
				val = selValStyle.Render("["+summary+"]") + " " + arrowStyle.Render("▸")
			} else {
				val = nodeHintStyle.Render("["+summary+"]") + " " + dimStyle.Render("▸")
			}
		}

		b.WriteString(fmt.Sprintf("  %s %s\n", label, val))
	}

	b.WriteString("\n")

	// Submit button
	if m.focused == fCount {
		btnStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FAFB")).
			Background(lipgloss.Color("#7C3AED")).
			Padding(0, 3)
		b.WriteString("  " + btnStyle.Render("[ Submit ]"))
	} else {
		btnStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Padding(0, 3)
		b.WriteString("  " + btnStyle.Render("[ Submit ]"))
	}
	b.WriteString("\n")

	if m.message != "" {
		b.WriteString("\n")
		if m.messageIsErr {
			errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
			b.WriteString("  " + errStyle.Render(m.message))
		} else {
			okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
			b.WriteString("  " + okStyle.Render(m.message))
		}
	}

	if m.inFileBrowser {
		overlay := m.renderFileBrowserOverlay()
		return m.compositeOverlay(b.String(), overlay)
	}

	if m.inNodeSelect {
		overlay := m.renderNodeOverlay()
		return m.compositeOverlay(b.String(), overlay)
	}

	return b.String()
}

func (m Model) Hint() string {
	if m.inNodeSelect || m.inFileBrowser {
		return ""
	}
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	if m.focused < fCount && m.fields[m.focused].customMode {
		return dim.Render("  Enter: confirm  Esc: cancel  ") + accent.Render("typing custom value")
	}
	hint := "  ↑/↓: navigate  ◀▶: cycle presets  Enter: select/submit  Tab: switch tabs"
	if m.focused == fScript {
		return dim.Render(hint+"  ") + accent.Render("Ctrl+F: browse files")
	}
	if m.focused < fCount && m.fields[m.focused].allowCustom {
		return dim.Render(hint+"  ") + accent.Render("type a number to override")
	}
	return dim.Render(hint)
}

func nodeSummary(chosen map[string]bool) string {
	if len(chosen) == 0 {
		return "none"
	}
	names := make([]string, 0, len(chosen))
	for k := range chosen {
		names = append(names, k)
	}
	sort.Strings(names)
	if len(names) <= 3 {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s +%d more", strings.Join(names[:2], ", "), len(names)-2)
}

func (m Model) overlayHeight() int {
	h := m.height - 8
	if h > len(m.availableNodes)+4 {
		h = len(m.availableNodes) + 4
	}
	if h < 6 {
		h = 6
	}
	return h
}

func (m Model) renderNodeOverlay() string {
	f := &m.fields[m.nodeSelectIdx]
	title := f.label

	overlayW := 36
	if m.width > 50 {
		overlayW = 40
	}
	innerW := overlayW - 4 // border padding

	oh := m.overlayHeight()
	visibleRows := oh - 4 // top border+title, bottom border+help

	var rows []string
	if len(m.availableNodes) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Render("  No nodes loaded yet"))
	} else {
		end := m.nodeScroll + visibleRows
		if end > len(m.availableNodes) {
			end = len(m.availableNodes)
		}
		for i := m.nodeScroll; i < end; i++ {
			node := m.availableNodes[i]
			check := "[ ]"
			if f.chosen[node] {
				check = "[x]"
			}
			cursor := "  "
			if i == m.nodeCursor {
				cursor = "> "
			}
			line := fmt.Sprintf("%s%s %s", cursor, check, node)
			if len(line) > innerW {
				line = line[:innerW]
			}
			style := lipgloss.NewStyle()
			if i == m.nodeCursor {
				style = style.Foreground(lipgloss.Color("#7C3AED")).Bold(true)
			}
			rows = append(rows, style.Render(line))
		}
	}

	// Pad to fill visible area
	for len(rows) < visibleRows {
		rows = append(rows, "")
	}

	content := strings.Join(rows, "\n")
	helpLine := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Render("Space: toggle  Enter: done")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Width(overlayW).
		Padding(0, 1)

	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED")).Render(title)

	return box.Render(header + "\n" + content + "\n" + helpLine)
}

func (m Model) browserOverlayHeight() int {
	items := len(m.browserEntries) + 1 // +1 for ".." parent
	h := m.height - 8
	if h > items+4 {
		h = items + 4
	}
	if h < 8 {
		h = 8
	}
	return h
}

func (m Model) renderFileBrowserOverlay() string {
	overlayW := 52
	if m.width > 70 {
		overlayW = 60
	}
	innerW := overlayW - 4

	oh := m.browserOverlayHeight()
	visibleRows := oh - 4

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED"))
	dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))

	// Truncate path if too long
	displayPath := m.browserPath
	if len(displayPath) > innerW-2 {
		displayPath = "..." + displayPath[len(displayPath)-innerW+5:]
	}
	header := headerStyle.Render("Browse: " + displayPath)

	var rows []string
	if m.browserLoading {
		rows = append(rows, dimStyle.Render("  Loading..."))
	} else if m.browserErr != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
		rows = append(rows, errStyle.Render("  "+m.browserErr))
	} else {
		// Build display list: ".." + entries
		type displayItem struct {
			name  string
			isDir bool
		}
		var items []displayItem
		if m.browserPath != "/" {
			items = append(items, displayItem{name: "..", isDir: true})
		}
		for _, e := range m.browserEntries {
			items = append(items, displayItem{name: e.Name, isDir: e.IsDir})
		}

		if len(items) == 0 {
			rows = append(rows, dimStyle.Render("  (empty directory)"))
		} else {
			end := m.browserScroll + visibleRows
			if end > len(items) {
				end = len(items)
			}
			for i := m.browserScroll; i < end; i++ {
				item := items[i]
				cursor := "  "
				if i == m.browserCursor {
					cursor = "> "
				}

				var line string
				if item.isDir {
					line = cursor + item.name + "/"
				} else {
					line = cursor + item.name
				}
				if len(line) > innerW {
					line = line[:innerW]
				}

				if i == m.browserCursor {
					if item.isDir {
						rows = append(rows, cursorStyle.Render(cursor)+dirStyle.Bold(true).Render(line[2:]))
					} else {
						rows = append(rows, cursorStyle.Render(line))
					}
				} else if item.isDir {
					rows = append(rows, dirStyle.Render(line))
				} else {
					rows = append(rows, line)
				}
			}
		}
	}

	for len(rows) < visibleRows {
		rows = append(rows, "")
	}

	content := strings.Join(rows, "\n")
	helpLine := dimStyle.Render("↑/↓: navigate  Enter: open  Esc: close")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Width(overlayW).
		Padding(0, 1)

	return box.Render(header + "\n" + content + "\n" + helpLine)
}

func (m Model) compositeOverlay(bg, overlay string) string {
	return lipgloss.Place(m.width, m.height-4,
		lipgloss.Center, lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
	)
}

// --- Submission ---

func (m Model) buildSubmission() (slurm.JobSubmission, error) {
	account := m.fields[fAccount].options[m.fields[fAccount].sel]
	partition := m.fields[fPartition].options[m.fields[fPartition].sel]

	script := strings.TrimSpace(m.fields[fScript].input.Value())
	if script == "" {
		return slurm.JobSubmission{}, fmt.Errorf("script path or command is required")
	}

	gpuCount, _ := strconv.Atoi(m.fields[fGPUCount].options[m.fields[fGPUCount].sel])
	cpus, _ := strconv.Atoi(m.fields[fCPUs].options[m.fields[fCPUs].sel])

	gpuType := m.fields[fGPUType].options[m.fields[fGPUType].sel]
	if gpuType == "any" {
		gpuType = ""
	}

	sub := slurm.JobSubmission{
		Account:      account,
		Partition:    partition,
		JobName:      strings.TrimSpace(m.fields[fJobName].input.Value()),
		GPUType:      gpuType,
		GPUs:         gpuCount,
		CPUs:         cpus,
		Memory:       m.fields[fMemory].options[m.fields[fMemory].sel],
		TimeLimit:    m.fields[fTimeLimit].options[m.fields[fTimeLimit].sel],
		OutputPath:   strings.TrimSpace(m.fields[fOutput].input.Value()),
		WorkDir:      strings.TrimSpace(m.fields[fWorkDir].input.Value()),
		NodeList:     joinChosen(m.fields[fInclude].chosen),
		ExcludeNodes: joinChosen(m.fields[fExclude].chosen),
	}

	if strings.HasPrefix(script, "/") || strings.HasPrefix(script, "~") || strings.HasPrefix(script, ".") {
		sub.ScriptPath = script
	} else {
		sub.Command = script
	}

	return sub, nil
}

func joinChosen(chosen map[string]bool) string {
	if len(chosen) == 0 {
		return ""
	}
	names := make([]string, 0, len(chosen))
	for k := range chosen {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}
