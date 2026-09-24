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
// It is the only way anything is drawn. A saver never sees a colour, a
// glyph or a position.
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

// pixelCell is one pixel on the terminal: the glyph and its space.
var pixelCell = pixelGlyph + " "

// board is the pixel grid: w pixels across (half the columns), h down (the
// rows above the status row).
type board struct {
	w, h int
	lit  []bool
}

func newBoard(w, h int) board {
	return board{w: max(0, w), h: max(0, h), lit: make([]bool, max(0, w)*max(0, h))}
}

func (b board) at(x, y int) bool { return b.lit[y*b.w+x] }

func (b *board) set(x, y int) {
	if x >= 0 && x < b.w && y >= 0 && y < b.h {
		b.lit[y*b.w+x] = true
	}
}

func (b board) same(o board) bool { return b.w == o.w && b.h == o.h }

func (b board) clone() board {
	c := board{w: b.w, h: b.h, lit: make([]bool, len(b.lit))}
	copy(c.lit, b.lit)
	return c
}

// count is how many pixels are lit.
func (b board) count() int {
	n := 0
	for _, l := range b.lit {
		if l {
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
	blocks []placed
	beside bool
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
	for _, p := range l.blocks {
		bw, bh := blockSize(f, p.lines, p.k)
		ox, oy := (b.w-bw)/2, y0
		if l.beside {
			ox, oy = x0, (b.h-bh)/2
		}
		for i, line := range p.lines {
			if line == "" {
				continue
			}
			stampLine(&b, f, line, p.k, ox+(bw-lineW(f, line, p.k))/2, oy+i*(f.h*p.k+gap(p.k)))
		}
		if l.beside {
			x0 += bw + l.blockGap()
		} else {
			y0 += bh + l.blockGap()
		}
	}
	return b
}

// stampLine lights line on b at scale k, its top-left cell at x, y.
func stampLine(b *board, f face, line string, k, x, y int) {
	g := gap(k)
	for _, r := range line {
		if gl, ok := f.g[r]; ok && r != ' ' {
			for fy := 0; fy < f.h; fy++ {
				for fx := 0; fx < len(gl[fy]); fx++ {
					if gl[fy][fx] != '#' {
						continue
					}
					for dy := 0; dy < k; dy++ {
						for dx := 0; dx < k; dx++ {
							b.set(x+fx*k+dx, y+fy*k+dy)
						}
					}
				}
			}
		}
		x += glyphCells(f, r, k) + g
	}
}

// A game scene needs room to be played: the runner, a jump over the
// tallest cactus, the ground, and a runway — in its own pixels. A game
// has no size setting (user, 2026-09-24): it is drawn at the largest
// scale, up to sceneMaxScale, that leaves the scene its room.
const (
	sceneMinW     = 40
	sceneMinH     = 25
	sceneMaxScale = 3
)

// fitScene picks a game's scale on a cols × rows canvas: from most,
// stepped down until a scene has its room — or 1, and the game clips
// what it must. The scene is the whole board, which is why the ground
// runs edge to edge.
func fitScene(most, cols, rows int) (k, w, h int) {
	cells := cols / 2
	for k = max(1, most); k > 1; k-- {
		if cells/k >= sceneMinW && rows/k >= sceneMinH {
			break
		}
	}
	return k, cells / k, rows / k
}

// paintScene lights a game's frame at scale k — every scene pixel k × k
// cells, the odd cells left over at the edges dark. Nothing is lettered
// over it: no score, no clock (user, 2026-09-24).
func paintScene(sc saver.Scene, k, cols, rows int) board {
	b := newBoard(cols/2, rows)
	ox, oy := (b.w-sc.W*k)/2, (b.h-sc.H*k)/2
	for y := 0; y < sc.H; y++ {
		for x := 0; x < sc.W; x++ {
			if !sc.Pix[y*sc.W+x] {
				continue
			}
			for dy := 0; dy < k; dy++ {
				for dx := 0; dx < k; dx++ {
					b.set(ox+x*k+dx, oy+y*k+dy)
				}
			}
		}
	}
	return b
}

// boardRows draws the board, one string per row, each exactly cols wide:
// runs of pixels in one colour are rendered together, so a row costs a few
// escape sequences rather than one per cell. An odd terminal leaves its
// rightmost column blank (function.md §5.3). While dimmed — the PIN prompt
// is up — the lit pixels step back to Surface2 and become its backdrop.
func boardRows(b board, bg, fg lipgloss.Color, cols int, dimmed bool) []string {
	if dimmed {
		fg = backdropColor
	}
	off := lipgloss.NewStyle().Foreground(bg)
	on := lipgloss.NewStyle().Foreground(fg)
	tail := spaces(cols - b.w*2)
	rows := make([]string, b.h)
	for y := 0; y < b.h; y++ {
		var sb strings.Builder
		for x := 0; x < b.w; {
			lit := b.at(x, y)
			run := x
			for run < b.w && b.at(run, y) == lit {
				run++
			}
			cells := strings.Repeat(pixelCell, run-x)
			if lit {
				sb.WriteString(on.Render(cells))
			} else {
				sb.WriteString(off.Render(cells))
			}
			x = run
		}
		sb.WriteString(tail)
		rows[y] = sb.String()
	}
	return rows
}

// plainRows is the fallback when no scale fits: the board is laid all dark
// and the lines are set over it as ordinary text in the fg colour, centred.
func plainRows(lines []string, bg, fg lipgloss.Color, cols, rows int, dimmed bool) []string {
	if dimmed {
		fg = backdropColor
	}
	off := lipgloss.NewStyle().Foreground(bg)
	txt := lipgloss.NewStyle().Foreground(fg).Bold(true)
	w := cols / 2
	blank := off.Render(strings.Repeat(pixelCell, w)) + spaces(cols-w*2)
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
		out[y] = off.Render(strings.Repeat(pixelCell, leftPx)) + spaces(leftCols-leftPx*2) +
			txt.Render(l) +
			spaces(rightCols-rightPx*2) + off.Render(strings.Repeat(pixelCell, rightPx))
	}
	return out
}

// statusRow is the last row (function.md §5.4, ui.md §1.2): who is locked
// out of what, since when — and, whatever show says, whether there is a
// PIN at all, and whether the file could be read. Exactly cols wide.
func statusRow(cols int, show bool, user, host string, lockedAt time.Time, noPIN bool, problem string) string {
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
	return clipANSI(" "+b.String(), cols) + spaces(cols-1-plainW)
}
