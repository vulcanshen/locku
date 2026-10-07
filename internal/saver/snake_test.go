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
// not the snake's. While there have been colours enough — fewer lumps on
// their way down than there are colours but the snake's — no lump is the
// snake's colour or the apple's either.
func check(t *testing.T, s *Snake, step int, enough bool) {
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
	for _, l := range s.lumps {
		if (l.at < 0 && l.at != snakeBitten) || l.at >= len(s.body) {
			t.Fatalf("step %d: a lump at %d, off a body %d long", step, l.at, len(s.body))
		}
		if enough && (l.colour == s.colour || (s.apple >= 0 && l.colour == s.next)) {
			t.Fatalf("step %d: a lump in the colour of the snake (%d) or the apple (%d): %v", step, s.colour, s.next, s.lumps)
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
	if enough && s.apple >= 0 && s.next == s.colour {
		t.Fatalf("step %d: the apple is the snake's colour", step)
	}
}

// It plays the board full without ever running into itself, holds the
// full board, and starts again. An apple eaten is a lump in its colour,
// in the jaws first; the snake takes the colour when the lump goes past
// the tail, and only then.
func TestSnakeFillsTheBoardAndStartsAgain(t *testing.T) {
	for i, sz := range [][2]int{{15, 11}, {21, 13}, {40, 23}, {31, 19}, {16, 9}, {76, 31}, {50, 29}, {49, 31}, {11, 7}, {100, 59}} {
		s := NewSnake(uint64(4+i), SnakeSpeedDefault)
		s.Draw(sz[0], sz[1])
		n := len(s.tour)
		if len(s.body) != snakeStart || n == 0 {
			t.Fatalf("%v: start %d long on %d cells", sz, len(s.body), n)
		}
		steps, enough := 0, true
		for s.apple >= 0 {
			long, colour, next, before := len(s.body), s.colour, s.next, slices.Clone(s.lumps)
			s.Step()
			steps++
			if len(s.lumps) >= len(ownColours)-1 {
				enough = false
			}
			check(t, s, steps, enough)
			if l, ok := s.lumpAt(snakeBitten); (len(s.body) > long) != (ok && l.colour == next) {
				t.Fatalf("%v step %d: %d long, then %d; in the jaws %v, the apple was %d", sz, steps, long, len(s.body), s.lumps, next)
			}
			if len(s.body) == long && s.next != next {
				t.Fatalf("%v step %d: the apple changed colour with nothing eaten", sz, steps)
			}
			if want := gone(before, len(s.body), colour); s.colour != want {
				t.Fatalf("%v step %d: the snake is %d, the lumps were %v and are %v", sz, steps, s.colour, before, s.lumps)
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
			// The body is still; the lumps run on, a place a frame, and the
			// apple in the jaws is swallowed.
			var want []lump
			for _, l := range before {
				if l.at < 0 {
					l.at = 0
				} else {
					l.at++
				}
				if l.at < n {
					want = append(want, l)
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

// gone is the snake's colour after a move that left it long: the last
// lump of before two places on — an apple in the jaws at the head — and
// so past the tail, or the colour it was.
func gone(before []lump, long, colour int) int {
	for _, l := range before {
		if at := max(l.at+2, 0); l.at >= 0 && at >= long {
			colour = l.colour
		}
	}
	return colour
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
// the right up or down a column — nothing over the snout. Touching: the
// same a pixel on, the link behind left to the body. Open, round the apple at
// the node: the upper jaw over it and two on, the lower under it and
// one on, the link behind — the node left to the apple. Swallowing, the
// head is shut and the body swells a pixel behind the crown. The body
// round an apple on its way down is three pixels over its node, the
// node left to the apple.
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
		touch, want := set(snakeTouch, dx, dy), map[[2]int]bool{node: true}
		for q := range shut {
			if q != link1 && q != link2 {
				want[back(q, -1)] = true
			}
		}
		if !reflect.DeepEqual(touch, want) {
			t.Errorf("going %d,%d: touching, the head is %v, not the shut one a pixel on", dx, dy, touch)
		}
		open := set(snakeOpen, dx, dy)
		if len(open) != 7 || open[node] || open[snout] || !open[link1] || !open[link2] ||
			!open[c.crown1] || !open[c.crown2] || !open[back(c.crown2, 1)] || !open[across(c.crown1)] || !open[across(c.crown2)] {
			t.Errorf("going %d,%d: open, the head is %v", dx, dy, open)
		}
		if gulp := set(snakeGulp, dx, dy); len(gulp) != 1 || !gulp[back(c.crown2, 3)] {
			t.Errorf("going %d,%d: just swallowed, the swelling is at %v", dx, dy, gulp)
		}
		if lump := set(snakeLump, dx, dy); len(lump) != 3 || lump[node] || !lump[back(c.crown1, -1)] || !lump[c.crown1] || !lump[c.crown2] {
			t.Errorf("going %d,%d: the lump is %v", dx, dy, lump)
		}
	}
}

// A frame is the snake in its colour — every node, the two pixels of
// every link, its head — and in their own an apple on its way down, its
// node and its bulge, and the apple every other few frames.
func TestSnakeDrawsItsBodyHeadLumpsAndApple(t *testing.T) {
	s := NewSnake(6, SnakeSpeedDefault)
	s.Draw(76, 31)
	for i := 0; i < 200; i++ {
		s.Step()
	}
	// A lump a few segments down, wherever the last apple is.
	bean := otherColour(s.rng, s.colour)
	s.lumps = []lump{{2, bean}}
	ink := uint8(1 + s.colour)
	at := func(sc Scene, x, y int) uint8 { return sc.Pix[y*sc.W+x] }
	lit := map[bool]bool{}
	for f := 0; f < 2*s.frames(snakeBlink); f++ {
		sc := s.Draw(76, 31)
		for i, c := range s.body {
			x, y := s.cellAt(c)
			want := ink
			if i == 2 {
				want = uint8(1 + bean)
			}
			if i > 0 && at(sc, x, y) != want {
				t.Fatalf("frame %d: node %d is ink %d, want %d", f, i, at(sc, x, y), want)
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
		head, dx, dy := s.head()
		for _, p := range head {
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
				t.Fatalf("frame %d: the body round the apple at %d,%d is ink %d", f, px, py, at(sc, lx+px, ly+py))
			}
		}
		ax, ay := s.cellAt(s.apple)
		switch at(sc, ax, ay) {
		case uint8(1 + s.next):
			lit[true] = true
		case 0:
			lit[false] = true
		default:
			t.Fatalf("frame %d: the apple is ink %d, its colour %d", f, at(sc, ax, ay), 1+s.next)
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
	if len(s.lumps) != 1 || s.lumps[0].at != snakeBitten {
		t.Fatalf("just eaten: lumps %v", s.lumps)
	}
	colour, bean := s.colour, s.lumps[0].colour
	at, moves := snakeBitten, 0
	for {
		s.Step()
		moves++
		if at+2 >= len(s.body) {
			if len(s.lumps) > 0 && s.lumps[0].at >= at {
				t.Fatalf("past the tail, %d long: lumps %v", len(s.body), s.lumps)
			}
			if s.colour != bean {
				t.Fatalf("past the tail, the snake is %d, the apple was %d", s.colour, bean)
			}
			break
		}
		if s.lumps[0].at != at+2 || s.lumps[0].colour != bean {
			t.Fatalf("move %d: the lump at %d, then %v", moves, at, s.lumps)
		}
		// Others ahead of it may go past the tail, and turn the snake;
		// never its own colour while it is on its way down.
		if s.colour == bean {
			t.Fatalf("move %d: the snake is the apple's colour, the lump not yet at the tail", moves)
		}
		if s.colour != colour {
			t.Fatalf("move %d: the snake turned %d with no lump past the tail", moves, s.colour)
		}
		at += 2
	}
	if moves < 2 {
		t.Errorf("gone in %d moves", moves)
	}
}

// The snake eats in three moves (user, 2026-10-06): the move before, the
// head touches the apple; the move it eats, the jaws are round it, the
// apple in its colour at the node; the move after, it swallows, the
// apple a pixel behind the shut head, still its colour, and the snake
// still its own. Shut the rest of the time; and drawn so.
func TestSnakeEatsInThreeMoves(t *testing.T) {
	s := NewSnake(8, SnakeSpeedDefault)
	s.Draw(76, 31)
	px := func(sc Scene, p [2]int) uint8 {
		x, y := s.cellAt(s.body[0])
		_, dx, dy := s.head()
		px, py := turn(p, dx, dy)
		x, y = x+px, y+py
		if x < 0 || x >= sc.W || y < 0 || y >= sc.H {
			return 0
		}
		return sc.Pix[y*sc.W+x]
	}
	touched, opened, swallowed, eats := 0, 0, 0, 0
	for i := 0; i < 2000; i++ {
		jaws, bitten := s.lumpAt(snakeBitten)
		gulp, swallowing := s.lumpAt(0)
		next := s.apple >= 0 && s.move() == s.apple
		want := snakeHead
		switch {
		case bitten:
			want, opened = snakeOpen, opened+1
		case next:
			want, touched = snakeTouch, touched+1
		}
		head, hdx, hdy := s.head()
		if !reflect.DeepEqual(head, want) {
			t.Fatalf("move %d: bitten %v, the apple next %v: the head %v", i, bitten, next, head)
		}
		// Touching, it faces the apple; else the way it came.
		if wdx, wdy := s.way(s.body[1], s.body[0]); next && !bitten {
			if ax, ay := s.way(s.body[0], s.apple); hdx != ax || hdy != ay {
				t.Fatalf("move %d: touching, it faces %d,%d, the apple %d,%d", i, hdx, hdy, ax, ay)
			}
		} else if hdx != wdx || hdy != wdy {
			t.Fatalf("move %d: it faces %d,%d, it came %d,%d", i, hdx, hdy, wdx, wdy)
		}
		sc := s.Draw(76, 31)
		body := uint8(1 + s.colour)
		if bitten && (px(sc, [2]int{0, 0}) != uint8(1+jaws.colour) || px(sc, [2]int{0, 1}) != body || px(sc, [2]int{0, -1}) != body || px(sc, [2]int{3, 0}) != body) {
			t.Fatalf("move %d: the apple %d in jaws %d, %d, the body %d; the apple is %d, the snake %d",
				i, px(sc, [2]int{0, 0}), px(sc, [2]int{0, -1}), px(sc, [2]int{0, 1}), px(sc, [2]int{3, 0}), jaws.colour, s.colour)
		}
		if swallowing && !bitten && !next {
			swallowed++
			if px(sc, [2]int{0, 0}) != uint8(1+gulp.colour) || px(sc, [2]int{4, -1}) != body || px(sc, [2]int{-1, 0}) != body {
				t.Fatalf("move %d: just swallowed, the apple is ink %d, the swelling %d, the snout %d", i, px(sc, [2]int{0, 0}), px(sc, [2]int{4, -1}), px(sc, [2]int{-1, 0}))
			}
		}
		if next && !bitten {
			// The snout is the pixel before the apple's.
			hx, hy := s.cellAt(s.body[0])
			ax, ay := s.cellAt(s.apple)
			sx, sy := turn([2]int{-2, 0}, hdx, hdy)
			if px(sc, [2]int{-2, 0}) != body || hx+sx+hdx != ax || hy+sy+hdy != ay {
				t.Fatalf("move %d: the snout does not touch the apple", i)
			}
		}
		long := len(s.body)
		s.Step()
		if len(s.body) > long {
			eats++
		}
	}
	if eats < 10 || opened != eats || touched < eats/2 || swallowed < eats/3 {
		t.Errorf("%d apples: touched %d, opened %d, swallowed %d", eats, touched, opened, swallowed)
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

// The body goes round an apple on its way down (user, 2026-10-06): at
// any segment — along a straight, at a corner — the snake's colour runs
// on unbroken from the link on one side of the apple to the link on the
// other; at a corner round the outside, the pixels about the apple but
// the two links and the dark inner corner between them.
func TestSnakeBodyGoesRoundTheApple(t *testing.T) {
	s := NewSnake(6, SnakeSpeedDefault)
	s.Draw(76, 31)
	corners, straights := 0, 0
	for moment := 0; moment < 20; moment++ {
		for i := 0; i < 137; i++ {
			s.Step()
		}
		corners, straights = roundEverySegment(t, s, corners, straights)
	}
	if corners == 0 || straights == 0 {
		t.Errorf("%d corners, %d straights", corners, straights)
	}
}

// roundEverySegment puts an apple in each segment of s but the ends in
// turn, and looks at the body round it.
func roundEverySegment(t *testing.T, s *Snake, corners, straights int) (int, int) {
	t.Helper()
	bean := otherColour(s.rng, s.colour)
	ink := uint8(1 + s.colour)
	for i := 1; i < len(s.body)-1; i++ {
		s.lumps = []lump{{i, bean}}
		sc := s.Draw(76, 31)
		x, y := s.cellAt(s.body[i])
		at := func(p [2]int) uint8 { return sc.Pix[(y+p[1])*sc.W+x+p[0]] }
		if at([2]int{0, 0}) != uint8(1+bean) {
			t.Fatalf("segment %d: the apple is ink %d", i, at([2]int{0, 0}))
		}
		adx, ady := s.way(s.body[i], s.body[i-1])
		bdx, bdy := s.way(s.body[i], s.body[i+1])
		a, b := [2]int{adx, ady}, [2]int{bdx, bdy}
		// From one link to the other through the snake's colour about the
		// apple, a step at a time.
		seen, todo := map[[2]int]bool{a: true}, [][2]int{a}
		for len(todo) > 0 {
			p := todo[0]
			todo = todo[1:]
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				q := [2]int{p[0] + d[0], p[1] + d[1]}
				if q[0] < -1 || q[0] > 1 || q[1] < -1 || q[1] > 1 || seen[q] || at(q) != ink {
					continue
				}
				seen[q] = true
				todo = append(todo, q)
			}
		}
		if !seen[b] {
			t.Fatalf("segment %d: the body does not go round the apple from %v to %v", i, a, b)
		}
		if a == [2]int{-b[0], -b[1]} {
			straights++
			continue
		}
		corners++
		for py := -1; py <= 1; py++ {
			for px := -1; px <= 1; px++ {
				p := [2]int{px, py}
				if p == [2]int{0, 0} {
					continue
				}
				if inner := p == [2]int{a[0] + b[0], a[1] + b[1]}; (at(p) == ink) == inner {
					t.Fatalf("segment %d, a corner: the pixel %v is ink %d", i, p, at(p))
				}
			}
		}
	}
	return corners, straights
}
