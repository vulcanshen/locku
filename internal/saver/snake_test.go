package saver

import (
	"reflect"
	"slices"
	"testing"
	"time"
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
// cells, each beside the next, marked taken and nothing else, lumps on
// the body alone; the apple on an empty cell of the cycle, in a colour
// not the snake's.
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
	for _, i := range s.lumps {
		if i < 0 || i >= len(s.body) {
			t.Fatalf("step %d: a lump at %d, off a body %d long", step, i, len(s.body))
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
	if s.next == s.colour {
		t.Fatalf("step %d: the apple is the snake's colour", step)
	}
}

// It plays the board full without ever running into itself, holds the
// full board, and starts again; an apple eaten turns the snake its
// colour, leaves a lump where it was, and the next is in another.
func TestSnakeFillsTheBoardAndStartsAgain(t *testing.T) {
	for i, sz := range [][2]int{{15, 11}, {21, 13}, {40, 23}, {31, 19}, {16, 9}, {76, 31}, {50, 29}, {49, 31}, {11, 7}, {100, 59}} {
		s := NewSnake(uint64(4+i), SnakeSpeedDefault)
		s.Draw(sz[0], sz[1])
		n := len(s.tour)
		if len(s.body) != snakeStart || n == 0 {
			t.Fatalf("%v: start %d long on %d cells", sz, len(s.body), n)
		}
		steps := 0
		for s.apple >= 0 {
			long, colour, next := len(s.body), s.colour, s.next
			s.Step()
			steps++
			check(t, s, steps)
			if len(s.body) > long && (s.colour != next || !slices.Contains(s.lumps, 0)) {
				t.Fatalf("%v step %d: ate, and is colour %d (the apple was %d), lumps %v", sz, steps, s.colour, next, s.lumps)
			}
			if len(s.body) == long && (s.colour != colour || s.next != next) {
				t.Fatalf("%v step %d: the colours changed with nothing eaten", sz, steps)
			}
			if steps > n*n {
				t.Fatalf("%v: %d long after %d steps", sz, len(s.body), steps)
			}
		}
		if len(s.body) != n {
			t.Fatalf("%v: no apple, %d of %d", sz, len(s.body), n)
		}
		// The full board sits in the middle of the scene with a pixel
		// round it for the jaw and the crown, the dark left over halved
		// as near as it goes.
		used := s.cw
		if s.cw%2 != 0 && s.ch%2 != 0 {
			used--
		}
		l, r := s.ox-1, sz[0]-1-(s.ox+snakePitch*(used-1))-1
		u, d := s.oy-1, sz[1]-1-(s.oy+snakePitch*(s.ch-1))-1
		if l < 0 || r < 0 || u < 0 || d < 0 || l-r > 1 || r-l > 1 || u-d > 1 || d-u > 1 {
			t.Errorf("%v: %d, %d dark either side and %d, %d above and below, past the head's room", sz, l, r, u, d)
		}
		for j := 0; j < s.frames(snakeHold)-1; j++ {
			before := slices.Clone(s.lumps)
			s.Step()
			if len(s.body) != n {
				t.Fatalf("%v: the full board went after %d frames", sz, j+1)
			}
			// The body is still; the lumps run on, a place a frame.
			var want []int
			for _, i := range before {
				if i+1 < n {
					want = append(want, i+1)
				}
			}
			if !slices.Equal(s.lumps, want) {
				t.Fatalf("%v: the lumps %v on a full board, then %v", sz, before, s.lumps)
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
	s := NewSnake(11, SnakeSpeedDefault)
	s.Draw(76, 31)
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

// The head, turned for each way the snake goes (user, 2026-10-06).
// Shut: the snout a pixel ahead, the node, the link behind, and the
// crown two pixels over the node and the link — above along a row, to
// the right up or down a column — nothing over the snout. Open: the
// line on to the snout, the lips over and under the two pixels ahead of
// the node, the mouth between them dark next to the apple, the crown a
// pixel back. Swallowing: shut, and a pixel of lump behind the crown. A
// lump is the crown's two pixels.
func TestSnakeHeadTurnsWithTheWay(t *testing.T) {
	set := func(sh shape, dx, dy int) map[[2]int]bool {
		at := map[[2]int]bool{}
		for _, p := range sh {
			x, y := turn(p, dx, dy)
			at[[2]int{x, y}] = true
		}
		return at
	}
	for _, c := range []struct {
		dx, dy         int
		crown1, crown2 [2]int
	}{
		{-1, 0, [2]int{0, -1}, [2]int{1, -1}},
		{1, 0, [2]int{0, -1}, [2]int{-1, -1}},
		{0, -1, [2]int{1, 0}, [2]int{1, 1}},
		{0, 1, [2]int{1, 0}, [2]int{1, -1}},
	} {
		dx, dy := c.dx, c.dy
		back := func(p [2]int, by int) [2]int { return [2]int{p[0] - by*dx, p[1] - by*dy} }
		across := func(p [2]int) [2]int {
			if dy == 0 {
				return [2]int{p[0], -p[1]}
			}
			return [2]int{-p[0], p[1]}
		}
		snout, node, link1, link2 := [2]int{dx, dy}, [2]int{0, 0}, [2]int{-dx, -dy}, [2]int{-2 * dx, -2 * dy}
		mouth := [2]int{dx + c.crown1[0], dy + c.crown1[1]}

		shut := set(snakeHead, dx, dy)
		if len(shut) != 6 || !shut[snout] || !shut[node] || !shut[link1] || !shut[link2] || !shut[c.crown1] || !shut[c.crown2] || shut[mouth] {
			t.Errorf("going %d,%d: shut, the head is %v", dx, dy, shut)
		}
		open := set(snakeOpen, dx, dy)
		lip1, lip2 := back(c.crown1, -2), back(c.crown2, -2)
		if len(open) != 10 || !open[snout] || open[[2]int{2 * dx, 2 * dy}] || !open[node] || !open[link1] || !open[link2] ||
			!open[lip1] || !open[lip2] || !open[across(lip1)] || !open[across(lip2)] || !open[back(c.crown1, 1)] || !open[back(c.crown2, 1)] || open[c.crown1] {
			t.Errorf("going %d,%d: open, the head is %v", dx, dy, open)
		}
		swallow := set(snakeSwallow, dx, dy)
		delete(swallow, back(c.crown2, 3))
		if !reflect.DeepEqual(swallow, shut) {
			t.Errorf("going %d,%d: swallowing is not the shut head and a pixel of lump: %v", dx, dy, swallow)
		}
		if lump := set(snakeLump, dx, dy); len(lump) != 2 || !lump[c.crown1] || !lump[c.crown2] {
			t.Errorf("going %d,%d: the lump is %v", dx, dy, lump)
		}
	}
}

// A frame is the snake in its colour — every node, the two pixels of
// every link, its head, its lumps — and the apple in the next colour
// every other few frames.
func TestSnakeDrawsItsBodyHeadLumpsAndApple(t *testing.T) {
	s := NewSnake(6, SnakeSpeedDefault)
	s.Draw(76, 31)
	for i := 0; i < 200; i++ {
		s.Step()
	}
	// A lump a few segments down, wherever the last apple is.
	s.lumps = []int{2}
	ink := uint8(1 + s.colour)
	at := func(sc Scene, x, y int) uint8 { return sc.Pix[y*sc.W+x] }
	lit := map[bool]bool{}
	for f := 0; f < 2*s.frames(snakeBlink); f++ {
		sc := s.Draw(76, 31)
		for i, c := range s.body {
			x, y := s.cellAt(c)
			if i > 0 && at(sc, x, y) != ink {
				t.Fatalf("frame %d: node %d is ink %d", f, i, at(sc, x, y))
			}
			if i > 0 {
				px, py := s.cellAt(s.body[i-1])
				for k := 1; k < snakePitch; k++ {
					if at(sc, x+(px-x)*k/snakePitch, y+(py-y)*k/snakePitch) != ink {
						t.Fatalf("frame %d: the link to segment %d", f, i)
					}
				}
			}
		}
		hx, hy := s.cellAt(s.body[0])
		dx, dy := s.way(s.body[1], s.body[0])
		for _, p := range s.head() {
			px, py := turn(p, dx, dy)
			if at(sc, hx+px, hy+py) != ink {
				t.Fatalf("frame %d: the head at %d,%d", f, px, py)
			}
		}
		lx, ly := s.cellAt(s.body[2])
		ldx, ldy := s.way(s.body[2], s.body[1])
		for _, p := range snakeLump {
			px, py := turn(p, ldx, ldy)
			if at(sc, lx+px, ly+py) != ink {
				t.Fatalf("frame %d: the lump at %d,%d", f, px, py)
			}
		}
		ax, ay := s.cellAt(s.apple)
		switch at(sc, ax, ay) {
		case uint8(1 + s.next):
			lit[true] = true
		case 0:
			lit[false] = true
		default:
			t.Fatalf("frame %d: the apple is ink %d, the next colour %d", f, at(sc, ax, ay), 1+s.next)
		}
		s.t++
	}
	if !lit[true] || !lit[false] {
		t.Errorf("the apple is lit %v", lit)
	}
	if inks := s.Inks(); len(inks) != 1+len(ownColours) || inks[0] != ownGround {
		t.Errorf("inks %v", inks)
	}
}

// The same seed is the same game; a new size is a new one, and a scene
// too small has none and draws nothing.
func TestSnakeIsTheSeedsAndStartsOverOnAResize(t *testing.T) {
	a, b := NewSnake(9, SnakeSpeedDefault), NewSnake(9, SnakeSpeedDefault)
	for i := 0; i < 300; i++ {
		if !reflect.DeepEqual(a.Draw(40, 23), b.Draw(40, 23)) {
			t.Fatalf("frame %d differs", i)
		}
		a.Step()
		b.Step()
	}
	a.Draw(50, 29)
	if a.cw != 16 || a.ch != 9 || len(a.body) != snakeStart {
		t.Errorf("after a resize: %dx%d, %d long", a.cw, a.ch, len(a.body))
	}
	tiny := NewSnake(1, SnakeSpeedDefault)
	sc := tiny.Draw(4, 7)
	tiny.Step()
	for _, k := range sc.Pix {
		if k != 0 {
			t.Fatal("a scene too small lit a pixel")
		}
	}
}

// An apple eaten is a lump that runs down the body to the tail (user,
// 2026-10-06): the body carries it a place a move, and it runs a place
// more, so it is two further from the head each move, and gone past the
// tail.
func TestSnakeLumpRunsToTheTail(t *testing.T) {
	s := NewSnake(3, SnakeSpeedDefault)
	s.Draw(76, 31)
	for i := 0; len(s.lumps) == 0; i++ {
		if i > 1000 {
			t.Fatal("no lump in a thousand moves")
		}
		s.Step()
	}
	if !slices.Equal(s.lumps, []int{0}) {
		t.Fatalf("just eaten: lumps %v", s.lumps)
	}
	at, moves := 0, 0
	for {
		s.Step()
		moves++
		if at+2 >= len(s.body) {
			if slices.Contains(s.lumps, at+2) || (len(s.lumps) > 0 && s.lumps[0] >= at) {
				t.Fatalf("past the tail, %d long: lumps %v", len(s.body), s.lumps)
			}
			break
		}
		if s.lumps[0] != at+2 {
			t.Fatalf("move %d: the lump at %d, then %v", moves, at, s.lumps)
		}
		at += 2
	}
	if moves < 2 {
		t.Errorf("gone in %d moves", moves)
	}
}

// The mouth opens to eat (user, 2026-10-06): open the move before the
// apple, swallowing the move it is eaten, shut the rest of the time —
// and drawn so: open, the node is dark and the lower lip lit; else the
// snout is out.
func TestSnakeOpensItsMouthToEat(t *testing.T) {
	s := NewSnake(8, SnakeSpeedDefault)
	s.Draw(76, 31)
	px := func(sc Scene, p [2]int) bool {
		x, y := s.cellAt(s.body[0])
		dx, dy := s.way(s.body[1], s.body[0])
		px, py := turn(p, dx, dy)
		x, y = x+px, y+py
		return x >= 0 && x < sc.W && y >= 0 && y < sc.H && sc.Pix[y*sc.W+x] != 0
	}
	opened, swallowed, eats := 0, 0, 0
	for i := 0; i < 2000; i++ {
		next := s.apple >= 0 && s.move() == s.apple
		just := slices.Contains(s.lumps, 0)
		want := snakeHead
		switch {
		case next:
			want, opened = snakeOpen, opened+1
		case just:
			want, swallowed = snakeSwallow, swallowed+1
		}
		if !reflect.DeepEqual(s.head(), want) {
			t.Fatalf("move %d: the apple next %v, just eaten %v: the head %v", i, next, just, s.head())
		}
		sc := s.Draw(76, 31)
		if open := !px(sc, [2]int{-2, 0}) && px(sc, [2]int{-2, 1}) && px(sc, [2]int{-2, -1}); open != next {
			t.Fatalf("move %d: drawn open %v, the apple next %v", i, open, next)
		}
		long := len(s.body)
		s.Step()
		if len(s.body) > long {
			eats++
		}
	}
	if eats < 10 || opened != eats || swallowed != eats {
		t.Errorf("%d apples: opened %d times, swallowed %d", eats, opened, swallowed)
	}
}

// The speed is cells a second (user, 2026-10-06): a move every second
// over it, from one to thirty, twelve for any other; a full board stays
// three seconds, and the apple blinks a third of a second, at any speed.
func TestSnakeSpeed(t *testing.T) {
	now := time.Date(2026, time.October, 6, 21, 5, 0, 0, time.UTC)
	for _, c := range []struct {
		speed, moves, hold, blink int
	}{
		{1, 1, 3, 1},
		{12, 12, 36, 3},
		{30, 30, 90, 9},
		{0, 12, 36, 3},
		{31, 12, 36, 3},
		{-5, 12, 36, 3},
	} {
		s := NewSnake(1, c.speed)
		if got := s.Next(now).Sub(now); got != time.Second/time.Duration(c.moves) {
			t.Errorf("speed %d: a move every %v", c.speed, got)
		}
		if h, b := s.frames(snakeHold), s.frames(snakeBlink); h != c.hold || b != c.blink {
			t.Errorf("speed %d: holds %d moves, blinks every %d", c.speed, h, b)
		}
	}
	// At thirty a second the apple is lit nine moves, then dark nine.
	s := NewSnake(1, 30)
	s.Draw(76, 31)
	x, y := s.cellAt(s.apple)
	for f := 0; f < 36; f++ {
		s.t = f
		sc := s.Draw(76, 31)
		if lit := sc.Pix[y*sc.W+x] != 0; lit != ((f/9)%2 == 0) {
			t.Fatalf("move %d: the apple lit %v", f, lit)
		}
	}
}
