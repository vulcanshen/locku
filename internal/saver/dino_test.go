package saver

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

// hit reports whether any runner's box is in an obstacle this frame.
func hit(d *Dino) (obstacle, bool) {
	for i := range d.air {
		dx, dw := d.runnerX(i), d.runnerW(i)
		for _, o := range d.obs {
			s := d.scene.obstacles[o.kind]
			if o.x < dx+dw && o.x+s.w() > dx && d.lift(i) < s.h() {
				return o, true
			}
		}
	}
	return obstacle{}, false
}

// litIn counts the scene's lit pixels inside a box.
func litIn(sc Scene, x0, y0, x1, y1 int) int {
	n := 0
	for y := max(0, y0); y < min(sc.H, y1); y++ {
		for x := max(0, x0); x < min(sc.W, x1); x++ {
			if sc.Pix[y*sc.W+x] != 0 {
				n++
			}
		}
	}
	return n
}

// lit is how many of a sprite's pixels are lit.
func lit(sp sprite) int {
	n := 0
	for _, row := range sp {
		for i := 0; i < len(row); i++ {
			if row[i] == '#' {
				n++
			}
		}
	}
	return n
}

// The run is endless and no runner ever touches anything: every
// obstacle is jumped — in every scene, by one runner or two, as every
// character (2026-10-06) — at the user's terminal size and at the least
// the canvas allows.
func TestDinoNeverHitsAnything(t *testing.T) {
	for _, character := range Characters {
		for _, runner := range Participants {
			for _, scene := range Scenes {
				for _, sz := range [][2]int{{76, 31}, {40, 28}, {100, 59}} {
					runAndWatch(t, character, runner, scene, sz)
				}
			}
		}
	}
}

// runAndWatch runs one dino for 6000 frames: it never runs into
// anything, it jumps, and it is drawn on its ground.
func runAndWatch(t *testing.T, character, runner, scene string, sz [2]int) {
	t.Helper()
	d := NewDino(42, runner, character, scene, BackgroundNight, time.Now)
	d.Draw(sz[0], sz[1])
	jumps, seen := 0, 0
	for i := 0; i < 6000; i++ {
		d.Step()
		for _, a := range d.air {
			if a == 0 {
				jumps++
			}
		}
		seen = max(seen, len(d.obs))
		if o, ok := hit(d); ok {
			t.Fatalf("%s %s in %s at %dx%d, frame %d: ran into obstacle %d at %d", runner, character, scene, sz[0], sz[1], i, o.kind, o.x)
		}
	}
	if jumps < 20*d.runner.count() || seen == 0 {
		t.Errorf("%s in %s at %dx%d: %d jumps, %d obstacles at most", runner, scene, sz[0], sz[1], jumps, seen)
	}
	sc := d.Draw(sz[0], sz[1])
	if len(sc.Pix) != sz[0]*sz[1] {
		t.Errorf("%dx%d: scene %d px", sz[0], sz[1], len(sc.Pix))
	}
	// The ground line runs the whole width, two rows up from
	// the bottom, and every runner stands on it.
	for x := 0; x < sz[0]; x++ {
		if sc.Pix[(sz[1]-groundH)*sz[0]+x] == 0 {
			t.Fatalf("%dx%d: no ground at %d", sz[0], sz[1], x)
		}
	}
	for i, f := range d.runner.figures {
		least := min(lit(f.run[0]), lit(f.run[1]), lit(f.air))
		if n := litIn(sc, d.runnerX(i), 0, d.runnerX(i)+d.runnerW(i), sz[1]-groundH); n < least {
			t.Errorf("%s %s at %dx%d: runner %d is %d pixels, its least pose %d", runner, character, sz[0], sz[1], i, n, least)
		}
	}
}

// Two runners stand one behind the other in the order the name reads
// — small-big is the small one behind, the big one in front — and
// jump on their own: over a long run they are in the air at different
// times.
func TestTwoRunnersJumpOnTheirOwn(t *testing.T) {
	for _, c := range []struct {
		name   string
		w0, w1 int
	}{
		{RunnerBigBig, 12, 12},
		{RunnerSmallSmall, 8, 8},
		{RunnerSmallBig, 8, 12},
		{RunnerBigSmall, 12, 8},
	} {
		d := NewDino(9, c.name, CharacterTRex, SceneDesert, BackgroundNight, time.Now)
		d.Draw(76, 31)
		if len(d.air) != 2 || d.runnerX(1) != d.runnerX(0)+d.runnerW(0)+runnerGap || d.runnerW(0) != c.w0 || d.runnerW(1) != c.w1 {
			t.Fatalf("%s: runners at %d (%d wide) and %d (%d wide)", c.name, d.runnerX(0), d.runnerW(0), d.runnerX(1), d.runnerW(1))
		}
		apart, together := 0, 0
		for i := 0; i < 6000; i++ {
			d.Step()
			a, b := d.air[0] >= 0, d.air[1] >= 0
			switch {
			case a != b:
				apart++
			case a && b:
				together++
			}
		}
		if apart == 0 || together == 0 {
			t.Errorf("%s: in the air apart %d frames, together %d: they must jump each on their own, and cross", c.name, apart, together)
		}
	}
}

// Nothing runs before the first draw, and a run is its seed.
func TestDinoIsItsSeed(t *testing.T) {
	d := NewDino(7, RunnerBig, CharacterTRex, SceneGrass, BackgroundNight, time.Now)
	d.Step()
	if d.t != 0 {
		t.Error("stepped before it was drawn")
	}
	a, b := NewDino(7, RunnerBig, CharacterTRex, SceneGrass, BackgroundNight, time.Now), NewDino(7, RunnerBig, CharacterTRex, SceneGrass, BackgroundNight, time.Now)
	a.Draw(76, 31)
	b.Draw(76, 31)
	for i := 0; i < 300; i++ {
		a.Step()
		b.Step()
	}
	if !reflect.DeepEqual(a.Draw(76, 31), b.Draw(76, 31)) {
		t.Error("two runs from one seed differ")
	}
	if reflect.DeepEqual(a.Draw(76, 31), NewDino(8, RunnerBig, CharacterTRex, SceneGrass, BackgroundNight, time.Now).Draw(76, 31)) {
		t.Error("a different seed is the same run")
	}
	// A resize keeps the run going, the clouds back in the sky.
	c := a.Draw(40, 25)
	if c.W != 40 || c.H != 25 || len(c.Pix) != 1000 {
		t.Errorf("resized scene %dx%d", c.W, c.H)
	}
	for _, cl := range a.clouds {
		if cl.y < 1 || cl.y >= a.groundY() {
			t.Errorf("cloud at %d after the resize", cl.y)
		}
	}
}

// The art the names pick, and the shapes: a runner's figures are the
// name's, back to front, twelve wide for a big T-Rex and eight for a
// small one; an unknown name is the first choice; the pyramids are
// stepped, widest at the ground.
func TestArtByName(t *testing.T) {
	for _, c := range []struct {
		name   string
		widths []int
	}{
		{RunnerBig, []int{12}},
		{RunnerSmall, []int{8}},
		{RunnerBigBig, []int{12, 12}},
		{RunnerSmallSmall, []int{8, 8}},
		{RunnerSmallBig, []int{8, 12}},
		{RunnerBigSmall, []int{12, 8}},
		{"nonsense", []int{12}},
	} {
		var got []int
		for _, f := range runnerOf(c.name, CharacterTRex).figures {
			got = append(got, f.air.w())
		}
		if !reflect.DeepEqual(got, c.widths) {
			t.Errorf("%s: figures %v wide, want %v", c.name, got, c.widths)
		}
	}
	if sceneOf("nonsense").tuftEvery != grassland.tuftEvery || sceneOf(SceneDesert).obstacles[2].h() != 5 || sceneOf(SceneCity).obstacles[2].h() != 10 {
		t.Error("sceneOf")
	}
	if greatPyramid.w() != 9 || greatPyramid[0] != "....#...." || greatPyramid[4] != "#########" {
		t.Errorf("great pyramid:\n%s", greatPyramid)
	}
}

// Each scene's obstacles come in three sizes — small, medium and large
// (user, 2026-09-25: there had been only small and medium) — each
// tier taller than the one under it, the large one the tallest thing
// in the scene, and every obstacle marked with its tier. A jump a tier
// (user, the same day: the height had been the same over everything),
// each higher than the one under it, each topping the tallest of its
// tier in either scene by the one pixel the arcs promise; none longer
// than the large tier's sixteen frames, which the least gap is timed
// to; the sky starts over the highest; and nothing is wider than the
// thirteen the arcs are timed for.
func TestObstaclesComeInThreeTiers(t *testing.T) {
	var tallest [3]int // of each tier, in either scene
	for _, c := range []struct {
		name    string
		scene   sceneArt
		tiers   []sprite
		heights []int
	}{
		{SceneGrass, grassland, []sprite{cactus, tallCactus, bigCactus}, []int{5, 7, 10}},
		{SceneDesert, desert, []sprite{pyramid, greatPyramid, hugePyramid}, []int{3, 5, 7}},
		{SceneCity, city, []sprite{bungalow, block, skyscraper}, []int{5, 7, 10}},
	} {
		for i, art := range c.tiers {
			if !slices.ContainsFunc(c.scene.obstacles, func(o obstacleArt) bool { return slices.Equal(o.sprite, art) && o.tier == tier(i) }) {
				t.Errorf("%s: tier %d is not among the obstacles as that tier", c.name, i)
			}
			if art.h() != c.heights[i] || (i > 0 && art.h() <= c.tiers[i-1].h()) {
				t.Errorf("%s: tier %d is %d high, want %d and more than the one under it", c.name, i, art.h(), c.heights[i])
			}
		}
		widest := 0
		for _, o := range c.scene.obstacles {
			if o.tier > large || o.h() > c.tiers[2].h() {
				t.Errorf("%s: an obstacle %d high of tier %d", c.name, o.h(), o.tier)
			}
			tallest[o.tier] = max(tallest[o.tier], o.h())
			widest = max(widest, o.w())
		}
		if widest > 13 {
			t.Errorf("%s: widest %d", c.name, widest)
		}
	}
	for tr := small; tr <= large; tr++ {
		top := slices.Max(arcs[tr])
		if top != tallest[tr]+1 || (tr > small && top <= slices.Max(arcs[tr-1])) || len(arcs[tr]) > len(arcs[large]) {
			t.Errorf("tier %d: tallest %d, the jump %d high, %d frames", tr, tallest[tr], top, len(arcs[tr]))
		}
	}
	if len(arcs[large]) != 16 || arcTop != slices.Max(arcs[large]) || minGap != len(arcs[large])*speed+trex.air.w() {
		t.Errorf("the large jump is %d frames and %d high, the sky from %d, the least gap %d", len(arcs[large]), slices.Max(arcs[large]), arcTop, minGap)
	}
}

// The jump is the obstacle's size (user, 2026-09-25: the height had
// been the same over everything): over a small one the low arc, over a
// medium one the middling, over a large one the high — in either scene,
// by a big T-Rex or a small one, each kind of obstacle put in its way
// alone: the runner takes the arc of its tier, peaks at that arc's top,
// and clears it.
func TestJumpIsTheObstaclesSize(t *testing.T) {
	for _, scene := range Scenes {
		for _, runner := range []string{RunnerBig, RunnerSmall} {
			for kind, o := range sceneOf(scene).obstacles {
				d := NewDino(3, runner, CharacterTRex, scene, BackgroundNight, time.Now)
				d.Draw(76, 31)
				d.gap = 1000 // nothing else is due
				d.obs = []obstacle{{x: 60, kind: kind}}
				peak, jumped := 0, false
				for i := 0; !jumped || d.air[0] >= 0; i++ {
					if i > 100 {
						t.Fatalf("%s in %s: no jump over obstacle %d", runner, scene, kind)
					}
					d.Step()
					if d.air[0] == 0 {
						jumped = true
						if d.jump[0] != o.tier {
							t.Errorf("%s in %s over obstacle %d, tier %d: the tier %d jump", runner, scene, kind, o.tier, d.jump[0])
						}
					}
					peak = max(peak, d.lift(0))
					if _, ok := hit(d); ok {
						t.Fatalf("%s in %s: ran into obstacle %d", runner, scene, kind)
					}
				}
				if peak != slices.Max(arcs[o.tier]) {
					t.Errorf("%s in %s over obstacle %d, tier %d: peaked at %d, want %d", runner, scene, kind, o.tier, peak, slices.Max(arcs[o.tier]))
				}
			}
		}
	}
}

// The city's windows (user, 2026-10-08): some of each building's are lit
// at night, in the gold of the runner's eyes; the rest are dark, the sky
// through them, and so are all of them by day and at dusk, which have no
// eyes to light (TestRunnerInks).
func TestCityWindowsLightAtNight(t *testing.T) {
	for kind, want := range []int{2, 7, 6} {
		o := city.obstacles[kind]
		d := NewDino(3, RunnerSmall, CharacterTRex, SceneCity, BackgroundNight, time.Now)
		d.Draw(76, 31)
		d.obs = []obstacle{{x: 50, kind: kind}}
		sc := d.Draw(76, 31)
		top, lit, dark := d.groundY()-o.h(), 0, 0
		for y, row := range o.sprite {
			for x := range row {
				got, want := sc.Pix[(top+y)*sc.W+50+x], map[byte]uint8{'#': 1, '.': 0, 'w': inkEye}[row[x]]
				if got != want {
					t.Errorf("obstacle %d at %d,%d: ink %d, want %d", kind, x, y, got, want)
				}
				if row[x] == 'w' {
					lit++
				}
				if row[x] == '.' && x > 0 && x < len(row)-1 && row[x-1] != '.' && row[x+1] != '.' && y < len(o.sprite)-1 {
					dark++ // inside, above the door's row
				}
			}
		}
		if lit != want || dark == 0 {
			t.Errorf("obstacle %d: %d windows lit, want %d, and %d dark, want some", kind, lit, want, dark)
		}
	}
}

func TestBesideAlignsTheBottoms(t *testing.T) {
	got := beside(1, cactus, tallCactus)
	if got.w() != 9 || got.h() != 7 {
		t.Fatalf("%dx%d", got.w(), got.h())
	}
	if got[0] != "......#.." || got[6] != ".#....#.." {
		t.Errorf("rows:\n%s\n%s", got[0], got[6])
	}
}

// Every character comes big and small (user, 2026-10-06): each as wide
// as the T-Rex of its size, twelve or eight, which the jumps are timed
// to, and no taller than it; its three poses one height, and its two
// running ones not the same, or it would not run. A runner's figures
// are the character's, in the runner's sizes; an unknown character is
// the T-Rex.
func TestEveryCharacterFitsTheRun(t *testing.T) {
	for _, c := range Characters {
		for i, f := range cast[c] {
			w, h := trex.air.w(), trex.air.h()
			if i == 1 {
				w, h = smallTRex.air.w(), smallTRex.air.h()
			}
			for _, p := range []sprite{f.run[0], f.run[1], f.air} {
				if p.h() != f.air.h() || p.h() > h {
					t.Errorf("%s %d: a pose %d tall, the air %d, at most %d", c, i, p.h(), f.air.h(), h)
				}
				for _, row := range p {
					if len(row) != w {
						t.Fatalf("%s %d: a row %d wide, want %d: %q", c, i, len(row), w, row)
					}
				}
			}
			if reflect.DeepEqual(f.run[0], f.run[1]) {
				t.Errorf("%s %d: its two running poses are the same", c, i)
			}
		}
	}
	if got := runnerOf(RunnerSmallBig, CharacterCat).figures; !reflect.DeepEqual(got, []figure{smallCat, cat}) {
		t.Errorf("small-big cats: %v", got)
	}
	if got := runnerOf(RunnerBig, "dragon").figures; !reflect.DeepEqual(got, []figure{trex}) {
		t.Errorf("an unknown character: %v", got)
	}
}

// Who runs, by name (user, 2026-10-06): the giraffe was the horse till
// the day after (user: it is more like a giraffe), drawn the same.
func TestRunnerCharacters(t *testing.T) {
	if !slices.Equal(Characters, []string{"t-rex", "cat", "rabbit", "giraffe", "ghost"}) {
		t.Errorf("characters %v", Characters)
	}
}
