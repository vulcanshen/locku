package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// The snake is a game on the board in its profile's two colours
// (2026-10-06): the Nokia's greens unless the profile says otherwise;
// every frame it moves.
func TestSnakeLockIsInTheProfilesColours(t *testing.T) {
	m := testLock(t, "1234", func(c *config.Config) {
		p := config.NewProfile("s", saver.KindSnake)
		p.FG = "#ffffff"
		c.Profiles = []config.Profile{p}
		c.Profile = "s"
	})
	if _, ok := m.game.(*saver.Snake); !ok {
		t.Fatalf("the game is %T", m.game)
	}
	if inks := m.inks(); len(inks) < 2 || inks[0] != lipgloss.Color(config.SnakeBG) || inks[1] != lipgloss.Color("#ffffff") {
		t.Fatalf("inks %v", inks)
	}
	if n := m.shown.count(); n < 5 || n > 6 {
		t.Fatalf("%d lit: a snake three long, maybe its apple", n)
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

// On the settings screen the snake has its two colours and nothing else
// to set; a profile of it starts in the Nokia's greens.
func TestSnakeHasOnlyItsColours(t *testing.T) {
	m := newTestApp(t).press("G", "k", "k", "k", "k") // the snake, above the custom saver
	if it := m.sideAt(); it.kind != sideSaver || saver.Kinds[it.ref] != saver.KindSnake {
		t.Fatalf("the snake sits above the custom saver, not %+v", it)
	}
	v := m.View()
	if !strings.Contains(v, "the Nokia snake") || !strings.Contains(v, " bg ") || !strings.Contains(v, " fg ") || strings.Contains(v, "layout") || strings.Contains(v, "runner") {
		t.Errorf("the snake's [2]:\n%s", v)
	}
	m = m.press("n", "enter")
	it := m.sideAt()
	if it.kind != sideProfile || m.cfg.Profiles[it.ref] != (config.Profile{Name: "snake", Saver: saver.KindSnake, BG: config.SnakeBG, FG: config.SnakeFG}) {
		t.Fatalf("the new profile: %+v", m.cfg.Profiles)
	}
	keys := m.press("2", " ").menu.menuKeys()
	if hotkeyIndex(keys, "P") < 0 || hotkeyIndex(keys, "S") < 0 || hotkeyIndex(keys, "R") < 0 {
		t.Errorf("its menu: %v", keys)
	}
}
