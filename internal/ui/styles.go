package ui

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

// AnimatedFocusBorderStyle returns the panel style with the focus border animation at the given phase.
// Use when a panel is focused and the border wrap animation is running.
func AnimatedFocusBorderStyle(phase float64, width, height int) lipgloss.Style {
	perimeter := 2*width + 2*height
	offset := int(phase * float64(perimeter))
	return lipgloss.NewStyle().
		Inherit(PanelStyle).
		BorderForegroundBlend(RotatedFocusedBorderBlend(0)...).
		BorderForegroundBlendOffset(offset)
}

// RotatedFocusedBorderBlend returns focusedBorderBlend rotated by phase (0..1 = one full wrap).
func RotatedFocusedBorderBlend(phase float64) []color.Color {
	const n = 5
	if n == 0 {
		return focusedBorderBlend
	}
	offset := int(phase*float64(n)) % n
	if offset < 0 {
		offset += n
	}
	out := make([]color.Color, n)
	for i := 0; i < n; i++ {
		out[i] = focusedBorderBlend[(offset+i)%n]
	}
	return out
}

// colorProfile is detected once for terminal color capability (ANSI/256/truecolor).
var colorProfile = colorprofile.Detect(os.Stdout, os.Environ())

// completeColor converts a hex color to a terminal-appropriate color (profile-aware, lipgloss v2 Complete).
func completeColor(hex string) color.Color {
	return colorProfile.Convert(lipgloss.Color(hex))
}

// Color codes (ANSI 256)
const (
	PrimaryColorCode   = "#808080" // Gray
	SecondaryColorCode = "241"     // Gray
	AccentColorCode    = "#30c9b0" // Cyan
)

// Colors (for lipgloss styles)
var (
	primaryColor   = completeColor(PrimaryColorCode)
	secondaryColor = lipgloss.Color(SecondaryColorCode)
	accentColor    = completeColor(AccentColorCode)

	unfocusedBorderBlend = []color.Color{
		primaryColor,
		completeColor("#454545"),
		primaryColor,
		completeColor("#3d3d3d"),
		primaryColor,
	}

	focusedBorderBlend = []color.Color{
		accentColor,
		completeColor("#0d4d44"),
		accentColor,
		completeColor("#1e1e1e"),
		accentColor,
	}
)

// Styles for the application
var (
	PanelStyle = lipgloss.
			NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForegroundBlend(unfocusedBorderBlend...)

	FocusedPanelStyle = lipgloss.
				NewStyle().
				BorderForegroundBlend(focusedBorderBlend...).
				Inherit(PanelStyle)

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

	// ShortCodeStyle for the unique prefix of change IDs (matches jj's default)
	ShortCodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")). // Bright magenta - matches jj
			Bold(true).
			Inline(true)
)

// PanelTitle returns a formatted panel title with optional focus indicator
func PanelTitle(num int, title string, focused bool) string {
	prefix := ""
	// if focused {
	// 	prefix = "●"
	// }
	titleText := prefix + "[" + string(rune('0'+num)) + "] " + title

	if focused {
		return FocusedTitleStyle.Render(titleText)
	}
	return TitleStyle.Render(titleText)
}
