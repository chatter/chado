// Package help provides help display components for the TUI.
package help

import (
	"github.com/charmbracelet/bubbles/key"
)

// Category represents a logical grouping of keybindings for help display
type Category string

const (
	CategoryNavigation Category = "Navigation"
	CategoryActions    Category = "Actions"
	CategoryDiff       Category = "Diff"
)

// HelpBinding contains display information for a keybinding.
// This is the display-only version; app.ActionBinding adds the Action field.
type HelpBinding struct {
	Binding  key.Binding
	Category Category
	Order    int // lower = higher priority for inline status bar
}
