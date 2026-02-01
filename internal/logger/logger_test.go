package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func TestInit_ValidLevels(t *testing.T) {
	validLevels := []string{"debug", "info", "warn", "error", "DEBUG", "INFO", "WARN", "ERROR", "Debug", "Info", "Warn", "Error"}

	for _, level := range validLevels {
		t.Run(level, func(t *testing.T) {
			// Use temp directory for test
			tempDir := t.TempDir()
			t.Setenv("XDG_STATE_HOME", tempDir)

			err := Init(level)
			if err != nil {
				t.Errorf("Init(%q) returned error: %v", level, err)
			}
			Close()

			// Verify log file was created
			logDir := filepath.Join(tempDir, "chado")
			entries, err := os.ReadDir(logDir)
			if err != nil {
				t.Errorf("failed to read log directory: %v", err)
			}
			if len(entries) != 1 {
				t.Errorf("expected 1 log file, got %d", len(entries))
			}
		})
	}
}

func TestInit_InvalidLevels(t *testing.T) {
	invalidLevels := []string{"trace", "verbose", "warning", "fatal", "critical", "all", "none", "off", "123", ""}

	// Filter out empty string as that's a valid "disabled" case
	for _, level := range invalidLevels {
		if level == "" {
			continue
		}
		t.Run(level, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Setenv("XDG_STATE_HOME", tempDir)

			err := Init(level)
			if err == nil {
				Close()
				t.Errorf("Init(%q) should return error for invalid level", level)
				return
			}

			if !strings.Contains(err.Error(), "invalid log level") {
				t.Errorf("error should mention 'invalid log level', got: %v", err)
			}
		})
	}
}

func TestInit_InvalidLevels_Property(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate random strings that are NOT valid levels
		level := rapid.StringMatching(`[a-z]{1,10}`).Draw(rt, "level")

		// Skip if it happens to be a valid level
		lower := strings.ToLower(level)
		if lower == "debug" || lower == "info" || lower == "warn" || lower == "error" {
			rt.Skip("valid level generated")
		}

		// For property test, we just verify the error is returned
		// We can't easily use temp dirs in rapid.T, so test error path only
		err := Init(level)
		if err == nil {
			Close()
			rt.Errorf("Init(%q) should return error for invalid level", level)
		}
	})
}

func TestInit_EmptyLevel_NoOpLogger(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tempDir)

	err := Init("")
	if err != nil {
		t.Errorf("Init(\"\") returned error: %v", err)
	}
	defer Close()

	// Logging should not panic
	Debug("test debug")
	Info("test info")
	Warn("test warn")
	Error("test error")

	// No log file should be created
	logDir := filepath.Join(tempDir, "chado")
	_, err = os.Stat(logDir)
	if !os.IsNotExist(err) {
		t.Errorf("log directory should not exist for empty level")
	}
}

func TestInit_CreatesLogDirectory(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tempDir)

	err := Init("info")
	if err != nil {
		t.Errorf("Init returned error: %v", err)
	}
	defer Close()

	logDir := filepath.Join(tempDir, "chado")
	info, err := os.Stat(logDir)
	if err != nil {
		t.Errorf("log directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("log path should be a directory")
	}
}

func TestInit_LogFileContainsPID(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tempDir)

	err := Init("debug")
	if err != nil {
		t.Errorf("Init returned error: %v", err)
	}
	defer Close()

	logDir := filepath.Join(tempDir, "chado")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("failed to read log directory: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 log file, got %d", len(entries))
	}

	filename := entries[0].Name()
	expectedPrefix := "chado-"
	expectedSuffix := ".log"

	if !strings.HasPrefix(filename, expectedPrefix) {
		t.Errorf("log filename should start with %q, got %q", expectedPrefix, filename)
	}
	if !strings.HasSuffix(filename, expectedSuffix) {
		t.Errorf("log filename should end with %q, got %q", expectedSuffix, filename)
	}

	// Extract PID from filename
	pidStr := strings.TrimSuffix(strings.TrimPrefix(filename, expectedPrefix), expectedSuffix)
	if pidStr == "" {
		t.Errorf("log filename should contain PID")
	}
}

func TestInit_ClobbersExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tempDir)

	// First init
	err := Init("debug")
	if err != nil {
		t.Fatalf("first Init returned error: %v", err)
	}
	Info("first session")
	Close()

	// Read file content
	logDir := filepath.Join(tempDir, "chado")
	entries, _ := os.ReadDir(logDir)
	logPath := filepath.Join(logDir, entries[0].Name())
	firstContent, _ := os.ReadFile(logPath)

	// Second init with same PID (simulated by not changing process)
	err = Init("debug")
	if err != nil {
		t.Fatalf("second Init returned error: %v", err)
	}
	Info("second session")
	Close()

	// Read file content again
	secondContent, _ := os.ReadFile(logPath)

	// File should be clobbered (not contain first session message)
	if strings.Contains(string(secondContent), "first session") {
		t.Errorf("log file should be clobbered, still contains first session content")
	}
	if !strings.Contains(string(secondContent), "second session") {
		t.Errorf("log file should contain second session content")
	}

	// Second file should be smaller or equal (clobbered, not appended)
	if len(secondContent) > len(firstContent)*2 {
		t.Errorf("log file appears to be appended rather than clobbered")
	}
}

func TestLevelFiltering_DebugLogsAll(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tempDir)

	err := Init("debug")
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")
	Close()

	content := readLogFile(t, tempDir)
	if !strings.Contains(content, "debug msg") {
		t.Errorf("debug level should log debug messages")
	}
	if !strings.Contains(content, "info msg") {
		t.Errorf("debug level should log info messages")
	}
	if !strings.Contains(content, "warn msg") {
		t.Errorf("debug level should log warn messages")
	}
	if !strings.Contains(content, "error msg") {
		t.Errorf("debug level should log error messages")
	}
}

func TestLevelFiltering_InfoFiltersDebug(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tempDir)

	err := Init("info")
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")
	Close()

	content := readLogFile(t, tempDir)
	if strings.Contains(content, "debug msg") {
		t.Errorf("info level should NOT log debug messages")
	}
	if !strings.Contains(content, "info msg") {
		t.Errorf("info level should log info messages")
	}
	if !strings.Contains(content, "warn msg") {
		t.Errorf("info level should log warn messages")
	}
	if !strings.Contains(content, "error msg") {
		t.Errorf("info level should log error messages")
	}
}

func TestLevelFiltering_WarnFiltersInfoAndDebug(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tempDir)

	err := Init("warn")
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")
	Close()

	content := readLogFile(t, tempDir)
	if strings.Contains(content, "debug msg") {
		t.Errorf("warn level should NOT log debug messages")
	}
	if strings.Contains(content, "info msg") {
		t.Errorf("warn level should NOT log info messages")
	}
	if !strings.Contains(content, "warn msg") {
		t.Errorf("warn level should log warn messages")
	}
	if !strings.Contains(content, "error msg") {
		t.Errorf("warn level should log error messages")
	}
}

func TestLevelFiltering_ErrorFiltersAll(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tempDir)

	err := Init("error")
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")
	Close()

	content := readLogFile(t, tempDir)
	if strings.Contains(content, "debug msg") {
		t.Errorf("error level should NOT log debug messages")
	}
	if strings.Contains(content, "info msg") {
		t.Errorf("error level should NOT log info messages")
	}
	if strings.Contains(content, "warn msg") {
		t.Errorf("error level should NOT log warn messages")
	}
	if !strings.Contains(content, "error msg") {
		t.Errorf("error level should log error messages")
	}
}

func TestLogging_StructuredArgs(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tempDir)

	err := Init("debug")
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	Info("test message", "key1", "value1", "key2", 42)
	Close()

	content := readLogFile(t, tempDir)
	if !strings.Contains(content, "key1=value1") {
		t.Errorf("log should contain structured key1=value1")
	}
	if !strings.Contains(content, "key2=42") {
		t.Errorf("log should contain structured key2=42")
	}
}

// readLogFile reads the first log file in the chado log directory
func readLogFile(t *testing.T, stateDir string) string {
	t.Helper()
	logDir := filepath.Join(stateDir, "chado")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("failed to read log directory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no log files found")
	}
	content, err := os.ReadFile(filepath.Join(logDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	return string(content)
}
