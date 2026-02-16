package help

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// StatusBar renders a minimal status line: key hints and right-aligned version.
type StatusBar struct {
	width   int
	version string

	// Styles
	keyStyle  lipgloss.Style
	descStyle lipgloss.Style
	sepStyle  lipgloss.Style
}

// NewStatusBar creates a new status bar that displays the given version string.
func NewStatusBar(version string) *StatusBar {
	return &StatusBar{
		version:   version,
		keyStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#999999")),
		descStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#777777")),
		sepStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")),
	}
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

	help := s.keyStyle.Render("?") + " " + s.descStyle.Render("help")
	quit := s.keyStyle.Render("q") + " " + s.descStyle.Render("quit")
	sep := s.sepStyle.Render(" • ")

	left := help + sep + quit
	leftWidth := lipgloss.Width(left)

	// If hints + version don't fit, drop the version.
	const minGap = 1

	version := s.version
	versionWidth := lipgloss.Width(version)

	if leftWidth+minGap+versionWidth > s.width {
		padding := max(s.width-leftWidth, 0)

		return strings.Repeat(" ", padding) + left
	}

	padding := s.width - leftWidth - versionWidth

	return left + strings.Repeat(" ", padding) + version
}
