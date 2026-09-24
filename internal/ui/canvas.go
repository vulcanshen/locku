package ui

import (
	"strings"
	"time"
	"unicode/utf8"

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
	gapX       = 1 // pixels between two letters
	gapY       = 2 // pixels between two lines
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

// widest is the longest line in runes.
func widest(lines []string) int {
	n := 0
	for _, l := range lines {
		n = max(n, utf8.RuneCountInString(l))
	}
	return n
}

// pixelSize is the block lines need in font pixels, before scaling: n
// letters cost 5n + (n − 1), m lines 7m + 2(m − 1) (function.md §5.3).
func pixelSize(lines []string) (w, h int) {
	n, m := widest(lines), len(lines)
	if n == 0 || m == 0 {
		return 0, 0
	}
	return n*fontW + (n-1)*gapX, m*fontH + (m-1)*gapY
}

// scaleFor is the scale at which lines fit a canvas of cols × rows inside
// the margins: kx, the largest whole factor that fits both ways, and ky,
// the same factor stretched taller — up to half again — where the rows
// allow it. A pixel is two columns by one row, near enough square, so
// kx = ky draws the font as designed; the extra height makes the digits
// tall, the way a clock's are, instead of leaving the rows above and
// below empty (user, 2026-09-24). Pixels are never made wider than tall.
// (0, 0) when even 1 × 1 does not fit. rows is the canvas: the terminal's
// height less the status row.
func scaleFor(lines []string, cols, rows int) (kx, ky int) {
	pw, ph := pixelSize(lines)
	if pw == 0 {
		return 0, 0
	}
	availPx := (cols - marginCols) / 2
	availRows := rows - marginRows
	if availPx <= 0 || availRows <= 0 {
		return 0, 0
	}
	kx = min(availPx/pw, availRows/ph)
	if kx < 1 {
		return 0, 0
	}
	ky = min(availRows/ph, kx+kx/2)
	return kx, ky
}

// fit walks the saver down its degrade ladder until its lines fit, and
// says at what scale. kx == 0 means the ladder ran out: the lines it ends
// on are drawn as plain text over the board (function.md §5.3 step 4).
func fit(s saver.Saver, now time.Time, cols, rows int) (lines []string, kx, ky int) {
	for {
		lines = s.Lines(now)
		if kx, ky = scaleFor(lines, cols, rows); kx >= 1 {
			return lines, kx, ky
		}
		next, ok := s.Degrade()
		if !ok {
			return lines, 0, 0
		}
		s = next
	}
}

// paint lights lines at scale kx × ky on a fresh board for a cols × rows
// canvas, the block centred, each line centred within the block in whole
// font pixels. kx < 1 gives an empty board.
func paint(lines []string, kx, ky, cols, rows int) board {
	b := newBoard(cols/2, rows)
	if kx < 1 || ky < 1 {
		return b
	}
	pw, ph := pixelSize(lines)
	ox := (b.w - pw*kx) / 2
	oy := (b.h - ph*ky) / 2
	for i, line := range lines {
		runes := []rune(line)
		if len(runes) == 0 {
			continue
		}
		lineW := len(runes)*fontW + (len(runes)-1)*gapX
		lx := ox + (pw-lineW)/2*kx
		ly := oy + i*(fontH+gapY)*ky
		for j, r := range runes {
			g, ok := font[r]
			if !ok {
				continue
			}
			gx := lx + j*(fontW+gapX)*kx
			for fy := 0; fy < fontH; fy++ {
				for fx := 0; fx < fontW; fx++ {
					if g[fy][fx] != '#' {
						continue
					}
					for dy := 0; dy < ky; dy++ {
						for dx := 0; dx < kx; dx++ {
							b.set(gx+fx*kx+dx, ly+fy*ky+dy)
						}
					}
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
