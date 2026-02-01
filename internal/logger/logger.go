// Package logger provides structured file-based logging for TUI applications.
// Logs are written to session-based files in the XDG state directory.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

var log *slog.Logger
var logFile *os.File

// Init initializes the logger. If level is empty, uses a no-op logger.
// Valid levels: debug, info, warn, error (case-insensitive).
func Init(level string) error {
	if level == "" {
		// No-op logger - zero overhead
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
		return nil
	}

	// Parse level
	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		return fmt.Errorf("invalid log level: %s (use debug, info, warn, error)", level)
	}

	// Create log directory
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not determine home directory: %w", err)
		}
		stateDir = filepath.Join(home, ".local", "state")
	}
	logDir := filepath.Join(stateDir, "chado")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("could not create log directory: %w", err)
	}

	// Open session-based log file (clobber existing)
	logPath := filepath.Join(logDir, fmt.Sprintf("chado-%d.log", os.Getpid()))
	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("could not open log file: %w", err)
	}

	// Create handler with level filtering
	handler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slogLevel,
	})
	log = slog.New(handler)

	log.Info("chado started", "pid", os.Getpid(), "level", level, "log_path", logPath)
	return nil
}

// Close closes the log file if open.
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

// Debug logs a debug message with optional key-value pairs.
func Debug(msg string, args ...any) {
	log.Debug(msg, args...)
}

// Info logs an info message with optional key-value pairs.
func Info(msg string, args ...any) {
	log.Info(msg, args...)
}

// Warn logs a warning message with optional key-value pairs.
func Warn(msg string, args ...any) {
	log.Warn(msg, args...)
}

// Error logs an error message with optional key-value pairs.
func Error(msg string, args ...any) {
	log.Error(msg, args...)
}
