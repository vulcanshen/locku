package saver

import (
	"reflect"
	"testing"
)

// hit reports whether any runner's box is in an obstacle this frame.
func hit(d *Dino) (obstacle, bool) {
	dw := d.runner.air.w()
	for i := range d.air {
		dx := d.runnerX(i)
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
			for _, sz := range [][2]int{{76, 31}, {40, 25}, {100, 59}} {
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
				if jumps < 20*d.runner.count || seen == 0 {
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
					if n := litIn(sc, d.runnerX(i), 0, d.runnerX(i)+d.runner.air.w(), sz[1]-groundH); n < 40 {
						t.Errorf("%s at %dx%d: runner %d is %d pixels", runner, sz[0], sz[1], i, n)
					}
				}
			}
		}
	}
}

// Two runners stand one behind the other and jump on their own: over a
// long run they are in the air at different times.
func TestTwoRunnersJumpOnTheirOwn(t *testing.T) {
	d := NewDino(9, RunnerTwoTRex, SceneDesert)
	d.Draw(76, 31)
	if len(d.air) != 2 || d.runnerX(1) != d.runnerX(0)+d.runner.air.w()+runnerGap {
		t.Fatalf("runners at %d and %d", d.runnerX(0), d.runnerX(1))
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
		t.Errorf("in the air apart %d frames, together %d: they must jump each on their own, and cross", apart, together)
	}
}

// Nothing runs before the first draw, and a run is its seed.
func TestDinoIsItsSeed(t *testing.T) {
	d := NewDino(7, RunnerTRex, SceneGrass)
	d.Step()
	if d.t != 0 {
		t.Error("stepped before it was drawn")
	}
	a, b := NewDino(7, RunnerTRex, SceneGrass), NewDino(7, RunnerTRex, SceneGrass)
	a.Draw(76, 31)
	b.Draw(76, 31)
	for i := 0; i < 300; i++ {
		a.Step()
		b.Step()
	}
	if !reflect.DeepEqual(a.Draw(76, 31), b.Draw(76, 31)) {
		t.Error("two runs from one seed differ")
	}
	if reflect.DeepEqual(a.Draw(76, 31), NewDino(8, RunnerTRex, SceneGrass).Draw(76, 31)) {
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

// The art the names pick, and the shapes: an unknown name is the first
// choice; the pyramids are stepped, widest at the ground.
func TestArtByName(t *testing.T) {
	if runnerOf("nonsense").count != 1 || runnerOf(RunnerTwoTRex).count != 2 {
		t.Error("runnerOf")
	}
	if sceneOf("nonsense").tuftEvery != grassland.tuftEvery || sceneOf(SceneDesert).obstacles[2].h() != 5 {
		t.Error("sceneOf")
	}
	if greatPyramid.w() != 9 || greatPyramid[0] != "....#...." || greatPyramid[4] != "#########" {
		t.Errorf("great pyramid:\n%s", greatPyramid)
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
