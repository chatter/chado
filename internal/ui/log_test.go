package ui

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/chatter/chado/internal/jj"
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

func TestLogPanel_SetContent_PreservesSelectionByID(t *testing.T) {
	panel := NewLogPanel()
	panel.SetSize(80, 24)

	// Initial content: A[0], B[1], C[2], D[3], E[4]
	changes := make([]jj.Change, 5)
	var content strings.Builder
	for i := 0; i < 5; i++ {
		changeID := fmt.Sprintf("aaaaaaa%c", 'a'+i)
		changes[i] = jj.Change{ChangeID: changeID}
		fmt.Fprintf(&content, "○ %s description\n", changeID)
	}
	panel.SetContent(content.String(), changes)

	// Select change C (index 2, ID "aaaaaaac")
	panel.cursor = 2
	if panel.SelectedChange().ChangeID != "aaaaaaac" {
		t.Fatalf("should have selected 'aaaaaaac', got '%s'", panel.SelectedChange().ChangeID)
	}

	// Simulate squash: D and E are gone, but A, B, C remain
	// New order: A[0], B[1], C[2]
	smallerChanges := make([]jj.Change, 3)
	var smallerContent strings.Builder
	for i := 0; i < 3; i++ {
		changeID := fmt.Sprintf("aaaaaaa%c", 'a'+i)
		smallerChanges[i] = jj.Change{ChangeID: changeID}
		fmt.Fprintf(&smallerContent, "○ %s description\n", changeID)
	}
	panel.SetContent(smallerContent.String(), smallerChanges)

	// Cursor should still point to C (now at index 2)
	if panel.cursor != 2 {
		t.Fatalf("cursor should be 2 (still on C), got %d", panel.cursor)
	}
	if panel.SelectedChange().ChangeID != "aaaaaaac" {
		t.Fatalf("should still have 'aaaaaaac' selected, got '%s'", panel.SelectedChange().ChangeID)
	}
}

func TestLogPanel_SetContent_SelectionRemovedDefaultsToFirst(t *testing.T) {
	panel := NewLogPanel()
	panel.SetSize(80, 24)

	// Initial content: A[0], B[1], C[2]
	changes := make([]jj.Change, 3)
	var content strings.Builder
	for i := 0; i < 3; i++ {
		changeID := fmt.Sprintf("aaaaaaa%c", 'a'+i)
		changes[i] = jj.Change{ChangeID: changeID}
		fmt.Fprintf(&content, "○ %s description\n", changeID)
	}
	panel.SetContent(content.String(), changes)

	// Select change C (index 2)
	panel.cursor = 2

	// Simulate abandon of C: only A and B remain
	smallerChanges := make([]jj.Change, 2)
	var smallerContent strings.Builder
	for i := 0; i < 2; i++ {
		changeID := fmt.Sprintf("aaaaaaa%c", 'a'+i)
		smallerChanges[i] = jj.Change{ChangeID: changeID}
		fmt.Fprintf(&smallerContent, "○ %s description\n", changeID)
	}
	panel.SetContent(smallerContent.String(), smallerChanges)

	// C is gone, cursor should default to first (A)
	if panel.cursor != 0 {
		t.Fatalf("cursor should default to 0 when selection removed, got %d", panel.cursor)
	}
	if panel.SelectedChange().ChangeID != "aaaaaaaa" {
		t.Fatalf("should default to first change 'aaaaaaaa', got '%s'", panel.SelectedChange().ChangeID)
	}
}

func TestLogPanel_SetContent_EmptyChanges(t *testing.T) {
	panel := NewLogPanel()
	panel.SetSize(80, 24)

	// Set some initial content
	changes := []jj.Change{{ChangeID: "aaaaaaaa"}}
	panel.SetContent("○ aaaaaaaa desc\n", changes)
	panel.cursor = 0

	// Refresh with empty changes
	panel.SetContent("", []jj.Change{})

	// Cursor should be 0, SelectedChange should be nil
	if panel.cursor != 0 {
		t.Fatalf("cursor should be 0 for empty changes, got %d", panel.cursor)
	}
	if panel.SelectedChange() != nil {
		t.Fatal("SelectedChange should be nil for empty changes")
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

// =============================================================================
// Mouse Click Property Tests
// =============================================================================

// Property: After any click, cursor stays in valid range [0, len(changes)-1]
func TestLogPanel_Click_CursorInBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewLogPanel()
		width := rapid.IntRange(40, 200).Draw(t, "width")
		height := rapid.IntRange(10, 100).Draw(t, "height")
		panel.SetSize(width, height)

		// Generate changes
		numChanges := rapid.IntRange(1, 30).Draw(t, "numChanges")
		changes := make([]jj.Change, numChanges)
		for i := 0; i < numChanges; i++ {
			changes[i] = jj.Change{
				ChangeID: rapid.StringMatching(`[a-z]{8}`).Draw(t, "changeID"),
			}
		}
		panel.SetContent("test content", changes)

		// Click at any Y position (including invalid: negative, huge)
		clickY := rapid.IntRange(-100, 500).Draw(t, "clickY")
		panel.HandleClick(clickY)

		// Invariant: cursor in bounds
		if panel.cursor < 0 {
			t.Fatalf("cursor should be >= 0, got %d", panel.cursor)
		}
		if panel.cursor >= numChanges {
			t.Fatalf("cursor should be < %d, got %d", numChanges, panel.cursor)
		}
	})
}

// Property: Click at visual line selects correct change (multi-line entries)
func TestLogPanel_Click_SelectsCorrectChange(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewLogPanel()
		width := rapid.IntRange(40, 200).Draw(t, "width")
		height := rapid.IntRange(10, 100).Draw(t, "height")
		panel.SetSize(width, height)

		numChanges := rapid.IntRange(2, 10).Draw(t, "numChanges")
		linesPerChange := rapid.IntRange(1, 4).Draw(t, "linesPerChange")

		// Build realistic log content with multi-line entries
		var logContent strings.Builder
		changes := make([]jj.Change, numChanges)
		for i := 0; i < numChanges; i++ {
			// Use letter-based change IDs to match the regex [a-z]{8,}
			changeID := fmt.Sprintf("aaaaaaa%c", 'a'+i) // e.g., "aaaaaaaa", "aaaaaaab"
			changes[i] = jj.Change{ChangeID: changeID}

			// First line has change marker
			logContent.WriteString(fmt.Sprintf("○  %s user@example.com\n", changeID))
			// Additional description lines
			for j := 1; j < linesPerChange; j++ {
				logContent.WriteString("│  description line\n")
			}
		}
		panel.SetContent(logContent.String(), changes)

		// Pick a target change and click somewhere within its visual lines
		targetChange := rapid.IntRange(0, numChanges-1).Draw(t, "targetChange")
		lineWithinChange := rapid.IntRange(0, linesPerChange-1).Draw(t, "lineWithinChange")
		clickY := targetChange*linesPerChange + lineWithinChange

		panel.HandleClick(clickY)

		// Invariant: cursor matches target change
		if panel.cursor != targetChange {
			t.Fatalf("clicking line %d (change %d, offset %d) should select change %d, got %d",
				clickY, targetChange, lineWithinChange, targetChange, panel.cursor)
		}
	})
}

// Property: Click outside bounds doesn't change cursor
func TestLogPanel_Click_OutOfBounds_NoChange(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewLogPanel()
		width := rapid.IntRange(40, 200).Draw(t, "width")
		height := rapid.IntRange(10, 100).Draw(t, "height")
		panel.SetSize(width, height)

		numChanges := rapid.IntRange(1, 30).Draw(t, "numChanges")
		changes := make([]jj.Change, numChanges)
		for i := 0; i < numChanges; i++ {
			changes[i] = jj.Change{
				ChangeID: rapid.StringMatching(`[a-z]{8}`).Draw(t, "changeID"),
			}
		}
		panel.SetContent("test content", changes)

		// Set cursor to random valid position
		startCursor := rapid.IntRange(0, numChanges-1).Draw(t, "startCursor")
		panel.cursor = startCursor

		// Click outside bounds (negative)
		invalidY := rapid.IntRange(-100, -1).Draw(t, "negativeY")
		changed := panel.HandleClick(invalidY)

		// Invariant: cursor unchanged, returns false
		if changed {
			t.Fatalf("HandleClick should return false for negative click")
		}
		if panel.cursor != startCursor {
			t.Fatalf("cursor should remain %d after negative click, got %d", startCursor, panel.cursor)
		}
	})
}

// Property: Clicking past all changes does nothing (consistent with files panel)
func TestLogPanel_Click_PastEnd_NoChange(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewLogPanel()
		width := rapid.IntRange(40, 200).Draw(t, "width")
		height := rapid.IntRange(10, 100).Draw(t, "height")
		panel.SetSize(width, height)

		numChanges := rapid.IntRange(2, 20).Draw(t, "numChanges")
		linesPerChange := rapid.IntRange(1, 4).Draw(t, "linesPerChange")

		// Build log content
		var content strings.Builder
		changes := make([]jj.Change, numChanges)
		for i := 0; i < numChanges; i++ {
			changeID := fmt.Sprintf("aaaaaaa%c", 'a'+i)
			changes[i] = jj.Change{ChangeID: changeID}
			fmt.Fprintf(&content, "○ %s description\n", changeID)
			for j := 1; j < linesPerChange; j++ {
				content.WriteString("  extra line\n")
			}
		}
		panel.SetContent(content.String(), changes)

		// Set cursor somewhere
		startCursor := rapid.IntRange(0, numChanges-1).Draw(t, "startCursor")
		panel.cursor = startCursor

		// Click way past the end
		totalLines := numChanges * linesPerChange
		clickY := rapid.IntRange(totalLines, totalLines+100).Draw(t, "clickY")
		changed := panel.HandleClick(clickY)

		// Invariant: cursor unchanged, returns false
		if changed {
			t.Fatalf("HandleClick should return false when clicking past end")
		}
		if panel.cursor != startCursor {
			t.Fatalf("cursor should remain %d after clicking past end, got %d", startCursor, panel.cursor)
		}
	})
}

// Property: Clicking same position returns false
func TestLogPanel_Click_SamePosition_ReturnsFalse(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewLogPanel()
		width := rapid.IntRange(40, 200).Draw(t, "width")
		height := rapid.IntRange(10, 100).Draw(t, "height")
		panel.SetSize(width, height)

		numChanges := rapid.IntRange(1, 20).Draw(t, "numChanges")

		// Build single-line log content
		var content strings.Builder
		changes := make([]jj.Change, numChanges)
		for i := 0; i < numChanges; i++ {
			changeID := fmt.Sprintf("aaaaaaa%c", 'a'+i)
			changes[i] = jj.Change{ChangeID: changeID}
			fmt.Fprintf(&content, "○ %s description\n", changeID)
		}
		panel.SetContent(content.String(), changes)

		// Set cursor to a position
		cursorPos := rapid.IntRange(0, numChanges-1).Draw(t, "cursorPos")
		panel.cursor = cursorPos

		// Click on the same position
		changed := panel.HandleClick(cursorPos)

		// Invariant: returns false, cursor unchanged
		if changed {
			t.Fatalf("HandleClick should return false when clicking already-selected change")
		}
		if panel.cursor != cursorPos {
			t.Fatalf("cursor should remain %d, got %d", cursorPos, panel.cursor)
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
