package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// testAction creates an action that sends a specific message
func testAction(msg string) Action {
	return func(m *Model) (Model, tea.Cmd) {
		return *m, func() tea.Msg { return testMsg{msg} }
	}
}

type testMsg struct {
	value string
}

func TestDispatch_MatchesAndExecutes(t *testing.T) {
	bindings := []HelpBinding{
		{
			Binding:  key.NewBinding(key.WithKeys("a")),
			Category: CategoryNavigation,
			Action:   testAction("action-a"),
		},
		{
			Binding:  key.NewBinding(key.WithKeys("b")),
			Category: CategoryNavigation,
			Action:   testAction("action-b"),
		},
	}

	m := &Model{}
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}

	newModel, cmd := dispatchKey(m, keyMsg, bindings)
	if newModel == nil {
		t.Fatal("expected model to be returned")
	}

	if cmd == nil {
		t.Fatal("expected cmd to be returned")
	}

	msg := cmd()
	if tm, ok := msg.(testMsg); !ok || tm.value != "action-b" {
		t.Errorf("expected action-b, got %v", msg)
	}
}

func TestDispatch_NoMatchNoAction(t *testing.T) {
	bindings := []HelpBinding{
		{
			Binding:  key.NewBinding(key.WithKeys("a")),
			Category: CategoryNavigation,
			Action:   testAction("action-a"),
		},
	}

	m := &Model{}
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}}

	newModel, cmd := dispatchKey(m, keyMsg, bindings)
	if newModel != nil {
		t.Error("expected nil model for no match")
	}
	if cmd != nil {
		t.Error("expected nil cmd for no match")
	}
}

func TestDispatch_NilActionSkipped(t *testing.T) {
	bindings := []HelpBinding{
		{
			Binding:  key.NewBinding(key.WithKeys("a")),
			Category: CategoryNavigation,
			Action:   nil, // display-only binding
		},
		{
			Binding:  key.NewBinding(key.WithKeys("a")), // same key, but with action
			Category: CategoryNavigation,
			Action:   testAction("fallback"),
		},
	}

	m := &Model{}
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}

	newModel, cmd := dispatchKey(m, keyMsg, bindings)
	if newModel == nil || cmd == nil {
		t.Fatal("expected to fall through to second binding")
	}

	msg := cmd()
	if tm, ok := msg.(testMsg); !ok || tm.value != "fallback" {
		t.Errorf("expected fallback action, got %v", msg)
	}
}

func TestDispatch_FirstMatchWins(t *testing.T) {
	bindings := []HelpBinding{
		{
			Binding:  key.NewBinding(key.WithKeys("a")),
			Category: CategoryNavigation,
			Action:   testAction("first"),
		},
		{
			Binding:  key.NewBinding(key.WithKeys("a")), // same key
			Category: CategoryNavigation,
			Action:   testAction("second"),
		},
	}

	m := &Model{}
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}

	_, cmd := dispatchKey(m, keyMsg, bindings)
	if cmd == nil {
		t.Fatal("expected cmd")
	}

	msg := cmd()
	if tm, ok := msg.(testMsg); !ok || tm.value != "first" {
		t.Errorf("expected first action to win, got %v", msg)
	}
}

func TestDispatch_DisabledBindingSkipped(t *testing.T) {
	disabledBinding := key.NewBinding(key.WithKeys("a"))
	disabledBinding.SetEnabled(false)

	bindings := []HelpBinding{
		{
			Binding:  disabledBinding,
			Category: CategoryNavigation,
			Action:   testAction("disabled"),
		},
		{
			Binding:  key.NewBinding(key.WithKeys("a")), // same key, enabled
			Category: CategoryNavigation,
			Action:   testAction("enabled"),
		},
	}

	m := &Model{}
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}

	_, cmd := dispatchKey(m, keyMsg, bindings)
	if cmd == nil {
		t.Fatal("expected cmd")
	}

	msg := cmd()
	if tm, ok := msg.(testMsg); !ok || tm.value != "enabled" {
		t.Errorf("expected enabled action, got %v", msg)
	}
}
