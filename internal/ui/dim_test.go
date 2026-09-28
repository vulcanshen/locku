package ui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
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

// dimmed is c faded as D2 has it.
func dimmed(c lipgloss.Color) lipgloss.Color {
	d := dimRGB(hexRGB(string(c)))
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

// D2: c × 0.45 + base × 0.55, and never lighter than c.
func TestDimRGB(t *testing.T) {
	if got := dimRGB([3]int{0x89, 0xb4, 0xfa}); got != [3]int{78, 98, 138} {
		t.Errorf("blue: %v", got)
	}
	// Black is darker than the base: it stays black.
	if got := dimRGB([3]int{0, 0, 0}); got != [3]int{0, 0, 0} {
		t.Errorf("black must not lighten: %v", got)
	}
	// A background fades as a foreground does; a reset leaves the dimmed
	// text colour behind it.
	if got := dimSGR("48;2;137;180;250"); got != "48;2;78;98;138" {
		t.Errorf("background: %q", got)
	}
	if got := dimSGR("0"); got != "0;"+sgrRGB(true, dimText) {
		t.Errorf("reset: %q", got)
	}
}

// With a popup up, everything below the topmost fades — every colour its
// own way, backgrounds too, so the capsules, the cursor bars and a draft's
// yellow unsaved chip are still there, only darker; a popup under it
// fades frame and all, its frame its layer colour dimmed; the top is
// itself; a toast fades nothing (tdp F8, D2, 2026-09-28).
func TestPopupsDimBelowTheTop(t *testing.T) {
	colours(t)
	m := newTestApp(t)
	if l := line(m.View(), "[1] locku"); !has(l, focusColor) || !hasBG(l, focusColor) {
		t.Fatalf("no popup: the focused panel's frame and capsule are bright: %q", l)
	}

	// The Space menu is the top: the panels fade — the [1] capsule's
	// background, the cursor bar's — the menu does not.
	// (On clock2: the active profile cannot be deleted, and X below is to
	// open a confirm.)
	m = m.press("j", " ")
	v := m.View()
	if l := line(v, "[1] locku"); hasBG(l, focusColor) || !hasBG(l, dimmed(focusColor)) {
		t.Errorf("under the menu the [1] capsule must fade, not go: %q", l)
	}
	if l := line(v, "║   clock2"); hasBG(l, handColor) || !hasBG(l, dimmed(handColor)) {
		t.Errorf("under the menu the cursor bar must fade, not go: %q", l)
	}
	menuFrame := popupLayerColor(1)
	if l := line(v, "Global operation"); !has(l, menuFrame) {
		t.Errorf("the menu on top must keep its colour: %q", l)
	}

	// A confirm over it is the top: the menu fades, frame and all, its
	// frame its own colour dimmed, its cursor bar still there.
	m = m.press("X")
	v = m.View()
	if l := line(v, "Global operation"); has(l, menuFrame) || !has(l, dimmed(menuFrame)) {
		t.Errorf("under the confirm the menu's frame must be its colour dimmed: %q", l)
	}
	if l := line(v, "item operation"); l != "" && hasBG(l, handColor) {
		t.Errorf("under the confirm nothing of the menu is bright: %q", l)
	}
	if l := line(v, "Delete"); !has(l, popupLayerColor(2)) {
		t.Errorf("the confirm on top must keep its colour: %q", l)
	}

	// Esc: the menu is the top again the moment the confirm starts to go,
	// not once it has gone — as the keys are the menu's then (tdp D3).
	m, _ = m.sends("esc")
	if !m.confirm.isActive() {
		t.Fatal("the confirm must still be closing")
	}
	if l := line(m.View(), "Global operation"); !has(l, menuFrame) {
		t.Errorf("after Esc the menu must be bright again: %q", l)
	}
	m = m.settle()

	// A toast alone fades nothing.
	m = m.press("esc")
	m.toast.show("PIN set", toastInfo)
	m = m.settle()
	if l := line(m.View(), "[1] locku"); !has(l, focusColor) {
		t.Errorf("a toast must not fade the screen: %q", l)
	}
}

// A colour draft's yellow unsaved chip fades as itself, under a popup.
func TestDimKeepsTheUnsavedChip(t *testing.T) {
	colours(t)
	m := newTestApp(t).press("2")
	d := m.cfg.Profiles[0].Colours()
	d.BG = "#000000"
	m.drafts[profileKey(m.cfg.Profiles[0].Name)] = d
	if l := line(m.View(), "unsaved"); !hasBG(l, yellowColor) {
		t.Fatalf("no popup: the unsaved chip is yellow: %q", l)
	}
	m = m.press(" ")
	if l := line(m.View(), "unsaved"); hasBG(l, yellowColor) || !hasBG(l, dimmed(yellowColor)) {
		t.Errorf("under the menu the unsaved chip must fade, not go: %q", l)
	}
}
