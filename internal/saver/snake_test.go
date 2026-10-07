package saver

import (
	"reflect"
	"slices"
	"strings"
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
	for i, l := range s.lumps {
		if (l.at < 0 && l.at != snakeBitten) || l.at >= len(s.body) || l.colour >= len(ownColours) {
			t.Fatalf("step %d: a lump at %d, off a body %d long", step, l.at, len(s.body))
		}
		// One apple a segment, the oldest the furthest down; the one in
		// the jaws is at the head.
		if i > 0 && s.lumps[i-1].at <= max(l.at, 0) {
			t.Fatalf("step %d: two apples in a segment: %v", step, s.lumps)
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
	if s.apple >= 0 && (s.next >= len(ownColours) || (enough && s.next == s.colour)) {
		t.Fatalf("step %d: the apple is colour %d, the snake %d", step, s.next, s.colour)
	}
}

// It plays the board full without ever running into itself, holds the
// full board, and starts again. The snake starts white; an apple eaten
// is a lump in its colour, in the jaws first, and the snake takes the
// colour when the lump is at the tail and goes, and only then.
func TestSnakeFillsTheBoardAndStartsAgain(t *testing.T) {
	for i, sz := range [][2]int{{15, 11}, {21, 13}, {40, 23}, {31, 19}, {16, 9}, {76, 31}, {50, 29}, {49, 31}, {11, 7}, {100, 59}} {
		s := NewSnake(uint64(4+i), SpeedNormal)
		s.Draw(sz[0], sz[1])
		n := len(s.tour)
		if len(s.body) != snakeStart || n == 0 || s.colour != len(ownColours) {
			t.Fatalf("%v: start %d long on %d cells, colour %d", sz, len(s.body), n, s.colour)
		}
		steps, eats, enough := 0, 0, true
		for s.apple >= 0 {
			long, colour, next, apple, before := len(s.body), s.colour, s.next, s.apple, slices.Clone(s.lumps)
			tail := s.body[long-1]
			s.Step()
			steps++
			if len(s.lumps) >= len(ownColours)-1 {
				enough = false
			}
			check(t, s, steps, enough)
			ate := s.body[0] == apple
			if ate {
				eats++
				if l, ok := s.lumpAt(snakeBitten); !ok || l.colour != next {
					t.Fatalf("%v step %d: ate, and in the jaws %v; the apple was %d", sz, steps, s.lumps, next)
				}
			} else if s.next != next {
				t.Fatalf("%v step %d: the apple changed colour with nothing eaten", sz, steps)
			}
			// Every apple eaten is on its way down, or a segment the tail
			// has to grow, or has grown one; a segment a move at most.
			if len(s.body)+s.grow+len(s.lumps) != snakeStart+eats || s.grow < 0 || len(s.body)-long > 1 || len(s.body) < long {
				t.Fatalf("%v step %d: %d long (was %d), %d to grow, %d on the way down, %d eaten", sz, steps, len(s.body), long, s.grow, len(s.lumps), eats)
			}
			if want := turned(before, s.lumps, colour); s.colour != want {
				t.Fatalf("%v step %d: the snake is %d, the lumps were %v and are %v", sz, steps, s.colour, before, s.lumps)
			}
			// The move a lump is the body, the tail grows with it — but
			// where the head goes into the tail.
			added := 0
			if ate {
				added = 1
			}
			if len(before)+added > len(s.lumps) && s.body[0] != tail && len(s.body) != long+1 {
				t.Fatalf("%v step %d: an apple is the body, and the snake is %d long, was %d", sz, steps, len(s.body), long)
			}
			if steps > 4*n*n {
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
			// The body is still; the apples go on down at their pace.
			if len(s.lumps) > len(before) {
				t.Fatalf("%v: on a full board, the lumps %v then %v", sz, before, s.lumps)
			}
		}
		s.Step()
		if len(s.body) != snakeStart || s.apple < 0 || s.colour != len(ownColours) {
			t.Fatalf("%v: no new game: %d long, colour %d", sz, len(s.body), s.colour)
		}
		t.Logf("%v: %d cells in %d steps", sz, n, steps)
	}
}

// turned is the snake's colour after a move: that of the last lump to be
// the body now — or the colour it was. One apple a segment, the oldest
// the furthest down: the move drops from the front those that are the
// body, and an apple eaten is a new one at the back.
func turned(before, after []lump, colour int) int {
	added := 0
	if len(after) > 0 && after[len(after)-1].at == snakeBitten {
		added = 1
	}
	if dropped := len(before) + added - len(after); dropped > 0 {
		colour = before[dropped-1].colour
	}
	return colour
}

// Short cuts while the board is mostly empty: a game takes fewer steps
// than following the cycle for every apple would.
func TestSnakeCutsWhileThereIsRoom(t *testing.T) {
	s := NewSnake(11, SpeedNormal)
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
// one on, the link behind — the node left to the apple.
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
	}
}

// A frame is the snake in its colour — every node, the two pixels of
// every link, its head — and in their own an apple on its way down at
// its node, and the apple every other few frames.
func TestSnakeDrawsItsBodyHeadLumpsAndApple(t *testing.T) {
	s := NewSnake(6, SpeedNormal)
	s.Draw(76, 31)
	for i := 0; i < 200; i++ {
		s.Step()
	}
	// A lump a few segments down, wherever the last apple is.
	bean := otherColour(s.rng, s.colour)
	s.lumps = []lump{{at: 2, colour: bean}}
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
	if inks := s.Inks(); len(inks) != 2+len(ownColours) || inks[0] != ownGround || inks[len(inks)-1] != "#ffffff" {
		t.Errorf("inks %v", inks)
	}
}

// The same seed is the same game; a new size is a new one, and a scene
// too small has none and draws nothing.
func TestSnakeIsTheSeedsAndStartsOverOnAResize(t *testing.T) {
	a, b := NewSnake(9, SpeedNormal), NewSnake(9, SpeedNormal)
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
	tiny := NewSnake(1, SpeedNormal)
	sc := tiny.Draw(4, 7)
	tiny.Step()
	for _, k := range sc.Pix {
		if k != 0 {
			t.Fatal("a scene too small lit a pixel")
		}
	}
}

// An apple eaten goes down the body to the tail at its own pace (user,
// 2026-10-06), whatever the snake does: in the throat the move after,
// then a segment a second — seven moves at normal speed; a second at
// the tail and it is the body: the snake its colour and the tail a
// segment to grow, both at once (user, 2026-10-07).
func TestSnakeLumpGoesDownASegmentASecond(t *testing.T) {
	s := NewSnake(3, SpeedNormal)
	s.Draw(76, 31)
	for i := 0; i < 40; i++ {
		s.Step()
	}
	s.lumps, s.grow = []lump{{at: snakeBitten, colour: s.next}}, 0
	colour, bean, long := s.colour, s.next, len(s.body)
	second := s.frames(snakeDigest)
	if second != 7 || long < 3 {
		t.Fatalf("a second is %d moves, the snake %d long", second, long)
	}
	s.digest()
	for at := 0; at < long; at++ {
		for k := 0; k < second; k++ {
			if len(s.lumps) != 1 || s.lumps[0].at != at || s.colour != colour || s.grow != 0 {
				t.Fatalf("segment %d, move %d: lumps %v, the snake %d, %d to grow", at, k, s.lumps, s.colour, s.grow)
			}
			s.digest()
		}
	}
	if len(s.lumps) != 0 || s.colour != bean || s.grow != 1 {
		t.Errorf("the body: lumps %v, the snake %d (the apple %d), %d to grow", s.lumps, s.colour, bean, s.grow)
	}
	// And the tail grows it, the next move the head is not going where
	// the tail is.
	s.Step()
	if len(s.body) != long+1 && !(len(s.body) == long && s.grow == 1) {
		t.Errorf("%d long (was %d), %d to grow", len(s.body), long, s.grow)
	}
}

// One apple a segment (user, 2026-10-06; at the tail too, 2026-10-07):
// the move an apple is in the jaws, the one still in the throat goes on
// a segment, and the next, down the line; with an apple in every
// segment, the one at the tail is pushed into the body — the snake its
// colour and the tail a segment longer, at once. One whose second is up
// while the apple ahead still holds the next segment waits for it.
func TestSnakeApplesQueue(t *testing.T) {
	s := NewSnake(3, SpeedNormal)
	s.Draw(76, 31)
	for i := 0; i < 40; i++ {
		s.Step()
	}
	bite := func(lumps []lump) {
		s.lumps, s.grow = lumps, 0
		s.apple = s.tour[(s.pos[s.body[0]]+1)%len(s.tour)]
		s.Step()
		if !s.bitten() {
			t.Fatalf("no apple in the jaws: %v", s.lumps)
		}
	}
	bite([]lump{{at: 1, colour: 1, wait: 5}, {at: 0, colour: 2, wait: 9}})
	if got := []int{s.lumps[0].at, s.lumps[1].at, s.lumps[2].at}; !slices.Equal(got, []int{2, 1, snakeBitten}) {
		t.Errorf("after a bite, the apples at %v", got)
	}
	long := len(s.body)
	if long < 3 || long >= len(ownColours) {
		t.Fatalf("the snake %d long", long)
	}
	var full []lump
	for at := long - 1; at >= 0; at-- {
		full = append(full, lump{at: at, colour: (s.colour + 1 + at) % len(ownColours), wait: 5})
	}
	oldest := full[0].colour
	bite(full)
	var at []int
	for _, l := range s.lumps {
		at = append(at, l.at)
	}
	want := []int{snakeBitten}
	for a := 1; a < long; a++ {
		want = append([]int{a}, want...)
	}
	if !slices.Equal(at, want) || s.colour != oldest || len(s.body) != long+1 || s.grow != 0 {
		t.Errorf("a bite with an apple in every segment: the apples at %v, the snake %d (the oldest %d), %d long (was %d), %d to grow",
			at, s.colour, oldest, len(s.body), long, s.grow)
	}
	last := len(s.body) - 1
	s.lumps = []lump{{at: last, colour: 1, wait: 3}, {at: last - 1, colour: 2, wait: 1}}
	for k := 0; k < 2; k++ {
		s.digest()
		if s.lumps[1].at != last-1 {
			t.Fatalf("move %d: the apple behind went on to %d, the one ahead at %d", k, s.lumps[1].at, s.lumps[0].at)
		}
	}
	s.digest()
	if len(s.lumps) != 1 || s.lumps[0].at != last {
		t.Errorf("the one ahead is the body, and the one behind at %v", s.lumps)
	}
}

// The snake eats in three moves (user, 2026-10-06): the move before, the
// head touches the apple; the move it eats, the jaws are round it, the
// apple in its colour at the node; the move after, it swallows, the
// apple a pixel behind the shut head, still its colour, and the snake
// still its own — and the body behind the crown is dark, open or shut.
// Shut the rest of the time; and drawn so.
func TestSnakeEatsInThreeMoves(t *testing.T) {
	s := NewSnake(8, SpeedNormal)
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
		_, behind := s.lumpAt(1) // an apple on its way down, just behind the head
		if bitten && (px(sc, [2]int{0, 0}) != uint8(1+jaws.colour) || px(sc, [2]int{0, 1}) != body || px(sc, [2]int{0, -1}) != body ||
			(!behind && px(sc, [2]int{3, 0}) != body) || px(sc, [2]int{4, -1}) != 0) {
			t.Fatalf("move %d: the apple %d in jaws %d, %d, the body %d, behind the jaw %d; the apple is %d, the snake %d",
				i, px(sc, [2]int{0, 0}), px(sc, [2]int{0, -1}), px(sc, [2]int{0, 1}), px(sc, [2]int{3, 0}), px(sc, [2]int{4, -1}), jaws.colour, s.colour)
		}
		if swallowing && !bitten && !next {
			swallowed++
			if px(sc, [2]int{0, 0}) != uint8(1+gulp.colour) || px(sc, [2]int{-1, 0}) != body || px(sc, [2]int{4, -1}) != 0 {
				t.Fatalf("move %d: just swallowed, the apple is ink %d, the snout %d, behind the crown %d", i, px(sc, [2]int{0, 0}), px(sc, [2]int{-1, 0}), px(sc, [2]int{4, -1}))
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
		s.Step()
		if s.bitten() {
			eats++
		}
	}
	if eats < 10 || opened != eats || touched < eats/2 || swallowed < eats/3 {
		t.Errorf("%d apples: touched %d, opened %d, swallowed %d", eats, touched, opened, swallowed)
	}
}

// The head about the apple, going left, as the user drew it (2026-10-07):
// shut; open round the apple at the node, the snout's pixel dark; and
// the move after, shut, the apple in the throat. Nothing else about the
// head — a pixel of the body swelled behind it, open and shut, and meant
// nothing. 'x' the snake, 'o' the apple.
func TestSnakeHeadRoundTheAppleIsAsDrawn(t *testing.T) {
	s := NewSnake(1, SpeedNormal)
	s.Draw(snakeRoom.W, snakeRoom.H)
	s.body, s.apple = nil, -1
	for i := 0; i < 5; i++ {
		s.body = append(s.body, 4*s.cw+2+i)
	}
	bean := otherColour(s.rng, s.colour)
	for _, c := range []struct {
		what  string
		lumps []lump
		want  string
	}{
		{"shut", nil, `
 xx
xxxxxxxxxxxxxx`},
		{"open", []lump{{at: snakeBitten, colour: bean}}, `
 xxx
 oxxxxxxxxxxxx
 xx`},
		{"swallowed", []lump{{at: 0, colour: bean}}, `
 xx
xoxxxxxxxxxxxx`},
	} {
		s.lumps = c.lumps
		sc := s.Draw(snakeRoom.W, snakeRoom.H)
		hx, hy := s.cellAt(s.body[0])
		var rows []string
		for y := hy - 2; y <= hy+2; y++ {
			var row strings.Builder
			for x := hx - 2; x <= hx+snakePitch*len(s.body); x++ {
				switch sc.Pix[y*sc.W+x] {
				case 0:
					row.WriteString(" ")
				case uint8(1 + s.colour):
					row.WriteString("x")
				case uint8(1 + bean):
					row.WriteString("o")
				default:
					row.WriteString("?")
				}
			}
			rows = append(rows, strings.TrimRight(row.String(), " "))
		}
		// The drawing, and a dark pixel more round it.
		want := "\n " + strings.ReplaceAll(strings.TrimPrefix(c.want, "\n"), "\n", "\n ")
		if got := strings.TrimRight(strings.Join(rows, "\n"), "\n"); got != want {
			t.Errorf("%s:%s\nwant%s", c.what, got, want)
		}
	}
}

// The speed is picked by name (user, 2026-10-07): slow, normal, fast,
// very fast and super fast — five, seven, ten, fourteen and twenty cells
// a second, each about 1.4 times the one before — and normal for any
// other, a number among them; a full board stays three seconds, and the
// apple blinks a third of a second, at any speed.
func TestSnakeSpeed(t *testing.T) {
	if !slices.Equal(Speeds, []string{"slow", "normal", "fast", "very-fast", "super-fast"}) {
		t.Errorf("speeds %v", Speeds)
	}
	now := time.Date(2026, time.October, 7, 11, 5, 0, 0, time.UTC)
	for _, c := range []struct {
		speed              string
		moves, hold, blink int
	}{
		{SpeedSlow, 5, 15, 1},
		{SpeedNormal, 7, 21, 2},
		{SpeedFast, 10, 30, 3},
		{SpeedVeryFast, 14, 42, 4},
		{SpeedSuperFast, 20, 60, 6},
		{"", 7, 21, 2},
		{"12", 7, 21, 2},
		{"Fast", 7, 21, 2},
	} {
		s := NewSnake(1, c.speed)
		if got := s.Next(now).Sub(now); got != time.Second/time.Duration(c.moves) {
			t.Errorf("speed %q: a move every %v", c.speed, got)
		}
		if h, b := s.frames(snakeHold), s.frames(snakeBlink); h != c.hold || b != c.blink {
			t.Errorf("speed %q: holds %d moves, blinks every %d", c.speed, h, b)
		}
	}
	// Super fast, the apple is lit six moves, then dark six.
	s := NewSnake(1, SpeedSuperFast)
	s.Draw(76, 31)
	x, y := s.cellAt(s.apple)
	for f := 0; f < 24; f++ {
		s.t = f
		sc := s.Draw(76, 31)
		if lit := sc.Pix[y*sc.W+x] != 0; lit != ((f/6)%2 == 0) {
			t.Fatalf("move %d: the apple lit %v", f, lit)
		}
	}
}

// An apple on its way down is its segment's node in its colour and
// nothing more (user, 2026-10-07: the body swelled round it, and the user
// took that away): at any segment — along a straight, at a corner, at the
// tail — there is nothing about it but the links to the segments either
// side.
func TestSnakeAppleInTheBodyIsANode(t *testing.T) {
	s := NewSnake(6, SpeedNormal)
	s.Draw(76, 31)
	corners, straights := 0, 0
	for moment := 0; moment < 20; moment++ {
		for i := 0; i < 137; i++ {
			s.Step()
		}
		corners, straights = appleInEverySegment(t, s, corners, straights)
	}
	if corners == 0 || straights == 0 {
		t.Errorf("%d corners, %d straights", corners, straights)
	}
}

// appleInEverySegment puts an apple in each segment of s but the head in
// turn, and looks about it.
func appleInEverySegment(t *testing.T, s *Snake, corners, straights int) (int, int) {
	t.Helper()
	bean := otherColour(s.rng, s.colour)
	ink := uint8(1 + s.colour)
	for i := 1; i < len(s.body); i++ {
		s.lumps = []lump{{at: i, colour: bean}}
		sc := s.Draw(76, 31)
		x, y := s.cellAt(s.body[i])
		var links [][2]int
		for _, j := range []int{i - 1, i + 1} {
			if j < len(s.body) {
				dx, dy := s.way(s.body[i], s.body[j])
				links = append(links, [2]int{dx, dy})
			}
		}
		for py := -1; py <= 1; py++ {
			for px := -1; px <= 1; px++ {
				want := uint8(0)
				switch p := [2]int{px, py}; {
				case p == [2]int{0, 0}:
					want = uint8(1 + bean)
				case slices.Contains(links, p):
					want = ink
				}
				if got := sc.Pix[(y+py)*sc.W+x+px]; got != want {
					t.Fatalf("segment %d of %d: the pixel %d,%d is ink %d, want %d", i, len(s.body), px, py, got, want)
				}
			}
		}
		switch {
		case len(links) < 2:
		case links[0] == [2]int{-links[1][0], -links[1][1]}:
			straights++
		default:
			corners++
		}
	}
	return corners, straights
}

// The end of an apple (user, 2026-10-06; nothing about it, 2026-10-07):
// at the tail, the apple at the node and the one link, and nothing else.
func TestSnakeAppleAtTheTail(t *testing.T) {
	s := NewSnake(3, SpeedNormal)
	s.Draw(76, 31)
	for i := 0; i < 40; i++ {
		s.Step()
	}
	last := len(s.body) - 1
	bean := otherColour(s.rng, s.colour)
	ink := uint8(1 + s.colour)
	x, y := s.cellAt(s.body[last])
	dx, dy := s.way(s.body[last], s.body[last-1])
	s.lumps = []lump{{at: last, colour: bean, wait: 5}}
	sc := s.Draw(76, 31)
	for py := -1; py <= 1; py++ {
		for px := -1; px <= 1; px++ {
			want := uint8(0)
			switch {
			case px == 0 && py == 0:
				want = uint8(1 + bean)
			case px == dx && py == dy:
				want = ink
			}
			if got := sc.Pix[(y+py)*sc.W+x+px]; got != want {
				t.Errorf("the pixel %d,%d is ink %d, want %d", px, py, got, want)
			}
		}
	}
}

// A segment to grow where the head is going into the tail: the tail
// moves on, and grows once the head goes elsewhere, or the head would
// run into it. On a
// board of four cells by two, the cycle along the top from the second
// column, back along the bottom and up the first: a snake from (1,0)
// round the right to (2,0), with (0,0) and (0,1) behind its head, has
// its tail next.
func TestSnakeGrowsAMoveLaterWhereTheHeadGoesIntoItsTail(t *testing.T) {
	s := NewSnake(1, SpeedNormal)
	s.Draw(12, 6)
	if s.cw != 4 || s.ch != 2 {
		t.Fatalf("a board %dx%d", s.cw, s.ch)
	}
	cell := func(x, y int) int { return y*s.cw + x }
	s.body = []int{cell(1, 0), cell(1, 1), cell(2, 1), cell(3, 1), cell(3, 0), cell(2, 0)}
	s.taken = make([]bool, s.cw*s.ch)
	for _, c := range s.body {
		s.taken[c] = true
	}
	s.apple, s.lumps, s.grow = cell(0, 1), nil, 1
	if s.move() != cell(2, 0) {
		t.Fatalf("the head goes to %d, not its tail", s.move())
	}
	s.Step()
	check(t, s, 1, true)
	if len(s.body) != 6 || s.grow != 1 || s.body[0] != cell(2, 0) {
		t.Fatalf("into the tail: %v, %d to grow", s.body, s.grow)
	}
	for step := 2; s.grow > 0; step++ {
		if step > 20 {
			t.Fatalf("never grew: %v", s.body)
		}
		s.Step()
		check(t, s, step, true)
	}
	if len(s.body) != 7 {
		t.Errorf("grown: %v", s.body)
	}
}
