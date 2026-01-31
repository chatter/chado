package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/chatter/lazyjj/internal/jj"
)

// FilesPanel displays the list of files in a change
type FilesPanel struct {
	viewport viewport.Model
	files    []jj.File
	cursor   int
	focused  bool
	width    int
	height   int
	changeID string
}

// NewFilesPanel creates a new files panel
func NewFilesPanel() FilesPanel {
	vp := viewport.New(0, 0)
	return FilesPanel{
		viewport: vp,
		files:    []jj.File{},
		cursor:   0,
	}
}

// SetSize sets the panel dimensions
func (p *FilesPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	// Account for border (2) and title (1)
	p.viewport.Width = width - 2
	p.viewport.Height = height - 3
}

// SetFocused sets the focus state
func (p *FilesPanel) SetFocused(focused bool) {
	p.focused = focused
}

// SetFiles sets the file list
func (p *FilesPanel) SetFiles(changeID string, files []jj.File) {
	p.changeID = changeID
	p.files = files
	p.cursor = 0
	p.updateViewport()
}

// SelectedFile returns the currently selected file
func (p *FilesPanel) SelectedFile() *jj.File {
	if p.cursor >= 0 && p.cursor < len(p.files) {
		return &p.files[p.cursor]
	}
	return nil
}

// ChangeID returns the current change ID
func (p *FilesPanel) ChangeID() string {
	return p.changeID
}

// CursorUp moves the cursor up
func (p *FilesPanel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
		p.updateViewport()
	}
}

// CursorDown moves the cursor down
func (p *FilesPanel) CursorDown() {
	if p.cursor < len(p.files)-1 {
		p.cursor++
		p.updateViewport()
	}
}

// GotoTop moves to the first item
func (p *FilesPanel) GotoTop() {
	p.cursor = 0
	p.updateViewport()
}

// GotoBottom moves to the last item
func (p *FilesPanel) GotoBottom() {
	if len(p.files) > 0 {
		p.cursor = len(p.files) - 1
		p.updateViewport()
	}
}

func (p *FilesPanel) updateViewport() {
	if len(p.files) == 0 {
		p.viewport.SetContent("No files changed")
		return
	}

	var content strings.Builder
	for i, file := range p.files {
		// Status indicator with color
		var status string
		switch file.Status {
		case jj.FileAdded:
			status = "\033[32mA\033[0m" // Green
		case jj.FileDeleted:
			status = "\033[31mD\033[0m" // Red
		case jj.FileModified:
			status = "\033[33mM\033[0m" // Yellow
		default:
			status = string(file.Status)
		}

		// Selection indicator
		cursor := "  "
		if i == p.cursor {
			cursor = "→ "
		}

		content.WriteString(fmt.Sprintf("%s%s %s\n", cursor, status, file.Path))
	}

	p.viewport.SetContent(content.String())

	// Ensure cursor is visible
	if p.cursor < p.viewport.YOffset {
		p.viewport.SetYOffset(p.cursor)
	} else if p.cursor >= p.viewport.YOffset+p.viewport.Height {
		p.viewport.SetYOffset(p.cursor - p.viewport.Height + 1)
	}
}

// HandleClick selects the file at the given Y coordinate (relative to content area)
func (p *FilesPanel) HandleClick(y int) {
	// Account for viewport scroll offset
	visualLine := y + p.viewport.YOffset

	if visualLine >= 0 && visualLine < len(p.files) {
		p.cursor = visualLine
		p.updateViewport()
	}
}

// Update handles input
func (p *FilesPanel) Update(msg tea.Msg) tea.Cmd {
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
func (p FilesPanel) View() string {
	title := PanelTitle(1, p.changeID+" / files", p.focused)

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
