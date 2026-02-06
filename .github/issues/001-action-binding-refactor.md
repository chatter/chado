# Action-Binding Refactor

## Summary

Refactor keybinding architecture to co-locate bindings with their actions, replacing the switch statement in `Update()` with a simple binding iteration pattern.

## Motivation

Currently, keybindings are defined in `keys.go` but their actions are scattered across a large switch statement in `app.go`. This creates:
- Duplication between binding definition and action
- Help text that can drift from actual behavior
- Boilerplate for each new binding

The action-binding pattern unifies these: each binding knows its own action.

## Implementation

### New Types (`internal/ui/help/types.go`)

```go
type Category string

const (
    CategoryNavigation Category = "Navigation"
    CategoryActions    Category = "Actions"
    CategoryDiff       Category = "Diff"
)

type Action func(m *Model) tea.Cmd

type HelpBinding struct {
    Binding  key.Binding
    Category Category
    Order    int     // lower = higher priority for inline status bar
    Action   Action  // nil = display-only
}

type HelpKeyMap interface {
    HelpBindings() []HelpBinding
}
```

### Refactored Dispatch (`app.go`)

```go
case tea.KeyMsg:
    for _, hb := range m.activeBindings() {
        if key.Matches(msg, hb.Binding) && hb.Action != nil {
            return hb.Action(m)
        }
    }
```

## Tasks

- [ ] Create `internal/ui/help/` package with `types.go`
- [ ] Define `Category` enum with initial categories
- [ ] Define `HelpBinding` struct with `Action` field
- [ ] Define `HelpKeyMap` interface
- [ ] Refactor `internal/app/keys.go` to return `[]HelpBinding`
- [ ] Add `activeBindings()` method to Model
- [ ] Replace switch statement in `Update()` with binding iteration
- [ ] Ensure all existing keybindings work as before

## Tests

| Test | Description |
|------|-------------|
| `TestDispatch_MatchesAndExecutes` | Matching key pressed → correct action executes |
| `TestDispatch_NoMatchNoAction` | Unbound key pressed → no action |
| `TestDispatch_NilActionSkipped` | Binding with nil action → no panic |
| `TestDispatch_FirstMatchWins` | Overlapping bindings → first match wins |
| `TestDispatch_DisabledBindingSkipped` | Disabled binding → skipped |

## Acceptance Criteria

- [ ] All existing keybindings function identically
- [ ] No switch statement for key dispatch in `Update()`
- [ ] `HelpBinding` includes category, order, and action
- [ ] Tests pass
