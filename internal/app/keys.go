package app

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/chatter/chado/internal/ui/help"
)

// Action is a function that executes a keybinding's behavior
type Action func(m *Model) (Model, tea.Cmd)

// ActionBinding combines a display binding with its action for dispatch.
type ActionBinding struct {
	help.HelpBinding        // embedded for display (Binding, Category, Order)
	Action           Action // nil = display-only (no action)
}

// dispatchKey iterates through bindings and executes the first matching action.
// Returns nil, nil if no binding matches.
func dispatchKey(m *Model, msg tea.KeyMsg, bindings []ActionBinding) (*Model, tea.Cmd) {
	for _, ab := range bindings {
		if key.Matches(msg, ab.Binding) && ab.Action != nil {
			newModel, cmd := ab.Action(m)
			return &newModel, cmd
		}
	}
	return nil, nil
}

// ToHelpBindings extracts display-only bindings from action bindings.
func ToHelpBindings(abs []ActionBinding) []help.HelpBinding {
	result := make([]help.HelpBinding, len(abs))
	for i, ab := range abs {
		result[i] = ab.HelpBinding
	}
	return result
}

// KeyMap defines the key bindings for the application
type KeyMap struct {
	// Navigation between panes
	FocusPane0 key.Binding
	FocusPane1 key.Binding
	FocusPane2 key.Binding
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
	Enter    key.Binding
	Back     key.Binding
	Abandon  key.Binding
	Describe key.Binding
	Edit     key.Binding
	New      key.Binding
	Quit     key.Binding
	Help     key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		FocusPane0: key.NewBinding(
			key.WithKeys("0"),
			key.WithHelp("#", "focus pane"), // Combined display
		),
		FocusPane1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "focus pane"), // Hidden in help (duplicate)
		),
		FocusPane2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "focus pane"), // Hidden in help (duplicate)
		),
		NextPane: key.NewBinding(
			key.WithKeys("tab", "l", "right"),
			key.WithHelp("→/l/⇥", "next pane"),
		),
		PrevPane: key.NewBinding(
			key.WithKeys("shift+tab", "h", "left"),
			key.WithHelp("←/h/⇧⇥", "prev pane"),
		),
		Left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "prev pane"),
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
			key.WithHelp("gg", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("⏎", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("⎋", "back"),
		),
		Abandon: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "abandon"),
		),
		Describe: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "describe"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}
