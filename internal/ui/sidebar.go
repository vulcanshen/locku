package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/saver"
)

// Panel [1] (ui.md §1.1): three groups, their titles in blue, one under
// the other with no gap. Profiles first: the ones the user has set up —
// named, configured savers: the objects. Then Savers, the kinds there
// are — the clock, the dino: the classes, which have no name but their
// own and are not made or deleted (user, 2026-09-24, who drew the line
// and put the profiles on top). The active profile — the one preference
// › profile names — carries a green dot, and the dot is all it is: it
// shows, it does not set. Settings is one row, preference. The group
// titles are not stops.

type sideKind int

const (
	sideProfile    sideKind = iota // a profile, by its index in cfg.Profiles
	sideSaver                      // a kind of saver, by its index in saver.Kinds
	sidePreference                 // the one settings row
)

// sideItem is one stop of the cursor.
type sideItem struct {
	kind sideKind
	ref  int
}

// sideItems is the stops in order: the profiles, the savers, preference.
func (m AppModel) sideItems() []sideItem {
	items := make([]sideItem, 0, len(m.cfg.Profiles)+len(saver.Kinds)+1)
	for i := range m.cfg.Profiles {
		items = append(items, sideItem{kind: sideProfile, ref: i})
	}
	for i := range saver.Kinds {
		items = append(items, sideItem{kind: sideSaver, ref: i})
	}
	return append(items, sideItem{kind: sidePreference})
}

// profileItem is the cursor index of the profile at i.
func profileItem(i int) int { return i }

// sideAt is the item under the cursor.
func (m AppModel) sideAt() sideItem {
	items := m.sideItems()
	return items[clamp(m.cur1, 0, len(items)-1)]
}

// sideLine is one display row: an item's index, or -1 for a group title.
type sideLine struct {
	text string
	item int
}

func (m AppModel) sideLines() []sideLine {
	var out []sideLine
	out = append(out, sideLine{text: "Profiles", item: -1})
	for i, p := range m.cfg.Profiles {
		mark := "  "
		if p.Name == m.cfg.Profile {
			mark = "● "
		}
		out = append(out, sideLine{text: mark + p.Name, item: profileItem(i)})
	}
	n := len(m.cfg.Profiles)
	out = append(out, sideLine{text: "Savers", item: -1})
	for i, k := range saver.Kinds {
		out = append(out, sideLine{text: "  " + k, item: n + i})
	}
	out = append(out, sideLine{text: "Settings", item: -1})
	out = append(out, sideLine{text: "  preference", item: n + len(saver.Kinds)})
	return out
}

// sidebarBody draws panel [1]'s rows at innerW × innerH. The window follows
// the cursor when the rows outgrow the panel.
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
	title := lipgloss.NewStyle().Foreground(focusColor)
	live := lipgloss.NewStyle().Foreground(liveColor)

	out := make([]string, 0, innerH)
	for i := top; i < len(lines) && len(out) < innerH; i++ {
		l := lines[i]
		switch {
		case l.item < 0:
			out = append(out, title.Render(padRight(" "+l.text, innerW)))
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
