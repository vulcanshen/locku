package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// The snake is a game on the board in colours of its own (user,
// 2026-10-06), the bouncing box's and white: it is white at the start
// (user, 2026-10-07), and takes an apple's colour when the apple has gone
// down its body; every frame it moves.
func TestSnakeLockWearsItsOwnColours(t *testing.T) {
	m := testLock(t, "1234", func(c *config.Config) {
		c.Profiles = []config.Profile{config.NewProfile("s", saver.KindSnake)}
		c.Profile = "s"
	})
	if _, ok := m.game.(*saver.Snake); !ok {
		t.Fatalf("the game is %T", m.game)
	}
	if inks := m.inks(); len(inks) != 12 || inks[0] != lipgloss.Color(config.DefaultBG) || inks[1] != lipgloss.Color("#f2b753") || inks[11] != lipgloss.Color("#ffffff") {
		t.Fatalf("inks %v", inks)
	}
	white := 0
	for _, k := range m.shown.ink {
		if k == 11 {
			white++
		}
	}
	if white < 7*m.shown.count()/10 {
		t.Fatalf("%d of %d lit white at the start", white, m.shown.count())
	}
	before := m.shown.clone()
	m, cmd := m.step(clockTickMsg{gen: m.tickGen})
	if cmd == nil || m.rev != nil {
		t.Fatal("a frame must schedule the next and never reveal")
	}
	same := true
	for i := range before.ink {
		if before.ink[i] != m.shown.ink[i] {
			same = false
		}
	}
	if same {
		t.Error("the snake did not move")
	}
}

// On the settings screen the snake has its speed to set and nothing
// else, not even colours (user, 2026-10-06): cells a second, typed in a
// number box — one to thirty, empty for twelve, anything else refused in
// its error row.
func TestSnakeHasItsSpeed(t *testing.T) {
	m := newTestApp(t).press("G", "k", "k", "k", "k") // the snake, above the custom saver
	if it := m.sideAt(); it.kind != sideSaver || saver.Kinds[it.ref] != saver.KindSnake {
		t.Fatalf("the snake sits above the custom saver, not %+v", it)
	}
	v := m.View()
	if !strings.Contains(v, "the Nokia snake") || !strings.Contains(v, "12 cells a second") || strings.Contains(v, " bg ") || strings.Contains(v, " fg ") || strings.Contains(v, "layout") || strings.Contains(v, "runner") {
		t.Errorf("the snake's [2]:\n%s", v)
	}
	m = m.press("n", "enter")
	it := m.sideAt()
	if it.kind != sideProfile || m.cfg.Profiles[it.ref] != (config.Profile{Name: "snake", Saver: saver.KindSnake, Speed: 12}) {
		t.Fatalf("the new profile: %+v", m.cfg.Profiles)
	}
	keys := m.press("2", " ").menu.menuKeys()
	if hotkeyIndex(keys, "P") < 0 || hotkeyIndex(keys, "S") >= 0 || hotkeyIndex(keys, "R") >= 0 {
		t.Errorf("its menu: %v", keys)
	}
	m = m.press("2", "j")
	if r := m.rowAt(); r.kind != rowSpeed {
		t.Fatalf("the row under the name: %+v", r)
	}
	m = m.press("enter")
	if !m.input.isInteractive() || m.input.title != "number" || m.input.value != "12" {
		t.Fatalf("the box %+v", m.input)
	}
	for _, bad := range []string{"0", "31", "-3", "fast"} {
		m = m.press("ctrl+u").typed(bad).press("enter")
		if m.input.err != "a whole number, 1 to 30" || m.cfg.Profiles[it.ref].Speed != 12 {
			t.Fatalf("%q: error %q, speed %d", bad, m.input.err, m.cfg.Profiles[it.ref].Speed)
		}
	}
	m = m.press("ctrl+u").typed("20").press("enter")
	if m.cfg.Profiles[it.ref].Speed != 20 || saved(t).Profiles[it.ref].Speed != 20 || !strings.Contains(m.View(), "20 cells a second") {
		t.Errorf("20: speed %d", m.cfg.Profiles[it.ref].Speed)
	}
	m = m.press("enter", "ctrl+u", "enter")
	if m.cfg.Profiles[it.ref].Speed != 12 {
		t.Errorf("empty: speed %d", m.cfg.Profiles[it.ref].Speed)
	}
}

// The lock runs the snake at the profile speed (user, 2026-10-06).
func TestSnakeLockRunsAtTheProfileSpeed(t *testing.T) {
	m := testLock(t, "1234", func(c *config.Config) {
		p := config.NewProfile("s", saver.KindSnake)
		p.Speed = 20
		c.Profiles = []config.Profile{p}
		c.Profile = "s"
	})
	if d := m.game.Next(at).Sub(at); d != 50*time.Millisecond {
		t.Errorf("a move every %v at 20 cells a second", d)
	}
}
