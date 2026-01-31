package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/chatter/lazyjj/internal/jj"
)

// ansiRegex matches ANSI escape codes
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

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
	changeIdx := -1

	for _, line := range lines {
		isStart := false
		if isStart = isChangeStart(line); isStart {
			changeIdx++
		}

		// Add selection indicator
		if changeIdx == p.cursor && isStart {
			fmt.Fprintf(&result, "→ %s\n", line)
		} else {
			fmt.Fprintf(&result, "  %s\n", line)
		}
	}

	p.viewport.SetContent(result.String())
	p.ensureCursorVisible()
}

func (p *LogPanel) ensureCursorVisible() {
	if p.cursor < 0 || p.rawLog == "" {
		return
	}

	lines := strings.Split(p.rawLog, "\n")
	changeIdx := -1
	cursorLine := 0

	for i, line := range lines {
		if isChangeStart(line) {
			changeIdx++
			if changeIdx == p.cursor {
				cursorLine = i
				break
			}
		}
	}

	viewTop := p.viewport.YOffset
	viewBottom := viewTop + p.viewport.Height

	if cursorLine < viewTop {
		p.viewport.SetYOffset(cursorLine)
	} else if cursorLine >= viewBottom {
		p.viewport.SetYOffset(cursorLine - p.viewport.Height + 2)
	}
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
