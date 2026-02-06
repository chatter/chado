package jj

// Change represents a jj change/commit
type Change struct {
	ChangeID    string   // Short change ID (e.g., "xsssnyux")
	CommitID    string   // Git commit hash
	Author      string   // Author email
	Timestamp   string   // Formatted timestamp
	Description string   // Full commit message
	Bookmarks   []string // Bookmarks pointing to this change
	IsEmpty     bool     // Does this change have no diff?
	Raw         string   // Raw line from jj log (with ANSI colors)
}

// Operation represents a jj operation from op log
type Operation struct {
	OpID        string // Short operation ID (e.g., "bbc9fee12c4d")
	User        string // User and host
	Timestamp   string // When the operation occurred
	Duration    string // How long it took
	Description string // What the operation did
	Args        string // The jj command args
	Raw         string // Raw line from jj op log (with ANSI colors)
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
