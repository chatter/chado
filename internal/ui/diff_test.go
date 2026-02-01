package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chatter/chado/internal/jj"
	"pgregory.net/rapid"
)

// =============================================================================
// Test Helpers
// =============================================================================

// generateHunks creates a slice of non-overlapping hunks with random sizes
func generateHunks(t *rapid.T, count int) []jj.Hunk {
	if count == 0 {
		return nil
	}
	hunks := make([]jj.Hunk, count)
	startLine := 0
	for i := 0; i < count; i++ {
		size := rapid.IntRange(1, 10).Draw(t, "hunkSize")
		hunks[i] = jj.Hunk{
			StartLine: startLine,
			EndLine:   startLine + size - 1,
		}
		startLine += size
	}
	return hunks
}

// setupPanelWithHunks creates a panel with generated hunks and enough content
func setupPanelWithHunks(t *rapid.T) (*DiffPanel, int, int) {
	panel := NewDiffPanel()
	panel.SetSize(80, 24)

	numHunks := rapid.IntRange(1, 10).Draw(t, "numHunks")
	headerLines := rapid.IntRange(0, 10).Draw(t, "headerLines")

	panel.hunks = generateHunks(t, numHunks)
	panel.headerLines = headerLines

	// Calculate total content lines needed
	totalLines := headerLines
	if len(panel.hunks) > 0 {
		lastHunk := panel.hunks[len(panel.hunks)-1]
		totalLines += lastHunk.EndLine + 10 // Extra padding
	}
	content := strings.Repeat("line\n", totalLines+50)
	panel.viewport.SetContent(content)

	return &panel, numHunks, headerLines
}

// =============================================================================
// Unit Tests
// =============================================================================

func TestDiffPanel_SetSize(t *testing.T) {
	panel := NewDiffPanel()
	panel.SetSize(120, 40)

	if panel.width != 120 {
		t.Errorf("width should be 120, got %d", panel.width)
	}
	if panel.height != 40 {
		t.Errorf("height should be 40, got %d", panel.height)
	}
}

func TestDiffPanel_Focus(t *testing.T) {
	panel := NewDiffPanel()

	if panel.focused {
		t.Error("panel should not be focused initially")
	}

	panel.SetFocused(true)
	if !panel.focused {
		t.Error("panel should be focused after SetFocused(true)")
	}
}

func TestDiffPanel_SetTitle(t *testing.T) {
	panel := NewDiffPanel()

	if panel.title != "Diff" {
		t.Errorf("default title should be 'Diff', got %s", panel.title)
	}

	panel.SetTitle("Patch")
	if panel.title != "Patch" {
		t.Errorf("title should be 'Patch', got %s", panel.title)
	}
}

func TestDiffPanel_SetShowDetails(t *testing.T) {
	panel := NewDiffPanel()

	if !panel.showDetails {
		t.Error("showDetails should be true by default")
	}

	panel.SetShowDetails(false)
	if panel.showDetails {
		t.Error("showDetails should be false after SetShowDetails(false)")
	}
}

func TestDiffPanel_SetDetails(t *testing.T) {
	panel := NewDiffPanel()
	panel.SetSize(80, 24)

	details := DetailsHeader{
		ChangeID:    "xsssnyux",
		CommitID:    "abc123def",
		Author:      "test@example.com",
		Date:        "2026-01-29",
		Description: "Test description",
	}

	panel.SetDetails(details)

	if panel.details.ChangeID != "xsssnyux" {
		t.Errorf("ChangeID should be 'xsssnyux', got %s", panel.details.ChangeID)
	}
	if panel.details.Description != "Test description" {
		t.Errorf("Description should be 'Test description', got %s", panel.details.Description)
	}
}

func TestDiffPanel_HunkNavigation(t *testing.T) {
	panel := NewDiffPanel()
	panel.SetSize(80, 40) // Taller to allow scrolling

	// Set diff with multiple sections (more lines so viewport can scroll)
	diff := `Added regular file main.go:
        1: package main
        2: func main() {}
        3:
        4:
        5:
        6:
        7:
        8:
        9:
       10:
Added regular file app.go:
        1: package app
        2: func init() {}
        3:
        4:
        5:
        6:
        7:
        8:
        9:
       10:
Added regular file test.go:
        1: package test
        2:
        3:
        4:
        5:`

	panel.SetDiff(diff)

	// Should have 3 hunks (one for each Added line)
	if len(panel.hunks) != 3 {
		t.Errorf("expected 3 hunks, got %d", len(panel.hunks))
	}

	// Test navigation - starts with no hunk selected (in header)
	if panel.currentHunk != noHunkSelected {
		t.Errorf("currentHunk should start at noHunkSelected, got %d", panel.currentHunk)
	}

	panel.NextHunk()
	if panel.currentHunk != 0 {
		t.Errorf("currentHunk should be 0 after first NextHunk, got %d", panel.currentHunk)
	}

	panel.NextHunk()
	if panel.currentHunk != 1 {
		t.Errorf("currentHunk should be 1 after second NextHunk, got %d", panel.currentHunk)
	}

	panel.NextHunk()
	if panel.currentHunk != 2 {
		t.Errorf("currentHunk should be 2 after third NextHunk, got %d", panel.currentHunk)
	}

	// Should stop at last hunk
	panel.NextHunk()
	if panel.currentHunk != 2 {
		t.Errorf("currentHunk should stay at 2, got %d", panel.currentHunk)
	}

	panel.PrevHunk()
	if panel.currentHunk != 1 {
		t.Errorf("currentHunk should be 1 after PrevHunk, got %d", panel.currentHunk)
	}

	// Test GotoTop/GotoBottom
	panel.GotoBottom()
	if panel.currentHunk != 2 {
		t.Errorf("currentHunk should be 2 after GotoBottom, got %d", panel.currentHunk)
	}

	panel.GotoTop()
	if panel.currentHunk != noHunkSelected {
		t.Errorf("currentHunk should be noHunkSelected after GotoTop, got %d", panel.currentHunk)
	}
}

func TestDiffPanel_EmptyDiff(t *testing.T) {
	panel := NewDiffPanel()
	panel.SetSize(80, 24)
	panel.SetDiff("")

	if len(panel.hunks) != 0 {
		t.Errorf("empty diff should have 0 hunks, got %d", len(panel.hunks))
	}

	// Navigation should not panic
	panel.NextHunk()
	panel.PrevHunk()
	panel.GotoTop()
	panel.GotoBottom()
}

func TestParseDetailsFromShow(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected DetailsHeader
	}{
		{
			name:     "empty input",
			input:    "",
			expected: DetailsHeader{},
		},
		{
			name: "full details",
			input: `Commit ID: abc123def456
Change ID: xsssnyux
Author: test@example.com
Date: 2026-01-29 12:00:00
Description: This is a test commit

diff --git a/main.go b/main.go`,
			expected: DetailsHeader{
				ChangeID:    "xsssnyux",
				CommitID:    "abc123def456",
				Author:      "test@example.com",
				Date:        "2026-01-29 12:00:00",
				Description: "This is a test commit",
			},
		},
		{
			name: "with ansi codes",
			input: "\x1b[1mCommit ID:\x1b[0m abc123\n" +
				"\x1b[1mChange ID:\x1b[0m xsssnyux\n",
			expected: DetailsHeader{
				ChangeID: "xsssnyux",
				CommitID: "abc123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDetailsFromShow(tt.input)
			if result.ChangeID != tt.expected.ChangeID {
				t.Errorf("ChangeID: got %q, want %q", result.ChangeID, tt.expected.ChangeID)
			}
			if result.CommitID != tt.expected.CommitID {
				t.Errorf("CommitID: got %q, want %q", result.CommitID, tt.expected.CommitID)
			}
		})
	}
}

func TestStripANSI_UI(t *testing.T) {
	// Test the UI package's stripANSI function
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"\x1b[1;32mbold green\x1b[0m", "bold green"},
	}

	for _, tt := range tests {
		result := stripANSI(tt.input)
		if result != tt.expected {
			t.Errorf("stripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// =============================================================================
// Property Tests
// =============================================================================

// Property: currentHunk should always be within bounds
func TestDiffPanel_HunkBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewDiffPanel()
		panel.SetSize(80, 24)

		// Generate random diff with sections
		numSections := rapid.IntRange(0, 20).Draw(t, "numSections")
		var lines []string
		for i := 0; i < numSections; i++ {
			status := rapid.SampledFrom([]string{"Added", "Modified", "Removed"}).Draw(t, "status")
			filename := rapid.StringMatching(`[a-z]{3,10}\.go`).Draw(t, "filename")
			lines = append(lines, status+" regular file "+filename+":")
			lines = append(lines, "        1: content")
		}
		diff := strings.Join(lines, "\n")
		panel.SetDiff(diff)

		// Perform random operations
		numOps := rapid.IntRange(0, 50).Draw(t, "numOps")
		for i := 0; i < numOps; i++ {
			op := rapid.IntRange(0, 3).Draw(t, "op")
			switch op {
			case 0:
				panel.NextHunk()
			case 1:
				panel.PrevHunk()
			case 2:
				panel.GotoTop()
			case 3:
				panel.GotoBottom()
			}
		}

		// Check invariants - currentHunk can be noHunkSelected or 0 to len-1
		if panel.currentHunk < noHunkSelected {
			t.Fatalf("currentHunk should be >= noHunkSelected, got %d", panel.currentHunk)
		}
		if len(panel.hunks) > 0 && panel.currentHunk >= len(panel.hunks) {
			t.Fatalf("currentHunk %d should be < len(hunks) %d", panel.currentHunk, len(panel.hunks))
		}
	})
}

// Property: GotoTop always resets currentHunk to 0
func TestDiffPanel_GotoTopResetsHunk(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewDiffPanel()
		panel.SetSize(80, 24)

		// Generate diff with some hunks
		numSections := rapid.IntRange(1, 10).Draw(t, "numSections")
		var lines []string
		for i := 0; i < numSections; i++ {
			lines = append(lines, "Added regular file file"+string(rune('a'+i))+".go:")
			lines = append(lines, "        1: content")
		}
		panel.SetDiff(strings.Join(lines, "\n"))

		// Navigate around
		moves := rapid.IntRange(0, 20).Draw(t, "moves")
		for i := 0; i < moves; i++ {
			panel.NextHunk()
		}

		panel.GotoTop()
		if panel.currentHunk != noHunkSelected {
			t.Fatalf("currentHunk should be noHunkSelected after GotoTop, got %d", panel.currentHunk)
		}
	})
}

// Property: SetDiff resets viewport to top and currentHunk to noHunkSelected
func TestDiffPanel_SetDiffResetsHunk(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewDiffPanel()
		panel.SetSize(80, 24)

		// Set initial state
		content := strings.Repeat("line\n", 100)
		panel.viewport.SetContent(content)
		panel.viewport.SetYOffset(rapid.IntRange(1, 50).Draw(t, "initialOffset"))
		panel.currentHunk = rapid.IntRange(0, 10).Draw(t, "initialHunk")

		// Set new diff
		panel.SetDiff("new diff content\n")

		if panel.viewport.YOffset != 0 {
			t.Fatalf("expected YOffset=0 after SetDiff, got %d", panel.viewport.YOffset)
		}
		if panel.currentHunk != noHunkSelected {
			t.Fatalf("currentHunk should be noHunkSelected after SetDiff, got %d", panel.currentHunk)
		}
	})
}

// Property: NextHunk increments currentHunk and positions viewport at hunk start
func TestNextHunk_IncrementsAndPositions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, numHunks, headerLines := setupPanelWithHunks(t)

		// Start at a random hunk (not the last one)
		if numHunks < 2 {
			return // Need at least 2 hunks to test NextHunk
		}
		startHunk := rapid.IntRange(0, numHunks-2).Draw(t, "startHunk")
		panel.currentHunk = startHunk
		panel.viewport.SetYOffset(panel.hunks[startHunk].StartLine + headerLines)

		panel.NextHunk()

		expectedHunk := startHunk + 1
		expectedOffset := panel.hunks[expectedHunk].StartLine + headerLines

		if panel.currentHunk != expectedHunk {
			t.Fatalf("expected currentHunk=%d, got %d", expectedHunk, panel.currentHunk)
		}
		if panel.viewport.YOffset != expectedOffset {
			t.Fatalf("expected YOffset=%d, got %d", expectedOffset, panel.viewport.YOffset)
		}
	})
}

// Property: NextHunk from noHunkSelected goes to first hunk
func TestNextHunk_FromNoSelection_GoesToFirstHunk(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, _, headerLines := setupPanelWithHunks(t)

		panel.currentHunk = noHunkSelected
		panel.viewport.SetYOffset(0)

		panel.NextHunk()

		expectedOffset := panel.hunks[0].StartLine + headerLines

		if panel.currentHunk != 0 {
			t.Fatalf("expected currentHunk=0, got %d", panel.currentHunk)
		}
		if panel.viewport.YOffset != expectedOffset {
			t.Fatalf("expected YOffset=%d, got %d", expectedOffset, panel.viewport.YOffset)
		}
	})
}

// Property: NextHunk at last hunk stays at last hunk
func TestNextHunk_AtLastHunk_StaysAtLastHunk(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, numHunks, headerLines := setupPanelWithHunks(t)

		lastHunk := numHunks - 1
		panel.currentHunk = lastHunk
		expectedOffset := panel.hunks[lastHunk].StartLine + headerLines
		panel.viewport.SetYOffset(expectedOffset)

		panel.NextHunk()

		if panel.currentHunk != lastHunk {
			t.Fatalf("expected currentHunk=%d (unchanged), got %d", lastHunk, panel.currentHunk)
		}
		if panel.viewport.YOffset != expectedOffset {
			t.Fatalf("expected YOffset=%d (unchanged), got %d", expectedOffset, panel.viewport.YOffset)
		}
	})
}

// Property: PrevHunk at start of hunk goes to previous hunk
func TestPrevHunk_AtHunkStart_GoesToPreviousHunk(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, numHunks, headerLines := setupPanelWithHunks(t)

		if numHunks < 2 {
			return // Need at least 2 hunks
		}
		// Start at hunk 1 or later
		startHunk := rapid.IntRange(1, numHunks-1).Draw(t, "startHunk")
		panel.currentHunk = startHunk
		panel.viewport.SetYOffset(panel.hunks[startHunk].StartLine + headerLines)

		panel.PrevHunk()

		expectedHunk := startHunk - 1
		expectedOffset := panel.hunks[expectedHunk].StartLine + headerLines

		if panel.currentHunk != expectedHunk {
			t.Fatalf("expected currentHunk=%d, got %d", expectedHunk, panel.currentHunk)
		}
		if panel.viewport.YOffset != expectedOffset {
			t.Fatalf("expected YOffset=%d, got %d", expectedOffset, panel.viewport.YOffset)
		}
	})
}

// Property: PrevHunk in middle of hunk goes to start of current hunk
func TestPrevHunk_InMiddleOfHunk_GoesToHunkStart(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, numHunks, headerLines := setupPanelWithHunks(t)

		// Pick a hunk with size > 1 so we can be "in the middle"
		var hunkIdx int
		var hunk jj.Hunk
		for i := 0; i < numHunks; i++ {
			h := panel.hunks[i]
			if h.EndLine > h.StartLine { // Has at least 2 lines
				hunkIdx = i
				hunk = h
				break
			}
		}
		if hunk.EndLine == hunk.StartLine {
			return // No multi-line hunks, skip
		}

		panel.currentHunk = hunkIdx
		// Position somewhere after the start but within the hunk
		offsetInHunk := rapid.IntRange(1, hunk.EndLine-hunk.StartLine).Draw(t, "offsetInHunk")
		panel.viewport.SetYOffset(hunk.StartLine + headerLines + offsetInHunk)

		panel.PrevHunk()

		expectedOffset := hunk.StartLine + headerLines

		if panel.currentHunk != hunkIdx {
			t.Fatalf("expected currentHunk=%d (same), got %d", hunkIdx, panel.currentHunk)
		}
		if panel.viewport.YOffset != expectedOffset {
			t.Fatalf("expected YOffset=%d (hunk start), got %d", expectedOffset, panel.viewport.YOffset)
		}
	})
}

// Property: PrevHunk at first hunk start goes to top (noHunkSelected)
func TestPrevHunk_AtFirstHunkStart_GoesToTop(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, _, headerLines := setupPanelWithHunks(t)

		panel.currentHunk = 0
		panel.viewport.SetYOffset(panel.hunks[0].StartLine + headerLines)

		panel.PrevHunk()

		if panel.currentHunk != noHunkSelected {
			t.Fatalf("expected currentHunk=noHunkSelected, got %d", panel.currentHunk)
		}
		if panel.viewport.YOffset != 0 {
			t.Fatalf("expected YOffset=0, got %d", panel.viewport.YOffset)
		}
	})
}

// Property: PrevHunk with noHunkSelected stays at top
func TestPrevHunk_NoHunkSelected_StaysAtTop(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, _, _ := setupPanelWithHunks(t)

		panel.currentHunk = noHunkSelected
		panel.viewport.SetYOffset(0)

		panel.PrevHunk()

		if panel.currentHunk != noHunkSelected {
			t.Fatalf("expected currentHunk=noHunkSelected, got %d", panel.currentHunk)
		}
		if panel.viewport.YOffset != 0 {
			t.Fatalf("expected YOffset=0, got %d", panel.viewport.YOffset)
		}
	})
}

// Property: syncCurrentHunk correctly identifies which hunk contains viewport
func TestSyncCurrentHunk_IdentifiesCorrectHunk(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, numHunks, headerLines := setupPanelWithHunks(t)

		// Pick a random hunk and position within it
		hunkIdx := rapid.IntRange(0, numHunks-1).Draw(t, "hunkIdx")
		hunk := panel.hunks[hunkIdx]
		offsetInHunk := rapid.IntRange(0, hunk.EndLine-hunk.StartLine).Draw(t, "offsetInHunk")
		panel.viewport.SetYOffset(hunk.StartLine + headerLines + offsetInHunk)
		panel.currentHunk = 999 // Wrong value

		panel.syncCurrentHunk()

		if panel.currentHunk != hunkIdx {
			t.Fatalf("expected currentHunk=%d, got %d (viewport at %d, hunk range %d-%d)",
				hunkIdx, panel.currentHunk, panel.viewport.YOffset, hunk.StartLine, hunk.EndLine)
		}
	})
}

// Property: syncCurrentHunk in header area sets noHunkSelected
func TestSyncCurrentHunk_InHeader_SetsNoHunkSelected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, _, headerLines := setupPanelWithHunks(t)

		if headerLines == 0 {
			return // No header to test
		}

		// Position in header (before first hunk)
		panel.viewport.SetYOffset(rapid.IntRange(0, headerLines-1).Draw(t, "headerPos"))
		panel.currentHunk = 999 // Wrong value

		panel.syncCurrentHunk()

		if panel.currentHunk != noHunkSelected {
			t.Fatalf("expected currentHunk=noHunkSelected, got %d", panel.currentHunk)
		}
	})
}

// Property: Viewport offset after navigation always accounts for headerLines
func TestHunkNavigation_ViewportIncludesHeaderOffset(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, numHunks, headerLines := setupPanelWithHunks(t)

		// Navigate to a random hunk
		targetHunk := rapid.IntRange(0, numHunks-1).Draw(t, "targetHunk")

		// Start from noHunkSelected and navigate forward
		panel.currentHunk = noHunkSelected
		panel.viewport.SetYOffset(0)

		for i := 0; i <= targetHunk; i++ {
			panel.NextHunk()
		}

		expectedOffset := panel.hunks[targetHunk].StartLine + headerLines

		if panel.viewport.YOffset != expectedOffset {
			t.Fatalf("expected YOffset=%d (hunk %d start %d + header %d), got %d",
				expectedOffset, targetHunk, panel.hunks[targetHunk].StartLine, headerLines, panel.viewport.YOffset)
		}
	})
}

// =============================================================================
// Mouse Support Property Tests
// =============================================================================

// setupScrollablePanel creates a panel with enough content to scroll
func setupScrollablePanel(t *rapid.T) (*DiffPanel, int, int) {
	panel := NewDiffPanel()

	viewportHeight := rapid.IntRange(10, 50).Draw(t, "viewportHeight")
	// Ensure at least 10 lines more than viewport so there's room to scroll
	numLines := rapid.IntRange(viewportHeight+10, 300).Draw(t, "numLines")

	panel.SetSize(80, viewportHeight+3) // +3 for border and title

	// Create content without trailing newline to avoid off-by-one in line count
	// strings.Split("a\nb\n", "\n") = ["a", "b", ""] (3 elements, not 2)
	content := strings.TrimSuffix(strings.Repeat("line\n", numLines), "\n")
	panel.viewport.SetContent(content)

	return &panel, numLines, viewportHeight
}

// Property: After any sequence of mouse scroll events, viewport stays within bounds
func TestDiffPanel_MouseScroll_StaysInBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, numLines, viewportHeight := setupScrollablePanel(t)

		// Random starting position
		startOffset := rapid.IntRange(0, numLines-viewportHeight).Draw(t, "startOffset")
		panel.viewport.SetYOffset(startOffset)

		// Perform random scroll events
		numScrolls := rapid.IntRange(0, 100).Draw(t, "numScrolls")
		for i := 0; i < numScrolls; i++ {
			scrollUp := rapid.Bool().Draw(t, "scrollUp")
			if scrollUp {
				panel.HandleMouseScroll(tea.MouseButtonWheelUp)
			} else {
				panel.HandleMouseScroll(tea.MouseButtonWheelDown)
			}
		}

		// Invariant: YOffset in valid range [0, maxScroll]
		maxScroll := numLines - viewportHeight
		if maxScroll < 0 {
			maxScroll = 0
		}
		if panel.viewport.YOffset < 0 {
			t.Fatalf("YOffset should be >= 0, got %d", panel.viewport.YOffset)
		}
		if panel.viewport.YOffset > maxScroll {
			t.Fatalf("YOffset should be <= %d, got %d", maxScroll, panel.viewport.YOffset)
		}
	})
}

// Property: Mouse wheel up decreases YOffset (when not at top)
func TestDiffPanel_MouseWheelUp_ScrollsUp(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, numLines, viewportHeight := setupScrollablePanel(t)

		// Start at a random non-zero position (so we can scroll up)
		maxScroll := numLines - viewportHeight
		if maxScroll <= 0 {
			return // Can't scroll
		}
		startOffset := rapid.IntRange(mouseScrollLines, maxScroll).Draw(t, "startOffset") // At least mouseScrollLines so scroll has room
		panel.viewport.SetYOffset(startOffset)

		beforeOffset := panel.viewport.YOffset
		panel.HandleMouseScroll(tea.MouseButtonWheelUp)

		// Invariant: offset must decrease (we had room to scroll)
		if panel.viewport.YOffset >= beforeOffset {
			t.Fatalf("wheel up should decrease offset: before=%d, after=%d",
				beforeOffset, panel.viewport.YOffset)
		}
	})
}

// Property: Mouse wheel down increases YOffset (when not at bottom)
func TestDiffPanel_MouseWheelDown_ScrollsDown(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, numLines, viewportHeight := setupScrollablePanel(t)

		maxScroll := numLines - viewportHeight
		if maxScroll <= mouseScrollLines {
			return // Not enough room to scroll
		}
		// Start at a position with room to scroll down
		startOffset := rapid.IntRange(0, maxScroll-mouseScrollLines).Draw(t, "startOffset")
		panel.viewport.SetYOffset(startOffset)

		beforeOffset := panel.viewport.YOffset
		panel.HandleMouseScroll(tea.MouseButtonWheelDown)

		// Invariant: offset must increase (we had room to scroll)
		if panel.viewport.YOffset <= beforeOffset {
			t.Fatalf("wheel down should increase offset: before=%d, after=%d",
				beforeOffset, panel.viewport.YOffset)
		}
	})
}

// Property: Mouse scroll syncs currentHunk correctly
func TestDiffPanel_MouseScroll_SyncsHunk(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel, numHunks, headerLines := setupPanelWithHunks(t)

		// Scroll randomly
		numScrolls := rapid.IntRange(1, 20).Draw(t, "numScrolls")
		for i := 0; i < numScrolls; i++ {
			scrollUp := rapid.Bool().Draw(t, "scrollUp")
			if scrollUp {
				panel.HandleMouseScroll(tea.MouseButtonWheelUp)
			} else {
				panel.HandleMouseScroll(tea.MouseButtonWheelDown)
			}
		}

		// Invariant: currentHunk matches viewport position
		pos := panel.viewport.YOffset - headerLines
		expectedHunk := noHunkSelected
		for i := numHunks - 1; i >= 0; i-- {
			if pos >= panel.hunks[i].StartLine {
				expectedHunk = i
				break
			}
		}

		if panel.currentHunk != expectedHunk {
			t.Fatalf("currentHunk=%d doesn't match viewport position (expected %d, offset=%d, headerLines=%d)",
				panel.currentHunk, expectedHunk, panel.viewport.YOffset, headerLines)
		}
	})
}

// Benchmark for ParseDetailsFromShow
func BenchmarkParseDetailsFromShow(b *testing.B) {
	input := `Commit ID: abc123def456789
Change ID: xsssnyuxabcd
Author: test@example.com
Timestamp: 2026-01-29 12:00:00
Description: This is a test commit with a longer description
that spans multiple lines and contains various details
about the changes made.

diff --git a/main.go b/main.go
@@ -1,5 +1,10 @@`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseDetailsFromShow(input)
	}
}
