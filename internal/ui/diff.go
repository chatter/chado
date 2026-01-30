package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/chatter/lazyjj/internal/jj"
)

// DiffPanel displays diff content with optional details header
type DiffPanel struct {
	viewport    viewport.Model
	focused     bool
	width       int
	height      int
	title       string
	showDetails bool
	details     DetailsHeader
	diffContent string
	hunks       []jj.Hunk
	currentHunk int
	headerLines int // Number of lines in the header (offset for hunk positions)
}

// DetailsHeader contains the commit details shown above the diff
type DetailsHeader struct {
	ChangeID    string
	CommitID    string
	Author      string
	Date        string
	Description string
}

// NewDiffPanel creates a new diff panel
func NewDiffPanel() DiffPanel {
	vp := viewport.New(0, 0)
	return DiffPanel{
		viewport:    vp,
		title:       "Diff",
		showDetails: true,
	}
}

// SetSize sets the panel dimensions
func (p *DiffPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	// Account for border (2) and title (1)
	p.viewport.Width = width - 2
	p.viewport.Height = height - 3
}

// SetFocused sets the focus state
func (p *DiffPanel) SetFocused(focused bool) {
	p.focused = focused
}

// SetTitle sets the panel title
func (p *DiffPanel) SetTitle(title string) {
	p.title = title
}

// SetShowDetails controls whether to show the details header
func (p *DiffPanel) SetShowDetails(show bool) {
	p.showDetails = show
}

// SetDetails sets the commit details header
func (p *DiffPanel) SetDetails(details DetailsHeader) {
	p.details = details
	p.updateContent()
}

// SetDiff sets the diff content
func (p *DiffPanel) SetDiff(diff string) {
	p.diffContent = diff
	p.hunks = jj.FindHunks(diff)
	p.currentHunk = 0
	p.updateContent()
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
			p.headerLines += strings.Count(p.details.Description, "\n") + 2
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
		content.WriteString(strings.Repeat("─", p.viewport.Width))
		content.WriteString("\n")
		p.headerLines++
	}

	// Add diff content
	content.WriteString(p.diffContent)

	p.viewport.SetContent(content.String())
}

// NextHunk jumps to the next hunk/section, or scrolls down if no hunks
func (p *DiffPanel) NextHunk() {
	if len(p.hunks) == 0 {
		// Fall back to line-by-line scrolling
		p.viewport.ScrollDown(3)
		return
	}
	if p.currentHunk < len(p.hunks)-1 {
		p.currentHunk++

		// Add header offset to get correct viewport position
		p.viewport.SetYOffset(p.hunks[p.currentHunk].StartLine + p.headerLines)
	}
}

// PrevHunk jumps to the previous hunk/section, or scrolls up if no hunks
func (p *DiffPanel) PrevHunk() {
	if len(p.hunks) == 0 {
		// Fall back to line-by-line scrolling
		p.viewport.ScrollUp(3)
		return
	}
	if p.currentHunk > 0 {
		p.currentHunk--
		// Add header offset to get correct viewport position
		p.viewport.SetYOffset(p.hunks[p.currentHunk].StartLine + p.headerLines)
	}
}

// GotoTop scrolls to the top
func (p *DiffPanel) GotoTop() {
	p.viewport.GotoTop()
	p.currentHunk = 0
}

// GotoBottom scrolls to the bottom
func (p *DiffPanel) GotoBottom() {
	p.viewport.GotoBottom()
	if len(p.hunks) > 0 {
		p.currentHunk = len(p.hunks) - 1
	}
}

// Update handles input
func (p *DiffPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			p.NextHunk()
		case "k", "up":
			p.PrevHunk()
		case "g":
			p.GotoTop()
		case "G":
			p.GotoBottom()
		}
	}

	return nil
}

// View renders the panel
func (p DiffPanel) View() string {
	title := PanelTitle(0, p.title, p.focused)

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

// ParseDetailsFromShow parses jj show output to extract details
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

// stripANSI removes ANSI escape codes
func stripANSI(s string) string {
	ansiRe := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRe.ReplaceAllString(s, "")
}
