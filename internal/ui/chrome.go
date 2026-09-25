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

// chip is one segment of a panel's title: its text, and its fill — the
// border's colour when border is set, whatever fill says otherwise, and
// unlit, the canvas with dim ink, when neither.
type chip struct {
	text   string
	fill   lipgloss.Color
	border bool
}

// titleChain draws chips as ONE powerline strip, the way the family's
// tab rows and webu's pagetab are drawn: round cap, segments run
// together, a slanted seam between neighbours, round cap. Every chip is
// a fill with the canvas for ink (chipFill): a panel without the keys
// wears one fill all along, the unfocused border's, and with them the
// first chip lights in the border's blue and a state chip in its own
// colour. locku drew its titles as words parted by a middle dot until
// 2026-09-25 — the one member still doing so — and the user asked for
// the family's capsules: the dot cost three cells where a seam costs
// one, and a chain lets each part wear its own colour, so what the
// panel is, what kind it is and what state it is in are read apart.
func titleChain(chips []chip, bc lipgloss.Color, focused bool) string {
	canvas := lipgloss.Color(baseHex)
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(chipFill(chips[0], bc, focused)).Render(capLeft))
	for i, c := range chips {
		f := chipFill(c, bc, focused)
		if i > 0 {
			div, fg, bg := divider(chipFill(chips[i-1], bc, focused), f)
			b.WriteString(lipgloss.NewStyle().Foreground(fg).Background(bg).Render(div))
		}
		// Flush against the caps, as sshu's and filu's capsules are; a
		// space either side of a seam, so no word touches the triangle
		// (user, 2026-09-25: the space after the round cap was one too
		// many).
		seg := c.text
		if i > 0 {
			seg = " " + seg
		}
		if i < len(chips)-1 {
			seg += " "
		}
		b.WriteString(lipgloss.NewStyle().Foreground(canvas).Background(f).Bold(f != borderDim).Render(seg))
	}
	b.WriteString(lipgloss.NewStyle().Foreground(chipFill(chips[len(chips)-1], bc, focused)).Render(capRight))
	return b.String()
}

// chipFill is the colour a chip wears in a frame whose border is bc:
// the border's own for the first chip; for a chip with a colour of its
// own — installed's green, unsaved's yellow — that colour while the
// panel has the keys; everything else, and everything on a panel
// without them, the unfocused border's grey (user, 2026-09-25: the
// chips are one grey strip until the panel is focused, and then
// unsaved lights while the rest keep the grey — the state that
// wants noticing is the one that lights).
func chipFill(c chip, bc lipgloss.Color, focused bool) lipgloss.Color {
	switch {
	case c.border:
		return bc
	case focused && c.fill != "":
		return c.fill
	}
	return borderDim
}

// divider is the seam between two chips: where the fills differ, the
// left chip's own edge — a filled triangle in its colour over the
// right's — and between two alike, a thin slash in the canvas's ink.
// The triangle belongs to the chip on its LEFT (webu: get it backwards
// and the seam reads as a notch cut out of the wrong chip).
func divider(prev, cur lipgloss.Color) (glyph string, fg, bg lipgloss.Color) {
	if prev == cur {
		return dividerSoft, lipgloss.Color(baseHex), cur
	}
	return dividerHard, prev, cur
}

// chainW is the cells a chain takes: two caps, each chip's text, and
// between neighbours a seam with a space either side.
func chainW(chips []chip) int {
	w := 2 + 3*(len(chips)-1)
	for _, c := range chips {
		w += dispW(c.text)
	}
	return w
}

// panelFrame frames body — every line already innerW cells — with a title
// chain flush after the top border's first corner, as sshu seats its
// capsule, a tag — what kind of thing the panel shows — flush before
// its other corner, on its own (user, 2026-09-25: the kind is not part
// of the title), and, for the unfocused rounded frame, a hint in the
// bottom border. Focused frames carry no hint: the hint is the config's
// path, a resting fact, and it lives on [2] whichever side has the
// keys. A chain too wide for the border sheds its state chip; a tag
// with no room beside it is left out, and so is a chain wider than the
// border.
func panelFrame(innerW int, body []string, title []chip, tag, hint string, focused bool) string {
	bc := borderDim
	tl, tr, bl, br, h, v := "╭", "╮", "╰", "╯", "─", "│"
	if focused {
		bc = focusColor
		tl, tr, bl, br, h, v = "╔", "╗", "╚", "╝", "═", "║"
	}
	bs := lipgloss.NewStyle().Foreground(bc)
	hs := lipgloss.NewStyle().Foreground(dimColor)

	out := make([]string, 0, len(body)+2)
	titleW, top := 0, ""
	if len(title) > 1 && chainW(title) > innerW {
		title = title[:1]
	}
	if len(title) > 0 && chainW(title) <= innerW {
		titleW = chainW(title)
		top = titleChain(title, bc, focused)
	}
	tagW, right := 0, ""
	if tag != "" && titleW+dispW(tag)+2+1 <= innerW {
		tagW = dispW(tag) + 2
		right = titleChain([]chip{{text: tag, border: true}}, bc, focused)
	}
	out = append(out, bs.Render(tl)+top+bs.Render(strings.Repeat(h, max(0, innerW-titleW-tagW)))+right+bs.Render(tr))
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
