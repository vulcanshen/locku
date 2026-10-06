package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

func bounceLock(t *testing.T) LockModel {
	t.Helper()
	return testLock(t, "1234", func(c *config.Config) {
		c.Profiles = []config.Profile{config.NewProfile("box", saver.KindBounce)}
		c.Profile = "box"
	})
}

// The bouncing box draws in its own colours (user, 2026-10-06): the
// ground and ten, the box in one of them; every frame it moves.
func TestBounceLockWearsItsOwnColours(t *testing.T) {
	m := bounceLock(t)
	if _, ok := m.game.(*saver.Bounce); !ok {
		t.Fatalf("the game is %T", m.game)
	}
	inks := m.inks()
	if len(inks) != 11 || inks[0] != lipgloss.Color(config.DefaultBG) || inks[1] != lipgloss.Color("#f2b753") || inks[2] != lipgloss.Color("#f38ba8") {
		t.Fatalf("inks %v", inks)
	}
	lit := map[uint8]int{}
	for _, k := range m.shown.ink {
		if k != inkOff {
			lit[k]++
		}
	}
	if len(lit) != 1 {
		t.Fatalf("the box in %d inks: %v", len(lit), lit)
	}
	before := m.shown.clone()
	m, cmd := m.step(clockTickMsg{gen: m.tickGen})
	if cmd == nil || m.rev != nil {
		t.Fatal("a frame must schedule the next and never reveal")
	}
	same := true
	for i := range before.ink {
		if (before.ink[i] != inkOff) != (m.shown.ink[i] != inkOff) {
			same = false
		}
	}
	if same {
		t.Error("the box did not move")
	}
	if m.style != (config.Style{BG: config.DefaultBG, FG: config.DefaultFG}) {
		t.Errorf("a saver with no bg / fg wears the defaults where it needs any: %+v", m.style)
	}
}

// spell sets a line as the board sets it: the box's time is the clock's.
func TestSpellIsTheBoardsLettering(t *testing.T) {
	rows := spell(faceShort, "21 05")
	b := paint(faceShort, one([]string{"21 05"}, 1), 80, 23)
	x0, y0, x1, y1 := box(b)
	if len(rows) != y1-y0+1 || len(rows[0]) != x1-x0+1 {
		t.Fatalf("spelled %dx%d, painted %dx%d", len(rows[0]), len(rows), x1-x0+1, y1-y0+1)
	}
	for y, row := range rows {
		for x := range row {
			if (row[x] == '#') != b.at(x0+x, y0+y) {
				t.Fatalf("pixel %d,%d differs:\n%s", x, y, strings.Join(rows, "\n"))
			}
		}
	}
}

// On the settings screen the bouncing box is a saver with nothing to
// set: no rows of its own, no colours, and so no draft to save or reset
// — a profile of it is its name and its saver, and P shows it.
func TestBounceHasNothingToSet(t *testing.T) {
	m := newTestApp(t).press("G", "k", "k", "k", "k") // the bouncing box, above the custom saver
	if it := m.sideAt(); it.kind != sideSaver || saver.Kinds[it.ref] != saver.KindBounce {
		t.Fatalf("the bouncing box sits above the custom saver, not %+v", it)
	}
	if v := m.View(); !strings.Contains(v, "the time in a box, bouncing, changing colour") || strings.Contains(v, " bg ") || strings.Contains(v, " fg ") || strings.Contains(v, "layout") {
		t.Errorf("the bouncing box's [2]:\n%s", v)
	}
	m = m.press("n", "enter")
	it := m.sideAt()
	if it.kind != sideProfile || m.cfg.Profiles[it.ref] != (config.Profile{Name: "bounce", Saver: saver.KindBounce}) {
		t.Fatalf("the new profile: %+v", m.cfg.Profiles)
	}
	keys := m.press("2", " ").menu.menuKeys()
	if hotkeyIndex(keys, "P") < 0 || hotkeyIndex(keys, "S") >= 0 || hotkeyIndex(keys, "R") >= 0 {
		t.Errorf("its menu: %v", keys)
	}
	if v := m.press("2").View(); strings.Contains(v, " bg ") || !strings.Contains(v, "bounce") {
		t.Errorf("its rows:\n%s", v)
	}
	if m = m.press("P"); m.preview == nil {
		t.Fatal("P shows nothing")
	} else if _, ok := m.preview.game.(*saver.Bounce); !ok {
		t.Errorf("the preview's game is %T", m.preview.game)
	}
}
