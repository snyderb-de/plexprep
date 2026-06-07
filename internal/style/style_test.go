package style

import (
	"strings"
	"testing"
)

func init() { Enabled = false } // deterministic, no ANSI in test output

func TestTier(t *testing.T) {
	cases := []struct {
		pct  float64
		want Color
	}{
		{5, Amber}, {14.9, Amber}, {15, Blue}, {34.9, Blue}, {35, Green}, {90, Green},
	}
	for _, c := range cases {
		if got := Tier(c.pct); got != c.want {
			t.Errorf("Tier(%v) = %v, want %v", c.pct, got, c.want)
		}
	}
}

func TestBarFill(t *testing.T) {
	// 50% of width 10 → 5 filled blocks.
	got := Bar(50, 10)
	if n := strings.Count(got, "█"); n != 5 {
		t.Errorf("Bar(50,10) filled=%d, want 5 (%q)", n, got)
	}
	if n := strings.Count(got, "░"); n != 5 {
		t.Errorf("Bar(50,10) empty=%d, want 5", n)
	}
	// Clamp.
	if n := strings.Count(Bar(250, 10), "█"); n != 10 {
		t.Errorf("Bar over 100%% should clamp to full, got %d", n)
	}
	if n := strings.Count(Bar(-5, 10), "█"); n != 0 {
		t.Errorf("Bar negative should clamp to empty, got %d", n)
	}
}

func TestPad(t *testing.T) {
	if got := Pad("ab", 5); got != "ab   " {
		t.Errorf("Pad = %q", got)
	}
	if got := PadL("ab", 5); got != "   ab" {
		t.Errorf("PadL = %q", got)
	}
}

func TestTrunc(t *testing.T) {
	if got := Trunc("hello", 10); got != "hello" {
		t.Errorf("Trunc no-op = %q", got)
	}
	if got := Trunc("hello world", 5); got != "hell…" {
		t.Errorf("Trunc = %q", got)
	}
}

func TestFrameContainsTitleAndBorders(t *testing.T) {
	out := Frame("HELLO", []string{"line one", "two"})
	for _, want := range []string{"HELLO", "line one", "┌", "└", "│"} {
		if !strings.Contains(out, want) {
			t.Errorf("frame missing %q in:\n%s", want, out)
		}
	}
}
