package ui

import (
	"crypto/sha256"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/chatter/chado/internal/jj"
	"github.com/chatter/chado/internal/ui/help"
)

// noHunkSelected indicates viewport is in header area, before any hunk.
const noHunkSelected = -1

// mouseScrollLines is the number of lines to scroll per mouse wheel tick.
const mouseScrollLines = 3

// DiffPanel displays diff content with optional details header.
type DiffPanel struct {
	viewport        viewport.Model
	focused         bool
	width           int
	height          int
	title           string
	showDetails     bool
	details         DetailsHeader
	diffContent     string
	hunks           []jj.Hunk
	currentHunk     int
	headerLines     int      // Number of lines in the header (offset for hunk positions)
	contentHash     [32]byte // SHA-256 of diffContent; used to skip no-op SetDiff calls
	borderAnimPhase float64  // 0..1 for focus border animation
	borderAnimating bool     // true only while the one-shot wrap is running
}

// DetailsHeader contains the commit details shown above the diff.
type DetailsHeader struct {
	ChangeID    string
	CommitID    string
	Author      string
	Date        string
	Description string
}

// NewDiffPanel creates a new diff panel.
func NewDiffPanel() DiffPanel {
	vp := viewport.New()

	return DiffPanel{
		viewport:    vp,
		title:       "Diff",
		showDetails: true,
	}
}

// SetSize sets the panel dimensions.
func (p *DiffPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.viewport.SetWidth(width - PanelBorderWidth)
	p.viewport.SetHeight(height - PanelChromeHeight)
}

// SetFocused sets the focus state.
func (p *DiffPanel) SetFocused(focused bool) {
	p.focused = focused
}

// SetBorderAnimPhase sets the border animation phase (0..1) for the focus wrap effect.
func (p *DiffPanel) SetBorderAnimPhase(phase float64) {
	p.borderAnimPhase = phase
}

// SetBorderAnimating sets whether the focus border animation is running.
func (p *DiffPanel) SetBorderAnimating(animating bool) {
	p.borderAnimating = animating
}

// SetTitle sets the panel title.
func (p *DiffPanel) SetTitle(title string) {
	p.title = title
}

// SetShowDetails controls whether to show the details header.
func (p *DiffPanel) SetShowDetails(show bool) {
	p.showDetails = show
}

// SetDetails sets the commit details header.
func (p *DiffPanel) SetDetails(details DetailsHeader) {
	p.details = details
	p.updateContent()
}

// SetDiff sets the diff content. If the content is unchanged (same SHA-256
// hash), it returns immediately — no viewport update, no scroll reset.
func (p *DiffPanel) SetDiff(diff string) {
	hash := sha256.Sum256([]byte(diff))
	if hash == p.contentHash {
		return
	}

	p.contentHash = hash
	p.diffContent = diff
	p.hunks = jj.FindHunks(diff)
	p.currentHunk = noHunkSelected
	p.updateContent()
	p.viewport.GotoTop()
}

// NextHunk jumps to the next hunk/section.
func (p *DiffPanel) NextHunk() {
	if len(p.hunks) == 0 || p.currentHunk >= len(p.hunks)-1 {
		return
	}

	p.currentHunk++
	p.viewport.SetYOffset(p.hunks[p.currentHunk].StartLine + p.headerLines)
}

// PrevHunk jumps to start of current hunk, or previous hunk if already at start.
func (p *DiffPanel) PrevHunk() {
	if len(p.hunks) == 0 {
		return
	}

	// If no hunk selected, go to top
	if p.currentHunk == noHunkSelected {
		p.viewport.GotoTop()
		return
	}

	currentHunkStart := p.hunks[p.currentHunk].StartLine + p.headerLines

	// If not at start of current hunk, go to start of current hunk
	if p.viewport.YOffset() > currentHunkStart {
		p.viewport.SetYOffset(currentHunkStart)
		return
	}

	// Already at start of current hunk, go to previous hunk (or top if at hunk 0)
	p.currentHunk--
	if p.currentHunk >= 0 {
		p.viewport.SetYOffset(p.hunks[p.currentHunk].StartLine + p.headerLines)
	} else {
		p.viewport.GotoTop()
	}
}

// GotoTop scrolls to the top.
func (p *DiffPanel) GotoTop() {
	p.viewport.GotoTop()
	p.currentHunk = noHunkSelected
}

// GotoBottom scrolls to the bottom.
func (p *DiffPanel) GotoBottom() {
	p.viewport.GotoBottom()

	if len(p.hunks) > 0 {
		p.currentHunk = len(p.hunks) - 1
	}
}

// HandleMouseScroll handles mouse wheel events.
func (p *DiffPanel) HandleMouseScroll(button tea.MouseButton) {
	switch button {
	case tea.MouseWheelUp:
		p.viewport.ScrollUp(mouseScrollLines)
	case tea.MouseWheelDown:
		p.viewport.ScrollDown(mouseScrollLines)
	}

	p.syncCurrentHunk()
}

// Update handles input.
func (p *DiffPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.focused {
		return nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "j", "down":
			p.viewport.ScrollDown(1)
			p.syncCurrentHunk()
		case "k", "up":
			p.viewport.ScrollUp(1)
			p.syncCurrentHunk()
		case "}":
			p.NextHunk()
		case "{":
			p.PrevHunk()
		case "g":
			p.GotoTop()
		case "G":
			p.GotoBottom()
		}
	}

	return nil
}

// View renders the panel.
func (p *DiffPanel) View() string {
	title := PanelTitle(0, p.title, p.focused)

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

	style = style.Height(p.height - PanelBorderHeight)

	// Build content with title
	content := title + "\n" + p.viewport.View()

	return style.Render(content)
}

// ParseDetailsFromShow parses jj show output to extract details.
func ParseDetailsFromShow(showOutput string) DetailsHeader {
	details := DetailsHeader{}
	lines := strings.Split(showOutput, "\n")

	// jj show output format varies, but typically has:
	// Commit ID: ...
	// Change ID: ...
	// Author: ...
	// Date: ...
	// Description

	changeIDRe := regexp.MustCompile(`(?i)change\s*id[:\s]+([a-z0-9]+)`)
	commitIDRe := regexp.MustCompile(`(?i)commit\s*id[:\s]+([a-f0-9]+)`)
	authorRe := regexp.MustCompile(`(?i)author[:\s]+(.+)`)
	dateRe := regexp.MustCompile(`(?i)(?:date|timestamp)[:\s]+(.+)`)
	inDescription := false

	var descLines []string

	for _, line := range lines {
		stripped := stripANSI(line)

		if match := changeIDRe.FindStringSubmatch(stripped); match != nil {
			details.ChangeID = match[1]
			continue
		}

		if match := commitIDRe.FindStringSubmatch(stripped); match != nil {
			details.CommitID = match[1]
			continue
		}

		if match := authorRe.FindStringSubmatch(stripped); match != nil {
			details.Author = strings.TrimSpace(match[1])
			continue
		}

		if match := dateRe.FindStringSubmatch(stripped); match != nil {
			details.Date = strings.TrimSpace(match[1])
			continue
		}

		// Check for description start
		if strings.HasPrefix(strings.ToLower(stripped), "description:") {
			inDescription = true
			desc := strings.TrimPrefix(stripped, "Description:")
			desc = strings.TrimPrefix(desc, "description:")

			if strings.TrimSpace(desc) != "" {
				descLines = append(descLines, strings.TrimSpace(desc))
			}

			continue
		}

		// Collect description lines
		if inDescription {
			if strings.HasPrefix(stripped, "diff ") {
				// End of description, start of diff
				break
			}

			if strings.TrimSpace(stripped) != "" {
				descLines = append(descLines, strings.TrimSpace(stripped))
			}
		}
	}

	details.Description = strings.Join(descLines, "\n")

	return details
}

// stripANSI removes ANSI escape codes.
func stripANSI(s string) string {
	ansiRe := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRe.ReplaceAllString(s, "")
}

// HelpBindings returns the keybindings for this panel (display-only, for status bar).
func (p *DiffPanel) HelpBindings() []help.Binding {
	return []help.Binding{
		{
			Key:      key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "up/down")),
			Category: help.CategoryNavigation,
			Order:    PanelOrderPrimary,
		},
		{
			Key:      key.NewBinding(key.WithKeys("{", "}"), key.WithHelp("{/}", "next/prev hunk")),
			Category: help.CategoryDiff,
			Order:    PanelOrderPrimary,
		},
		{
			Key:      key.NewBinding(key.WithKeys("g", "G"), key.WithHelp("g/G", "top/bottom")),
			Category: help.CategoryNavigation,
			Order:    PanelOrderSecondary,
		},
	}
}

// syncCurrentHunk updates currentHunk based on viewport position.
func (p *DiffPanel) syncCurrentHunk() {
	if len(p.hunks) == 0 {
		p.currentHunk = noHunkSelected
		return
	}

	pos := p.viewport.YOffset() - p.headerLines

	for i := len(p.hunks) - 1; i >= 0; i-- {
		if pos >= p.hunks[i].StartLine {
			p.currentHunk = i
			return
		}
	}

	p.currentHunk = noHunkSelected
}

func (p *DiffPanel) updateContent() {
	var content strings.Builder

	p.headerLines = 0

	// Add details header if enabled
	if p.showDetails && p.details.ChangeID != "" {
		// Description first
		if p.details.Description != "" {
			content.WriteString(p.details.Description)
			content.WriteString("\n\n")
			// Count lines in description
			// +1 for the description line itself, +1 for the trailing blank line
			descriptionChrome := 2
			p.headerLines += strings.Count(p.details.Description, "\n") + descriptionChrome
		}

		// Metadata line
		content.WriteString("Change: ")
		content.WriteString(p.details.ChangeID)

		if p.details.CommitID != "" {
			content.WriteString("  Commit: ")
			content.WriteString(p.details.CommitID)
		}

		content.WriteString("\n")

		p.headerLines++

		if p.details.Author != "" {
			content.WriteString("Author: ")
			content.WriteString(p.details.Author)
		}

		if p.details.Date != "" {
			content.WriteString("  ")
			content.WriteString(p.details.Date)
		}

		content.WriteString("\n")

		p.headerLines++

		// Separator
		content.WriteString(strings.Repeat("─", p.viewport.Width()))
		content.WriteString("\n")

		p.headerLines++
	}

	// Add diff content - apply word wrap to fit viewport width
	viewportWidth := p.viewport.Width()
	if viewportWidth > 0 {
		wrapped := lipgloss.NewStyle().Width(viewportWidth).Render(p.diffContent)
		content.WriteString(wrapped)
	} else {
		content.WriteString(p.diffContent)
	}

	p.viewport.SetContent(content.String())
}
