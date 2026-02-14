// Package jj wraps the jj CLI, executing commands and parsing their output.
package jj

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/chatter/chado/internal/logger"
)

// Runner executes jj commands and returns output.
type Runner struct {
	ctx     context.Context
	workDir string
	log     *logger.Logger
}

// NewRunner creates a new jj command runner.
func NewRunner(ctx context.Context, workDir string, log *logger.Logger) *Runner {
	return &Runner{ctx: ctx, workDir: workDir, log: log}
}

// Run executes a jj command and returns the output with colors preserved.
func (r *Runner) Run(args ...string) (string, error) {
	r.log.Debug("executing jj command", "args", args)

	cmd := exec.CommandContext(r.ctx, "jj", args...)
	cmd.Dir = r.workDir

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Return stderr content for debugging
		if stderr.Len() > 0 {
			jjErr := &Error{
				Command: strings.Join(args, " "),
				Stderr:  stderr.String(),
				Err:     err,
			}
			r.log.Error("jj command failed", "args", args, "err", jjErr)

			return "", jjErr
		}

		r.log.Error("jj command failed", "args", args, "err", err)

		return "", fmt.Errorf("jj command failed: %w", err)
	}

	r.log.Debug("jj command completed", "args", args, "output_len", len(stdout.String()))

	return stdout.String(), nil
}

// Log returns the jj log output with colors.
func (r *Runner) Log() (string, error) {
	return r.Run("log", "--color=always")
}

// LogWithTemplate returns jj log with a custom template.
func (r *Runner) LogWithTemplate(template string) (string, error) {
	return r.Run("log", "--color=always", "-T", template)
}

// Show returns details for a specific revision.
func (r *Runner) Show(rev string) (string, error) {
	return r.Run("show", "-r", rev, "--color=always")
}

// Diff returns the diff for a revision.
func (r *Runner) Diff(rev string) (string, error) {
	return r.Run("diff", "-r", rev, "--color=always")
}

// DiffFile returns the diff for a specific file in a revision.
func (r *Runner) DiffFile(rev, file string) (string, error) {
	return r.Run("diff", "-r", rev, "--color=always", file)
}

// Status returns jj status output.
func (r *Runner) Status() (string, error) {
	return r.Run("status", "--color=always")
}

// OpLog returns the jj operation log output with colors.
func (r *Runner) OpLog() (string, error) {
	return r.Run("op", "log", "--color=always")
}

// evoLogTemplate formats evolog output to show operation details
// instead of change IDs, producing entries that look like op log
// output with operation IDs usable by OpShow.
const evoLogTemplate = `self.operation().id().short(12) ++ " " ++ self.operation().user() ++ " " ++ self.operation().time().start().ago() ++ ", lasted " ++ self.operation().time().duration() ++ "\n" ++ self.operation().description()`

// EvoLog returns the evolution log for a specific change.
func (r *Runner) EvoLog(rev string) (string, error) {
	return r.Run("evolog", "-r", rev, "--color=always", "-T", evoLogTemplate)
}

// OpShow returns details for a specific operation.
func (r *Runner) OpShow(opID string) (string, error) {
	return r.Run("op", "show", opID, "--color=always", "--patch")
}

// Describe updates the description (commit message) for a revision.
func (r *Runner) Describe(rev, message string) error {
	_, err := r.Run("describe", "-r", rev, "-m", message)
	return err
}

// Edit makes a revision the working copy, allowing it to be edited.
func (r *Runner) Edit(rev string) error {
	_, err := r.Run("edit", rev)
	return err
}

// New creates a new empty change on top of the current working copy.
func (r *Runner) New() error {
	_, err := r.Run("new")
	return err
}

// Abandon removes a revision from the repository.
func (r *Runner) Abandon(rev string) error {
	_, err := r.Run("abandon", rev)
	return err
}

// ShortestChangeID returns the shortest unique prefix for a change ID.
func (r *Runner) ShortestChangeID(rev string) (string, error) {
	output, err := r.Run("log", "-r", rev, "-T", "change_id.shortest()", "--no-graph")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

// LogStat returns log with file stats.
func (r *Runner) LogStat(rev string) (string, error) {
	return r.Run("log", "-r", rev, "--stat", "--color=always")
}

// ParseLogLines parses the raw log output into Change structs.
// For now, we keep the raw lines and just extract basic info.
func (r *Runner) ParseLogLines(output string) []Change {
	lines := strings.Split(output, "\n")

	var changes []Change

	var currentChange *Change

	var descLines []string

	// Regex to detect change lines - requires a graph symbol (@○◆◇●), not just whitespace
	// Matches lines like: "@ xsssnyux ..." or "○ nlkzwoyt/2 ..." or "◆ kyztkmnt ..."
	// Symbols: @ (working copy), ○ (normal), ◆ (immutable), ◇ (empty), ● (hidden), × (conflict)
	// Change IDs use reverse-hex [k-z] and may have version suffix /N
	changeLineRe := regexp.MustCompile(`^[│├└\s]*[@○◆◇●×]\s*([k-z]{8,}(?:/\d+)?)\s`)

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
				ChangeID: changeID,
				Raw:      line,
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

// ParseOpLogLines parses the raw op log or evolog output into Operation structs.
// Works with both operation IDs (12 hex chars) and change IDs (8+ letters).
func (r *Runner) ParseOpLogLines(output string) []Operation {
	lines := strings.Split(output, "\n")

	var operations []Operation

	var currentOp *Operation

	var descLines []string

	for _, line := range lines {
		stripped := stripANSI(line)
		if match := EntryLineRe.FindStringSubmatch(stripped); match != nil {
			// Save previous operation if exists
			if currentOp != nil {
				currentOp.Description = strings.TrimSpace(strings.Join(descLines, " "))
				operations = append(operations, *currentOp)
			}

			// Extract ID from named groups - one of opID or changeID will match
			// match[1] is opID (if hex), match[2] is changeID (if letters)
			opID := match[1]
			if opID == "" {
				opID = match[2]
			}

			currentOp = &Operation{
				OpID: opID,
				Raw:  line,
			}
			descLines = nil
		} else if currentOp != nil && strings.TrimSpace(line) != "" {
			// This is a continuation line (description, args, etc.)
			stripped := stripANSI(line)
			trimmed := strings.TrimSpace(strings.TrimPrefix(stripped, "│"))

			// Check for args line
			if after, found := strings.CutPrefix(trimmed, "args:"); found {
				currentOp.Args = strings.TrimSpace(after)
			} else if trimmed != "" {
				descLines = append(descLines, trimmed)
			}

			// Keep appending raw lines for display
			currentOp.Raw += "\n" + line
		}
	}

	// Don't forget the last operation
	if currentOp != nil {
		currentOp.Description = strings.TrimSpace(strings.Join(descLines, " "))
		operations = append(operations, *currentOp)
	}

	return operations
}

// ParseFiles parses diff output to extract file list.
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

	for lineIdx, line := range lines {
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
		statusCheck:
			for j := lineIdx + 1; j < len(lines) && j < lineIdx+5; j++ {
				nextLine := stripANSI(lines[j])

				switch {
				case newFileRe.MatchString(nextLine):
					file.Status = FileAdded
					break statusCheck
				case deletedFileRe.MatchString(nextLine):
					file.Status = FileDeleted
					break statusCheck
				case strings.HasPrefix(nextLine, "diff --git"):
					break statusCheck
				}
			}

			files = append(files, file)
		}
	}

	return files
}

// FindHunks finds all hunk/section positions in diff output.
// Supports both git-style @@ hunks and jj-style file headers.
func FindHunks(diffOutput string) []Hunk {
	var hunks []Hunk

	lines := strings.Split(diffOutput, "\n")

	// Git-style hunk header
	gitHunkRe := regexp.MustCompile(`^@@.*@@`)
	// jj-style file headers
	jjFileRe := regexp.MustCompile(`^(Added|Modified|Removed) regular file .+:$`)

	var currentHunk *Hunk

	for lineIdx, line := range lines {
		stripped := stripANSI(line)

		isSection := gitHunkRe.MatchString(stripped) || jjFileRe.MatchString(stripped)

		if isSection {
			// Close previous hunk
			if currentHunk != nil {
				currentHunk.EndLine = lineIdx - 1
				hunks = append(hunks, *currentHunk)
			}
			// Start new hunk/section
			currentHunk = &Hunk{
				Header:    stripped,
				StartLine: lineIdx,
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

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	ansiRe := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRe.ReplaceAllString(s, "")
}

// Error represents an error from running a jj command.
type Error struct {
	Command string
	Stderr  string
	Err     error
}

// Error returns a human-readable description of the failed jj command.
func (e *Error) Error() string {
	return "jj " + e.Command + ": " + e.Stderr
}
