package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	terminu "github.com/vulcanshen/terminu-go"
)

// withIcons draws with icons n cells wide, as DetectIconWidth would set
// it, for the rest of the test.
func withIcons(t *testing.T, n int) {
	t.Helper()
	icon, powerline := terminu.IconWidth(), terminu.PowerlineWidth()
	terminu.SetIconWidth(n, powerline)
	t.Cleanup(func() { terminu.SetIconWidth(icon, powerline) })
}

// The lock's PIN prompt alone, and the width a custom saver is handed
// with it: the icon in its title counted (tdp D6).
func TestPromptBoxWithIcons(t *testing.T) {
	withIcons(t, 2)
	m := openPrompt(t, testLock(t, "1234", nil))
	box := strings.Split(boxText(m.prompt, m.now()), "\n")
	want := pinBoxW
	for r, l := range box {
		if dispW(l) != want {
			t.Errorf("row %d is %d wide, want %d: %q", r, dispW(l), want, ansi.Strip(l))
		}
	}
	// On the board it sits in the middle, as with icons one cell: every
	// row being as wide is not enough — a row a column off to the right
	// is as wide too.
	at := (80 - want) / 2
	for _, r := range []string{"┏", "┃", "┗"} {
		if l := line(m.View(), r); dispW(l[:strings.Index(l, r)]) != at {
			t.Errorf("the box's %s is at %d, want %d", r, dispW(l[:strings.Index(l, r)]), at)
		}
	}
	cfg := testLock(t, "1234", nil).cfg
	var widths []int
	p := NewLockPrompt(cfg, "", 100, 30, func(_ string, w int) { widths = append(widths, w) }, Tries{})
	var pm tea.Model = p
	for i := 0; i < openFrames+1; i++ {
		pm, _ = pm.(LockModel).Update(AnimTickMsg{})
	}
	if len(widths) == 0 || widths[len(widths)-1] != pinBoxW {
		t.Errorf("the box's width for the custom saver: %v, want %d", widths, pinBoxW)
	}
}

// On a lock a few rows high the PIN prompt is no higher than the screen
// less two (tdp F7), its top border on the screen, and nothing of it past
// the screen's edges, on the board and over a custom saver's program alike:
// handed on whole, the rows past the bottom would land on the last one
// (tdp E5).
func TestPromptOnATinyScreen(t *testing.T) {
	for _, sz := range [][2]int{{20, 5}, {11, 4}} {
		m := testLock(t, "1234", nil)
		m, _ = m.step(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		m = openPrompt(t, m)
		if v := m.View(); !strings.Contains(v, "┏") || len(strings.Split(v, "\n")) != sz[1] {
			t.Errorf("%v on the board:\n%s", sz, ansi.Strip(v))
		}
		var box string
		var w int
		var pm tea.Model = NewLockPrompt(m.cfg, "", sz[0], sz[1], func(b string, bw int) { box, w = b, bw }, Tries{})
		for i := 0; i < openFrames+1; i++ {
			pm, _ = pm.(LockModel).Update(AnimTickMsg{})
		}
		lines := strings.Split(box, "\n")
		if len(lines) > sz[1] || !strings.Contains(lines[0], "┏") || w > sz[0] {
			t.Errorf("%v over a custom saver: %d rows, %d wide, the top %q", sz, len(lines), w, ansi.Strip(lines[0]))
		}
		for r, l := range lines {
			if dispW(l) > sz[0] {
				t.Errorf("%v over a custom saver: row %d is %d wide", sz, r, dispW(l))
			}
		}
	}
}
