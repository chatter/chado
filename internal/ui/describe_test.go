package ui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"pgregory.net/rapid"
)

// =============================================================================
// Unit Tests
// =============================================================================

func TestDescribeInput_New(t *testing.T) {
	input := NewDescribeInput()

	if input == nil {
		t.Fatal("NewDescribeInput should not return nil")
	}

	// Check initial state
	if input.Value() != "" {
		t.Errorf("initial value should be empty, got %q", input.Value())
	}
	if input.ChangeID() != "" {
		t.Errorf("initial changeID should be empty, got %q", input.ChangeID())
	}
}

func TestDescribeInput_SetValue(t *testing.T) {
	input := NewDescribeInput()

	input.SetValue("test description")
	if input.Value() != "test description" {
		t.Errorf("expected 'test description', got %q", input.Value())
	}

	// Test overwriting
	input.SetValue("new description")
	if input.Value() != "new description" {
		t.Errorf("expected 'new description', got %q", input.Value())
	}
}

func TestDescribeInput_SetChangeID(t *testing.T) {
	input := NewDescribeInput()

	input.SetChangeID("xsssnyux")
	if input.ChangeID() != "xsssnyux" {
		t.Errorf("expected 'xsssnyux', got %q", input.ChangeID())
	}
}

func TestDescribeInput_SetSize(t *testing.T) {
	input := NewDescribeInput()

	input.SetSize(80, 10)
	if input.width != 80 {
		t.Errorf("expected width 80, got %d", input.width)
	}
	if input.height != 10 {
		t.Errorf("expected height 10, got %d", input.height)
	}
}

func TestDescribeInput_Update_Submit(t *testing.T) {
	input := NewDescribeInput()
	input.SetChangeID("testchange")
	input.SetValue("my description")

	// Simulate Enter key
	keyMsg := tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	cmd := input.Update(keyMsg)

	if cmd == nil {
		t.Fatal("expected cmd on enter")
	}

	msg := cmd()
	submitMsg, ok := msg.(DescribeSubmitMsg)
	if !ok {
		t.Fatalf("expected DescribeSubmitMsg, got %T", msg)
	}
	if submitMsg.ChangeID != "testchange" {
		t.Errorf("expected changeID 'testchange', got %q", submitMsg.ChangeID)
	}
	if submitMsg.Description != "my description" {
		t.Errorf("expected description 'my description', got %q", submitMsg.Description)
	}
}

func TestDescribeInput_Update_Cancel(t *testing.T) {
	input := NewDescribeInput()
	input.SetChangeID("testchange")
	input.SetValue("my description")

	// Simulate Esc key
	keyMsg := tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})
	cmd := input.Update(keyMsg)

	if cmd == nil {
		t.Fatal("expected cmd on escape")
	}

	msg := cmd()
	_, ok := msg.(DescribeCancelMsg)
	if !ok {
		t.Fatalf("expected DescribeCancelMsg, got %T", msg)
	}
}

func TestDescribeInput_View_ContainsElements(t *testing.T) {
	input := NewDescribeInput()
	input.SetChangeID("xsssnyux")
	input.SetValue("test description")
	input.SetSize(60, 10)

	view := input.View()

	// Should contain the change ID in title
	if !strings.Contains(view, "xsssnyux") {
		t.Error("view should contain the change ID")
	}

	// Should contain hint text with symbols
	if !strings.Contains(view, "⏎") {
		t.Error("view should contain enter symbol")
	}
	if !strings.Contains(view, "⎋") {
		t.Error("view should contain escape symbol")
	}
}

func TestDescribeInput_WidthHeight(t *testing.T) {
	input := NewDescribeInput()
	input.SetChangeID("xsssnyux")
	input.SetValue("test")
	input.SetSize(60, 10)

	width := input.Width()
	height := input.Height()

	if width <= 0 {
		t.Errorf("width should be positive, got %d", width)
	}
	if height <= 0 {
		t.Errorf("height should be positive, got %d", height)
	}
}

// =============================================================================
// Property Tests
// =============================================================================

// Property: SetValue/Value round-trips correctly
func TestDescribeInput_ValueRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := NewDescribeInput()
		// Use printable ASCII strings (textinput may filter control chars)
		value := rapid.StringMatching(`[a-zA-Z0-9 .,!?'-]{0,100}`).Draw(t, "value")

		input.SetValue(value)
		if input.Value() != value {
			t.Fatalf("value mismatch: set %q, got %q", value, input.Value())
		}
	})
}

// Property: SetChangeID/ChangeID round-trips correctly
func TestDescribeInput_ChangeIDRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := NewDescribeInput()
		changeID := rapid.StringMatching(`[a-z]{8,12}`).Draw(t, "changeID")

		input.SetChangeID(changeID)
		if input.ChangeID() != changeID {
			t.Fatalf("changeID mismatch: set %q, got %q", changeID, input.ChangeID())
		}
	})
}

// Property: Enter always produces DescribeSubmitMsg with correct values
func TestDescribeInput_SubmitPreservesValues(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := NewDescribeInput()
		changeID := rapid.StringMatching(`[a-z]{8}`).Draw(t, "changeID")
		// Use printable ASCII strings (textinput may filter control chars)
		description := rapid.StringMatching(`[a-zA-Z0-9 .,!?'-]{0,100}`).Draw(t, "description")

		input.SetChangeID(changeID)
		input.SetValue(description)

		keyMsg := tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
		cmd := input.Update(keyMsg)

		if cmd == nil {
			t.Fatal("expected cmd on enter")
		}

		msg := cmd()
		submitMsg, ok := msg.(DescribeSubmitMsg)
		if !ok {
			t.Fatalf("expected DescribeSubmitMsg, got %T", msg)
		}
		if submitMsg.ChangeID != changeID {
			t.Fatalf("changeID mismatch: expected %q, got %q", changeID, submitMsg.ChangeID)
		}
		if submitMsg.Description != description {
			t.Fatalf("description mismatch: expected %q, got %q", description, submitMsg.Description)
		}
	})
}

// Property: Width and Height are always positive after SetSize
func TestDescribeInput_SizeAlwaysPositive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := NewDescribeInput()
		input.SetChangeID("testtest")
		input.SetValue("test")

		width := rapid.IntRange(20, 200).Draw(t, "width")
		height := rapid.IntRange(5, 50).Draw(t, "height")
		input.SetSize(width, height)

		if input.Width() <= 0 {
			t.Fatalf("Width should be positive, got %d", input.Width())
		}
		if input.Height() <= 0 {
			t.Fatalf("Height should be positive, got %d", input.Height())
		}
	})
}

// Property: Key bindings for submit and cancel are correctly configured
func TestDescribeInput_KeyBindingsConfigured(t *testing.T) {
	input := NewDescribeInput()

	// Check submit binding matches enter
	enterKey := tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	if !key.Matches(enterKey, input.submit) {
		t.Error("submit binding should match enter key")
	}

	// Check cancel binding matches escape
	escKey := tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})
	if !key.Matches(escKey, input.cancel) {
		t.Error("cancel binding should match escape key")
	}
}
