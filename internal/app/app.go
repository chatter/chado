package app

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/chatter/lazyjj/internal/jj"
	"github.com/chatter/lazyjj/internal/ui"
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

	// Panels
	logPanel   ui.LogPanel
	filesPanel ui.FilesPanel
	diffPanel  ui.DiffPanel

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

	// Set initial focus - log panel starts focused
	logPanel.SetFocused(true)
	filesPanel.SetFocused(true)
	diffPanel.SetFocused(false)

	return Model{
		workDir:     workDir,
		version:     version,
		keys:        DefaultKeyMap(),
		runner:      runner,
		viewMode:    ViewLog,
		focusedPane: PaneLog,
		logPanel:    logPanel,
		filesPanel:  filesPanel,
		diffPanel:   diffPanel,
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
		// Global keys
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.watcher != nil {
				m.watcher.Close()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keys.FocusPane0):
			m.focusedPane = PaneDiff
			m.updatePanelFocus()

		case key.Matches(msg, m.keys.FocusPane1):
			m.focusedPane = PaneLog
			m.updatePanelFocus()

		case key.Matches(msg, m.keys.NextPane), key.Matches(msg, m.keys.Right):
			m.focusedPane = (m.focusedPane + 1) % 2
			m.updatePanelFocus()

		case key.Matches(msg, m.keys.PrevPane), key.Matches(msg, m.keys.Left):
			m.focusedPane = (m.focusedPane + 1) % 2
			m.updatePanelFocus()

		case key.Matches(msg, m.keys.Enter):
			cmds = append(cmds, m.handleEnter())

		case key.Matches(msg, m.keys.Back):
			// Only handle Esc when we're in a drilled-down view AND focused on left pane
			if m.viewMode != ViewLog && m.focusedPane == PaneLog {
				m.handleBack()
			}
			// Otherwise, Esc does nothing (or could pass to focused panel later)

		default:
			// Pass to focused panel
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
		if m.viewMode == ViewLog && len(msg.changes) > 0 {
			// Load diff for currently selected change (or first if none selected)
			if selected := m.logPanel.SelectedChange(); selected != nil {
				cmds = append(cmds, m.loadDiff(selected.ChangeID))
			} else {
				cmds = append(cmds, m.loadDiff(msg.changes[0].ChangeID))
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

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	// Calculate panel boundaries
	leftWidth := m.width * 40 / 100
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

			// Dispatch to appropriate panel
			if m.viewMode == ViewLog {
				m.logPanel.HandleClick(contentY)
				// Load diff for new selection
				if change := m.logPanel.SelectedChange(); change != nil {
					return m.loadDiff(change.ChangeID)
				}
			} else {
				m.filesPanel.HandleClick(contentY)
				// Load file diff for new selection
				if file := m.filesPanel.SelectedFile(); file != nil {
					changeID := m.filesPanel.ChangeID()
					return m.loadFileDiff(changeID, file.Path)
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
	return lipgloss.JoinVertical(lipgloss.Left, panels, statusBar)
}

func (m Model) renderStatusBar() string {
	// For now, just version on the right
	version := "lazyjj v" + m.version

	// Pad to full width
	padding := max(m.width-len(version), 0)

	return ui.StatusBarStyle.Render(strings.Repeat(" ", padding) + version)
}
