package ui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/saver"
)

var at = time.Date(2026, time.September, 24, 21, 5, 9, 0, time.UTC)

// one is a layout of a single block; under stacks blocks; beside sets
// them side by side.
func one(lines []string, k int) layout { return layout{blocks: []placed{{lines: lines, k: k}}} }
func under(blocks ...placed) layout    { return layout{blocks: blocks} }
func beside(blocks ...placed) layout   { return layout{blocks: blocks, beside: true} }

// The layouts function.md §5.3 expects, with rows = terminal height − 1
// for the status row: each block keeps its size while any of its content
// fits at it, the time laid out first, the date in what is left. Widths
// are cells: a digit 3k, the gap unit 1 at 1 and 2, 2 at 3.
func TestFitLaysTheBlocksOut(t *testing.T) {
	hm := saver.Clock{Time: saver.TimeHM, Date: saver.DateOff, Layout: saver.LayoutRow}
	hms := saver.Clock{Time: saver.TimeHMS, Date: saver.DateOff, Layout: saver.LayoutRow}
	full := saver.Clock{Time: saver.TimeHMS, Date: saver.DateYMD, Layout: saver.LayoutRow}
	col := saver.Clock{Time: saver.TimeHMS, Date: saver.DateOff, Layout: saver.LayoutColumn}
	colFull := saver.Clock{Time: saver.TimeHMS, Date: saver.DateYMD, Layout: saver.LayoutColumn}
	timeAt := func(k int) placed { return placed{[]string{"21 05 09"}, k} }
	cases := []struct {
		name       string
		s          saver.Clock
		cols, rows int
		size       int
		want       layout
		plain      []string
	}{
		// HH MM at 3 is 46 cells: 80 columns hold 38, so large steps down to 2 (29).
		{"80x24 HH MM large steps down to 2", hm, 80, 23, 3, one([]string{"21 05"}, 2), nil},
		{"120x40 HH MM large", hm, 120, 39, 3, one([]string{"21 05"}, 3), nil},
		{"200x60 HH MM large", hm, 200, 59, 3, one([]string{"21 05"}, 3), nil},
		{"200x60 HH MM medium stays 2", hm, 200, 59, 2, one([]string{"21 05"}, 2), nil},
		{"200x60 HH MM small stays 1", hm, 200, 59, 1, one([]string{"21 05"}, 1), nil},
		// The size goes before the units: HH MM SS at 1 (27) outranks HH MM at 2 (29 — too wide anyway).
		{"80x24 HH MM SS medium: small, with the seconds", hms, 80, 23, 2, one([]string{"21 05 09"}, 1), nil},
		// The date is laid out on its own under the time. On 80 columns
		// (38 cells) the time is small; the date's year (39) does not fit at
		// any size, so the short date follows, small.
		{"80x24 full clock small", full, 80, 23, 1, under(timeAt(1), placed{[]string{"09-24"}, 1}), nil},
		{"80x24 full clock medium: both small, the year gone", full, 80, 23, 2, under(timeAt(1), placed{[]string{"09-24"}, 1}), nil},
		{"200x60 full clock medium: all of it", full, 200, 59, 2, under(timeAt(2), placed{[]string{"2026-09-24"}, 2}), nil},
		// 120 columns hold 58 cells: the time at 2 is 45, the full date at
		// 2 is 69 — so the date drops to 1 and keeps its year, while the
		// time stays at 2 (user, 2026-09-24).
		{"120x39 full clock medium: time at 2, date at 1 with its year", full, 120, 38, 2, under(timeAt(2), placed{[]string{"2026-09-24"}, 1}), nil},
		// The user's terminal, 152 columns: 74 cells hold both at 2.
		{"152x39 full clock medium: all of it", full, 152, 38, 2, under(timeAt(2), placed{[]string{"2026-09-24"}, 2}), nil},
		// And large: HH MM SS at 3 is 72 cells — it fits, with 21 rows of
		// 29; the 4-row gap leaves 4, too few for a date at 1 (7).
		{"152x32 full clock large: the time at 3, the date left out", full, 152, 31, 3, one([]string{"21 05 09"}, 3), nil},
		// Too short for a second block: the date is left out, the time is whole.
		{"130x24 full clock medium: the date is left out", full, 130, 23, 2, one([]string{"21 05 09"}, 2), nil},
		// 40 columns hold 18 cells: HH MM at 1 is 17.
		{"40x12 HH MM is 1", hm, 40, 11, 2, one([]string{"21 05"}, 1), nil},
		{"40x12 full clock: HH MM at 1, nothing else fits", full, 40, 11, 3, one([]string{"21 05"}, 1), nil},
		{"30x8 HH MM is plain text", hm, 30, 7, 2, layout{}, []string{"21 05"}},
		{"30x8 full clock falls back to plain HH MM", full, 30, 7, 3, layout{}, []string{"21 05"}},
		// A column is two digits wide: 13 cells at 2; three lines are 44
		// rows at 2, 23 at 1. 39 rows hold them at 1, and the size goes
		// before the seconds.
		{"120x40 column HH MM SS medium: three lines at 1", col, 120, 39, 2, beside(placed{[]string{"21", "05", "09"}, 1}), nil},
		{"120x80 column HH MM SS large", col, 120, 79, 3, beside(placed{[]string{"21", "05", "09"}, 3}), nil},
		// With a date the column layout is two columns, the date on the
		// left, the time on the right, each at its own size: 160 columns
		// hold 78 cells and 43 rows 40 — the time's 44 rows fit only at 1,
		// and so do the date's.
		{"160x43 column full clock medium: date left, time right, both at 1", colFull, 160, 42, 2,
			beside(placed{[]string{"2026", "09", "24"}, 1}, placed{[]string{"21", "05", "09"}, 1}), nil},
		// 160×72 holds both at 3: the time is 20 cells wide, the date 42;
		// 32 and 74 with the 12-cell gutter, of 78.
		{"160x72 column full clock large: both at 3", colFull, 160, 71, 3,
			beside(placed{[]string{"2026", "09", "24"}, 3}, placed{[]string{"21", "05", "09"}, 3}), nil},
		// 60 columns hold 28 cells: the time at 3 (20) and the gutter leave
		// nothing, and the date at 1 is 15 — left out.
		{"60x72 column full clock large: the date is left out", colFull, 60, 71, 3,
			beside(placed{[]string{"21", "05", "09"}, 3}), nil},
	}
	for _, c := range cases {
		got, plain := fit(faceTall, c.s, at, c.cols, c.rows, c.size)
		if !reflect.DeepEqual(got, c.want) || !reflect.DeepEqual(plain, c.plain) {
			t.Errorf("%s: got %v plain %q, want %v plain %q", c.name, got, plain, c.want, c.plain)
		}
	}
	// The short face on the user's terminal at large: the time is 15 rows,
	// and 10 are left under it — the date at 2 (69 cells, 10 rows).
	if l, _ := fit(faceShort, full, at, 152, 31, 3); !reflect.DeepEqual(l, under(timeAt(3), placed{[]string{"2026-09-24"}, 2})) {
		t.Errorf("152x32 short face large: %v", l)
	}
}

// The widths the sizes are reasoned from (function.md §5.3), in cells at
// 1: digits and most letters three, M and W five, the hyphen three, the
// space one gap unit, a gap of one between.
func TestLineWidths(t *testing.T) {
	for line, want := range map[string]int{
		"21 05":      17, // 3+1+3 +1+1+1+ 3+1+3
		"21 05 09":   27,
		"2026-09-24": 39, // 8 digits, 2 hyphens, 9 gaps
		"09-24":      19,
		"SEP-24":     23,
		"MAR-24":     25,
		"21":         7,
		"2026":       15,
	} {
		if got := lineW(faceTall, line, 1); got != want {
			t.Errorf("%q: %d cells, want %d", line, got, want)
		}
	}
	// The gap unit is 1 at 2 and 2 at 3: HH MM SS is 45 cells at 2, 72
	// at 3 — 144 columns, which the user's 152 hold (2026-09-24).
	if w := lineW(faceTall, "21 05 09", 2); w != 45 {
		t.Errorf("HH MM SS at 2 is %d cells", w)
	}
	if w := lineW(faceTall, "21 05 09", 3); w != 72 {
		t.Errorf("HH MM SS at 3 is %d cells", w)
	}
	if w := lineW(faceTall, "21 05", 3); w != 46 {
		t.Errorf("HH MM at 3 is %d cells", w)
	}
	// A column of HH / MM at 3 is 44 rows; with SS 67.
	if _, h := blockSize(faceTall, []string{"21", "05"}, 3); h != 44 {
		t.Errorf("HH/MM at 3 is %d rows", h)
	}
	if _, h := blockSize(faceTall, []string{"21", "05", "09"}, 3); h != 67 {
		t.Errorf("HH/MM/SS at 3 is %d rows", h)
	}
	// The short face: the same widths, five rows a line — a column of
	// HH / MM / SS at 3 is 49 rows.
	if w, h := blockSize(faceShort, []string{"21 05 09"}, 1); w != 27 || h != 5 {
		t.Errorf("short HH MM SS is %d×%d", w, h)
	}
	if _, h := blockSize(faceShort, []string{"21", "05", "09"}, 3); h != 49 {
		t.Errorf("short HH/MM/SS at 3 is %d rows", h)
	}
	// A column of HH / MM / SS at large needs 52 rows in the short face
	// and 70 in the tall one; on 43 rows the short face holds it at 2, the
	// tall one at 1 — the size steps down, the seconds stay.
	col := saver.Clock{Time: saver.TimeHMS, Layout: saver.LayoutColumn}
	if l, _ := fit(faceShort, col, at, 160, 51, 3); !reflect.DeepEqual(l, beside(placed{[]string{"21", "05", "09"}, 3})) {
		t.Errorf("short column at 160x52: %v", l)
	}
	if l, _ := fit(faceShort, col, at, 160, 42, 3); !reflect.DeepEqual(l, beside(placed{[]string{"21", "05", "09"}, 2})) {
		t.Errorf("short column at 160x43: %v", l)
	}
	if l, _ := fit(faceTall, col, at, 160, 42, 3); !reflect.DeepEqual(l, beside(placed{[]string{"21", "05", "09"}, 1})) {
		t.Errorf("tall column at 160x43: %v", l)
	}
}

// box is the lit pixels' bounding box.
func box(b board) (minX, minY, maxX, maxY int) {
	minX, minY, maxX, maxY = b.w, b.h, -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.at(x, y) {
				minX, maxX = min(minX, x), max(maxX, x)
				minY, maxY = min(minY, y), max(maxY, y)
			}
		}
	}
	return
}

// clear blanks a rectangle of the board, to look at what is left.
func clear(b board, x1, y1, x2, y2 int) board {
	c := b.clone()
	for y := y1; y < y2; y++ {
		for x := x1; x < x2; x++ {
			c.lit[y*c.w+x] = false
		}
	}
	return c
}

func TestPaintCentresTheBlocks(t *testing.T) {
	// 120×40 at 2: "21 05" is 29 × 14 cells on a 60 × 39 board.
	b := paint(faceTall, one([]string{"21 05"}, 2), 120, 39)
	if b.w != 60 || b.h != 39 {
		t.Fatalf("board %d×%d", b.w, b.h)
	}
	if minX, minY, maxX, maxY := box(b); minX != 15 || maxX != 43 || minY != 12 || maxY != 25 {
		t.Errorf("lit box x %d..%d y %d..%d; want 15..43, 12..25", minX, maxX, minY, maxY)
	}
	// Scaling multiplies the lit count by k².
	if oneK := paint(faceTall, one([]string{"21 05"}, 1), 120, 39).count(); b.count() != oneK*4 {
		t.Errorf("k=2 lights %d, k=1 lights %d", b.count(), oneK)
	}
	// Two blocks under each other at different scales: the time at 2 (14
	// rows), a gap of 2, the date at 1 (7 rows) — 23 rows, centred from
	// row 8; the date (19 cells) centred on its own, from column 20.
	two := paint(faceTall, under(placed{[]string{"21 05"}, 2}, placed{[]string{"09-24"}, 1}), 120, 39)
	if minX, minY, maxX, maxY := box(two); minX != 15 || maxX != 43 || minY != 8 || maxY != 30 {
		t.Errorf("two blocks: lit box x %d..%d y %d..%d; want 15..43, 8..30", minX, maxX, minY, maxY)
	}
	if minX, minY, maxX, maxY := box(clear(two, 0, 0, 60, 24)); minX != 20 || maxX != 38 || minY != 24 || maxY != 30 {
		t.Errorf("the date block: x %d..%d y %d..%d; want 20..38, 24..30", minX, maxX, minY, maxY)
	}
	// Two blocks beside each other: the date (15 cells) at 1 on the left,
	// a gutter of 6, the time (7) at 1 on the right — 28 cells, centred
	// from column 16, the time from 37; each 23 rows, centred from row 8.
	side := paint(faceTall, beside(placed{[]string{"2026", "09", "24"}, 1}, placed{[]string{"21", "05", "09"}, 1}), 120, 39)
	if minX, minY, maxX, maxY := box(side); minX != 16 || maxX != 43 || minY != 8 || maxY != 30 {
		t.Errorf("beside: lit box x %d..%d y %d..%d; want 16..43, 8..30", minX, maxX, minY, maxY)
	}
	if minX, _, maxX, _ := box(clear(side, 0, 0, 34, 39)); minX != 37 || maxX != 43 {
		t.Errorf("the time column: x %d..%d; want 37..43", minX, maxX)
	}
	// Beside, at different scales, each column is centred up and down on
	// its own: the time at 2 is 44 rows on 59, from row 7; the date at 1
	// is 23, from row 18.
	mixed := paint(faceTall, beside(placed{[]string{"2026", "09", "24"}, 1}, placed{[]string{"21", "05", "09"}, 2}), 120, 59)
	if _, minY, _, maxY := box(clear(mixed, 0, 0, 30, 59)); minY != 7 || maxY != 50 {
		t.Errorf("the time column at 2: y %d..%d; want 7..50", minY, maxY)
	}
	if _, minY, _, maxY := box(clear(mixed, 30, 0, 60, 59)); minY != 18 || maxY != 40 {
		t.Errorf("the date column at 1: y %d..%d; want 18..40", minY, maxY)
	}
	// The gap unit at 3: two digits 9 cells wide with 2 dark between —
	// "21" at 3 spans 20 cells, and column 9 is dark all the way down.
	big := paint(faceTall, one([]string{"21"}, 3), 120, 39)
	minX, minY, maxX, _ := box(big)
	if maxX-minX != 19 {
		t.Errorf("21 at 3 spans %d cells, want 20", maxX-minX+1)
	}
	for y := minY; y < minY+21; y++ {
		if big.at(minX+9, y) || big.at(minX+10, y) {
			t.Fatalf("row %d: the gap between 2 and 1 is lit", y)
		}
	}
}

// A game scene is the whole board at the largest scale that leaves it
// room to be played, and nothing is lettered over it.
func TestSceneFitsAndPaints(t *testing.T) {
	for _, c := range []struct{ size, cols, rows, k, w, h int }{
		{3, 152, 31, 1, 76, 31}, // the user's terminal: 31 rows hold one scale
		{2, 200, 59, 2, 50, 29},
		{3, 200, 59, 2, 50, 29},  // 59/3 = 19 rows is too few
		{2, 200, 54, 1, 100, 54}, // 54/2 = 27 rows would clip the jump: 28 are needed
		{3, 300, 90, 3, 50, 30},
		{1, 300, 90, 1, 150, 90},
		{2, 40, 12, 1, 20, 12}, // too small at any scale: 1, and the game clips
	} {
		if k, w, h := fitScene(c.size, c.cols, c.rows); k != c.k || w != c.w || h != c.h {
			t.Errorf("size %d on %dx%d: k %d scene %dx%d, want k %d %dx%d", c.size, c.cols, c.rows, k, w, h, c.k, c.w, c.h)
		}
	}
	d := saver.NewDino(3, saver.RunnerBig, saver.SceneGrass)
	sc := d.Draw(50, 29)
	b := paintScene(sc, 2, 200, 59)
	if b.w != 100 || b.h != 59 {
		t.Fatalf("board %dx%d", b.w, b.h)
	}
	// Every scene pixel is 2 × 2 cells: the ground line, at scene row 27,
	// is board rows 54 and 55 across the scene's 100 cells.
	oy := (59 - 58) / 2
	for x := 0; x < 100; x++ {
		if !b.at(x, oy+54) || !b.at(x, oy+55) {
			t.Fatalf("ground missing at %d", x)
		}
	}
	// The board is the scene and nothing more: exactly the scene's lit
	// pixels, each four cells.
	lit := 0
	for _, p := range sc.Pix {
		if p {
			lit++
		}
	}
	if b.count() != lit*4 {
		t.Errorf("%d cells lit for %d scene pixels: something lettered over the scene", b.count(), lit)
	}
}

func TestRowsAreExactlyTheTerminalWide(t *testing.T) {
	bg, fg := lipgloss.Color("#313244"), lipgloss.Color("#f2b753")
	for _, cols := range []int{80, 81, 40, 7} {
		rows := 23
		l, plain := fit(faceTall, saver.Clock{Time: saver.TimeHM, Date: saver.DateYMD}, at, cols, rows, 2)
		var out []string
		if len(l.blocks) > 0 {
			out = boardRows(paint(faceTall, l, cols, rows), bg, fg, cols, false)
		} else {
			out = plainRows(plain, bg, fg, cols, rows, true)
		}
		if len(out) != rows {
			t.Fatalf("cols %d: %d rows, want %d", cols, len(out), rows)
		}
		for i, r := range out {
			if w := lipgloss.Width(r); w != cols {
				t.Errorf("cols %d row %d is %d wide", cols, i, w)
			}
		}
		if s := statusRow(cols, true, "vulcan", "prod-db-01", at, true, "", ""); lipgloss.Width(s) != cols {
			t.Errorf("cols %d status is %d wide", cols, lipgloss.Width(s))
		}
	}
}

func TestPlainTextCarriesTheLines(t *testing.T) {
	out := plainRows([]string{"21 05", "09-24"}, lipgloss.Color("#000000"), lipgloss.Color("#ffffff"), 40, 11, false)
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "21 05") || !strings.Contains(joined, "09-24") {
		t.Errorf("text missing:\n%s", joined)
	}
}

func TestStatusRowSaysWhatMatters(t *testing.T) {
	s := statusRow(80, true, "vulcan", "host", at, false, "", "")
	if !strings.Contains(s, "vulcan@host · locked since 21:05") {
		t.Errorf("%q", s)
	}
	s = statusRow(80, false, "vulcan", "host", at, true, "", "")
	if strings.Contains(s, "vulcan") || !strings.Contains(s, "no PIN · any key unlocks") {
		t.Errorf("status off must still say there is no PIN: %q", s)
	}
	s = statusRow(80, true, "v", "h", at, true, "config.yaml: yaml: line 1", "")
	if !strings.Contains(s, "config error: config.yaml") || !strings.Contains(s, "no PIN") {
		t.Errorf("%q", s)
	}
}
