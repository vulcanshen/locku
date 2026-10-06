package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// The snake is a game on the board in colours of its own (user,
// 2026-10-06), the bouncing box's: it changes them at every apple; every
// frame it moves.
func TestSnakeLockWearsItsOwnColours(t *testing.T) {
	m := testLock(t, "1234", func(c *config.Config) {
		c.Profiles = []config.Profile{config.NewProfile("s", saver.KindSnake)}
		c.Profile = "s"
	})
	if _, ok := m.game.(*saver.Snake); !ok {
		t.Fatalf("the game is %T", m.game)
	}
	if inks := m.inks(); len(inks) != 11 || inks[0] != lipgloss.Color(config.DefaultBG) || inks[1] != lipgloss.Color("#f2b753") {
		t.Fatalf("inks %v", inks)
	}
	if m.shown.count() == 0 {
		t.Fatal("nothing lit")
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

// On the settings screen the snake has nothing to set, not even
// colours; a profile of it is its name and its saver.
func TestSnakeHasNothingToSet(t *testing.T) {
	m := newTestApp(t).press("G", "k", "k", "k", "k") // the snake, above the custom saver
	if it := m.sideAt(); it.kind != sideSaver || saver.Kinds[it.ref] != saver.KindSnake {
		t.Fatalf("the snake sits above the custom saver, not %+v", it)
	}
	v := m.View()
	if !strings.Contains(v, "the Nokia snake") || strings.Contains(v, " bg ") || strings.Contains(v, " fg ") || strings.Contains(v, "layout") || strings.Contains(v, "runner") {
		t.Errorf("the snake's [2]:\n%s", v)
	}
	m = m.press("n", "enter")
	it := m.sideAt()
	if it.kind != sideProfile || m.cfg.Profiles[it.ref] != (config.Profile{Name: "snake", Saver: saver.KindSnake}) {
		t.Fatalf("the new profile: %+v", m.cfg.Profiles)
	}
	keys := m.press("2", " ").menu.menuKeys()
	if hotkeyIndex(keys, "P") < 0 || hotkeyIndex(keys, "S") >= 0 || hotkeyIndex(keys, "R") >= 0 {
		t.Errorf("its menu: %v", keys)
	}
}
