package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/chatter/chado/internal/jj"
	"github.com/chatter/chado/internal/ui/help"
)

const (
	// opLogPanelNumber is the panel index shown in the title gutter.
	opLogPanelNumber = 2
)

// OpLogMode represents the current display mode of the OpLogPanel.
type OpLogMode int

const (
	ModeOpLog  OpLogMode = iota // Global operation log (jj op log)
	ModeEvoLog                  // Evolution log for a specific change (jj evolog -r)
)

// OpLogPanel displays the jj operation log or evolution log.
type OpLogPanel struct {
	viewport        viewport.Model
	operations      []jj.Operation
	cursor          int
	focused         bool
	width           int
	height          int
	rawLog          string  // Keep raw log for display
	opStartLines    []int   // Line number where each operation starts (pre-computed)
	totalLines      int     // Total number of lines in rawLog (for bounds checking)
	borderAnimPhase float64 // 0..1 for focus border animation
	borderAnimating bool    // true only while the one-shot wrap is running

	// Mode fields for evolog support
	mode      OpLogMode // Current display mode (op log or evolog)
	changeID  string    // Change ID when in evolog mode
	shortCode string    // Shortest unique prefix for highlighting
}

// NewOpLogPanel creates a new operation log panel.
func NewOpLogPanel() OpLogPanel {
	vp := viewport.New()
	vp.SoftWrap = false // Disable word wrap, allow horizontal scrolling

	return OpLogPanel{
		viewport:   vp,
		operations: []jj.Operation{},
		cursor:     0,
	}
}

// SetSize sets the panel dimensions.
func (p *OpLogPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	// Account for border and title
	p.viewport.SetWidth(width - PanelBorderWidth)
	p.viewport.SetHeight(height - PanelChromeHeight)
}

// SetFocused sets the focus state.
func (p *OpLogPanel) SetFocused(focused bool) {
	p.focused = focused
}

// SetBorderAnimPhase sets the border animation phase (0..1) for the focus wrap effect.
func (p *OpLogPanel) SetBorderAnimPhase(phase float64) {
	p.borderAnimPhase = phase
}

// SetBorderAnimating sets whether the focus border animation is running.
func (p *OpLogPanel) SetBorderAnimating(animating bool) {
	p.borderAnimating = animating
}

// SetContent sets the op log content from raw jj output.
func (p *OpLogPanel) SetContent(rawLog string, operations []jj.Operation) {
	// Capture current selection before overwriting
	var selectedID string
	if sel := p.SelectedOperation(); sel != nil {
		selectedID = sel.OpID
	}

	p.rawLog = rawLog
	p.operations = operations

	// Try to preserve selection by operation ID
	if selectedID != "" {
		if idx := findOpIndex(operations, selectedID); idx >= 0 {
			p.cursor = idx
		} else {
			// Operation was removed, default to first
			p.cursor = 0
		}
	}

	p.computeOpStartLines()
	p.updateViewport()
}

// findOpIndex returns the index of the operation with the given ID, or -1 if not found.
func findOpIndex(operations []jj.Operation, opID string) int {
	for i, op := range operations {
		if op.OpID == opID {
			return i
		}
	}

	return -1
}

// SetOpLogContent switches to global op log mode and sets content.
func (p *OpLogPanel) SetOpLogContent(rawLog string, operations []jj.Operation) {
	p.mode = ModeOpLog
	p.changeID = ""
	p.shortCode = ""
	p.SetContent(rawLog, operations)
}

// SetEvoLogContent switches to evolog mode for a specific change and sets content.
func (p *OpLogPanel) SetEvoLogContent(changeID, shortCode, rawLog string, operations []jj.Operation) {
	p.mode = ModeEvoLog
	p.changeID = changeID
	p.shortCode = shortCode
	p.SetContent(rawLog, operations)
}

// isEntryStart checks if a line starts a new entry (operation or change).
func isEntryStart(line string) bool {
	stripped := ansiRegex.ReplaceAllString(line, "")
	return jj.EntryLineRe.MatchString(stripped)
}

// SelectedOperation returns the currently selected operation.
func (p *OpLogPanel) SelectedOperation() *jj.Operation {
	if p.cursor >= 0 && p.cursor < len(p.operations) {
		return &p.operations[p.cursor]
	}

	return nil
}

// CursorUp moves the cursor up.
func (p *OpLogPanel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
		p.updateViewport()
	}
}

// CursorDown moves the cursor down.
func (p *OpLogPanel) CursorDown() {
	if p.cursor < len(p.operations)-1 {
		p.cursor++
		p.updateViewport()
	}
}

// GotoTop moves to the first item.
func (p *OpLogPanel) GotoTop() {
	p.cursor = 0
	p.updateViewport()
}

// GotoBottom moves to the last item.
func (p *OpLogPanel) GotoBottom() {
	if len(p.operations) > 0 {
		p.cursor = len(p.operations) - 1
		p.updateViewport()
	}
}

// HandleClick selects the operation at the given Y coordinate (relative to content area).
// Returns true if the selection changed.
func (p *OpLogPanel) HandleClick(y int) bool {
	// Account for viewport scroll offset
	visualLine := y + p.viewport.YOffset()

	opIdx := p.lineToOpIndex(visualLine)
	if opIdx >= 0 && opIdx < len(p.operations) && opIdx != p.cursor {
		p.cursor = opIdx
		p.updateViewport()

		return true
	}

	return false
}

// Update handles input.
func (p *OpLogPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.focused {
		return nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "j", "down":
			p.CursorDown()
		case "k", "up":
			p.CursorUp()
		case "g":
			p.GotoTop()
		case "G":
			p.GotoBottom()
		}
	}

	return nil
}

// View renders the panel.
func (p *OpLogPanel) View() string {
	var title string

	switch p.mode {
	case ModeEvoLog:
		// Build change ID with shortcode highlighted (like FilesPanel)
		coloredID := p.changeID
		if p.shortCode != "" && len(p.shortCode) <= len(p.changeID) {
			rest := p.changeID[len(p.shortCode):]

			var outerColorCode string

			if p.focused {
				outerColorCode = AccentColorCode
			} else {
				outerColorCode = PrimaryColorCode
			}

			coloredID = ReplaceResetWithColor(ShortCodeStyle.Render(p.shortCode), outerColorCode) + rest
		}

		title = PanelTitle(opLogPanelNumber, "Evolution: "+coloredID, p.focused)
	default:
		title = PanelTitle(opLogPanelNumber, "Operations Log", p.focused)
	}

	// Get the appropriate border style
	var style lipgloss.Style

	switch {
	case p.focused && p.borderAnimating:
		style = AnimatedFocusBorderStyle(p.borderAnimPhase, p.width, p.height)
	case p.focused:
		style = FocusedPanelStyle
	default:
		style = PanelStyle
	}

	// Build content with title
	content := title + "\n" + p.viewport.View()

	return style.Render(content)
}

// HelpBindings returns the keybindings for this panel (display-only, for status bar).
func (p *OpLogPanel) HelpBindings() []help.Binding {
	return []help.Binding{
		{
			Key:      key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "up/down")),
			Category: help.CategoryNavigation,
			Order:    PanelOrderPrimary,
		},
		{
			Key:      key.NewBinding(key.WithKeys("g", "G"), key.WithHelp("g/G", "top/bottom")),
			Category: help.CategoryNavigation,
			Order:    PanelOrderSecondary,
		},
	}
}

// computeOpStartLines pre-computes the line number where each operation starts.
func (p *OpLogPanel) computeOpStartLines() {
	p.opStartLines = nil
	p.totalLines = 0

	if p.rawLog == "" {
		return
	}

	// Count actual lines (newlines), not split elements (which includes trailing empty)
	p.totalLines = strings.Count(p.rawLog, "\n")

	lines := strings.Split(p.rawLog, "\n")
	for i, line := range lines {
		if isEntryStart(line) {
			p.opStartLines = append(p.opStartLines, i)
		}
	}
}

func (p *OpLogPanel) ensureCursorVisible() {
	if p.cursor < 0 || p.cursor >= len(p.opStartLines) {
		return
	}

	cursorLine := p.opStartLines[p.cursor]
	viewTop := p.viewport.YOffset()
	viewBottom := viewTop + p.viewport.Height()

	if cursorLine < viewTop {
		p.viewport.SetYOffset(cursorLine)
	} else if cursorLine >= viewBottom {
		p.viewport.SetYOffset(cursorLine - p.viewport.Height() + ScrollPadding)
	}
}

// lineToOpIndex maps a visual line number to an operation index.
// Returns -1 if the line is outside content bounds or before any operation.
func (p *OpLogPanel) lineToOpIndex(visualLine int) int {
	if len(p.opStartLines) == 0 || visualLine < 0 || visualLine >= p.totalLines {
		return -1
	}

	// Find the largest operation index where opStartLines[i] <= visualLine
	opIdx := -1

	for i, startLine := range p.opStartLines {
		if startLine <= visualLine {
			opIdx = i
		} else {
			break
		}
	}

	return opIdx
}

func (p *OpLogPanel) updateViewport() {
	if p.rawLog == "" {
		p.viewport.SetContent("No operations")
		return
	}

	var result strings.Builder

	nextOpIdx := 0

	lines := strings.Split(p.rawLog, "\n")
	for i, line := range lines {
		// Check if this line starts an operation (using pre-computed array)
		isStart := nextOpIdx < len(p.opStartLines) && i == p.opStartLines[nextOpIdx]

		// Add selection indicator on the start line of the selected operation
		if isStart && nextOpIdx == p.cursor {
			fmt.Fprintf(&result, "→ %s\n", line)
		} else {
			fmt.Fprintf(&result, "  %s\n", line)
		}

		if isStart {
			nextOpIdx++
		}
	}

	p.viewport.SetContent(result.String())
	p.ensureCursorVisible()
}
