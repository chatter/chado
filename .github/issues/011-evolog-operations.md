# Context-Aware Evolog Panel

## Summary

Transform the bottom-left panel to be context-aware: showing global `jj op log` in Log view, and scoped `jj evolog -r <change-id>` (operations that affected a specific change) when drilled into Files view.

## Motivation

The evolution log (`jj evolog`) shows how a specific change has been modified over time - rewrites, description changes, squashes, etc. When drilling into a change's files, showing the evolog in the bottom panel provides immediate context about that change's history without leaving the current view.

## Key Insight

`jj evolog -r <rev>` outputs operations in the same format as `jj op log`, just filtered to operations that modified the specified change. This means:

- Reuse the existing `Operation` type
- Reuse `ParseOpLogLines()` to parse evolog output
- Reuse `OpShow(opID)` for the diff pane
- Only the panel title and data source change between modes

## Architecture

```
LOG VIEW (top level)                    FILES VIEW (drilled in)
┌─────────────────┬──────────────┐      ┌─────────────────┬──────────────┐
│   LogPanel      │              │      │  FilesPanel     │              │
│                 │              │      │  (change files) │              │
│  jj log         │   DiffPanel  │      │                 │   DiffPanel  │
├─────────────────┤              │      ├─────────────────┤              │
│  OpLogPanel     │  jj op show  │      │  OpLogPanel     │  jj op show  │
│                 │    <opID>    │      │  (evolog mode)  │    <opID>    │
│  jj op log      │              │      │                 │              │
│                 │              │      │  jj evolog -r   │              │
│                 │              │      │    <changeID>   │              │
└─────────────────┴──────────────┘      └─────────────────┴──────────────┘
        │                                       ▲
        │         Enter on change               │
        └───────────────────────────────────────┘
                         Esc
```

## Implementation (TDD)

### Phase 1: Runner - EvoLog Method

#### 1a. Write Failing Tests

Add to `internal/jj/runner_test.go`:

```go
func TestEvoLog_CommandArgs(t *testing.T) {
    // Test that EvoLog calls jj with correct arguments:
    // jj evolog -r <rev> --color=always
}

func TestEvoLog_ParsesAsOperations(t *testing.T) {
    // Test that ParseOpLogLines correctly parses evolog output
    // (same format as op log, should work unchanged)
}
```

#### 1b. Implement to Pass

Add to `internal/jj/runner.go`:

```go
// EvoLog returns the evolution log for a specific change (operations that affected it)
func (r *Runner) EvoLog(rev string) (string, error) {
    return r.Run("evolog", "-r", rev, "--color=always")
}
```

### Phase 2: OpLogPanel - Mode Toggle

#### 2a. Write Failing Tests

Add to `internal/ui/oplog_test.go`:

```go
func TestOpLogPanel_DefaultModeIsOpLog(t *testing.T) {
    // New panel should be in OpLog mode
}

func TestOpLogPanel_SetEvoLogContent_SwitchesMode(t *testing.T) {
    // Calling SetEvoLogContent should switch to EvoLog mode
}

func TestOpLogPanel_SetOpLogContent_SwitchesMode(t *testing.T) {
    // Calling SetOpLogContent should switch to OpLog mode
}

func TestOpLogPanel_TitleByMode(t *testing.T) {
    // OpLog mode: title contains "Operations Log"
    // EvoLog mode: title contains "Evolution:" and the shortCode
}

func TestOpLogPanel_EvoLogTitle_HighlightsShortCode(t *testing.T) {
    // ShortCode portion should be styled differently (like FilesPanel)
}
```

#### 2b. Implement to Pass

Update `internal/ui/oplog.go`:

```go
type OpLogMode int

const (
    ModeOpLog  OpLogMode = iota // Global operation log
    ModeEvoLog                   // Evolution log for a specific change
)

type OpLogPanel struct {
    // ... existing fields ...
    mode      OpLogMode
    changeID  string    // Set when in evolog mode
    shortCode string    // Short prefix for title display
}

// SetEvoLogContent switches to evolog mode for a specific change
func (p *OpLogPanel) SetEvoLogContent(changeID, shortCode, rawLog string, operations []jj.Operation) {
    p.mode = ModeEvoLog
    p.changeID = changeID
    p.shortCode = shortCode
    p.SetContent(rawLog, operations)
}

// SetOpLogContent switches to global op log mode
func (p *OpLogPanel) SetOpLogContent(rawLog string, operations []jj.Operation) {
    p.mode = ModeOpLog
    p.changeID = ""
    p.shortCode = ""
    p.SetContent(rawLog, operations)
}
```

Update `View()` to show contextual title:
- Op log mode: "Operations Log"
- Evolog mode: "Evolution: `<shortCode>`" (with shortCode highlighted like in FilesPanel)

### Phase 3: App Integration

#### 3a. Write Failing Tests

Add to `internal/app/dispatch_test.go`:

```go
func TestDrillIn_LoadsEvoLog(t *testing.T) {
    // When entering files view, evolog should be loaded for the change
}

func TestDrillOut_RestoresOpLog(t *testing.T) {
    // When exiting files view, global op log should be restored
}

func TestEvoLog_OpShowWorks(t *testing.T) {
    // Selecting evolog entry should load op show in diff pane
    // (may already pass since it reuses existing OpShow logic)
}
```

#### 3b. Implement to Pass

Add to `internal/app/app.go`:

```go
type evoLogLoadedMsg struct {
    changeID   string
    shortCode  string
    raw        string
    operations []jj.Operation
}

func (m Model) loadEvoLog(changeID, shortCode string) tea.Cmd {
    return func() tea.Msg {
        output, err := m.runner.EvoLog(changeID)
        if err != nil {
            return errMsg{err}
        }
        operations := m.runner.ParseOpLogLines(output)
        return evoLogLoadedMsg{
            changeID:   changeID,
            shortCode:  shortCode,
            raw:        output,
            operations: operations,
        }
    }
}
```

Wire up:
- `handleEnter()`: Load evolog when drilling into files view
- `handleBack()`: Restore global op log when returning to log view
- Message handler for `evoLogLoadedMsg`

## Tasks

- [ ] Add failing tests for `EvoLog` runner method
- [ ] Implement `EvoLog(rev)` method to make tests pass
- [ ] Add failing tests for OpLogPanel mode toggle and dynamic title
- [ ] Implement mode toggle in OpLogPanel to make tests pass
- [ ] Add failing tests for evolog loading and view transitions
- [ ] Wire up `loadEvoLog`, `evoLogLoadedMsg`, and view transitions to make tests pass

## Tests

### Property Tests (rapid)

| Test | Invariant |
|------|-----------|
| `TestEvoLog_ParsesAsOperations` | Evolog output parses with same logic as op log |

### Unit Tests

| Test | Description |
|------|-------------|
| `TestEvoLog_CommandArgs` | `EvoLog(rev)` calls `jj evolog -r rev --color=always` |
| `TestOpLogPanel_DefaultModeIsOpLog` | New panel starts in OpLog mode |
| `TestOpLogPanel_SetEvoLogContent_SwitchesMode` | SetEvoLogContent switches to EvoLog mode |
| `TestOpLogPanel_SetOpLogContent_SwitchesMode` | SetOpLogContent switches to OpLog mode |
| `TestOpLogPanel_TitleByMode` | Title shows "Operations Log" vs "Evolution: xxx" |
| `TestDrillIn_LoadsEvoLog` | Entering files view triggers evolog load |
| `TestDrillOut_RestoresOpLog` | Exiting files view reloads op log |

### Test Files

| File | Tests |
|------|-------|
| `internal/jj/runner_test.go` | `EvoLog` command tests |
| `internal/ui/oplog_test.go` | Mode toggle and title tests |
| `internal/app/dispatch_test.go` | View transition tests |

## Files Changed

| File | Change |
|------|--------|
| `internal/jj/runner_test.go` | Add `TestEvoLog_*` tests |
| `internal/jj/runner.go` | Add `EvoLog(rev)` method |
| `internal/ui/oplog_test.go` | Add mode toggle and title tests |
| `internal/ui/oplog.go` | Add mode field, `SetEvoLogContent()`, `SetOpLogContent()`, dynamic title |
| `internal/app/dispatch_test.go` | Add view transition tests |
| `internal/app/app.go` | Add `loadEvoLog()`, `evoLogLoadedMsg`, wire up transitions |

## Out of Scope (Future)

- Restore change to previous evolution state (`jj restore`)
- Evolog navigation actions (jump to operation in global op log)
- Evolog filtering/searching

## Labels

`enhancement`, `ui`
