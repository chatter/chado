package ui

import (
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/chatter/lazyjj/internal/jj"
)

// =============================================================================
// Unit Tests
// =============================================================================

func TestIsChangeStart(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "working copy marker",
			input:    "@  xsssnyux test",
			expected: true,
		},
		{
			name:     "regular commit",
			input:    "○  nlkzwoyt test",
			expected: true,
		},
		{
			name:     "immutable commit",
			input:    "◆  zzzzzzzz root()",
			expected: true,
		},
		{
			name:     "empty commit marker",
			input:    "◇  abcdefgh empty",
			expected: true,
		},
		{
			name:     "with ansi codes",
			input:    "\x1b[1;35m@\x1b[0m  \x1b[1;34mxsssnyux\x1b[0m test",
			expected: true,
		},
		{
			name:     "description continuation",
			input:    "│  this is a description",
			expected: false,
		},
		{
			name:     "graph line only",
			input:    "│",
			expected: false,
		},
		{
			name:     "empty line",
			input:    "",
			expected: false,
		},
		{
			name:     "spaces only",
			input:    "    ",
			expected: false,
		},
		{
			name:     "merge commit graph",
			input:    "├─╮",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isChangeStart(tt.input)
			if result != tt.expected {
				t.Errorf("isChangeStart(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLogPanel_CursorBounds(t *testing.T) {
	panel := NewLogPanel()

	// Set up with some changes
	changes := []jj.Change{
		{ChangeID: "aaaaaaaa", Raw: "@ aaaaaaaa"},
		{ChangeID: "bbbbbbbb", Raw: "○ bbbbbbbb"},
		{ChangeID: "cccccccc", Raw: "○ cccccccc"},
	}
	panel.SetContent("@ aaaaaaaa\n○ bbbbbbbb\n○ cccccccc", changes)
	panel.SetSize(80, 24)

	// Test cursor stays at 0 when moving up from top
	panel.CursorUp()
	if panel.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", panel.cursor)
	}

	// Test cursor moves down
	panel.CursorDown()
	if panel.cursor != 1 {
		t.Errorf("cursor should be 1 after CursorDown, got %d", panel.cursor)
	}

	// Test cursor stops at last item
	panel.CursorDown()
	panel.CursorDown()
	panel.CursorDown() // Try to go past end
	if panel.cursor != 2 {
		t.Errorf("cursor should stop at 2, got %d", panel.cursor)
	}

	// Test GotoTop
	panel.GotoTop()
	if panel.cursor != 0 {
		t.Errorf("cursor should be 0 after GotoTop, got %d", panel.cursor)
	}

	// Test GotoBottom
	panel.GotoBottom()
	if panel.cursor != 2 {
		t.Errorf("cursor should be 2 after GotoBottom, got %d", panel.cursor)
	}
}

func TestLogPanel_SelectedChange(t *testing.T) {
	panel := NewLogPanel()

	// Empty panel
	if panel.SelectedChange() != nil {
		t.Error("SelectedChange should be nil for empty panel")
	}

	// With changes
	changes := []jj.Change{
		{ChangeID: "aaaaaaaa"},
		{ChangeID: "bbbbbbbb"},
	}
	panel.SetContent("test", changes)

	selected := panel.SelectedChange()
	if selected == nil {
		t.Fatal("SelectedChange should not be nil")
	}
	if selected.ChangeID != "aaaaaaaa" {
		t.Errorf("expected first change, got %s", selected.ChangeID)
	}

	// Move cursor and check selection
	panel.CursorDown()
	selected = panel.SelectedChange()
	if selected.ChangeID != "bbbbbbbb" {
		t.Errorf("expected second change, got %s", selected.ChangeID)
	}
}

func TestLogPanel_Focus(t *testing.T) {
	panel := NewLogPanel()

	if panel.focused {
		t.Error("panel should not be focused initially")
	}

	panel.SetFocused(true)
	if !panel.focused {
		t.Error("panel should be focused after SetFocused(true)")
	}

	panel.SetFocused(false)
	if panel.focused {
		t.Error("panel should not be focused after SetFocused(false)")
	}
}

func TestLogPanel_SetSize(t *testing.T) {
	panel := NewLogPanel()
	panel.SetSize(100, 50)

	if panel.width != 100 {
		t.Errorf("width should be 100, got %d", panel.width)
	}
	if panel.height != 50 {
		t.Errorf("height should be 50, got %d", panel.height)
	}
	// Viewport should account for borders
	if panel.viewport.Width != 98 {
		t.Errorf("viewport.Width should be 98, got %d", panel.viewport.Width)
	}
	if panel.viewport.Height != 47 {
		t.Errorf("viewport.Height should be 47, got %d", panel.viewport.Height)
	}
}

// =============================================================================
// Property Tests
// =============================================================================

// Property: Cursor should always be within valid bounds
func TestLogPanel_CursorAlwaysInBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewLogPanel()

		// Generate random number of changes
		numChanges := rapid.IntRange(0, 100).Draw(t, "numChanges")
		changes := make([]jj.Change, numChanges)
		for i := 0; i < numChanges; i++ {
			changes[i] = jj.Change{ChangeID: rapid.StringMatching(`[a-z]{8}`).Draw(t, "changeID")}
		}
		panel.SetContent("test", changes)

		// Perform random operations
		numOps := rapid.IntRange(0, 50).Draw(t, "numOps")
		for i := 0; i < numOps; i++ {
			op := rapid.IntRange(0, 3).Draw(t, "op")
			switch op {
			case 0:
				panel.CursorUp()
			case 1:
				panel.CursorDown()
			case 2:
				panel.GotoTop()
			case 3:
				panel.GotoBottom()
			}
		}

		// Check invariants
		if numChanges == 0 {
			if panel.cursor != 0 {
				t.Fatalf("cursor should be 0 for empty panel, got %d", panel.cursor)
			}
		} else {
			if panel.cursor < 0 {
				t.Fatalf("cursor should never be negative, got %d", panel.cursor)
			}
			if panel.cursor >= numChanges {
				t.Fatalf("cursor %d should be < numChanges %d", panel.cursor, numChanges)
			}
		}
	})
}

// Property: SelectedChange should match cursor position
func TestLogPanel_SelectedChangeMatchesCursor(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewLogPanel()

		numChanges := rapid.IntRange(1, 50).Draw(t, "numChanges")
		changes := make([]jj.Change, numChanges)
		for i := 0; i < numChanges; i++ {
			changes[i] = jj.Change{ChangeID: rapid.StringMatching(`[a-z]{8}`).Draw(t, "changeID")}
		}
		panel.SetContent("test", changes)

		// Random cursor position
		targetPos := rapid.IntRange(0, numChanges-1).Draw(t, "targetPos")
		for i := 0; i < targetPos; i++ {
			panel.CursorDown()
		}

		selected := panel.SelectedChange()
		if selected == nil {
			t.Fatal("SelectedChange should not be nil")
		}
		if selected.ChangeID != changes[panel.cursor].ChangeID {
			t.Fatalf("selected change ID mismatch: got %s, expected %s",
				selected.ChangeID, changes[panel.cursor].ChangeID)
		}
	})
}

// Property: GotoTop always results in cursor=0
func TestLogPanel_GotoTopAlwaysZero(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewLogPanel()

		numChanges := rapid.IntRange(1, 100).Draw(t, "numChanges")
		changes := make([]jj.Change, numChanges)
		for i := 0; i < numChanges; i++ {
			changes[i] = jj.Change{ChangeID: "test"}
		}
		panel.SetContent("test", changes)

		// Move to random position
		moves := rapid.IntRange(0, 50).Draw(t, "moves")
		for i := 0; i < moves; i++ {
			panel.CursorDown()
		}

		panel.GotoTop()
		if panel.cursor != 0 {
			t.Fatalf("cursor should be 0 after GotoTop, got %d", panel.cursor)
		}
	})
}

// Property: GotoBottom always results in cursor at last item
func TestLogPanel_GotoBottomAlwaysLast(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewLogPanel()

		numChanges := rapid.IntRange(1, 100).Draw(t, "numChanges")
		changes := make([]jj.Change, numChanges)
		for i := 0; i < numChanges; i++ {
			changes[i] = jj.Change{ChangeID: "test"}
		}
		panel.SetContent("test", changes)

		panel.GotoBottom()
		if panel.cursor != numChanges-1 {
			t.Fatalf("cursor should be %d after GotoBottom, got %d", numChanges-1, panel.cursor)
		}
	})
}

// Property: isChangeStart should be consistent with ANSI stripping
func TestIsChangeStart_ANSIInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a change start line
		symbol := rapid.SampledFrom([]string{"@", "○", "◆", "◇"}).Draw(t, "symbol")
		changeID := rapid.StringMatching(`[a-z]{8}`).Draw(t, "changeID")
		plainLine := symbol + "  " + changeID + " description"

		// Both plain and ANSI-decorated versions should give same result
		ansiLine := "\x1b[1;35m" + symbol + "\x1b[0m  \x1b[1;34m" + changeID + "\x1b[0m description"

		plainResult := isChangeStart(plainLine)
		ansiResult := isChangeStart(ansiLine)

		if plainResult != ansiResult {
			t.Fatalf("ANSI invariant violated: plain=%v, ansi=%v for symbol=%s",
				plainResult, ansiResult, symbol)
		}
	})
}

// Benchmark for isChangeStart
func BenchmarkIsChangeStart(b *testing.B) {
	lines := []string{
		"@  xsssnyux test@example.com 2026-01-29 12:00:00",
		"\x1b[1;35m@\x1b[0m  \x1b[1;34mxsssnyux\x1b[0m test",
		"│  this is a description line",
		"○  nlkzwoyt another change",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, line := range lines {
			isChangeStart(line)
		}
	}
}

// Helper to generate log output
func generateLogOutput(numChanges int) string {
	var lines []string
	for i := 0; i < numChanges; i++ {
		symbol := "○"
		if i == 0 {
			symbol = "@"
		}
		lines = append(lines, symbol+"  "+strings.Repeat("x", 8)+" description "+string(rune('a'+i)))
		lines = append(lines, "│  continued")
	}
	return strings.Join(lines, "\n")
}
