package jj

import (
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// WatcherMsg is sent when the jj repo changes
type WatcherMsg struct{}

// Watcher watches the .jj directory for changes
type Watcher struct {
	watcher *fsnotify.Watcher
	done    chan struct{}
}

// NewWatcher creates a new file watcher for the jj repo
func NewWatcher(repoPath string) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Watch the .jj/repo directory for operation changes
	jjPath := filepath.Join(repoPath, ".jj", "repo")
	if err := watcher.Add(jjPath); err != nil {
		watcher.Close()
		return nil, err
	}

	// Also watch op_heads specifically
	opHeadsPath := filepath.Join(repoPath, ".jj", "repo", "op_heads")
	_ = watcher.Add(opHeadsPath) // Ignore error if doesn't exist

	return &Watcher{
		watcher: watcher,
		done:    make(chan struct{}),
	}, nil
}

// Events returns the channel of fsnotify events
func (w *Watcher) Events() <-chan fsnotify.Event {
	return w.watcher.Events
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
