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
		if len(inks) != 5 || inks[0] != top || inks[1] != c.want.fg || len(ground) != 31 || ground[0] != top || ground[30] != bottom {
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

// The sun by day and the moon by night (user, 2026-10-07), up to the
// right, the clouds in front of them; at dusk the setting sun, half of
// it on the ground under where the sun was. Each is in its colour while
// its sky is there; out of it, it is not drawn at all.
func TestRunnerSunAndMoon(t *testing.T) {
	at := func(minute int) func() time.Time {
		return func() time.Time { return time.Date(2026, time.October, 7, 14, minute, 59, 0, time.Local) }
	}
	for _, c := range []struct {
		background        string
		minute            int
		sun, moon, sunset bool
	}{
		{BackgroundDay, 1, true, false, false},
		{BackgroundNight, 0, false, true, false},
		{BackgroundTimeShifting, 0, true, false, false},
		{BackgroundTimeShifting, 1, false, false, true},
		{BackgroundTimeShifting, 2, false, true, false},
	} {
		d := NewDino(1, RunnerBig, CharacterTRex, SceneGrass, c.background, at(c.minute))
		sc := d.Draw(76, 31)
		count := map[uint8]int{}
		for _, k := range sc.Pix {
			count[k]++
		}
		if sun, moon, sunset := count[inkSun] > 0, count[inkMoon] > 0, count[inkSunset] > 0; sun != c.sun || moon != c.moon || sunset != c.sunset {
			t.Errorf("%q at minute %d: the sun %v, the moon %v, the setting sun %v", c.background, c.minute, sun, moon, sunset)
		}
		inks := d.Inks()
		if (inks[2] == sunColour) != c.sun || (inks[3] == moonColour) != c.moon || (inks[4] == sunsetColour) != c.sunset {
			t.Errorf("%q at minute %d: the sun %s, the moon %s, the setting sun %s", c.background, c.minute, inks[2], inks[3], inks[4])
		}
		// The setting sun under the sun, its flat side on the ground.
		if tx, ty := d.sunsetAt(); tx != d.w-d.w/6-sunArt.w() || ty+sunsetArt.h() != d.groundY() || sunsetArt.h() != 4 || sunsetArt.w() != 7 {
			t.Errorf("the setting sun at %d,%d", tx, ty)
		}
		// Each where it sits, up to the right, the moon left of the sun.
		sx, sy := d.sunAt()
		mx, my := d.moonAt()
		if sy != 1 || my != 1 || mx+moonArt.w() > sx || sx+sunArt.w() > 76 || mx < 76/2 {
			t.Errorf("the sun at %d,%d, the moon at %d,%d", sx, sy, mx, my)
		}
		if c.sun && sc.Pix[(sy+3)*76+sx+3] != inkSun && sc.Pix[(sy+3)*76+sx+3] != 1 {
			t.Errorf("%q at minute %d: no sun at its middle", c.background, c.minute)
		}
	}
}

// A cloud going by the sun is in front of it, and an obstacle going by
// the setting sun.
func TestRunnerCloudsPassInFrontOfTheSun(t *testing.T) {
	d := NewDino(1, RunnerBig, CharacterTRex, SceneGrass, BackgroundDay, time.Now)
	d.Draw(76, 31)
	sx, sy := d.sunAt()
	d.clouds = []cloud{{x: sx, y: sy}}
	if sc := d.Draw(76, 31); sc.Pix[(sy+1)*76+sx+3] != 1 {
		t.Errorf("the cloud over the sun is ink %d", sc.Pix[(sy+1)*76+sx+3])
	}
	dusk := func() time.Time { return time.Date(2026, time.October, 7, 14, 1, 59, 0, time.Local) }
	d = NewDino(1, RunnerBig, CharacterTRex, SceneGrass, BackgroundTimeShifting, dusk)
	d.Draw(76, 31)
	tx, _ := d.sunsetAt()
	d.obs = []obstacle{{x: tx, kind: 4}} // the big cactus, ten tall
	if sc := d.Draw(76, 31); sc.Pix[(d.groundY()-2)*76+tx+2] != 1 && sc.Pix[(d.groundY()-2)*76+tx+3] != 1 {
		t.Errorf("the cactus over the setting sun: %d %d", sc.Pix[(d.groundY()-2)*76+tx+2], sc.Pix[(d.groundY()-2)*76+tx+3])
	}
}

// Time-shifting fades each part of the day in from the one before over
// its first twenty seconds (user, 2026-10-07: night to day was sudden):
// the sky, the runner's colour, and the sun and the moon, both drawn
// while one fades out and the other in. Second by second round the
// whole day no colour moves by more than a little.
func TestRunnerSkyFades(t *testing.T) {
	moment := time.Date(2026, time.October, 7, 14, 3, 10, 0, time.Local) // ten seconds into day, from night
	d := NewDino(1, RunnerBig, CharacterTRex, SceneGrass, BackgroundTimeShifting, func() time.Time { return moment })
	sc := d.Draw(76, 31)
	inks, ground := d.Inks(), d.Ground(31)
	if ground[0] != mix(nightSky.stops[0], daySky.stops[0], 0.5) || ground[30] != mix("#45475a", "#ffffff", 0.5) || inks[1] != mix(nightSky.fg, daySky.fg, 0.5) {
		t.Errorf("half way from night to day: the ground %s … %s, the runner %s", ground[0], ground[30], inks[1])
	}
	_, sy := d.sunAt()
	if inks[2] != mix(skyAt(nightSky, daySky, 0.5, d.down(sy+3)), sunColour, 0.5) || inks[2] == sunColour || inks[3] == moonColour {
		t.Errorf("half way: the sun %s, the moon %s", inks[2], inks[3])
	}
	count := map[uint8]int{}
	for _, k := range sc.Pix {
		count[k]++
	}
	if count[inkSun] == 0 || count[inkMoon] == 0 || count[inkSunset] != 0 {
		t.Errorf("half way, the sun and the moon both, and no setting sun: %v", count)
	}
	// From day to dusk, the sun going and the setting sun coming.
	moment = time.Date(2026, time.October, 7, 14, 1, 10, 0, time.Local)
	sc = d.Draw(76, 31)
	count = map[uint8]int{}
	for _, k := range sc.Pix {
		count[k]++
	}
	inks = d.Inks()
	if count[inkSun] == 0 || count[inkSunset] == 0 || count[inkMoon] != 0 || inks[2] == sunColour || inks[4] == sunsetColour {
		t.Errorf("half way from day to dusk: %v, the sun %s, the setting sun %s", count, inks[2], inks[4])
	}
	step := func(a, b string) int {
		ca, cb := channels(a), channels(b)
		most := 0
		for k := range ca {
			most = max(most, ca[k]-cb[k], cb[k]-ca[k])
		}
		return most
	}
	var last []string
	for s := 0; s <= 3*60; s++ {
		moment = time.Date(2026, time.October, 7, 14, 0, 0, 0, time.Local).Add(time.Duration(s) * time.Second)
		now := append(d.Ground(31), d.Inks()[:2]...)
		for i := range last {
			if step(last[i], now[i]) > 12 {
				t.Fatalf("second %d: %s to %s", s, last[i], now[i])
			}
		}
		last = now
	}
}
