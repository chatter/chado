package app

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/chatter/chado/internal/jj"
	"github.com/chatter/chado/internal/logger"
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
	PaneDiff  FocusedPane = iota // [0] Right pane
	PaneLog                      // [1] Left pane - log
	PaneOpLog                    // [2] Left pane - op log
)

const (
	// watcherDebounceDelay is the pause before flushing batched file-watcher events.
	watcherDebounceDelay = 300 * time.Millisecond

	// paneCount is the total number of navigable panes.
	paneCount = 3

	// borderAnimTickInterval is the frame interval for the focus border animation.
	borderAnimTickInterval = 15 * time.Millisecond

	// describeOverlayWidth is the fixed width of the describe-input overlay.
	describeOverlayWidth = 60

	// describeOverlayHeight is the fixed height of the describe-input overlay.
	describeOverlayHeight = 10

	// Help binding display order values (lower = shown first in status bar).
	orderSelect     = 10
	orderBack       = 11
	orderDescribe   = 12
	orderEdit       = 13
	orderNew        = 14
	orderAbandon    = 15
	orderNextPane   = 20
	orderPrevPane   = 21
	orderFocusPane0 = 50
	orderFocusPane1 = 51
	orderFocusPane2 = 52
	orderHelp       = 99
	orderQuit       = 100

	// percentDivisor converts a percentage numerator to a fraction.
	percentDivisor = 100

	// centerDivisor halves a dimension to find the center point.
	centerDivisor = 2

	// leftPanelWidthPct is the left panel's share of screen width.
	leftPanelWidthPct = 40

	// leftPanelSplitDivisor divides the left panel vertically into equal halves.
	leftPanelSplitDivisor = 2

	// modalWidthPct and modalHeightPct control the help modal's screen share.
	modalWidthPct  = 80
	modalHeightPct = 70

	// minModalWidth and minModalHeight are floor sizes for the help modal.
	minModalWidth  = 40
	minModalHeight = 10

	// modalEdgePadding is the gap kept between the modal and the screen edge.
	modalEdgePadding = 4

	// statusBarHeight is the vertical space reserved for the status bar.
	statusBarHeight = 1

	// contentYOffset accounts for border (1) + title line (1) in a panel.
	contentYOffset = 2
)

// Model is the main application model
type Model struct {
	// Core state
	workDir string
	version string
	keys    KeyMap
	log     *logger.Logger

	// JJ integration
	runner  *jj.Runner
	watcher *jj.Watcher

	// View state
	viewMode      ViewMode
	focusedPane   FocusedPane
	showHelp      bool
	editMode      bool
	describeInput *ui.DescribeInput

	// Panels
	logPanel   ui.LogPanel
	opLogPanel ui.OpLogPanel
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

	// Focus border animation (one wrap when any panel is focused)
	logPanelBorderPhase  float64
	borderAnimGeneration int // incremented on each focus change so stale ticks are ignored

	// Watcher coalescing: one refresh per burst of file-system events
	watcherPending bool // true while a watcherFlushMsg tick is in flight
}

// borderAnimTickMsg is sent each frame during the focus border wrap animation.
type borderAnimTickMsg struct {
	Phase      float64
	Generation int // must match Model.borderAnimGeneration or tick is ignored (stale)
}

// New creates a new application model
func New(workDir string, version string, log *logger.Logger) Model {
	runner := jj.NewRunner(workDir, log)

	logPanel := ui.NewLogPanel()
	opLogPanel := ui.NewOpLogPanel()
	filesPanel := ui.NewFilesPanel()
	diffPanel := ui.NewDiffPanel()
	statusBar := help.NewStatusBar("chado " + version)
	floatingHelp := help.NewFloatingHelp()
	describeInput := ui.NewDescribeInput()

	// Set initial focus - log panel starts focused
	logPanel.SetFocused(true)
	opLogPanel.SetFocused(false)
	filesPanel.SetFocused(true)
	diffPanel.SetFocused(false)

	return Model{
		workDir:       workDir,
		version:       version,
		keys:          DefaultKeyMap(),
		log:           log,
		runner:        runner,
		viewMode:      ViewLog,
		focusedPane:   PaneLog,
		logPanel:      logPanel,
		opLogPanel:    opLogPanel,
		filesPanel:    filesPanel,
		diffPanel:     diffPanel,
		statusBar:     statusBar,
		floatingHelp:  floatingHelp,
		describeInput: describeInput,
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	m.log.Info("initializing app", "workdir", m.workDir, "version", m.version)

	return tea.Batch(
		m.loadLog(),
		m.loadOpLog(),
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

		// Get the shortest unique prefix for coloring
		shortCode, _ := m.runner.ShortestChangeID(changeID)
		if shortCode == "" {
			shortCode = changeID // Fallback to full ID if call fails
		}

		files := m.runner.ParseFiles(diffOutput)

		return filesLoadedMsg{changeID: changeID, shortCode: shortCode, files: files, diffOutput: diffOutput}
	}
}

// loadOpLog fetches the jj operation log
func (m Model) loadOpLog() tea.Cmd {
	return func() tea.Msg {
		output, err := m.runner.OpLog()
		if err != nil {
			return errMsg{err}
		}

		operations := m.runner.ParseOpLogLines(output)

		return opLogLoadedMsg{raw: output, operations: operations}
	}
}

// loadOpShow fetches details for a specific operation
func (m Model) loadOpShow(opID string) tea.Cmd {
	return func() tea.Msg {
		output, err := m.runner.OpShow(opID)
		if err != nil {
			return errMsg{err}
		}

		return opShowLoadedMsg{opID: opID, output: output}
	}
}

// loadEvoLog fetches the evolution log for a specific change
func (m Model) loadEvoLog(changeID, shortCode string) tea.Cmd {
	return func() tea.Msg {
		output, err := m.runner.EvoLog(changeID)
		if err != nil {
			return errMsg{err}
		}

		operations := m.runner.ParseOpLogLines(output)

		return evoLogLoadedMsg{
			changeID:   changeID,
			shortCode:  shortCode,
			raw:        output,
			operations: operations,
		}
	}
}

// startWatcher starts the file system watcher
func (m Model) startWatcher() tea.Cmd {
	return func() tea.Msg {
		watcher, err := jj.NewWatcher(m.workDir, m.log)
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
		<-m.watcher.Events() // Block until valid event
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
	shortCode  string
	files      []jj.File
	diffOutput string
}

type opLogLoadedMsg struct {
	raw        string
	operations []jj.Operation
}

type evoLogLoadedMsg struct {
	changeID   string
	shortCode  string
	raw        string
	operations []jj.Operation
}

type opShowLoadedMsg struct {
	opID   string
	output string
}

type watcherStartedMsg struct {
	watcher *jj.Watcher
	err     error
}

// watcherFlushMsg fires after the coalescing delay; triggers one refresh.
type watcherFlushMsg struct{}

type errMsg struct {
	err error
}

type describeCompleteMsg struct {
	changeID string
}

type editCompleteMsg struct {
	changeID string
}

type newCompleteMsg struct{}

type abandonCompleteMsg struct {
	changeID string
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// When edit mode is active, forward to describe input
		if m.editMode {
			cmd := m.describeInput.Update(msg)
			return m, cmd
		}

		// When help modal is open, only handle ?, esc, and q
		if m.showHelp {
			if msg.String() == "?" || msg.String() == "esc" {
				m.showHelp = false
				return m, nil
			}

			if msg.String() == "q" {
				if m.watcher != nil {
					m.watcher.Close()
				}

				return m, tea.Quit
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
		// Only load diff if we're in log view AND log panel is focused
		if m.viewMode == ViewLog && m.focusedPane == PaneLog {
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
		m.filesPanel.SetFiles(msg.changeID, msg.shortCode, msg.files)
		m.currentDiff = msg.diffOutput
		// Load evolog for this change (shows operations that affected it)
		cmds = append(cmds, m.loadEvoLog(msg.changeID, msg.shortCode))
		// If there are files, show diff for the first one
		if len(msg.files) > 0 {
			cmds = append(cmds, m.loadFileDiff(msg.changeID, msg.files[0].Path))
		}

	case fileDiffLoadedMsg:
		m.diffPanel.SetShowDetails(false)
		m.diffPanel.SetTitle("Patch")
		m.diffPanel.SetDiff(msg.diffOutput)

	case opLogLoadedMsg:
		m.opLogPanel.SetOpLogContent(msg.raw, msg.operations)
		// If op log panel is focused, load op show for selected operation
		if m.focusedPane == PaneOpLog {
			if selected := m.opLogPanel.SelectedOperation(); selected != nil {
				cmds = append(cmds, m.loadOpShow(selected.OpID))
			}
		}

	case evoLogLoadedMsg:
		m.opLogPanel.SetEvoLogContent(msg.changeID, msg.shortCode, msg.raw, msg.operations)
		// If op log panel is focused, load op show for selected operation
		if m.focusedPane == PaneOpLog {
			if selected := m.opLogPanel.SelectedOperation(); selected != nil {
				cmds = append(cmds, m.loadOpShow(selected.OpID))
			}
		}

	case opShowLoadedMsg:
		m.diffPanel.SetShowDetails(false)
		m.diffPanel.SetTitle("Operation")
		m.diffPanel.SetDiff(msg.output)

	case watcherStartedMsg:
		if msg.err != nil {
			m.log.Warn("watcher failed to start", "err", msg.err)
		}

		if m.watcher = msg.watcher; m.watcher != nil {
			cmds = append(cmds, m.waitForChange())
		}

	case jj.WatcherMsg:
		// Coalesce: schedule a single flush after a short delay.
		// Do NOT refresh or re-arm waitForChange here.
		if !m.watcherPending {
			m.watcherPending = true

			cmds = append(cmds, tea.Tick(watcherDebounceDelay, func(time.Time) tea.Msg {
				return watcherFlushMsg{}
			}))
		}

	case watcherFlushMsg:
		// One refresh per burst, then re-arm the watcher.
		m.watcherPending = false
		cmds = append(cmds, m.loadLog(), m.loadOpLog(), m.waitForChange())

		// If drilled into files view, reload file list and current diff
		if m.viewMode == ViewFiles {
			if change := m.filesPanel.ChangeID(); change != "" {
				cmds = append(cmds, m.loadFiles(change))
				if file := m.filesPanel.SelectedFile(); file != nil {
					cmds = append(cmds, m.loadFileDiff(change, file.Path))
				}
			}
		}

	case errMsg:
		m.log.Error("app error", "err", msg.err)
		m.lastError = msg.err.Error()

	case ui.DescribeSubmitMsg:
		// Run jj describe and reload
		m.editMode = false
		cmds = append(cmds, m.runDescribe(msg.ChangeID, msg.Description))

	case ui.DescribeCancelMsg:
		// Just close the edit mode
		m.editMode = false

	case describeCompleteMsg:
		// Description updated, reload the log
		cmds = append(cmds, m.loadLog(), m.loadOpLog())

	case editCompleteMsg:
		// Edit complete, reload the log
		cmds = append(cmds, m.loadLog(), m.loadOpLog())

	case newCompleteMsg:
		// New change created, reload the log
		cmds = append(cmds, m.loadLog(), m.loadOpLog())

	case abandonCompleteMsg:
		// Change abandoned, reload the log
		cmds = append(cmds, m.loadLog(), m.loadOpLog())

	case borderAnimTickMsg:
		if msg.Generation != m.borderAnimGeneration {
			break // stale tick from a previous focus; ignore
		}

		const animSteps = 120

		nextPhase := msg.Phase + 1.0/animSteps
		if nextPhase > 1 {
			nextPhase = 1
		}

		m.logPanelBorderPhase = nextPhase
		m.setFocusBorderAnimPhase(nextPhase)

		if nextPhase >= 1 {
			m.setFocusBorderAnimating(false) // animation complete; show static focus border
		}

		if nextPhase < 1 {
			cmds = append(cmds, m.startLogPanelBorderAnimWithPhase(nextPhase, m.borderAnimGeneration))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleEnter() tea.Cmd {
	switch m.viewMode {
	case ViewLog:
		// Drill into files
		if change := m.logPanel.SelectedChange(); change != nil {
			m.log.Debug("drilling into files view", "change_id", change.ChangeID)
			m.viewMode = ViewFiles
			m.focusedPane = PaneLog
			m.updatePanelFocus() // files now visible in left slot; focused, not animated

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

func (m *Model) handleBack() tea.Cmd {
	switch m.viewMode {
	case ViewFiles:
		// Go back to log view
		m.viewMode = ViewLog
		m.updatePanelFocus() // log now visible in left slot; focused, not animated
		m.diffPanel.SetShowDetails(true)
		m.diffPanel.SetTitle("Diff")
		// Restore full diff for selected change
		if change := m.logPanel.SelectedChange(); change != nil {
			m.diffPanel.SetDiff(m.currentDiff)
		}
		// Restore global op log (switch back from evolog mode)
		return m.loadOpLog()
	}

	return nil
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
	case PaneOpLog:
		cmd = m.opLogPanel.Update(msg)
		// Update diff pane with op show when selection changes
		if op := m.opLogPanel.SelectedOperation(); op != nil {
			return tea.Batch(cmd, m.loadOpShow(op.OpID))
		}
	case PaneDiff:
		cmd = m.diffPanel.Update(msg)
	}

	return cmd
}

func (m *Model) updatePanelFocus() {
	// Only the panel visible in the left slot gets focused when PaneLog is active
	m.logPanel.SetFocused(m.focusedPane == PaneLog && m.viewMode == ViewLog)
	m.filesPanel.SetFocused(m.focusedPane == PaneLog && m.viewMode == ViewFiles)
	m.opLogPanel.SetFocused(m.focusedPane == PaneOpLog)
	m.diffPanel.SetFocused(m.focusedPane == PaneDiff)
	// Clear animating so focus-without-animation (e.g. back from files) shows static border
	m.logPanel.SetBorderAnimating(false)
	m.filesPanel.SetBorderAnimating(false)
	m.diffPanel.SetBorderAnimating(false)
	m.opLogPanel.SetBorderAnimating(false)
}

// Action methods for keybindings

func (m *Model) actionQuit() (Model, tea.Cmd) {
	if m.watcher != nil {
		m.watcher.Close()
	}

	return *m, tea.Quit
}

func (m *Model) actionFocusPane0() (Model, tea.Cmd) {
	prevPane := m.focusedPane
	m.focusedPane = PaneDiff
	m.updatePanelFocus()

	return *m, tea.Batch(m.handleFocusChange(prevPane, m.focusedPane), m.startLogPanelBorderAnim())
}

func (m *Model) actionFocusPane1() (Model, tea.Cmd) {
	prevPane := m.focusedPane
	m.focusedPane = PaneLog
	m.updatePanelFocus()
	cmds := []tea.Cmd{m.handleFocusChange(prevPane, m.focusedPane), m.startLogPanelBorderAnim()}

	return *m, tea.Batch(cmds...)
}

func (m *Model) actionFocusPane2() (Model, tea.Cmd) {
	prevPane := m.focusedPane
	m.focusedPane = PaneOpLog
	m.updatePanelFocus()

	return *m, tea.Batch(m.handleFocusChange(prevPane, m.focusedPane), m.startLogPanelBorderAnim())
}

func (m *Model) actionNextPane() (Model, tea.Cmd) {
	prevPane := m.focusedPane
	m.focusedPane = (m.focusedPane + 1) % paneCount
	m.updatePanelFocus()
	cmds := []tea.Cmd{m.handleFocusChange(prevPane, m.focusedPane), m.startLogPanelBorderAnim()}

	return *m, tea.Batch(cmds...)
}

func (m *Model) actionPrevPane() (Model, tea.Cmd) {
	prevPane := m.focusedPane
	m.focusedPane = (m.focusedPane + paneCount - 1) % paneCount
	m.updatePanelFocus()
	cmds := []tea.Cmd{m.handleFocusChange(prevPane, m.focusedPane), m.startLogPanelBorderAnim()}

	return *m, tea.Batch(cmds...)
}

// startLogPanelBorderAnim starts the one-shot border wrap animation for the focused panel.
func (m *Model) startLogPanelBorderAnim() tea.Cmd {
	m.borderAnimGeneration++
	m.logPanelBorderPhase = 0
	m.setFocusBorderAnimPhase(0)
	m.setFocusBorderAnimating(true) // only explicit focus (key/mouse) runs the animation

	return m.startLogPanelBorderAnimWithPhase(0, m.borderAnimGeneration)
}

// setFocusBorderAnimPhase sets the border anim phase on whichever panel currently has focus.
func (m *Model) setFocusBorderAnimPhase(phase float64) {
	switch m.focusedPane {
	case PaneLog:
		if m.viewMode == ViewLog {
			m.logPanel.SetBorderAnimPhase(phase)
		} else {
			m.filesPanel.SetBorderAnimPhase(phase)
		}
	case PaneDiff:
		m.diffPanel.SetBorderAnimPhase(phase)
	case PaneOpLog:
		m.opLogPanel.SetBorderAnimPhase(phase)
	}
}

// setFocusBorderAnimating sets the border animating flag on whichever panel currently has focus.
func (m *Model) setFocusBorderAnimating(animating bool) {
	switch m.focusedPane {
	case PaneLog:
		if m.viewMode == ViewLog {
			m.logPanel.SetBorderAnimating(animating)
		} else {
			m.filesPanel.SetBorderAnimating(animating)
		}
	case PaneDiff:
		m.diffPanel.SetBorderAnimating(animating)
	case PaneOpLog:
		m.opLogPanel.SetBorderAnimating(animating)
	}
}

// startLogPanelBorderAnimWithPhase schedules the next tick with phase and generation.
func (m *Model) startLogPanelBorderAnimWithPhase(phase float64, generation int) tea.Cmd {
	return tea.Tick(borderAnimTickInterval, func(_ time.Time) tea.Msg {
		return borderAnimTickMsg{Phase: phase, Generation: generation}
	})
}

// handleFocusChange loads appropriate content when focus changes between panes
func (m *Model) handleFocusChange(from, to FocusedPane) tea.Cmd {
	if from == to {
		return nil
	}

	// When focusing op log, show op details in diff pane
	if to == PaneOpLog {
		if op := m.opLogPanel.SelectedOperation(); op != nil {
			return m.loadOpShow(op.OpID)
		}
	}

	// When focusing log (from op log), show change diff in diff pane
	if to == PaneLog && from == PaneOpLog {
		if m.viewMode == ViewLog {
			if change := m.logPanel.SelectedChange(); change != nil {
				return m.loadDiff(change.ChangeID)
			}
		} else {
			if file := m.filesPanel.SelectedFile(); file != nil {
				return m.loadFileDiff(m.filesPanel.ChangeID(), file.Path)
			}
		}
	}

	return nil
}

func (m *Model) actionEnter() (Model, tea.Cmd) {
	cmd := m.handleEnter()
	return *m, cmd
}

func (m *Model) actionBack() (Model, tea.Cmd) {
	// Only handle Esc when we're in a drilled-down view AND focused on left pane
	if m.viewMode != ViewLog && m.focusedPane == PaneLog {
		cmd := m.handleBack()
		return *m, cmd
	}

	return *m, nil
}

func (m *Model) actionToggleHelp() (Model, tea.Cmd) {
	m.showHelp = !m.showHelp
	return *m, nil
}

func (m *Model) actionDescribe() (Model, tea.Cmd) {
	// Only allow describe when log panel is focused and in log view
	if m.focusedPane != PaneLog || m.viewMode != ViewLog {
		return *m, nil
	}

	selected := m.logPanel.SelectedChange()
	if selected == nil {
		return *m, nil
	}

	// Initialize describe input with current description
	m.describeInput.SetChangeID(selected.ChangeID)
	// If no real description, leave empty so placeholder shows and typing replaces
	desc := selected.Description
	if desc == "" || desc == "(no description set)" {
		desc = ""
	}

	m.describeInput.SetValue(desc)
	m.describeInput.SetSize(describeOverlayWidth, describeOverlayHeight)
	m.editMode = true

	return *m, m.describeInput.Focus()
}

// runDescribe executes jj describe and returns a completion message
func (m *Model) runDescribe(changeID, message string) tea.Cmd {
	return func() tea.Msg {
		if err := m.runner.Describe(changeID, message); err != nil {
			return errMsg{err}
		}

		return describeCompleteMsg{changeID: changeID}
	}
}

func (m *Model) actionEdit() (Model, tea.Cmd) {
	// Only allow edit when log panel is focused and in log view
	if m.focusedPane != PaneLog || m.viewMode != ViewLog {
		return *m, nil
	}

	selected := m.logPanel.SelectedChange()
	if selected == nil {
		return *m, nil
	}

	return *m, m.runEdit(selected.ChangeID)
}

// runEdit executes jj edit and returns a completion message
func (m *Model) runEdit(changeID string) tea.Cmd {
	return func() tea.Msg {
		if err := m.runner.Edit(changeID); err != nil {
			return errMsg{err}
		}

		return editCompleteMsg{changeID: changeID}
	}
}

func (m *Model) actionNew() (Model, tea.Cmd) {
	// New creates an empty change on top of current working copy
	// Works from any context
	return *m, m.runNew()
}

// runNew executes jj new and returns a completion message
func (m *Model) runNew() tea.Cmd {
	return func() tea.Msg {
		if err := m.runner.New(); err != nil {
			return errMsg{err}
		}

		return newCompleteMsg{}
	}
}

func (m *Model) actionAbandon() (Model, tea.Cmd) {
	// Only allow abandon when log panel is focused and in log view
	if m.focusedPane != PaneLog || m.viewMode != ViewLog {
		return *m, nil
	}

	selected := m.logPanel.SelectedChange()
	if selected == nil {
		return *m, nil
	}

	return *m, m.runAbandon(selected.ChangeID)
}

// runAbandon executes jj abandon and returns a completion message
func (m *Model) runAbandon(changeID string) tea.Cmd {
	return func() tea.Msg {
		err := m.runner.Abandon(changeID)
		if err != nil {
			return errMsg{err}
		}

		return abandonCompleteMsg{changeID: changeID}
	}
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
func (m *Model) activeHelpBindings() []help.Binding {
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
	case PaneOpLog:
		bindings = append(bindings, m.opLogPanel.HelpBindings()...)
	case PaneDiff:
		bindings = append(bindings, m.diffPanel.HelpBindings()...)
	}

	return bindings
}

// globalBindings returns the app-level keybindings with their actions.
func (m *Model) globalBindings() []ActionBinding {
	return []ActionBinding{
		// Quit - pinned, always visible
		{
			Binding: help.Binding{
				Key:      m.keys.Quit,
				Category: help.CategoryActions,
				Order:    orderQuit,
				Pinned:   true,
			},
			Action: (*Model).actionQuit,
		},
		// Pane focus - "#" represents 0/1 (deduped by description)
		{
			Binding: help.Binding{
				Key:      m.keys.FocusPane0,
				Category: help.CategoryNavigation,
				Order:    orderFocusPane0,
			},
			Action: (*Model).actionFocusPane0,
		},
		{
			Binding: help.Binding{
				Key:      m.keys.FocusPane1,
				Category: help.CategoryNavigation,
				Order:    orderFocusPane1,
			},
			Action: (*Model).actionFocusPane1,
		},
		{
			Binding: help.Binding{
				Key:      m.keys.FocusPane2,
				Category: help.CategoryNavigation,
				Order:    orderFocusPane2,
			},
			Action: (*Model).actionFocusPane2,
		},
		// Next/prev pane - combined keys
		{
			Binding: help.Binding{
				Key:      m.keys.NextPane,
				Category: help.CategoryNavigation,
				Order:    orderNextPane,
			},
			Action: (*Model).actionNextPane,
		},
		{
			Binding: help.Binding{
				Key:      m.keys.PrevPane,
				Category: help.CategoryNavigation,
				Order:    orderPrevPane,
			},
			Action: (*Model).actionPrevPane,
		},
		// Actions
		{
			Binding: help.Binding{
				Key:      m.keys.Enter,
				Category: help.CategoryActions,
				Order:    orderSelect,
			},
			Action: (*Model).actionEnter,
		},
		{
			Binding: help.Binding{
				Key:      m.keys.Back,
				Category: help.CategoryActions,
				Order:    orderBack,
			},
			Action: (*Model).actionBack,
		},
		{
			Binding: help.Binding{
				Key:      m.keys.Describe,
				Category: help.CategoryActions,
				Order:    orderDescribe,
			},
			Action: (*Model).actionDescribe,
		},
		{
			Binding: help.Binding{
				Key:      m.keys.Edit,
				Category: help.CategoryActions,
				Order:    orderEdit,
			},
			Action: (*Model).actionEdit,
		},
		{
			Binding: help.Binding{
				Key:      m.keys.New,
				Category: help.CategoryActions,
				Order:    orderNew,
			},
			Action: (*Model).actionNew,
		},
		{
			Binding: help.Binding{
				Key:      m.keys.Abandon,
				Category: help.CategoryActions,
				Order:    orderAbandon,
			},
			Action: (*Model).actionAbandon,
		},
		// Help toggle - pinned, always visible
		{
			Binding: help.Binding{
				Key:      m.keys.Help,
				Category: help.CategoryActions,
				Order:    orderHelp,
				Pinned:   true,
			},
			Action: (*Model).actionToggleHelp,
		},
	}
}

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	// Get the underlying mouse event
	mouse := msg.Mouse()

	// Get left panel width from rendered content
	var leftWidth int
	if m.viewMode == ViewLog {
		leftWidth = lipgloss.Width(m.logPanel.View())
	} else {
		leftWidth = lipgloss.Width(m.filesPanel.View())
	}

	// Calculate panel heights for vertical split
	contentHeight := m.height - statusBarHeight
	leftTopHeight := contentHeight / leftPanelSplitDivisor

	// Panel content starts after border (1) and title line (1)

	// Determine which panel was interacted with
	inLeftPanel := mouse.X < leftWidth
	inRightPanel := mouse.X >= leftWidth
	inTopLeftPanel := inLeftPanel && mouse.Y < leftTopHeight
	inBottomLeftPanel := inLeftPanel && mouse.Y >= leftTopHeight

	// Handle scroll events (wheel)
	if mouse.Button == tea.MouseWheelUp || mouse.Button == tea.MouseWheelDown {
		if inRightPanel {
			m.diffPanel.HandleMouseScroll(mouse.Button)
		}

		return nil
	}

	// Handle click events
	if mouse.Button == tea.MouseLeft {
		if inTopLeftPanel {
			// Y relative to top panel content area
			contentY := mouse.Y - contentYOffset

			// Focus top-left panel
			m.focusedPane = PaneLog
			m.updatePanelFocus()

			if m.viewMode == ViewLog {
				var cmd tea.Cmd

				if m.logPanel.HandleClick(contentY) {
					if change := m.logPanel.SelectedChange(); change != nil {
						cmd = m.loadDiff(change.ChangeID)
					}
				}

				return tea.Batch(cmd, m.startLogPanelBorderAnim())
			}

			if m.filesPanel.HandleClick(contentY) {
				if file := m.filesPanel.SelectedFile(); file != nil {
					changeID := m.filesPanel.ChangeID()

					return tea.Batch(m.loadFileDiff(changeID, file.Path), m.startLogPanelBorderAnim())
				}
			}

			return m.startLogPanelBorderAnim()
		} else if inBottomLeftPanel {
			// Y relative to bottom panel content area
			contentY := mouse.Y - leftTopHeight - contentYOffset

			// Focus op log panel
			m.focusedPane = PaneOpLog
			m.updatePanelFocus()

			if m.opLogPanel.HandleClick(contentY) {
				if op := m.opLogPanel.SelectedOperation(); op != nil {
					return tea.Batch(m.loadOpShow(op.OpID), m.startLogPanelBorderAnim())
				}
			}

			return m.startLogPanelBorderAnim()
		} else if inRightPanel {
			// Focus right panel
			m.focusedPane = PaneDiff
			m.updatePanelFocus()

			return m.startLogPanelBorderAnim()
		}
	}

	return nil
}

func (m *Model) updatePanelSizes() {
	// Leave room for status bar
	contentHeight := m.height - statusBarHeight

	// Split horizontally: left panel ~40%, right panel ~60%
	leftWidth := m.width * leftPanelWidthPct / percentDivisor
	rightWidth := m.width - leftWidth

	// Left pane splits vertically: log 50%, op log 50%
	leftTopHeight := contentHeight / leftPanelSplitDivisor
	leftBottomHeight := contentHeight - leftTopHeight

	m.logPanel.SetSize(leftWidth, leftTopHeight)
	m.opLogPanel.SetSize(leftWidth, leftBottomHeight)
	m.filesPanel.SetSize(leftWidth, leftTopHeight) // Files panel uses same size as log
	m.diffPanel.SetSize(rightWidth, contentHeight)
}

// View renders the application
func (m Model) View() tea.View {
	view := tea.NewView("")
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion

	if m.width == 0 || m.height == 0 {
		view.SetContent("Loading...")
		return view
	}

	// Render left panels (log/files + op log stacked)
	var leftTop string

	switch m.viewMode {
	case ViewLog:
		leftTop = m.logPanel.View()
	case ViewFiles:
		leftTop = m.filesPanel.View()
	}

	leftBottom := m.opLogPanel.View()
	leftPanel := lipgloss.JoinVertical(lipgloss.Left, leftTop, leftBottom)

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
		view.SetContent(m.renderWithOverlay(base))
	} else if m.editMode {
		view.SetContent(m.renderWithDescribeOverlay(base))
	} else {
		view.SetContent(base)
	}

	return view
}

// renderWithOverlay composites the help modal on top of the base view
// using lipgloss v2 Canvas/Layer for true transparency.
func (m Model) renderWithOverlay(base string) string {
	// Calculate modal size (centered, ~80% of screen)
	modalWidth := m.width * modalWidthPct / percentDivisor
	modalHeight := m.height * modalHeightPct / percentDivisor

	if modalWidth < minModalWidth {
		modalWidth = min(minModalWidth, m.width-modalEdgePadding)
	}

	if modalHeight < minModalHeight {
		modalHeight = min(minModalHeight, m.height-modalEdgePadding)
	}

	// Set up and render floating help
	m.floatingHelp.SetSize(modalWidth, modalHeight)
	m.floatingHelp.SetBindings(m.activeHelpBindings())
	modal := m.floatingHelp.View()

	// Calculate center position
	overlayWidth := lipgloss.Width(modal)
	overlayHeight := lipgloss.Height(modal)
	overlayX := (m.width - overlayWidth) / centerDivisor
	overlayY := (m.height - overlayHeight) / centerDivisor

	// Create base layer (full screen)
	baseLayer := lipgloss.NewLayer(base).
		Width(m.width).
		Height(m.height).
		X(0).Y(0).Z(0)

	// Create overlay layer (centered, on top)
	overlayLayer := lipgloss.NewLayer(modal).
		X(overlayX).Y(overlayY).Z(1)

	// Composite and render
	canvas := lipgloss.NewCanvas(baseLayer, overlayLayer)

	return canvas.Render()
}

func (m Model) renderStatusBar() string {
	m.statusBar.SetWidth(m.width)
	m.statusBar.SetBindings(m.activeHelpBindings())

	return ui.StatusBarStyle.Render(m.statusBar.View())
}

// renderWithDescribeOverlay composites the describe input on top of the base view
// using lipgloss v2 Canvas/Layer for true transparency.
func (m Model) renderWithDescribeOverlay(base string) string {
	// Render the describe input
	describeView := m.describeInput.View()
	overlayWidth := m.describeInput.Width()
	overlayHeight := m.describeInput.Height()

	// Calculate center position
	overlayX := (m.width - overlayWidth) / centerDivisor
	overlayY := (m.height - overlayHeight) / centerDivisor

	// Create base layer (full screen)
	baseLayer := lipgloss.NewLayer(base).
		Width(m.width).
		Height(m.height).
		X(0).Y(0).Z(0)

	// Create overlay layer (centered, on top)
	overlayLayer := lipgloss.NewLayer(describeView).
		X(overlayX).Y(overlayY).Z(1)

	// Composite and render
	canvas := lipgloss.NewCanvas(baseLayer, overlayLayer)

	return canvas.Render()
}
