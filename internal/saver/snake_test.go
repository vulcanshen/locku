package saver

import (
	"reflect"
	"testing"
)

func adjacent(cw, a, b int) bool {
	ax, ay, bx, by := a%cw, a/cw, b%cw, b/cw
	d := (ax-bx)*(ax-bx) + (ay-by)*(ay-by)
	return d == 1
}

// The cycle goes through every cell once — but the last column when
// neither side is even — each step to a neighbour, and the last back to
// the first; a grid too small has none.
func TestHamiltonIsACycle(t *testing.T) {
	for cw := 1; cw <= 12; cw++ {
		for ch := 1; ch <= 12; ch++ {
			tour := hamilton(cw, ch)
			used := cw
			if cw%2 != 0 && ch%2 != 0 {
				used = cw - 1
			}
			want := used * ch
			if used < 2 || ch < 2 {
				if tour != nil {
					t.Errorf("%dx%d: a cycle of %d where there is none", cw, ch, len(tour))
				}
				continue
			}
			if len(tour) != want {
				t.Fatalf("%dx%d: %d cells, want %d", cw, ch, len(tour), want)
			}
			seen := map[int]bool{}
			for i, c := range tour {
				if seen[c] {
					t.Fatalf("%dx%d: cell %d twice", cw, ch, c)
				}
				seen[c] = true
				if cw%2 != 0 && ch%2 != 0 && c%cw == cw-1 {
					t.Fatalf("%dx%d: the left-out column is on it", cw, ch)
				}
				if next := tour[(i+1)%len(tour)]; !adjacent(cw, c, next) {
					t.Fatalf("%dx%d: %d then %d", cw, ch, c, next)
				}
			}
		}
	}
}

// check is the game's state as it must always be: the body on distinct
// cells, each beside the next, marked taken and nothing else; the apple
// on an empty cell of the cycle.
func check(t *testing.T, s *Snake, step int) {
	t.Helper()
	on := map[int]bool{}
	for i, c := range s.body {
		if on[c] {
			t.Fatalf("step %d: the body crosses itself at %d", step, c)
		}
		on[c] = true
		if s.pos[c] < 0 {
			t.Fatalf("step %d: off the cycle at %d", step, c)
		}
		if i > 0 && !adjacent(s.cw, s.body[i-1], c) {
			t.Fatalf("step %d: segments %d and %d apart", step, s.body[i-1], c)
		}
	}
	for c, k := range s.taken {
		if k != on[c] {
			t.Fatalf("step %d: cell %d taken %v, body %v", step, c, k, on[c])
		}
	}
	// Along the cycle the body lies behind the head in order: from the
	// tail forward through every segment to the head, and on to the tail
	// again, is once round.
	round := s.ahead(s.body[0], s.body[len(s.body)-1])
	for i := 1; i < len(s.body); i++ {
		round += s.ahead(s.body[i], s.body[i-1])
	}
	if len(s.body) > 1 && round != len(s.tour) {
		t.Fatalf("step %d: the body is out of the cycle's order (%d round %d)", step, round, len(s.tour))
	}
	if s.apple >= 0 && (on[s.apple] || s.pos[s.apple] < 0) {
		t.Fatalf("step %d: the apple at %d", step, s.apple)
	}
}

// It plays the board full without ever running into itself, holds the
// full board, and starts again.
func TestSnakeFillsTheBoardAndStartsAgain(t *testing.T) {
	for i, sz := range [][2]int{{15, 11}, {21, 13}, {40, 23}, {31, 19}, {16, 9}, {15, 11}, {21, 13}, {31, 19}, {16, 9}, {11, 7}} {
		s := NewSnake(uint64(4 + i))
		s.Draw(sz[0], sz[1])
		n := len(s.tour)
		if len(s.body) != snakeStart || n == 0 {
			t.Fatalf("%v: start %d long on %d cells", sz, len(s.body), n)
		}
		steps := 0
		for s.apple >= 0 {
			s.Step()
			steps++
			check(t, s, steps)
			if steps > n*n {
				t.Fatalf("%v: %d long after %d steps", sz, len(s.body), steps)
			}
		}
		if len(s.body) != n {
			t.Fatalf("%v: no apple, %d of %d", sz, len(s.body), n)
		}
		// The full board sits in the middle of the scene, the dark left
		// over halved as near as it goes.
		sc := s.Draw(sz[0], sz[1])
		x0, y0, x1, y1, lit := sc.W, sc.H, -1, -1, 0
		for y := 0; y < sc.H; y++ {
			for x := 0; x < sc.W; x++ {
				if sc.Pix[y*sc.W+x] != 0 {
					x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
					lit++
				}
			}
		}
		if lit != 2*n-1 {
			t.Errorf("%v: %d lit for a full board of %d: some of it is off the scene", sz, lit, n)
		}
		if l, r, u, d := x0, sc.W-1-x1, y0, sc.H-1-y1; l-r > 1 || r-l > 1 || u-d > 1 || d-u > 1 {
			t.Errorf("%v: the full board has %d, %d dark either side and %d, %d above and below", sz, l, r, u, d)
		}
		for i := 0; i < snakeHold-1; i++ {
			s.Step()
			if len(s.body) != n {
				t.Fatalf("%v: the full board went after %d frames", sz, i+1)
			}
		}
		s.Step()
		if len(s.body) != snakeStart || s.apple < 0 {
			t.Fatalf("%v: no new game: %d long", sz, len(s.body))
		}
		t.Logf("%v: %d cells in %d steps", sz, n, steps)
	}
}

// Short cuts while the board is mostly empty: a game takes fewer steps
// than following the cycle for every apple would.
func TestSnakeCutsWhileThereIsRoom(t *testing.T) {
	s := NewSnake(11)
	s.Draw(40, 23)
	n := len(s.tour)
	steps := 0
	for len(s.body) < n/4 {
		s.Step()
		steps++
	}
	// Along the cycle each apple is n/2 away on average.
	if eaten := len(s.body) - snakeStart; steps > eaten*n/4 {
		t.Errorf("%d steps for %d apples on %d cells: no short cuts", steps, eaten, n)
	}
}

// A frame lights each cell of the body, the link between each two, and
// the apple every other few frames.
func TestSnakeDrawsNodesLinksAndABlinkingApple(t *testing.T) {
	s := NewSnake(6)
	s.Draw(40, 23)
	for i := 0; i < 30; i++ {
		s.Step()
	}
	lit := func(sc Scene) int {
		n := 0
		for _, k := range sc.Pix {
			if k != 0 {
				n++
			}
		}
		return n
	}
	seen := map[int]bool{}
	for i := 0; i < 2*snakeBlink; i++ {
		sc := s.Draw(40, 23)
		body := 2*len(s.body) - 1
		switch got := lit(sc); got {
		case body, body + 1:
			seen[got-body] = true
		default:
			t.Fatalf("frame %d: %d lit for %d segments", i, got, len(s.body))
		}
		for _, c := range s.body {
			x, y := s.cellAt(c)
			if sc.Pix[y*sc.W+x] != 1 || (x-s.ox)%2 != 0 || (y-s.oy)%2 != 0 {
				t.Fatalf("frame %d: node %d at %d,%d", i, c, x, y)
			}
		}
		s.t++
	}
	if !seen[0] || !seen[1] {
		t.Errorf("the apple is lit %v", seen)
	}
}

// The same seed is the same game; a new size is a new one, and a scene
// too small has none and draws nothing.
func TestSnakeIsTheSeedsAndStartsOverOnAResize(t *testing.T) {
	a, b := NewSnake(9), NewSnake(9)
	for i := 0; i < 300; i++ {
		if !reflect.DeepEqual(a.Draw(40, 23), b.Draw(40, 23)) {
			t.Fatalf("frame %d differs", i)
		}
		a.Step()
		b.Step()
	}
	a.Draw(50, 29)
	if a.cw != 25 || a.ch != 15 || len(a.body) != snakeStart {
		t.Errorf("after a resize: %dx%d, %d long", a.cw, a.ch, len(a.body))
	}
	tiny := NewSnake(1)
	sc := tiny.Draw(1, 5)
	tiny.Step()
	for _, k := range sc.Pix {
		if k != 0 {
			t.Fatal("a scene too small lit a pixel")
		}
	}
}
