package saver

import (
	"math/rand/v2"
	"slices"
	"time"
)

// The falling blocks (user, 2026-10-07): the seven tetrominoes, falling
// and stacking, playing themselves, as the snake does — not in a narrow
// well but across the screen, framed in grey — a pixel at the top and
// the sides, two at the foot, the floor the stack stands on (user, the
// same day; it was two all round). A row full from wall to wall goes: it
// flashes white, goes from the middle out to both ends, and the rows
// above come down. A block is two pixels by two (user, the same day): a
// piece is four of them, a 2 × 2 square each. The pieces come in from
// behind the top of the frame, in the middle.
//
// The next piece and the one after are at the field's right, at the top,
// in two boxes one over the other, a pixel a block (user, the same day:
// they were in the field's top corners at its own size, and the field
// was what the boxes left); the boxes are framed as the field is, a
// pixel thick, the side by the field the field's own and the line
// between the two the one. Nothing is lettered: which is which is where
// it is, the next on top.
//
// It plays as a player would, a piece at a time: of every spot the piece
// can get to — round what is in the way, as it can move — it
// takes the one where it lands lowest and leaves fewest holes; and it
// goes there a step a move — a turn or a block across, and a block down
// with it where that keeps the way open. The pieces come seven to a bag,
// one of each in some order, so none is long in coming.
//
// A stack that reaches the top is the end (user, the same day): all of
// it goes black, a row at a time from the top, the frames too — and END
// is lettered in red in the middle of the field, in the lock's own pixel
// letters, a block a pixel; two seconds, and a new game.
//
// Its colours are its own (user, 2026-10-06: a saver of many colours
// brings them): the pieces in the colours they usually have, in
// catppuccin's, the frame in grey, the end in catppuccin's base and its
// red. Its one setting is its speed, by name as the
// snake's.

const KindTetromino = "tetromino"

// tetroSpeeds are the speeds in steps a second: a step is a block, as
// the snake's is a cell, so they are the snake's.
var tetroSpeeds = map[string]int{SpeedSlow: 5, SpeedNormal: 7, SpeedFast: 10, SpeedVeryFast: 14, SpeedSuperFast: 20}

const (
	tetroBlock = 2 // a block's side in pixels (user, 2026-10-07)
	// The frame's width in pixels: one at the top and the sides, two at
	// the foot (user, the same day; it was two all round).
	tetroEdge, tetroFloor = 1, 2
	// A box is six pixels by four inside: the long piece, a pixel a
	// block, and a pixel round it.
	tetroBoxW, tetroBoxH = 6, 4
	// The least field: the usual well's ten blocks across, and as high.
	tetroMinCols, tetroMinRows = 10, 10
	// The clear goes at a pace of its own, whatever the speed: a flash,
	// then the row going from the middle out, a frame a twenty-fifth of a
	// second.
	tetroFx    = 40 * time.Millisecond
	tetroFlash = 280 * time.Millisecond
	tetroWipe  = 400 * time.Millisecond
	// The end at the same pace: all going black in five seconds (user,
	// 2026-10-07: too fast in one), then END for two (user, the same day).
	tetroCurtain = 5 * time.Second
	tetroEnd     = 2 * time.Second
	tetroHide    = 4 // rows over the field a piece may be in, behind the frame
)

// tetroRoom is the least field, its frame and the boxes beside it,
// never drawn larger: a block is two pixels whatever the screen, and a
// larger screen is a wider field.
var tetroRoom = Room{W: 3*tetroEdge + tetroBlock*tetroMinCols + tetroBoxW, H: tetroEdge + tetroFloor + tetroBlock*tetroMinRows, Most: 1}

// The pieces in their boxes as they come in, in the order of their
// colours below: I, O, T, S, Z, J, L.
var tetroArt = [][]string{
	{"....", "####", "....", "...."},
	{"##", "##"},
	{".#.", "###", "..."},
	{".##", "##.", "..."},
	{"##.", ".##", "..."},
	{"#..", "###", "..."},
	{"..#", "###", "..."},
}

// The pieces' colours, as they usually are, in catppuccin-mocha's: the
// long one sky, the square yellow, the T mauve, the S green, the Z red,
// the J blue and the L peach.
var tetroColours = []string{"#89dceb", "#f9e2af", "#cba6f7", "#a6e3a1", "#f38ba8", "#89b4fa", "#fab387"}

const (
	tetroWhite = "#ffffff" // a row going
	tetroGrey  = "#7f849c" // the frame: catppuccin's overlay1
	// The end's black: catppuccin's base, the dark the runner is by night.
	// Crust, darker, is the terminal's own ground, and the squares went
	// into it (user, 2026-10-07: the squares were gone).
	tetroBlack = "#1e1e2e"
	tetroRed   = "#f38ba8" // END: catppuccin's red
)

// The inks past the pieces': ink 1 + p is piece p's.
var (
	inkTetroWhite = uint8(1 + len(tetroColours))
	inkTetroFrame = inkTetroWhite + 1
	inkTetroBlack = inkTetroWhite + 2
	inkTetroRed   = inkTetroWhite + 3
)

// tetroTurns are each piece's blocks in its box, a turn clockwise after
// the other till it is back as it came; the square has the one.
var tetroTurns = func() [][][][2]int {
	out := make([][][][2]int, len(tetroArt))
	for p, art := range tetroArt {
		var blocks [][2]int
		for y, row := range art {
			for x := range row {
				if row[x] == '#' {
					blocks = append(blocks, [2]int{x, y})
				}
			}
		}
		n := len(art)
		for {
			out[p] = append(out[p], blocks)
			next := make([][2]int, len(blocks))
			for i, b := range blocks {
				next[i] = [2]int{n - 1 - b[1], b[0]}
			}
			if sameBlocks(next, out[p][0]) {
				break
			}
			blocks = next
		}
	}
	return out
}()

// sameBlocks says whether a and b are the same blocks, in any order.
func sameBlocks(a, b [][2]int) bool {
	for _, x := range a {
		found := false
		for _, y := range b {
			found = found || x == y
		}
		if !found {
			return false
		}
	}
	return len(a) == len(b)
}

// spot is where a piece is: its turn and its box's top-left, in blocks.
type spot struct{ turn, x, y int }

// Tetromino is one game in progress.
type Tetromino struct {
	rng        *rand.Rand
	w, h       int     // the scene as last drawn; nothing moves before the first draw
	cols, rows int     // the field, in blocks; none when too small
	cells      []uint8 // the stack: a block's ink, 0 none
	bag        []int   // the pieces still to come from this bag
	cur        int     // the piece falling
	next, then int     // the next, in the top box, and the one after, under it
	at, goal   spot    // where the piece is, and where it is going
	dist       []int32 // steps from a spot to the goal; -1 none
	going      []int   // the rows going, the clear under way
	over       bool    // the end under way
	fx         int     // frames of the clear, or of the end, so far
	frame      time.Duration
	end        []string // END, as rows of '#' and '.'
}

// NewTetromino is a game from its first frame, at a speed out of Speeds
// — normal for any other. spell sets a line in the board's pixel font,
// as rows of '#' and '.'. The same seed is the same game.
func NewTetromino(seed uint64, speed string, spell func(string) []string) *Tetromino {
	n, ok := tetroSpeeds[speed]
	if !ok {
		n = tetroSpeeds[SpeedNormal]
	}
	return &Tetromino{
		rng:   rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
		frame: time.Second / time.Duration(n),
		end:   spell("END"),
	}
}

// Room is the least field and its frame.
func (g *Tetromino) Room() Room { return tetroRoom }

// Next is when the next step is due: at the speed, or the clear's pace
// while a row is going or the game ends.
func (g *Tetromino) Next(now time.Time) time.Time {
	if len(g.going) > 0 || g.over {
		return now.Add(tetroFx)
	}
	return now.Add(g.frame)
}

// Inks are the ground, the pieces' colours, white, the frame's grey, and
// the end's black and red.
func (g *Tetromino) Inks() []string {
	return append(append([]string{ownGround}, tetroColours...), tetroWhite, tetroGrey, tetroBlack, tetroRed)
}

// reset is a new game on a w × h scene: the field as wide and as high
// as the frame round it and the boxes beside it leave, empty, and a
// piece coming in. The frame is at the scene's top left; what is left
// over, less than a block, is dark to the right and under it.
func (g *Tetromino) reset(w, h int) {
	g.w, g.h = w, h
	g.cols, g.rows = (w-3*tetroEdge-tetroBoxW)/tetroBlock, (h-tetroEdge-tetroFloor)/tetroBlock
	if g.cols < tetroMinCols || g.rows < tetroMinRows {
		g.cols, g.rows = 0, 0
		return
	}
	g.cells = make([]uint8, g.cols*g.rows)
	g.bag, g.going, g.over, g.fx = nil, nil, false, 0
	g.next, g.then = g.deal(), g.deal()
	g.come()
}

// deal is the next piece from the bag: seven to a bag, one of each.
func (g *Tetromino) deal() int {
	if len(g.bag) == 0 {
		g.bag = g.rng.Perm(len(tetroArt))
	}
	p := g.bag[0]
	g.bag = g.bag[1:]
	return p
}

// open says whether block x, y can take a piece's block: in the field
// and not the stack's, or over it, behind the frame.
func (g *Tetromino) open(x, y int) bool {
	if x < 0 || x >= g.cols || y >= g.rows {
		return false
	}
	return y < 0 || g.cells[y*g.cols+x] == 0
}

// fits says whether the piece can be at s.
func (g *Tetromino) fits(s spot) bool {
	for _, b := range tetroTurns[g.cur][s.turn] {
		if !g.open(s.x+b[0], s.y+b[1]) {
			return false
		}
	}
	return true
}

// spots is how many places a piece can be in: every turn, a box's
// width left of the field to its right edge, the hidden rows over it to
// its foot.
func (g *Tetromino) spots() int { return 4 * (g.cols + 4) * (g.rows + tetroHide) }

// index is s's place among spots, or -1 off them.
func (g *Tetromino) index(s spot) int {
	x, y := s.x+4, s.y+tetroHide
	if s.turn < 0 || s.turn >= 4 || x < 0 || x >= g.cols+4 || y < 0 || y >= g.rows+tetroHide {
		return -1
	}
	return (s.turn*(g.cols+4)+x)*(g.rows+tetroHide) + y
}

// A move is what a piece does in a step: a turn or a block across, or
// neither, and a block down, or not.
type move struct{ turn, dx int }

var tetroMoves = []move{{0, 0}, {1, 0}, {0, -1}, {0, 1}}

// try is a move to try, and whether down with it.
type try struct {
	m    move
	down bool
}

// step is s after m, and a block down when down is set, if the piece
// fits all the way.
func (g *Tetromino) step(s spot, m move, down bool) (spot, bool) {
	n := len(tetroTurns[g.cur])
	t := spot{(s.turn + m.turn) % n, s.x + m.dx, s.y}
	if m != (move{}) && !g.fits(t) {
		return s, false
	}
	if down {
		t.y++
		if !g.fits(t) {
			return s, false
		}
	}
	return t, true
}

// come brings the next piece in, behind the top of the frame in the
// middle, its lowest block a row over the field; and picks
// where it goes, and the way there. A piece that can come to rest only
// over the field, the stack at the top, is the end (Step).
func (g *Tetromino) come() {
	g.cur, g.next, g.then = g.next, g.then, g.deal()
	blocks := tetroTurns[g.cur][0]
	lo, hi, foot := blocks[0][0], blocks[0][0], 0
	for _, b := range blocks {
		lo, hi, foot = min(lo, b[0]), max(hi, b[0]), max(foot, b[1])
	}
	g.at = spot{0, (g.cols-(hi-lo+1))/2 - lo, -1 - foot}
	g.goal = g.choose()
	g.route()
}

// choose is where the piece goes: of the spots it can get to and come
// to rest in, the one the stack is best for.
func (g *Tetromino) choose() spot {
	seen := make([]bool, g.spots())
	seen[g.index(g.at)] = true
	queue := []spot{g.at}
	best, score := g.at, 0
	found := false
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		if _, falls := g.step(s, move{}, true); !falls {
			if v := g.worth(s); !found || v > score {
				best, score, found = s, v, true
			}
		}
		for _, m := range tetroMoves {
			for _, down := range []bool{true, false} {
				t, ok := g.step(s, m, down)
				if i := g.index(t); ok && i >= 0 && !seen[i] {
					seen[i] = true
					queue = append(queue, t)
				}
			}
		}
	}
	return best
}

// worth is how good the stack is with the piece at rest at s, the
// higher the better: the lower it lands and the fewer holes it leaves
// under the stack, with the rows it fills gone — a hole as bad as
// landing two rows higher. How low it lands is what keeps the stack low
// and across the field, not piled up on one side: on a field this wide
// the stack's height counts for nothing else, four blocks more wherever
// they go (2026-10-07: rated by height, holes and how even the stack
// was, the pieces piled up in the corner they came to first; a
// player's six-point rule, Dellacherie's, played no better than these
// two). A piece any of which is over the field is the end, worth less
// than any other.
func (g *Tetromino) worth(s spot) int {
	cells := append([]uint8(nil), g.cells...)
	top, foot := g.rows, 0
	for _, b := range tetroTurns[g.cur][s.turn] {
		x, y := s.x+b[0], s.y+b[1]
		if y < 0 {
			return -1 << 30
		}
		cells[y*g.cols+x] = 1
		top, foot = min(top, y), max(foot, y)
	}
	cells = g.without(cells, g.full(cells))
	holes := 0
	for x := 0; x < g.cols; x++ {
		stacked := false
		for y := 0; y < g.rows; y++ {
			if cells[y*g.cols+x] != 0 {
				stacked = true
			} else if stacked {
				holes++
			}
		}
	}
	// Twice the height it lands at, its middle: rows from the foot.
	lands := 2*g.rows - top - foot
	return -lands - 4*holes
}

// full are the rows of cells with every block taken, the top first.
func (g *Tetromino) full(cells []uint8) []int {
	var out []int
	for y := 0; y < g.rows; y++ {
		whole := true
		for x := 0; x < g.cols && whole; x++ {
			whole = cells[y*g.cols+x] != 0
		}
		if whole {
			out = append(out, y)
		}
	}
	return out
}

// without is cells with the rows gone, the top first, and every row
// over them down as far.
func (g *Tetromino) without(cells []uint8, gone []int) []uint8 {
	out := make([]uint8, len(cells))
	y := g.rows - 1
	for from := g.rows - 1; from >= 0; from-- {
		if slices.Contains(gone, from) {
			continue
		}
		copy(out[y*g.cols:(y+1)*g.cols], cells[from*g.cols:(from+1)*g.cols])
		y--
	}
	return out
}

// route is how many steps every spot is from the goal, backwards from
// it: a spot one step before another is one further.
func (g *Tetromino) route() {
	g.dist = make([]int32, g.spots())
	for i := range g.dist {
		g.dist[i] = -1
	}
	g.dist[g.index(g.goal)] = 0
	queue := []spot{g.goal}
	n := len(tetroTurns[g.cur])
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		d := g.dist[g.index(s)]
		for _, m := range tetroMoves {
			for _, down := range []bool{true, false} {
				if m == (move{}) && !down {
					continue
				}
				// The spot that m, and down, take to s.
				p := s
				if down {
					p.y--
				}
				p = spot{(p.turn - m.turn + n) % n, p.x - m.dx, p.y}
				if t, ok := g.step(p, m, down); !ok || t != s || !g.fits(p) {
					continue
				}
				if i := g.index(p); i >= 0 && g.dist[i] < 0 {
					g.dist[i] = d + 1
					queue = append(queue, p)
				}
			}
		}
	}
}

// toward is the piece's step to its goal, one nearer: of the steps that
// are, a turn while it is not turned as it will lie, else a block across
// towards it — with a block down at once, then down alone, then the turn
// or across alone; so it goes over as it falls where the way stays open.
// Any other step nearer, failing those.
func (g *Tetromino) toward() spot {
	d := g.dist[g.index(g.at)]
	m := move{}
	switch {
	case g.at.turn != g.goal.turn:
		m.turn = 1
	case g.at.x < g.goal.x:
		m.dx = 1
	case g.at.x > g.goal.x:
		m.dx = -1
	}
	tries := []try{{m, true}, {move{}, true}, {m, false}}
	for _, o := range tetroMoves {
		tries = append(tries, try{o, true}, try{o, false})
	}
	for _, t := range tries {
		if t.m == (move{}) && !t.down {
			continue
		}
		if s, ok := g.step(g.at, t.m, t.down); ok {
			if i := g.index(s); i >= 0 && g.dist[i] == d-1 {
				return s
			}
		}
	}
	return g.goal
}

// Step is a step of the game: the clear or the end on a frame — the end
// over, a new game; or the piece a step on to its goal; or, there, set
// in the stack — the rows it fills going, or, reaching over the field,
// the end — and the next coming in.
func (g *Tetromino) Step() {
	if g.w == 0 || g.cols == 0 {
		return
	}
	if g.over {
		if g.fx++; g.fx >= g.frames(tetroCurtain)+g.frames(tetroEnd) {
			g.reset(g.w, g.h)
		}
		return
	}
	if len(g.going) > 0 {
		if g.fx++; g.fx >= g.frames(tetroFlash)+g.frames(tetroWipe) {
			g.cells = g.without(g.cells, g.going)
			g.going, g.fx = nil, 0
			g.come()
		}
		return
	}
	if g.at != g.goal {
		g.at = g.toward()
		return
	}
	over := false
	for _, b := range tetroTurns[g.cur][g.at.turn] {
		x, y := g.at.x+b[0], g.at.y+b[1]
		if y < 0 {
			over = true
			continue
		}
		g.cells[y*g.cols+x] = uint8(1 + g.cur)
	}
	if over {
		g.over, g.fx = true, 0
		return
	}
	if g.going = g.full(g.cells); len(g.going) == 0 {
		g.come()
	}
}

// frames is how many of the clear's frames d is; one at least.
func (g *Tetromino) frames(d time.Duration) int { return max(1, int(d/tetroFx)) }

// wiped says whether block x of a row going is gone by now: the row
// goes from the middle of the field out, the blocks the same distance
// from the ends at once, after the flash.
func (g *Tetromino) wiped(x int) bool {
	k := g.fx - g.frames(tetroFlash) + 1
	if k <= 0 {
		return false
	}
	rings := (g.cols-1)/2 + 1
	gone := (rings*k + g.frames(tetroWipe) - 1) / g.frames(tetroWipe)
	return min(x, g.cols-1-x) >= rings-gone
}

// size is the field's frame's, the field in it, in pixels.
func (g *Tetromino) size() (int, int) {
	return 2*tetroEdge + tetroBlock*g.cols, tetroEdge + tetroFloor + tetroBlock*g.rows
}

// block lights block x, y of the field in ink.
func (g *Tetromino) block(sc *Scene, x, y int, ink uint8) {
	px, py := tetroEdge+tetroBlock*x, tetroEdge+tetroBlock*y
	for dy := 0; dy < tetroBlock; dy++ {
		for dx := 0; dx < tetroBlock; dx++ {
			sc.put(px+dx, py+dy, ink)
		}
	}
}

// preview draws piece p as it comes in, a pixel a block, in the middle
// of the box whose inside's top left is x, y.
func (g *Tetromino) preview(sc *Scene, p, x, y int) {
	blocks := tetroTurns[p][0]
	lx, hx, ly, hy := blocks[0][0], blocks[0][0], blocks[0][1], blocks[0][1]
	for _, b := range blocks {
		lx, hx, ly, hy = min(lx, b[0]), max(hx, b[0]), min(ly, b[1]), max(hy, b[1])
	}
	px, py := x+(tetroBoxW-(hx-lx+1))/2, y+(tetroBoxH-(hy-ly+1))/2
	for _, b := range blocks {
		sc.put(px+b[0]-lx, py+b[1]-ly, uint8(1+p))
	}
}

// Draw is the frame at w × h pixels: the frames in grey, the next two
// pieces in the boxes, the stack — the rows going
// white, as far as they have not gone — and the piece falling, what of
// it is in the field; or, the game over, as much of all that as is not
// black yet, and END. A scene of a new size is a new game.
func (g *Tetromino) Draw(w, h int) Scene {
	if w != g.w || h != g.h {
		g.reset(w, h)
	}
	sc := newScene(w, h)
	if g.cols == 0 {
		return sc
	}
	fw, fh := g.size()
	for x := 0; x < fw; x++ {
		for y := 0; y < fh; y++ {
			if y < tetroEdge || y >= fh-tetroFloor || x < tetroEdge || x >= fw-tetroEdge {
				sc.put(x, y, inkTetroFrame)
			}
		}
	}
	// The boxes at the field's right, its side theirs: their top, the
	// line between them and their foot, and their far side.
	bx, bw, bh := fw-1, tetroBoxW+tetroEdge, tetroBoxH+tetroEdge
	for i := 0; i <= 2; i++ {
		for x := 0; x <= bw; x++ {
			sc.put(bx+x, i*bh, inkTetroFrame)
		}
	}
	for y := 0; y <= 2*bh; y++ {
		sc.put(bx+bw, y, inkTetroFrame)
	}
	g.preview(&sc, g.next, fw, tetroEdge)
	g.preview(&sc, g.then, fw, tetroEdge+bh)
	for y := 0; y < g.rows; y++ {
		going := slices.Contains(g.going, y)
		for x := 0; x < g.cols; x++ {
			ink := g.cells[y*g.cols+x]
			switch {
			case ink == 0:
			case !going:
				g.block(&sc, x, y, ink)
			case !g.wiped(x):
				g.block(&sc, x, y, inkTetroWhite)
			}
		}
	}
	if len(g.going) == 0 {
		for _, b := range tetroTurns[g.cur][g.at.turn] {
			if y := g.at.y + b[1]; y >= 0 {
				g.block(&sc, g.at.x+b[0], y, uint8(1+g.cur))
			}
		}
	}
	if g.over {
		g.drawEnd(&sc)
	}
	return sc
}

// drawEnd blacks the whole scene out from the top, the frames and all, a
// row at a time, as far as the end has gone — five seconds for the whole
// — and, then, letters END in red in the middle of the field.
func (g *Tetromino) drawEnd(sc *Scene) {
	fw, fh := g.size()
	n := g.frames(tetroCurtain)
	dark := min(sc.H, (sc.H*(g.fx+1)+n-1)/n)
	for y := 0; y < dark; y++ {
		for x := 0; x < sc.W; x++ {
			sc.put(x, y, inkTetroBlack)
		}
	}
	if g.fx < n {
		return
	}
	px := (fw - tetroBlock*len(g.end[0])) / 2
	py := (fh - tetroBlock*len(g.end)) / 2
	for dy, row := range g.end {
		for dx := 0; dx < len(row); dx++ {
			if row[dx] != '#' {
				continue
			}
			for k := 0; k < tetroBlock*tetroBlock; k++ {
				sc.put(px+tetroBlock*dx+k%tetroBlock, py+tetroBlock*dy+k/tetroBlock, inkTetroRed)
			}
		}
	}
}
