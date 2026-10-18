package ui

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// The pets are a game on the board in colours of their own (user,
// 2026-10-07): the outdoors' — the sky over a green field a row at a
// time, the ground's grass lit from the first frame, the trees' — the
// z's, and the cats' coats; a frame every 80 ms, and the cats go about.
func TestPetsLockWearsItsOwnColours(t *testing.T) {
	m := testLock(t, "1234", func(c *config.Config) {
		c.Profiles = []config.Profile{config.NewProfile("p", saver.KindPets)}
		c.Profile = "p"
	})
	if _, ok := m.game.(*saver.Pets); !ok {
		t.Fatalf("the game is %T", m.game)
	}
	if inks := m.inks(); len(inks) != 32 || inks[0] != lipgloss.Color("#7e9bbb") || inks[1] != lipgloss.Color("#4f7d36") || inks[2] != lipgloss.Color("#847260") || inks[7] != lipgloss.Color("#f6b06a") {
		t.Fatalf("inks %v", inks)
	}
	if s := m.shading(); s == nil || s.Looks[0].Ground[0] != "#7e9bbb" || s.Looks[0].Ground[len(s.Looks[0].Ground)-1] != "#1c2f1a" {
		t.Fatalf("no backdrop: %+v", s)
	}
	grass := 0
	for _, k := range m.shown.ink {
		if k == 1 {
			grass++
		}
	}
	if grass < m.shown.w {
		t.Fatalf("%d lit grass at the start: not even the ground", grass)
	}
	if d := m.game.Next(at).Sub(at); d != 80*time.Millisecond {
		t.Errorf("a frame every %v", d)
	}
	before := m.shown.clone()
	same := true
	for i := 0; i < 60 && same; i++ {
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
		t.Error("no cat moved")
	}
}
