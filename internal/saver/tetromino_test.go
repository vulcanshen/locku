package saver

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// tetroBlockAt is the ink of block x, y of the field, its four pixels
// alike, or the test fails.
func tetroBlockAt(t *testing.T, g *Tetromino, sc Scene, x, y int) uint8 {
	t.Helper()
	px, py := g.ox+tetroBorder+tetroBlock*x, g.oy+tetroBorder+tetroBlock*y
	ink := sc.Pix[py*sc.W+px]
	for dy := 0; dy < tetroBlock; dy++ {
		for dx := 0; dx < tetroBlock; dx++ {
			if got := sc.Pix[(py+dy)*sc.W+px+dx]; got != ink {
				t.Fatalf("block %d,%d is not one ink: %d and %d", x, y, ink, got)
			}
		}
	}
	return ink
}

// tetroSpell sets END as the lock's 3x5 letters do.
func tetroSpell(line string) []string {
	if line != "END" {
		panic(line)
	}
	return []string{
		"###.###.##.",
		"#...#.#.#.#",
		"##..#.#.#.#",
		"#...#.#.#.#",
		"###.#.#.##.",
	}
}

// tetroGame is a game drawn once at w × h, its first piece coming in.
func tetroGame(seed uint64, w, h int) *Tetromino {
	g := NewTetromino(seed, SpeedNormal, tetroSpell)
	g.Draw(w, h)
	return g
}

// Seven pieces of four blocks each; a turn is the one before turned a
// quarter clockwise in its box, till it is back as it came — the square
// once, the others four times.
func TestTetrominoPieces(t *testing.T) {
	if len(tetroTurns) != 7 || len(tetroColours) != 7 {
		t.Fatalf("%d pieces, %d colours", len(tetroTurns), len(tetroColours))
	}
	for p, turns := range tetroTurns {
		want := 4
		if p == 1 {
			want = 1
		}
		if len(turns) != want {
			t.Errorf("piece %d has %d turns, want %d", p, len(turns), want)
		}
		n := len(tetroArt[p])
		for i, blocks := range turns {
			if len(blocks) != 4 {
				t.Errorf("piece %d turn %d has %d blocks", p, i, len(blocks))
			}
			next := turns[(i+1)%len(turns)]
			for _, b := range blocks {
				if r := [2]int{n - 1 - b[1], b[0]}; !slices.Contains(next, r) {
					t.Errorf("piece %d turn %d: %v turned is %v, not in the next turn %v", p, i, b, r, next)
				}
			}
		}
	}
	// The long piece lies across its second row, then stands in its third
	// column.
	if !sameBlocks(tetroTurns[0][0], [][2]int{{0, 1}, {1, 1}, {2, 1}, {3, 1}}) ||
		!sameBlocks(tetroTurns[0][1], [][2]int{{2, 0}, {2, 1}, {2, 2}, {2, 3}}) {
		t.Errorf("the long piece turns as %v", tetroTurns[0][:2])
	}
}

// The pieces come seven to a bag: every seven from the first are one of
// each.
func TestTetrominoBagOfSeven(t *testing.T) {
	g := NewTetromino(3, SpeedNormal, tetroSpell)
	for bag := 0; bag < 20; bag++ {
		var got []int
		for i := 0; i < 7; i++ {
			got = append(got, g.deal())
		}
		slices.Sort(got)
		if !slices.Equal(got, []int{0, 1, 2, 3, 4, 5, 6}) {
			t.Fatalf("bag %d is %v", bag, got)
		}
	}
}

// The frame is two pixels round the whole field, grey, and the boxes in
// the top corners are walled as it is, a block thick, inside it: as the
// user drew it, a block a character.
func TestTetrominoFrameAndBoxes(t *testing.T) {
	g := tetroGame(1, 60, 35)
	sc := g.Draw(60, 35)
	if g.cols != 28 || g.rows != 15 || g.ox != 0 || g.oy != 0 {
		t.Fatalf("field %dx%d at %d,%d, want 28x15 at 0,0", g.cols, g.rows, g.ox, g.oy)
	}
	want := []string{
		"##############################",
		"#......#..............#......#",
		"#......#..............#......#",
		"#......#..............#......#",
		"#......#..............#......#",
		"########..............########",
		"#............................#",
		"#............................#",
		"#............................#",
		"#............................#",
		"#............................#",
		"#............................#",
		"#............................#",
		"#............................#",
		"#............................#",
		"#............................#",
		"##############################",
	}
	var got []string
	for y := 0; y < 34; y += 2 {
		var b strings.Builder
		for x := 0; x < 60; x += 2 {
			if sc.Pix[y*60+x] == inkTetroFrame {
				b.WriteByte('#')
			} else {
				b.WriteByte('.')
			}
		}
		got = append(got, b.String())
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("the frame:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// Two pixels thick, the frame and the walls alike; the pixel row left
	// over is under it, dark.
	for _, p := range [][2]int{{30, 0}, {30, 1}, {0, 20}, {1, 20}, {58, 20}, {59, 20}, {30, 32}, {30, 33}, {14, 6}, {15, 6}, {6, 10}, {6, 11}} {
		if sc.Pix[p[1]*60+p[0]] != inkTetroFrame {
			t.Errorf("pixel %v is not the frame", p)
		}
	}
	for _, p := range [][2]int{{30, 2}, {2, 20}, {57, 20}, {30, 31}, {16, 6}, {6, 12}, {30, 34}} {
		if sc.Pix[p[1]*60+p[0]] == inkTetroFrame {
			t.Errorf("pixel %v is the frame", p)
		}
	}
	if g.Room() != (Room{W: 40, H: 22, Most: 1}) {
		t.Errorf("room %+v", g.Room())
	}
}

// The left box holds the next piece and the right the one after, each
// as it will come in, in its colour, in the middle of the box.
func TestTetrominoBoxesHoldTheNextTwo(t *testing.T) {
	for seed := uint64(1); seed <= 7; seed++ {
		g := tetroGame(seed, 60, 35)
		sc := g.Draw(60, 35)
		for _, box := range []struct{ piece, x int }{{g.next, 0}, {g.then, g.cols - tetroBoxW}} {
			px, py := tetroBorder+tetroBlock*box.x, tetroBorder
			var lit [][2]int
			for y := 0; y < tetroBlock*tetroBoxH; y++ {
				for x := 0; x < tetroBlock*tetroBoxW; x++ {
					switch sc.Pix[(py+y)*60+px+x] {
					case 0:
					case uint8(1 + box.piece):
						lit = append(lit, [2]int{x, y})
					default:
						t.Fatalf("seed %d: box at %d has ink %d", seed, box.x, sc.Pix[(py+y)*60+px+x])
					}
				}
			}
			if len(lit) != 16 {
				t.Fatalf("seed %d: box at %d has %d pixels lit, want the 16 of a piece", seed, box.x, len(lit))
			}
			lx, hx, ly, hy := lit[0][0], lit[0][0], lit[0][1], lit[0][1]
			for _, p := range lit {
				lx, hx, ly, hy = min(lx, p[0]), max(hx, p[0]), min(ly, p[1]), max(hy, p[1])
			}
			if lx != tetroBlock*tetroBoxW-1-hx || ly != tetroBlock*tetroBoxH-1-hy {
				t.Errorf("seed %d: piece %d in box %d not in the middle: x %d…%d, y %d…%d", seed, box.piece, box.x, lx, hx, ly, hy)
			}
			// And it is the piece as it comes in, a block two pixels square.
			blocks := tetroTurns[box.piece][0]
			bx, by := blocks[0][0], blocks[0][1]
			for _, b := range blocks {
				bx, by = min(bx, b[0]), min(by, b[1])
			}
			for _, b := range blocks {
				for d := 0; d < 4; d++ {
					p := [2]int{lx + tetroBlock*(b[0]-bx) + d%2, ly + tetroBlock*(b[1]-by) + d/2}
					if !slices.Contains(lit, p) {
						t.Errorf("seed %d: piece %d in box %d: pixel %v dark", seed, box.piece, box.x, p)
					}
				}
			}
		}
		// The next piece comes in next, and the one after moves over.
		next, then := g.next, g.then
		g.come()
		if g.cur != next || g.next != then {
			t.Errorf("seed %d: came %d, next %d; want %d, %d", seed, g.cur, g.next, next, then)
		}
	}
}

// A game played a while: a piece comes in behind the top of the frame
// in the middle of the gap, and goes a step at a time — a turn or a
// block across, and a block down or not, never up — sometimes across and
// down at once; it never overlaps the stack, a wall or a box; the stack
// never gets into a box; rows fill and go. It plays well: the stack
// stays low, under half the field, with few holes in it (2026-10-07:
// it was six rows high at most and two holes, where without minding how
// high a piece lands it filled the field).
func TestTetrominoPlays(t *testing.T) {
	g := tetroGame(5, 60, 35)
	if g.open(tetroBoxW, -1) || g.open(g.cols-1-tetroBoxW, -1) || !g.open(tetroBoxW+1, -1) || !g.open(g.cols-2-tetroBoxW, -1) {
		t.Error("over the field, a piece may be in the gap and nowhere else")
	}
	cleared, diagonal, pieces := 0, 0, 0
	for i := 0; i < 3000; i++ {
		before, cur, going := g.at, g.cur, len(g.going)
		g.Step()
		g.Draw(60, 35)
		high, holes := 0, 0
		for x := 0; x < g.cols; x++ {
			stacked := false
			for y := 0; y < g.rows; y++ {
				switch {
				case g.cells[y*g.cols+x] != 0:
					high, stacked = max(high, g.rows-y), true
				case stacked:
					holes++
				}
			}
		}
		if high > 7 || holes > 5 {
			t.Fatalf("step %d: the stack %d high, %d holes", i, high, holes)
		}
		for y := 0; y < g.rows; y++ {
			for x := 0; x < g.cols; x++ {
				if g.boxed(x, y) && g.cells[y*g.cols+x] != 0 {
					t.Fatalf("step %d: the stack in a box at %d,%d", i, x, y)
				}
			}
		}
		switch {
		case going == 0 && len(g.going) > 0:
			cleared += len(g.going)
		case len(g.going) > 0 || going > 0:
		case g.cur != cur || g.at.y < before.y:
			// A piece came in: hidden, its lowest block a row over the
			// field, in the middle of the gap.
			pieces++
			blocks := tetroTurns[g.cur][g.at.turn]
			lo, hi, foot := g.cols, 0, -9
			for _, b := range blocks {
				lo, hi, foot = min(lo, g.at.x+b[0]), max(hi, g.at.x+b[0]), max(foot, g.at.y+b[1])
			}
			if foot != -1 || lo-(tetroBoxW+1) != (g.cols-tetroBoxW-2)-hi && lo-(tetroBoxW+1) != (g.cols-tetroBoxW-2)-hi-1 {
				t.Fatalf("step %d: piece %d came in at x %d…%d, foot %d", i, g.cur, lo, hi, foot)
			}
		default:
			dx, dy := g.at.x-before.x, g.at.y-before.y
			turned := g.at.turn != before.turn
			if dy < 0 || dy > 1 || dx < -1 || dx > 1 || turned && dx != 0 || g.at == before && g.at != g.goal {
				t.Fatalf("step %d: %+v to %+v", i, before, g.at)
			}
			if dx != 0 && dy == 1 {
				diagonal++
			}
			if !g.fits(g.at) {
				t.Fatalf("step %d: the piece does not fit at %+v", i, g.at)
			}
		}
	}
	if cleared < 10 || diagonal < 50 || pieces < 100 {
		t.Errorf("%d rows cleared, %d steps across and down, %d pieces", cleared, diagonal, pieces)
	}
}

// A row full goes in white: the flash, then from the middle out, the
// blocks the same distance from the ends at once; then the rows over it
// come down, and the next piece comes in. The clear is at its own pace
// whatever the speed.
func TestTetrominoClear(t *testing.T) {
	g := tetroGame(2, 60, 35)
	foot := g.rows - 1
	for x := 0; x < g.cols; x++ {
		if x < 10 || x > 13 {
			g.cells[foot*g.cols+x] = 3
		}
	}
	g.cells[(foot-1)*g.cols+2] = 5
	g.cur, g.at = 0, spot{0, 10, foot - 1} // the long piece, lying in the gap
	g.goal = g.at
	if !g.fits(g.at) {
		t.Fatal("the long piece does not fit")
	}
	g.Step()
	if !slices.Equal(g.going, []int{foot}) || g.fx != 0 {
		t.Fatalf("going %v, fx %d", g.going, g.fx)
	}
	if g.Next(time.Unix(0, 0)) != time.Unix(0, 0).Add(40*time.Millisecond) {
		t.Errorf("the clear's next frame at %v", g.Next(time.Unix(0, 0)))
	}
	flash, wipe := g.frames(tetroFlash), g.frames(tetroWipe)
	if flash != 7 || wipe != 10 {
		t.Fatalf("flash %d frames, wipe %d", flash, wipe)
	}
	row := func() string {
		sc := g.Draw(60, 35)
		var b strings.Builder
		for x := 0; x < g.cols; x++ {
			switch tetroBlockAt(t, g, sc, x, foot) {
			case inkTetroWhite:
				b.WriteByte('W')
			case 0:
				b.WriteByte('.')
			default:
				b.WriteByte('?')
			}
		}
		return b.String()
	}
	white := strings.Repeat("W", 28)
	for i := 0; i < flash; i++ {
		if got := row(); got != white {
			t.Fatalf("flash frame %d: %s", i, got)
		}
		g.Step()
	}
	// Fourteen blocks from the middle to each end, in ten frames.
	wants := []string{
		"WWWWWWWWWWWW....WWWWWWWWWWWW",
		"WWWWWWWWWWW......WWWWWWWWWWW",
		"WWWWWWWWW..........WWWWWWWWW",
		"WWWWWWWW............WWWWWWWW",
		"WWWWWWW..............WWWWWWW",
		"WWWWW..................WWWWW",
		"WWWW....................WWWW",
		"WW........................WW",
		"W..........................W",
		"............................",
	}
	for i, want := range wants {
		if got := row(); got != want {
			t.Fatalf("wipe frame %d:\n%s\nwant\n%s", i, got, want)
		}
		g.Step()
	}
	if len(g.going) != 0 {
		t.Fatalf("still going: %v", g.going)
	}
	for x := 0; x < g.cols; x++ {
		want := uint8(0)
		if x == 2 {
			want = 5
		}
		if g.cells[foot*g.cols+x] != want || g.cells[(foot-1)*g.cols+x] != 0 {
			t.Fatalf("after the clear, column %d: %d over %d", x, g.cells[(foot-1)*g.cols+x], g.cells[foot*g.cols+x])
		}
	}
	if g.at.y >= 0 {
		t.Errorf("no piece came in: %+v", g.at)
	}
}

// A row in the gap between the boxes is full when the gap is, and goes
// from the middle of the gap; the rows under the boxes come down with
// the rest.
func TestTetrominoClearInTheGap(t *testing.T) {
	g := tetroGame(2, 60, 35)
	for x := tetroBoxW + 1; x < g.cols-tetroBoxW-1; x++ {
		g.cells[2*g.cols+x] = 4
	}
	g.cells[1*g.cols+9] = 6
	if full := g.full(g.cells); !slices.Equal(full, []int{2}) {
		t.Fatalf("full rows %v", full)
	}
	g.going, g.fx = []int{2}, g.frames(tetroFlash)
	sc := g.Draw(60, 35)
	var b strings.Builder
	for x := tetroBoxW + 1; x < g.cols-tetroBoxW-1; x++ {
		if tetroBlockAt(t, g, sc, x, 2) == inkTetroWhite {
			b.WriteByte('W')
		} else {
			b.WriteByte('.')
		}
	}
	if b.String() != "WWWWWW..WWWWWW" {
		t.Errorf("the gap's row, the first frame it goes: %s", b.String())
	}
	got := g.without(g.cells, []int{2})
	if got[2*g.cols+9] != 6 || got[1*g.cols+9] != 0 {
		t.Errorf("the block over the row did not come down")
	}
}

// Its best spot under a box, the piece gets there: down past the box,
// then across under it.
func TestTetrominoGoesUnderTheBoxes(t *testing.T) {
	g := tetroGame(4, 60, 35)
	foot := g.rows - 1
	for x := 4; x < g.cols; x++ {
		g.cells[foot*g.cols+x] = 2
	}
	g.cur = 0
	g.at = spot{0, 12, -2}
	g.goal = g.choose()
	g.route()
	// Lying in the four blocks of the bottom row free, under the box.
	for _, b := range tetroTurns[0][g.goal.turn] {
		if x, y := g.goal.x+b[0], g.goal.y+b[1]; x > 3 || y != foot {
			t.Fatalf("goal %+v: a block at %d,%d", g.goal, x, y)
		}
	}
	for i := 0; i < 100 && len(g.going) == 0; i++ {
		g.Step()
	}
	if !slices.Equal(g.going, []int{foot}) {
		t.Errorf("going %v", g.going)
	}
}

// A spot is rated with the rows it fills gone: a hole under one of them
// is a hole no more.
func TestTetrominoRatesWithTheFullRowsGone(t *testing.T) {
	g := tetroGame(4, 60, 35)
	foot := g.rows - 1
	for x := 0; x < g.cols; x++ {
		if x != 2 {
			g.cells[foot*g.cols+x] = 2
		}
		if x < 10 || x > 13 {
			g.cells[(foot-1)*g.cols+x] = 2
		}
	}
	g.cur = 0
	s := spot{0, 10, foot - 2} // the long piece, lying in the row over the hole
	if !g.fits(s) {
		t.Fatal("the long piece does not fit")
	}
	// Landing on the second row, twice its height is four; a hole would
	// be four more.
	if got, want := g.worth(s), -4; got != want {
		t.Errorf("worth %d, want %d: landing a row up, no hole", got, want)
	}
}

// A piece that can come to rest in the field never rests over it, the
// end, where it has the choice.
func TestTetrominoKeepsOffTheTop(t *testing.T) {
	g := tetroGame(4, 60, 35)
	for x := tetroBoxW + 1; x <= 12; x++ {
		g.cells[x] = 2 // the top row, under where the piece comes in
	}
	g.at = spot{0, 12, -2}
	g.goal = g.choose()
	for _, b := range tetroTurns[g.cur][g.goal.turn] {
		if g.goal.y+b[1] < 0 {
			t.Fatalf("goal %+v, over the field", g.goal)
		}
	}
}

// A stack that reaches the top is the end (user, 2026-10-07): all of it
// goes black, a block's row at a time from the top in a second, the
// frame and the boxes too; then END in red in the middle, a block a
// pixel of the lock's letters, for two seconds; then a new game. It is
// at the clear's pace.
func TestTetrominoEnds(t *testing.T) {
	g := tetroGame(6, 60, 35)
	for y := 0; y < g.rows; y++ {
		hole := 0
		if y <= tetroBoxH {
			hole = tetroBoxW + 1 + 3*(y%2)
		}
		for x := 0; x < g.cols; x++ {
			if !g.boxed(x, y) && x != hole {
				g.cells[y*g.cols+x] = 1
			}
		}
	}
	g.at = spot{0, 12, -2}
	g.goal = g.choose()
	g.route()
	g.Step()
	if !g.over || len(g.going) != 0 {
		t.Fatalf("over %v, going %v", g.over, g.going)
	}
	if g.Next(time.Unix(0, 0)) != time.Unix(0, 0).Add(40*time.Millisecond) {
		t.Errorf("the end's next frame at %v", g.Next(time.Unix(0, 0)))
	}
	curtain, end := g.frames(tetroCurtain), g.frames(tetroEnd)
	if curtain != 25 || end != 50 {
		t.Fatalf("curtain %d frames, END %d", curtain, end)
	}
	// dark is how many pixel rows from the top are black, all across the
	// frame, and the rest not.
	dark := func(sc Scene) int {
		n := 0
		for n < 34 && sc.Pix[n*60] == inkTetroBlack {
			n++
		}
		for y := 0; y < 34; y++ {
			for x := 0; x < 60; x++ {
				if (sc.Pix[y*60+x] == inkTetroBlack) != (y < n) {
					t.Fatalf("%d rows black, and pixel %d,%d is %d", n, x, y, sc.Pix[y*60+x])
				}
			}
		}
		return n
	}
	// Seventeen rows of blocks, the frame's two and the field's fifteen,
	// in twenty-five frames: the frame it ends, the frame's top; ten
	// frames in, seven of them.
	if n := dark(g.Draw(60, 35)); n != 2 {
		t.Fatalf("the first frame of the end: %d pixel rows black", n)
	}
	for i := 0; i < 9; i++ {
		g.Step()
	}
	sc := g.Draw(60, 35)
	if n := dark(sc); n != 14 || slices.Contains(sc.Pix, inkTetroRed) {
		t.Fatalf("ten frames in: %d pixel rows black", n)
	}
	for g.fx < curtain-1 {
		g.Step()
	}
	sc = g.Draw(60, 35)
	if n := dark(sc); n != 34 || slices.Contains(sc.Pix, inkTetroRed) {
		t.Fatalf("the curtain down: %d pixel rows black", n)
	}
	g.Step()
	// END, eleven pixels by five of the letters, twenty-two by ten in
	// the scene, in the middle of the field's fifty-six by thirty.
	sc = g.Draw(60, 35)
	art := tetroSpell("END")
	for y := 0; y < 35; y++ {
		for x := 0; x < 60; x++ {
			ax, ay := (x-2-17)/2, (y-2-10)/2
			lit := x >= 19 && y >= 12 && ax < len(art[0]) && ay < len(art) && art[ay][ax] == '#'
			if (sc.Pix[y*60+x] == inkTetroRed) != lit || !lit && y < 34 && sc.Pix[y*60+x] != inkTetroBlack {
				t.Fatalf("END at pixel %d,%d: %d", x, y, sc.Pix[y*60+x])
			}
		}
	}
	for i := 0; i < end-1; i++ {
		g.Step()
	}
	if !g.over || !slices.Contains(g.Draw(60, 35).Pix, inkTetroRed) {
		t.Fatal("END went before two seconds")
	}
	g.Step()
	if g.over || slices.ContainsFunc(g.cells, func(c uint8) bool { return c != 0 }) || g.at.y >= 0 {
		t.Fatalf("no new game: over %v, at %+v", g.over, g.at)
	}
	if slices.Contains(g.Draw(60, 35).Pix, inkTetroBlack) {
		t.Error("the new game is black")
	}
}

// Its speeds are the snake's, a block a step; any other is normal.
func TestTetrominoSpeed(t *testing.T) {
	now := time.Unix(0, 0)
	for speed, n := range map[string]int{SpeedSlow: 5, SpeedNormal: 7, SpeedFast: 10, SpeedVeryFast: 14, SpeedSuperFast: 20, "": 7, "7": 7} {
		g := NewTetromino(1, speed, tetroSpell)
		if got := g.Next(now).Sub(now); got != time.Second/time.Duration(n) {
			t.Errorf("speed %q: a step every %v, want %v", speed, got, time.Second/time.Duration(n))
		}
	}
}

// The same seed is the same game; a scene of a new size is a new one.
// Too small a scene has no game in it, and nothing drawn.
func TestTetrominoIsTheSeedsAndStartsOverOnAResize(t *testing.T) {
	a, b := tetroGame(9, 60, 35), tetroGame(9, 60, 35)
	for i := 0; i < 500; i++ {
		a.Step()
		b.Step()
		if !slices.Equal(a.Draw(60, 35).Pix, b.Draw(60, 35).Pix) {
			t.Fatalf("step %d: two games of one seed differ", i)
		}
	}
	a.Draw(80, 40)
	if a.cols != 38 || a.rows != 18 || slices.ContainsFunc(a.cells, func(c uint8) bool { return c != 0 }) {
		t.Errorf("after a resize: %dx%d, stack %v", a.cols, a.rows, slices.ContainsFunc(a.cells, func(c uint8) bool { return c != 0 }))
	}
	for _, s := range [][2]int{{39, 40}, {60, 21}} {
		g := tetroGame(1, s[0], s[1])
		g.Step()
		if sc := g.Draw(s[0], s[1]); slices.ContainsFunc(sc.Pix, func(c uint8) bool { return c != 0 }) {
			t.Errorf("%dx%d: something drawn", s[0], s[1])
		}
	}
	g := tetroGame(1, 40, 22)
	if sc := g.Draw(40, 22); sc.Pix[0] != inkTetroFrame {
		t.Errorf("40x22: no frame")
	}
	NewTetromino(1, SpeedNormal, tetroSpell).Step() // before the first draw: nothing, and no panic
}

// Its colours: the ground, the pieces' as they usually are, white for a
// row going, grey for the frame, and the end's black and red.
func TestTetrominoInks(t *testing.T) {
	want := []string{"#313244", "#89dceb", "#f9e2af", "#cba6f7", "#a6e3a1", "#f38ba8", "#89b4fa", "#fab387", "#ffffff", "#7f849c", "#11111b", "#f38ba8"}
	if got := NewTetromino(1, "", tetroSpell).Inks(); !slices.Equal(got, want) {
		t.Errorf("inks %v", got)
	}
}
