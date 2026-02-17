package help

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"pgregory.net/rapid"
)

func TestStatusBar_WidthNeverExceeded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(20, 200).Draw(t, "width")

		sb := NewStatusBar("v1.0.0")
		sb.SetWidth(width)

		view := sb.View()
		viewWidth := lipgloss.Width(view)

		if viewWidth > width {
			t.Errorf("view width %d exceeds specified width %d: %q", viewWidth, width, view)
		}
	})
}

func TestStatusBar_VersionPresentWhenRoomAvailable(t *testing.T) {
	sb := NewStatusBar("v1.0.0")
	sb.SetWidth(80)

	view := sb.View()

	if !strings.Contains(view, "v1.0.0") {
		t.Errorf("version should appear at width 80: %q", view)
	}

	if !strings.HasSuffix(strings.TrimSpace(view), "v1.0.0") {
		t.Errorf("version should be right-aligned: %q", view)
	}
}

func TestStatusBar_VersionDroppedWhenNarrow(t *testing.T) {
	sb := NewStatusBar("v1.0.0")
	sb.SetWidth(20)

	view := sb.View()

	if strings.Contains(view, "v1.0.0") {
		t.Errorf("version should be dropped at narrow width: %q", view)
	}

	// Key hints should still be present
	if !strings.Contains(view, "help") {
		t.Errorf("help hint should still appear: %q", view)
	}
}

func TestStatusBar_ContainsHelpAndQuit(t *testing.T) {
	sb := NewStatusBar("v1.0.0")
	sb.SetWidth(80)

	view := sb.View()

	if !strings.Contains(view, "help") {
		t.Errorf("expected 'help' in view: %q", view)
	}

	if !strings.Contains(view, "quit") {
		t.Errorf("expected 'quit' in view: %q", view)
	}
}

func TestStatusBar_ZeroWidthReturnsEmpty(t *testing.T) {
	sb := NewStatusBar("v1.0.0")
	sb.SetWidth(0)

	if view := sb.View(); view != "" {
		t.Errorf("expected empty view for zero width, got: %q", view)
	}
}
