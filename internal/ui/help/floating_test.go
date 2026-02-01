package help

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"pgregory.net/rapid"
)

func generateFloatingBindings(t *rapid.T) []HelpBinding {
	numBindings := rapid.IntRange(0, 30).Draw(t, "numBindings")
	bindings := make([]HelpBinding, numBindings)
	for i := 0; i < numBindings; i++ {
		keyStr := string(rune('a' + i%26))
		desc := rapid.StringMatching(`[a-z]{3,12}`).Draw(t, "desc")
		category := rapid.SampledFrom([]Category{CategoryNavigation, CategoryActions, CategoryDiff}).Draw(t, "category")
		enabled := rapid.Float64Range(0, 1).Draw(t, "enabledChance") > 0.2 // 80% enabled

		binding := key.NewBinding(key.WithKeys(keyStr), key.WithHelp(keyStr, desc))
		if !enabled {
			binding.SetEnabled(false)
		}

		bindings[i] = HelpBinding{
			Binding:  binding,
			Category: category,
			Order:    i,
		}
	}
	return bindings
}

func TestFloating_AllEnabledBindingsAppear_WhenEnoughSpace(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(60, 120).Draw(t, "width")

		// Create a small number of bindings so they all fit
		numBindings := rapid.IntRange(1, 5).Draw(t, "numBindings")
		bindings := make([]HelpBinding, numBindings)
		for i := 0; i < numBindings; i++ {
			keyStr := string(rune('a' + i))
			desc := "desc" + string(rune('0'+i))
			bindings[i] = HelpBinding{
				Binding:  key.NewBinding(key.WithKeys(keyStr), key.WithHelp(keyStr, desc)),
				Category: CategoryNavigation,
				Order:    i,
			}
		}

		// Ensure enough height for all bindings + header + footer + border
		height := numBindings + 10

		fh := NewFloatingHelp()
		fh.SetSize(width, height)
		fh.SetBindings(bindings)

		view := fh.View()

		for _, hb := range bindings {
			desc := hb.Binding.Help().Desc
			if !strings.Contains(view, desc) {
				t.Errorf("enabled binding %q not found in view with sufficient space", desc)
			}
		}
	})
}

func TestFloating_DisabledBindingsNeverAppear(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(60, 120).Draw(t, "width")
		height := rapid.IntRange(20, 40).Draw(t, "height")

		// Create only disabled bindings with unique descriptions
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

		fh := NewFloatingHelp()
		fh.SetSize(width, height)
		fh.SetBindings(bindings)

		view := fh.View()

		for i := 0; i < numBindings; i++ {
			desc := "disabled" + string(rune('0'+i))
			if strings.Contains(view, desc) {
				t.Errorf("disabled binding %q should not appear in view", desc)
			}
		}
	})
}

func TestFloating_CategoriesAppearAsHeaders(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(60, 120).Draw(t, "width")
		height := rapid.IntRange(20, 40).Draw(t, "height")

		// Create bindings in each category
		bindings := []HelpBinding{
			{
				Binding:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "nav action")),
				Category: CategoryNavigation,
				Order:    1,
			},
			{
				Binding:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "action action")),
				Category: CategoryActions,
				Order:    2,
			},
			{
				Binding:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "diff action")),
				Category: CategoryDiff,
				Order:    3,
			},
		}

		fh := NewFloatingHelp()
		fh.SetSize(width, height)
		fh.SetBindings(bindings)

		view := fh.View()

		// Each category with bindings should appear as a header
		if !strings.Contains(view, string(CategoryNavigation)) {
			t.Errorf("Navigation category header not found")
		}
		if !strings.Contains(view, string(CategoryActions)) {
			t.Errorf("Actions category header not found")
		}
		if !strings.Contains(view, string(CategoryDiff)) {
			t.Errorf("Diff category header not found")
		}
	})
}

func TestFloating_BindingsGroupedByCategory(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(80, 120).Draw(t, "width")
		height := rapid.IntRange(30, 50).Draw(t, "height")

		// Create multiple bindings per category
		bindings := []HelpBinding{
			{Binding: key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "nav1")), Category: CategoryNavigation, Order: 1},
			{Binding: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "nav2")), Category: CategoryNavigation, Order: 2},
			{Binding: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "act1")), Category: CategoryActions, Order: 3},
			{Binding: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "act2")), Category: CategoryActions, Order: 4},
		}

		fh := NewFloatingHelp()
		fh.SetSize(width, height)
		fh.SetBindings(bindings)

		view := fh.View()

		// Within the same category, bindings should appear together
		// Check that nav1 and nav2 are closer to each other than to act1/act2
		nav1Pos := strings.Index(view, "nav1")
		nav2Pos := strings.Index(view, "nav2")
		act1Pos := strings.Index(view, "act1")

		if nav1Pos >= 0 && nav2Pos >= 0 && act1Pos >= 0 {
			navDist := abs(nav2Pos - nav1Pos)
			crossDist := abs(act1Pos - nav1Pos)
			if navDist > crossDist {
				t.Errorf("bindings in same category should be grouped together")
			}
		}
	})
}

func TestFloating_SizeConstraintsRespected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(40, 120).Draw(t, "width")
		height := rapid.IntRange(10, 40).Draw(t, "height")
		bindings := generateFloatingBindings(t)

		fh := NewFloatingHelp()
		fh.SetSize(width, height)
		fh.SetBindings(bindings)

		view := fh.View()

		viewWidth := lipgloss.Width(view)
		viewHeight := lipgloss.Height(view)

		if viewWidth > width {
			t.Errorf("view width %d exceeds specified width %d", viewWidth, width)
		}
		if viewHeight > height {
			t.Errorf("view height %d exceeds specified height %d", viewHeight, height)
		}
	})
}

func TestFloating_EmptyBindingsShowsEmptyModal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(40, 120).Draw(t, "width")
		height := rapid.IntRange(10, 40).Draw(t, "height")

		fh := NewFloatingHelp()
		fh.SetSize(width, height)
		fh.SetBindings(nil)

		view := fh.View()

		// Should still render something (border, title)
		if len(view) == 0 {
			t.Errorf("empty bindings should still render modal frame")
		}
	})
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
