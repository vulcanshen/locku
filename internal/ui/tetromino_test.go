package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// The falling blocks are a game on the board in colours of their own
// (user, 2026-10-07): the ground, the seven pieces', white for a row
// going, the frame's grey, which is lit from the first frame, and the
// end's black and red; and a piece comes in and falls.
func TestTetrominoLockWearsItsOwnColours(t *testing.T) {
	m := testLock(t, "1234", func(c *config.Config) {
		c.Profiles = []config.Profile{config.NewProfile("t", saver.KindTetromino)}
		c.Profile = "t"
	})
	if _, ok := m.game.(*saver.Tetromino); !ok {
		t.Fatalf("the game is %T", m.game)
	}
	if inks := m.inks(); len(inks) != 12 || inks[0] != lipgloss.Color(config.DefaultBG) || inks[1] != lipgloss.Color("#89dceb") || inks[8] != lipgloss.Color("#ffffff") || inks[9] != lipgloss.Color("#7f849c") || inks[10] != lipgloss.Color("#11111b") || inks[11] != lipgloss.Color("#f38ba8") {
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

// On the settings screen the falling blocks have their speed to set and
// nothing else, as the snake (user, 2026-10-07): from the five by name,
// normal to begin with, written at once.
func TestTetrominoHasItsSpeed(t *testing.T) {
	m := newTestApp(t).press("G", "k", "k", "k", "k") // above the custom saver
	if it := m.sideAt(); it.kind != sideSaver || saver.Kinds[it.ref] != saver.KindTetromino {
		t.Fatalf("the falling blocks sit above the custom saver, not %+v", it)
	}
	v := m.View()
	if !strings.Contains(v, "falling blocks") || !strings.Contains(v, "normal") || strings.Contains(v, " bg ") || strings.Contains(v, " fg ") || strings.Contains(v, "layout") || strings.Contains(v, "Tetris") {
		t.Errorf("the falling blocks' [2]:\n%s", v)
	}
	m = m.press("n", "enter")
	it := m.sideAt()
	if it.kind != sideProfile || m.cfg.Profiles[it.ref] != (config.Profile{Name: "tetromino", Saver: saver.KindTetromino, Speed: saver.SpeedNormal}) {
		t.Fatalf("the new profile: %+v", m.cfg.Profiles)
	}
	m = m.press("2", "j")
	if r := m.rowAt(); r.kind != rowSpeed || r.label != "speed" || r.value != "normal" {
		t.Fatalf("the row under the name: %+v", r)
	}
	m = m.press("enter")
	if !m.options.isInteractive() || len(m.options.items) != len(saver.Speeds) || m.options.items[m.options.cursor].label != "normal" {
		t.Fatalf("options %+v", m.options.items)
	}
	m = m.press("k", "enter")
	if got := m.cfg.Profiles[it.ref].Speed; got != saver.SpeedSlow || saved(t).Profiles[it.ref].Speed != saver.SpeedSlow {
		t.Errorf("chose %q", got)
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
