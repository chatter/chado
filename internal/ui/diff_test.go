package ui

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

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
	panel.SetSize(80, 24)

	// Set diff with multiple sections
	diff := `Added regular file main.go:
        1: package main
        2: func main() {}
Added regular file app.go:
        1: package app
        2: func init() {}
Added regular file test.go:
        1: package test`

	panel.SetDiff(diff)

	// Should have 3 hunks (one for each Added line)
	if len(panel.hunks) != 3 {
		t.Errorf("expected 3 hunks, got %d", len(panel.hunks))
	}

	// Test navigation
	if panel.currentHunk != 0 {
		t.Errorf("currentHunk should start at 0, got %d", panel.currentHunk)
	}

	panel.NextHunk()
	if panel.currentHunk != 1 {
		t.Errorf("currentHunk should be 1 after NextHunk, got %d", panel.currentHunk)
	}

	panel.NextHunk()
	if panel.currentHunk != 2 {
		t.Errorf("currentHunk should be 2 after NextHunk, got %d", panel.currentHunk)
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
	if panel.currentHunk != 0 {
		t.Errorf("currentHunk should be 0 after GotoTop, got %d", panel.currentHunk)
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

		// Check invariants
		if len(panel.hunks) == 0 {
			if panel.currentHunk != 0 {
				t.Fatalf("currentHunk should be 0 for empty hunks, got %d", panel.currentHunk)
			}
		} else {
			if panel.currentHunk < 0 {
				t.Fatalf("currentHunk should never be negative, got %d", panel.currentHunk)
			}
			if panel.currentHunk >= len(panel.hunks) {
				t.Fatalf("currentHunk %d should be < len(hunks) %d", panel.currentHunk, len(panel.hunks))
			}
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
		if panel.currentHunk != 0 {
			t.Fatalf("currentHunk should be 0 after GotoTop, got %d", panel.currentHunk)
		}
	})
}

// Property: SetDiff should reset currentHunk to 0
func TestDiffPanel_SetDiffResetsHunk(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewDiffPanel()
		panel.SetSize(80, 24)

		// Set initial diff and navigate
		panel.SetDiff("Added regular file a.go:\n        1: a\nAdded regular file b.go:\n        1: b")
		panel.NextHunk()

		// Set new diff
		numSections := rapid.IntRange(0, 5).Draw(t, "numSections")
		var lines []string
		for i := 0; i < numSections; i++ {
			lines = append(lines, "Modified regular file x"+string(rune('a'+i))+".go:")
		}
		panel.SetDiff(strings.Join(lines, "\n"))

		if panel.currentHunk != 0 {
			t.Fatalf("currentHunk should be 0 after SetDiff, got %d", panel.currentHunk)
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
