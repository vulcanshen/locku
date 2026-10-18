package ui

import (
	"testing"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

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
