package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/chatter/lazyjj/internal/jj"
)

// LogPanel displays the jj log
type LogPanel struct {
	viewport viewport.Model
	changes  []jj.Change
	cursor   int
	focused  bool
	width    int
	height   int
	rawLog   string // Keep raw log for display
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

// SetContent sets the log content from raw jj output
func (p *LogPanel) SetContent(rawLog string, changes []jj.Change) {
	p.rawLog = rawLog
	p.changes = changes
	p.updateViewport()
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

func (p *LogPanel) updateViewport() {
	// For now, just display the raw log
	// We highlight the selected change by adding a marker
	if p.rawLog == "" {
		p.viewport.SetContent("No changes")
		return
	}

	// Split into change blocks and mark the selected one
	lines := strings.Split(p.rawLog, "\n")
	var result strings.Builder
	changeIdx := -1

	for _, line := range lines {
		// Check if this is a new change line
		if isChangeStart(line) {
			changeIdx++
		}

		// Add selection indicator
		if changeIdx == p.cursor && isChangeStart(line) {
			// Add cursor indicator
			result.WriteString("→ ")
			result.WriteString(line)
		} else if changeIdx == p.cursor {
			// Continuation of selected change
			result.WriteString("  ")
			result.WriteString(line)
		} else {
			result.WriteString("  ")
			result.WriteString(line)
		}
		result.WriteString("\n")
	}

	p.viewport.SetContent(result.String())

	// Ensure cursor is visible
	p.ensureCursorVisible()
}

func (p *LogPanel) ensureCursorVisible() {
	// Calculate approximate line position of cursor
	// Each change typically takes 2-3 lines
	approxLine := p.cursor * 2
	viewTop := p.viewport.YOffset
	viewBottom := viewTop + p.viewport.Height

	if approxLine < viewTop {
		p.viewport.SetYOffset(approxLine)
	} else if approxLine >= viewBottom {
		p.viewport.SetYOffset(approxLine - p.viewport.Height + 2)
	}
}

// isChangeStart checks if a line starts a new change entry
func isChangeStart(line string) bool {
	// Change lines typically start with graph characters followed by change ID
	// Look for patterns like "@", "○", "◆", "◇" after stripping ANSI
	trimmed := strings.TrimLeft(line, " │├└")
	if len(trimmed) == 0 {
		return false
	}
	// Check for graph symbols
	for _, r := range trimmed {
		switch r {
		case '@', '○', '◆', '◇', '●':
			return true
		case ' ', '│', '├', '└':
			continue
		default:
			return false
		}
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
	title := PanelTitle(1, "Log", p.focused)

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
