package ui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	terminu "github.com/vulcanshen/terminu-go"
)

// colours on: a test's output is no terminal, and lipgloss would draw
// none.
func colours(t *testing.T) {
	was := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(was) })
}

var (
	fgRe = regexp.MustCompile(`(?:^|[;\[])38;2;(\d+);(\d+);(\d+)`)
	bgRe = regexp.MustCompile(`(?:^|[;\[])48;2;(\d+);(\d+);(\d+)`)
)

// near reports whether re finds c in l — within two steps a channel,
// which is how far lipgloss rounds (#89b4fa comes out as 137;179;250).
func near(re *regexp.Regexp, l string, c [3]int) bool {
	close := func(a, b int) bool { return a-b <= 2 && b-a <= 2 }
	for _, m := range re.FindAllStringSubmatch(l, -1) {
		var x, y, z int
		fmt.Sscanf(m[1]+" "+m[2]+" "+m[3], "%d %d %d", &x, &y, &z)
		if close(x, c[0]) && close(y, c[1]) && close(z, c[2]) {
			return true
		}
	}
	return false
}

// has: l sets the foreground to c somewhere; hasBG: the background.
func has(l string, c lipgloss.Color) bool   { return near(fgRe, l, hexRGB(string(c))) }
func hasBG(l string, c lipgloss.Color) bool { return near(bgRe, l, hexRGB(string(c))) }

// dimmed is c faded as components/color has it.
func dimmed(c lipgloss.Color) lipgloss.Color {
	h := hexRGB(string(c))
	d := terminu.DimRGB(terminu.RGB{uint8(h[0]), uint8(h[1]), uint8(h[2])})
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", d[0], d[1], d[2]))
}

// line is the first line of v that has want in it, colours and all.
func line(v, want string) string {
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(ansi.Strip(l), want) {
			return l
		}
	}
	return ""
}
