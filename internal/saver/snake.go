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
// merely lie side by side can be told apart, as on the Nokia's screen;
// the apple blinks. Each frame the head and the tail move, nothing else.
//
// It lights in one ink: its colours are the profile's bg / fg.

const KindSnake = "snake"

// SnakeFrame is the time between two moves: twelve and a half a second.
const SnakeFrame = 80 * time.Millisecond

const (
	snakeStart = 3  // cells long at the start
	snakeHold  = 38 // frames a full board stays, three seconds
	snakeBlink = 4  // frames the apple is lit, then dark
)

// snakeRoom is a board of sixteen cells by ten, a pixel a node and a
// pixel a link, drawn at most three times over.
var snakeRoom = Room{W: 31, H: 19, Most: 3}

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
	apple  int    // -1 when the board is full
	t      int    // frames played
	hold   int    // frames a full board has still to stay
}

// NewSnake is a game from its first frame. The same seed is the same
// game.
func NewSnake(seed uint64) *Snake {
	return &Snake{rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))}
}

// Room is the board the game needs.
func (s *Snake) Room() Room { return snakeRoom }

// Next is when the next move is due.
func (s *Snake) Next(now time.Time) time.Time { return now.Add(SnakeFrame) }

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
// snakeStart long somewhere on it, and an apple.
func (s *Snake) reset(w, h int) {
	s.w, s.h = w, h
	s.cw, s.ch = (w+1)/2, (h+1)/2
	s.tour = hamilton(s.cw, s.ch)
	s.pos = make([]int, s.cw*s.ch)
	for i := range s.pos {
		s.pos[i] = -1
	}
	for i, c := range s.tour {
		s.pos[c] = i
	}
	s.body, s.taken, s.apple, s.hold = nil, make([]bool, s.cw*s.ch), -1, 0
	n := len(s.tour)
	if n < snakeStart+1 {
		s.tour = nil
		return
	}
	// The grid in the middle of the scene: an odd column left out is
	// dark on either side, as near as it halves.
	used := s.cw
	if s.ch%2 != 0 && s.cw%2 != 0 {
		used--
	}
	s.ox, s.oy = (w-(2*used-1))/2, (h-(2*s.ch-1))/2
	p := s.rng.IntN(n)
	for i := 0; i < snakeStart; i++ {
		c := s.tour[(p-i+n)%n]
		s.body = append(s.body, c)
		s.taken[c] = true
	}
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
// eaten, and a new apple. A full board stays a while, then a new game.
func (s *Snake) Step() {
	if s.w == 0 || s.tour == nil {
		return
	}
	s.t++
	if s.apple < 0 {
		if s.hold--; s.hold <= 0 {
			s.reset(s.w, s.h)
		}
		return
	}
	next := s.move()
	if next != s.apple {
		tail := s.body[len(s.body)-1]
		s.body = s.body[:len(s.body)-1]
		s.taken[tail] = false
	}
	s.body = append([]int{next}, s.body...)
	s.taken[next] = true
	if next == s.apple {
		if s.place(); s.apple < 0 {
			s.hold = snakeHold
		}
	}
}

// cellAt is where cell c's node is in the scene.
func (s *Snake) cellAt(c int) (int, int) { return s.ox + 2*(c%s.cw), s.oy + 2*(c/s.cw) }

// Draw is the frame at w × h pixels: every cell of the body and the link
// between each two, and the apple when it is lit. A scene of a new size
// is a new game.
func (s *Snake) Draw(w, h int) Scene {
	if w != s.w || h != s.h {
		s.reset(w, h)
	}
	sc := newScene(w, h)
	for i, c := range s.body {
		x, y := s.cellAt(c)
		sc.set(x, y)
		if i > 0 {
			px, py := s.cellAt(s.body[i-1])
			sc.set((x+px)/2, (y+py)/2)
		}
	}
	if s.apple >= 0 && (s.t/snakeBlink)%2 == 0 {
		sc.set(s.cellAt(s.apple))
	}
	return sc
}
