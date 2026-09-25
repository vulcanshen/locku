package saver

import (
	"reflect"
	"slices"
	"testing"
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
			if sc.Pix[y*sc.W+x] {
				n++
			}
		}
	}
	return n
}

// The run is endless and no runner ever touches anything: every
// obstacle is jumped — in every scene, by one runner or two — at the
// user's terminal size and at the least the canvas allows.
func TestDinoNeverHitsAnything(t *testing.T) {
	for _, runner := range Runners {
		for _, scene := range Scenes {
			for _, sz := range [][2]int{{76, 31}, {40, 28}, {100, 59}} {
				d := NewDino(42, runner, scene)
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
						t.Fatalf("%s in %s at %dx%d, frame %d: ran into obstacle %d at %d", runner, scene, sz[0], sz[1], i, o.kind, o.x)
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
					if !sc.Pix[(sz[1]-groundH)*sz[0]+x] {
						t.Fatalf("%dx%d: no ground at %d", sz[0], sz[1], x)
					}
				}
				for i := range d.air {
					if n := litIn(sc, d.runnerX(i), 0, d.runnerX(i)+d.runnerW(i), sz[1]-groundH); n < 30 {
						t.Errorf("%s at %dx%d: runner %d is %d pixels", runner, sz[0], sz[1], i, n)
					}
				}
			}
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
		d := NewDino(9, c.name, SceneDesert)
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
	d := NewDino(7, RunnerBig, SceneGrass)
	d.Step()
	if d.t != 0 {
		t.Error("stepped before it was drawn")
	}
	a, b := NewDino(7, RunnerBig, SceneGrass), NewDino(7, RunnerBig, SceneGrass)
	a.Draw(76, 31)
	b.Draw(76, 31)
	for i := 0; i < 300; i++ {
		a.Step()
		b.Step()
	}
	if !reflect.DeepEqual(a.Draw(76, 31), b.Draw(76, 31)) {
		t.Error("two runs from one seed differ")
	}
	if reflect.DeepEqual(a.Draw(76, 31), NewDino(8, RunnerBig, SceneGrass).Draw(76, 31)) {
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
		for _, f := range runnerOf(c.name).figures {
			got = append(got, f.air.w())
		}
		if !reflect.DeepEqual(got, c.widths) {
			t.Errorf("%s: figures %v wide, want %v", c.name, got, c.widths)
		}
	}
	if sceneOf("nonsense").tuftEvery != grassland.tuftEvery || sceneOf(SceneDesert).obstacles[2].h() != 5 {
		t.Error("sceneOf")
	}
	if greatPyramid.w() != 9 || greatPyramid[0] != "....#...." || greatPyramid[4] != "#########" {
		t.Errorf("great pyramid:\n%s", greatPyramid)
	}
}

// Each scene's obstacles come in three sizes — small, medium and large
// (user, 2026-09-25: there had been only small and medium) — each
// tier taller than the one under it, the large one the tallest thing
// in the scene; the jump tops it by the pixel the arc promises, and
// nothing is wider than the thirteen the arc is timed for.
func TestObstaclesComeInThreeTiers(t *testing.T) {
	for _, c := range []struct {
		name    string
		scene   sceneArt
		tiers   []sprite
		heights []int
	}{
		{SceneGrass, grassland, []sprite{cactus, tallCactus, bigCactus}, []int{5, 7, 10}},
		{SceneDesert, desert, []sprite{pyramid, greatPyramid, hugePyramid}, []int{3, 5, 7}},
	} {
		for i, tier := range c.tiers {
			if !slices.ContainsFunc(c.scene.obstacles, func(s sprite) bool { return slices.Equal(s, tier) }) {
				t.Errorf("%s: tier %d is not among the obstacles", c.name, i)
			}
			if tier.h() != c.heights[i] || (i > 0 && tier.h() <= c.tiers[i-1].h()) {
				t.Errorf("%s: tier %d is %d high, want %d and more than the one under it", c.name, i, tier.h(), c.heights[i])
			}
		}
		tallest, widest := 0, 0
		for _, s := range c.scene.obstacles {
			tallest, widest = max(tallest, s.h()), max(widest, s.w())
		}
		if tallest != c.tiers[2].h() || arcTop < tallest+1 || widest > 13 {
			t.Errorf("%s: tallest %d, widest %d, the jump %d high", c.name, tallest, widest, arcTop)
		}
	}
	if len(arc) != 16 || minGap != len(arc)*speed+trex.air.w() {
		t.Errorf("a jump is %d frames, the least gap %d", len(arc), minGap)
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
