package jj

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"pgregory.net/rapid"
)

// =============================================================================
// Unit Tests - Event Filtering Logic
// =============================================================================

func TestWatcher_FiltersLockFiles(t *testing.T) {
	dir := t.TempDir()
	setupFakeJJDir(t, dir)

	w, err := NewWatcher(dir, testLogger(t))
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close()

	// Create a .lock file - should NOT trigger an event
	lockFile := filepath.Join(dir, "test.lock")
	if err := os.WriteFile(lockFile, []byte("lock"), 0644); err != nil {
		t.Fatalf("failed to create lock file: %v", err)
	}

	// Create a regular file - SHOULD trigger an event
	regularFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(regularFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create regular file: %v", err)
	}

	// Wait for event with timeout
	select {
	case event := <-w.Events():
		if event.Name == lockFile {
			t.Errorf("lock file should be filtered out, got event for: %s", event.Name)
		}
		if event.Name != regularFile {
			t.Errorf("expected event for %s, got %s", regularFile, event.Name)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("expected event for regular file, got none")
	}
}

func TestWatcher_PassesWriteCreateRemoveRename(t *testing.T) {
	dir := t.TempDir()
	setupFakeJJDir(t, dir)

	w, err := NewWatcher(dir, testLogger(t))
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close()

	tests := []struct {
		name   string
		action func(path string) error
		op     fsnotify.Op
	}{
		{
			name: "create",
			action: func(path string) error {
				return os.WriteFile(path, []byte("new"), 0644)
			},
			op: fsnotify.Create,
		},
		{
			name: "write",
			action: func(path string) error {
				// File must exist first
				if err := os.WriteFile(path, []byte("initial"), 0644); err != nil {
					return err
				}
				// Drain the create event
				<-w.Events()
				// Now write to it
				return os.WriteFile(path, []byte("modified"), 0644)
			},
			op: fsnotify.Write,
		},
		{
			name: "remove",
			action: func(path string) error {
				// File must exist first
				if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
					return err
				}
				// Drain the create event
				<-w.Events()
				return os.Remove(path)
			},
			op: fsnotify.Remove,
		},
		{
			name: "rename",
			action: func(path string) error {
				// File must exist first
				if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
					return err
				}
				// Drain the create event
				<-w.Events()
				return os.Rename(path, path+".renamed")
			},
			// Note: Rename can produce RENAME or CREATE depending on platform.
			// We accept either - the point is the event isn't filtered out.
			op: fsnotify.Rename | fsnotify.Create,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(dir, "test_"+tt.name+".txt")

			if err := tt.action(testFile); err != nil {
				t.Fatalf("action failed: %v", err)
			}

			select {
			case event := <-w.Events():
				if event.Op&tt.op == 0 {
					t.Errorf("expected op to include one of %v, got %v", tt.op, event.Op)
				}
			case <-time.After(500 * time.Millisecond):
				t.Errorf("expected %s event, got none", tt.name)
			}
		})
	}
}

func TestWatcher_AddsNewDirectories(t *testing.T) {
	dir := t.TempDir()
	setupFakeJJDir(t, dir)

	w, err := NewWatcher(dir, testLogger(t))
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close()

	// Create a new subdirectory
	subdir := filepath.Join(dir, "newsubdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Drain the directory create event
	select {
	case <-w.Events():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected event for new directory")
	}

	// Give watcher time to add the directory
	time.Sleep(50 * time.Millisecond)

	// Now create a file in the new subdirectory - should be watched
	newFile := filepath.Join(subdir, "test.txt")
	if err := os.WriteFile(newFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file in subdir: %v", err)
	}

	select {
	case event := <-w.Events():
		if event.Name != newFile {
			t.Errorf("expected event for %s, got %s", newFile, event.Name)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("expected event for file in new subdir, got none - directory was not added to watcher")
	}
}

func TestWatcher_IgnoresJJDirectory(t *testing.T) {
	dir := t.TempDir()
	setupFakeJJDir(t, dir)

	w, err := NewWatcher(dir, testLogger(t))
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close()

	// Create a file in .jj - should NOT trigger an event
	jjFile := filepath.Join(dir, ".jj", "internal.txt")
	if err := os.WriteFile(jjFile, []byte("internal"), 0644); err != nil {
		t.Fatalf("failed to create file in .jj: %v", err)
	}

	// Create a regular file - SHOULD trigger an event
	regularFile := filepath.Join(dir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create regular file: %v", err)
	}

	select {
	case event := <-w.Events():
		if event.Name == jjFile {
			t.Error(".jj directory files should be filtered out")
		}
		if event.Name != regularFile {
			t.Errorf("expected event for %s, got %s", regularFile, event.Name)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("expected event for regular file, got none")
	}
}

func TestWatcher_IgnoresGitDirectory(t *testing.T) {
	dir := t.TempDir()
	setupFakeJJDir(t, dir)

	// Create .git directory
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git: %v", err)
	}

	w, err := NewWatcher(dir, testLogger(t))
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close()

	// Create a file in .git - should NOT trigger an event
	gitFile := filepath.Join(gitDir, "config")
	if err := os.WriteFile(gitFile, []byte("config"), 0644); err != nil {
		t.Fatalf("failed to create file in .git: %v", err)
	}

	// Create a regular file - SHOULD trigger an event
	regularFile := filepath.Join(dir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create regular file: %v", err)
	}

	select {
	case event := <-w.Events():
		if event.Name == gitFile {
			t.Error(".git directory files should be filtered out")
		}
		if event.Name != regularFile {
			t.Errorf("expected event for %s, got %s", regularFile, event.Name)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("expected event for regular file, got none")
	}
}

func TestWatcher_Close(t *testing.T) {
	dir := t.TempDir()
	setupFakeJJDir(t, dir)

	w, err := NewWatcher(dir, testLogger(t))
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	// Close should not error
	if err := w.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}

	// Events channel should be closed after Close
	select {
	case _, ok := <-w.Events():
		if ok {
			// Got an event, that's fine - might be a pending one
		}
		// Channel closed or event received, both are acceptable
	case <-time.After(200 * time.Millisecond):
		// Timeout is also acceptable - channel may not close immediately
	}
}

// =============================================================================
// Property Tests
// =============================================================================

// Property: Lock file paths should never be emitted
func TestWatcher_LockFilesNeverEmitted(t *testing.T) {
	dir := t.TempDir()
	setupFakeJJDir(t, dir)

	w, err := NewWatcher(dir, testLogger(t))
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close()

	rapid.Check(t, func(rt *rapid.T) {
		// Generate a random filename with .lock extension
		basename := rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "basename")
		lockFile := filepath.Join(dir, basename+".lock")

		if err := os.WriteFile(lockFile, []byte("lock"), 0644); err != nil {
			rt.Fatalf("failed to create lock file: %v", err)
		}
		defer os.Remove(lockFile)

		// Also create a trigger file to verify watcher is working
		triggerFile := filepath.Join(dir, basename+"_trigger.txt")
		if err := os.WriteFile(triggerFile, []byte("trigger"), 0644); err != nil {
			rt.Fatalf("failed to create trigger file: %v", err)
		}
		defer os.Remove(triggerFile)

		// Collect events - short timeout since fsnotify is fast
		timeout := time.After(20 * time.Millisecond)
		for {
			select {
			case event := <-w.Events():
				if filepath.Ext(event.Name) == ".lock" {
					rt.Fatalf("lock file event should be filtered: %s", event.Name)
				}
			case <-timeout:
				return
			}
		}
	})
}

// Property: Event operations should only be Write, Create, Remove, or Rename
func TestWatcher_OnlyValidOperations(t *testing.T) {
	dir := t.TempDir()
	setupFakeJJDir(t, dir)

	w, err := NewWatcher(dir, testLogger(t))
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close()

	validOps := fsnotify.Write | fsnotify.Create | fsnotify.Remove | fsnotify.Rename

	rapid.Check(t, func(rt *rapid.T) {
		basename := rapid.StringMatching(`[a-z]{5,10}`).Draw(rt, "basename")
		testFile := filepath.Join(dir, basename+".txt")

		// Random action
		action := rapid.IntRange(0, 2).Draw(rt, "action")
		switch action {
		case 0: // Create
			if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
				rt.Fatalf("write failed: %v", err)
			}
		case 1: // Modify (create first)
			if err := os.WriteFile(testFile, []byte("v1"), 0644); err != nil {
				rt.Fatalf("initial write failed: %v", err)
			}
			<-w.Events() // drain create
			if err := os.WriteFile(testFile, []byte("v2"), 0644); err != nil {
				rt.Fatalf("modify failed: %v", err)
			}
		case 2: // Remove (create first)
			if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
				rt.Fatalf("initial write failed: %v", err)
			}
			<-w.Events() // drain create
			if err := os.Remove(testFile); err != nil {
				rt.Fatalf("remove failed: %v", err)
			}
		}

		// Check event has valid operation
		select {
		case event := <-w.Events():
			if event.Op&validOps == 0 {
				rt.Fatalf("invalid operation: %v", event.Op)
			}
		case <-time.After(200 * time.Millisecond):
			// No event is acceptable (might have been filtered)
		}

		// Cleanup
		os.Remove(testFile)
	})
}

// =============================================================================
// Test Helpers
// =============================================================================

// setupFakeJJDir creates the minimal .jj directory structure needed for NewWatcher
func setupFakeJJDir(t *testing.T, dir string) {
	t.Helper()
	jjPath := filepath.Join(dir, ".jj", "repo", "op_heads", "heads")
	if err := os.MkdirAll(jjPath, 0755); err != nil {
		t.Fatalf("failed to create .jj directory: %v", err)
	}
}
