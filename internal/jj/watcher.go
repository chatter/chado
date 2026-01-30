package jj

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// WatcherMsg is sent when the jj repo changes
type WatcherMsg struct{}

// Watcher watches the .jj directory for changes
type Watcher struct {
	watcher  *fsnotify.Watcher
	filtered chan fsnotify.Event
	done     chan struct{}
}

// NewWatcher creates a new file watcher for the jj repo
func NewWatcher(repoPath string) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Watch the .jj/repo/op_heads/heads directory for changes
	jjPath := filepath.Join(repoPath, ".jj", "repo", "op_heads", "heads")
	if err := watcher.Add(jjPath); err != nil {
		watcher.Close()
		return nil, err
	}

	// Walk the repo directory and add all subdirectories to the watcher
	_ = filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if strings.Contains(path, ".jj") || strings.Contains(path, ".git") {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}

		return nil
	})

	self := &Watcher{
		watcher:  watcher,
		filtered: make(chan fsnotify.Event),
		done:     make(chan struct{}),
	}

	go self.filterEvents()

	return self, nil
}

func (w *Watcher) filterEvents() {
	defer close(w.filtered)

	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return // Channel closed
			}

			// Add new directories to the watcher
			if event.Has(fsnotify.Create) {
				if strings.Contains(event.Name, ".jj") || strings.Contains(event.Name, ".git") {
					continue // Ignore .jj and .git directories
				}

				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					w.watcher.Add(event.Name)
				}
			}

			if strings.HasSuffix(event.Name, ".lock") {
				continue // Ignore lock files
			}

			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) == 0 {
				continue // Ignore other operations
			}

			w.filtered <- event
		case <-w.watcher.Errors:
			// handle errors if needed
		}
	}
}

// Events returns the channel of filtered fsnotify events
func (w *Watcher) Events() <-chan fsnotify.Event {
	return w.filtered
}

// Errors returns the channel of fsnotify errors
func (w *Watcher) Errors() <-chan error {
	return w.watcher.Errors
}

// Close stops the watcher
func (w *Watcher) Close() error {
	close(w.done)
	return w.watcher.Close()
}
