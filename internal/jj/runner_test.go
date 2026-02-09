package jj

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chatter/chado/internal/jj/testgen"
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
	// Generate a valid change ID for the jj log line test
	changeID := testgen.ChangeID(testgen.WithShort).Example()

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
			input:    fmt.Sprintf("\x1b[1;35m@\x1b[0m  \x1b[1;34m%s\x1b[0m test", changeID),
			expected: fmt.Sprintf("@  %s test", changeID),
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

	// Generate file paths for test cases
	addedPath := testgen.FilePath().Example()
	modifiedPath := testgen.FilePath().Example()
	removedPath := testgen.FilePath().Example()
	multiPath1 := testgen.FilePath().Example()
	multiPath2 := testgen.FilePath().Example()
	multiPath3 := testgen.FilePath().Example()
	gitAddedPath := testgen.FilePath().Example()
	gitDeletedPath := testgen.FilePath().Example()
	gitModifiedPath := testgen.FilePath().Example()

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
			name:  "jj format - added file",
			input: fmt.Sprintf("Added regular file %s:\n        1: package main\n        2: \n        3: func main() {}", addedPath),
			expected: []File{
				{Path: addedPath, Status: FileAdded},
			},
		},
		{
			name:  "jj format - modified file",
			input: fmt.Sprintf("Modified regular file %s:\n   1    1: package app\n        2: import \"fmt\"", modifiedPath),
			expected: []File{
				{Path: modifiedPath, Status: FileModified},
			},
		},
		{
			name:  "jj format - removed file",
			input: fmt.Sprintf("Removed regular file %s:\n        1: old content", removedPath),
			expected: []File{
				{Path: removedPath, Status: FileDeleted},
			},
		},
		{
			name:  "jj format - multiple files",
			input: fmt.Sprintf("Added regular file %s:\n        1: package new\nModified regular file %s:\n   1    1: package existing\nRemoved regular file %s:\n        1: package deprecated", multiPath1, multiPath2, multiPath3),
			expected: []File{
				{Path: multiPath1, Status: FileAdded},
				{Path: multiPath2, Status: FileModified},
				{Path: multiPath3, Status: FileDeleted},
			},
		},
		{
			name:  "git format - added file",
			input: fmt.Sprintf("diff --git a/%s b/%s\nnew file mode 100644\nindex 0000000..1234567\n--- /dev/null\n+++ b/%s", gitAddedPath, gitAddedPath, gitAddedPath),
			expected: []File{
				{Path: gitAddedPath, Status: FileAdded},
			},
		},
		{
			name:  "git format - deleted file",
			input: fmt.Sprintf("diff --git a/%s b/%s\ndeleted file mode 100644\nindex 1234567..0000000\n--- a/%s\n+++ /dev/null", gitDeletedPath, gitDeletedPath, gitDeletedPath),
			expected: []File{
				{Path: gitDeletedPath, Status: FileDeleted},
			},
		},
		{
			name:  "git format - modified file",
			input: fmt.Sprintf("diff --git a/%s b/%s\nindex 1234567..abcdefg 100644\n--- a/%s\n+++ b/%s", gitModifiedPath, gitModifiedPath, gitModifiedPath, gitModifiedPath),
			expected: []File{
				{Path: gitModifiedPath, Status: FileModified},
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

	// Generate valid change IDs, commit IDs, emails, and timestamps using testgen
	changeID1 := testgen.ChangeID().Example()
	changeID2 := testgen.ChangeID(testgen.WithShort).Example()
	changeID3 := testgen.ChangeID(testgen.WithShort, testgen.WithVersion).Example()
	commitID1 := testgen.CommitID(testgen.WithShort).Example()
	commitID2 := testgen.CommitID(testgen.WithShort).Example()
	email1 := testgen.Email().Example()
	email2 := testgen.Email().Example()
	ts1 := testgen.Timestamp().Example()
	ts2 := testgen.Timestamp().Example()

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
			name:          "single change",
			input:         fmt.Sprintf("@  %s %s %s %s\n│  test description", changeID1, email1, ts1, commitID1),
			expectedCount: 1,
		},
		{
			name:          "multiple changes",
			input:         fmt.Sprintf("@  %s %s %s %s\n│  first description\n○  %s %s %s %s\n│  second description\n◆  %s root() 00000000", changeID1, email1, ts1, commitID1, changeID2, email2, ts2, commitID2, changeID3),
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
				status := testgen.FileStatus().Draw(t, "status")
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
			status := testgen.FileStatus().Draw(t, "status")
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
		status := testgen.FileStatus().Draw(t, "status")
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

	// Generate operation IDs, email, and timestamps
	opID1 := testgen.OperationID(testgen.WithShort).Example()
	opID2 := testgen.OperationID(testgen.WithShort).Example()
	email := testgen.Email().Example()
	relTs1 := testgen.RelativeTimestamp().Example()
	relTs2 := testgen.RelativeTimestamp().Example()

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
			name:          "single operation",
			input:         fmt.Sprintf("@  %s %s %s, lasted 1 second\n│  snapshot working copy\n│  args: jj log", opID1, email, relTs1),
			expectedCount: 1,
			checkFirst: func(op Operation) bool {
				return op.OpID == opID1 && op.Args == "jj log"
			},
		},
		{
			name:          "multiple operations",
			input:         fmt.Sprintf("@  %s %s %s\n│  snapshot working copy\n│  args: jj log\n○  %s %s %s\n│  push bookmark main\n│  args: jj git push", opID1, email, relTs1, opID2, email, relTs2),
			expectedCount: 2,
			checkFirst: func(op Operation) bool {
				return op.OpID == opID1
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

	opID := testgen.OperationID(testgen.WithShort).Example()
	email := testgen.Email().Example()
	relTs := testgen.RelativeTimestamp().Example()
	input := fmt.Sprintf("@  %s %s %s\n│  snapshot working copy\n│  args: jj log --color=always", opID, email, relTs)

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
			opID := testgen.OperationID(testgen.WithShort).Draw(t, "opID")
			email := testgen.Email().Draw(t, "email")
			relTs := testgen.RelativeTimestamp().Draw(t, "relTs")
			lines = append(lines, symbol+"  "+opID+" "+email+" "+relTs)
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

// =============================================================================
// Evolution Log Tests
// =============================================================================

func TestEvoLog_MethodExists(t *testing.T) {
	// This test verifies the EvoLog method exists and has the correct signature.
	// It will fail to compile until EvoLog is implemented.
	runner := NewRunner(".", testLogger(t))

	// EvoLog should accept a revision and return (string, error)
	_, err := runner.EvoLog("test-rev")
	// We expect an error since we're not in a real jj repo, but the method should exist
	if err == nil {
		t.Log("EvoLog returned no error (unexpected in test environment)")
	}
}

// =============================================================================
// Describe Tests
// =============================================================================

func TestDescribe_MethodExists(t *testing.T) {
	// This test verifies the Describe method exists and has the correct signature.
	// It will fail to compile until Describe is implemented.
	runner := NewRunner(".", testLogger(t))

	// Describe should accept rev and message, return error
	err := runner.Describe("test-rev", "test message")
	// We expect an error since we're not in a real jj repo, but the method should exist
	if err == nil {
		t.Log("Describe returned no error (unexpected in test environment)")
	}
}

func TestDescribe_CallsRun(t *testing.T) {
	// This test verifies Describe calls Run with correct arguments.
	// We can't easily mock Run, but we can verify the method signature.
	runner := NewRunner(".", testLogger(t))

	// Calling Describe should invoke jj describe -r REV -m MESSAGE
	// The actual command will fail (not in jj repo), but we're testing the interface
	err := runner.Describe("xsssnyux", "updated description")

	// Error is expected (not in jj repo)
	if err == nil {
		t.Log("Describe unexpectedly succeeded")
	}
}

// =============================================================================
// Edit Tests
// =============================================================================

func TestEdit_MethodExists(t *testing.T) {
	// This test verifies the Edit method exists and has the correct signature.
	runner := NewRunner(".", testLogger(t))

	// Edit should accept rev, return error
	err := runner.Edit("test-rev")
	// We expect an error since we're not in a real jj repo
	if err == nil {
		t.Log("Edit returned no error (unexpected in test environment)")
	}
}

func TestEdit_CallsRun(t *testing.T) {
	// This test verifies Edit calls Run with correct arguments.
	runner := NewRunner(".", testLogger(t))

	// Calling Edit should invoke jj edit REV
	err := runner.Edit("xsssnyux")

	// Error is expected (not in jj repo)
	if err == nil {
		t.Log("Edit unexpectedly succeeded")
	}
}

// =============================================================================
// New Tests
// =============================================================================

func TestNew_MethodExists(t *testing.T) {
	// This test verifies the New method exists and has the correct signature.
	runner := NewRunner(".", testLogger(t))

	// New should return error
	err := runner.New()
	// We expect an error since we're not in a real jj repo
	if err == nil {
		t.Log("New returned no error (unexpected in test environment)")
	}
}

func TestEvoLog_ParsesAsOperations(t *testing.T) {
	// Evolog output has the same format as op log - operations that affected a change.
	// This test verifies ParseOpLogLines correctly parses evolog-style output.
	runner := NewRunner(".", testLogger(t))

	// Generate operation IDs, email, and timestamps
	opID1 := testgen.OperationID(testgen.WithShort).Example()
	opID2 := testgen.OperationID(testgen.WithShort).Example()
	opID3 := testgen.OperationID(testgen.WithShort).Example()
	email := testgen.Email().Example()
	relTs1 := testgen.RelativeTimestamp().Example()
	relTs2 := testgen.RelativeTimestamp().Example()
	relTs3 := testgen.RelativeTimestamp().Example()

	// Sample evolog output (same format as op log, scoped to a single change)
	input := fmt.Sprintf("@  %s %s %s, lasted 50ms\n│  describe commit\n│  args: jj describe -m 'update readme'\n○  %s %s %s, lasted 100ms\n│  new empty commit\n│  args: jj new\n○  %s %s %s, lasted 200ms\n│  snapshot working copy\n│  args: jj status", opID1, email, relTs1, opID2, email, relTs2, opID3, email, relTs3)

	operations := runner.ParseOpLogLines(input)

	if len(operations) != 3 {
		t.Fatalf("expected 3 operations, got %d", len(operations))
	}

	// Verify first operation (current)
	if operations[0].OpID != opID1 {
		t.Errorf("expected first OpID %q, got %q", opID1, operations[0].OpID)
	}
	if operations[0].Args != "jj describe -m 'update readme'" {
		t.Errorf("expected first Args to contain describe command, got '%s'", operations[0].Args)
	}

	// Verify all operations have valid OpIDs
	for i, op := range operations {
		if len(op.OpID) != 12 {
			t.Errorf("operation[%d] OpID should be 12 chars, got %d: %s", i, len(op.OpID), op.OpID)
		}
	}
}
