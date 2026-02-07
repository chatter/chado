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

func TestIsEntryStart(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Operation log entries (12 hex characters)
		{
			name:     "current operation marker",
			input:    "@  bbc9fee12c4d user@host 4 minutes ago",
			expected: true,
		},
		{
			name:     "regular operation",
			input:    "○  86d0094c958f user@host 4 days ago",
			expected: true,
		},
		{
			name:     "operation with ansi codes",
			input:    "\x1b[1;35m@\x1b[0m  \x1b[1;34mbbc9fee12c4d\x1b[0m user@host",
			expected: true,
		},
		// Evolog entries (8+ lowercase letters)
		{
			name:     "evolog current change",
			input:    "@  mkvurkku user@host 2 hours ago",
			expected: true,
		},
		{
			name:     "evolog regular change",
			input:    "○  xsssnyux user@host 1 day ago",
			expected: true,
		},
		{
			name:     "evolog with ansi codes",
			input:    "\x1b[1;35m@\x1b[0m  \x1b[1;34mmkvurkku\x1b[0m user@host",
			expected: true,
		},
		{
			name:     "evolog longer change id",
			input:    "@  mkvurkkulong user@host now",
			expected: true,
		},
		// Evolog entries with version suffix (historical versions)
		{
			name:     "evolog version 1",
			input:    "○  npwtzrzq/1 user@host 1 hour ago",
			expected: true,
		},
		{
			name:     "evolog version 10",
			input:    "○  mkvurkku/10 user@host 2 hours ago",
			expected: true,
		},
		{
			name:     "evolog version with ansi",
			input:    "\x1b[1;35m○\x1b[0m  \x1b[1;34mnpwtzrzq/5\x1b[0m user@host",
			expected: true,
		},
		// Non-matching lines
		{
			name:     "description continuation",
			input:    "│  snapshot working copy",
			expected: false,
		},
		{
			name:     "args line",
			input:    "│  args: jj log",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEntryStart(tt.input)
			if result != tt.expected {
				t.Errorf("isEntryStart(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestOpLogPanel_CursorBounds(t *testing.T) {
	panel := NewOpLogPanel()

	// Set up with some operations
	operations := []jj.Operation{
		{OpID: "aaaaaaaaaaaa", Raw: "@ aaaaaaaaaaaa"},
		{OpID: "bbbbbbbbbbbb", Raw: "○ bbbbbbbbbbbb"},
		{OpID: "cccccccccccc", Raw: "○ cccccccccccc"},
	}
	panel.SetContent("@ aaaaaaaaaaaa\n○ bbbbbbbbbbbb\n○ cccccccccccc", operations)
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

func TestOpLogPanel_SelectedOperation(t *testing.T) {
	panel := NewOpLogPanel()

	// Empty panel
	if panel.SelectedOperation() != nil {
		t.Error("SelectedOperation should be nil for empty panel")
	}

	// With operations
	operations := []jj.Operation{
		{OpID: "aaaaaaaaaaaa"},
		{OpID: "bbbbbbbbbbbb"},
	}
	panel.SetContent("test", operations)

	selected := panel.SelectedOperation()
	if selected == nil {
		t.Fatal("SelectedOperation should not be nil")
	}
	if selected.OpID != "aaaaaaaaaaaa" {
		t.Errorf("expected first operation, got %s", selected.OpID)
	}

	// Move cursor and check selection
	panel.CursorDown()
	selected = panel.SelectedOperation()
	if selected.OpID != "bbbbbbbbbbbb" {
		t.Errorf("expected second operation, got %s", selected.OpID)
	}
}

func TestOpLogPanel_Focus(t *testing.T) {
	panel := NewOpLogPanel()

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

func TestOpLogPanel_SetSize(t *testing.T) {
	panel := NewOpLogPanel()
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

func TestOpLogPanel_SetContent_PreservesSelectionByID(t *testing.T) {
	panel := NewOpLogPanel()
	panel.SetSize(80, 24)

	initial_op_count := 5
	operations := make([]jj.Operation, initial_op_count)
	var content strings.Builder
	for i := range initial_op_count {
		opID := fmt.Sprintf("%012x", i) // 12-char hex
		operations[i] = jj.Operation{OpID: opID}
		fmt.Fprintf(&content, "○  %s description\n", opID)
	}
	panel.SetContent(content.String(), operations)

	// Select operation at index 2
	panel.cursor = 2
	selectedID := panel.SelectedOperation().OpID

	// Refresh with same operations (simulating watcher update)
	panel.SetContent(content.String(), operations)

	// Cursor should still point to same operation
	if panel.SelectedOperation().OpID != selectedID {
		t.Fatalf("should still have '%s' selected, got '%s'", selectedID, panel.SelectedOperation().OpID)
	}
}

func TestOpLogPanel_SetContent_SelectionRemovedDefaultsToFirst(t *testing.T) {
	panel := NewOpLogPanel()
	panel.SetSize(80, 24)

	// Initial content: 3 operations
	initial_op_count := 3
	operations := make([]jj.Operation, initial_op_count)
	var content strings.Builder
	for i := range initial_op_count {
		opID := fmt.Sprintf("%012x", i)
		operations[i] = jj.Operation{OpID: opID}
		fmt.Fprintf(&content, "○  %s description\n", opID)
	}
	panel.SetContent(content.String(), operations)

	// Select last operation (index 2)
	panel.cursor = 2

	// Refresh with fewer operations (last one gone)
	smallerOps := operations[:2]
	var smallerContent strings.Builder
	for i := range 2 {
		fmt.Fprintf(&smallerContent, "○  %s description\n", operations[i].OpID)
	}
	panel.SetContent(smallerContent.String(), smallerOps)

	// Cursor should default to first
	if panel.cursor != 0 {
		t.Fatalf("cursor should default to 0 when selection removed, got %d", panel.cursor)
	}
}

// =============================================================================
// Property Tests
// =============================================================================

// Property: Cursor should always be within valid bounds
func TestOpLogPanel_CursorAlwaysInBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewOpLogPanel()

		// Generate random number of operations
		numOps := rapid.IntRange(0, 100).Draw(t, "numOps")
		operations := make([]jj.Operation, numOps)
		for i := range numOps {
			operations[i] = jj.Operation{OpID: rapid.StringMatching(`[0-9a-f]{12}`).Draw(t, "opID")}
		}
		panel.SetContent("test", operations)

		// Perform random operations
		numActions := rapid.IntRange(0, 50).Draw(t, "numActions")
		for range numActions {
			action := rapid.IntRange(0, 3).Draw(t, "action")
			switch action {
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
		if numOps == 0 {
			if panel.cursor != 0 {
				t.Fatalf("cursor should be 0 for empty panel, got %d", panel.cursor)
			}
		} else {
			if panel.cursor < 0 {
				t.Fatalf("cursor should never be negative, got %d", panel.cursor)
			}
			if panel.cursor >= numOps {
				t.Fatalf("cursor %d should be < numOps %d", panel.cursor, numOps)
			}
		}
	})
}

// Property: SelectedOperation should match cursor position
func TestOpLogPanel_SelectedOperationMatchesCursor(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewOpLogPanel()

		numOps := rapid.IntRange(1, 50).Draw(t, "numOps")
		operations := make([]jj.Operation, numOps)
		for i := range numOps {
			operations[i] = jj.Operation{OpID: rapid.StringMatching(`[0-9a-f]{12}`).Draw(t, "opID")}
		}
		panel.SetContent("test", operations)

		// Random cursor position
		targetPos := rapid.IntRange(0, numOps-1).Draw(t, "targetPos")
		for range targetPos {
			panel.CursorDown()
		}

		selected := panel.SelectedOperation()
		if selected == nil {
			t.Fatal("SelectedOperation should not be nil")
			return
		}
		if selected.OpID != operations[panel.cursor].OpID {
			t.Fatalf("selected operation ID mismatch: got %s, expected %s",
				selected.OpID, operations[panel.cursor].OpID)
		}
	})
}

// Property: GotoTop always results in cursor=0
func TestOpLogPanel_GotoTopAlwaysZero(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewOpLogPanel()

		numOps := rapid.IntRange(1, 100).Draw(t, "numOps")
		operations := make([]jj.Operation, numOps)
		for i := range numOps {
			operations[i] = jj.Operation{OpID: "test"}
		}
		panel.SetContent("test", operations)

		// Move to random position
		moves := rapid.IntRange(0, 50).Draw(t, "moves")
		for range moves {
			panel.CursorDown()
		}

		panel.GotoTop()
		if panel.cursor != 0 {
			t.Fatalf("cursor should be 0 after GotoTop, got %d", panel.cursor)
		}
	})
}

// Property: GotoBottom always results in cursor at last item
func TestOpLogPanel_GotoBottomAlwaysLast(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewOpLogPanel()

		numOps := rapid.IntRange(1, 100).Draw(t, "numOps")
		operations := make([]jj.Operation, numOps)
		for i := range numOps {
			operations[i] = jj.Operation{OpID: "test"}
		}
		panel.SetContent("test", operations)

		panel.GotoBottom()
		if panel.cursor != numOps-1 {
			t.Fatalf("cursor should be %d after GotoBottom, got %d", numOps-1, panel.cursor)
		}
	})
}

// Property: isEntryStart should be consistent with ANSI stripping for operation IDs
func TestIsEntryStart_ANSIInvariant_OpID(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate an operation start line (12 hex chars)
		symbol := rapid.SampledFrom([]string{"@", "○"}).Draw(t, "symbol")
		opID := rapid.StringMatching(`[0-9a-f]{12}`).Draw(t, "opID")
		plainLine := symbol + "  " + opID + " user@host now"

		// Both plain and ANSI-decorated versions should give same result
		ansiLine := "\x1b[1;35m" + symbol + "\x1b[0m  \x1b[1;34m" + opID + "\x1b[0m user@host now"

		plainResult := isEntryStart(plainLine)
		ansiResult := isEntryStart(ansiLine)

		if plainResult != ansiResult {
			t.Fatalf("ANSI invariant violated: plain=%v, ansi=%v for symbol=%s",
				plainResult, ansiResult, symbol)
		}
	})
}

// Property: isEntryStart should be consistent with ANSI stripping for change IDs
func TestIsEntryStart_ANSIInvariant_ChangeID(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate an evolog entry line (8+ lowercase letters)
		symbol := rapid.SampledFrom([]string{"@", "○"}).Draw(t, "symbol")
		changeID := rapid.StringMatching(`[a-z]{8,12}`).Draw(t, "changeID")
		plainLine := symbol + "  " + changeID + " user@host now"

		// Both plain and ANSI-decorated versions should give same result
		ansiLine := "\x1b[1;35m" + symbol + "\x1b[0m  \x1b[1;34m" + changeID + "\x1b[0m user@host now"

		plainResult := isEntryStart(plainLine)
		ansiResult := isEntryStart(ansiLine)

		if plainResult != ansiResult {
			t.Fatalf("ANSI invariant violated: plain=%v, ansi=%v for symbol=%s changeID=%s",
				plainResult, ansiResult, symbol, changeID)
		}
	})
}

// Property: SelectedOperation returns nil iff operations empty
func TestOpLogPanel_SelectionConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewOpLogPanel()

		numOps := rapid.IntRange(0, 50).Draw(t, "numOps")
		operations := make([]jj.Operation, numOps)
		for i := range numOps {
			operations[i] = jj.Operation{OpID: rapid.StringMatching(`[0-9a-f]{12}`).Draw(t, "opID")}
		}
		panel.SetContent("test", operations)

		selected := panel.SelectedOperation()

		if numOps == 0 {
			if selected != nil {
				t.Fatal("SelectedOperation should be nil when operations empty")
			}
		} else {
			if selected == nil {
				t.Fatal("SelectedOperation should not be nil when operations exist")
			}
		}
	})
}

// =============================================================================
// Mouse Click Property Tests
// =============================================================================

// Property: After any click, cursor stays in valid range [0, len(operations)-1]
func TestOpLogPanel_Click_CursorInBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewOpLogPanel()
		width := rapid.IntRange(40, 200).Draw(t, "width")
		height := rapid.IntRange(10, 100).Draw(t, "height")
		panel.SetSize(width, height)

		// Generate operations
		numOps := rapid.IntRange(1, 30).Draw(t, "numOps")
		operations := make([]jj.Operation, numOps)
		for i := range numOps {
			operations[i] = jj.Operation{
				OpID: rapid.StringMatching(`[0-9a-f]{12}`).Draw(t, "opID"),
			}
		}
		panel.SetContent("test content", operations)

		// Click at any Y position (including invalid: negative, huge)
		clickY := rapid.IntRange(-100, 500).Draw(t, "clickY")
		panel.HandleClick(clickY)

		// Invariant: cursor in bounds
		if panel.cursor < 0 {
			t.Fatalf("cursor should be >= 0, got %d", panel.cursor)
		}
		if panel.cursor >= numOps {
			t.Fatalf("cursor should be < %d, got %d", numOps, panel.cursor)
		}
	})
}

// =============================================================================
// Mode Toggle Tests (Evolog Support)
// =============================================================================

func TestOpLogPanel_DefaultModeIsOpLog(t *testing.T) {
	panel := NewOpLogPanel()

	if panel.mode != ModeOpLog {
		t.Errorf("default mode should be ModeOpLog, got %v", panel.mode)
	}
}

func TestOpLogPanel_SetEvoLogContent_SwitchesMode(t *testing.T) {
	panel := NewOpLogPanel()

	operations := []jj.Operation{
		{OpID: "aaaaaaaaaaaa", Raw: "@ aaaaaaaaaaaa"},
	}

	panel.SetEvoLogContent("mkvurkku", "mkv", "@ aaaaaaaaaaaa", operations)

	if panel.mode != ModeEvoLog {
		t.Errorf("mode should be ModeEvoLog after SetEvoLogContent, got %v", panel.mode)
	}
	if panel.changeID != "mkvurkku" {
		t.Errorf("changeID should be 'mkvurkku', got '%s'", panel.changeID)
	}
	if panel.shortCode != "mkv" {
		t.Errorf("shortCode should be 'mkv', got '%s'", panel.shortCode)
	}
}

func TestOpLogPanel_SetOpLogContent_SwitchesMode(t *testing.T) {
	panel := NewOpLogPanel()

	// First switch to evolog mode
	operations := []jj.Operation{
		{OpID: "aaaaaaaaaaaa", Raw: "@ aaaaaaaaaaaa"},
	}
	panel.SetEvoLogContent("mkvurkku", "mkv", "@ aaaaaaaaaaaa", operations)

	// Now switch back to oplog mode
	panel.SetOpLogContent("@ bbbbbbbbbbbb", operations)

	if panel.mode != ModeOpLog {
		t.Errorf("mode should be ModeOpLog after SetOpLogContent, got %v", panel.mode)
	}
	if panel.changeID != "" {
		t.Errorf("changeID should be empty, got '%s'", panel.changeID)
	}
	if panel.shortCode != "" {
		t.Errorf("shortCode should be empty, got '%s'", panel.shortCode)
	}
}

func TestOpLogPanel_TitleByMode_OpLog(t *testing.T) {
	panel := NewOpLogPanel()
	panel.SetSize(80, 24)

	operations := []jj.Operation{
		{OpID: "aaaaaaaaaaaa", Raw: "@ aaaaaaaaaaaa"},
	}
	panel.SetOpLogContent("@ aaaaaaaaaaaa", operations)

	view := panel.View()

	// Title should contain "Operations Log"
	if !strings.Contains(view, "Operations Log") {
		t.Errorf("view should contain 'Operations Log' in oplog mode, got: %s", view)
	}
}

func TestOpLogPanel_TitleByMode_EvoLog(t *testing.T) {
	panel := NewOpLogPanel()
	panel.SetSize(80, 24)

	operations := []jj.Operation{
		{OpID: "aaaaaaaaaaaa", Raw: "@ aaaaaaaaaaaa"},
	}
	panel.SetEvoLogContent("mkvurkku", "mkv", "@ aaaaaaaaaaaa", operations)

	view := panel.View()

	// Title should contain "Evolution:" and NOT "Operations Log"
	if strings.Contains(view, "Operations Log") {
		t.Errorf("view should NOT contain 'Operations Log' in evolog mode")
	}

	// Should contain the change ID (stripped view check - ANSI codes complicate exact match)
	stripped := stripTestANSI(view)
	if !strings.Contains(stripped, "Evolution") {
		t.Errorf("view should contain 'Evolution' in evolog mode, got: %s", stripped)
	}
}

// stripTestANSI is a helper to strip ANSI codes for test assertions
func stripTestANSI(s string) string {
	// Simple ANSI stripper for tests
	result := s
	for strings.Contains(result, "\x1b[") {
		start := strings.Index(result, "\x1b[")
		end := start + 2
		for end < len(result) && result[end] != 'm' {
			end++
		}
		if end < len(result) {
			result = result[:start] + result[end+1:]
		} else {
			break
		}
	}
	return result
}
