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
	m := newTestApp(t).press("G", "k", "k", "k", "k", "k", "k", "k", "n", "enter") // a profile of the dino
	it := m.sideAt()
	if it.kind != sideProfile || m.cfg.Profiles[it.ref].Saver != saver.KindRunner || m.cfg.Profiles[it.ref].Character != saver.CharacterTRex {
		t.Fatalf("the new profile: %+v", m.cfg.Profiles)
	}
	m = m.press("2", "j", "j")
	if r := m.rowAt(); r.kind != rowCharacter || r.label != "character" || r.value != "t-rex" {
		t.Fatalf("the row under the participants: %+v", r)
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
			p := config.NewProfile("d", saver.KindRunner)
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

// A runner's background is chosen on [2] as its character is (user,
// 2026-10-07): day, night or time-shifting, time-shifting to begin
// with, written at once; it has no bg / fg, so no swatches and no
// channels.
func TestRunnerBackgroundIsChosen(t *testing.T) {
	m := newTestApp(t).press("G", "k", "k", "k", "k", "k", "k", "k", "n", "enter") // a profile of the runner
	it := m.sideAt()
	if p := m.cfg.Profiles[it.ref]; it.kind != sideProfile || p.Saver != saver.KindRunner || p.Background != saver.BackgroundTimeShifting || p.BG != "" || p.FG != "" {
		t.Fatalf("the new profile: %+v", m.cfg.Profiles)
	}
	m = m.press("2", "j", "j", "j", "j")
	if r := m.rowAt(); r.kind != rowBackground || r.label != "background" || r.value != "time-shifting" {
		t.Fatalf("the row under the scene: %+v", r)
	}
	if v := m.View(); strings.Contains(v, " bg ") || strings.Contains(v, " fg ") {
		t.Errorf("a runner wears no colours of the profile's:\n%s", v)
	}
	m = m.press("enter")
	if !m.options.isInteractive() || len(m.options.items) != len(saver.Backgrounds) || m.options.items[m.options.cursor].label != "time-shifting" {
		t.Fatalf("options %+v", m.options.items)
	}
	m = m.press("k", "k", "enter")
	if got := m.cfg.Profiles[it.ref].Background; got != saver.BackgroundDay || saved(t).Profiles[it.ref].Background != saver.BackgroundDay {
		t.Errorf("chose %q", got)
	}
}
