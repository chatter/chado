# Inline Status Bar Help

## Summary

Display context-sensitive keybinding hints in the status bar, showing as many bindings as fit (ordered by priority) with graceful truncation.

## Motivation

Users shouldn't have to memorize keybindings. The status bar should show relevant actions for the current context, similar to how `lazygit` and `htop` display contextual help.

## Design

```
j/k up/down • h/l pane • enter drill • ? help                    chado v0.1.0
└─────────────────────────────────────┘                          └───────────┘
         bindings (left)                                         version (right)
```

- Bindings sorted by `Order` (ascending)
- Renders left-to-right until width exhausted
- Truncates with `…` if bindings don't fit
- Version stays right-aligned
- Context-sensitive: shows global + focused panel bindings

## Implementation

### StatusBarHelp Component (`internal/ui/help/statusbar.go`)

```go
type StatusBarHelp struct {
    width    int
    version  string
    bindings []HelpBinding
}

func (s *StatusBarHelp) SetBindings(bindings []HelpBinding)
func (s *StatusBarHelp) SetWidth(width int)
func (s StatusBarHelp) View() string
```

### Context Collection

Each panel implements `HelpKeyMap`:

```go
func (p *LogPanel) HelpBindings() []HelpBinding
func (p *FilesPanel) HelpBindings() []HelpBinding  
func (p *DiffPanel) HelpBindings() []HelpBinding
```

App merges based on context:

```go
func (m *Model) activeBindings() []HelpBinding {
    bindings := m.globalBindings()
    switch m.focusedPane {
    case PaneLog:
        if m.viewMode == ViewLog {
            bindings = append(bindings, m.logPanel.HelpBindings()...)
        } else {
            bindings = append(bindings, m.filesPanel.HelpBindings()...)
        }
    case PaneDiff:
        bindings = append(bindings, m.diffPanel.HelpBindings()...)
    }
    return bindings
}
```

## Tasks

- [ ] Create `StatusBarHelp` component in `internal/ui/help/statusbar.go`
- [ ] Implement order-based sorting
- [ ] Implement width-aware truncation with `…`
- [ ] Add `HelpBindings()` to `LogPanel`
- [ ] Add `HelpBindings()` to `FilesPanel`
- [ ] Add `HelpBindings()` to `DiffPanel`
- [ ] Add `activeBindings()` method to Model
- [ ] Integrate into `renderStatusBar()`
- [ ] Style bindings (key style, desc style, separator)

## Tests

| Test | Description |
|------|-------------|
| `TestStatusBar_OrderedByPriority` | Bindings with orders [3,1,2] → renders [1,2,3] |
| `TestStatusBar_TruncatesWithEllipsis` | Width=40, 10 bindings → shows as many as fit + `…` |
| `TestStatusBar_VersionRightAligned` | Version stays right-aligned |
| `TestStatusBar_EmptyBindings` | No bindings → just shows version |
| `TestStatusBar_ContextSensitive` | Focused panel → shows global + panel bindings |

## Acceptance Criteria

- [ ] Status bar shows keybinding hints
- [ ] Hints change based on focused panel
- [ ] Gracefully truncates when terminal is narrow
- [ ] Version remains visible and right-aligned
- [ ] Tests pass

## Dependencies

- #1 Action-Binding Refactor (provides `HelpBinding` types)
