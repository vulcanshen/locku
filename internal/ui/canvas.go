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
// the board's bg colour, lit pixels its fg. The saver's lines are set in the
// 5 × 7 font, scaled by the largest whole k that fits, and centred.
//
// It is the only way anything is drawn. A saver never sees a colour, a
// glyph or a position.

const (
	gapX       = 1 // pixels between two glyphs
	gapY       = 1 // pixels between two lines (2 until 2026-09-24: a column's lines are rows the large size cannot spare)
	marginCols = 4 // two columns of margin either side
	marginRows = 2 // one row above and below
)

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

// lineW is a line's width in font pixels: its glyphs and a gap between
// each two.
func lineW(line string) int {
	w := 0
	for i, r := range line {
		if i > 0 {
			w += gapX
		}
		w += glyphW(r)
	}
	return w
}

// pixelSize is the block lines need in font pixels, before scaling: the
// widest line, and m lines at 7 rows each with a gap between (function.md
// §5.3).
func pixelSize(lines []string) (w, h int) {
	m := len(lines)
	for _, l := range lines {
		w = max(w, lineW(l))
	}
	if w == 0 || m == 0 {
		return 0, 0
	}
	return w, m*fontH + (m-1)*gapY
}

// fitsAt reports whether lines fit a canvas of cols × rows at scale k,
// inside the margins. A pixel is two columns by one row, near enough
// square, so k × k draws the font as designed. rows is the canvas: the
// terminal's height less the status row.
func fitsAt(lines []string, k, cols, rows int) bool {
	pw, ph := pixelSize(lines)
	if pw == 0 || k < 1 {
		return false
	}
	availPx := (cols - marginCols) / 2
	availRows := rows - marginRows
	return pw*k <= availPx && ph*k <= availRows
}

// fit finds what to draw and how big (function.md §5.3): the size the
// saver asks for — small, medium, large: 1, 2, 3 — is kept as long as
// some of the content fits at it, the content stepping down the degrade
// ladder (the year, the seconds, the date); only when nothing fits at
// that size does the size step down, and only when nothing fits at 1 do
// the lines come back with k == 0, to be drawn as plain text over the
// board (user, 2026-09-24: the size is a choice, not a computation).
func fit(s saver.Saver, now time.Time, cols, rows, size int) (lines []string, k int) {
	for k = max(1, size); k >= 1; k-- {
		sv := s
		for {
			lines = sv.Lines(now)
			if fitsAt(lines, k, cols, rows) {
				return lines, k
			}
			next, ok := sv.Degrade()
			if !ok {
				break
			}
			sv = next
		}
	}
	return lines, 0
}

// paint lights lines at scale k on a fresh board for a cols × rows canvas,
// the block centred, each line centred within the block in whole font
// pixels. k < 1 gives an empty board.
func paint(lines []string, k, cols, rows int) board {
	b := newBoard(cols/2, rows)
	if k < 1 {
		return b
	}
	pw, ph := pixelSize(lines)
	ox := (b.w - pw*k) / 2
	oy := (b.h - ph*k) / 2
	for i, line := range lines {
		if line == "" {
			continue
		}
		lx := ox + (pw-lineW(line))/2*k
		ly := oy + i*(fontH+gapY)*k
		x := 0 // in font pixels along the line
		for _, r := range line {
			g, ok := font[r]
			if !ok {
				x += fontW + gapX
				continue
			}
			gx := lx + x*k
			for fy := 0; fy < fontH; fy++ {
				for fx := 0; fx < len(g[fy]); fx++ {
					if g[fy][fx] != '#' {
						continue
					}
					for dy := 0; dy < k; dy++ {
						for dx := 0; dx < k; dx++ {
							b.set(gx+fx*k+dx, ly+fy*k+dy)
						}
					}
				}
			}
			x += glyphW(r) + gapX
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
