// Package logger provides structured file-based logging for TUI applications.
// Logs are written to session-based files in the XDG state directory.
package logger

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ErrInvalidLogLevel is returned when an unrecognised log level is provided.
var ErrInvalidLogLevel = errors.New("invalid log level")

const (
	// dirPermissions is the mode for the log directory (owner rwx, group/other rx).
	dirPermissions = 0o755

	// filePermissions is the mode for individual log files (owner rw, group/other r).
	filePermissions = 0o644
)

// Logger wraps slog with file-based output for TUI applications.
type Logger struct {
	log     *slog.Logger
	logFile *os.File
}

// New creates a new Logger. If level is empty, returns a no-op logger.
// Valid levels: debug, info, warn, error (case-insensitive).
func New(level string) (*Logger, error) {
	if level == "" {
		// No-op logger - zero overhead
		return &Logger{
			log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		}, nil
	}

	// Parse level
	slogLevel, err := parseLogLevel(level)
	if err != nil {
		return nil, err
	}

	// Create log directory
	logDir, err := createLogDir()
	if err != nil {
		return nil, err
	}

	// Open session-based log file (clobber existing)
	logFile, err := openLogFile(logDir)
	if err != nil {
		return nil, err
	}

	// Create slog handler
	handler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slogLevel,
	})

	logger := &Logger{
		log:     slog.New(handler),
		logFile: logFile,
	}

	logger.Info("chado started", "pid", os.Getpid(), "level", level, "log_path", logFile.Name())

	return logger, nil
}

// Close closes the log file if open.
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}

// Debug logs a debug message with optional key-value pairs.
func (l *Logger) Debug(msg string, args ...any) {
	l.log.Debug(msg, args...)
}

// Info logs an info message with optional key-value pairs.
func (l *Logger) Info(msg string, args ...any) {
	l.log.Info(msg, args...)
}

// Warn logs a warning message with optional key-value pairs.
func (l *Logger) Warn(msg string, args ...any) {
	l.log.Warn(msg, args...)
}

// Error logs an error message with optional key-value pairs.
func (l *Logger) Error(msg string, args ...any) {
	l.log.Error(msg, args...)
}

func createLogDir() (string, error) {
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine home directory: %w", err)
		}

		stateDir = filepath.Join(home, ".local", "state")
	}

	logDir := filepath.Join(stateDir, "chado")
	if err := os.MkdirAll(logDir, dirPermissions); err != nil {
		return "", fmt.Errorf("could not create log directory: %w", err)
	}

	return logDir, nil
}

func openLogFile(logDir string) (*os.File, error) {
	logPath := filepath.Join(logDir, fmt.Sprintf("chado-%d.log", os.Getpid()))

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePermissions)
	if err != nil {
		return nil, fmt.Errorf("could not open log file: %w", err)
	}

	return logFile, nil
}

func parseLogLevel(level string) (slog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return -1, fmt.Errorf("%w: %s (use debug, info, warn, error)", ErrInvalidLogLevel, level)
	}
}
