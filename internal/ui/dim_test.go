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

var fgRe = regexp.MustCompile(`(?:^|[;\[])38;2;(\d+);(\d+);(\d+)`)

// has reports whether l sets the foreground to c somewhere — within two
// steps a channel, which is how far lipgloss rounds (#89b4fa comes out
// as 137;179;250).
func has(l string, c lipgloss.Color) bool {
	var r, g, b int
	fmt.Sscanf(string(c), "#%02x%02x%02x", &r, &g, &b)
	near := func(a, b int) bool { return a-b <= 2 && b-a <= 2 }
	for _, m := range fgRe.FindAllStringSubmatch(l, -1) {
		var x, y, z int
		fmt.Sscanf(m[1]+" "+m[2]+" "+m[3], "%d %d %d", &x, &y, &z)
		if near(x, r) && near(y, g) && near(z, b) {
			return true
		}
	}
	return false
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

// With a popup up, everything below the topmost is dimmed — the panels,
// and a popup under it, whose frame keeps its layer in a dimmed version
// of its colour; the top is itself; a toast dims nothing (tdp F8, v0.1.9,
// 2026-09-28).
func TestPopupsDimBelowTheTop(t *testing.T) {
	colours(t)
	m := newTestApp(t)
	if l := line(m.View(), "[1] locku"); !has(l, focusColor) {
		t.Fatalf("no popup: the focused panel's frame is bright: %q", l)
	}

	// The Space menu is the top: the panels go dim, the menu does not.
	// (On clock2: the active profile cannot be deleted, and X below is
	// to open a confirm.)
	m = m.press("j", " ")
	v := m.View()
	if l := line(v, "Profiles"); has(l, focusColor) || !has(l, dimColor) {
		t.Errorf("under the menu the panels must be dim: %q", l)
	}
	menuFrame := popupLayerColor(1)
	if l := line(v, "Global operation"); !has(l, menuFrame) {
		t.Errorf("the menu on top must keep its colour: %q", l)
	}

	// A confirm over it is the top: the menu goes dim, frame and all, its
	// frame in its own colour dimmed.
	m = m.press("X")
	v = m.View()
	l := line(v, "Global operation")
	if has(l, menuFrame) || !has(l, dimOf(menuFrame)) || !has(l, dimColor) {
		t.Errorf("under the confirm the menu must be dim, its frame its colour dimmed: %q", l)
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

	// A toast alone dims nothing.
	m = m.press("esc")
	m.toast.show("PIN set", toastInfo)
	m = m.settle()
	if l := line(m.View(), "[1] locku"); !has(l, focusColor) {
		t.Errorf("a toast must not dim the screen: %q", l)
	}
}
