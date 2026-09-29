package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// withIcons draws with icons n cells wide, as DetectIconWidth would set
// it, for the rest of the test.
func withIcons(t *testing.T, n int) {
	t.Helper()
	was := iconCells
	iconCells = n
	t.Cleanup(func() { iconCells = was })
}

// The icons locku draws take two cells on such a font; the powerline caps
// and the ambiguous marks do not (tdp D6).
func TestIsWideIcon(t *testing.T) {
	for _, g := range []string{pixelGlyph, glyphLock, glyphMenu, glyphList, glyphHelp, glyphWarn, glyphInfo, glyphInput} {
		if !isWideIcon([]rune(g)[0]) {
			t.Errorf("%U is an icon", []rune(g)[0])
		}
	}
	for _, g := range []string{capLeft, capRight, dividerHard, dividerSoft, "●", "…", "–", "→", "─", "a"} {
		if isWideIcon([]rune(g)[0]) {
			t.Errorf("%U is no icon", []rune(g)[0])
		}
	}
}

func TestWidthsCountTheIcon(t *testing.T) {
	s := "a" + pixelGlyph + "b"
	if dispW(s) != 3 {
		t.Errorf("one cell: %d", dispW(s))
	}
	withIcons(t, 2)
	if dispW(s) != 4 || dispW("\x1b[31m"+s+"\x1b[0m") != 4 {
		t.Errorf("two cells: %d", dispW(s))
	}
	// An icon that does not fit whole is left out, never cut.
	if got := clipANSI(s, 2); got != "a" {
		t.Errorf("clip to 2: %q", got)
	}
	if got := clipANSI(s, 3); got != "a"+pixelGlyph {
		t.Errorf("clip to 3: %q", got)
	}
	for w := 1; w <= 5; w++ {
		if got := padRight(s, w); dispW(got) != w {
			t.Errorf("padRight to %d is %d wide: %q", w, dispW(got), got)
		}
	}
	if got := dispCutLeft(s, 2); got != " b" {
		t.Errorf("half an icon cut from the left is a space: %q", got)
	}
	if got := joinH("a"+pixelGlyph+"\nab", "|\n|"); got != "a"+pixelGlyph+"|\nab |" {
		t.Errorf("a column with an icon keeps the next straight: %q", got)
	}
}

// compositeDisp puts a box where overlay.Composite does, and every row
// keeps the background's width whatever icon is in either (tdp D6).
func TestCompositeDisp(t *testing.T) {
	bg := strings.Repeat("0123456789\n", 4) + "0123456789"
	// With icons one cell, and a box that fits, it is overlay.Composite:
	// the centre, the toast's place, a box pushed against the edge.
	for _, c := range []struct {
		fg         string
		x, y       overlay.Position
		xOff, yOff int
	}{
		{"ab\ncd", overlay.Center, overlay.Center, 0, 0},
		{"abc", overlay.Center, overlay.Bottom, 0, -2},
		{"abcd\nefgh", overlay.Left, overlay.Top, 8, 0},
	} {
		if got, want := compositeDisp(c.fg, bg, c.x, c.y, c.xOff, c.yOff), overlay.Composite(c.fg, bg, c.x, c.y, c.xOff, c.yOff); got != want {
			t.Errorf("%q: %q, overlay has %q", c.fg, got, want)
		}
	}
	withIcons(t, 2)
	rows := func(s string) []string { return strings.Split(s, "\n") }
	check := func(label, out string) {
		t.Helper()
		for i, l := range rows(out) {
			if dispW(l) != 10 {
				t.Errorf("%s: row %d is %d wide: %q", label, i, dispW(l), l)
			}
		}
	}
	out := compositeDisp("x"+pixelGlyph+"y", bg, overlay.Center, overlay.Center, 0, 0)
	check("an icon in the box", out)
	if rows(out)[2] != "012x"+pixelGlyph+"y789" {
		t.Errorf("centred: %q", rows(out)[2])
	}
	iconBG := strings.Repeat("0"+pixelGlyph+"3456789\n", 4) + "0" + pixelGlyph + "3456789"
	check("an icon under it", compositeDisp("xyz", iconBG, overlay.Left, overlay.Top, 5, 0))
	// The icon takes cells 1–2: a box from 2 halves it on the left, one
	// over 0–1 on the right.
	if got := rows(compositeDisp("xyz", iconBG, overlay.Left, overlay.Top, 2, 0))[0]; got != "0 xyz56789" {
		t.Errorf("cut on its left: %q", got)
	}
	if got := rows(compositeDisp("xy", iconBG, overlay.Left, overlay.Top, 0, 0))[0]; got != "xy 3456789" {
		t.Errorf("cut on its right: %q", got)
	}
	// Larger than the screen — the frame a resize lands in, a PIN prompt
	// on a lock a few rows high: from 0, what falls off is cut, never a
	// panic (tdp D6, v0.1.21).
	tall := "a\nb\nc\nd\ne\nf\ng"
	for _, pos := range []overlay.Position{overlay.Center, overlay.Bottom} {
		if got := rows(compositeDisp(tall, bg, overlay.Center, pos, 0, -1)); len(got) != 5 || got[0] != "01234a6789" || got[4] != "01234e6789" {
			t.Errorf("a box taller than the screen, from the top: %q", got)
		}
	}
	wide := strings.Repeat("W", 14) + "\n" + strings.Repeat("W", 14)
	out = compositeDisp(wide, bg, overlay.Center, overlay.Center, 0, 0)
	check("a box wider than the screen", out)
	if rows(out)[1] != strings.Repeat("W", 10) {
		t.Errorf("a box wider than the screen, from the left: %q", rows(out)[1])
	}
	both := strings.Repeat(strings.Repeat("B", 14)+"\n", 7) + strings.Repeat("B", 14)
	out = compositeDisp(both, bg, overlay.Center, overlay.Center, 0, 0)
	check("a box larger both ways", out)
	if len(rows(out)) != 5 {
		t.Errorf("a box larger both ways: %d rows", len(rows(out)))
	}
	// An icon on the screen's last cell is not half drawn.
	check("an icon at the edge", compositeDisp(strings.Repeat("W", 9)+pixelGlyph, bg, overlay.Left, overlay.Top, 0, 0))
}

// Every popup of the settings screen, with icons one cell and two: the box
// alone is as wide on every row, and on the screen every row is the
// terminal's width (tdp D6, L4).
func TestEveryPopupEveryRowWithIcons(t *testing.T) {
	type popup struct {
		name string
		open func(m AppModel) (AppModel, string)
	}
	unsaved := func(m AppModel) AppModel {
		return m.press("2").typed(strings.Repeat("j", stopBgR)).press("enter", "G", "enter")
	}
	popups := []popup{
		{"Space menu", func(m AppModel) (AppModel, string) { m = m.press(" "); return m, m.menu.view() }},
		{"global operation", func(m AppModel) (AppModel, string) {
			m = m.press(" ", "G", "enter")
			return m, m.globalMenu.view()
		}},
		{"options", func(m AppModel) (AppModel, string) { m = m.press("2", "j", "enter"); return m, m.options.view() }},
		{"key reference", func(m AppModel) (AppModel, string) { m = m.press("?"); return m, m.help.view() }},
		{"glossary", func(m AppModel) (AppModel, string) { m = m.press("G", "2", "?"); return m, m.help.view() }},
		{"confirm", func(m AppModel) (AppModel, string) { m = m.press("j", "X"); return m, m.confirm.view() }},
		{"input", func(m AppModel) (AppModel, string) { m = m.press("r"); return m, m.input.view() }},
		{"path, with an offer", func(m AppModel) (AppModel, string) {
			m = m.press("G", "k", "k", "2", "j", "enter")
			return m, m.input.view()
		}},
		{"PIN box", func(m AppModel) (AppModel, string) { m = m.press("G", "2", "enter"); return m, m.input.view() }},
		{"quit confirm", func(m AppModel) (AppModel, string) { m = unsaved(m).press("q"); return m, m.quitAsk.view() }},
		{"toast", func(m AppModel) (AppModel, string) {
			m = m.press("G", "2", "enter").typed("1234").press("enter").typed("1234").press("enter")
			return m, m.toast.view()
		}},
	}
	for _, cells := range []int{1, 2} {
		withIcons(t, cells)
		for _, sz := range [][2]int{{100, 30}, {40, 12}} {
			for _, p := range popups {
				m, box := p.open(newTestApp(t).size(sz[0], sz[1]))
				lines := strings.Split(box, "\n")
				if len(lines) < 3 {
					t.Fatalf("icons %d, %dx%d, %s: no box", cells, sz[0], sz[1], p.name)
				}
				for r, l := range lines {
					if dispW(l) != dispW(lines[len(lines)-1]) {
						t.Errorf("icons %d, %dx%d, %s alone: row %d is %d wide, the bottom %d\n  %q", cells, sz[0], sz[1], p.name, r, dispW(l), dispW(lines[len(lines)-1]), ansi.Strip(l))
					}
				}
				for r, l := range strings.Split(m.View(), "\n") {
					if dispW(l) != sz[0] {
						t.Errorf("icons %d, %dx%d, %s on the screen: row %d is %d wide\n  %q", cells, sz[0], sz[1], p.name, r, dispW(l), ansi.Strip(l))
					}
				}
			}
		}
	}
}

// The lock's PIN prompt alone, and the width a custom saver is handed
// with it: the icon in its title counted (tdp D6).
func TestPromptBoxWithIcons(t *testing.T) {
	withIcons(t, 2)
	m := openPrompt(t, testLock(t, "1234", nil))
	box := strings.Split(m.prompt.view(m.now()), "\n")
	want := popupInnerW(80) + 2
	for r, l := range box {
		if dispW(l) != want {
			t.Errorf("row %d is %d wide, want %d: %q", r, dispW(l), want, ansi.Strip(l))
		}
	}
	// On the board it sits a column in, as with icons one cell: every row
	// being 80 wide is not enough — a box a pixel off to the right, its
	// last pixel dropped, is 80 too.
	for _, r := range []string{"╭", "│", "╰"} {
		if l := line(m.View(), r); dispW(l[:strings.Index(l, r)]) != 1 {
			t.Errorf("the box's %s is at %d, want 1", r, dispW(l[:strings.Index(l, r)]))
		}
	}
	cfg := testLock(t, "1234", nil).cfg
	var widths []int
	p := NewLockPrompt(cfg, "", 100, 30, func(_ string, w int) { widths = append(widths, w) })
	var pm tea.Model = p
	for i := 0; i < animFrames+1; i++ {
		pm, _ = pm.(LockModel).Update(AnimTickMsg{Target: "pinprompt"})
	}
	if len(widths) == 0 || widths[len(widths)-1] != popupInnerW(100)+2 {
		t.Errorf("the box's width for the custom saver: %v, want %d", widths, popupInnerW(100)+2)
	}
}

// The splash's pixels are two cells either way, and it fills the screen.
func TestSplashWithIcons(t *testing.T) {
	for _, cells := range []int{1, 2} {
		withIcons(t, cells)
		s := newSplashModel()
		s.show()
		s.revealedCount = len(s.pixelOrder) / 2 // lit and dark side by side
		s, _ = s.update(splashIdentityMsg{})
		s, _ = s.update(splashHintMsg{})
		logoW := len(logoPixels[0]) * 2
		for r, l := range strings.Split(s.render(logoW, 0), "\n")[:len(logoPixels)] {
			if dispW(l) != logoW {
				t.Errorf("icons %d: logo row %d is %d wide, want %d", cells, r, dispW(l), logoW)
			}
		}
		for r, l := range strings.Split(s.render(100, 40), "\n") {
			if dispW(l) != 100 {
				t.Errorf("icons %d: splash row %d is %d wide, want 100", cells, r, dispW(l))
			}
		}
	}
}

// On a lock a few rows high the PIN prompt starts at the top and what
// falls off is cut, on the board and over a custom saver's program alike:
// handed on whole, the rows past the bottom would land on the last one
// (tdp D6, v0.1.21).
func TestPromptOnATinyScreen(t *testing.T) {
	for _, sz := range [][2]int{{20, 5}, {11, 4}} {
		m := testLock(t, "1234", nil)
		m, _ = m.step(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		m = openPrompt(t, m)
		if top := strings.Split(m.View(), "\n")[0]; !strings.Contains(top, "╭") {
			t.Errorf("%v on the board: the top row is %q", sz, ansi.Strip(top))
		}
		var box string
		var w int
		var pm tea.Model = NewLockPrompt(m.cfg, "", sz[0], sz[1], func(b string, bw int) { box, w = b, bw })
		for i := 0; i < animFrames+1; i++ {
			pm, _ = pm.(LockModel).Update(AnimTickMsg{Target: "pinprompt"})
		}
		lines := strings.Split(box, "\n")
		if len(lines) != sz[1] || !strings.Contains(lines[0], "╭") || w > sz[0] {
			t.Errorf("%v over a custom saver: %d rows, %d wide, the top %q", sz, len(lines), w, ansi.Strip(lines[0]))
		}
		for r, l := range lines {
			if dispW(l) > sz[0] {
				t.Errorf("%v over a custom saver: row %d is %d wide", sz, r, dispW(l))
			}
		}
	}
}
