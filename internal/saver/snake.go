package saver

import (
	"math/rand/v2"
	"time"
)

// The snake (user, 2026-09-27, made 2026-10-06): the old Nokia game,
// playing itself until the board is full, then again from the start.
// It cannot lose: it follows a cycle through every cell — a Hamiltonian
// cycle, back and forth across the board and up the first column — and
// takes a short cut towards the apple only while the cut leaves the
// cells ahead of it, up to its tail, empty and room to spare; once it
// covers half the board it takes none. Every cell is a node and the link
// to the next is lit, so where the body turns and where two parts of it
// merely lie side by side can be told apart, as on the Nokia's screen.
//
// The rest is the user's, the same day. It has a head: two pixels over
// the body's line, the snout a pixel ahead of it (it was three square
// with an eye, and too big). It eats in three moves, as the user drew
// them: the move before, the apple the next cell, its head reaches the
// apple, the snout touching it; the move it eats, the jaws are round the
// apple — the upper three pixels over it and on behind, the lower two
// under it; the move after, the mouth is shut, the apple in its throat
// and a pixel of the body swelling behind the head. (The jaws opened
// short of the apple at first, and the snake looked broken.) The apple
// blinks in a colour of its own, at random, and keeps it all the way
// through: an apple eaten runs down the body to the tail, a segment a
// move, in the line of the body in its colour, and the body goes round
// it — three pixels over it, in the snake's — so it is seen going
// through the gut (user, the same day); past the tail it is gone and
// the snake is its colour (it turned the moment it ate, at first; the
// lump stayed where it was swallowed before that, as on the Nokia). An
// apple is never the snake's colour nor any lump's on its way down, so
// it stands out the whole way. The snake is the longer for it at once,
// not when the lump reaches the tail: a tail held back then could be
// the cell the head was going into. The head and the lump stand out on
// one side of the body: above it along a row, to the right of it up or
// down a column.
//
// Its colours are its own (user, 2026-10-06: a saver of many colours
// brings them, and has no bg / fg). Its one setting is its speed, in
// cells a second (user, the same day).

const KindSnake = "snake"

// The snake's speed, cells a second (user, 2026-10-06): the user's
// number, from one to thirty — thirty frames a second is as many as the
// lock draws, or a key in the PIN box would wait on the output — and
// twelve when none is given, near the twelve and a half it ran at.
const (
	SnakeSpeedMin     = 1
	SnakeSpeedMax     = 30
	SnakeSpeedDefault = 12
)

const (
	snakeStart = 3                      // cells long at the start
	snakeHold  = 3 * time.Second        // a full board stays
	snakeBlink = 320 * time.Millisecond // the apple is lit, then dark, at any speed
	// snakePitch is a cell: its node and the two pixels of a link. The
	// two dark rows between two runs of the body side by side hold the
	// head or a lump, a pixel proud of the line, and keep a dark row
	// between it and the run beside.
	snakePitch = 3
)

// snakeRoom is a board of sixteen cells by ten, with a pixel round it
// for the jaw and the crown, drawn at most twice over.
var snakeRoom = Room{W: snakePitch * 16, H: snakePitch * 10, Most: 2}

// A shape is pixels about a cell's node, drawn for a snake going left;
// turned for the way it goes.
type shape [][2]int

var (
	// The head: the snout ahead, the node and the link behind, and the
	// crown over the node and the link's first pixel.
	snakeHead = shape{{-1, 0}, {0, 0}, {1, 0}, {2, 0}, {0, -1}, {1, -1}}
	// Touching the apple in the next cell: the head a pixel on towards
	// it, the snout against it. It faces the apple, which may be round a
	// corner: the link behind is the body's to draw, from wherever the
	// head came.
	snakeTouch = shape{{-2, 0}, {-1, 0}, {0, 0}, {-1, -1}, {0, -1}}
	// The jaws round the apple, which is at the node: the upper jaw over
	// it and two on, the lower under it and one on, the link behind. The
	// node is the apple's to draw.
	snakeOpen = shape{{1, 0}, {2, 0}, {0, -1}, {1, -1}, {2, -1}, {0, 1}, {1, 1}}
	// Swallowing, the head shut and the apple at its node: the body
	// swelling, a pixel behind the crown.
	snakeGulp = shape{{4, -1}}
	// A lump: the body round an apple on its way through, which is at the
	// node — three pixels over it (user, 2026-10-06: they were the
	// apple's, two, beside the node, before the body went round it).
	snakeLump = shape{{-1, -1}, {0, -1}, {1, -1}}
)

// turn is a pixel of a shape for a snake going dx, dy: going right the
// shape is mirrored and still stands up; going up or down it is turned
// a quarter and stands to the right.
func turn(p [2]int, dx, dy int) (int, int) {
	switch {
	case dx > 0:
		return -p[0], p[1]
	case dy < 0:
		return -p[1], p[0]
	case dy > 0:
		return -p[1], -p[0]
	}
	return p[0], p[1]
}

// lump is an apple on its way down: its place in the body, and its
// colour.
type lump struct{ at, colour int }

// Snake is one game in progress.
type Snake struct {
	rng    *rand.Rand
	w, h   int    // the scene as last drawn; nothing moves before the first draw
	cw, ch int    // the grid of cells
	ox, oy int    // where the grid sits in the scene
	tour   []int  // the cells in the cycle's order
	pos    []int  // a cell's place on the cycle; -1 off it
	body   []int  // head first
	taken  []bool // a cell the body is on
	lumps  []lump // the apples on their way down, the oldest first
	apple  int    // -1 when the board is full
	colour int    // the snake's, in ownColours
	next   int    // the apple's: the snake's when it has gone down
	t      int    // frames played
	hold   int    // frames a full board has still to stay
	frame  time.Duration
}

// NewSnake is a game from its first frame, speed cells a second — out
// of SnakeSpeedMin … SnakeSpeedMax, SnakeSpeedDefault. The same seed is
// the same game.
func NewSnake(seed uint64, speed int) *Snake {
	if speed < SnakeSpeedMin || speed > SnakeSpeedMax {
		speed = SnakeSpeedDefault
	}
	return &Snake{rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)), frame: time.Second / time.Duration(speed)}
}

// frames is how many moves d is at the snake's speed; one at least.
func (s *Snake) frames(d time.Duration) int { return max(1, int(d/s.frame)) }

// Room is the board the game needs.
func (s *Snake) Room() Room { return snakeRoom }

// Next is when the next move is due.
func (s *Snake) Next(now time.Time) time.Time { return now.Add(s.frame) }

// Inks are the ground and the colours.
func (s *Snake) Inks() []string { return ownInks() }

// hamilton is a cycle through a cw × ch grid's cells, as cell indices:
// along the first row from the second column, back along the next, and
// so on, then up the first column home — which closes when the rows
// are even in number; when they are not the grid is turned on its side,
// and when neither side is even the last column is left out. Too small
// a grid has none.
func hamilton(cw, ch int) []int {
	a, b, turned := cw, ch, false
	if b%2 != 0 {
		if a%2 != 0 {
			a--
		}
		a, b, turned = b, a, true
	}
	if a < 2 || b < 2 {
		return nil
	}
	var out []int
	add := func(i, j int) {
		if turned {
			i, j = j, i
		}
		out = append(out, j*cw+i)
	}
	add(0, 0)
	for j := 0; j < b; j++ {
		if j%2 == 0 {
			for i := 1; i < a; i++ {
				add(i, j)
			}
		} else {
			for i := a - 1; i >= 1; i-- {
				add(i, j)
			}
		}
	}
	for j := b - 1; j >= 1; j-- {
		add(0, j)
	}
	return out
}

// reset is a new game on a w × h scene: the grid, its cycle, a snake
// snakeStart long somewhere on it in a colour, and an apple in another.
func (s *Snake) reset(w, h int) {
	s.w, s.h = w, h
	s.cw, s.ch = w/snakePitch, h/snakePitch
	s.tour = hamilton(s.cw, s.ch)
	s.pos = make([]int, s.cw*s.ch)
	for i := range s.pos {
		s.pos[i] = -1
	}
	for i, c := range s.tour {
		s.pos[c] = i
	}
	s.body, s.taken, s.lumps, s.apple, s.hold = nil, make([]bool, s.cw*s.ch), nil, -1, 0
	n := len(s.tour)
	if n < snakeStart+1 {
		s.tour = nil
		return
	}
	// The grid in the middle of the scene, a pixel round it for the jaw
	// and the crown. An odd column left out is dark on either side, as
	// near as it halves.
	used := s.cw
	if s.ch%2 != 0 && s.cw%2 != 0 {
		used--
	}
	s.ox = 1 + (w-snakePitch*used)/2
	s.oy = 1 + (h-snakePitch*s.ch)/2
	p := s.rng.IntN(n)
	for i := 0; i < snakeStart; i++ {
		c := s.tour[(p-i+n)%n]
		s.body = append(s.body, c)
		s.taken[c] = true
	}
	s.colour = s.rng.IntN(len(ownColours))
	s.next = otherColour(s.rng, s.colour)
	s.place()
}

// place puts the apple on an empty cell, or none on a full board.
func (s *Snake) place() {
	free := len(s.tour) - len(s.body)
	if free == 0 {
		s.apple = -1
		return
	}
	k := s.rng.IntN(free)
	for _, c := range s.tour {
		if !s.taken[c] {
			if k == 0 {
				s.apple = c
				return
			}
			k--
		}
	}
}

// ahead is how far c is along the cycle past from.
func (s *Snake) ahead(from, c int) int {
	n := len(s.tour)
	return (s.pos[c] - s.pos[from] + n) % n
}

// neighbours are the cells on the cycle beside c.
func (s *Snake) neighbours(c int) []int {
	x, y := c%s.cw, c/s.cw
	var out []int
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, ny := x+d[0], y+d[1]
		if nx >= 0 && nx < s.cw && ny >= 0 && ny < s.ch && s.pos[ny*s.cw+nx] >= 0 {
			out = append(out, ny*s.cw+nx)
		}
	}
	return out
}

// move is where the head goes next. Along the cycle the body lies behind
// the head, in order, and every cell ahead of it up to the tail is empty;
// a cut to a cell further ahead, short of the tail, keeps that so — which
// is all it takes never to run into itself. The cut is as long as it can
// be without passing the apple, nor coming within the body's length and
// a margin of the tail: so none at all once the snake is half the board.
// With no cut, the next cell on the cycle: empty, or the tail, which
// moves away as the head comes.
func (s *Snake) move() int {
	n := len(s.tour)
	head, tail := s.body[0], s.body[len(s.body)-1]
	room := min(s.ahead(head, tail)-len(s.body)-3, s.ahead(head, s.apple))
	best, far := -1, 0
	for _, c := range s.neighbours(head) {
		if d := s.ahead(head, c); d <= room && d > far {
			best, far = c, d
		}
	}
	if best < 0 {
		best = s.tour[(s.pos[head]+1)%n]
	}
	return best
}

// Step is a move: the head on, the tail after it unless the apple is
// eaten, and every lump a segment further down — one past the tail
// turning the snake its colour; an apple eaten is a lump in the jaws,
// and a new apple is put in a fresh colour. A full board stays a while,
// its lumps still running, then a new game.
func (s *Snake) Step() {
	if s.w == 0 || s.tour == nil {
		return
	}
	s.t++
	if s.apple < 0 {
		s.digest(1)
		if s.hold--; s.hold <= 0 {
			s.reset(s.w, s.h)
		}
		return
	}
	next := s.move()
	ate := next == s.apple
	if !ate {
		tail := s.body[len(s.body)-1]
		s.body = s.body[:len(s.body)-1]
		s.taken[tail] = false
	}
	s.body = append([]int{next}, s.body...)
	s.taken[next] = true
	// The body has a new segment at the front: every lump is a place
	// further from the head for it, and one more for its run.
	s.digest(2)
	if ate {
		s.lumps = append(s.lumps, lump{snakeBitten, s.next})
		s.next = s.fresh()
		if s.place(); s.apple < 0 {
			s.hold = s.frames(snakeHold)
		}
	}
}

// snakeBitten is the place of an apple still in the jaws, the move it
// is eaten: the move after it is at the head, swallowed, and then it
// runs.
const snakeBitten = -2

// lumpAt is the lump at place i in the body, if there is one.
func (s *Snake) lumpAt(i int) (lump, bool) {
	for _, l := range s.lumps {
		if l.at == i {
			return l, true
		}
	}
	return lump{}, false
}

// bitten says whether the apple is in the jaws this move.
func (s *Snake) bitten() bool { _, ok := s.lumpAt(snakeBitten); return ok }

// fresh is a colour at random that is neither the snake's nor any apple's
// on its way down: whichever of them the snake has turned by the time it
// is eaten, and on the way down after, it stands out.
func (s *Snake) fresh() int {
	used := make([]bool, len(ownColours))
	used[s.colour] = true
	for _, l := range s.lumps {
		used[l.colour] = true
	}
	var free []int
	for c, u := range used {
		if !u {
			free = append(free, c)
		}
	}
	if len(free) == 0 {
		return otherColour(s.rng, s.colour)
	}
	return free[s.rng.IntN(len(free))]
}

// head is the head's shape this move and the way it faces: the jaws
// round the apple when it is eaten this move; touching it, facing it,
// when it is the next move's — the move is the same whenever it is
// worked out; shut else, swallowing too. But touching, it faces the way
// it came.
func (s *Snake) head() (shape, int, int) {
	dx, dy := s.way(s.body[1], s.body[0])
	switch {
	case s.bitten():
		return snakeOpen, dx, dy
	case s.apple >= 0 && s.move() == s.apple:
		dx, dy = s.way(s.body[0], s.apple)
		return snakeTouch, dx, dy
	}
	return snakeHead, dx, dy
}

// digest moves every lump by places towards the tail; one past it is
// gone, and the snake is its colour. An apple in the jaws is at the
// head the move after, whatever the body does.
func (s *Snake) digest(by int) {
	keep := s.lumps[:0]
	for _, l := range s.lumps {
		if l.at < 0 {
			l.at += -snakeBitten
		} else {
			l.at += by
		}
		if l.at < len(s.body) {
			keep = append(keep, l)
		} else {
			s.colour = l.colour
		}
	}
	s.lumps = keep
}

// cellAt is where cell c's node is in the scene.
func (s *Snake) cellAt(c int) (int, int) {
	return s.ox + snakePitch*(c%s.cw), s.oy + snakePitch*(c/s.cw)
}

// way is the step from cell from to the cell beside it, to.
func (s *Snake) way(from, to int) (int, int) { return to%s.cw - from%s.cw, to/s.cw - from/s.cw }

// stamp draws a shape about cell c's node, for a snake going dx, dy.
func (s *Snake) stamp(sc *Scene, sh shape, c, dx, dy int, ink uint8) {
	x, y := s.cellAt(c)
	for _, p := range sh {
		px, py := turn(p, dx, dy)
		sc.put(x+px, y+py, ink)
	}
}

// Draw is the frame at w × h pixels, in the snake's colour: every cell
// of the body and the link between each two, the head, and the body
// swelling round the apples on their way down; in theirs, those apples —
// in the jaws, in the throat, in the body — and the apple, when it is
// lit. A scene of a new size is a new game.
func (s *Snake) Draw(w, h int) Scene {
	if w != s.w || h != s.h {
		s.reset(w, h)
	}
	sc := newScene(w, h)
	if len(s.body) == 0 {
		return sc
	}
	ink := uint8(1 + s.colour)
	for i, c := range s.body {
		if i == 0 {
			continue // the head's node is the head's to draw: open, it is the apple
		}
		x, y := s.cellAt(c)
		sc.put(x, y, ink)
		px, py := s.cellAt(s.body[i-1])
		for k := 1; k < snakePitch; k++ {
			sc.put(x+(px-x)*k/snakePitch, y+(py-y)*k/snakePitch, ink)
		}
	}
	head, dx, dy := s.head()
	s.stamp(&sc, head, s.body[0], dx, dy, ink)
	for _, l := range s.lumps {
		at := max(l.at, 0) // in the jaws, it is at the head too
		c := s.body[at]
		switch {
		case l.at == 0:
			s.stamp(&sc, snakeGulp, c, dx, dy, ink)
		case l.at > 0:
			ldx, ldy := s.way(c, s.body[at-1])
			s.stamp(&sc, snakeLump, c, ldx, ldy, ink)
		}
		x, y := s.cellAt(c)
		sc.put(x, y, uint8(1+l.colour))
	}
	if s.apple >= 0 && (s.t/s.frames(snakeBlink))%2 == 0 {
		x, y := s.cellAt(s.apple)
		sc.put(x, y, uint8(1+s.next))
	}
	return sc
}
