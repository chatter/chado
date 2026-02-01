# File watcher doesn't refresh file list in ViewFiles mode

## Summary

When drilled into file view, filesystem changes (create, move, delete) don't update the file list until backing out and re-entering.

## Reproduction

1. Drill into file view in the app (select a change, press Enter)
2. In another terminal:
   - `touch .github/issues/test.md` (create)
   - `mv .github/issues/test.md ./test.md` (move)
   - `rm ./test.md` (delete)
3. **Expected**: File list updates to reflect changes
4. **Actual**: File list stays stale; must back out (Esc) and re-enter to see changes

## Root Causes

### 1. Watcher filters out `Rename` events

In `internal/jj/watcher.go` (lines 91-93):

```go
if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) == 0 {
    continue // Ignore other operations
}
```

`mv` triggers `fsnotify.Rename`, which is not in the filter mask.

**Fix**: Add `fsnotify.Rename` to the accepted operations:
```go
if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
```

### 2. ViewFiles mode doesn't reload file list

In `internal/app/app.go` (lines 295-306):

```go
case jj.WatcherMsg:
    cmds = append(cmds, m.loadLog(), m.waitForChange())

    if m.viewMode == ViewFiles {
        // Only reloads the selected file's diff, NOT the file list
        cmds = append(cmds, m.loadFileDiff(change, file.Path))
    }
```

**Fix**: Also call `loadFiles()` to refresh the file list:
```go
if m.viewMode == ViewFiles {
    if change := m.filesPanel.ChangeID(); change != "" {
        cmds = append(cmds, m.loadFiles(change))  // <-- Add this
        if file := m.filesPanel.SelectedFile(); file != nil {
            cmds = append(cmds, m.loadFileDiff(change, file.Path))
        }
    }
}
```

## Testing

Create a test that:
1. Sets up a watcher on a temp directory
2. Creates/moves/deletes files
3. Verifies watcher events are emitted for each operation
4. Verifies file list is refreshed when in ViewFiles mode

## Labels

`bug`, `watcher`, `ux`
