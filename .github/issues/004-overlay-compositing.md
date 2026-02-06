# Consider bubbletea-overlay for floating help modal

## Summary

Evaluate using [bubbletea-overlay](https://github.com/rmhubbert/bubbletea-overlay) for the floating help modal to achieve true compositing (see-through) effect.

## Current Behavior

The floating help modal uses `lipgloss.Place` which centers the modal and fills the background with a solid color, obscuring the view behind it.

## Proposed Enhancement

Use the overlay library to composite the help modal on top of the existing view, allowing the panels behind to remain partially visible.

**Visual concept:** The modal would "float" in the bottom-left corner (above the status bar), creating a visual connection to the inline help while letting you see the panels behind/around it.

## Implementation

```go
import overlay "github.com/rmhubbert/bubbletea-overlay"

// Position: bottom-left with slight offset
overlay.New(helpModel, baseModel, overlay.Left, overlay.Bottom, 2, 1)
```

## Trade-offs

**Pros:**
- True compositing effect
- Clean positioning API (Left/Right/Top/Bottom/Center + offsets)
- Well-maintained (v0.6.4, MIT license)

**Cons:**
- Additional dependency
- Visual polish, not functional improvement

## Priority

Low - nice-to-have polish. Current implementation is functional.

## Labels

`enhancement`, `polish`, `low-priority`
