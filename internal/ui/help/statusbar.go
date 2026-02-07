package help

import (
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
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
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#999999"))

	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))

	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

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

	// Dedupe bindings by description (keep first occurrence)
	seen := make(map[string]bool)
	var deduped []HelpBinding
	for _, hb := range s.bindings {
		if !hb.Binding.Enabled() {
			continue
		}
		desc := hb.Binding.Help().Desc
		if seen[desc] {
			continue
		}
		seen[desc] = true
		deduped = append(deduped, hb)
	}

	// Separate pinned and regular bindings
	var pinned, regular []HelpBinding
	for _, hb := range deduped {
		if hb.Pinned {
			pinned = append(pinned, hb)
		} else {
			regular = append(regular, hb)
		}
	}

	// Sort regular bindings by order
	sort.Slice(regular, func(i, j int) bool {
		return regular[i].Order < regular[j].Order
	})

	// Sort pinned bindings by order too
	sort.Slice(pinned, func(i, j int) bool {
		return pinned[i].Order < pinned[j].Order
	})

	// Build help text, respecting width
	separator := s.sepStyle.Render(" • ")
	separatorWidth := lipgloss.Width(separator)

	// Reserve space for version
	versionText := s.version
	versionWidth := len(versionText)

	// Calculate space needed for pinned bindings
	var pinnedParts []string
	pinnedWidth := 0
	for _, hb := range pinned {
		help := hb.Binding.Help()
		part := s.keyStyle.Render(help.Key) + " " + s.descStyle.Render(help.Desc)
		pinnedParts = append(pinnedParts, part)
		if pinnedWidth > 0 {
			pinnedWidth += separatorWidth
		}
		pinnedWidth += lipgloss.Width(part)
	}

	// Available width for regular bindings
	availableWidth := s.width - versionWidth - pinnedWidth - 4 // 4 for padding and separators

	var regularParts []string
	currentWidth := 0
	ellipsis := "…"
	ellipsisWidth := lipgloss.Width(ellipsis)

	for i, hb := range regular {
		help := hb.Binding.Help()
		part := s.keyStyle.Render(help.Key) + " " + s.descStyle.Render(help.Desc)
		partWidth := lipgloss.Width(part)

		// Calculate width with separator
		widthWithSep := partWidth
		if len(regularParts) > 0 {
			widthWithSep += separatorWidth
		}

		// Check if adding this part would exceed available width
		hasMore := i < len(regular)-1
		reserveForEllipsis := 0
		if hasMore {
			reserveForEllipsis = ellipsisWidth + separatorWidth
		}

		if currentWidth+widthWithSep+reserveForEllipsis > availableWidth {
			// Can't fit this one, add ellipsis and stop
			if len(regularParts) > 0 {
				regularParts = append(regularParts, ellipsis)
			}
			break
		}

		regularParts = append(regularParts, part)
		currentWidth += widthWithSep
	}

	// Combine regular and pinned parts
	var allParts []string
	allParts = append(allParts, regularParts...)
	allParts = append(allParts, pinnedParts...)

	// Join help parts with separator
	helpText := strings.Join(allParts, separator)
	helpWidth := lipgloss.Width(helpText)

	// Calculate padding between help and version
	padding := max(s.width-helpWidth-versionWidth, 1)

	return helpText + strings.Repeat(" ", padding) + versionText
}
