# Log capture for testing

## Summary

Add a test helper that captures log output during test execution, similar to Elixir's ExUnit `capture_log` pattern. This enables asserting that code logs the expected messages at the expected levels.

## Motivation

With the struct-based `Logger` and dependency injection now in place, we can inject a test-specific logger that captures output. This allows tests to verify:

- Correct log messages are emitted
- Log levels are appropriate (e.g., errors logged at Error, not Debug)
- Structured key-value pairs contain expected data

## Proposed API

```go
// Wrap code and capture logs
log, capture := logger.NewCapture()
runner := NewRunner(".", log)

runner.Run("log", "--color=always")

// Assert on captured output
msgs := capture.Messages()
assert.Contains(t, msgs[0], "executing jj command")
assert.Equal(t, slog.LevelDebug, capture.Entries()[0].Level)

// Or capture_log style helper
logs := logger.Capture(func() {
    runner.Run("log")
})
assert.Contains(t, logs, "executing jj command")
```

## Implementation Options

### Option A: Buffer-based Logger

Create a `CaptureLogger` that writes to a `bytes.Buffer` instead of a file:

```go
type CaptureLogger struct {
    *Logger
    buf *bytes.Buffer
}

func NewCapture() (*Logger, *CaptureLogger) {
    buf := &bytes.Buffer{}
    handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
    log := &Logger{log: slog.New(handler)}
    return log, &CaptureLogger{Logger: log, buf: buf}
}

func (c *CaptureLogger) String() string {
    return c.buf.String()
}
```

### Option A: Structured capture with custom handler

Implement a custom `slog.Handler` that stores `slog.Record` entries for richer assertions:

```go
type CaptureHandler struct {
    entries []slog.Record
    mu      sync.Mutex
}

func (h *CaptureHandler) Handle(ctx context.Context, r slog.Record) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.entries = append(h.entries, r)
    return nil
}

func (h *CaptureHandler) Entries() []slog.Record {
    return h.entries
}
```

This allows assertions on level, message, and attributes separately.

## Trade-offs

| Approach | Pros | Cons |
|----------|------|------|
| Buffer-based | Simple, matches current impl | String matching only |
| Custom handler | Structured assertions, level filtering | More code |

## Priority

Low - useful for ensuring logging behavior, but not blocking.

## Labels

`enhancement`, `testing`, `dx`
