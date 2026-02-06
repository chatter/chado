package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/chatter/chado/internal/jj"
	"github.com/chatter/chado/internal/ui/help"
)

// OpLogPanel displays the jj operation log
type OpLogPanel struct {
	viewport     viewport.Model
	operations   []jj.Operation
	cursor       int
	focused      bool
	width        int
	height       int
	rawLog       string // Keep raw log for display
	opStartLines []int  // Line number where each operation starts (pre-computed)
	totalLines   int    // Total number of lines in rawLog (for bounds checking)
}

// NewOpLogPanel creates a new operation log panel
func NewOpLogPanel() OpLogPanel {
	vp := viewport.New(0, 0)
	return OpLogPanel{
		viewport:   vp,
		operations: []jj.Operation{},
		cursor:     0,
	}
}

// SetSize sets the panel dimensions
func (p *OpLogPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	// Account for border (2) and title (1)
	p.viewport.Width = width - 2
	p.viewport.Height = height - 3
}

// SetFocused sets the focus state
func (p *OpLogPanel) SetFocused(focused bool) {
	p.focused = focused
}

// findOpIndex returns the index of the operation with the given ID, or -1 if not found
func findOpIndex(operations []jj.Operation, opID string) int {
	for i, op := range operations {
		if op.OpID == opID {
			return i
		}
	}
	return -1
}

// SetContent sets the op log content from raw jj output
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

// opLineRe matches operation lines - requires @ or ○ symbol followed by operation ID
var opLineRe = regexp.MustCompile(`^[│├└\s]*[@○]\s+([0-9a-f]{12})\s`)

// isOpStart checks if a line starts a new operation entry
func isOpStart(line string) bool {
	stripped := ansiRegex.ReplaceAllString(line, "")
	return opLineRe.MatchString(stripped)
}

// computeOpStartLines pre-computes the line number where each operation starts
func (p *OpLogPanel) computeOpStartLines() {
	p.opStartLines = nil
	p.totalLines = 0
	if p.rawLog == "" {
		return
	}

	lines := strings.Split(p.rawLog, "\n")
	// Count actual lines (newlines), not split elements (which includes trailing empty)
	p.totalLines = strings.Count(p.rawLog, "\n")
	for i, line := range lines {
		if isOpStart(line) {
			p.opStartLines = append(p.opStartLines, i)
		}
	}
}

// SelectedOperation returns the currently selected operation
func (p *OpLogPanel) SelectedOperation() *jj.Operation {
	if p.cursor >= 0 && p.cursor < len(p.operations) {
		return &p.operations[p.cursor]
	}
	return nil
}

// CursorUp moves the cursor up
func (p *OpLogPanel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
		p.updateViewport()
	}
}

// CursorDown moves the cursor down
func (p *OpLogPanel) CursorDown() {
	if p.cursor < len(p.operations)-1 {
		p.cursor++
		p.updateViewport()
	}
}

// GotoTop moves to the first item
func (p *OpLogPanel) GotoTop() {
	p.cursor = 0
	p.updateViewport()
}

// GotoBottom moves to the last item
func (p *OpLogPanel) GotoBottom() {
	if len(p.operations) > 0 {
		p.cursor = len(p.operations) - 1
		p.updateViewport()
	}
}

func (p *OpLogPanel) updateViewport() {
	if p.rawLog == "" {
		p.viewport.SetContent("No operations")
		return
	}

	lines := strings.Split(p.rawLog, "\n")
	var result strings.Builder
	nextOpIdx := 0

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

func (p *OpLogPanel) ensureCursorVisible() {
	if p.cursor < 0 || p.cursor >= len(p.opStartLines) {
		return
	}

	cursorLine := p.opStartLines[p.cursor]
	viewTop := p.viewport.YOffset
	viewBottom := viewTop + p.viewport.Height

	if cursorLine < viewTop {
		p.viewport.SetYOffset(cursorLine)
	} else if cursorLine >= viewBottom {
		p.viewport.SetYOffset(cursorLine - p.viewport.Height + 2)
	}
}

// lineToOpIndex maps a visual line number to an operation index
// Returns -1 if the line is outside content bounds or before any operation
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

// HandleClick selects the operation at the given Y coordinate (relative to content area)
// Returns true if the selection changed
func (p *OpLogPanel) HandleClick(y int) bool {
	// Account for viewport scroll offset
	visualLine := y + p.viewport.YOffset

	opIdx := p.lineToOpIndex(visualLine)
	if opIdx >= 0 && opIdx < len(p.operations) && opIdx != p.cursor {
		p.cursor = opIdx
		p.updateViewport()
		return true
	}
	return false
}

// Update handles input
func (p *OpLogPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
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

// View renders the panel
func (p OpLogPanel) View() string {
	title := PanelTitle(2, "Operations", p.focused)

	// Get the appropriate border style
	var style lipgloss.Style
	if p.focused {
		style = FocusedPanelStyle
	} else {
		style = PanelStyle
	}

	// Set dimensions
	style = style.Width(p.width - 2).Height(p.height - 2)

	// Build content with title
	content := title + "\n" + p.viewport.View()

	return style.Render(content)
}

// HelpBindings returns the keybindings for this panel (display-only, for status bar)
func (p OpLogPanel) HelpBindings() []help.HelpBinding {
	return []help.HelpBinding{
		{
			Binding:  key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "up/down")),
			Category: help.CategoryNavigation,
			Order:    1,
		},
		{
			Binding:  key.NewBinding(key.WithKeys("g", "G"), key.WithHelp("g/G", "top/bottom")),
			Category: help.CategoryNavigation,
			Order:    2,
		},
	}
}
