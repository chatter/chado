package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/chatter/chado/internal/jj"
	"github.com/chatter/chado/internal/ui"
	"github.com/chatter/chado/internal/ui/help"
)

// ViewMode represents the current view hierarchy
type ViewMode int

const (
	ViewLog   ViewMode = iota // Top level: log view
	ViewFiles                 // Drill down: files in a change
)

// FocusedPane represents which pane has focus
type FocusedPane int

const (
	PaneDiff FocusedPane = iota // [0] Right pane
	PaneLog                     // [1] Left pane
)

// Model is the main application model
type Model struct {
	// Core state
	workDir string
	version string
	keys    KeyMap

	// JJ integration
	runner  *jj.Runner
	watcher *jj.Watcher

	// View state
	viewMode    ViewMode
	focusedPane FocusedPane
	showHelp    bool

	// Panels
	logPanel   ui.LogPanel
	filesPanel ui.FilesPanel
	diffPanel  ui.DiffPanel

	// Help
	statusBar    *help.StatusBar
	floatingHelp *help.FloatingHelp

	// Data
	changes     []jj.Change
	currentDiff string

	// Window size
	width  int
	height int

	// Error state
	lastError string
}

// New creates a new application model
func New(workDir string, version string) Model {
	runner := jj.NewRunner(workDir)

	logPanel := ui.NewLogPanel()
	filesPanel := ui.NewFilesPanel()
	diffPanel := ui.NewDiffPanel()
	statusBar := help.NewStatusBar("chado " + version)
	floatingHelp := help.NewFloatingHelp()

	// Set initial focus - log panel starts focused
	logPanel.SetFocused(true)
	filesPanel.SetFocused(true)
	diffPanel.SetFocused(false)

	return Model{
		workDir:      workDir,
		version:      version,
		keys:         DefaultKeyMap(),
		runner:       runner,
		viewMode:     ViewLog,
		focusedPane:  PaneLog,
		logPanel:     logPanel,
		filesPanel:   filesPanel,
		diffPanel:    diffPanel,
		statusBar:    statusBar,
		floatingHelp: floatingHelp,
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadLog(),
		m.startWatcher(),
	)
}

// loadLog fetches the jj log
func (m Model) loadLog() tea.Cmd {
	return func() tea.Msg {
		output, err := m.runner.Log()
		if err != nil {
			return errMsg{err}
		}
		changes := m.runner.ParseLogLines(output)
		return logLoadedMsg{raw: output, changes: changes}
	}
}

// loadDiff fetches the diff for a change
func (m Model) loadDiff(changeID string) tea.Cmd {
	return func() tea.Msg {
		// Get show output for details
		showOutput, _ := m.runner.Show(changeID)

		// Get diff
		diffOutput, err := m.runner.Diff(changeID)
		if err != nil {
			return errMsg{err}
		}

		return diffLoadedMsg{
			changeID:   changeID,
			showOutput: showOutput,
			diffOutput: diffOutput,
		}
	}
}

// loadFileDiff fetches the diff for a specific file
func (m Model) loadFileDiff(changeID, filePath string) tea.Cmd {
	return func() tea.Msg {
		diffOutput, err := m.runner.DiffFile(changeID, filePath)
		if err != nil {
			return errMsg{err}
		}
		return fileDiffLoadedMsg{diffOutput: diffOutput}
	}
}

// loadFiles parses files from diff output
func (m Model) loadFiles(changeID string) tea.Cmd {
	return func() tea.Msg {
		diffOutput, err := m.runner.Diff(changeID)
		if err != nil {
			return errMsg{err}
		}
		files := m.runner.ParseFiles(diffOutput)
		return filesLoadedMsg{changeID: changeID, files: files, diffOutput: diffOutput}
	}
}

// startWatcher starts the file system watcher
func (m Model) startWatcher() tea.Cmd {
	return func() tea.Msg {
		watcher, err := jj.NewWatcher(m.workDir)
		if err != nil {
			// Don't fail if watcher can't start, just disable auto-refresh
			return watcherStartedMsg{watcher: nil, err: err}
		}
		return watcherStartedMsg{watcher: watcher, err: nil}
	}
}

// waitForChange waits for file system changes
func (m Model) waitForChange() tea.Cmd {
	if m.watcher == nil {
		return nil
	}

	return func() tea.Msg {
		<-m.watcher.Events()               // Block until valid event
		time.Sleep(100 * time.Millisecond) // Debounce
		return jj.WatcherMsg{}
	}
}

// Message types
type logLoadedMsg struct {
	raw     string
	changes []jj.Change
}

type diffLoadedMsg struct {
	changeID   string
	showOutput string
	diffOutput string
}

type fileDiffLoadedMsg struct {
	diffOutput string
}

type filesLoadedMsg struct {
	changeID   string
	files      []jj.File
	diffOutput string
}

type watcherStartedMsg struct {
	watcher *jj.Watcher
	err     error
}

type errMsg struct {
	err error
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// When help modal is open, only handle ? and esc
		if m.showHelp {
			if msg.String() == "?" || msg.String() == "esc" {
				m.showHelp = false
			}
			// Absorb all other keys
			return m, nil
		}

		// Try active bindings first
		if newModel, cmd := dispatchKey(&m, msg, m.activeBindings()); newModel != nil {
			m = *newModel
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		} else {
			// No binding matched, pass to focused panel
			cmds = append(cmds, m.updateFocusedPanel(msg))
		}

	case tea.MouseMsg:
		cmds = append(cmds, m.handleMouse(msg))

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updatePanelSizes()

	case logLoadedMsg:
		m.changes = msg.changes
		m.logPanel.SetContent(msg.raw, msg.changes)
		// Only load diff if we're in log view (not drilled into files)
		if m.viewMode == ViewLog {
			if selected := m.logPanel.SelectedChange(); selected != nil {
				cmds = append(cmds, m.loadDiff(selected.ChangeID))
			}
		}

	case diffLoadedMsg:
		m.currentDiff = msg.diffOutput
		details := ui.ParseDetailsFromShow(msg.showOutput)
		if details.ChangeID == "" {
			details.ChangeID = msg.changeID
		}
		m.diffPanel.SetDetails(details)
		m.diffPanel.SetDiff(msg.diffOutput)

	case filesLoadedMsg:
		m.filesPanel.SetFiles(msg.changeID, msg.files)
		m.currentDiff = msg.diffOutput
		// If there are files, show diff for the first one
		if len(msg.files) > 0 {
			cmds = append(cmds, m.loadFileDiff(msg.changeID, msg.files[0].Path))
		}

	case fileDiffLoadedMsg:
		m.diffPanel.SetShowDetails(false)
		m.diffPanel.SetTitle("Patch")
		m.diffPanel.SetDiff(msg.diffOutput)

	case watcherStartedMsg:
		m.watcher = msg.watcher
		if msg.watcher != nil {
			cmds = append(cmds, m.waitForChange())
		}

	case jj.WatcherMsg:
		// Refresh on file system changes
		cmds = append(cmds, m.loadLog(), m.waitForChange())

		// If drilled into file, reload it
		if m.viewMode == ViewFiles {
			if change := m.filesPanel.ChangeID(); change != "" {
				if file := m.filesPanel.SelectedFile(); file != nil {
					cmds = append(cmds, m.loadFileDiff(change, file.Path))
				}
			}
		}

	case errMsg:
		m.lastError = msg.err.Error()
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleEnter() tea.Cmd {
	switch m.viewMode {
	case ViewLog:
		// Drill into files
		if change := m.logPanel.SelectedChange(); change != nil {
			m.viewMode = ViewFiles
			m.focusedPane = PaneLog
			m.updatePanelFocus()
			return m.loadFiles(change.ChangeID)
		}
	case ViewFiles:
		// Could drill into hunks later, for now just update diff
		if file := m.filesPanel.SelectedFile(); file != nil {
			changeID := m.filesPanel.ChangeID()
			return m.loadFileDiff(changeID, file.Path)
		}
	}
	return nil
}

func (m *Model) handleBack() {
	switch m.viewMode {
	case ViewFiles:
		// Go back to log view
		m.viewMode = ViewLog
		m.diffPanel.SetShowDetails(true)
		m.diffPanel.SetTitle("Diff")
		// Restore full diff for selected change
		if change := m.logPanel.SelectedChange(); change != nil {
			m.diffPanel.SetDiff(m.currentDiff)
		}
	}
}

func (m *Model) updateFocusedPanel(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	// Handle navigation in focused panel
	switch m.focusedPane {
	case PaneLog:
		if m.viewMode == ViewLog {
			cmd = m.logPanel.Update(msg)
			// Update diff when selection changes
			if change := m.logPanel.SelectedChange(); change != nil {
				return tea.Batch(cmd, m.loadDiff(change.ChangeID))
			}
		} else {
			cmd = m.filesPanel.Update(msg)
			// Update diff when file selection changes
			if file := m.filesPanel.SelectedFile(); file != nil {
				changeID := m.filesPanel.ChangeID()
				return tea.Batch(cmd, m.loadFileDiff(changeID, file.Path))
			}
		}
	case PaneDiff:
		cmd = m.diffPanel.Update(msg)
	}

	return cmd
}

func (m *Model) updatePanelFocus() {
	switch m.focusedPane {
	case PaneLog:
		m.logPanel.SetFocused(true)
		m.filesPanel.SetFocused(true)
		m.diffPanel.SetFocused(false)
	case PaneDiff:
		m.logPanel.SetFocused(false)
		m.filesPanel.SetFocused(false)
		m.diffPanel.SetFocused(true)
	}
}

// Action methods for keybindings

func (m *Model) actionQuit() (Model, tea.Cmd) {
	if m.watcher != nil {
		m.watcher.Close()
	}
	return *m, tea.Quit
}

func (m *Model) actionFocusPane0() (Model, tea.Cmd) {
	m.focusedPane = PaneDiff
	m.updatePanelFocus()
	return *m, nil
}

func (m *Model) actionFocusPane1() (Model, tea.Cmd) {
	m.focusedPane = PaneLog
	m.updatePanelFocus()
	return *m, nil
}

func (m *Model) actionNextPane() (Model, tea.Cmd) {
	m.focusedPane = (m.focusedPane + 1) % 2
	m.updatePanelFocus()
	return *m, nil
}

func (m *Model) actionPrevPane() (Model, tea.Cmd) {
	m.focusedPane = (m.focusedPane + 1) % 2
	m.updatePanelFocus()
	return *m, nil
}

func (m *Model) actionEnter() (Model, tea.Cmd) {
	cmd := m.handleEnter()
	return *m, cmd
}

func (m *Model) actionBack() (Model, tea.Cmd) {
	// Only handle Esc when we're in a drilled-down view AND focused on left pane
	if m.viewMode != ViewLog && m.focusedPane == PaneLog {
		m.handleBack()
	}
	return *m, nil
}

func (m *Model) actionToggleHelp() (Model, tea.Cmd) {
	m.showHelp = !m.showHelp
	return *m, nil
}

// activeBindings returns all currently active keybindings for dispatch.
// Merges global bindings with context-specific panel bindings.
func (m *Model) activeBindings() []ActionBinding {
	return m.globalBindings()
	// Note: Panel bindings (j/k, g/G) are handled by updateFocusedPanel()
	// They don't need to be in activeBindings() for dispatch since they're
	// not ActionBindings - they're handled directly by the panels.
}

// activeHelpBindings returns all display bindings for the current context.
// Used by the status bar to show context-sensitive help.
func (m *Model) activeHelpBindings() []help.HelpBinding {
	// Start with global bindings
	bindings := ToHelpBindings(m.globalBindings())

	// Add panel-specific bindings based on focus
	switch m.focusedPane {
	case PaneLog:
		if m.viewMode == ViewLog {
			bindings = append(bindings, m.logPanel.HelpBindings()...)
		} else {
			bindings = append(bindings, m.filesPanel.HelpBindings()...)
		}
	case PaneDiff:
		bindings = append(bindings, m.diffPanel.HelpBindings()...)
	}

	return bindings
}

// globalBindings returns the app-level keybindings with their actions.
func (m *Model) globalBindings() []ActionBinding {
	return []ActionBinding{
		// Quit - highest order (always visible)
		{
			HelpBinding: help.HelpBinding{
				Binding:  m.keys.Quit,
				Category: help.CategoryActions,
				Order:    100,
			},
			Action: (*Model).actionQuit,
		},
		// Pane focus
		{
			HelpBinding: help.HelpBinding{
				Binding:  m.keys.FocusPane0,
				Category: help.CategoryNavigation,
				Order:    50,
			},
			Action: (*Model).actionFocusPane0,
		},
		{
			HelpBinding: help.HelpBinding{
				Binding:  m.keys.FocusPane1,
				Category: help.CategoryNavigation,
				Order:    51,
			},
			Action: (*Model).actionFocusPane1,
		},
		{
			HelpBinding: help.HelpBinding{
				Binding:  m.keys.NextPane,
				Category: help.CategoryNavigation,
				Order:    20,
			},
			Action: (*Model).actionNextPane,
		},
		{
			HelpBinding: help.HelpBinding{
				Binding:  m.keys.PrevPane,
				Category: help.CategoryNavigation,
				Order:    21,
			},
			Action: (*Model).actionPrevPane,
		},
		{
			HelpBinding: help.HelpBinding{
				Binding:  m.keys.Right,
				Category: help.CategoryNavigation,
				Order:    22,
			},
			Action: (*Model).actionNextPane,
		},
		{
			HelpBinding: help.HelpBinding{
				Binding:  m.keys.Left,
				Category: help.CategoryNavigation,
				Order:    23,
			},
			Action: (*Model).actionPrevPane,
		},
		// Actions
		{
			HelpBinding: help.HelpBinding{
				Binding:  m.keys.Enter,
				Category: help.CategoryActions,
				Order:    10,
			},
			Action: (*Model).actionEnter,
		},
		{
			HelpBinding: help.HelpBinding{
				Binding:  m.keys.Back,
				Category: help.CategoryActions,
				Order:    11,
			},
			Action: (*Model).actionBack,
		},
		// Help toggle - pinned, always visible
		{
			HelpBinding: help.HelpBinding{
				Binding:  m.keys.Help,
				Category: help.CategoryActions,
				Order:    99,
				Pinned:   true, // Always visible in status bar
			},
			Action: (*Model).actionToggleHelp,
		},
	}
}

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	// Get left panel width from rendered content
	var leftWidth int
	if m.viewMode == ViewLog {
		leftWidth = lipgloss.Width(m.logPanel.View())
	} else {
		leftWidth = lipgloss.Width(m.filesPanel.View())
	}
	// Panel content starts after border (1) and title line (1)
	contentYOffset := 2

	// Determine which panel was interacted with
	inLeftPanel := msg.X < leftWidth
	inRightPanel := msg.X >= leftWidth

	// Handle scroll events (wheel)
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		if inRightPanel {
			m.diffPanel.HandleMouseScroll(msg.Button)
		}
		return nil
	}

	// Handle click events
	if msg.Button == tea.MouseButtonLeft {
		// Y relative to content area
		contentY := msg.Y - contentYOffset

		if inLeftPanel {
			// Focus left panel
			m.focusedPane = PaneLog
			m.updatePanelFocus()

			// Dispatch to appropriate panel, only reload if selection changed
			if m.viewMode == ViewLog {
				if m.logPanel.HandleClick(contentY) {
					if change := m.logPanel.SelectedChange(); change != nil {
						return m.loadDiff(change.ChangeID)
					}
				}
			} else {
				if m.filesPanel.HandleClick(contentY) {
					if file := m.filesPanel.SelectedFile(); file != nil {
						changeID := m.filesPanel.ChangeID()
						return m.loadFileDiff(changeID, file.Path)
					}
				}
			}
		} else if inRightPanel {
			// Focus right panel
			m.focusedPane = PaneDiff
			m.updatePanelFocus()
		}
	}

	return nil
}

func (m *Model) updatePanelSizes() {
	// Leave room for status bar
	contentHeight := m.height - 1

	// Split horizontally: left panel ~40%, right panel ~60%
	leftWidth := m.width * 40 / 100
	rightWidth := m.width - leftWidth

	m.logPanel.SetSize(leftWidth, contentHeight)
	m.filesPanel.SetSize(leftWidth, contentHeight)
	m.diffPanel.SetSize(rightWidth, contentHeight)
}

// View renders the application
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Render left panel (log or files)
	var leftPanel string
	switch m.viewMode {
	case ViewLog:
		leftPanel = m.logPanel.View()
	case ViewFiles:
		leftPanel = m.filesPanel.View()
	}

	// Render right panel (diff)
	rightPanel := m.diffPanel.View()

	// Join panels horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Status bar
	statusBar := m.renderStatusBar()

	// Join vertically
	base := lipgloss.JoinVertical(lipgloss.Left, panels, statusBar)

	// Show floating help modal if active
	if m.showHelp {
		return m.renderWithOverlay(base)
	}

	return base
}

func (m Model) renderWithOverlay(base string) string {
	// Calculate modal size (centered, ~80% of screen)
	modalWidth := m.width * 80 / 100
	modalHeight := m.height * 70 / 100

	if modalWidth < 40 {
		modalWidth = min(40, m.width-4)
	}
	if modalHeight < 10 {
		modalHeight = min(10, m.height-4)
	}

	// Set up and render floating help
	m.floatingHelp.SetSize(modalWidth, modalHeight)
	m.floatingHelp.SetBindings(m.activeHelpBindings())
	modal := m.floatingHelp.View()

	// Center the modal on the base content
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

func (m Model) renderStatusBar() string {
	m.statusBar.SetWidth(m.width)
	m.statusBar.SetBindings(m.activeHelpBindings())
	return ui.StatusBarStyle.Render(m.statusBar.View())
}
