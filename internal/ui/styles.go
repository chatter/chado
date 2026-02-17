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

// Color codes (ANSI 256).
const (
	PrimaryColorCode   = "#808080" // Gray
	SecondaryColorCode = "241"     // Gray
	AccentColorCode    = "#30c9b0" // Cyan
)

// Styles holds all lipgloss styles for the application, constructed from a detected color profile.
type Styles struct {
	Panel        lipgloss.Style
	FocusedPanel lipgloss.Style
	Title        lipgloss.Style
	FocusedTitle lipgloss.Style
	StatusBar    lipgloss.Style
	Version      lipgloss.Style
	Selected     lipgloss.Style
	Dim          lipgloss.Style
	ShortCode    lipgloss.Style

	// Border color blends for panel focus animation.
	unfocusedBorderBlend []color.Color
	focusedBorderBlend   []color.Color
}

// NewStyles creates the application styles using the detected terminal color profile.
func NewStyles() *Styles {
	profile := colorprofile.Detect(os.Stdout, os.Environ())

	complete := func(hex string) color.Color {
		return profile.Convert(lipgloss.Color(hex))
	}

	primary := complete(PrimaryColorCode)
	secondary := lipgloss.Color(SecondaryColorCode)
	accent := complete(AccentColorCode)

	unfocusedBlend := []color.Color{
		primary,
		complete("#454545"),
		primary,
		complete("#3d3d3d"),
		primary,
	}

	focusedBlend := []color.Color{
		accent,
		complete("#0d4d44"),
		accent,
		complete("#1e1e1e"),
		accent,
	}

	panel := lipgloss.
		NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForegroundBlend(unfocusedBlend...)

	return &Styles{
		Panel: panel,
		FocusedPanel: lipgloss.
			NewStyle().
			BorderForegroundBlend(focusedBlend...).
			Inherit(panel),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(primary).
			Padding(0, 1),
		FocusedTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Padding(0, 1),

		StatusBar: lipgloss.NewStyle().
			Foreground(secondary),
		Version: lipgloss.NewStyle().
			Foreground(secondary).
			Align(lipgloss.Right),

		Selected: lipgloss.NewStyle().
			Bold(true),
		Dim: lipgloss.NewStyle().
			Foreground(secondary),
		ShortCode: lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")). // Bright magenta - matches jj
			Bold(true).
			Inline(true),

		unfocusedBorderBlend: unfocusedBlend,
		focusedBorderBlend:   focusedBlend,
	}
}

// AnimatedFocusBorderStyle returns the panel style with the focus border animation at the given phase.
func (s *Styles) AnimatedFocusBorderStyle(phase float64, width, height int) lipgloss.Style {
	perimeter := borderSidesPerDimension*width + borderSidesPerDimension*height
	offset := int(phase * float64(perimeter))

	return lipgloss.NewStyle().
		Inherit(s.Panel).
		BorderForegroundBlend(s.RotatedFocusedBorderBlend(0)...).
		BorderForegroundBlendOffset(offset)
}

// RotatedFocusedBorderBlend returns focusedBorderBlend rotated by phase (0..1 = one full wrap).
func (s *Styles) RotatedFocusedBorderBlend(phase float64) []color.Color {
	const blendCount = 5
	if blendCount == 0 {
		return s.focusedBorderBlend
	}

	offset := int(phase*float64(blendCount)) % blendCount
	if offset < 0 {
		offset += blendCount
	}

	out := make([]color.Color, blendCount)
	for i := range blendCount {
		out[i] = s.focusedBorderBlend[(offset+i)%blendCount]
	}

	return out
}

// PanelTitle returns a formatted panel title with optional focus indicator.
func (s *Styles) PanelTitle(num int, title string, focused bool) string {
	titleText := "[" + string(rune('0'+num)) + "] " + title

	if focused {
		return s.FocusedTitle.Render(titleText)
	}

	return s.Title.Render(titleText)
}
