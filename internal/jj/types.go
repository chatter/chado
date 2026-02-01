package jj

// Change represents a jj change/commit
type Change struct {
	ChangeID      string   // Short change ID (e.g., "xsssnyux")
	CommitID      string   // Git commit hash
	Author        string   // Author email
	Timestamp     string   // Formatted timestamp
	Description   string   // Full commit message
	IsWorkingCopy bool     // Is this the @ commit?
	IsImmutable   bool     // Is this an immutable commit?
	Bookmarks     []string // Bookmarks pointing to this change
	IsEmpty       bool     // Does this change have no diff?
	Raw           string   // Raw line from jj log (with ANSI colors)
}

// File represents a file changed in a commit
type File struct {
	Path   string
	Status FileStatus
}

// FileStatus represents the type of change to a file
type FileStatus string

const (
	FileModified FileStatus = "M"
	FileAdded    FileStatus = "A"
	FileDeleted  FileStatus = "D"
	FileRenamed  FileStatus = "R"
	FileCopied   FileStatus = "C"
)

// Hunk represents a diff hunk
type Hunk struct {
	Header    string // The @@ line
	StartLine int    // Line number where hunk starts in the diff output
	EndLine   int    // Line number where hunk ends
}
