# File-based logging with configurable log levels

## Summary

Implement structured file-based logging using Go's standard library `log/slog`, with `--log-level` CLI flag, session-based log files per PID, defaulting to no logging.

## Motivation

Debugging a TUI is painful - stdout is taken by the UI, so `fmt.Println` isn't an option. Currently requires adding/removing file creation code to trace issues. A proper logging solution allows dialing in verbosity without code changes.

## Design

### CLI Flag

```bash
chado --log-level debug   # Verbose tracing
chado -l info             # Key events only
chado -l warn             # Warnings and errors
chado -l error            # Errors only
chado                     # No logging (default)
```

### Log File Location

- Path: `$XDG_STATE_HOME/chado/chado-{pid}.log`
- Default: `~/.local/state/chado/chado-12345.log`
- Session-based (PID in filename) to support multiple instances
- File truncated on startup (clobbers existing file with same PID)
- No file created when logging is disabled

### Log Levels

| Level | What's logged | Use case |
|-------|---------------|----------|
| debug | All messages | Development, debugging issues |
| info  | info + warn + error | Normal troubleshooting |
| warn  | warn + error | Errors and warnings only |
| error | error only | Minimal, only failures |

## Implementation

### New package: `internal/logger/logger.go`

```go
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
func Init(level string) error {
    if level == "" {
        log = slog.New(slog.NewTextHandler(io.Discard, nil))
        return nil
    }

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
        return fmt.Errorf("invalid log level: %s", level)
    }

    // XDG-compliant log directory
    stateDir := os.Getenv("XDG_STATE_HOME")
    if stateDir == "" {
        home, _ := os.UserHomeDir()
        stateDir = filepath.Join(home, ".local", "state")
    }
    logDir := filepath.Join(stateDir, "chado")
    os.MkdirAll(logDir, 0755)

    // Session-based log file (truncated on start)
    logPath := filepath.Join(logDir, fmt.Sprintf("chado-%d.log", os.Getpid()))
    logFile, _ = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

    handler := slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slogLevel})
    log = slog.New(handler)
    return nil
}

func Close() { if logFile != nil { logFile.Close() } }

func Debug(msg string, args ...any) { log.Debug(msg, args...) }
func Info(msg string, args ...any)  { log.Info(msg, args...) }
func Warn(msg string, args ...any)  { log.Warn(msg, args...) }
func Error(msg string, args ...any) { log.Error(msg, args...) }
```

### Changes to `main.go`

Add flag parsing before `run()`:

```go
func main() {
    logLevel := flag.String("log-level", "", "log level: debug, info, warn, error")
    flag.StringVar(logLevel, "l", "", "log level (shorthand)")
    flag.Parse()

    if err := logger.Init(*logLevel); err != nil {
        fmt.Fprintf(os.Stderr, "warning: %v\n", err)
    }
    defer logger.Close()

    // ... rest of main
}
```

## Tasks

- [ ] Create `internal/logger/logger.go` with Init(), Close(), Debug/Info/Warn/Error wrappers
- [ ] Add `--log-level`/`-l` flag to `main.go`
- [ ] Add logging calls to key areas: app init, watcher events, jj commands, errors

## Future Enhancements (not in scope)

- Log file cleanup (delete old session logs)
- Config file option

## Labels

`enhancement`, `dx`
