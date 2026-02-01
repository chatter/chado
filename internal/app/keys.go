package app

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Category represents a logical grouping of keybindings for help display
type Category string

const (
	CategoryNavigation Category = "Navigation"
	CategoryActions    Category = "Actions"
	CategoryDiff       Category = "Diff"
)

// Action is a function that executes a keybinding's behavior
type Action func(m *Model) (Model, tea.Cmd)

// HelpBinding combines a key binding with its category, display order, and action
type HelpBinding struct {
	Binding  key.Binding
	Category Category
	Order    int    // lower = higher priority for inline status bar
	Action   Action // nil = display-only (no action)
}

// dispatchKey iterates through bindings and executes the first matching action
// Returns nil, nil if no binding matches
func dispatchKey(m *Model, msg tea.KeyMsg, bindings []HelpBinding) (*Model, tea.Cmd) {
	for _, hb := range bindings {
		if key.Matches(msg, hb.Binding) && hb.Action != nil {
			newModel, cmd := hb.Action(m)
			return &newModel, cmd
		}
	}
	return nil, nil
}

// KeyMap defines the key bindings for the application
type KeyMap struct {
	// Navigation between panes
	FocusPane0 key.Binding
	FocusPane1 key.Binding
	NextPane   key.Binding
	PrevPane   key.Binding
	Left       key.Binding
	Right      key.Binding

	// Navigation within pane
	Up     key.Binding
	Down   key.Binding
	Top    key.Binding
	Bottom key.Binding

	// Actions
	Enter key.Binding
	Back  key.Binding
	Quit  key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		FocusPane0: key.NewBinding(
			key.WithKeys("0"),
			key.WithHelp("0", "focus diff pane"),
		),
		FocusPane1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "focus log pane"),
		),
		NextPane: key.NewBinding(
			key.WithKeys("tab", "l"),
			key.WithHelp("tab/l", "next pane"),
		),
		PrevPane: key.NewBinding(
			key.WithKeys("shift+tab", "h"),
			key.WithHelp("shift+tab/h", "previous pane"),
		),
		Left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "previous pane"),
		),
		Right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "next pane"),
		),
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "go to top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to bottom"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "drill down / select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "go back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}
