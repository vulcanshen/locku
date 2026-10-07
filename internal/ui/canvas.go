package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/saver"
)

// The canvas is one LED board (function.md §5.3, ui.md §1.2): every cell of
// the terminal but the status row is one pixel — the square glyph and a
// space, two columns wide, so a pixel is close to square. Dark pixels wear
// the board's bg colour, lit pixels its fg. The saver's lines are set in
// the pixel font at the saver's size, and centred.
//
// It is the only way anything is drawn. A saver never sees a glyph or a
// position, nor a colour — but a game with colours of its own, which
// names them (saver.Inked; user, 2026-10-06), and has no bg / fg.
//
// Two units (user, 2026-09-24): the DISPLAY unit is one font pixel, k × k
// cells at size k; the GAP unit is the dark between things — two glyphs,
// two lines, and the space that parts two groups is one of them — and it
// grows with the display unit but not as fast, or a large clock is mostly
// gap. Everything below is measured in cells.

const (
	marginCols = 4 // two columns of margin either side
	marginRows = 2 // one row above and below
	// The room between two blocks, in gap units: under each other, two;
	// beside each other, six — two columns need a gutter wider than the
	// space between two groups, or the date reads as part of the time.
	underGap  = 2
	besideGap = 6
)

// gap is the gap unit at scale k, in cells: one at small and medium, two
// at large.
func gap(k int) int { return (k + 1) / 2 }

// pixelCell is one pixel on the terminal, two cells: the glyph and its
// space, or the glyph alone where it takes two (tdp D6, 2026-09-29).
func pixelCell() string { return pixelGlyph + spaces(2-iconCells) }

// board is the pixel grid: w pixels across (half the columns), h down (the
// rows above the status row). Each pixel is an ink: 0 is dark, the
// board's ground; the others are lit, each in its own colour, which the
// one drawing the board gives (boardRows). A clock lights in one ink; a
// custom saver's ending board lights the code after EXIT in a second,
// the accent (2026-09-25); a saver with colours of its own lights in as
// many as it has (2026-10-06).
type board struct {
	w, h int
	ink  []uint8
}

// The inks of a board the clock draws: dark, the fg, and the accent.
const (
	inkOff uint8 = iota
	inkOn
	inkAccent
)

func newBoard(w, h int) board {
	return board{w: max(0, w), h: max(0, h), ink: make([]uint8, max(0, w)*max(0, h))}
}

// at reports whether the pixel at x, y is lit, in any ink.
func (b board) at(x, y int) bool { return b.ink[y*b.w+x] != inkOff }

// put inks the pixel at x, y, if it is on the board.
func (b *board) put(x, y int, ink uint8) {
	if x >= 0 && x < b.w && y >= 0 && y < b.h {
		b.ink[y*b.w+x] = ink
	}
}

func (b *board) set(x, y int) { b.put(x, y, inkOn) }

func (b board) same(o board) bool { return b.w == o.w && b.h == o.h }

func (b board) clone() board {
	c := board{w: b.w, h: b.h, ink: make([]uint8, len(b.ink))}
	copy(c.ink, b.ink)
	return c
}

// toned is a clone of b wearing o's inks where both are lit: a reveal
// from b to o lights and darkens the pixels one by one, but the colours
// are the destination's from the first frame.
func (b board) toned(o board) board {
	c := b.clone()
	if b.same(o) {
		for i, k := range o.ink {
			if k != inkOff && c.ink[i] != inkOff {
				c.ink[i] = k
			}
		}
	}
	return c
}

// count is how many pixels are lit.
func (b board) count() int {
	n := 0
	for _, k := range b.ink {
		if k != inkOff {
			n++
		}
	}
	return n
}

// glyphCells is a glyph's width in cells at scale k: its pixels at k —
// but the space, which is one gap unit, so that with the gap either side
// of it a space parts two groups by three gap units.
func glyphCells(f face, r rune, k int) int {
	if r == ' ' {
		return gap(k)
	}
	return f.glyphW(r) * k
}

// lineW is a line's width in cells at scale k: its glyphs and a gap
// between each two.
func lineW(f face, line string, k int) int {
	w := 0
	for i, r := range line {
		if i > 0 {
			w += gap(k)
		}
		w += glyphCells(f, r, k)
	}
	return w
}

// blockSize is the cells lines take at scale k: the widest line, and m
// lines at the face's height each with a gap between (function.md §5.3).
func blockSize(f face, lines []string, k int) (w, h int) {
	m := len(lines)
	for _, l := range lines {
		w = max(w, lineW(f, l, k))
	}
	if w == 0 || m == 0 {
		return 0, 0
	}
	return w, m*f.h*k + (m-1)*gap(k)
}

// fitsIn reports whether lines at scale k fit availW cells across and
// availRows down. A cell is two columns by one row, near enough square,
// so k × k draws the font as designed.
func fitsIn(f face, lines []string, k, availW, availRows int) bool {
	w, h := blockSize(f, lines, k)
	if w == 0 || k < 1 {
		return false
	}
	return w <= availW && h <= availRows
}

// placed is one block as it will be drawn: its lines, at its scale.
type placed struct {
	lines []string
	k     int
}

// layout is what the board draws: the blocks in the order they sit, one
// under the other, or — beside — left to right.
type layout struct {
	// accentFrom is the rune of the first block's first line from which
	// the pixels are the accent's: the code after EXIT; 0 accents nothing.
	accentFrom int
	blocks     []placed
	beside     bool
}

// blockGap is the room between two blocks, in cells along the axis they
// follow: gap units of the time's size — the block fitted first, which is
// the first drawn under each other and the last drawn beside.
func (l layout) blockGap() int {
	if len(l.blocks) == 0 {
		return 0
	}
	if l.beside {
		return besideGap * gap(l.blocks[len(l.blocks)-1].k)
	}
	return underGap * gap(l.blocks[0].k)
}

// fit lays the saver's blocks out, each on its own (function.md §5.3):
// the time first, in the whole canvas; the date in what is left — under
// the time in the row layout, beside it in the column layout, where the
// date takes the left and the time the right. For each block the size
// the saver asks for — small, medium, large: 1, 2, 3 — is tried first and
// stepped down before any of the block's content goes: the whole variant
// at every size, then the next variant at every size (user, 2026-09-24:
// a date too wide for its size shrinks, and the time keeps its own). A
// later block that fits at nothing is left out. The first block fitting
// at nothing is the fallback: its least lines come back as plain, to be
// drawn as text over the board. rows is the canvas: the terminal's height
// less the status row.
func fit(f face, s saver.Saver, now time.Time, cols, rows, size int) (layout, []string) {
	l := layout{beside: s.Beside()}
	wLeft := (cols - marginCols) / 2
	rowsLeft := rows - marginRows
	for i, b := range s.Blocks(now) {
		var got *placed
		for _, v := range b.Variants {
			for k := max(1, size); k >= 1 && got == nil; k-- {
				if fitsIn(f, v, k, wLeft, rowsLeft) {
					got = &placed{lines: v, k: k}
				}
			}
			if got != nil {
				break
			}
		}
		if got == nil {
			if i == 0 {
				return layout{}, b.Variants[len(b.Variants)-1]
			}
			break
		}
		bw, bh := blockSize(f, got.lines, got.k)
		if l.beside {
			// The time is fitted first and drawn last: the date goes to
			// its left.
			l.blocks = append([]placed{*got}, l.blocks...)
			wLeft -= bw + l.blockGap()
		} else {
			l.blocks = append(l.blocks, *got)
			rowsLeft -= bh + l.blockGap()
		}
	}
	return l, nil
}

// paint lights a layout on a fresh board for a cols × rows canvas. Blocks
// under each other: the stack centred, each block centred across. Blocks
// beside each other: the row centred, each block centred up and down on
// its own. blockGap between two. Within a block each line is centred.
func paint(f face, l layout, cols, rows int) board {
	b := newBoard(cols/2, rows)
	// The stack's extent along the axis the blocks follow.
	total := 0
	for i, p := range l.blocks {
		bw, bh := blockSize(f, p.lines, p.k)
		if l.beside {
			total += bw
		} else {
			total += bh
		}
		if i < len(l.blocks)-1 {
			total += l.blockGap()
		}
	}
	x0, y0 := 0, (b.h-total)/2
	if l.beside {
		x0, y0 = (b.w-total)/2, 0
	}
	for bi, p := range l.blocks {
		bw, bh := blockSize(f, p.lines, p.k)
		ox, oy := (b.w-bw)/2, y0
		if l.beside {
			ox, oy = x0, (b.h-bh)/2
		}
		for i, line := range p.lines {
			if line == "" {
				continue
			}
			from := 0
			if bi == 0 && i == 0 {
				from = l.accentFrom
			}
			stampLine(&b, f, line, p.k, ox+(bw-lineW(f, line, p.k))/2, oy+i*(f.h*p.k+gap(p.k)), from)
		}
		if l.beside {
			x0 += bw + l.blockGap()
		} else {
			y0 += bh + l.blockGap()
		}
	}
	return b
}

// stampLine lights line on b at scale k, its top-left cell at x, y; the
// runes from accentFrom on, when it is above 0, are in the accent's ink.
func stampLine(b *board, f face, line string, k, x, y, accentFrom int) {
	g := gap(k)
	n := 0
	for _, r := range line {
		if gl, ok := f.g[r]; ok && r != ' ' {
			ink := inkOn
			if accentFrom > 0 && n >= accentFrom {
				ink = inkAccent
			}
			for fy := 0; fy < f.h; fy++ {
				for fx := 0; fx < len(gl[fy]); fx++ {
					if gl[fy][fx] != '#' {
						continue
					}
					for dy := 0; dy < k; dy++ {
						for dx := 0; dx < k; dx++ {
							b.put(x+fx*k+dx, y+fy*k+dy, ink)
						}
					}
				}
			}
		}
		x += glyphCells(f, r, k) + g
		n++
	}
}

// spell is line in face f at scale 1, as rows of '#' lit and '.' dark,
// spaced as the board spaces it: what a game lettering something of its
// own — the bouncing box's time — draws.
func spell(f face, line string) []string {
	b := newBoard(lineW(f, line, 1), f.h)
	stampLine(&b, f, line, 1, 0, 0, 0)
	rows := make([]string, b.h)
	for y := range rows {
		var sb strings.Builder
		for x := 0; x < b.w; x++ {
			if b.at(x, y) {
				sb.WriteByte('#')
			} else {
				sb.WriteByte('.')
			}
		}
		rows[y] = sb.String()
	}
	return rows
}

// fitScene picks a game's scale on a cols × rows canvas: from the room's
// most, stepped down until the scene has its room — or 1, and the game
// clips what it must. The scene is the whole board, which is why the
// dino's ground runs edge to edge.
func fitScene(r saver.Room, cols, rows int) (k, w, h int) {
	cells := cols / 2
	for k = max(1, r.Most); k > 1; k-- {
		if cells/k >= r.W && rows/k >= r.H {
			break
		}
	}
	return k, cells / k, rows / k
}

// paintScene lights a game's frame at scale k — every scene pixel k × k
// cells in its ink, the odd cells left over at the edges dark. Nothing is
// lettered over it: no score, no clock (user, 2026-09-24).
func paintScene(sc saver.Scene, k, cols, rows int) board {
	b := newBoard(cols/2, rows)
	ox, oy := (b.w-sc.W*k)/2, (b.h-sc.H*k)/2
	for y := 0; y < sc.H; y++ {
		for x := 0; x < sc.W; x++ {
			ink := sc.Pix[y*sc.W+x]
			if ink == inkOff {
				continue
			}
			for dy := 0; dy < k; dy++ {
				for dx := 0; dx < k; dx++ {
					b.put(ox+x*k+dx, oy+y*k+dy, ink)
				}
			}
		}
	}
	return b
}

// boardRows draws the board, one string per row, each exactly cols wide:
// runs of pixels in one colour are rendered together, so a row costs a few
// escape sequences rather than one per cell. An odd terminal leaves its
// rightmost column blank (function.md §5.3). Under the PIN prompt the
// lock fades whole, as drawn (LockModel.View). inks are the colours, one
// an ink, the ground first; an ink past the end wears the last.
func boardRows(b board, inks []lipgloss.Color, shade *saver.Shading, cols int) []string {
	// A square's colour: its ink's — or, on a shaded board (user,
	// 2026-10-07: the runner's sky), its look's, the ground's for the
	// ground and for an ink the look has not.
	colour := func(x, y int) lipgloss.Color {
		ink := b.ink[y*b.w+x]
		if shade == nil {
			return inks[min(int(ink), len(inks)-1)]
		}
		l := shade.Looks[shade.Look(x, y)]
		if ink == inkOff || int(ink) >= len(l.Inks) || l.Inks[ink] == "" {
			return lipgloss.Color(l.Ground[y])
		}
		return lipgloss.Color(l.Inks[ink])
	}
	// A run is the squares of one ink, in one look.
	key := func(x, y int) int {
		k := int(b.ink[y*b.w+x])
		if shade != nil {
			k += 256 * shade.Look(x, y)
		}
		return k
	}
	tail := spaces(cols - b.w*2)
	rows := make([]string, b.h)
	for y := 0; y < b.h; y++ {
		var sb strings.Builder
		for x := 0; x < b.w; {
			k := key(x, y)
			run := x
			for run < b.w && key(run, y) == k {
				run++
			}
			cells := strings.Repeat(pixelCell(), run-x)
			sb.WriteString(lipgloss.NewStyle().Foreground(colour(x, y)).Render(cells))
			x = run
		}
		sb.WriteString(tail)
		rows[y] = sb.String()
	}
	return rows
}

// plainRows is the fallback when no scale fits: the board is laid all dark
// and the lines are set over it as ordinary text in the fg colour, centred.
func plainRows(lines []string, bg, fg lipgloss.Color, cols, rows int) []string {
	off := lipgloss.NewStyle().Foreground(bg)
	txt := lipgloss.NewStyle().Foreground(fg).Bold(true)
	w := cols / 2
	blank := off.Render(strings.Repeat(pixelCell(), w)) + spaces(cols-w*2)
	out := make([]string, max(0, rows))
	for i := range out {
		out[i] = blank
	}
	top := (rows - len(lines)) / 2
	for i, l := range lines {
		y := top + i
		if y < 0 || y >= rows {
			continue
		}
		l = truncate(l, cols)
		tw := dispW(l)
		leftCols := (cols - tw) / 2
		leftPx := leftCols / 2
		rightCols := cols - leftCols - tw
		rightPx := rightCols / 2
		out[y] = off.Render(strings.Repeat(pixelCell(), leftPx)) + spaces(leftCols-leftPx*2) +
			txt.Render(l) +
			spaces(rightCols-rightPx*2) + off.Render(strings.Repeat(pixelCell(), rightPx))
	}
	return out
}

// statusRow is the last row (function.md §5.4, ui.md §1.2): who is locked
// out of what, since when — and, whatever show says, whether there is a
// PIN at all, and whether the file could be read. Exactly cols wide.
func statusRow(cols int, show bool, user, host string, lockedAt time.Time, noPIN bool, problem, note string) string {
	var b strings.Builder
	plainW := 0
	add := func(s string, c lipgloss.Color) {
		if plainW > 0 {
			s = " · " + s
		}
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render(s))
		plainW += dispW(s)
	}
	if show {
		add(user+"@"+host+" · locked since "+lockedAt.Format("15:04"), dimColor)
	}
	if problem != "" {
		add("config error: "+problem, warnColor)
	}
	if noPIN {
		add("no PIN · any key unlocks", yellowColor)
	}
	// A custom saver's ending: why the board says what it says.
	if note != "" {
		add(note, warnColor)
	}
	return clipANSI(" "+b.String(), cols) + spaces(cols-1-plainW)
}
