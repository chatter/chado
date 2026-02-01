package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors
var (
	primaryColor   = lipgloss.Color("62")  // Purple
	secondaryColor = lipgloss.Color("241") // Gray
	accentColor    = lipgloss.Color("86")  // Cyan
	borderColor    = lipgloss.Color("240") // Dark gray
	focusBorder    = lipgloss.Color("62")  // Purple for focused
)

// Styles for the application
var (
	// Panel styles
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor)

	FocusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(focusBorder)

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1)

	FocusedTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor).
				Padding(0, 1)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	VersionStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Align(lipgloss.Right)

	// Selected item
	SelectedStyle = lipgloss.NewStyle().
			Bold(true)

	// Dim style for non-focused content
	DimStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	// ShortCodeStyle for the unique prefix of change IDs (matches jj's bright magenta)
	ShortCodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))
)

// PanelTitle returns a formatted panel title with optional focus indicator
func PanelTitle(num int, title string, focused bool) string {
	prefix := ""
	if focused {
		prefix = "● "
	}
	titleText := prefix + "[" + string(rune('0'+num)) + "] " + title

	if focused {
		return FocusedTitleStyle.Render(titleText)
	}
	return TitleStyle.Render(titleText)
}
