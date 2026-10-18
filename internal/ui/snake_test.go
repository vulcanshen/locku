package ui

import (
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

// The lock runs the snake at the profile speed (user, 2026-10-06):
// super fast is twenty cells a second.
func TestSnakeLockRunsAtTheProfileSpeed(t *testing.T) {
	m := testLock(t, "1234", func(c *config.Config) {
		p := config.NewProfile("s", saver.KindSnake)
		p.Speed = saver.SpeedSuperFast
		c.Profiles = []config.Profile{p}
		c.Profile = "s"
	})
	if d := m.game.Next(at).Sub(at); d != 50*time.Millisecond {
		t.Errorf("a move every %v super fast", d)
	}
}
