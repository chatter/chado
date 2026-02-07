package ui

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// Unit Tests
// =============================================================================

func TestReplaceResetWithColor(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		color    string
		expected string
	}{
		{
			name:     "replaces single reset",
			input:    "\x1b[38;5;205mtext\x1b[0m",
			color:    "86",
			expected: "\x1b[38;5;205mtext\x1b[38;5;86m",
		},
		{
			name:     "replaces multiple resets",
			input:    "\x1b[1mfoo\x1b[0mbar\x1b[0m",
			color:    "62",
			expected: "\x1b[1mfoo\x1b[38;5;62mbar\x1b[38;5;62m",
		},
		{
			name:     "no reset present",
			input:    "\x1b[38;5;205mtext",
			color:    "86",
			expected: "\x1b[38;5;205mtext",
		},
		{
			name:     "empty string",
			input:    "",
			color:    "86",
			expected: "",
		},
		{
			name:     "plain text no ansi",
			input:    "plain text",
			color:    "86",
			expected: "plain text",
		},
		{
			name:     "reset only",
			input:    "\x1b[0m",
			color:    "241",
			expected: "\x1b[38;5;241m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplaceResetWithColor(tt.input, tt.color)
			if result != tt.expected {
				t.Errorf("ReplaceResetWithColor(%q, %q) = %q, want %q",
					tt.input, tt.color, result, tt.expected)
			}
		})
	}
}

func TestReplaceResetWithColor_PreservesOtherCodes(t *testing.T) {
	// Verify that other ANSI codes (bold, underline, etc.) are preserved
	input := "\x1b[1;38;5;205mtext\x1b[0m"
	color := "86"

	result := ReplaceResetWithColor(input, color)

	// Should still have the bold+color at the start
	if result[:len("\x1b[1;38;5;205m")] != "\x1b[1;38;5;205m" {
		t.Errorf("expected bold+color prefix to be preserved, got %q", result)
	}

	// Should end with color restoration, not reset
	expectedSuffix := "\x1b[38;5;86m"
	if result[len(result)-len(expectedSuffix):] != expectedSuffix {
		t.Errorf("expected color restoration suffix %q, got %q",
			expectedSuffix, result[len(result)-len(expectedSuffix):])
	}
}

// =============================================================================
// Property Tests
// =============================================================================

// Generator for strings with ANSI codes including resets
func ansiStringWithResets() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		parts := rapid.SliceOf(rapid.OneOf(
			rapid.Just("\x1b[31m"),       // red
			rapid.Just("\x1b[0m"),        // reset
			rapid.Just("\x1b[1;32m"),     // bold green
			rapid.Just("\x1b[38;5;196m"), // 256-color red
			rapid.Just("\x1b[0m"),        // reset (weighted)
			rapid.StringMatching(`[a-zA-Z0-9 ]{0,10}`),
		)).Draw(t, "parts")
		return strings.Join(parts, "")
	})
}

// Generator for valid 256-color codes as strings
func colorGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		code := rapid.IntRange(0, 255).Draw(t, "colorCode")
		return fmt.Sprintf("%d", code)
	})
}

// Property: Result should never contain reset codes
func TestReplaceResetWithColor_NoResetsRemain(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := ansiStringWithResets().Draw(t, "input")
		color := colorGen().Draw(t, "color")

		result := ReplaceResetWithColor(input, color)

		if strings.Contains(result, "\x1b[0m") {
			t.Fatalf("result still contains reset code: %q", result)
		}
	})
}

// Property: If input has no resets, output equals input
func TestReplaceResetWithColor_NoResetPassthrough(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate string without resets
		parts := rapid.SliceOf(rapid.OneOf(
			rapid.Just("\x1b[31m"),
			rapid.Just("\x1b[1;32m"),
			rapid.Just("\x1b[38;5;196m"),
			rapid.StringMatching(`[a-zA-Z0-9 ]{0,10}`),
		)).Draw(t, "parts")
		input := strings.Join(parts, "")

		color := colorGen().Draw(t, "color")

		result := ReplaceResetWithColor(input, color)

		if result != input {
			t.Fatalf("input without resets was modified: input=%q, result=%q", input, result)
		}
	})
}

// Property: Number of color insertions equals number of resets in original
func TestReplaceResetWithColor_ReplacementCount(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := ansiStringWithResets().Draw(t, "input")
		colorCode := rapid.IntRange(0, 255).Draw(t, "color")
		color := fmt.Sprintf("%d", colorCode)

		resetCount := strings.Count(input, "\x1b[0m")
		result := ReplaceResetWithColor(input, color)

		expectedCode := fmt.Sprintf("\x1b[38;5;%dm", colorCode)
		// Count new insertions, not pre-existing occurrences
		preExisting := strings.Count(input, expectedCode)
		totalInResult := strings.Count(result, expectedCode)
		insertedCount := totalInResult - preExisting

		if insertedCount != resetCount {
			t.Fatalf("expected %d color insertions, got %d (input=%q, result=%q, preExisting=%d)",
				resetCount, insertedCount, input, result, preExisting)
		}
	})
}

// Property: Running twice with same color is same as running once (idempotent after first pass)
func TestReplaceResetWithColor_Idempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := ansiStringWithResets().Draw(t, "input")
		color := colorGen().Draw(t, "color")

		once := ReplaceResetWithColor(input, color)
		twice := ReplaceResetWithColor(once, color)

		if once != twice {
			t.Fatalf("not idempotent: once=%q, twice=%q", once, twice)
		}
	})
}

// Property: Plain text (no ANSI codes) passes through unchanged
func TestReplaceResetWithColor_PlainTextUnchanged(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.StringMatching(`[a-zA-Z0-9 ]{0,50}`).Draw(t, "plainText")
		color := colorGen().Draw(t, "color")

		result := ReplaceResetWithColor(input, color)

		if result != input {
			t.Fatalf("plain text was modified: input=%q, result=%q", input, result)
		}
	})
}
