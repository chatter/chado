package jj

import (
	"bytes"
	"os/exec"
	"regexp"
	"strings"
)

// Runner executes jj commands and returns output
type Runner struct {
	workDir string
}

// NewRunner creates a new jj command runner
func NewRunner(workDir string) *Runner {
	return &Runner{workDir: workDir}
}

// Run executes a jj command and returns the output with colors preserved
func (r *Runner) Run(args ...string) (string, error) {
	cmd := exec.Command("jj", args...)
	cmd.Dir = r.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Return stderr content for debugging
		if stderr.Len() > 0 {
			return "", &JJError{
				Command: strings.Join(args, " "),
				Stderr:  stderr.String(),
				Err:     err,
			}
		}
		return "", err
	}

	return stdout.String(), nil
}

// Log returns the jj log output with colors
func (r *Runner) Log() (string, error) {
	return r.Run("log", "--color=always")
}

// LogWithTemplate returns jj log with a custom template
func (r *Runner) LogWithTemplate(template string) (string, error) {
	return r.Run("log", "--color=always", "-T", template)
}

// Show returns details for a specific revision
func (r *Runner) Show(rev string) (string, error) {
	return r.Run("show", "-r", rev, "--color=always")
}

// Diff returns the diff for a revision
func (r *Runner) Diff(rev string) (string, error) {
	return r.Run("diff", "-r", rev, "--color=always")
}

// DiffFile returns the diff for a specific file in a revision
func (r *Runner) DiffFile(rev, file string) (string, error) {
	return r.Run("diff", "-r", rev, "--color=always", file)
}

// Status returns jj status output
func (r *Runner) Status() (string, error) {
	return r.Run("status", "--color=always")
}

// LogStat returns log with file stats
func (r *Runner) LogStat(rev string) (string, error) {
	return r.Run("log", "-r", rev, "--stat", "--color=always")
}

// ParseLogLines parses the raw log output into Change structs
// For now, we keep the raw lines and just extract basic info
func (r *Runner) ParseLogLines(output string) []Change {
	lines := strings.Split(output, "\n")
	var changes []Change
	var currentChange *Change
	var descLines []string

	// Regex to detect change lines - requires a graph symbol (@○◆◇●), not just whitespace
	// Matches lines like: "@ xsssnyux ..." or "○ nlkzwoyt ..." or "◆ kyztkmnt ..."
	// Symbols: @ (working copy), ○ (normal), ◆ (immutable), ◇ (empty), ● (hidden), × (conflict)
	changeLineRe := regexp.MustCompile(`^[│├└\s]*[@○◆◇●×]\s*([a-z]{8,})\s`)

	for _, line := range lines {
		stripped := stripANSI(line)
		if match := changeLineRe.FindStringSubmatch(stripped); match != nil {
			// Save previous change if exists
			if currentChange != nil {
				currentChange.Description = strings.TrimSpace(strings.Join(descLines, " "))
				changes = append(changes, *currentChange)
			}

			// Start new change
			changeID := match[1]

			currentChange = &Change{
				ChangeID:      changeID,
				Raw:           line,
				IsWorkingCopy: strings.Contains(stripped, "@"),
				IsImmutable:   strings.Contains(stripped, "◆"),
			}
			descLines = nil
		} else if currentChange != nil && strings.TrimSpace(line) != "" {
			// This is a continuation line (description, etc.)
			// Check if it's a description line (usually starts with │ and spaces)
			if strings.HasPrefix(stripped, "│") || strings.HasPrefix(stripped, " ") {
				desc := strings.TrimSpace(strings.TrimPrefix(stripped, "│"))
				if desc != "" {
					descLines = append(descLines, desc)
				}
			}
			// Keep appending raw lines for display
			currentChange.Raw += "\n" + line
		}
	}

	// Don't forget the last change
	if currentChange != nil {
		currentChange.Description = strings.TrimSpace(strings.Join(descLines, " "))
		changes = append(changes, *currentChange)
	}

	return changes
}

// ParseFiles parses diff output to extract file list
func (r *Runner) ParseFiles(diffOutput string) []File {
	var files []File
	lines := strings.Split(diffOutput, "\n")

	// jj uses format like:
	// "Added regular file path/to/file:"
	// "Modified regular file path/to/file:"
	// "Removed regular file path/to/file:"
	addedRe := regexp.MustCompile(`^Added regular file (.+):$`)
	modifiedRe := regexp.MustCompile(`^Modified regular file (.+):$`)
	removedRe := regexp.MustCompile(`^Removed regular file (.+):$`)

	// Also support git-style diff format (if using git backend with certain configs)
	gitDiffRe := regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	newFileRe := regexp.MustCompile(`^new file mode`)
	deletedFileRe := regexp.MustCompile(`^deleted file mode`)

	for i, line := range lines {
		stripped := stripANSI(line)

		// Check jj native format first
		if match := addedRe.FindStringSubmatch(stripped); match != nil {
			files = append(files, File{Path: match[1], Status: FileAdded})
			continue
		}
		if match := modifiedRe.FindStringSubmatch(stripped); match != nil {
			files = append(files, File{Path: match[1], Status: FileModified})
			continue
		}
		if match := removedRe.FindStringSubmatch(stripped); match != nil {
			files = append(files, File{Path: match[1], Status: FileDeleted})
			continue
		}

		// Fall back to git-style diff format
		if match := gitDiffRe.FindStringSubmatch(stripped); match != nil {
			file := File{
				Path:   match[2],
				Status: FileModified,
			}

			// Check next few lines for status
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				nextLine := stripANSI(lines[j])
				if newFileRe.MatchString(nextLine) {
					file.Status = FileAdded
					break
				} else if deletedFileRe.MatchString(nextLine) {
					file.Status = FileDeleted
					break
				} else if strings.HasPrefix(nextLine, "diff --git") {
					break
				}
			}

			files = append(files, file)
		}
	}

	return files
}

// FindHunks finds all hunk/section positions in diff output
// Supports both git-style @@ hunks and jj-style file headers
func FindHunks(diffOutput string) []Hunk {
	var hunks []Hunk
	lines := strings.Split(diffOutput, "\n")

	// Git-style hunk header
	gitHunkRe := regexp.MustCompile(`^@@.*@@`)
	// jj-style file headers
	jjFileRe := regexp.MustCompile(`^(Added|Modified|Removed) regular file .+:$`)

	var currentHunk *Hunk
	for i, line := range lines {
		stripped := stripANSI(line)

		isSection := gitHunkRe.MatchString(stripped) || jjFileRe.MatchString(stripped)

		if isSection {
			// Close previous hunk
			if currentHunk != nil {
				currentHunk.EndLine = i - 1
				hunks = append(hunks, *currentHunk)
			}
			// Start new hunk/section
			currentHunk = &Hunk{
				Header:    stripped,
				StartLine: i,
			}
		}
	}

	// Close last hunk
	if currentHunk != nil {
		currentHunk.EndLine = len(lines) - 1
		hunks = append(hunks, *currentHunk)
	}

	return hunks
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	ansiRe := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRe.ReplaceAllString(s, "")
}

// JJError represents an error from running a jj command
type JJError struct {
	Command string
	Stderr  string
	Err     error
}

func (e *JJError) Error() string {
	return "jj " + e.Command + ": " + e.Stderr
}
