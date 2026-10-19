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
	withWidths(t, n, terminu.PowerlineWidth())
}

// withWidths draws with icons and powerline symbols as wide as given, for
// the rest of the test (tdp E5: the two are measured apart).
func withWidths(t *testing.T, icon, powerline int) {
	t.Helper()
	wasIcon, wasPowerline := terminu.IconWidth(), terminu.PowerlineWidth()
	terminu.SetIconWidth(icon, powerline)
	t.Cleanup(func() { terminu.SetIconWidth(wasIcon, wasPowerline) })
}

// A box opened on a screen larger than the one it then lands on — the
// terminal made smaller under it — is cut at the screen's edges, never past
// them and never a panic: wider, higher and both (tdp E5).
func TestPromptLargerThanTheScreen(t *testing.T) {
	for _, w := range [][2]int{{1, 1}, {2, 2}} {
		withWidths(t, w[0], w[1])
		for _, sz := range [][2]int{{30, 24}, {80, 5}, {12, 4}, {1, 1}} {
			m := openPrompt(t, testLock(t, "1234", nil))
			m.step(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
			lines := strings.Split(m.View(), "\n")
			if len(lines) != sz[1] {
				t.Errorf("widths %v, %v: %d rows", w, sz, len(lines))
			}
			for r, l := range lines {
				if got := dispW(l); got != sz[0] {
					t.Errorf("widths %v, %v: row %d is %d wide", w, sz, r, got)
				}
			}
		}
	}
}

// The width a custom saver is handed with the box: the icon in its title
// counted (tdp D6).
func TestPromptBoxWithIcons(t *testing.T) {
	withIcons(t, 2)
	cfg := testLock(t, "1234", nil).cfg
	var widths []int
	var box string
	p := NewLockPrompt(cfg, "", 100, 30, func(b string, w int) { box, widths = b, append(widths, w) }, Tries{})
	runCmd(p, p.Init())
	if len(widths) == 0 || widths[len(widths)-1] != pinBoxW {
		t.Errorf("the box's width for the custom saver: %v, want %d", widths, pinBoxW)
	}
	for r, l := range strings.Split(box, "\n") {
		if dispW(l) != pinBoxW {
			t.Errorf("row %d is %d wide, want %d: %q", r, dispW(l), pinBoxW, ansi.Strip(l))
		}
	}
}

// On a lock a few rows high the box is no higher than the screen allows
// (tdp F7), its top border on the screen, and nothing of it past the
// screen's edges, on the board and over a custom saver's program alike
// (tdp E5).
func TestPromptOnATinyScreen(t *testing.T) {
	for _, sz := range [][2]int{{20, 5}, {11, 4}} {
		m := testLock(t, "1234", nil)
		m.step(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		openPrompt(t, m)
		if v := m.View(); !strings.Contains(v, "┏") || len(strings.Split(v, "\n")) != sz[1] {
			t.Errorf("%v on the board:\n%s", sz, ansi.Strip(v))
		}
		var box string
		var w int
		p := NewLockPrompt(m.cfg, "", sz[0], sz[1], func(b string, bw int) { box, w = b, bw }, Tries{})
		runCmd(p, p.Init())
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
