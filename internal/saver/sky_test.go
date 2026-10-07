package saver

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

// The runner's background (user, 2026-10-07): day, white with a pale
// blue above it and the runner in surface0; night, the ground and gold
// it had; or time-shifting, a day in three minutes by the clock — day at
// the hour and every third minute, dusk the minute after, night the one
// after that; anything else is time-shifting. Each is a gradient down
// the board, through every one of its colours, a row its own shade —
// never one colour — and once a sky is there, every square is in it.
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
		sh, inks := d.Shade(60, 31), d.Inks()
		ground := sh.Looks[1].Ground
		top, bottom := c.want.stops[0], c.want.stops[len(c.want.stops)-1]
		if len(inks) != 5 || inks[0] != top || inks[1] != c.want.fg || sh.Looks[1].Inks[1] != c.want.fg ||
			len(ground) != 31 || ground[0] != top || ground[30] != bottom {
			t.Errorf("%q at minute %d: inks %v, the ground %v", c.background, c.minute, inks, ground)
			continue
		}
		for x := 0; x < 60; x++ {
			for y := 0; y < 31; y++ {
				if sh.Look(x, y) != 1 {
					t.Fatalf("%q at minute %d: square %d,%d not in the sky", c.background, c.minute, x, y)
				}
			}
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

// Time-shifting turns from one part of the day to the next square by
// square (user, 2026-10-07: not a fade of the whole board): from right
// to left over ten seconds, the squares of a column at random; each
// square is the one sky or the other — its ground, the runner and the
// lights in it — never a mix, and a square turns once and stays. Half
// way from night to day the right edge is day, the left night, and in
// between are columns of both; the sun and the moon are both drawn, each
// seen in its own sky's squares.
func TestRunnerSkyTurnsSquareBySquare(t *testing.T) {
	var moment time.Time
	d := NewDino(1, RunnerBig, CharacterTRex, SceneGrass, BackgroundTimeShifting, func() time.Time { return moment })
	turn := time.Date(2026, time.October, 7, 14, 3, 0, 0, time.Local) // night to day
	moment = turn.Add(5 * time.Second)
	sc, sh := d.Draw(76, 31), d.Shade(60, 31)
	night, day := nightSky.look(31), daySky.look(31)
	if !reflect.DeepEqual(sh.Looks, [2]Look{night, day}) {
		t.Errorf("half way, the looks %+v", sh.Looks)
	}
	if night.Inks[inkSun] != "" || night.Inks[inkMoon] != moonColour || day.Inks[inkSun] != sunColour || day.Inks[inkMoon] != "" || day.Inks[1] != daySky.fg {
		t.Errorf("night %v, day %v", night.Inks, day.Inks)
	}
	mixed := 0
	for x := 0; x < 60; x++ {
		n := 0
		for y := 0; y < 31; y++ {
			n += sh.Look(x, y)
		}
		switch {
		case x == 59 && n != 31:
			t.Errorf("half way, the right edge: %d of 31 squares day", n)
		case x == 0 && n != 0:
			t.Errorf("half way, the left edge: %d of 31 squares day", n)
		case n > 0 && n < 31:
			mixed++
		}
	}
	if mixed < 5 {
		t.Errorf("half way, %d columns part day, part night", mixed)
	}
	count := map[uint8]int{}
	for _, k := range sc.Pix {
		count[k]++
	}
	if count[inkSun] == 0 || count[inkMoon] == 0 || count[inkSunset] != 0 {
		t.Errorf("half way, the sun and the moon both, and no setting sun: %v", count)
	}
	// A half second at a time through the turn: none turned at its start,
	// all by ten seconds, the right half ahead of the left, and none
	// turning back.
	turned := map[[2]int]bool{}
	for s := 0; s <= 24; s++ {
		moment = turn.Add(time.Duration(s) * time.Second / 2)
		sh := d.Shade(60, 31)
		right, left := 0, 0
		for x := 0; x < 60; x++ {
			for y := 0; y < 31; y++ {
				k := [2]int{x, y}
				switch {
				case sh.Look(x, y) == 1 && x >= 30:
					right, turned[k] = right+1, true
				case sh.Look(x, y) == 1:
					left, turned[k] = left+1, true
				case turned[k]:
					t.Fatalf("second %d: square %d,%d turned back", s, x, y)
				}
			}
		}
		switch {
		case s == 0 && right+left != 0, s >= 20 && right+left != 60*31, right < left:
			t.Errorf("half second %d: %d squares turned on the right, %d on the left", s, right, left)
		}
	}
	// From day to dusk, the sun going and the setting sun coming.
	moment = time.Date(2026, time.October, 7, 14, 1, 5, 0, time.Local)
	sc = d.Draw(76, 31)
	count = map[uint8]int{}
	for _, k := range sc.Pix {
		count[k]++
	}
	if count[inkSun] == 0 || count[inkSunset] == 0 || count[inkMoon] != 0 {
		t.Errorf("half way from day to dusk: %v", count)
	}
}

// The sun is yellow, not orange (user, 2026-10-07): its hue between 45
// and 60 degrees, and it stands out on the day sky. The moon is round
// (user, the same day): the sun's disc with a disc like it taken out to
// the right and up — every pixel of it in the disc, its left and bottom
// edges the disc's.
func TestRunnerSunYellowAndMoonRound(t *testing.T) {
	c := channels(sunColour)
	hue := 60 * float64(c[1]-c[2]) / float64(c[0]-c[2])
	if c[0] < c[1] || c[1] < c[2] || hue < 45 || hue > 60 {
		t.Errorf("the sun %s, hue %.0f", sunColour, hue)
	}
	if sky := channels(daySky.stops[0]); sky[2]-c[2] < 150 {
		t.Errorf("the sun %s on the day sky %s", sunColour, daySky.stops[0])
	}
	in := func(sp sprite, x, y int) bool {
		return y >= 0 && y < len(sp) && x >= 0 && x < len(sp[y]) && sp[y][x] == '#'
	}
	for y := range moonArt {
		for x := range moonArt[y] {
			if want := in(sunArt, x, y) && !in(sunArt, x-3, y+1); in(moonArt, x, y) != want {
				t.Errorf("the moon at %d,%d: %v", x, y, in(moonArt, x, y))
			}
		}
	}
}
