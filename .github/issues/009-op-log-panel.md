# Operation Log Panel

## Summary

Add a read-only panel displaying `jj op log` output below the existing log panel. The left side will split vertically 50/50 between the change log and operation log. When the op log panel is focused, the diff pane shows `jj op show` output for the selected operation.

## Motivation

The operation log provides visibility into jj's undo history - what commands were run, when, and their effects. This is useful for understanding repository state and preparing for potential `jj undo` actions.

## Architecture

```
┌─────────────────────────────┬───────────────────────────────────────────┐
│  Left Pane (40%)            │  Right Pane (60%)                         │
│ ┌─────────────────────────┐ │ ┌───────────────────────────────────────┐ │
│ │                         │ │ │                                       │ │
│ │  LogPanel (50% height)  │ │ │                                       │ │
│ │                         │ │ │                                       │ │
│ ├─────────────────────────┤ │ │           DiffPanel                   │ │
│ │                         │ │ │         (100% height)                 │ │
│ │ OpLogPanel (50% height) │ │ │                                       │ │
│ │                         │ │ │                                       │ │
│ └─────────────────────────┘ │ └───────────────────────────────────────┘ │
├─────────────────────────────┴───────────────────────────────────────────┤
│  StatusBar                                                              │
└─────────────────────────────────────────────────────────────────────────┘
```

## Implementation

### 0. Refactor: Rename `IsWorkingCopy` to `IsCurrent`

For consistency between `Change` and `Operation` types, rename `Change.IsWorkingCopy` to `Change.IsCurrent` in:
- `internal/jj/types.go` - struct field
- `internal/jj/runner.go` - where it's set in `ParseLogLines()`

### 1. New Type: `Operation` in `internal/jj/types.go`

```go
// Operation represents a jj operation from op log
type Operation struct {
    OpID        string // Short operation ID (e.g., "bbc9fee12c4d")
    User        string // User and host
    Timestamp   string // When the operation occurred
    Duration    string // How long it took
    Description string // What the operation did
    Args        string // The jj command args
    IsCurrent   bool   // Is this the @ operation?
    Raw         string // Raw line from jj op log (with ANSI colors)
}
```

### 2. New Runner Methods in `internal/jj/runner.go`

```go
// OpLog returns the jj operation log output with colors
func (r *Runner) OpLog() (string, error) {
    return r.Run("op", "log", "--color=always")
}

// OpShow returns details for a specific operation
func (r *Runner) OpShow(opID string) (string, error) {
    return r.Run("op", "show", opID, "--color=always")
}

// ParseOpLogLines parses op log output into Operation structs
func (r *Runner) ParseOpLogLines(output string) []Operation {
    // Similar pattern to ParseLogLines
    // Match lines like: "@ bbc9fee12c4d user@host 4 minutes ago, lasted 1 second"
}
```

### 3. New Panel: `internal/ui/oplog.go`

Follow the same structure as `LogPanel`:

- Viewport-based scrolling
- Cursor selection with `→` indicator
- j/k navigation, g/G for top/bottom
- Focus state styling
- `HelpBindings()` for status bar

```go
type OpLogPanel struct {
    viewport     viewport.Model
    operations   []jj.Operation
    cursor       int
    focused      bool
    width        int
    height       int
    rawLog       string
    opStartLines []int
    totalLines   int
}
```

### 4. App Integration in `internal/app/app.go`

Add to Model:

```go
opLogPanel ui.OpLogPanel
operations []jj.Operation
```

Add new focus pane:

```go
const (
    PaneDiff   FocusedPane = iota // [0] Right pane
    PaneLog                       // [1] Left pane - log
    PaneOpLog                     // [2] Left pane - op log
)
```

Add `FocusPane2` keybinding and `actionFocusPane2()` method.

Add loading command:

```go
func (m Model) loadOpLog() tea.Cmd {
    return func() tea.Msg {
        output, err := m.runner.OpLog()
        // ...
        return opLogLoadedMsg{raw: output, operations: ops}
    }
}
```

Update `updatePanelSizes()` to split left pane height:

```go
// Left pane splits vertically: log 50%, op log 50%
leftHeight := contentHeight / 2
m.logPanel.SetSize(leftWidth, leftHeight)
m.opLogPanel.SetSize(leftWidth, contentHeight - leftHeight)
```

Update `View()` to join log and oplog panels vertically on the left:

```go
leftPanel := lipgloss.JoinVertical(lipgloss.Left, 
    m.logPanel.View(), 
    m.opLogPanel.View(),
)
```

### 5. Focus Handling

The op log panel is focusable as pane `[2]`:

- Focusable via `2` key or `l`/`h` pane cycling
- Full j/k navigation and g/G top/bottom
- Cursor selection with `→` indicator
- Auto-refresh on watcher events alongside the main log
- No actions on Enter (read-only for now)

### 6. Diff Pane Shows Op Details

When op log panel is focused, the diff pane shows `jj op show` output for the selected operation:

```go
func (m Model) loadOpShow(opID string) tea.Cmd {
    return func() tea.Msg {
        output, err := m.runner.OpShow(opID)
        // ...
        return opShowLoadedMsg{output: output}
    }
}
```

The diff pane redraws when:
- Focus changes from log → op log (show op details)
- Focus changes from op log → log (show change diff)
- Selection changes within op log panel

Output example:
```
ab1a007d34d1 curtis.hatter@mac-mini.local 4 days ago, lasted 790 milliseconds
describe commit 95493238e678572ae728443574120d7d42c88ffb
args: jj describe -m 'fix #6: file view watcher, rename events'

Changed commits:
○  + mkvurkku cc2e6ce9 fix #6: file view watcher, rename events
   - mkvurkku/1 95493238 (hidden) (no description set)
```

## Tasks

- [ ] Rename `Change.IsWorkingCopy` to `Change.IsCurrent` across codebase
- [ ] Add `Operation` struct to `internal/jj/types.go`
- [ ] Add `OpLog()`, `OpShow()`, `ParseOpLogLines()` to `internal/jj/runner.go`
- [ ] Create `internal/ui/oplog.go` with `OpLogPanel`
- [ ] Add `FocusPane2` keybinding to `internal/app/keys.go`
- [ ] Integrate panel into `internal/app/app.go` (model, loading, layout, focus)
- [ ] Show `jj op show` in diff pane when op log focused
- [ ] Add property tests (rapid) for invariants
- [ ] Add unit tests for behavior

## Tests

### Property Tests (rapid)

| Test | Invariant |
|------|-----------|
| `TestParseOpLogLines_ValidOutput` | Any valid op log output parses without panic, produces >= 0 operations |
| `TestOpLogPanel_CursorBounds` | Cursor always in range `[0, len(operations)-1]` after any navigation |
| `TestOpLogPanel_SelectionConsistency` | `SelectedOperation()` returns nil iff operations empty |

### Unit Tests

| Test | Description |
|------|-------------|
| `TestOpLog_CommandArgs` | `OpLog()` calls jj with correct args |
| `TestOpShow_CommandArgs` | `OpShow(id)` calls jj with correct args |
| `TestParseOpLogLines_CurrentMarker` | `@` operation sets `IsCurrent = true` |
| `TestOpLogPanel_Navigation` | j/k/g/G move cursor correctly |
| `TestFocusPane2_ShowsOpDetails` | Focusing op log loads `jj op show` into diff pane |
| `TestFocusChange_RedrawsDiff` | Switching focus between log and op log redraws diff pane |

### Test Files

| File | Tests |
|------|-------|
| `internal/jj/runner_test.go` | `OpLog`, `OpShow`, `ParseOpLogLines` tests |
| `internal/ui/oplog_test.go` | `OpLogPanel` property and unit tests |
| `internal/app/dispatch_test.go` | Focus and op show loading tests |

## Files Changed

| File | Change |
|------|--------|
| `internal/jj/types.go` | Rename `IsWorkingCopy` to `IsCurrent`, add `Operation` struct |
| `internal/jj/runner.go` | Update field name, add `OpLog()`, `OpShow()`, `ParseOpLogLines()` |
| `internal/ui/oplog.go` | New file: `OpLogPanel` |
| `internal/app/app.go` | Add panel, layout, loading, focus handling, op show in diff pane |
| `internal/app/keys.go` | Add `FocusPane2` keybinding |

## Out of Scope (Future)

- `jj undo` integration from op log
- Collapsing/expanding the panel
- Dedicated op details pane (currently reuses diff pane)

## Labels

`enhancement`, `ui`
