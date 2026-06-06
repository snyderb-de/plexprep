package ui

import "github.com/charmbracelet/lipgloss"

// Palette — neon/charm vibes.
var (
	cPink   = lipgloss.Color("#FF6AC1")
	cPurple = lipgloss.Color("#A682FF")
	cCyan   = lipgloss.Color("#5EF6FF")
	cGreen  = lipgloss.Color("#6EE7A0")
	cYellow = lipgloss.Color("#FFD866")
	cRed    = lipgloss.Color("#FF6B6B")
	cDim    = lipgloss.Color("#6C7086")
	cFg     = lipgloss.Color("#E6E6F0")
	cBg     = lipgloss.Color("#1A1B26")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).Foreground(cBg).Background(cPink).
			Padding(0, 2).MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().Foreground(cPurple).Italic(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(cPurple).
			Padding(1, 2)

	helpStyle = lipgloss.NewStyle().Foreground(cDim)

	selectedRow = lipgloss.NewStyle().Foreground(cCyan).Bold(true)
	normalRow   = lipgloss.NewStyle().Foreground(cFg)
	dimRow      = lipgloss.NewStyle().Foreground(cDim)

	savingsStyle = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	growStyle    = lipgloss.NewStyle().Foreground(cRed).Bold(true)
	copyStyle    = lipgloss.NewStyle().Foreground(cYellow)
	badge4K      = lipgloss.NewStyle().Foreground(cBg).Background(cCyan).Bold(true).Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(cDim).PaddingTop(1).MarginTop(1)

	checkOn  = lipgloss.NewStyle().Foreground(cGreen).Render("◉")
	checkOff = lipgloss.NewStyle().Foreground(cDim).Render("○")

	okStyle  = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	errStyle = lipgloss.NewStyle().Foreground(cRed).Bold(true)
)

// gradientText paints s across a pink→purple→cyan gradient, char by char.
func gradientText(s string) string {
	stops := []lipgloss.Color{cPink, cPurple, cCyan}
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	out := ""
	for i, ch := range r {
		t := float64(i) / float64(max(1, len(r)-1))
		c := lerpStops(stops, t)
		out += lipgloss.NewStyle().Foreground(c).Render(string(ch))
	}
	return out
}

func lerpStops(stops []lipgloss.Color, t float64) lipgloss.Color {
	if t <= 0 {
		return stops[0]
	}
	if t >= 1 {
		return stops[len(stops)-1]
	}
	seg := t * float64(len(stops)-1)
	i := int(seg)
	return lerp(stops[i], stops[i+1], seg-float64(i))
}

func lerp(a, b lipgloss.Color, t float64) lipgloss.Color {
	ar, ag, ab := hexRGB(string(a))
	br, bg, bb := hexRGB(string(b))
	r := int(float64(ar) + (float64(br)-float64(ar))*t)
	g := int(float64(ag) + (float64(bg)-float64(ag))*t)
	bl := int(float64(ab) + (float64(bb)-float64(ab))*t)
	return lipgloss.Color(rgbHex(r, g, bl))
}

func hexRGB(h string) (int, int, int) {
	if len(h) == 7 && h[0] == '#' {
		var r, g, b int
		_, _ = sscanHex(h[1:3], &r), sscanHex(h[3:5], &g)
		_ = sscanHex(h[5:7], &b)
		return r, g, b
	}
	return 255, 255, 255
}

func sscanHex(s string, v *int) error {
	n := 0
	for _, c := range s {
		n <<= 4
		switch {
		case c >= '0' && c <= '9':
			n |= int(c - '0')
		case c >= 'a' && c <= 'f':
			n |= int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			n |= int(c-'A') + 10
		}
	}
	*v = n
	return nil
}

func rgbHex(r, g, b int) string {
	clamp := func(x int) int {
		if x < 0 {
			return 0
		}
		if x > 255 {
			return 255
		}
		return x
	}
	const hexd = "0123456789abcdef"
	r, g, b = clamp(r), clamp(g), clamp(b)
	return string([]byte{'#',
		hexd[r>>4], hexd[r&0xf],
		hexd[g>>4], hexd[g&0xf],
		hexd[b>>4], hexd[b&0xf]})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
