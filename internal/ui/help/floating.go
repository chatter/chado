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
			Padding(0, 1),
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

	// Calculate max inner dimensions from set size
	frameWidth := f.borderStyle.GetHorizontalFrameSize()
	frameHeight := f.borderStyle.GetVerticalFrameSize()
	maxInnerWidth := f.width - frameWidth
	maxInnerHeight := f.height - frameHeight

	if maxInnerWidth < 10 || maxInnerHeight < 3 {
		return f.borderStyle.Render("...")
	}

	// Group bindings by category
	groups := f.groupByCategory()
	if len(groups) == 0 {
		return f.borderStyle.Render("No keybindings")
	}

	// Build column-based content, respecting max width
	content, contentWidth, _ := f.renderColumns(groups, maxInnerWidth)

	// Build title and footer
	title := f.titleStyle.Render("Help")
	footer := f.footerStyle.Render("? to close")

	titleWidth := lipgloss.Width(title)
	footerWidth := lipgloss.Width(footer)

	// Modal width = max of title, content, footer (capped by maxInnerWidth)
	innerWidth := min(max(titleWidth, contentWidth, footerWidth), maxInnerWidth)

	// Right-align footer
	if footerWidth < innerWidth {
		footer = strings.Repeat(" ", innerWidth-footerWidth) + footer
	}

	// Calculate available height for content
	// title (1) + blank (1) + content + blank (1) + footer (1) = 4 + content
	availableContentHeight := maxInnerHeight - 4

	// Truncate content if needed
	contentLines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(contentLines) > availableContentHeight {
		contentLines = contentLines[:availableContentHeight]
	}
	content = strings.Join(contentLines, "\n")

	// Combine vertically with spacing
	fullContent := lipgloss.JoinVertical(lipgloss.Left, title, "", content, "", footer)

	return f.borderStyle.Render(fullContent)
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

// column represents a category rendered as a column
type column struct {
	lines  []string
	width  int
	height int
}

// renderColumns renders categories in a flowing column layout.
// Categories wrap to the next row if they don't fit horizontally.
// Returns the rendered content, its width, and its height.
func (f *FloatingHelp) renderColumns(groups map[Category][]HelpBinding, maxWidth int) (string, int, int) {
	if len(groups) == 0 {
		return "No keybindings available", 24, 1
	}

	// Build all category columns
	allColumns := f.buildColumns(groups)
	if len(allColumns) == 0 {
		return "No keybindings available", 24, 1
	}

	columnGap := "    " // 4 spaces between columns
	gapWidth := lipgloss.Width(columnGap)

	// Arrange columns into rows based on available width
	var rows [][]column
	var currentRow []column
	currentRowWidth := 0

	for _, col := range allColumns {
		needed := col.width
		if len(currentRow) > 0 {
			needed += gapWidth
		}

		// Does this column fit in the current row?
		if currentRowWidth+needed > maxWidth && len(currentRow) > 0 {
			// Start a new row
			rows = append(rows, currentRow)
			currentRow = []column{col}
			currentRowWidth = col.width
		} else {
			currentRow = append(currentRow, col)
			currentRowWidth += needed
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	// Render each row
	var renderedRows []string
	totalHeight := 0
	maxRowWidth := 0

	for _, row := range rows {
		rowStr, rowWidth, rowHeight := f.renderRow(row, columnGap)
		renderedRows = append(renderedRows, rowStr)
		totalHeight += rowHeight
		if rowWidth > maxRowWidth {
			maxRowWidth = rowWidth
		}
	}

	// Join rows with a blank line between them
	result := strings.Join(renderedRows, "\n\n")
	totalHeight += len(rows) - 1 // blank lines between rows

	return result, maxRowWidth, totalHeight
}

// buildColumns creates column structures for each category
func (f *FloatingHelp) buildColumns(groups map[Category][]HelpBinding) []column {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62")).Underline(true)

	var columns []column

	for _, cat := range categoryOrder {
		bindings, ok := groups[cat]
		if !ok || len(bindings) == 0 {
			continue
		}

		// Calculate max key width for this category
		maxKeyWidth := 0
		for _, hb := range bindings {
			w := lipgloss.Width(hb.Binding.Help().Key)
			if w > maxKeyWidth {
				maxKeyWidth = w
			}
		}

		// Build column lines
		var lines []string
		lines = append(lines, headerStyle.Render(string(cat)))
		colWidth := lipgloss.Width(string(cat))

		for _, hb := range bindings {
			help := hb.Binding.Help()
			key := keyStyle.Width(maxKeyWidth + 2).Render(help.Key) // +2 for breathing room
			desc := descStyle.Render(help.Desc)
			line := key + desc
			lines = append(lines, line)

			if w := lipgloss.Width(line); w > colWidth {
				colWidth = w
			}
		}

		columns = append(columns, column{
			lines:  lines,
			width:  colWidth,
			height: len(lines),
		})
	}

	return columns
}

// renderRow renders a single row of columns side by side
func (f *FloatingHelp) renderRow(row []column, gap string) (string, int, int) {
	if len(row) == 0 {
		return "", 0, 0
	}

	gapWidth := lipgloss.Width(gap)

	// Find max height in this row
	maxHeight := 0
	for _, col := range row {
		if col.height > maxHeight {
			maxHeight = col.height
		}
	}

	// Pad each column to same height and its own width
	var paddedColumns []string
	for _, col := range row {
		lines := make([]string, len(col.lines))
		copy(lines, col.lines)

		// Pad to max height
		for len(lines) < maxHeight {
			lines = append(lines, "")
		}

		// Pad each line to column width
		for j, line := range lines {
			lineWidth := lipgloss.Width(line)
			if lineWidth < col.width {
				lines[j] = line + strings.Repeat(" ", col.width-lineWidth)
			}
		}

		paddedColumns = append(paddedColumns, strings.Join(lines, "\n"))
	}

	// Join columns horizontally
	result := lipgloss.JoinHorizontal(lipgloss.Top, paddedColumns[0])
	for i := 1; i < len(paddedColumns); i++ {
		result = lipgloss.JoinHorizontal(lipgloss.Top, result, gap, paddedColumns[i])
	}

	// Calculate total width
	totalWidth := 0
	for i, col := range row {
		totalWidth += col.width
		if i < len(row)-1 {
			totalWidth += gapWidth
		}
	}

	return result, totalWidth, maxHeight
}
