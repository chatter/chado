// Package ignore provides hierarchical .gitignore matching for file paths.
//
// It uses go-git's glob-based gitignore implementation which is significantly
// faster than regex-based alternatives. Patterns are cached at two levels:
// per-directory pattern cache and combined matcher cache.
package ignore

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// Matcher checks paths against hierarchical .gitignore rules.
type Matcher struct {
	rootPath         string
	fastDirs         map[string]bool // O(1) lookup for commonly ignored directory names
	dirPatterns      sync.Map        // dir (string) -> []gitignore.Pattern
	combinedMatchers sync.Map        // dir (string) -> gitignore.Matcher
}

// NewMatcher creates a Matcher rooted at the given repository path.
// It will read .gitignore files hierarchically from rootPath downward.
func NewMatcher(rootPath string) *Matcher {
	return &Matcher{
		rootPath: rootPath,
		fastDirs: map[string]bool{
			// Version control
			".git": true,
			".jj":  true,
			".svn": true,
			".hg":  true,
			".bzr": true,

			// IDE / editor
			".vscode": true,
			".idea":   true,

			// Dependencies & caches
			"node_modules":  true,
			"__pycache__":   true,
			".pytest_cache": true,
			".cache":        true,

			// OS generated
			".Trash":          true,
			".Spotlight-V100": true,
			".fseventsd":      true,
		},
	}
}

// Match reports whether the given absolute path should be ignored.
// isDir must be true when path refers to a directory so that directory-only
// patterns (e.g. "backup/") are applied correctly.
func (m *Matcher) Match(path string, isDir bool) bool {
	base := filepath.Base(path)

	// Fast path: O(1) lookup for commonly ignored directories.
	if isDir && m.fastDirs[base] {
		return true
	}

	// Don't apply rules to the root itself.
	if path == m.rootPath {
		return false
	}

	relPath, err := filepath.Rel(m.rootPath, path)
	if err != nil {
		relPath = path
	}

	components := pathToComponents(relPath)
	if len(components) == 0 {
		return false
	}

	parentDir := filepath.Dir(path)
	matcher := m.getCombinedMatcher(parentDir)

	return matcher.Match(components, isDir)
}

// pathToComponents splits a slash-separated path into its individual parts.
func pathToComponents(path string) []string {
	path = filepath.ToSlash(path)
	if path == "" || path == "." {
		return nil
	}

	return strings.Split(path, "/")
}

// parsePatterns converts gitignore lines into Pattern objects.
// domain is the path components where the patterns are defined (nil for root).
func parsePatterns(lines []string, domain []string) []gitignore.Pattern {
	var patterns []gitignore.Pattern

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		patterns = append(patterns, gitignore.ParsePattern(line, domain))
	}

	return patterns
}

// getDirPatterns returns parsed patterns for a directory's .gitignore file.
// Results are cached.
func (m *Matcher) getDirPatterns(dir string) []gitignore.Pattern {
	if v, ok := m.dirPatterns.Load(dir); ok {
		patterns, _ := v.([]gitignore.Pattern)
		return patterns
	}

	var domain []string

	relPath, _ := filepath.Rel(m.rootPath, dir)
	if relPath != "" && relPath != "." {
		domain = pathToComponents(relPath)
	}

	var patterns []gitignore.Pattern

	ignPath := filepath.Join(dir, ".gitignore")
	if content, err := os.ReadFile(ignPath); err == nil {
		lines := strings.Split(string(content), "\n")
		patterns = append(patterns, parsePatterns(lines, domain)...)
	}

	m.dirPatterns.Store(dir, patterns)

	return patterns
}

// getCombinedMatcher returns a matcher that combines all .gitignore patterns
// from the root to the given directory. Results are cached per directory.
func (m *Matcher) getCombinedMatcher(dir string) gitignore.Matcher {
	if v, ok := m.combinedMatchers.Load(dir); ok {
		matcher, _ := v.(gitignore.Matcher)
		return matcher
	}

	var allPatterns []gitignore.Pattern

	// Collect patterns from root to this directory.
	relDir, _ := filepath.Rel(m.rootPath, dir)

	var pathParts []string
	if relDir != "" && relDir != "." {
		pathParts = pathToComponents(relDir)
	}

	currentPath := m.rootPath
	allPatterns = append(allPatterns, m.getDirPatterns(currentPath)...)

	for _, part := range pathParts {
		currentPath = filepath.Join(currentPath, part)
		allPatterns = append(allPatterns, m.getDirPatterns(currentPath)...)
	}

	matcher := gitignore.NewMatcher(allPatterns)
	m.combinedMatchers.Store(dir, matcher)

	return matcher
}
