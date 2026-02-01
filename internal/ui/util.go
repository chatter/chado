package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ReplaceResetWithColor replaces ANSI reset codes with a specific foreground color.
// This allows nested styles to restore the outer color instead of resetting completely.
//
// Example: If you have styled text that ends with a reset (\x1b[0m), this replaces
// that reset with a foreground color code, allowing the text that follows to
// continue with the specified color rather than falling back to terminal defaults.
func ReplaceResetWithColor(s string, color lipgloss.Color) string {
	colorCode := fmt.Sprintf("\x1b[38;5;%sm", string(color))
	return strings.ReplaceAll(s, "\x1b[0m", colorCode)
}
