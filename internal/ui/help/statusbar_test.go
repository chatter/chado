package help

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"pgregory.net/rapid"
)

// generateBinding creates a random HelpBinding
func generateBinding(t *rapid.T, idx int) HelpBinding {
	keyStr := string(rune('a' + idx%26))
	desc := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "desc")
	order := rapid.IntRange(0, 100).Draw(t, "order")
	category := rapid.SampledFrom([]Category{CategoryNavigation, CategoryActions, CategoryDiff}).Draw(t, "category")
	enabled := rapid.Bool().Draw(t, "enabled")

	binding := key.NewBinding(key.WithKeys(keyStr), key.WithHelp(keyStr, desc))
	if !enabled {
		binding.SetEnabled(false)
	}

	return HelpBinding{
		Binding:  binding,
		Category: category,
		Order:    order,
	}
}

func generateBindings(t *rapid.T) []HelpBinding {
	numBindings := rapid.IntRange(0, 20).Draw(t, "numBindings")
	bindings := make([]HelpBinding, numBindings)
	for i := 0; i < numBindings; i++ {
		bindings[i] = generateBinding(t, i)
	}
	return bindings
}

func TestStatusBar_WidthNeverExceeded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(20, 200).Draw(t, "width")
		bindings := generateBindings(t)

		sb := NewStatusBar("v1.0.0")
		sb.SetBindings(bindings)
		sb.SetWidth(width)

		view := sb.View()
		viewWidth := lipgloss.Width(view)

		if viewWidth > width {
			t.Errorf("view width %d exceeds specified width %d: %q", viewWidth, width, view)
		}
	})
}

func TestStatusBar_VersionAlwaysPresent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(20, 200).Draw(t, "width")
		version := rapid.StringMatching(`v[0-9]+\.[0-9]+\.[0-9]+`).Draw(t, "version")
		bindings := generateBindings(t)

		sb := NewStatusBar(version)
		sb.SetBindings(bindings)
		sb.SetWidth(width)

		view := sb.View()

		if !strings.Contains(view, version) {
			t.Errorf("version %q not found in view: %q", version, view)
		}
	})
}

func TestStatusBar_VersionAtEnd(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(20, 200).Draw(t, "width")
		bindings := generateBindings(t)

		sb := NewStatusBar("v1.0.0")
		sb.SetBindings(bindings)
		sb.SetWidth(width)

		view := sb.View()

		if !strings.HasSuffix(strings.TrimSpace(view), "v1.0.0") {
			t.Errorf("version not at end: %q", view)
		}
	})
}

func TestStatusBar_DisabledBindingsNeverAppear(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(50, 200).Draw(t, "width")

		// Create some disabled bindings with unique descriptions
		numBindings := rapid.IntRange(1, 10).Draw(t, "numBindings")
		bindings := make([]HelpBinding, numBindings)
		for i := 0; i < numBindings; i++ {
			desc := "disabled" + string(rune('0'+i))
			binding := key.NewBinding(key.WithKeys("x"), key.WithHelp("x", desc))
			binding.SetEnabled(false)
			bindings[i] = HelpBinding{
				Binding:  binding,
				Category: CategoryActions,
				Order:    i,
			}
		}

		sb := NewStatusBar("v1.0.0")
		sb.SetBindings(bindings)
		sb.SetWidth(width)

		view := sb.View()

		for i := 0; i < numBindings; i++ {
			desc := "disabled" + string(rune('0'+i))
			if strings.Contains(view, desc) {
				t.Errorf("disabled binding %q should not appear in view: %q", desc, view)
			}
		}
	})
}

func TestStatusBar_BindingsOrderedByPriority(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(100, 300).Draw(t, "width") // Wide enough to show multiple

		// Create bindings with distinct orders and descriptions that reveal order
		numBindings := rapid.IntRange(2, 5).Draw(t, "numBindings")
		bindings := make([]HelpBinding, numBindings)
		orders := make([]int, numBindings)

		for i := 0; i < numBindings; i++ {
			orders[i] = rapid.IntRange(0, 100).Draw(t, "order")
			desc := "d" + string(rune('a'+i)) // da, db, dc, etc.
			binding := key.NewBinding(key.WithKeys(string(rune('a'+i))), key.WithHelp(string(rune('a'+i)), desc))
			bindings[i] = HelpBinding{
				Binding:  binding,
				Category: CategoryNavigation,
				Order:    orders[i],
			}
		}

		sb := NewStatusBar("v1.0.0")
		sb.SetBindings(bindings)
		sb.SetWidth(width)

		view := sb.View()

		// Find positions of each description in the view
		// Verify they appear in order (by Order field, not by index)
		type orderPos struct {
			order int
			pos   int
		}
		var found []orderPos
		for i := 0; i < numBindings; i++ {
			desc := "d" + string(rune('a'+i))
			pos := strings.Index(view, desc)
			if pos >= 0 {
				found = append(found, orderPos{orders[i], pos})
			}
		}

		// Verify positions increase with order
		for i := 1; i < len(found); i++ {
			if found[i-1].order < found[i].order && found[i-1].pos > found[i].pos {
				t.Errorf("binding with order %d appears after binding with order %d", found[i-1].order, found[i].order)
			}
		}
	})
}

func TestStatusBar_SeparatorBetweenMultipleBindings(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(100, 300).Draw(t, "width")

		// Create exactly 2 enabled bindings
		bindings := []HelpBinding{
			{
				Binding:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "first")),
				Category: CategoryNavigation,
				Order:    1,
			},
			{
				Binding:  key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "second")),
				Category: CategoryNavigation,
				Order:    2,
			},
		}

		sb := NewStatusBar("v1.0.0")
		sb.SetBindings(bindings)
		sb.SetWidth(width)

		view := sb.View()

		// If both appear, there should be a separator between them
		if strings.Contains(view, "first") && strings.Contains(view, "second") {
			if !strings.Contains(view, "•") {
				t.Errorf("expected separator between bindings: %q", view)
			}
		}
	})
}

func TestStatusBar_EmptyBindingsShowsVersion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(20, 200).Draw(t, "width")
		version := rapid.StringMatching(`v[0-9]+\.[0-9]+\.[0-9]+`).Draw(t, "version")

		sb := NewStatusBar(version)
		sb.SetBindings(nil)
		sb.SetWidth(width)

		view := sb.View()

		if !strings.Contains(view, version) {
			t.Errorf("version %q not found in view with no bindings: %q", version, view)
		}

		// Should not have ellipsis
		if strings.Contains(view, "…") {
			t.Errorf("unexpected ellipsis with no bindings: %q", view)
		}
	})
}

func TestStatusBar_PinnedBindingsAlwaysAppear(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(40, 100).Draw(t, "width")

		// Create many regular bindings that will get truncated
		numRegular := rapid.IntRange(5, 15).Draw(t, "numRegular")
		bindings := make([]HelpBinding, numRegular+1)
		for i := 0; i < numRegular; i++ {
			bindings[i] = HelpBinding{
				Binding:  key.NewBinding(key.WithKeys(string(rune('a'+i))), key.WithHelp(string(rune('a'+i)), "action"+string(rune('0'+i)))),
				Category: CategoryNavigation,
				Order:    i,
				Pinned:   false,
			}
		}

		// Add one pinned binding
		bindings[numRegular] = HelpBinding{
			Binding:  key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
			Category: CategoryActions,
			Order:    99,
			Pinned:   true,
		}

		sb := NewStatusBar("v1.0.0")
		sb.SetBindings(bindings)
		sb.SetWidth(width)

		view := sb.View()

		// Pinned binding should always appear
		if !strings.Contains(view, "help") {
			t.Errorf("pinned binding 'help' should always appear: %q", view)
		}
	})
}
