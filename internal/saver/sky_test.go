package saver

import (
	"slices"
	"testing"
	"time"
)

// The runner's background (user, 2026-10-07): day, white with a pale
// blue above it and the runner in surface0; night, the ground and gold
// it had; or time-shifting, a day in three minutes by the clock — day at
// the hour and every third minute, dusk the minute after, night the one
// after that, each at once; anything else is time-shifting. Each is a
// gradient down the board, through every one of its colours, a row its
// own shade — never one colour.
func TestRunnerBackgrounds(t *testing.T) {
	if !slices.Equal(Backgrounds, []string{"day", "night", "time-shifting"}) {
		t.Errorf("backgrounds %v", Backgrounds)
	}
	if daySky.stops[len(daySky.stops)-1] != "#ffffff" || daySky.fg != "#313244" || !slices.Contains(nightSky.stops, "#313244") || nightSky.fg != "#f2b753" {
		t.Errorf("day %v, night %v", daySky, nightSky)
	}
	at := func(minute int) func() time.Time {
		return func() time.Time { return time.Date(2026, time.October, 7, 14, minute, 59, 0, time.Local) }
	}
	for _, c := range []struct {
		background string
		minute     int
		want       sky
	}{
		{BackgroundDay, 2, daySky},
		{BackgroundNight, 0, nightSky},
		{BackgroundTimeShifting, 0, daySky},
		{BackgroundTimeShifting, 1, duskSky},
		{BackgroundTimeShifting, 2, nightSky},
		{BackgroundTimeShifting, 3, daySky},
		{BackgroundTimeShifting, 58, duskSky},
		{"", 59, nightSky},
		{"noon", 30, daySky},
	} {
		d := NewDino(1, RunnerBig, CharacterTRex, SceneGrass, c.background, at(c.minute))
		inks, ground := d.Inks(), d.Ground(31)
		top, bottom := c.want.stops[0], c.want.stops[len(c.want.stops)-1]
		if len(inks) != 2 || inks[0] != top || inks[1] != c.want.fg || len(ground) != 31 || ground[0] != top || ground[30] != bottom {
			t.Errorf("%q at minute %d: inks %v, the ground %v", c.background, c.minute, inks, ground)
			continue
		}
		seen := map[string]bool{}
		for _, g := range ground {
			seen[g] = true
		}
		for _, s := range c.want.stops {
			if !seen[s] {
				t.Errorf("%q at minute %d: the ground misses %s: %v", c.background, c.minute, s, ground)
			}
		}
		if len(seen) < 20 {
			t.Errorf("%q at minute %d: %d shades in 31 rows", c.background, c.minute, len(seen))
		}
	}
}
