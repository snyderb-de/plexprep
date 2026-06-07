// Package style renders the plexprep "phosphor terminal" look for console
// output: 24-bit ANSI colors, box-drawing frames, and ASCII tier bars. Color is
// auto-disabled when stdout is not a terminal (piped/redirected) or NO_COLOR is
// set; box-drawing characters are always emitted.
package style

import (
	"fmt"
	"os"
	"strings"
)

// Enabled reports whether ANSI color is emitted.
var Enabled = colorEnabled()

func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// Palette (matches the HTML report).
var (
	Green  = rgb(58, 217, 104)  // phosphor
	Bright = rgb(168, 255, 196) // bright green
	Mid    = rgb(42, 135, 75)   // dim green
	Dim    = rgb(31, 107, 58)   // dimmer green
	Amber  = rgb(255, 204, 51)
	Blue   = rgb(77, 181, 255)
	Red    = rgb(255, 95, 86)
)

type Color struct{ r, g, b int }

func rgb(r, g, b int) Color { return Color{r, g, b} }

// S paints s with the color (no-op when color disabled).
func (c Color) S(s string) string {
	if !Enabled {
		return s
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", c.r, c.g, c.b, s)
}

// B paints s bold + colored.
func (c Color) B(s string) string {
	if !Enabled {
		return s
	}
	return fmt.Sprintf("\x1b[1;38;2;%d;%d;%dm%s\x1b[0m", c.r, c.g, c.b, s)
}

// Bar renders a fixed-width [██████░░░░] meter, tier-colored by percent.
func Bar(pct float64, width int) string {
	p := pct
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	filled := int(p/100*float64(width) + 0.5)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return Tier(pct).S(bar)
}

// ProgBar renders a fixed-width progress meter in phosphor green (not tiered).
func ProgBar(pct float64, width int) string {
	p := pct
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	filled := int(p/100*float64(width) + 0.5)
	return Green.S(strings.Repeat("█", filled)) + Dim.S(strings.Repeat("░", width-filled))
}

// Tier maps a saving percent to its color: <15 amber, <35 blue, else green.
func Tier(pct float64) Color {
	switch {
	case pct >= 35:
		return Green
	case pct >= 15:
		return Blue
	default:
		return Amber
	}
}

// visLen is the visible length of s (ANSI escape sequences excluded).
func visLen(s string) int {
	n, inEsc := 0, false
	for _, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == '\x1b' {
			inEsc = true
			continue
		}
		n++
	}
	return n
}

// Frame wraps lines in a box-drawing frame with a titled top border.
// The border is dim green, the title bright.
func Frame(title string, lines []string) string {
	inner := visLen(title) + 4
	for _, l := range lines {
		if w := visLen(l); w > inner {
			inner = w
		}
	}
	inner += 2 // padding

	var b strings.Builder
	// top: ┌─ TITLE ───────┐
	top := "┌─ " + Bright.B(title) + " "
	dashCount := inner - (visLen(title) + 3)
	b.WriteString(Dim.S("┌─ ") + Bright.B(title) + " " + Dim.S(strings.Repeat("─", max(0, dashCount))+"┐") + "\n")
	_ = top

	for _, l := range lines {
		pad := inner - visLen(l) - 1
		b.WriteString(Dim.S("│") + " " + l + strings.Repeat(" ", max(0, pad)) + Dim.S("│") + "\n")
	}
	b.WriteString(Dim.S("└" + strings.Repeat("─", inner) + "┘"))
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Pad left-aligns s to width w (visible length aware, truncates if longer).
func Pad(s string, w int) string {
	vl := visLen(s)
	if vl > w {
		return Trunc(s, w)
	}
	return s + strings.Repeat(" ", w-vl)
}

// PadL right-aligns s to width w (visible length aware).
func PadL(s string, w int) string {
	vl := visLen(s)
	if vl >= w {
		return s
	}
	return strings.Repeat(" ", w-vl) + s
}

// Trunc cuts a plain string to w runes with an ellipsis (no ANSI awareness;
// use only on uncolored strings).
func Trunc(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return string(r[:w])
	}
	return string(r[:w-1]) + "…"
}

// Rule returns a dim horizontal rule of width w.
func Rule(w int) string { return Dim.S(strings.Repeat("─", w)) }
