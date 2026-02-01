package help

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StatusBar renders keybinding hints in the status bar.
type StatusBar struct {
	width    int
	version  string
	bindings []HelpBinding

	// Styles
	keyStyle  lipgloss.Style
	descStyle lipgloss.Style
	sepStyle  lipgloss.Style
}

// NewStatusBar creates a new status bar help component.
func NewStatusBar(version string) *StatusBar {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#666666",
		Dark:  "#999999",
	})

	descStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#888888",
		Dark:  "#777777",
	})

	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#AAAAAA",
		Dark:  "#555555",
	})

	return &StatusBar{
		version:   version,
		keyStyle:  keyStyle,
		descStyle: descStyle,
		sepStyle:  sepStyle,
	}
}

// SetBindings sets the keybindings to display.
func (s *StatusBar) SetBindings(bindings []HelpBinding) {
	s.bindings = bindings
}

// SetWidth sets the available width for rendering.
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

// View renders the status bar.
func (s *StatusBar) View() string {
	if s.width <= 0 {
		return ""
	}

	// Sort bindings by order
	sorted := s.sortedBindings()

	// Build help text, respecting width
	separator := s.sepStyle.Render(" • ")
	separatorWidth := lipgloss.Width(separator)

	// Reserve space for version
	versionText := s.version
	versionWidth := len(versionText)

	// Available width for bindings (leave space for version + padding)
	availableWidth := s.width - versionWidth - 2 // 2 for minimum padding

	var helpParts []string
	currentWidth := 0
	ellipsis := "…"
	ellipsisWidth := lipgloss.Width(ellipsis)

	for i, hb := range sorted {
		if !hb.Binding.Enabled() {
			continue
		}

		help := hb.Binding.Help()
		part := s.keyStyle.Render(help.Key) + " " + s.descStyle.Render(help.Desc)
		partWidth := lipgloss.Width(part)

		// Calculate width with separator
		widthWithSep := partWidth
		if len(helpParts) > 0 {
			widthWithSep += separatorWidth
		}

		// Check if adding this part would exceed available width
		// Also need to reserve space for ellipsis if there are more bindings
		hasMore := i < len(sorted)-1
		reserveForEllipsis := 0
		if hasMore {
			reserveForEllipsis = ellipsisWidth + separatorWidth
		}

		if currentWidth+widthWithSep+reserveForEllipsis > availableWidth {
			// Can't fit this one, add ellipsis and stop
			if len(helpParts) > 0 {
				helpParts = append(helpParts, ellipsis)
			}
			break
		}

		helpParts = append(helpParts, part)
		currentWidth += widthWithSep
	}

	// Join help parts with separator
	helpText := strings.Join(helpParts, separator)
	helpWidth := lipgloss.Width(helpText)

	// Calculate padding between help and version
	padding := s.width - helpWidth - versionWidth
	if padding < 1 {
		padding = 1
	}

	return helpText + strings.Repeat(" ", padding) + versionText
}

// sortedBindings returns bindings sorted by order (ascending).
func (s *StatusBar) sortedBindings() []HelpBinding {
	if len(s.bindings) == 0 {
		return nil
	}

	sorted := make([]HelpBinding, len(s.bindings))
	copy(sorted, s.bindings)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Order < sorted[j].Order
	})

	return sorted
}
