package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelHasSevenNamedStages(t *testing.T) {
	m := New("9.9.9")
	want := []string{"Discover", "Connect", "Back Up", "Safeguards", "Identity", "Install", "Verify"}
	if len(m.stages) != len(want) {
		t.Fatalf("got %d stages, want %d", len(m.stages), len(want))
	}
	for i, s := range m.stages {
		if s.name != want[i] {
			t.Errorf("stage %d = %q, want %q", i, s.name, want[i])
		}
		if s.sub == "" || s.run == nil {
			t.Errorf("stage %q has empty sub or nil run", s.name)
		}
	}
}

// Every stage renderer must produce non-empty output without panicking — they
// call the real internal packages, so this doubles as an integration smoke test.
func TestAllRenderersProduceOutput(t *testing.T) {
	m := New("1.2.3")
	for _, s := range m.stages {
		out := s.run()
		if strings.TrimSpace(out) == "" {
			t.Errorf("renderer for %q produced empty output", s.name)
		}
	}
}

func sizedModel(t *testing.T) model {
	t.Helper()
	m := New("0.4.0")
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return nm.(model)
}

func TestViewContainsHeaderAndSelectedContent(t *testing.T) {
	m := sizedModel(t)
	v := m.View()
	for _, want := range []string{"swallow", "v0.4.0", "STEPS", "Discover", "Step 1 of 7"} {
		if !strings.Contains(v, want) {
			t.Errorf("View() missing %q", want)
		}
	}
}

func TestNavigationBounds(t *testing.T) {
	m := sizedModel(t)

	// Up at the top is a no-op.
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if nm.(model).sel != 0 {
		t.Errorf("up at top moved selection to %d", nm.(model).sel)
	}

	// Down advances.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if nm.(model).sel != 1 {
		t.Errorf("down: sel=%d want 1", nm.(model).sel)
	}

	// G jumps to last; down past the end is a no-op.
	m = sizedModel(t)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	last := len(m.stages) - 1
	if nm.(model).sel != last {
		t.Fatalf("G: sel=%d want %d", nm.(model).sel, last)
	}
	nm, _ = nm.(model).Update(tea.KeyMsg{Type: tea.KeyDown})
	if nm.(model).sel != last {
		t.Errorf("down at bottom moved to %d", nm.(model).sel)
	}

	// g jumps back to first.
	nm, _ = nm.(model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if nm.(model).sel != 0 {
		t.Errorf("g: sel=%d want 0", nm.(model).sel)
	}

	// The last stage renders its own content.
	m = sizedModel(t)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if v := nm.(model).View(); !strings.Contains(v, "Verify") || !strings.Contains(v, "Step 7 of 7") {
		t.Errorf("last-stage view missing Verify/breadcrumb")
	}
}

func TestQuitKeys(t *testing.T) {
	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("q")},
		{Type: tea.KeyCtrlC},
		{Type: tea.KeyEsc},
	} {
		m := sizedModel(t)
		_, cmd := m.Update(k)
		if cmd == nil {
			t.Errorf("key %v did not return a command (expected Quit)", k)
			continue
		}
		if msg := cmd(); msg == nil {
			t.Errorf("key %v command produced nil msg", k)
		}
	}
}

func TestViewSurvivesTinyTerminal(t *testing.T) {
	m := New("0.4.0")
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 8})
	// Must not panic and must still render something.
	if v := nm.(model).View(); strings.TrimSpace(v) == "" {
		t.Error("tiny terminal produced empty view")
	}
}
