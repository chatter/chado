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

// FilesPanel displays the list of files in a change
type FilesPanel struct {
	viewport        viewport.Model
	files           []jj.File
	cursor          int
	focused         bool
	width           int
	height          int
	changeID        string
	shortCode       string  // shortest unique prefix for coloring
	borderAnimPhase float64 // 0..1 for focus border animation
	borderAnimating bool    // true only while the one-shot wrap is running
}

// NewFilesPanel creates a new files panel
func NewFilesPanel() FilesPanel {
	vp := viewport.New()
	vp.SoftWrap = false // Disable word wrap, allow horizontal scrolling

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
	// Account for border and title
	p.viewport.SetWidth(width - PanelBorderWidth)
	p.viewport.SetHeight(height - PanelChromeHeight)
}

// SetFocused sets the focus state
func (p *FilesPanel) SetFocused(focused bool) {
	p.focused = focused
}

// SetBorderAnimPhase sets the border animation phase (0..1) for the focus wrap effect.
func (p *FilesPanel) SetBorderAnimPhase(phase float64) {
	p.borderAnimPhase = phase
}

// SetBorderAnimating sets whether the focus border animation is running.
func (p *FilesPanel) SetBorderAnimating(animating bool) {
	p.borderAnimating = animating
}

// SetFiles sets the file list
func (p *FilesPanel) SetFiles(changeID string, shortCode string, files []jj.File) {
	p.changeID = changeID
	p.shortCode = shortCode
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

	for idx, file := range p.files {
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
		if idx == p.cursor {
			cursor = "→ "
		}

		content.WriteString(fmt.Sprintf("%s%s %s\n", cursor, status, file.Path))
	}

	p.viewport.SetContent(content.String())

	// Ensure cursor is visible
	if p.cursor < p.viewport.YOffset() {
		p.viewport.SetYOffset(p.cursor)
	} else if p.cursor >= p.viewport.YOffset()+p.viewport.Height() {
		p.viewport.SetYOffset(p.cursor - p.viewport.Height() + 1)
	}
}

// HandleClick selects the file at the given Y coordinate (relative to content area)
func (p *FilesPanel) HandleClick(y int) bool {
	// Account for viewport scroll offset
	visualLine := y + p.viewport.YOffset()

	if visualLine >= 0 && visualLine < len(p.files) && visualLine != p.cursor {
		p.cursor = visualLine
		p.updateViewport()

		return true
	}

	return false
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
	// Build change ID with shortcode highlighted
	coloredID := p.changeID
	if p.shortCode != "" && len(p.shortCode) <= len(p.changeID) {
		rest := p.changeID[len(p.shortCode):]
		// Replace the reset with the outer title color so styling continues
		var outerColorCode string
		if p.focused {
			outerColorCode = AccentColorCode
		} else {
			outerColorCode = PrimaryColorCode
		}

		coloredID = ReplaceResetWithColor(ShortCodeStyle.Render(p.shortCode), outerColorCode) + rest
	}

	title := PanelTitle(1, coloredID+" / files", p.focused)

	// Get the appropriate border style
	var style lipgloss.Style
	if p.focused && p.borderAnimating {
		style = AnimatedFocusBorderStyle(p.borderAnimPhase, p.width, p.height)
	} else if p.focused {
		style = FocusedPanelStyle
	} else {
		style = PanelStyle
	}

	style = style.Height(p.height - PanelBorderHeight)

	// Build content with title
	content := title + "\n" + p.viewport.View()

	return style.Render(content)
}

// HelpBindings returns the keybindings for this panel (display-only, for status bar)
func (p FilesPanel) HelpBindings() []help.Binding {
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
