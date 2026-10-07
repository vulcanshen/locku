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
	px, py := tetroEdge+tetroBlock*x, tetroEdge+tetroBlock*y
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

// The frame is grey round the whole field, a pixel thick (user,
// 2026-10-07; it was two all round, then two at the foot); the boxes for the pieces to come are a column at its
// right, framed a pixel thick, the field's side theirs and one line
// between two boxes, as many as the frame's height holds whole, the
// rest under them dark (user, the same day; they were in the field's top
// corners, then the next two). A pixel a character, on the least scene
// and a pixel over each way, which is dark.
func TestTetrominoFrameAndBoxes(t *testing.T) {
	g := tetroGame(1, 30, 23)
	sc := g.Draw(30, 23)
	if g.cols != 10 || g.rows != 10 {
		t.Fatalf("field %dx%d, want 10x10", g.cols, g.rows)
	}
	want := []string{
		"#############################.",
		"#....................#......#.",
		"#....................#......#.",
		"#....................#......#.",
		"#....................#......#.",
		"#....................########.",
		"#....................#......#.",
		"#....................#......#.",
		"#....................#......#.",
		"#....................#......#.",
		"#....................########.",
		"#....................#......#.",
		"#....................#......#.",
		"#....................#......#.",
		"#....................#......#.",
		"#....................########.",
		"#....................#......#.",
		"#....................#......#.",
		"#....................#......#.",
		"#....................#......#.",
		"#....................########.",
		"######################........",
		"..............................",
	}
	var got []string
	for y := 0; y < 23; y++ {
		var b strings.Builder
		for x := 0; x < 30; x++ {
			if sc.Pix[y*30+x] == inkTetroFrame {
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
	if g.Room() != (Room{W: 29, H: 22, Most: 1}) {
		t.Errorf("room %+v", g.Room())
	}
	// 120 x 36 cells, a row of them the status: twenty-eight blocks by
	// sixteen, a pixel to spare each way.
	if g := tetroGame(1, 65, 35); g.cols != 28 || g.rows != 16 {
		t.Errorf("field %dx%d, want 28x16", g.cols, g.rows)
	}
}

// The pieces to come are in the column of boxes, the next at the top,
// each as it will come in, a pixel a block, in its colour, in the middle
// of its box as near as it halves; as many as the boxes, six on 120 x 36
// cells. The next comes in next, the others move up a box, and a new one
// is at the foot.
func TestTetrominoQueue(t *testing.T) {
	for seed := uint64(1); seed <= 7; seed++ {
		g := tetroGame(seed, 65, 35)
		sc := g.Draw(65, 35)
		if len(g.queue) != 6 {
			t.Fatalf("seed %d: %d to come", seed, len(g.queue))
		}
		fw, _ := g.size()
		for i, piece := range g.queue {
			px, py := fw, tetroEdge+i*(tetroBoxH+tetroEdge)
			var lit [][2]int
			for y := 0; y < tetroBoxH; y++ {
				for x := 0; x < tetroBoxW; x++ {
					switch ink := sc.Pix[(py+y)*65+px+x]; ink {
					case 0:
					case uint8(1 + piece):
						lit = append(lit, [2]int{x, y})
					default:
						t.Fatalf("seed %d: box %d has ink %d", seed, i, ink)
					}
				}
			}
			if len(lit) != 4 {
				t.Fatalf("seed %d: box %d has %d pixels lit, want the 4 of a piece", seed, i, len(lit))
			}
			lx, hx, ly, hy := lit[0][0], lit[0][0], lit[0][1], lit[0][1]
			for _, p := range lit {
				lx, hx, ly, hy = min(lx, p[0]), max(hx, p[0]), min(ly, p[1]), max(hy, p[1])
			}
			if d := tetroBoxW - 1 - hx - lx; d < 0 || d > 1 {
				t.Errorf("seed %d: piece %d in box %d not in the middle across: %d…%d", seed, piece, i, lx, hx)
			}
			if d := tetroBoxH - 1 - hy - ly; d < 0 || d > 1 {
				t.Errorf("seed %d: piece %d in box %d not in the middle down: %d…%d", seed, piece, i, ly, hy)
			}
			blocks := tetroTurns[piece][0]
			bx, by := blocks[0][0], blocks[0][1]
			for _, b := range blocks {
				bx, by = min(bx, b[0]), min(by, b[1])
			}
			for _, b := range blocks {
				if p := [2]int{lx + b[0] - bx, ly + b[1] - by}; !slices.Contains(lit, p) {
					t.Errorf("seed %d: piece %d in box %d: pixel %v dark", seed, piece, i, p)
				}
			}
		}
		// Under the sixth box, nothing: the column ends there.
		for y := 6*(tetroBoxH+tetroEdge) + 1; y < 35; y++ {
			for x := fw; x < 65; x++ {
				if sc.Pix[y*65+x] != 0 {
					t.Fatalf("seed %d: pixel %d,%d under the column is %d", seed, x, y, sc.Pix[y*65+x])
				}
			}
		}
		before := slices.Clone(g.queue)
		g.come()
		if g.cur != before[0] || !slices.Equal(g.queue[:5], before[1:]) || len(g.queue) != 6 {
			t.Errorf("seed %d: came %d, queue %v; was %v", seed, g.cur, g.queue, before)
		}
	}
	// As many boxes as the height holds: four on 80 x 24, nine on 200 x 50;
	// five where the frame is thirty pixels high, the sixth's foot under
	// the floor.
	for _, c := range []struct{ w, h, n int }{{40, 23, 4}, {100, 49, 9}, {65, 30, 5}} {
		if g := tetroGame(1, c.w, c.h); len(g.queue) != c.n {
			t.Errorf("%dx%d: %d boxes, want %d", c.w, c.h, len(g.queue), c.n)
		}
	}
}

// A game played a while: a piece comes in behind the top of the frame
// in the middle, and goes a step at a time — a turn or a block across,
// and a block down or not, never up — sometimes across and down at once;
// it never overlaps the stack or a wall; rows fill and go. It plays
// well: the stack stays low, under half the field, with few holes in it
// (2026-10-07: it was six rows high at most and two holes, where without
// minding how high a piece lands it filled the field).
func TestTetrominoPlays(t *testing.T) {
	g := tetroGame(5, 65, 35)
	if !g.open(0, -1) || !g.open(g.cols-1, -1) || g.open(-1, -1) || g.open(g.cols, -1) {
		t.Error("over the field, a piece may be anywhere across it and nowhere else")
	}
	cleared, diagonal, pieces := 0, 0, 0
	for i := 0; i < 3000; i++ {
		before, cur, going := g.at, g.cur, len(g.going)
		g.Step()
		g.Draw(65, 35)
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
		switch {
		case going == 0 && len(g.going) > 0:
			cleared += len(g.going)
		case len(g.going) > 0 || going > 0:
		case g.cur != cur || g.at.y < before.y:
			// A piece came in: hidden, its lowest block a row over the
			// field, in the middle.
			pieces++
			blocks := tetroTurns[g.cur][g.at.turn]
			lo, hi, foot := g.cols, 0, -9
			for _, b := range blocks {
				lo, hi, foot = min(lo, g.at.x+b[0]), max(hi, g.at.x+b[0]), max(foot, g.at.y+b[1])
			}
			if foot != -1 || lo != g.cols-1-hi && lo != g.cols-2-hi {
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
	g := tetroGame(2, 65, 35)
	foot := g.rows - 1
	for x := 0; x < g.cols; x++ {
		if x < 10 || x > 13 {
			g.cells[foot*g.cols+x] = 3
		}
	}
	g.cells[(foot-1)*g.cols+2] = 5
	g.cur, g.at = 0, spot{0, 10, foot - 1} // the long piece, lying in the gap in the row
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
		sc := g.Draw(65, 35)
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

// Its best spot under an overhang, the piece gets there: down beside it,
// then across under it.
func TestTetrominoSlidesUnder(t *testing.T) {
	g := tetroGame(4, 65, 35)
	foot := g.rows - 1
	for x := 0; x < g.cols; x++ {
		switch {
		case x >= 8:
			g.cells[foot*g.cols+x], g.cells[(foot-1)*g.cols+x] = 2, 2
		case x < 4:
			g.cells[(foot-2)*g.cols+x] = 2
		}
	}
	g.cur = 0
	g.at = spot{0, 12, -2}
	g.goal = g.choose()
	g.route()
	// Lying in the four blocks of the bottom row under the overhang.
	for _, b := range tetroTurns[0][g.goal.turn] {
		if x, y := g.goal.x+b[0], g.goal.y+b[1]; x > 3 || y != foot {
			t.Fatalf("goal %+v: a block at %d,%d", g.goal, x, y)
		}
	}
	for i := 0; i < 100 && g.at != g.goal; i++ {
		g.Step()
	}
	if g.at != g.goal {
		t.Errorf("at %+v, not the goal %+v", g.at, g.goal)
	}
}

// A spot is rated with the rows it fills gone: a hole under one of them
// is a hole no more.
func TestTetrominoRatesWithTheFullRowsGone(t *testing.T) {
	g := tetroGame(4, 65, 35)
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
	g := tetroGame(4, 65, 35)
	for x := 0; x <= 14; x++ {
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

// A stack that reaches the top is the end (user, 2026-10-07): every
// piece there is dims into the frame's grey, all together, over five
// seconds — the scene the same, the colours going; then END in red in
// the middle of the screen, a block a pixel of the lock's letters, on a
// plate of the ground, for two seconds; then a new game in the colours
// again. It is at the clear's pace.
func TestTetrominoEnds(t *testing.T) {
	g := tetroGame(6, 65, 35)
	// Full but for a hole a row, none beside another: nowhere for a piece.
	for y := 0; y < g.rows; y++ {
		for x := 0; x < g.cols; x++ {
			if x != 5+3*(y%2) {
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
	fade, end := g.frames(tetroFade), g.frames(tetroEnd)
	if fade != 125 || end != 50 {
		t.Fatalf("fade %d frames, END %d", fade, end)
	}
	// pieces are the pieces' inks now; the rest stay as they are.
	pieces := func() []string {
		inks := g.Inks()
		if inks[0] != ownGround || inks[8] != tetroWhite || inks[9] != tetroGrey || inks[10] != tetroRed {
			t.Fatalf("inks %v", inks)
		}
		return inks[1:8]
	}
	first := g.Draw(65, 35)
	for i, c := range pieces() {
		if want := mix(tetroColours[i], tetroGrey, 1.0/125); c != want {
			t.Fatalf("the frame it ends, piece %d is %s, want %s", i, c, want)
		}
	}
	for i := 0; i < 62; i++ {
		g.Step()
	}
	// Half way, half way into the grey, all alike; the scene as it was.
	for i, c := range pieces() {
		if want := mix(tetroColours[i], tetroGrey, 63.0/125); c != want {
			t.Fatalf("half way, piece %d is %s, want %s", i, c, want)
		}
	}
	if sc := g.Draw(65, 35); !slices.Equal(sc.Pix, first.Pix) {
		t.Fatal("the scene changed while the pieces dimmed")
	}
	for g.fx < fade-1 {
		g.Step()
	}
	for i, c := range pieces() {
		if c != tetroGrey {
			t.Fatalf("dimmed, piece %d is %s", i, c)
		}
	}
	if slices.Contains(g.Draw(65, 35).Pix, inkTetroRed) {
		t.Fatal("END before the pieces have dimmed")
	}
	g.Step()
	// END, eleven pixels by five of the letters, twenty-two by ten in
	// the scene, in the middle of its sixty-five by thirty-five, on the
	// ground two pixels round; the rest as it was.
	sc := g.Draw(65, 35)
	art := tetroSpell("END")
	for y := 0; y < 35; y++ {
		for x := 0; x < 65; x++ {
			ax, ay := (x-21)/2, (y-12)/2
			lit := x >= 21 && y >= 12 && ax < len(art[0]) && ay < len(art) && art[ay][ax] == '#'
			plate := x >= 19 && x < 45 && y >= 10 && y < 24
			got := sc.Pix[y*65+x]
			switch {
			case lit && got != inkTetroRed, !lit && plate && got != 0, !plate && got != first.Pix[y*65+x]:
				t.Fatalf("END at pixel %d,%d: %d", x, y, got)
			}
		}
	}
	for i := 0; i < end-1; i++ {
		g.Step()
	}
	if !g.over || !slices.Contains(g.Draw(65, 35).Pix, inkTetroRed) {
		t.Fatal("END went before two seconds")
	}
	g.Step()
	if g.over || slices.ContainsFunc(g.cells, func(c uint8) bool { return c != 0 }) || g.at.y >= 0 {
		t.Fatalf("no new game: over %v, at %+v", g.over, g.at)
	}
	if !slices.Equal(pieces(), tetroColours) || slices.Contains(g.Draw(65, 35).Pix, inkTetroRed) {
		t.Error("the new game is not in its colours")
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
	a, b := tetroGame(9, 65, 35), tetroGame(9, 65, 35)
	for i := 0; i < 500; i++ {
		a.Step()
		b.Step()
		if !slices.Equal(a.Draw(65, 35).Pix, b.Draw(65, 35).Pix) {
			t.Fatalf("step %d: two games of one seed differ", i)
		}
	}
	a.Draw(80, 40)
	if a.cols != 35 || a.rows != 19 || slices.ContainsFunc(a.cells, func(c uint8) bool { return c != 0 }) {
		t.Errorf("after a resize: %dx%d, stack %v", a.cols, a.rows, slices.ContainsFunc(a.cells, func(c uint8) bool { return c != 0 }))
	}
	for _, s := range [][2]int{{28, 40}, {65, 21}} {
		g := tetroGame(1, s[0], s[1])
		g.Step()
		if sc := g.Draw(s[0], s[1]); slices.ContainsFunc(sc.Pix, func(c uint8) bool { return c != 0 }) {
			t.Errorf("%dx%d: something drawn", s[0], s[1])
		}
	}
	g := tetroGame(1, 29, 22)
	if sc := g.Draw(29, 22); sc.Pix[0] != inkTetroFrame {
		t.Errorf("29x22: no frame")
	}
	NewTetromino(1, SpeedNormal, tetroSpell).Step() // before the first draw: nothing, and no panic
}

// Its colours: the ground, the pieces' as they usually are, white for a
// row going, grey for the frame, and red for END.
func TestTetrominoInks(t *testing.T) {
	want := []string{"#313244", "#89dceb", "#f9e2af", "#cba6f7", "#a6e3a1", "#f38ba8", "#89b4fa", "#fab387", "#ffffff", "#7f849c", "#f38ba8"}
	if got := NewTetromino(1, "", tetroSpell).Inks(); !slices.Equal(got, want) {
		t.Errorf("inks %v", got)
	}
}
