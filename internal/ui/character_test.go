package ui

import (
	"strings"
	"testing"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// A dino's character is chosen on [2] as its runner is (user,
// 2026-10-06): the five, the cursor on the one it is, written at once.
func TestDinoCharacterIsChosen(t *testing.T) {
	m := newTestApp(t).press("G", "k", "k", "k", "k", "k", "k", "n", "enter") // a profile of the dino
	it := m.sideAt()
	if it.kind != sideProfile || m.cfg.Profiles[it.ref].Saver != saver.KindDino || m.cfg.Profiles[it.ref].Character != saver.CharacterTRex {
		t.Fatalf("the new profile: %+v", m.cfg.Profiles)
	}
	m = m.press("2", "j", "j")
	if r := m.rowAt(); r.kind != rowCharacter || r.label != "character" || r.value != "t-rex" {
		t.Fatalf("the row under the runner: %+v", r)
	}
	m = m.press("enter")
	if !m.options.isInteractive() || len(m.options.items) != len(saver.Characters) || m.options.items[m.options.cursor].label != "t-rex" {
		t.Fatalf("options %+v", m.options.items)
	}
	m = m.press("j", "enter")
	if got := m.cfg.Profiles[it.ref].Character; got != saver.CharacterCat || saved(t).Profiles[it.ref].Character != saver.CharacterCat {
		t.Errorf("chose %q", got)
	}
	if v := m.View(); !strings.Contains(v, "character") || !strings.Contains(v, "cat") {
		t.Errorf("[2]:\n%s", v)
	}
}

// The lock runs the profile's character: in the runner's place on
// the first frame, a cat is not a T-Rex.
func TestTheLockRunsTheCharacter(t *testing.T) {
	runner := func(character string) int {
		m := testLock(t, "1234", func(c *config.Config) {
			p := config.NewProfile("d", saver.KindDino)
			p.Character = character
			c.Profiles = []config.Profile{p}
			c.Profile = "d"
		})
		sc := m.game.Draw(40, 28)
		n := 0
		for y := 28 - 2 - 14; y < 28-2; y++ {
			for x := 40 / 6; x < 40/6+12; x++ {
				if sc.Pix[y*sc.W+x] != 0 {
					n++
				}
			}
		}
		return n
	}
	if trex, cat := runner(saver.CharacterTRex), runner(saver.CharacterCat); trex == cat || cat == 0 {
		t.Errorf("the runner is %d pixels as a T-Rex, %d as a cat", trex, cat)
	}
}
