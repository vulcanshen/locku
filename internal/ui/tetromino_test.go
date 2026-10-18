package ui

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// The falling blocks are a game on the board in colours of their own
// (user, 2026-10-07): the ground, the seven pieces', white for a row
// going, the frame's grey, which is lit from the first frame, and END's
// red; and a piece comes in and falls.
func TestTetrominoLockWearsItsOwnColours(t *testing.T) {
	m := testLock(t, "1234", func(c *config.Config) {
		c.Profiles = []config.Profile{config.NewProfile("t", saver.KindTetromino)}
		c.Profile = "t"
	})
	if _, ok := m.game.(*saver.Tetromino); !ok {
		t.Fatalf("the game is %T", m.game)
	}
	if inks := m.inks(); len(inks) != 11 || inks[0] != lipgloss.Color(config.DefaultBG) || inks[1] != lipgloss.Color("#89dceb") || inks[8] != lipgloss.Color("#ffffff") || inks[9] != lipgloss.Color("#7f849c") || inks[10] != lipgloss.Color("#f38ba8") {
		t.Fatalf("inks %v", inks)
	}
	grey := 0
	for _, k := range m.shown.ink {
		if k == 9 {
			grey++
		}
	}
	if grey < m.shown.count()/2 {
		t.Fatalf("%d of %d lit grey at the start: the frame", grey, m.shown.count())
	}
	before := m.shown.clone()
	same := true
	for i := 0; i < 5 && same; i++ {
		next, cmd := m.step(clockTickMsg{gen: m.tickGen})
		if cmd == nil || next.rev != nil {
			t.Fatal("a frame must schedule the next and never reveal")
		}
		m = next
		for j := range before.ink {
			if before.ink[j] != m.shown.ink[j] {
				same = false
			}
		}
	}
	if same {
		t.Error("no piece came in")
	}
}

// The lock runs the falling blocks at the profile speed: slow is five
// steps a second.
func TestTetrominoLockRunsAtTheProfileSpeed(t *testing.T) {
	m := testLock(t, "1234", func(c *config.Config) {
		p := config.NewProfile("t", saver.KindTetromino)
		p.Speed = saver.SpeedSlow
		c.Profiles = []config.Profile{p}
		c.Profile = "t"
	})
	if d := m.game.Next(at).Sub(at); d != 200*time.Millisecond {
		t.Errorf("a step every %v slow", d)
	}
}
