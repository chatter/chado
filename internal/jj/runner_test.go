package jj

import (
	"strings"
	"testing"

	"github.com/chatter/chado/internal/logger"
	"pgregory.net/rapid"
)

// testLogger creates a no-op logger for tests
func testLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, _ := logger.New("")
	return log
}

// =============================================================================
// Unit Tests
// =============================================================================

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ansi codes",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "simple color code",
			input:    "\x1b[31mred\x1b[0m",
			expected: "red",
		},
		{
			name:     "multiple color codes",
			input:    "\x1b[1m\x1b[32mbold green\x1b[0m normal",
			expected: "bold green normal",
		},
		{
			name:     "256 color code",
			input:    "\x1b[38;5;196mred256\x1b[0m",
			expected: "red256",
		},
		{
			name:     "true color code",
			input:    "\x1b[38;2;255;0;0mtruered\x1b[0m",
			expected: "truered",
		},
		{
			name:     "jj log line with graph",
			input:    "\x1b[1;35m@\x1b[0m  \x1b[1;34mxsssnyux\x1b[0m test",
			expected: "@  xsssnyux test",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only ansi codes",
			input:    "\x1b[31m\x1b[0m",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseFiles(t *testing.T) {
	runner := NewRunner(".", testLogger(t))

	tests := []struct {
		name     string
		input    string
		expected []File
	}{
		{
			name:     "empty diff",
			input:    "",
			expected: nil,
		},
		{
			name: "jj format - added file",
			input: `Added regular file main.go:
        1: package main
        2: 
        3: func main() {}`,
			expected: []File{
				{Path: "main.go", Status: FileAdded},
			},
		},
		{
			name: "jj format - modified file",
			input: `Modified regular file internal/app/app.go:
   1    1: package app
        2: import "fmt"`,
			expected: []File{
				{Path: "internal/app/app.go", Status: FileModified},
			},
		},
		{
			name: "jj format - removed file",
			input: `Removed regular file old.txt:
        1: old content`,
			expected: []File{
				{Path: "old.txt", Status: FileDeleted},
			},
		},
		{
			name: "jj format - multiple files",
			input: `Added regular file new.go:
        1: package new
Modified regular file existing.go:
   1    1: package existing
Removed regular file deprecated.go:
        1: package deprecated`,
			expected: []File{
				{Path: "new.go", Status: FileAdded},
				{Path: "existing.go", Status: FileModified},
				{Path: "deprecated.go", Status: FileDeleted},
			},
		},
		{
			name: "git format - added file",
			input: `diff --git a/main.go b/main.go
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/main.go`,
			expected: []File{
				{Path: "main.go", Status: FileAdded},
			},
		},
		{
			name: "git format - deleted file",
			input: `diff --git a/old.txt b/old.txt
deleted file mode 100644
index 1234567..0000000
--- a/old.txt
+++ /dev/null`,
			expected: []File{
				{Path: "old.txt", Status: FileDeleted},
			},
		},
		{
			name: "git format - modified file",
			input: `diff --git a/main.go b/main.go
index 1234567..abcdefg 100644
--- a/main.go
+++ b/main.go`,
			expected: []File{
				{Path: "main.go", Status: FileModified},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.ParseFiles(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ParseFiles() returned %d files, want %d", len(result), len(tt.expected))
				return
			}
			for i, file := range result {
				if file.Path != tt.expected[i].Path {
					t.Errorf("file[%d].Path = %q, want %q", i, file.Path, tt.expected[i].Path)
				}
				if file.Status != tt.expected[i].Status {
					t.Errorf("file[%d].Status = %q, want %q", i, file.Status, tt.expected[i].Status)
				}
			}
		})
	}
}

func TestFindHunks(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
	}{
		{
			name:          "empty diff",
			input:         "",
			expectedCount: 0,
		},
		{
			name: "git style hunks",
			input: `diff --git a/main.go b/main.go
@@ -1,5 +1,6 @@
 package main
+import "fmt"
@@ -10,3 +11,5 @@
 func main() {}
+fmt.Println("hi")`,
			expectedCount: 2,
		},
		{
			name: "jj style file sections",
			input: `Added regular file main.go:
        1: package main
Modified regular file app.go:
   1    1: package app`,
			expectedCount: 2,
		},
		{
			name: "mixed format",
			input: `Added regular file new.go:
        1: package new
diff --git a/old.go b/old.go
@@ -1,3 +1,4 @@
 package old`,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunks := FindHunks(tt.input)
			if len(hunks) != tt.expectedCount {
				t.Errorf("FindHunks() returned %d hunks, want %d", len(hunks), tt.expectedCount)
			}
		})
	}
}

func TestFindHunks_ValidLineRanges(t *testing.T) {
	input := `Added regular file main.go:
        1: package main
        2: 
        3: func main() {}
Modified regular file app.go:
   1    1: package app
   2    2: 
   3    3: func init() {}`

	hunks := FindHunks(input)
	lines := strings.Split(input, "\n")

	for i, hunk := range hunks {
		if hunk.StartLine < 0 {
			t.Errorf("hunk[%d].StartLine = %d, should be >= 0", i, hunk.StartLine)
		}
		if hunk.EndLine < hunk.StartLine {
			t.Errorf("hunk[%d].EndLine = %d < StartLine = %d", i, hunk.EndLine, hunk.StartLine)
		}
		if hunk.EndLine >= len(lines) {
			t.Errorf("hunk[%d].EndLine = %d >= len(lines) = %d", i, hunk.EndLine, len(lines))
		}
	}
}

func TestParseLogLines(t *testing.T) {
	runner := NewRunner(".", testLogger(t))

	tests := []struct {
		name          string
		input         string
		expectedCount int
	}{
		{
			name:          "empty log",
			input:         "",
			expectedCount: 0,
		},
		{
			name: "single change",
			input: `@  xsssnyux test@example.com 2026-01-29 12:00:00 abc123
│  test description`,
			expectedCount: 1,
		},
		{
			name: "multiple changes",
			input: `@  xsssnyux test@example.com 2026-01-29 12:00:00 abc123
│  first description
○  nlkzwoyt test@example.com 2026-01-28 12:00:00 def456
│  second description
◆  zzzzzzzz root() 00000000`,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := runner.ParseLogLines(tt.input)
			if len(changes) != tt.expectedCount {
				t.Errorf("ParseLogLines() returned %d changes, want %d", len(changes), tt.expectedCount)
			}
		})
	}
}

// =============================================================================
// Property Tests
// =============================================================================

// Property: stripANSI should never increase string length
func TestStripANSI_NeverIncreasesLength(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.String().Draw(t, "input")
		result := stripANSI(input)
		if len(result) > len(input) {
			t.Fatalf("stripANSI increased length: input=%d, result=%d", len(input), len(result))
		}
	})
}

// Property: stripANSI should be idempotent
func TestStripANSI_Idempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.String().Draw(t, "input")
		once := stripANSI(input)
		twice := stripANSI(once)
		if once != twice {
			t.Fatalf("stripANSI not idempotent: once=%q, twice=%q", once, twice)
		}
	})
}

// Property: stripANSI result should not contain escape sequences
func TestStripANSI_NoEscapeSequences(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := ansiString().Draw(t, "input")
		result := stripANSI(input)
		if strings.Contains(result, "\x1b[") {
			t.Fatalf("stripANSI result still contains escape sequence: %q", result)
		}
	})
}

// Generator for strings with ANSI codes
func ansiString() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		parts := rapid.SliceOf(rapid.OneOf(
			rapid.Just("\x1b[31m"),
			rapid.Just("\x1b[0m"),
			rapid.Just("\x1b[1;32m"),
			rapid.Just("\x1b[38;5;196m"),
			rapid.StringMatching(`[a-zA-Z0-9 ]{0,20}`),
		)).Draw(t, "parts")
		return strings.Join(parts, "")
	})
}

// Property: stripANSI on strings with ANSI codes should remove all codes
func TestStripANSI_RemovesAllCodes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := ansiString().Draw(t, "ansiInput")
		result := stripANSI(input)
		// Result should not contain any escape character
		if strings.ContainsRune(result, '\x1b') {
			t.Fatalf("result contains escape char: %q", result)
		}
	})
}

// Property: FindHunks should return non-overlapping ranges
func TestFindHunks_NonOverlapping(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a plausible diff-like string
		numSections := rapid.IntRange(0, 10).Draw(t, "numSections")
		var lines []string
		for range numSections {
			// Randomly choose jj or git style header
			if rapid.Bool().Draw(t, "isJJStyle") {
				status := rapid.SampledFrom([]string{"Added", "Modified", "Removed"}).Draw(t, "status")
				filename := rapid.StringMatching(`[a-z]{1,10}\.go`).Draw(t, "filename")
				lines = append(lines, status+" regular file "+filename+":")
			} else {
				lines = append(lines, "@@ -1,5 +1,6 @@")
			}
			// Add some content lines
			contentLines := rapid.IntRange(1, 5).Draw(t, "contentLines")
			for range contentLines {
				lines = append(lines, "  content line")
			}
		}

		input := strings.Join(lines, "\n")
		hunks := FindHunks(input)

		// Check non-overlapping
		for i := 1; i < len(hunks); i++ {
			if hunks[i].StartLine <= hunks[i-1].EndLine {
				t.Fatalf("hunks overlap: hunk[%d].EndLine=%d, hunk[%d].StartLine=%d",
					i-1, hunks[i-1].EndLine, i, hunks[i].StartLine)
			}
		}
	})
}

// Property: ParseFiles should never return duplicate paths
func TestParseFiles_NoDuplicatePaths(t *testing.T) {
	runner := NewRunner(".", testLogger(t))

	rapid.Check(t, func(t *rapid.T) {
		// Generate a diff with unique filenames
		numFiles := rapid.IntRange(0, 10).Draw(t, "numFiles")
		var lines []string
		for i := range numFiles {
			status := rapid.SampledFrom([]string{"Added", "Modified", "Removed"}).Draw(t, "status")
			// Use index to ensure uniqueness
			filename := rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "basename")
			lines = append(lines, status+" regular file "+filename+"_"+string(rune('a'+i))+".go:")
			lines = append(lines, "        1: package test")
		}

		input := strings.Join(lines, "\n")
		files := runner.ParseFiles(input)

		// Check for duplicates
		seen := make(map[string]bool)
		for _, file := range files {
			if seen[file.Path] {
				t.Fatalf("duplicate path found: %s", file.Path)
			}
			seen[file.Path] = true
		}
	})
}

// Property: File status should only be one of the valid values
func TestParseFiles_ValidStatus(t *testing.T) {
	runner := NewRunner(".", testLogger(t))

	rapid.Check(t, func(t *rapid.T) {
		status := rapid.SampledFrom([]string{"Added", "Modified", "Removed"}).Draw(t, "status")
		filename := rapid.StringMatching(`[a-z]{3,10}\.go`).Draw(t, "filename")
		input := status + " regular file " + filename + ":\n        1: content"

		files := runner.ParseFiles(input)
		for _, file := range files {
			switch file.Status {
			case FileAdded, FileModified, FileDeleted, FileRenamed, FileCopied:
				// Valid
			default:
				t.Fatalf("invalid file status: %v", file.Status)
			}
		}
	})
}

// =============================================================================
// Operation Log Tests
// =============================================================================

func TestParseOpLogLines(t *testing.T) {
	runner := NewRunner(".", testLogger(t))

	tests := []struct {
		name          string
		input         string
		expectedCount int
		checkFirst    func(op Operation) bool
	}{
		{
			name:          "empty log",
			input:         "",
			expectedCount: 0,
		},
		{
			name: "single operation",
			input: `@  bbc9fee12c4d user@host 4 minutes ago, lasted 1 second
│  snapshot working copy
│  args: jj log`,
			expectedCount: 1,
			checkFirst: func(op Operation) bool {
				return op.OpID == "bbc9fee12c4d" && op.Args == "jj log"
			},
		},
		{
			name: "multiple operations",
			input: `@  bbc9fee12c4d user@host 4 minutes ago
│  snapshot working copy
│  args: jj log
○  86d0094c958f user@host 4 days ago
│  push bookmark main
│  args: jj git push`,
			expectedCount: 2,
			checkFirst: func(op Operation) bool {
				return op.OpID == "bbc9fee12c4d"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operations := runner.ParseOpLogLines(tt.input)
			if len(operations) != tt.expectedCount {
				t.Errorf("ParseOpLogLines() returned %d operations, want %d", len(operations), tt.expectedCount)
				return
			}
			if tt.checkFirst != nil && len(operations) > 0 {
				if !tt.checkFirst(operations[0]) {
					t.Errorf("first operation check failed: %+v", operations[0])
				}
			}
		})
	}
}

func TestParseOpLogLines_ArgsExtraction(t *testing.T) {
	runner := NewRunner(".", testLogger(t))

	input := `@  aaaaaaaaaaaa user@host now
│  snapshot working copy
│  args: jj log --color=always`

	operations := runner.ParseOpLogLines(input)
	if len(operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(operations))
	}

	if operations[0].Args != "jj log --color=always" {
		t.Errorf("expected args 'jj log --color=always', got '%s'", operations[0].Args)
	}
}

// Property: All parsed operations should have non-empty OpID
func TestParseOpLogLines_ValidOpID(t *testing.T) {
	runner := NewRunner(".", testLogger(t))

	rapid.Check(t, func(t *rapid.T) {
		// Generate valid op log format
		numOps := rapid.IntRange(0, 10).Draw(t, "numOps")
		var lines []string
		for i := range numOps {
			symbol := "@"
			if i > 0 {
				symbol = "○"
			}
			opID := rapid.StringMatching(`[0-9a-f]{12}`).Draw(t, "opID")
			lines = append(lines, symbol+"  "+opID+" user@host now")
			lines = append(lines, "│  description")
		}
		input := strings.Join(lines, "\n")

		operations := runner.ParseOpLogLines(input)
		for i, op := range operations {
			if op.OpID == "" {
				t.Fatalf("operation[%d] has empty OpID", i)
			}
			if len(op.OpID) != 12 {
				t.Fatalf("operation[%d] OpID should be 12 chars, got %d: %s", i, len(op.OpID), op.OpID)
			}
		}
	})
}
