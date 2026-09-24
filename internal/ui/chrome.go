package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// The settings screen's frame (ui.md §1.1, §5): two panels side by side
// and one footer row. Focus is the border — double line in blue where the
// keyboard is, a rounded thin line in Surface2 where it is not (kbu §8.4)
// — and nothing moves when it changes.

// sideW is panel [1]'s width, borders included: fixed, like webu's.
const sideW = 24

// narrowW is where the two panels stop fitting side by side: below it
// only the focused one is drawn (ui.md §1.3).
const narrowW = 60

// panelFrame frames body — every line already innerW cells — with a title
// in the top border and, for the unfocused rounded frame, a hint in the
// bottom one. Focused frames carry no hint: the hint is the config's
// path, a resting fact, and it lives on [2] whichever side has the keys.
func panelFrame(innerW int, body []string, title, hint string, focused bool) string {
	bc := borderDim
	tl, tr, bl, br, h, v := "╭", "╮", "╰", "╯", "─", "│"
	if focused {
		bc = focusColor
		tl, tr, bl, br, h, v = "╔", "╗", "╚", "╝", "═", "║"
	}
	bs := lipgloss.NewStyle().Foreground(bc)
	ts := lipgloss.NewStyle().Foreground(bc).Bold(true)
	hs := lipgloss.NewStyle().Foreground(dimColor)

	out := make([]string, 0, len(body)+2)
	// " title " inside the top border, or nothing when it cannot fit.
	titleW := 0
	top := ""
	if title != "" && dispW(title)+2 <= innerW {
		titleW = dispW(title) + 2
		top = " " + ts.Render(title) + " "
	}
	out = append(out, bs.Render(tl)+top+bs.Render(strings.Repeat(h, max(0, innerW-titleW))+tr))
	side := bs.Render(v)
	for _, l := range body {
		out = append(out, side+l+strings.Repeat(" ", max(0, innerW-dispW(l)))+side)
	}
	// The hint sits right of centre on the bottom border, four cells of
	// line after it, when there is room for it and some line before it.
	bottom := strings.Repeat(h, innerW)
	if hint != "" {
		hint = truncate(hint, innerW-8)
		if hw := dispW(hint); hw > 0 && hw+8 <= innerW {
			bottom = strings.Repeat(h, innerW-hw-6) + " " + hs.Render(hint) + " " + strings.Repeat(h, 4)
			out = append(out, bs.Render(bl)+bs.Render(strings.Repeat(h, innerW-hw-6))+" "+hs.Render(hint)+" "+bs.Render(strings.Repeat(h, 4)+br))
			return strings.Join(out, "\n")
		}
	}
	out = append(out, bs.Render(bl+bottom+br))
	return strings.Join(out, "\n")
}

// keyLegend renders the footer's "key desc" pairs: the standing
// disclosure of the two entry keys (§A.1 / §A.2). A user who never read a
// README learns Space and ? exist by reading this row. When the terminal
// is too narrow, pairs are dropped from the RIGHT — the entry keys are
// listed first precisely so they are the last thing to go.
func keyLegend(pairs [][2]string, w int) string {
	const sep = "   "
	plainW := func(n int) int {
		total := 1
		for i := 0; i < n; i++ {
			if i > 0 {
				total += dispW(sep)
			}
			total += dispW(pairs[i][0]) + 1 + dispW(pairs[i][1])
		}
		return total
	}
	n := len(pairs)
	for n > 0 && plainW(n) > w {
		n--
	}
	k := lipgloss.NewStyle().Foreground(focusColor)
	d := lipgloss.NewStyle().Foreground(dimColor)
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		parts = append(parts, k.Render(pairs[i][0])+" "+d.Render(pairs[i][1]))
	}
	return " " + strings.Join(parts, sep) + strings.Repeat(" ", max(0, w-plainW(n)))
}

// joinHorizontal is a display-width aware block join.
func joinHorizontal(blocks ...string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, blocks...)
}
