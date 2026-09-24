package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Panel [1] (ui.md §1.1): two groups. Savers lists every instance, the
// active one with a green dot — the sidebar's only green. Settings is two
// rows, config and style. The group labels are separators, not stops.

type sideKind int

const (
	sideSaver sideKind = iota
	sideConfig
	sideStyle
)

// sideItem is one stop of the cursor: a saver (by index) or a settings row.
type sideItem struct {
	kind  sideKind
	saver int
}

func (m AppModel) sideItems() []sideItem {
	items := make([]sideItem, 0, len(m.cfg.Savers)+2)
	for i := range m.cfg.Savers {
		items = append(items, sideItem{kind: sideSaver, saver: i})
	}
	return append(items, sideItem{kind: sideConfig}, sideItem{kind: sideStyle})
}

// sideAt is the item under the cursor.
func (m AppModel) sideAt() sideItem {
	items := m.sideItems()
	return items[clamp(m.cur1, 0, len(items)-1)]
}

// sideLine is one display row: an item's index, or -1 for a label or a
// blank.
type sideLine struct {
	text string
	item int
}

func (m AppModel) sideLines() []sideLine {
	var out []sideLine
	out = append(out, sideLine{text: "Savers", item: -1})
	for i, s := range m.cfg.Savers {
		mark := "  "
		if s.Name == m.cfg.Saver {
			mark = "● "
		}
		out = append(out, sideLine{text: mark + s.Name, item: i})
	}
	out = append(out, sideLine{item: -1})
	out = append(out, sideLine{text: "Settings", item: -1})
	out = append(out, sideLine{text: "  config", item: len(m.cfg.Savers)})
	out = append(out, sideLine{text: "  style", item: len(m.cfg.Savers) + 1})
	return out
}

// sidebarBody draws panel [1]'s rows at innerW × innerH. The window follows
// the cursor when the savers outgrow the panel.
func (m AppModel) sidebarBody(innerW, innerH int) []string {
	lines := m.sideLines()
	curRow := 0
	for i, l := range lines {
		if l.item == m.cur1 {
			curRow = i
		}
	}
	top := scrollTo(m.top1, curRow, innerH)

	cur := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(handColor)
	curOff := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(borderDim)
	txt := lipgloss.NewStyle().Foreground(textColor)
	dim := lipgloss.NewStyle().Foreground(dimColor)
	live := lipgloss.NewStyle().Foreground(liveColor)

	out := make([]string, 0, innerH)
	for i := top; i < len(lines) && len(out) < innerH; i++ {
		l := lines[i]
		switch {
		case l.item < 0:
			out = append(out, dim.Render(padRight(" "+l.text, innerW)))
		case l.item == m.cur1 && m.focus == panelSide:
			out = append(out, cur.Render(padRight(" "+l.text, innerW)))
		case l.item == m.cur1:
			out = append(out, curOff.Render(padRight(" "+l.text, innerW)))
		case strings.HasPrefix(l.text, "● "):
			// The dot is green; the name is text. Two styles, one row.
			out = append(out, " "+live.Render("●")+txt.Render(padRight(l.text[len("●"):], innerW-2)))
		default:
			out = append(out, txt.Render(padRight(" "+l.text, innerW)))
		}
	}
	return fitLines(out, innerW, innerH)
}

// scrollTo keeps cursor inside a window of h rows starting at top.
func scrollTo(top, cursor, h int) int {
	if h <= 0 {
		return 0
	}
	if cursor < top {
		return cursor
	}
	if cursor >= top+h {
		return cursor - h + 1
	}
	return max(0, top)
}
