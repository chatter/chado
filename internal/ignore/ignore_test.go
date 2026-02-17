package ignore_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chatter/chado/internal/ignore"
)

func TestFastIgnoreDirs(t *testing.T) {
	root := t.TempDir()

	m := ignore.NewMatcher(root)

	fastDirs := []string{".git", ".jj", "node_modules", "__pycache__", ".vscode", ".idea"}
	for _, name := range fastDirs {
		dir := filepath.Join(root, name)
		if !m.Match(dir, true) {
			t.Errorf("expected %s to be ignored (fast path)", name)
		}
	}
}

func TestFastIgnoreDirs_NotAppliedToFiles(t *testing.T) {
	root := t.TempDir()

	m := ignore.NewMatcher(root)

	// A file named "node_modules" should not be fast-path ignored.
	file := filepath.Join(root, "node_modules")
	if m.Match(file, false) {
		t.Error("fast-path should only apply to directories, not files")
	}
}

func TestGitignorePatterns(t *testing.T) {
	root := t.TempDir()

	gitignore := "*.log\nbuild/\n"
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		t.Fatal(err)
	}

	m := ignore.NewMatcher(root)

	tests := []struct {
		path  string
		isDir bool
		want  bool
		desc  string
	}{
		{filepath.Join(root, "app.log"), false, true, "*.log should match files"},
		{filepath.Join(root, "main.go"), false, false, "main.go should not be ignored"},
		{filepath.Join(root, "build"), true, true, "build/ pattern should match directories"},
		{filepath.Join(root, "build"), false, false, "build/ pattern should NOT match files"},
		{filepath.Join(root, "src"), true, false, "src/ should not be ignored"},
	}

	for _, tt := range tests {
		got := m.Match(tt.path, tt.isDir)
		if got != tt.want {
			t.Errorf("%s: Match(%q, isDir=%v) = %v, want %v", tt.desc, tt.path, tt.isDir, got, tt.want)
		}
	}
}

func TestHierarchicalGitignore(t *testing.T) {
	root := t.TempDir()

	// Root .gitignore ignores *.tmp
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*.tmp\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create subdir with its own .gitignore that ignores *.dat
	subdir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(subdir, ".gitignore"), []byte("*.dat\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := ignore.NewMatcher(root)

	tests := []struct {
		path  string
		isDir bool
		want  bool
		desc  string
	}{
		// Root pattern applies everywhere
		{filepath.Join(root, "foo.tmp"), false, true, "root *.tmp matches in root"},
		{filepath.Join(subdir, "bar.tmp"), false, true, "root *.tmp matches in subdir"},

		// Subdir pattern applies only in subdir
		{filepath.Join(subdir, "data.dat"), false, true, "sub *.dat matches in subdir"},
		{filepath.Join(root, "data.dat"), false, false, "sub *.dat should NOT match in root"},

		// Unignored files
		{filepath.Join(subdir, "code.go"), false, false, ".go not ignored anywhere"},
	}

	for _, tt := range tests {
		got := m.Match(tt.path, tt.isDir)
		if got != tt.want {
			t.Errorf("%s: Match(%q, isDir=%v) = %v, want %v", tt.desc, tt.path, tt.isDir, got, tt.want)
		}
	}
}

func TestRootPathNeverIgnored(t *testing.T) {
	root := t.TempDir()

	// Even with a wildcard gitignore, the root itself should not be ignored.
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := ignore.NewMatcher(root)

	if m.Match(root, true) {
		t.Error("root path should never be ignored")
	}
}

func TestNegationPattern(t *testing.T) {
	root := t.TempDir()

	// Ignore all .log files except important.log
	gitignore := "*.log\n!important.log\n"
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		t.Fatal(err)
	}

	m := ignore.NewMatcher(root)

	if !m.Match(filepath.Join(root, "debug.log"), false) {
		t.Error("debug.log should be ignored")
	}

	if m.Match(filepath.Join(root, "important.log"), false) {
		t.Error("important.log should NOT be ignored (negation pattern)")
	}
}

func TestNoGitignore(t *testing.T) {
	root := t.TempDir()

	m := ignore.NewMatcher(root)

	// Without a .gitignore, regular files/dirs should not be ignored.
	if m.Match(filepath.Join(root, "file.txt"), false) {
		t.Error("file.txt should not be ignored when no .gitignore exists")
	}

	if m.Match(filepath.Join(root, "src"), true) {
		t.Error("src/ should not be ignored when no .gitignore exists")
	}

	// But fast-path dirs should still be ignored.
	if !m.Match(filepath.Join(root, "node_modules"), true) {
		t.Error("node_modules should always be ignored via fast path")
	}
}
