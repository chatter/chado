package app

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/chatter/chado/internal/ui/help"
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
	bindings := []ActionBinding{
		{
			Binding: help.Binding{
				Key:      key.NewBinding(key.WithKeys("a")),
				Category: help.CategoryNavigation,
			},
			Action: testAction("action-a"),
		},
		{
			Binding: help.Binding{
				Key:      key.NewBinding(key.WithKeys("b")),
				Category: help.CategoryNavigation,
			},
			Action: testAction("action-b"),
		},
	}

	m := &Model{}
	keyMsg := tea.KeyPressMsg(tea.Key{Code: 'b'})

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
	bindings := []ActionBinding{
		{
			Binding: help.Binding{
				Key:      key.NewBinding(key.WithKeys("a")),
				Category: help.CategoryNavigation,
			},
			Action: testAction("action-a"),
		},
	}

	m := &Model{}
	keyMsg := tea.KeyPressMsg(tea.Key{Code: 'z'})

	newModel, cmd := dispatchKey(m, keyMsg, bindings)
	if newModel != nil {
		t.Error("expected nil model for no match")
	}
	if cmd != nil {
		t.Error("expected nil cmd for no match")
	}
}

func TestDispatch_NilActionSkipped(t *testing.T) {
	bindings := []ActionBinding{
		{
			Binding: help.Binding{
				Key:      key.NewBinding(key.WithKeys("a")),
				Category: help.CategoryNavigation,
			},
			Action: nil, // display-only binding
		},
		{
			Binding: help.Binding{
				Key:      key.NewBinding(key.WithKeys("a")), // same key, but with action
				Category: help.CategoryNavigation,
			},
			Action: testAction("fallback"),
		},
	}

	m := &Model{}
	keyMsg := tea.KeyPressMsg(tea.Key{Code: 'a'})

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
	bindings := []ActionBinding{
		{
			Binding: help.Binding{
				Key:      key.NewBinding(key.WithKeys("a")),
				Category: help.CategoryNavigation,
			},
			Action: testAction("first"),
		},
		{
			Binding: help.Binding{
				Key:      key.NewBinding(key.WithKeys("a")), // same key
				Category: help.CategoryNavigation,
			},
			Action: testAction("second"),
		},
	}

	m := &Model{}
	keyMsg := tea.KeyPressMsg(tea.Key{Code: 'a'})

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

	bindings := []ActionBinding{
		{
			Binding: help.Binding{
				Key:      disabledBinding,
				Category: help.CategoryNavigation,
			},
			Action: testAction("disabled"),
		},
		{
			Binding: help.Binding{
				Key:      key.NewBinding(key.WithKeys("a")), // same key, enabled
				Category: help.CategoryNavigation,
			},
			Action: testAction("enabled"),
		},
	}

	m := &Model{}
	keyMsg := tea.KeyPressMsg(tea.Key{Code: 'a'})

	_, cmd := dispatchKey(m, keyMsg, bindings)
	if cmd == nil {
		t.Fatal("expected cmd")
	}

	msg := cmd()
	if tm, ok := msg.(testMsg); !ok || tm.value != "enabled" {
		t.Errorf("expected enabled action, got %v", msg)
	}
}

// =============================================================================
// Evolog Integration Tests
// =============================================================================

func TestEvoLogLoadedMsg_TypeExists(t *testing.T) {
	// This test verifies the evoLogLoadedMsg type exists.
	// It will fail to compile until evoLogLoadedMsg is implemented.
	msg := evoLogLoadedMsg{
		changeID:  "testchange",
		shortCode: "tes",
		raw:       "@ aaaaaaaaaaaa",
	}

	if msg.changeID != "testchange" {
		t.Errorf("expected changeID 'testchange', got '%s'", msg.changeID)
	}
	if msg.shortCode != "tes" {
		t.Errorf("expected shortCode 'tes', got '%s'", msg.shortCode)
	}
}

func TestModel_LoadEvoLog_MethodExists(t *testing.T) {
	// This test verifies the loadEvoLog method exists on Model.
	// It will fail to compile until loadEvoLog is implemented.
	m := &Model{}

	// loadEvoLog should accept changeID and shortCode, return tea.Cmd
	cmd := m.loadEvoLog("testchange", "tes")

	// We can't really test the cmd without a proper setup, but method should exist
	_ = cmd
}

// =============================================================================
// Describe Tests
// =============================================================================

func TestDescribeCompleteMsg_TypeExists(t *testing.T) {
	// This test verifies the describeCompleteMsg type exists.
	msg := describeCompleteMsg{
		changeID: "testchange",
	}

	if msg.changeID != "testchange" {
		t.Errorf("expected changeID 'testchange', got '%s'", msg.changeID)
	}
}

func TestModel_RunDescribe_MethodExists(t *testing.T) {
	// This test verifies the runDescribe method exists on Model.
	m := &Model{}

	// runDescribe should accept changeID and message, return tea.Cmd
	cmd := m.runDescribe("testchange", "new description")

	// We can't really test the cmd without a proper setup, but method should exist
	_ = cmd
}

func TestModel_ActionDescribe_MethodExists(t *testing.T) {
	// This test verifies the actionDescribe method exists on Model.
	m := &Model{
		keys: DefaultKeyMap(),
	}

	// actionDescribe should return Model and tea.Cmd
	newModel, cmd := m.actionDescribe()

	// Without proper setup, should return unchanged model and nil cmd
	// (no selected change)
	_ = newModel
	_ = cmd
}

func TestDispatch_DescribeBinding(t *testing.T) {
	// Test that 'd' key is bound to describe action
	m := &Model{
		keys: DefaultKeyMap(),
	}

	bindings := m.globalBindings()

	// Find the describe binding
	found := false
	for _, ab := range bindings {
		if key.Matches(tea.KeyPressMsg(tea.Key{Code: 'd'}), ab.Key) {
			found = true
			if ab.Action == nil {
				t.Error("describe binding should have an action")
			}
			break
		}
	}

	if !found {
		t.Error("'d' key should be bound to describe action")
	}
}

func TestModel_EditModeState(t *testing.T) {
	// Test that editMode field exists and works
	m := &Model{}

	if m.editMode {
		t.Error("editMode should be false by default")
	}

	m.editMode = true
	if !m.editMode {
		t.Error("editMode should be true after setting")
	}
}

// =============================================================================
// Edit (jj edit) Tests
// =============================================================================

func TestEditCompleteMsg_TypeExists(t *testing.T) {
	// This test verifies the editCompleteMsg type exists.
	msg := editCompleteMsg{
		changeID: "testchange",
	}

	if msg.changeID != "testchange" {
		t.Errorf("expected changeID 'testchange', got '%s'", msg.changeID)
	}
}

func TestModel_RunEdit_MethodExists(t *testing.T) {
	// This test verifies the runEdit method exists on Model.
	m := &Model{}

	// runEdit should accept changeID, return tea.Cmd
	cmd := m.runEdit("testchange")

	// We can't really test the cmd without a proper setup, but method should exist
	_ = cmd
}

func TestModel_ActionEdit_MethodExists(t *testing.T) {
	// This test verifies the actionEdit method exists on Model.
	m := &Model{
		keys: DefaultKeyMap(),
	}

	// actionEdit should return Model and tea.Cmd
	newModel, cmd := m.actionEdit()

	// Without proper setup, should return unchanged model and nil cmd
	// (no selected change)
	_ = newModel
	_ = cmd
}

func TestDispatch_EditBinding(t *testing.T) {
	// Test that 'e' key is bound to edit action
	m := &Model{
		keys: DefaultKeyMap(),
	}

	bindings := m.globalBindings()

	// Find the edit binding
	found := false
	for _, ab := range bindings {
		if key.Matches(tea.KeyPressMsg(tea.Key{Code: 'e'}), ab.Key) {
			found = true
			if ab.Action == nil {
				t.Error("edit binding should have an action")
			}
			break
		}
	}

	if !found {
		t.Error("'e' key should be bound to edit action")
	}
}

// =============================================================================
// New (jj new) Tests
// =============================================================================

func TestNewCompleteMsg_TypeExists(t *testing.T) {
	// This test verifies the newCompleteMsg type exists.
	msg := newCompleteMsg{}
	_ = msg
}

func TestModel_RunNew_MethodExists(t *testing.T) {
	// This test verifies the runNew method exists on Model.
	m := &Model{}

	// runNew should return tea.Cmd
	cmd := m.runNew()

	// Method should exist
	_ = cmd
}

func TestModel_ActionNew_MethodExists(t *testing.T) {
	// This test verifies the actionNew method exists on Model.
	m := &Model{
		keys: DefaultKeyMap(),
	}

	// actionNew should return Model and tea.Cmd
	newModel, cmd := m.actionNew()

	_ = newModel
	_ = cmd
}

func TestDispatch_NewBinding(t *testing.T) {
	// Test that 'n' key is bound to new action
	m := &Model{
		keys: DefaultKeyMap(),
	}

	bindings := m.globalBindings()

	// Find the new binding
	found := false
	for _, ab := range bindings {
		if key.Matches(tea.KeyPressMsg(tea.Key{Code: 'n'}), ab.Key) {
			found = true
			if ab.Action == nil {
				t.Error("new binding should have an action")
			}
			break
		}
	}

	if !found {
		t.Error("'n' key should be bound to new action")
	}
}

func TestAbandonCompleteMsg_TypeExists(t *testing.T) {
	// This test verifies the abandonCompleteMsg type exists
	msg := abandonCompleteMsg{changeID: "abc123"}

	// Should be able to access changeID field
	if msg.changeID != "abc123" {
		t.Errorf("expected changeID abc123, got %s", msg.changeID)
	}
}

func TestModel_RunAbandon_MethodExists(t *testing.T) {
	// This test verifies the runAbandon method exists on Model.
	m := &Model{}

	// runAbandon should return tea.Cmd
	cmd := m.runAbandon("abc123")

	// Method should exist
	_ = cmd
}

func TestModel_ActionAbandon_MethodExists(t *testing.T) {
	// This test verifies the actionAbandon method exists on Model.
	m := &Model{
		keys: DefaultKeyMap(),
	}

	// actionAbandon should return Model and tea.Cmd
	newModel, cmd := m.actionAbandon()

	_ = newModel
	_ = cmd
}

func TestDispatch_AbandonBinding(t *testing.T) {
	// Test that 'a' key is bound to abandon action
	m := &Model{
		keys: DefaultKeyMap(),
	}

	bindings := m.globalBindings()

	// Find the abandon binding
	found := false
	for _, ab := range bindings {
		if key.Matches(tea.KeyPressMsg(tea.Key{Code: 'a'}), ab.Key) {
			found = true
			if ab.Action == nil {
				t.Error("abandon binding should have an action")
			}
			break
		}
	}

	if !found {
		t.Error("'a' key should be bound to abandon action")
	}
}
