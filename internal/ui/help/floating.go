package help

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FloatingHelp renders a modal with all keybindings organized by category.
type FloatingHelp struct {
	width    int
	height   int
	bindings []HelpBinding

	// Styles (cached for frame size calculations)
	borderStyle lipgloss.Style
	titleStyle  lipgloss.Style
	footerStyle lipgloss.Style
}

// NewFloatingHelp creates a new floating help modal.
func NewFloatingHelp() *FloatingHelp {
	return &FloatingHelp{
		borderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2),
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")),
		footerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
	}
}

// SetSize sets the available size for the modal.
func (f *FloatingHelp) SetSize(width, height int) {
	f.width = width
	f.height = height
}

// SetBindings sets the keybindings to display.
func (f *FloatingHelp) SetBindings(bindings []HelpBinding) {
	f.bindings = bindings
}

// View renders the floating help modal.
func (f *FloatingHelp) View() string {
	if f.width <= 0 || f.height <= 0 {
		return ""
	}

	// Calculate inner dimensions using lipgloss frame sizes
	frameWidth := f.borderStyle.GetHorizontalFrameSize()
	frameHeight := f.borderStyle.GetVerticalFrameSize()

	innerWidth := f.width - frameWidth
	innerHeight := f.height - frameHeight

	// Minimum size check
	if innerWidth < 20 || innerHeight < 5 {
		return f.borderStyle.Width(max(innerWidth, 10)).Render("...")
	}

	// Group bindings by category
	groups := f.groupByCategory()

	// Build content with dynamic key width, constrained to innerWidth
	content := f.renderContent(groups, innerWidth)

	// Build the modal content
	title := f.titleStyle.Render("Help")
	footer := f.footerStyle.Render("? to close")

	// Calculate available height for bindings
	contentHeight := innerHeight - 2 // title line + footer line

	// Truncate content if needed
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
	}
	content = strings.Join(contentLines, "\n")

	// Build inner content: title + bindings
	upperContent := lipgloss.JoinVertical(lipgloss.Left, title, content)

	// Place footer at bottom-right of the inner area
	innerContent := lipgloss.Place(
		innerWidth, innerHeight,
		lipgloss.Left, lipgloss.Top,
		upperContent,
	)

	// Overlay footer at bottom-right
	// Split into lines, replace last line's end with footer
	lines := strings.Split(innerContent, "\n")
	if len(lines) > 0 {
		lastIdx := len(lines) - 1
		lastLine := lines[lastIdx]
		footerWidth := lipgloss.Width(footer)
		lineWidth := lipgloss.Width(lastLine)

		if lineWidth >= footerWidth {
			// Replace the rightmost characters with footer
			// We need to be careful with ANSI codes, so just pad and place
			padding := innerWidth - footerWidth
			if padding < 0 {
				padding = 0
			}
			lines[lastIdx] = strings.Repeat(" ", padding) + footer
		}
		innerContent = strings.Join(lines, "\n")
	}

	return f.borderStyle.Render(innerContent)
}

// categoryOrder defines the display order of categories
var categoryOrder = []Category{
	CategoryNavigation,
	CategoryActions,
	CategoryDiff,
}

// groupByCategory groups enabled bindings by their category.
func (f *FloatingHelp) groupByCategory() map[Category][]HelpBinding {
	groups := make(map[Category][]HelpBinding)

	for _, hb := range f.bindings {
		if !hb.Binding.Enabled() {
			continue
		}
		groups[hb.Category] = append(groups[hb.Category], hb)
	}

	// Sort bindings within each category by order
	for cat := range groups {
		sort.Slice(groups[cat], func(i, j int) bool {
			return groups[cat][i].Order < groups[cat][j].Order
		})
	}

	return groups
}

// renderContent renders the bindings in a column-flow layout.
func (f *FloatingHelp) renderContent(groups map[Category][]HelpBinding, availableWidth int) string {
	if len(groups) == 0 {
		return "No keybindings available"
	}

	// Calculate max key width dynamically
	maxKeyWidth := 0
	for _, hb := range f.bindings {
		if !hb.Binding.Enabled() {
			continue
		}
		w := lipgloss.Width(hb.Binding.Help().Key)
		if w > maxKeyWidth {
			maxKeyWidth = w
		}
	}

	// Key column: indent (2) + key + gap (2)
	keyColumnWidth := maxKeyWidth + 2
	descMaxWidth := availableWidth - 2 - keyColumnWidth // 2 for indent
	if descMaxWidth < 10 {
		descMaxWidth = 10
	}

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Width(keyColumnWidth)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		MaxWidth(descMaxWidth)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62"))

	var lines []string

	// Render categories in defined order
	for _, cat := range categoryOrder {
		bindings, ok := groups[cat]
		if !ok || len(bindings) == 0 {
			continue
		}

		// Category header (with blank line before, except first)
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, headerStyle.Render(string(cat)))

		// Bindings
		for _, hb := range bindings {
			help := hb.Binding.Help()
			line := "  " + keyStyle.Render(help.Key) + descStyle.Render(help.Desc)
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}
