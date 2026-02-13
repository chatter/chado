package jj

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"

	"github.com/chatter/chado/internal/logger"
)

// WatcherMsg is sent when the jj repo changes
type WatcherMsg struct{}

// Watcher watches the .jj directory for changes
type Watcher struct {
	watcher  *fsnotify.Watcher
	filtered chan fsnotify.Event
	done     chan struct{}
	log      *logger.Logger
}

// NewWatcher creates a new file watcher for the jj repo
func NewWatcher(repoPath string, log *logger.Logger) (*Watcher, error) {
	log.Debug("creating file watcher", "repo_path", repoPath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error("failed to create fsnotify watcher", "err", err)
		return nil, err
	}

	// Watch the .jj/repo/op_heads/heads directory for changes
	jjPath := filepath.Join(repoPath, ".jj", "repo", "op_heads", "heads")
	if err := watcher.Add(jjPath); err != nil {
		log.Error("failed to watch .jj directory", "path", jjPath, "err", err)
		watcher.Close()
		return nil, err
	}

	// Walk the repo directory and add all subdirectories to the watcher
	watchCount := 0
	_ = filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if strings.Contains(path, ".jj") || strings.Contains(path, ".git") {
				return filepath.SkipDir
			}

			if err := watcher.Add(path); err == nil {
				watchCount++
			}

			return nil
		}

		return nil
	})

	log.Info("watcher started", "watched_dirs", watchCount)

	self := &Watcher{
		watcher:  watcher,
		filtered: make(chan fsnotify.Event, 1),
		done:     make(chan struct{}),
		log:      log,
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

			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue // Ignore other operations
			}

			w.log.Debug("file change detected", "path", event.Name, "op", event.Op.String())
			// Non-blocking send: drop event when channel is full so the
			// watcher goroutine never blocks during event bursts.
			select {
			case w.filtered <- event:
			default:
				w.log.Debug("watcher event dropped (pending)", "path", event.Name)
			}
		case err := <-w.watcher.Errors:
			if err != nil {
				w.log.Warn("watcher error", "err", err)
			}
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
