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
}

// NewFloatingHelp creates a new floating help modal.
func NewFloatingHelp() *FloatingHelp {
	return &FloatingHelp{}
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

	// Group bindings by category
	groups := f.groupByCategory()

	// Build content
	content := f.renderContent(groups)

	// Style the modal
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Align(lipgloss.Right)

	// Calculate inner dimensions
	innerWidth := f.width - 6  // border (2) + padding (4)
	innerHeight := f.height - 4 // border (2) + padding (2)

	if innerWidth < 10 || innerHeight < 3 {
		return borderStyle.Width(f.width - 2).Height(f.height - 2).Render("...")
	}

	// Build the modal content
	title := titleStyle.Render("Help")
	footer := footerStyle.Width(innerWidth).Render("? to close")

	// Calculate available height for bindings
	contentHeight := innerHeight - 2 // title + footer

	// Truncate content if needed
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
	}
	content = strings.Join(contentLines, "\n")

	// Pad content to fill space
	for len(strings.Split(content, "\n")) < contentHeight {
		content += "\n"
	}

	fullContent := title + "\n" + content + footer

	return borderStyle.Width(innerWidth).Render(fullContent)
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
func (f *FloatingHelp) renderContent(groups map[Category][]HelpBinding) string {
	if len(groups) == 0 {
		return "No keybindings available"
	}

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Width(10)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginTop(1)

	var lines []string

	// Render categories in defined order
	for _, cat := range categoryOrder {
		bindings, ok := groups[cat]
		if !ok || len(bindings) == 0 {
			continue
		}

		// Category header
		if len(lines) > 0 {
			lines = append(lines, "") // blank line between categories
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
