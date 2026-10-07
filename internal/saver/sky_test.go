package saver

import (
	"reflect"
	"slices"
	"strings"
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
		if len(inks) != 8 || inks[0] != top || inks[1] != c.want.fg || sh.Looks[1].Inks[1] != c.want.fg ||
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
		// Each where it sits, up to the right, the sun left of the moon.
		sx, sy := d.sunAt()
		mx, my := d.moonAt()
		if sy != 1 || my != 1 || sx+sunArt.w() > mx || mx+moonArt.w() > 76 || sx < 76/2 {
			t.Errorf("the sun at %d,%d, the moon at %d,%d", sx, sy, mx, my)
		}
		// The setting sun under the sun, its flat side on the ground.
		if tx, ty := d.sunsetAt(); tx != sx || ty+sunsetArt.h() != d.groundY() || sunsetArt.h() != 4 || sunsetArt.w() != 7 {
			t.Errorf("the setting sun at %d,%d", tx, ty)
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
// and 60 degrees, and it stands out on the day sky. The moon is as the
// user drew it (the same day), the sun's round along its left and its
// bottom.
func TestRunnerSunYellowAndMoonRound(t *testing.T) {
	c := channels(sunColour)
	hue := 60 * float64(c[1]-c[2]) / float64(c[0]-c[2])
	if c[0] < c[1] || c[1] < c[2] || hue < 45 || hue > 60 {
		t.Errorf("the sun %s, hue %.0f", sunColour, hue)
	}
	if sky := channels(daySky.stops[0]); sky[2]-c[2] < 150 {
		t.Errorf("the sun %s on the day sky %s", sunColour, daySky.stops[0])
	}
	drawn := sprite{
		"..##...",
		".##....",
		"##.....",
		"###...#",
		"####.##",
		".#####.",
		"..###..",
	}
	if !slices.Equal(moonArt, drawn) {
		t.Errorf("the moon\n%s", strings.Join(moonArt, "\n"))
	}
	for y := range moonArt {
		for x := range moonArt[y] {
			if moonArt[y][x] == '#' && sunArt[y][x] != '#' {
				t.Errorf("the moon at %d,%d is off the sun", x, y)
			}
		}
	}
}

// By night the runner keeps its dark (user, 2026-10-07): an outline round
// it, white as the moon — corners too, none under its feet — and its
// eyes in the night's gold, and nothing they pass drawn over; by day and
// at dusk there are none. The runner is the same dark in every sky, so from dusk
// to night only the outline comes in.
func TestRunnerOutlinedByNight(t *testing.T) {
	for _, c := range []struct {
		background string
		lined      bool
	}{{BackgroundNight, true}, {BackgroundDay, false}} {
		d := NewDino(1, RunnerBig, CharacterTRex, SceneGrass, c.background, time.Now)
		d.Draw(76, 31)
		d.obs, d.clouds = nil, nil
		pose := d.runner.figures[0].run[(d.t/3)%2]
		x, y := d.runnerX(0), d.groundY()-pose.h()
		// A cloud just left of the tail: the outline leaves it be.
		d.clouds = []cloud{{x: x - 10, y: y + 5}}
		sc := d.Draw(76, 31)
		at := func(px, py int) uint8 { return sc.Pix[py*sc.W+px] }
		body := func(px, py int) bool {
			return py >= 0 && py < pose.h() && px >= 0 && px < pose.w() && pose[py][px] == '#'
		}
		// The outline: not the body, beside it or at a corner of it, and
		// not under the feet.
		ring := map[[2]int]bool{}
		for py := -1; py < pose.h(); py++ {
			for px := -1; px <= pose.w(); px++ {
				for _, d := range [8][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}} {
					if !body(px, py) && body(px+d[0], py+d[1]) {
						ring[[2]int{px, py}] = true
					}
				}
			}
		}
		for py := -1; py <= pose.h(); py++ {
			for px := -1; px <= pose.w(); px++ {
				want := uint8(0)
				switch {
				case body(px, py):
					want = inkRunner
				case px == -1 && py == 6:
					want = 1 // the cloud
				case px == 8 && py == 1 && c.lined:
					want = inkEye
				case ring[[2]int{px, py}] && c.lined:
					want = inkOutline
				case py == pose.h():
					want = at(x+px, y+py) // the ground under its feet
				}
				if got := at(x+px, y+py); got != want || (py == pose.h() && got == inkOutline) {
					t.Fatalf("%s: the pixel %d,%d of the runner is ink %d, want %d", c.background, px, py, got, want)
				}
			}
		}
		if c.lined && at(x+8, y+1) != inkEye { // the eye
			t.Errorf("%s: the eye is ink %d", c.background, at(x+8, y+1))
		}
		// In the air too: under its feet the sky, beside them the outline.
		if c.lined {
			d.air[0], d.jump[0] = 4, small
			air := d.runner.figures[0].air
			y := d.groundY() - air.h() - d.lift(0)
			sc := d.Draw(76, 31)
			under := 0
			for px := -1; px <= air.w(); px++ {
				if k := sc.Pix[(y+air.h())*76+x+px]; k == inkOutline {
					t.Errorf("in the air, an outline under the feet at %d", px)
				} else if k == 0 {
					under++
				}
			}
			foot := strings.Index(air[air.h()-1], "#")
			if d.lift(0) < 2 || under == 0 || foot < 1 || sc.Pix[(y+air.h()-1)*76+x+foot-1] != inkOutline {
				t.Errorf("in the air, %d up: %d of the sky under its feet, no outline beside them", d.lift(0), under)
			}
		}
	}
	night, dusk, day := nightSky.look(1), duskSky.look(1), daySky.look(1)
	if night.Inks[inkRunner] != "#313244" || night.Inks[inkOutline] != "#cdd6f4" || night.Inks[inkEye] != "#f2b753" ||
		dusk.Inks[inkRunner] != night.Inks[inkRunner] || day.Inks[inkRunner] != night.Inks[inkRunner] ||
		dusk.Inks[inkOutline] != "" || day.Inks[inkOutline] != "" || dusk.Inks[inkEye] != "" || day.Inks[inkEye] != "" {
		t.Errorf("the runner by night %v, at dusk %v, by day %v", night.Inks, dusk.Inks, day.Inks)
	}
}

// The eyes are the ones drawn, marked 'e' (user, 2026-10-07: lit by
// night): the T-Rex, the cat, the rabbit and the giraffe one each — the
// small cat none — the ghost two of four pixels, the small ghost two of
// two; each closed round by the body. The gap between the T-Rex's legs
// mid stride is closed round too, and is no eye but outline.
func TestRunnerEyes(t *testing.T) {
	for name, want := range map[string][2]int{
		CharacterTRex: {1, 1}, CharacterCat: {1, 0}, CharacterRabbit: {1, 1}, CharacterGiraffe: {1, 1}, CharacterGhost: {8, 4},
	} {
		for size, f := range cast[name] {
			for _, pose := range append(f.run[:], f.air) {
				_, eyes := rim(pose)
				if len(eyes) != want[size] {
					t.Errorf("%s, size %d: %d eye pixels", name, size, len(eyes))
				}
				for _, e := range eyes {
					for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
						if c := pose[e[1]+d[1]][e[0]+d[0]]; c != '#' && c != 'e' {
							t.Errorf("%s, size %d: the eye at %v open at %v", name, size, e, d)
						}
					}
				}
			}
		}
	}
	edge, eyes := rim(trex.run[1])
	if !slices.Contains(edge, [2]int{5, 12}) || slices.Contains(eyes, [2]int{5, 12}) {
		t.Errorf("between the legs: outline %v, eye %v", slices.Contains(edge, [2]int{5, 12}), slices.Contains(eyes, [2]int{5, 12}))
	}
}

// The moon keeps clear of the runner (user, 2026-10-07: at the top of a
// jump they ran together): on every scene from the narrowest, the big
// runner's outline is five pixels or more short of it; and the outline
// is a white of its own, a cool one beside the moon's warm, so one that
// does come by it is told from it.
func TestRunnerKeepsClearOfTheMoon(t *testing.T) {
	for w := dinoRoom.W; w <= 3*dinoRoom.W; w++ {
		d := NewDino(1, RunnerBig, CharacterTRex, SceneGrass, BackgroundNight, time.Now)
		d.Draw(w, dinoRoom.H)
		mx, _ := d.moonAt()
		if edge := d.runnerX(0) + d.runner.figures[0].air.w(); mx-edge < 5 {
			t.Errorf("%d wide: the outline to %d, the moon from %d", w, edge, mx)
		}
	}
	out, moon := channels(nightSky.outline), channels(moonColour)
	if nightSky.outline == moonColour || out[2] <= out[0] || moon[2] >= moon[0] {
		t.Errorf("the outline %s, the moon %s", nightSky.outline, moonColour)
	}
}
