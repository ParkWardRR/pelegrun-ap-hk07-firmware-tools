package style

import (
	"strings"
	"testing"
)

func TestColorWrappingAndReset(t *testing.T) {
	on = true // force color regardless of the test env's NO_COLOR
	fns := []struct {
		name string
		fn   func(string) string
		code string
	}{
		{"Bold", Bold, "1"},
		{"Dim", Dim, "2"},
		{"Red", Red, "31"},
		{"Green", Green, "32"},
		{"Yellow", Yellow, "33"},
		{"Cyan", Cyan, "36"},
	}
	for _, c := range fns {
		got := c.fn("x")
		want := "\x1b[" + c.code + "mx\x1b[0m"
		if got != want {
			t.Errorf("%s = %q, want %q", c.name, got, want)
		}
		if !strings.HasSuffix(got, "\x1b[0m") {
			t.Errorf("%s did not reset", c.name)
		}
	}
}

func TestNoColorPassesThrough(t *testing.T) {
	on = false // NO_COLOR behavior
	for _, fn := range []func(string) string{Bold, Dim, Red, Green, Yellow, Cyan} {
		if got := fn("plain"); got != "plain" {
			t.Errorf("with color off, got %q, want %q", got, "plain")
		}
	}
	on = true
}
