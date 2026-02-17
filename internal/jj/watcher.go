package jj

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"

	"github.com/chatter/chado/internal/ignore"
	"github.com/chatter/chado/internal/logger"
)

// WatcherMsg is sent when the jj repo changes.
type WatcherMsg struct{}

// Watcher watches the .jj directory for changes.
type Watcher struct {
	watcher  *fsnotify.Watcher
	filtered chan fsnotify.Event
	done     chan struct{}
	log      *logger.Logger
	ignore   *ignore.Matcher
}

// NewWatcher creates a new file watcher for the jj repo.
func NewWatcher(repoPath string, log *logger.Logger) (*Watcher, error) {
	log.Debug("creating file watcher", "repo_path", repoPath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error("failed to create fsnotify watcher", "err", err)

		return nil, fmt.Errorf("creating fsnotify watcher: %w", err)
	}

	// Watch the .jj/repo/op_heads/heads directory for changes.
	jjPath := filepath.Join(repoPath, ".jj", "repo", "op_heads", "heads")
	if err := watcher.Add(jjPath); err != nil {
		log.Error("failed to watch .jj directory", "path", jjPath, "err", err)
		watcher.Close()

		return nil, fmt.Errorf("watching .jj directory: %w", err)
	}

	ignoreMatcher := ignore.NewMatcher(repoPath)

	// Walk the repo directory and add all non-ignored subdirectories.
	watchCount := 0
	_ = filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if ignoreMatcher.Match(path, true) {
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
		ignore:   ignoreMatcher,
	}

	go self.filterEvents()

	return self, nil
}

// Events returns the channel of filtered fsnotify events.
func (w *Watcher) Events() <-chan fsnotify.Event {
	return w.filtered
}

// Errors returns the channel of fsnotify errors.
func (w *Watcher) Errors() <-chan error {
	return w.watcher.Errors
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	close(w.done)

	if err := w.watcher.Close(); err != nil {
		return fmt.Errorf("closing fsnotify watcher: %w", err)
	}

	return nil
}

func (w *Watcher) filterEvents() {
	defer close(w.filtered)

	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			w.trackNewDirectory(event)

			if !w.shouldForward(event) {
				continue
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

// trackNewDirectory adds newly created directories to the watcher so that
// file changes in them are picked up. Ignored directories are skipped.
func (w *Watcher) trackNewDirectory(event fsnotify.Event) {
	if !event.Has(fsnotify.Create) {
		return
	}

	info, err := os.Stat(event.Name)
	if err != nil || !info.IsDir() {
		return
	}

	if w.ignore.Match(event.Name, true) {
		return
	}

	if err := w.watcher.Add(event.Name); err != nil {
		w.log.Debug("failed to watch new directory", "path", event.Name, "err", err)
	}
}

// shouldForward reports whether an event should be sent to consumers.
func (w *Watcher) shouldForward(event fsnotify.Event) bool {
	if strings.HasSuffix(event.Name, ".lock") {
		return false
	}

	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
		return false
	}

	if w.ignore.Match(event.Name, false) {
		return false
	}

	return true
}
