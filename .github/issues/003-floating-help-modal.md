# Floating Help Modal

## Summary

Add a `?` toggle to show a floating modal with all keybindings organized by category in a column-flow layout.

## Motivation

The inline status bar can only show a few bindings. Power users need a way to see all available keybindings at once, organized logically.

## Design

```
┌─ Help ────────────────────────────────────────────────────┐
│                                                           │
│  Navigation              Actions                          │
│    j/k    up/down          enter  drill down / select     │
│    h/l    switch pane      esc    go back                 │
│    g      top              q      quit                    │
│    G      bottom                                          │
│    0/1    focus pane     Diff                             │
│                            {/}    prev/next hunk          │
│                            gg/G   top/bottom              │
│                                                           │
│                                              ? to close   │
└───────────────────────────────────────────────────────────┘
```

- Toggle with `?` (same key opens and closes)
- Renders as overlay on top of existing UI
- Groups bindings by `Category`
- Categories ordered by enum value
- Column-flow layout: fills top-to-bottom, then wraps to next column
- Footer shows dismiss hint

## Implementation

### FloatingHelp Component (`internal/ui/help/floating.go`)

```go
type FloatingHelp struct {
    width    int
    height   int
    bindings []HelpBinding
}

func (f *FloatingHelp) SetSize(width, height int)
func (f *FloatingHelp) SetBindings(bindings []HelpBinding)
func (f FloatingHelp) View() string
```

### Column-Flow Algorithm

1. Group bindings by category
2. Calculate total rows needed per category (header + items)
3. Determine available rows per column
4. Flow categories into columns, allowing category to split across columns if needed

### Toggle Integration (`app.go`)

```go
type Model struct {
    // ...
    showHelp bool
}

// In Update():
case tea.KeyMsg:
    if msg.String() == "?" {
        m.showHelp = !m.showHelp
        return m, nil
    }
    if m.showHelp {
        return m, nil  // absorb other keys while modal open
    }
    // ... normal dispatch

// In View():
if m.showHelp {
    return m.renderWithOverlay(content, m.floatingHelp.View())
}
```

### Overlay Rendering

```go
func (m Model) renderWithOverlay(base, overlay string) string {
    // Center overlay on base content
    // Use lipgloss.Place or manual positioning
}
```

## Pinned Status Bar Binding

The `?` binding should **always** appear in the status bar, never truncated:

```
j/k up • h/l pane • …  • ? help                    chado v1.0.0
                       ^^^^^^^^
                       always visible
```

Add `Pinned bool` to `HelpBinding` or handle specially in StatusBar rendering.

## Tasks

- [ ] Create `FloatingHelp` component in `internal/ui/help/floating.go`
- [ ] Implement category grouping
- [ ] Implement column-flow layout algorithm
- [ ] Style modal (border, title, footer)
- [ ] Add `showHelp` state to Model
- [ ] Handle `?` toggle in Update()
- [ ] Absorb keypresses while modal is open
- [ ] Implement overlay rendering in View()
- [ ] Pass bindings to FloatingHelp on toggle
- [ ] Pin `?` binding in status bar (never truncated)

## Tests

| Test | Description |
|------|-------------|
| `TestFloating_GroupsByCategory` | Mixed category bindings → grouped correctly |
| `TestFloating_CategoryOrder` | Categories ordered by enum value |
| `TestFloating_ColumnFlow` | Many bindings → flows top-to-bottom, left-to-right |
| `TestFloating_ToggleOpensCloses` | `?` pressed → toggles visibility |
| `TestFloating_OverlayRendering` | Modal open → renders on top of content |

## Acceptance Criteria

- [ ] `?` opens floating help modal
- [ ] `?` again closes it
- [ ] Bindings grouped by category
- [ ] Column-flow layout works for various terminal sizes
- [ ] Modal is centered and styled
- [ ] Other keys absorbed while modal open
- [ ] Tests pass

## Dependencies

- #1 Action-Binding Refactor (provides `HelpBinding` types)
- #2 Inline Status Bar Help (provides panel `HelpBindings()` methods)
