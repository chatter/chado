# Upgrade Bubble Tea, Lip Gloss, and Bubbles to v2

## Overview

Migrate from v1 (`github.com/charmbracelet/*`) to v2 (`charm.land/*/v2`). All three packages are in beta/RC but work together.

## Current Versions

| Package | Current | Target |
|---------|---------|--------|
| bubbletea | v1.3.10 | charm.land/bubbletea/v2 (v2.0.0-rc.2) |
| bubbles | v0.21.0 | charm.land/bubbles/v2 (v2.0.0-rc.1) |
| lipgloss | v1.1.0 | charm.land/lipgloss/v2 (v2.0.0-beta.3) |

## Import Path Changes

All 17 files need import updates:

| Old Path | New Path |
|----------|----------|
| `github.com/charmbracelet/bubbletea` | `charm.land/bubbletea/v2` |
| `github.com/charmbracelet/bubbles/*` | `charm.land/bubbles/v2/*` |
| `github.com/charmbracelet/lipgloss` | `charm.land/lipgloss/v2` |

### Files to update

- `main.go`
- `internal/app/app.go`, `keys.go`, `dispatch_test.go`
- `internal/ui/styles.go`, `log.go`, `oplog.go`, `files.go`, `diff.go`, `util.go`
- `internal/ui/help/statusbar.go`, `floating.go`, `types.go`
- Test files: `*_test.go`

## Known v2 API Changes

### Bubble Tea v2

- **KeyMsg**: Structure updated, but `key.Matches(msg, binding)` pattern should still work
- **Context**: New optional `tea.WithContext(ctx)` for terminal feature detection
- **Synchronized output**: Enabled by default (good for us)

### Bubbles v2 (viewport)

- Method renames: `LineUp` → `ScrollUp`, `ViewUp` → `PageUp`
- We already use `ScrollUp`/`ScrollDown` - no changes needed
- `HighPerformanceRendering` deprecated (we don't use it)
- New: horizontal scrolling support (optional)

### Lipgloss v2

- **Deterministic styles**: More precise control over rendering
- **I/O control**: Better integration with Bubble Tea (lockstep)
- **New compositing**: `Layer` and `Canvas` types for overlay rendering
- **Padding chars**: Uses regular spaces instead of NBSP (better copy/paste)
- Our basic style usage should work unchanged

## Implementation Steps

- [ ] Update go.mod to use charm.land/*/v2 paths
- [ ] Replace all github.com/charmbracelet imports with charm.land/v2 equivalents
- [ ] Fix any API breaking changes (KeyMsg, Style, viewport methods)
- [ ] Run test suite and fix any failures
- [ ] Build and manually test the TUI

## Test Coverage

Current tests focus on behavior rather than visual output:

- **Good coverage**: Help components (statusbar, floating modal) have View() output tests
- **Behavioral only**: Main panels (log, files, diff, oplog) test state/navigation
- **No golden tests**: No exact visual output comparison

Existing tests plus manual testing should be sufficient for this upgrade.

## Risk Assessment

- **Low risk**: Basic APIs (Style, viewport, key matching) appear stable
- **Medium risk**: KeyMsg structure changes may affect dispatch
- **Opportunity**: Lipgloss v2 Layer/Canvas could simplify our help overlay

## Rollback

If issues arise, revert go.mod changes - v1 remains available at original paths.
