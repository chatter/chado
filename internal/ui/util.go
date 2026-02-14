package ui

import (
	"fmt"
	"regexp"
	"strings"
)

// ReplaceResetWithColor replaces ANSI reset codes with a specific foreground color.
// This allows nested styles to restore the outer color instead of resetting completely.
//
// Example: If you have styled text that ends with a reset (\x1b[0m), this replaces
// that reset with a foreground color code, allowing the text that follows to
// continue with the specified color rather than falling back to terminal defaults.
// The color parameter should be an ANSI 256 color code (e.g. "241" for gray).
func ReplaceResetWithColor(s string, color string) string {
	colorCode := fmt.Sprintf("\x1b[38;5;%sm", color)
	return strings.ReplaceAll(s, "\x1b[0m", colorCode)
}

// StripANSI removes ANSI escape codes.
func StripANSI(s string) string {
	ansiRe := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRe.ReplaceAllString(s, "")
}
