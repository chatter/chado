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

// ansiRegex matches ANSI escape codes
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// LogPanel displays the jj log
type LogPanel struct {
	viewport         viewport.Model
	changes          []jj.Change
	cursor           int
	focused          bool
	width            int
	height           int
	rawLog           string // Keep raw log for display
	changeStartLines []int  // Line number where each change starts (pre-computed)
	totalLines       int    // Total number of lines in rawLog (for bounds checking)
}

// NewLogPanel creates a new log panel
func NewLogPanel() LogPanel {
	vp := viewport.New(0, 0)
	return LogPanel{
		viewport: vp,
		changes:  []jj.Change{},
		cursor:   0,
	}
}

// SetSize sets the panel dimensions
func (p *LogPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	// Account for border (2) and title (1)
	p.viewport.Width = width - 2
	p.viewport.Height = height - 3
}

// SetFocused sets the focus state
func (p *LogPanel) SetFocused(focused bool) {
	p.focused = focused
}

// findChangeIndex returns the index of the change with the given ID, or -1 if not found
func findChangeIndex(changes []jj.Change, changeID string) int {
	for i, c := range changes {
		if c.ChangeID == changeID {
			return i
		}
	}
	return -1
}

// SetContent sets the log content from raw jj output
func (p *LogPanel) SetContent(rawLog string, changes []jj.Change) {
	// Capture current selection before overwriting
	var selectedID string
	if sel := p.SelectedChange(); sel != nil {
		selectedID = sel.ChangeID
	}

	p.rawLog = rawLog
	p.changes = changes

	// Try to preserve selection by change ID
	if selectedID != "" {
		if idx := findChangeIndex(changes, selectedID); idx >= 0 {
			p.cursor = idx
		} else {
			// Change was removed, default to first
			p.cursor = 0
		}
	}

	p.computeChangeStartLines()
	p.updateViewport()
}

// computeChangeStartLines pre-computes the line number where each change starts
func (p *LogPanel) computeChangeStartLines() {
	p.changeStartLines = nil
	p.totalLines = 0
	if p.rawLog == "" {
		return
	}

	lines := strings.Split(p.rawLog, "\n")
	// Count actual lines (newlines), not split elements (which includes trailing empty)
	p.totalLines = strings.Count(p.rawLog, "\n")
	for i, line := range lines {
		if isChangeStart(line) {
			p.changeStartLines = append(p.changeStartLines, i)
		}
	}
}

// SelectedChange returns the currently selected change
func (p *LogPanel) SelectedChange() *jj.Change {
	if p.cursor >= 0 && p.cursor < len(p.changes) {
		return &p.changes[p.cursor]
	}
	return nil
}

// CursorUp moves the cursor up
func (p *LogPanel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
		p.updateViewport()
	}
}

// CursorDown moves the cursor down
func (p *LogPanel) CursorDown() {
	if p.cursor < len(p.changes)-1 {
		p.cursor++
		p.updateViewport()
	}
}

// GotoTop moves to the first item
func (p *LogPanel) GotoTop() {
	p.cursor = 0
	p.updateViewport()
}

// GotoBottom moves to the last item
func (p *LogPanel) GotoBottom() {
	if len(p.changes) > 0 {
		p.cursor = len(p.changes) - 1
		p.updateViewport()
	}
}

// changeLineRe matches change lines - requires a graph symbol (not just whitespace)
// Symbols: @ (working copy), ○ (normal), ◆ (immutable), ◇ (empty), ● (hidden), × (conflict)
var changeLineRe = regexp.MustCompile(`^[│├└\s]*[@○◆◇●×]\s*([a-z]{8,})\s`)

// isChangeStart checks if a line starts a new change entry
func isChangeStart(line string) bool {
	stripped := ansiRegex.ReplaceAllString(line, "")
	return changeLineRe.MatchString(stripped)
}

func (p *LogPanel) updateViewport() {
	if p.rawLog == "" {
		p.viewport.SetContent("No changes")
		return
	}

	lines := strings.Split(p.rawLog, "\n")
	var result strings.Builder
	nextChangeIdx := 0

	for i, line := range lines {
		// Check if this line starts a change (using pre-computed array)
		isStart := nextChangeIdx < len(p.changeStartLines) && i == p.changeStartLines[nextChangeIdx]

		// Add selection indicator on the start line of the selected change
		if isStart && nextChangeIdx == p.cursor {
			fmt.Fprintf(&result, "→ %s\n", line)
		} else {
			fmt.Fprintf(&result, "  %s\n", line)
		}

		if isStart {
			nextChangeIdx++
		}
	}

	p.viewport.SetContent(result.String())
	p.ensureCursorVisible()
}

func (p *LogPanel) ensureCursorVisible() {
	if p.cursor < 0 || p.cursor >= len(p.changeStartLines) {
		return
	}

	cursorLine := p.changeStartLines[p.cursor]
	viewTop := p.viewport.YOffset
	viewBottom := viewTop + p.viewport.Height

	if cursorLine < viewTop {
		p.viewport.SetYOffset(cursorLine)
	} else if cursorLine >= viewBottom {
		p.viewport.SetYOffset(cursorLine - p.viewport.Height + 2)
	}
}

// lineToChangeIndex maps a visual line number to a change index
// Returns -1 if the line is outside content bounds or before any change
func (p *LogPanel) lineToChangeIndex(visualLine int) int {
	if len(p.changeStartLines) == 0 || visualLine < 0 || visualLine >= p.totalLines {
		return -1
	}

	// Find the largest change index where changeStartLines[i] <= visualLine
	changeIdx := -1
	for i, startLine := range p.changeStartLines {
		if startLine <= visualLine {
			changeIdx = i
		} else {
			break
		}
	}
	return changeIdx
}

// HandleClick selects the change at the given Y coordinate (relative to content area)
// Returns true if the selection changed
func (p *LogPanel) HandleClick(y int) bool {
	// Account for viewport scroll offset
	visualLine := y + p.viewport.YOffset

	changeIdx := p.lineToChangeIndex(visualLine)
	if changeIdx >= 0 && changeIdx < len(p.changes) && changeIdx != p.cursor {
		p.cursor = changeIdx
		p.updateViewport()
		return true
	}
	return false
}

// Update handles input
func (p *LogPanel) Update(msg tea.Msg) tea.Cmd {
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
			// Check for gg
			p.GotoTop()
		case "G":
			p.GotoBottom()
		}
	}

	return nil
}

// View renders the panel
func (p LogPanel) View() string {
	title := PanelTitle(1, "Change Log", p.focused)

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
func (p LogPanel) HelpBindings() []help.HelpBinding {
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
