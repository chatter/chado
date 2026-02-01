package ui

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/chatter/chado/internal/jj"
)

// =============================================================================
// Unit Tests
// =============================================================================

func TestFilesPanel_SetSize(t *testing.T) {
	panel := NewFilesPanel()
	panel.SetSize(80, 30)

	if panel.width != 80 {
		t.Errorf("width should be 80, got %d", panel.width)
	}
	if panel.height != 30 {
		t.Errorf("height should be 30, got %d", panel.height)
	}
}

func TestFilesPanel_Focus(t *testing.T) {
	panel := NewFilesPanel()

	if panel.focused {
		t.Error("panel should not be focused initially")
	}

	panel.SetFocused(true)
	if !panel.focused {
		t.Error("panel should be focused after SetFocused(true)")
	}
}

func TestFilesPanel_SetFiles(t *testing.T) {
	panel := NewFilesPanel()
	panel.SetSize(80, 24)

	files := []jj.File{
		{Path: "main.go", Status: jj.FileModified},
		{Path: "app.go", Status: jj.FileAdded},
		{Path: "old.go", Status: jj.FileDeleted},
	}

	panel.SetFiles("xsssnyux", "xsss", files)

	if panel.changeID != "xsssnyux" {
		t.Errorf("changeID should be 'xsssnyux', got %s", panel.changeID)
	}
	if len(panel.files) != 3 {
		t.Errorf("should have 3 files, got %d", len(panel.files))
	}
	if panel.cursor != 0 {
		t.Errorf("cursor should reset to 0, got %d", panel.cursor)
	}
}

func TestFilesPanel_CursorNavigation(t *testing.T) {
	panel := NewFilesPanel()
	panel.SetSize(80, 24)

	files := []jj.File{
		{Path: "a.go", Status: jj.FileModified},
		{Path: "b.go", Status: jj.FileAdded},
		{Path: "c.go", Status: jj.FileDeleted},
	}
	panel.SetFiles("test", "t", files)

	// Test CursorDown
	panel.CursorDown()
	if panel.cursor != 1 {
		t.Errorf("cursor should be 1 after CursorDown, got %d", panel.cursor)
	}

	// Test CursorUp
	panel.CursorUp()
	if panel.cursor != 0 {
		t.Errorf("cursor should be 0 after CursorUp, got %d", panel.cursor)
	}

	// Test bounds - can't go above 0
	panel.CursorUp()
	if panel.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", panel.cursor)
	}

	// Test bounds - can't go past last
	panel.CursorDown()
	panel.CursorDown()
	panel.CursorDown() // Try to go past
	if panel.cursor != 2 {
		t.Errorf("cursor should stop at 2, got %d", panel.cursor)
	}

	// Test GotoTop
	panel.GotoTop()
	if panel.cursor != 0 {
		t.Errorf("cursor should be 0 after GotoTop, got %d", panel.cursor)
	}

	// Test GotoBottom
	panel.GotoBottom()
	if panel.cursor != 2 {
		t.Errorf("cursor should be 2 after GotoBottom, got %d", panel.cursor)
	}
}

func TestFilesPanel_SelectedFile(t *testing.T) {
	panel := NewFilesPanel()

	// Empty panel
	if panel.SelectedFile() != nil {
		t.Error("SelectedFile should be nil for empty panel")
	}

	// With files
	files := []jj.File{
		{Path: "a.go", Status: jj.FileModified},
		{Path: "b.go", Status: jj.FileAdded},
	}
	panel.SetFiles("test", "t", files)

	selected := panel.SelectedFile()
	if selected == nil {
		t.Fatal("SelectedFile should not be nil")
	}
	if selected.Path != "a.go" {
		t.Errorf("expected first file, got %s", selected.Path)
	}

	// Move cursor and check
	panel.CursorDown()
	selected = panel.SelectedFile()
	if selected.Path != "b.go" {
		t.Errorf("expected second file, got %s", selected.Path)
	}
}

func TestFilesPanel_ChangeID(t *testing.T) {
	panel := NewFilesPanel()

	if panel.ChangeID() != "" {
		t.Error("ChangeID should be empty initially")
	}

	panel.SetFiles("xsssnyux", "xsss", nil)
	if panel.ChangeID() != "xsssnyux" {
		t.Errorf("ChangeID should be 'xsssnyux', got %s", panel.ChangeID())
	}
}

func TestFilesPanel_EmptyFiles(t *testing.T) {
	panel := NewFilesPanel()
	panel.SetSize(80, 24)
	panel.SetFiles("test", "", nil)

	// Should not panic
	panel.CursorDown()
	panel.CursorUp()
	panel.GotoTop()
	panel.GotoBottom()

	if panel.SelectedFile() != nil {
		t.Error("SelectedFile should be nil for empty files")
	}
}

// =============================================================================
// Property Tests
// =============================================================================

// Property: Cursor should always be within valid bounds
func TestFilesPanel_CursorAlwaysInBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewFilesPanel()
		panel.SetSize(80, 24)

		// Generate random files
		numFiles := rapid.IntRange(0, 100).Draw(t, "numFiles")
		files := make([]jj.File, numFiles)
		for i := range numFiles {
			files[i] = jj.File{
				Path:   rapid.StringMatching(`[a-z]{3,10}\.go`).Draw(t, "path"),
				Status: jj.FileStatus(rapid.SampledFrom([]string{"M", "A", "D"}).Draw(t, "status")),
			}
		}
		panel.SetFiles("test", "t", files)

		// Perform random operations
		numOps := rapid.IntRange(0, 50).Draw(t, "numOps")
		for range numOps {
			op := rapid.IntRange(0, 3).Draw(t, "op")
			switch op {
			case 0:
				panel.CursorUp()
			case 1:
				panel.CursorDown()
			case 2:
				panel.GotoTop()
			case 3:
				panel.GotoBottom()
			}
		}

		// Check invariants
		if numFiles == 0 {
			if panel.cursor != 0 {
				t.Fatalf("cursor should be 0 for empty files, got %d", panel.cursor)
			}
		} else {
			if panel.cursor < 0 {
				t.Fatalf("cursor should never be negative, got %d", panel.cursor)
			}
			if panel.cursor >= numFiles {
				t.Fatalf("cursor %d should be < numFiles %d", panel.cursor, numFiles)
			}
		}
	})
}

// Property: SelectedFile should match cursor position
func TestFilesPanel_SelectedFileMatchesCursor(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewFilesPanel()
		panel.SetSize(80, 24)

		numFiles := rapid.IntRange(1, 50).Draw(t, "numFiles")
		files := make([]jj.File, numFiles)
		for i := range numFiles {
			files[i] = jj.File{
				Path:   rapid.StringMatching(`[a-z]{5,10}\.go`).Draw(t, "path") + string(rune('0'+i)),
				Status: jj.FileModified,
			}
		}
		panel.SetFiles("test", "t", files)

		// Random cursor position
		targetPos := rapid.IntRange(0, numFiles-1).Draw(t, "targetPos")
		for range targetPos {
			panel.CursorDown()
		}

		selected := panel.SelectedFile()
		if selected == nil {
			t.Fatal("SelectedFile should not be nil")
			return
		}
		if selected.Path != files[panel.cursor].Path {
			t.Fatalf("selected file path mismatch: got %s, expected %s",
				selected.Path, files[panel.cursor].Path)
		}
	})
}

// Property: SetFiles should always reset cursor to 0
func TestFilesPanel_SetFilesResetsCursor(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewFilesPanel()
		panel.SetSize(80, 24)

		// Set initial files and move cursor
		initialFiles := []jj.File{{Path: "a.go"}, {Path: "b.go"}, {Path: "c.go"}}
		panel.SetFiles("first", "first", initialFiles)
		moves := rapid.IntRange(0, 5).Draw(t, "moves")
		for range moves {
			panel.CursorDown()
		}

		// Set new files
		numFiles := rapid.IntRange(0, 10).Draw(t, "numFiles")
		newFiles := make([]jj.File, numFiles)
		for i := range numFiles {
			newFiles[i] = jj.File{Path: rapid.String().Draw(t, "path")}
		}
		panel.SetFiles("second", "second", newFiles)

		if panel.cursor != 0 {
			t.Fatalf("cursor should be 0 after SetFiles, got %d", panel.cursor)
		}
	})
}

// Property: GotoTop always results in cursor=0
func TestFilesPanel_GotoTopAlwaysZero(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewFilesPanel()
		panel.SetSize(80, 24)

		numFiles := rapid.IntRange(1, 50).Draw(t, "numFiles")
		files := make([]jj.File, numFiles)
		for i := range numFiles {
			files[i] = jj.File{Path: "file" + string(rune('a'+i))}
		}
		panel.SetFiles("test", "t", files)

		// Move around
		moves := rapid.IntRange(0, 30).Draw(t, "moves")
		for range moves {
			panel.CursorDown()
		}

		panel.GotoTop()
		if panel.cursor != 0 {
			t.Fatalf("cursor should be 0 after GotoTop, got %d", panel.cursor)
		}
	})
}

// Property: GotoBottom always results in cursor at last item
func TestFilesPanel_GotoBottomAlwaysLast(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewFilesPanel()
		panel.SetSize(80, 24)

		numFiles := rapid.IntRange(1, 50).Draw(t, "numFiles")
		files := make([]jj.File, numFiles)
		for i := range numFiles {
			files[i] = jj.File{Path: "file" + string(rune('a'+i))}
		}
		panel.SetFiles("test", "t", files)

		panel.GotoBottom()
		if panel.cursor != numFiles-1 {
			t.Fatalf("cursor should be %d after GotoBottom, got %d", numFiles-1, panel.cursor)
		}
	})
}

// =============================================================================
// Mouse Click Property Tests
// =============================================================================

// Property: After any click, cursor stays in valid range [0, len(files)-1]
func TestFilesPanel_Click_CursorInBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewFilesPanel()
		width := rapid.IntRange(40, 200).Draw(t, "width")
		height := rapid.IntRange(10, 100).Draw(t, "height")
		panel.SetSize(width, height)

		// Generate files
		numFiles := rapid.IntRange(1, 50).Draw(t, "numFiles")
		files := make([]jj.File, numFiles)
		for i := range numFiles {
			files[i] = jj.File{Path: "file" + string(rune('a'+i)), Status: jj.FileModified}
		}
		panel.SetFiles("test", "t", files)

		// Click at any Y position (including invalid: negative, huge)
		clickY := rapid.IntRange(-100, 500).Draw(t, "clickY")
		panel.HandleClick(clickY)

		// Invariant: cursor in bounds
		if panel.cursor < 0 {
			t.Fatalf("cursor should be >= 0, got %d", panel.cursor)
		}
		if panel.cursor >= numFiles {
			t.Fatalf("cursor should be < %d, got %d", numFiles, panel.cursor)
		}
	})
}

// Property: Click at valid index selects that file
func TestFilesPanel_Click_SelectsCorrectFile(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewFilesPanel()
		width := rapid.IntRange(40, 200).Draw(t, "width")
		height := rapid.IntRange(10, 100).Draw(t, "height")
		panel.SetSize(width, height)

		numFiles := rapid.IntRange(1, 50).Draw(t, "numFiles")
		files := make([]jj.File, numFiles)
		for i := range numFiles {
			files[i] = jj.File{Path: "file" + string(rune('a'+i)), Status: jj.FileModified}
		}
		panel.SetFiles("test", "t", files)

		// Click at valid index
		targetIdx := rapid.IntRange(0, numFiles-1).Draw(t, "targetIdx")
		panel.HandleClick(targetIdx)

		// Invariant: cursor matches clicked index
		if panel.cursor != targetIdx {
			t.Fatalf("cursor should be %d, got %d", targetIdx, panel.cursor)
		}
	})
}

// Property: Click outside bounds doesn't change cursor
func TestFilesPanel_Click_OutOfBounds_NoChange(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewFilesPanel()
		width := rapid.IntRange(40, 200).Draw(t, "width")
		height := rapid.IntRange(10, 100).Draw(t, "height")
		panel.SetSize(width, height)

		numFiles := rapid.IntRange(1, 50).Draw(t, "numFiles")
		files := make([]jj.File, numFiles)
		for i := range numFiles {
			files[i] = jj.File{Path: "file" + string(rune('a'+i)), Status: jj.FileModified}
		}
		panel.SetFiles("test", "t", files)

		// Set cursor to random valid position
		startCursor := rapid.IntRange(0, numFiles-1).Draw(t, "startCursor")
		panel.cursor = startCursor

		// Click outside bounds (negative or >= numFiles)
		invalidY := rapid.SampledFrom([]int{
			rapid.IntRange(-100, -1).Draw(t, "negativeY"),
			rapid.IntRange(numFiles, numFiles+100).Draw(t, "tooLargeY"),
		}).Draw(t, "invalidY")

		changed := panel.HandleClick(invalidY)

		// Invariant: cursor unchanged, returns false
		if changed {
			t.Fatalf("HandleClick should return false for out-of-bounds click")
		}
		if panel.cursor != startCursor {
			t.Fatalf("cursor should remain %d after invalid click, got %d", startCursor, panel.cursor)
		}
	})
}

// Property: Clicking same position returns false
func TestFilesPanel_Click_SamePosition_ReturnsFalse(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		panel := NewFilesPanel()
		width := rapid.IntRange(40, 200).Draw(t, "width")
		height := rapid.IntRange(10, 100).Draw(t, "height")
		panel.SetSize(width, height)

		numFiles := rapid.IntRange(1, 50).Draw(t, "numFiles")
		files := make([]jj.File, numFiles)
		for i := range numFiles {
			files[i] = jj.File{Path: "file" + string(rune('a'+i)), Status: jj.FileModified}
		}
		panel.SetFiles("test", "t", files)

		// Set cursor to a position
		cursorPos := rapid.IntRange(0, numFiles-1).Draw(t, "cursorPos")
		panel.cursor = cursorPos

		// Click on the same position
		changed := panel.HandleClick(cursorPos)

		// Invariant: returns false, cursor unchanged
		if changed {
			t.Fatalf("HandleClick should return false when clicking already-selected file")
		}
		if panel.cursor != cursorPos {
			t.Fatalf("cursor should remain %d, got %d", cursorPos, panel.cursor)
		}
	})
}

// Benchmark for cursor navigation
func BenchmarkFilesPanel_Navigation(b *testing.B) {
	panel := NewFilesPanel()
	panel.SetSize(80, 24)

	files := make([]jj.File, 1000)
	for i := range 1000 {
		files[i] = jj.File{Path: "file" + string(rune(i))}
	}
	panel.SetFiles("test", "t", files)

	for b.Loop() {
		panel.CursorDown()
		panel.CursorDown()
		panel.CursorUp()
		panel.GotoBottom()
		panel.GotoTop()
	}
}
