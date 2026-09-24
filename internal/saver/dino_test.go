package saver

import (
	"reflect"
	"testing"
)

// hit reports whether the runner's box is in an obstacle this frame.
func hit(d *Dino) (obstacle, bool) {
	dx, dw := d.runnerX(), d.runner.air.w()
	for _, o := range d.obs {
		s := d.scene.obstacles[o.kind]
		if o.x < dx+dw && o.x+s.w() > dx && d.lift() < s.h() {
			return o, true
		}
	}
	return obstacle{}, false
}

// The run is endless and the runner never touches anything: every
// obstacle is jumped, at the user's terminal size and at the least the
// canvas allows.
func TestDinoNeverHitsAnything(t *testing.T) {
	for _, sz := range [][2]int{{76, 31}, {40, 25}, {100, 59}} {
		d := NewDino(42, RunnerTRex, SceneGrass)
		d.Draw(sz[0], sz[1])
		jumps, seen := 0, 0
		for i := 0; i < 6000; i++ {
			d.Step()
			if d.air == 0 {
				jumps++
			}
			seen = max(seen, len(d.obs))
			if o, ok := hit(d); ok {
				t.Fatalf("%dx%d frame %d: ran into obstacle %d at %d, lift %d", sz[0], sz[1], i, o.kind, o.x, d.lift())
			}
		}
		if jumps < 20 || seen == 0 {
			t.Errorf("%dx%d: %d jumps, %d obstacles at most", sz[0], sz[1], jumps, seen)
		}
		sc := d.Draw(sz[0], sz[1])
		if len(sc.Pix) != sz[0]*sz[1] || sc.Score != d.dist/10 || sc.Score == 0 {
			t.Errorf("%dx%d: scene %d px, score %d", sz[0], sz[1], len(sc.Pix), sc.Score)
		}
		// The ground line runs the whole width, two rows up from the
		// bottom, and the runner stands on it.
		for x := 0; x < sz[0]; x++ {
			if !sc.Pix[(sz[1]-groundH)*sz[0]+x] {
				t.Fatalf("%dx%d: no ground at %d", sz[0], sz[1], x)
			}
		}
		lit := 0
		for y := 0; y < sz[1]-groundH; y++ {
			for x := d.runnerX(); x < d.runnerX()+d.runner.air.w(); x++ {
				if sc.Pix[y*sz[0]+x] {
					lit++
				}
			}
		}
		if lit < 40 {
			t.Errorf("%dx%d: the runner is %d pixels", sz[0], sz[1], lit)
		}
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

func TestBesideAlignsTheBottoms(t *testing.T) {
	got := beside(1, cactus, tallCactus)
	if got.w() != 9 || got.h() != 7 {
		t.Fatalf("%dx%d", got.w(), got.h())
	}
	if got[0] != "......#.." || got[6] != ".#....#.." {
		t.Errorf("rows:\n%s\n%s", got[0], got[6])
	}
}
