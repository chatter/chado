// Package ui provides shared panel components, styles, and layout.
// constants for the TUI.
package ui

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

const (
	// PanelBorderWidth is the horizontal space consumed by left + right borders.
	PanelBorderWidth = 2

	// PanelBorderHeight is the vertical space consumed by top + bottom borders.
	PanelBorderHeight = 2

	// PanelTitleHeight is the vertical space consumed by the title row.
	PanelTitleHeight = 1

	// PanelChromeHeight is total vertical chrome: borders + title.
	PanelChromeHeight = PanelBorderHeight + PanelTitleHeight

	// borderSidesPerDimension is the rectangle side count per axis (used
	// in the perimeter formula: 2*width + 2*height).
	borderSidesPerDimension = 2

	// PanelOrderPrimary is the highest help binding display priority.
	PanelOrderPrimary = 1
	// PanelOrderSecondary is the next-highest display priority.
	PanelOrderSecondary = 2

	// ScrollPadding is the number of lines of context kept visible below the
	// cursor when scrolling the viewport to keep the cursor in view.
	ScrollPadding = 2
)

// AnimatedFocusBorderStyle returns the panel style with the focus border animation at the given phase.
// Use when a panel is focused and the border wrap animation is running.
func AnimatedFocusBorderStyle(phase float64, width, height int) lipgloss.Style {
	perimeter := borderSidesPerDimension*width + borderSidesPerDimension*height
	offset := int(phase * float64(perimeter))

	return lipgloss.NewStyle().
		Inherit(PanelStyle).
		BorderForegroundBlend(RotatedFocusedBorderBlend(0)...).
		BorderForegroundBlendOffset(offset)
}

// RotatedFocusedBorderBlend returns focusedBorderBlend rotated by phase (0..1 = one full wrap).
func RotatedFocusedBorderBlend(phase float64) []color.Color {
	const blendCount = 5
	if blendCount == 0 {
		return focusedBorderBlend
	}

	offset := int(phase*float64(blendCount)) % blendCount
	if offset < 0 {
		offset += blendCount
	}

	out := make([]color.Color, blendCount)
	for i := range blendCount {
		out[i] = focusedBorderBlend[(offset+i)%blendCount]
	}

	return out
}

// colorProfile is detected once for terminal color capability (ANSI/256/truecolor).
var colorProfile = colorprofile.Detect(os.Stdout, os.Environ())

// completeColor converts a hex color to a terminal-appropriate color (profile-aware, lipgloss v2 Complete).
func completeColor(hex string) color.Color {
	return colorProfile.Convert(lipgloss.Color(hex))
}

// Color codes (ANSI 256).
const (
	PrimaryColorCode   = "#808080" // Gray
	SecondaryColorCode = "241"     // Gray
	AccentColorCode    = "#30c9b0" // Cyan
)

// Colors (for lipgloss styles).
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

// Styles for the application.
var (
	PanelStyle = lipgloss.
			NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForegroundBlend(unfocusedBorderBlend...)

	FocusedPanelStyle = lipgloss.
				NewStyle().
				BorderForegroundBlend(focusedBorderBlend...).
				Inherit(PanelStyle)

	// TitleStyle renders panel titles.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1)

	FocusedTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor).
				Padding(0, 1)

	// StatusBarStyle renders the bottom status bar text.
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	VersionStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Align(lipgloss.Right)

	// SelectedStyle highlights the currently selected item.
	SelectedStyle = lipgloss.NewStyle().
			Bold(true)

	// DimStyle de-emphasises non-focused content.
	DimStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	// ShortCodeStyle for the unique prefix of change IDs (matches jj's default).
	ShortCodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")). // Bright magenta - matches jj
			Bold(true).
			Inline(true)
)

// PanelTitle returns a formatted panel title with optional focus indicator.
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
