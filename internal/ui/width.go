package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// Everything drawn into a fixed slot goes through these. A panel is a
// fixed-width box (tdp L2), so a field that miscounts its own width does not
// merely look off — it pushes the right border out and breaks the frame.
//
// Rule: measure and pad PLAIN text, then apply the style. dispW knows how to
// skip ANSI, but padding a styled string means the pad lands inside the
// styled span and picks up its background.
//
// Every width is measured here, and nowhere else by lipgloss or x/ansi
// (tdp D6, 2026-09-29; filu's width.go is the reference): with some fonts
// an icon moves the cursor two cells where they count one, and a box that
// counts it one comes out crooked.

// iconCells is how many cells a Nerd Font icon moves the cursor: 1 on most
// fonts, 2 on some made for CJK. DetectIconWidth sets it at start; 1 leaves
// every width as lipgloss and x/ansi measure it.
var iconCells = 1

// isWideIcon reports whether r is a Nerd Font icon that such a font draws
// two cells wide: the Private Use Areas, less the powerline caps
// (U+E0A0–E0D7), which stay one cell even there.
func isWideIcon(r rune) bool {
	if r >= 0xe0a0 && r <= 0xe0d7 {
		return false
	}
	return (r >= 0xe000 && r <= 0xf8ff) || (r >= 0xf0000 && r <= 0xffffd)
}

// iconCount is how many wide icons s holds, styles aside; none are counted
// while an icon is one cell.
func iconCount(s string) int {
	if iconCells == 1 {
		return 0
	}
	n := 0
	for _, r := range ansi.Strip(s) {
		if isWideIcon(r) {
			n++
		}
	}
	return n
}

// dispW is the terminal cell width of s: as measured, and a cell more for
// each icon where an icon takes two.
func dispW(s string) int { return ansi.StringWidth(s) + iconCount(s)*(iconCells-1) }

// truncate clips s to at most w cells, marking the cut with a single-cell "…".
// w <= 0 yields "". A string that already fits is returned untouched.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if dispW(s) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := dispW(string(r))
		if used+rw > w-1 { // leave one cell for the ellipsis
			break
		}
		b.WriteRune(r)
		used += rw
	}
	return b.String() + strings.Repeat(" ", w-1-used) + "…"
}

// truncateHead cuts from the FRONT, keeping the tail: where the cursor is.
func truncateHead(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if dispW(s) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	r := []rune(s)
	used, i := 0, len(r)
	for ; i > 0; i-- {
		rw := dispW(string(r[i-1]))
		if used+rw > w-1 { // leave one cell for the ellipsis
			break
		}
		used += rw
	}
	return "…" + strings.Repeat(" ", w-1-used) + string(r[i:])
}

// clipANSI cuts a possibly-styled string to w cells without severing an escape
// sequence. truncate() is for plain text; using it on styled output would cut
// mid-ANSI and bleed the style into everything after it. The first w measured
// cells are at least w on screen, and each icon among them takes one more, so
// it steps back until they fit.
func clipANSI(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if dispW(s) <= w {
		return s
	}
	for t := w; t > 0; t-- {
		if out := ansi.Truncate(s, t, ""); dispW(out) <= w {
			return out
		}
	}
	return ""
}

// dispCutLeft drops the first n cells of s and returns the rest, styles
// kept. An icon or wide character cut in half becomes spaces, so the rest
// is always dispW(s) − n wide.
func dispCutLeft(s string, n int) string {
	if n <= 0 {
		return s
	}
	total := dispW(s)
	if n >= total {
		return ""
	}
	// m measured cells hold at least n on screen; start where they would if
	// every icon so far were narrow, and step past a wide one.
	m := max(n-iconCount(s)*(iconCells-1), 0)
	for dispW(ansi.Truncate(s, m, "")) < n {
		m++
	}
	rest := ansi.TruncateLeft(s, m, "")
	return strings.Repeat(" ", max(total-n-dispW(rest), 0)) + rest
}

// compositeDisp draws fg over bg as overlay.Composite does, but every width
// is dispW, so an icon in the popup or in what it covers cannot push a line
// past the screen (tdp D6, L4). Left / Top at 0, Center at half the
// background less half the foreground (each halved on its own), Right /
// Bottom flush; then moved by the offsets and kept on screen.
//
// A box wider or taller than the screen — the frame a resize lands in, or
// the PIN prompt on a lock a few rows high — starts at 0 and what falls
// off the screen is cut (tdp D6, v0.1.21, 2026-09-29; its middle, as
// overlay shows it, until then). That holds when it is larger both ways
// too, where overlay handed back the box whole, wider than the screen.
func compositeDisp(fg, bg string, xPos, yPos overlay.Position, xOff, yOff int) string {
	if fg == "" {
		return bg
	}
	if bg == "" {
		return fg
	}
	fgLines, bgLines := strings.Split(fg, "\n"), strings.Split(bg, "\n")
	fgW, bgW := blockWidth(fgLines), blockWidth(bgLines)
	fgH, bgH := len(fgLines), len(bgLines)
	x := max(clampSpan(placeOffset(xPos, bgW, fgW)+xOff, bgW-fgW), 0)
	y := max(clampSpan(placeOffset(yPos, bgH, fgH)+yOff, bgH-fgH), 0)
	for i, line := range fgLines {
		if y+i >= bgH {
			break
		}
		line = clipANSI(line, bgW-x)
		row := bgLines[y+i]
		left := clipANSI(row, x)
		left += spaces(x - dispW(left)) // an icon cut at x, or a short row
		right := dispCutLeft(row, x+dispW(line))
		bgLines[y+i] = left + line + right
	}
	return strings.Join(bgLines, "\n")
}

// centerDisp centres s in a w × h area by dispW, as lipgloss.Place(w, h,
// Center, Center, s) does: the smaller half of the gap goes left and on
// top. In a direction s already fills it is left as it is; h 0 centres
// across only.
func centerDisp(w, h int, s string) string {
	lines := strings.Split(s, "\n")
	width := blockWidth(lines)
	if w > width {
		for i, l := range lines {
			gap := w - dispW(l)
			lines[i] = spaces(gap/2) + l + spaces(gap-gap/2)
		}
		width = w
	}
	if gap := h - len(lines); gap > 0 {
		blank := spaces(width)
		out := make([]string, 0, h)
		for range gap / 2 {
			out = append(out, blank)
		}
		out = append(out, lines...)
		for len(out) < h {
			out = append(out, blank)
		}
		lines = out
	}
	return strings.Join(lines, "\n")
}

// blockWidth is the width of the widest line.
func blockWidth(lines []string) int {
	w := 0
	for _, l := range lines {
		w = max(w, dispW(l))
	}
	return w
}

// placeOffset is where a span of size fg starts in one of size bg.
func placeOffset(p overlay.Position, bg, fg int) int {
	switch p {
	case overlay.Center:
		return bg/2 - fg/2
	case overlay.Right, overlay.Bottom:
		return bg - fg
	}
	return 0
}

// clampSpan keeps v between 0 and hi, either way round, as overlay does.
func clampSpan(v, hi int) int {
	lo := 0
	if lo > hi {
		lo, hi = hi, lo
	}
	return min(max(v, lo), hi)
}

// joinH lays blocks side by side, each block's lines padded to that
// block's own width, so an icon in one never shoves the next along — as
// lipgloss.JoinHorizontal does, whose widths count an icon one cell.
func joinH(blocks ...string) string {
	rows := make([][]string, len(blocks))
	widths := make([]int, len(blocks))
	maxRows := 0
	for i, b := range blocks {
		rows[i] = strings.Split(b, "\n")
		widths[i] = blockWidth(rows[i])
		maxRows = max(maxRows, len(rows[i]))
	}
	var out strings.Builder
	for r := 0; r < maxRows; r++ {
		for i := range rows {
			if r < len(rows[i]) {
				l := clipANSI(rows[i][r], widths[i])
				out.WriteString(l + spaces(widths[i]-dispW(l)))
			} else {
				out.WriteString(spaces(widths[i]))
			}
		}
		if r < maxRows-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

// padRight fits s into exactly w cells, truncating or right-padding as needed.
func padRight(s string, w int) string {
	s = truncate(s, w)
	return s + strings.Repeat(" ", max(0, w-dispW(s)))
}

// padLeft fits s into exactly w cells, right-aligned.
func padLeft(s string, w int) string {
	s = truncate(s, w)
	return strings.Repeat(" ", max(0, w-dispW(s))) + s
}

// spaces is n blanks.
func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

// fitLines forces body to exactly h lines, each padded to w cells — the
// invariant panelChrome relies on to keep its right border straight.
func fitLines(body []string, w, h int) []string {
	if len(body) > h {
		body = body[:max(0, h)]
	}
	out := make([]string, 0, max(0, h))
	for _, l := range body {
		out = append(out, l+strings.Repeat(" ", max(0, w-dispW(l))))
	}
	for len(out) < h {
		out = append(out, strings.Repeat(" ", max(0, w)))
	}
	return out
}

func clamp(v, lo, hi int) int { return min(max(v, lo), hi) }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// wrap breaks s into lines no wider than w, at spaces; a word wider than
// w is cut. Nothing comes back for an empty s or a w under one.
func wrap(s string, w int) []string {
	if w < 1 {
		return nil
	}
	var out []string
	line := ""
	for _, word := range strings.Fields(s) {
		for dispW(word) > w {
			if line != "" {
				out = append(out, line)
				line = ""
			}
			r := []rune(word)
			out = append(out, string(r[:w]))
			word = string(r[w:])
		}
		switch {
		case line == "":
			line = word
		case dispW(line)+1+dispW(word) <= w:
			line += " " + word
		default:
			out = append(out, line)
			line = word
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}
