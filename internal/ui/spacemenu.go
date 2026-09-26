package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// menuItem is one row of the Space menu. A commit dispatches key to the
// panel, so the menu is a discoverability shell over the letter hotkeys
// rather than a second implementation of them — which is what keeps tdp
// M3 honest: every letter hotkey IS a row here, and every row can be run
// without knowing its letter.
type menuItem struct {
	label string
	key   string // dispatched on commit; "enter" for the core-key action
	hint  string
	// header is a region's label — "item operation", "panel operation",
	// "global operation", "key reference" — dim and not a stop (tdp M2).
	header bool
	// rule is the line between two regions (tdp M2): not a stop.
	rule bool
	// ref is a row of the ? menu's key reference (tdp M4): a key and what
	// it does, to read, not to run — not a stop.
	ref bool
	// disabled: the action belongs here but cannot run right now. A row
	// that vanishes teaches that the action does not exist on this panel;
	// a dimmed row keeps the map honest. It keeps its own hint and does
	// nothing when pressed (tdp M6).
	disabled bool
}

// stop is a row the cursor can rest on: an action, run or dimmed.
func (it menuItem) stop() bool { return !it.header && !it.rule && !it.ref }

// regions lays out a menu's regions in order (tdp M2): an empty one left
// out, a rule between two, and the titles only when more than one is left.
func regions(titles []string, groups ...[]menuItem) []menuItem {
	var kept []int
	for i, g := range groups {
		if len(g) > 0 {
			kept = append(kept, i)
		}
	}
	var out []menuItem
	for n, i := range kept {
		if n > 0 {
			out = append(out, menuItem{rule: true})
		}
		if len(kept) > 1 {
			out = append(out, menuItem{label: titles[i], header: true})
		}
		out = append(out, groups[i]...)
	}
	return out
}

// spaceMenu is the Space menu (tdp K5, M2): "what can I do, here, now". A
// second instance is the options list — a saver's time shape, a colour
// channel's 256 numbers — because that is a menu too, only its rows are
// values rather than actions; a third is the ? menu (tdp M4).
type spaceMenu struct {
	anim   popupAnimator
	glyph  string
	items  []menuItem
	cursor int
	top    int // first row shown: the window follows the cursor
	// rows is a window of that many rows when the menu asked for one — a
	// channel's numbers show ten at a time (ux.md §2.1). Zero is what the
	// screen holds.
	rows int
	// pendingG holds the first half of the gg chord, the menu's own.
	pendingG bool
	title    string
	layer    int
	screenW  int
	screenH  int
}

func newSpaceMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("spacemenu"), glyph: glyphMenu}
}

func newOptionsMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("options"), glyph: glyphList}
}

func newHelpMenu() spaceMenu {
	return spaceMenu{anim: newPopupAnimator("helpmenu"), glyph: glyphHelp}
}

func (m *spaceMenu) setItems(items []menuItem, title string, layer int) {
	m.items, m.title, m.layer = items, title, layer
	m.top, m.rows, m.pendingG = 0, 0, false
	m.cursor = m.firstStop()
}

func (m spaceMenu) isActive() bool      { return m.anim.isActive() }
func (m spaceMenu) isInteractive() bool { return m.anim.isInteractive() }
func (m *spaceMenu) open() tea.Cmd      { return m.anim.open() }
func (m *spaceMenu) close() tea.Cmd     { return m.anim.close() }
func (m *spaceMenu) setSize(w, h int)   { m.screenW, m.screenH = w, h }

func (m spaceMenu) firstStop() int {
	for i, it := range m.items {
		if it.stop() {
			return i
		}
	}
	return 0
}

func (m spaceMenu) lastStop() int {
	for i := len(m.items) - 1; i >= 0; i-- {
		if m.items[i].stop() {
			return i
		}
	}
	return 0
}

// center puts the cursor in the middle of the window: an options list
// opens on the current value with room either side (ux.md §2.1).
func (m *spaceMenu) center() {
	vis := m.visible()
	m.top = clamp(m.cursor-vis/2, 0, max(0, len(m.items)-vis))
}

// step moves one row in direction d, over the headers, and WRAPS at the
// ends — off the bottom is the top. A list is a ring, and the last item is
// one keystroke from the first.
func (m *spaceMenu) step(d int) {
	n := len(m.items)
	if n == 0 {
		return
	}
	at := m.cursor
	for i := 0; i < n; i++ {
		at = (at + d + n) % n
		if m.items[at].stop() {
			m.cursor = at
			m.scroll()
			return
		}
	}
}

// jump moves up to |d| rows in direction d and STOPS at the ends: a
// half-page is a movement you aim, and one that teleports to the other
// end is worse than one that stops.
func (m *spaceMenu) jump(d int) {
	dir, n := 1, d
	if d < 0 {
		dir, n = -1, -d
	}
	at := m.cursor
	for i := 0; i < n; i++ {
		next := at + dir
		for next >= 0 && next < len(m.items) && !m.items[next].stop() {
			next += dir
		}
		if next < 0 || next >= len(m.items) {
			break
		}
		at = next
	}
	m.cursor = at
	m.scroll()
}

// visible is how many rows the box shows: capRows' budget, or the window
// the menu asked for when that is smaller.
func (m spaceMenu) visible() int {
	v := max(1, m.screenH-6)
	if m.rows > 0 {
		v = min(v, m.rows)
	}
	return v
}

// scroll keeps the cursor's row in the window.
func (m *spaceMenu) scroll() {
	vis := m.visible()
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+vis {
		m.top = m.cursor - vis + 1
	}
	m.top = max(0, min(m.top, max(0, len(m.items)-vis)))
}

// update handles one keystroke. The returned string is the committed key
// ("" when nothing committed) — the caller dispatches it and closes the menu.
func (m spaceMenu) update(msg tea.KeyMsg) (spaceMenu, string) {
	if !m.anim.isInteractive() {
		return m, ""
	}
	k := msg.String()
	if m.pendingG {
		m.pendingG = false
		if k == "g" {
			m.cursor = m.firstStop()
			m.scroll()
			return m, ""
		}
	} else if k == "g" {
		m.pendingG = true
		return m, ""
	}
	switch k {
	case "j", "down":
		m.step(1)
	case "k", "up":
		m.step(-1)
	case "G":
		m.cursor = m.lastStop()
		m.scroll()
	case "d", "ctrl+d":
		m.jump(max(1, m.visible()/2))
	case "u", "ctrl+u":
		m.jump(-max(1, m.visible()/2))
	case "enter":
		// A dimmed row does nothing, and the menu stays (tdp M6).
		if m.cursor < len(m.items) && m.items[m.cursor].stop() && !m.items[m.cursor].disabled {
			return m, m.items[m.cursor].key
		}
	default:
		// Letter hotkeys work from inside the menu too: the menu is the
		// slow path and the letter is the fast one, and they must agree.
		keys := m.menuKeys()
		if i := hotkeyIndex(keys, k); i >= 0 && !m.items[i].disabled {
			return m, keys[i]
		}
	}
	return m, ""
}

func (m spaceMenu) view() string {
	labelW, hintW, headW := 0, 0, 0
	for _, it := range m.items {
		switch {
		case it.rule:
			continue
		case it.header:
			headW = max(headW, dispW(it.label)+2)
			continue
		case it.ref:
			labelW = max(labelW, dispW(it.label))
			hintW = max(hintW, dispW(it.hint))
			continue
		}
		labelW = max(labelW, dispW(bracketHotkey(it.label, it.key)))
		hintW = max(hintW, dispW(it.hint))
	}
	legend := hintLegend([][2]string{{"j/k", "move"}, {"Enter", "run"}, {"Esc", "close"}})
	if len(m.items) == 0 {
		legend = hintLegend([][2]string{{"Esc", "close"}})
	}
	innerW := popupInnerW(m.screenW, max(dispW(m.title)+6, labelW+hintW+4, headW, dispW(legend)+1))
	hintW = max(0, min(hintW, innerW-labelW-3))

	dim := lipgloss.NewStyle().Foreground(dimColor)
	txt := lipgloss.NewStyle().Foreground(textColor)
	cur := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(handColor)
	curOff := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(borderDim)

	rows := make([]string, 0, len(m.items))
	for i, it := range m.items {
		switch {
		case it.rule:
			rows = append(rows, dim.Render(strings.Repeat("─", innerW)))
			continue
		case it.header:
			rows = append(rows, dim.Render(padRight(" "+it.label, innerW)))
			continue
		case it.ref:
			// A key and what it does, to read: the key in the text colour,
			// never under the cursor.
			rows = append(rows, txt.Render(padRight(" "+it.label, innerW-hintW-1))+dim.Render(padLeft(it.hint, hintW)+" "))
			continue
		}
		label := padRight(" "+bracketHotkey(it.label, it.key), innerW-hintW-1)
		hint := padLeft(it.hint, hintW) + " "
		switch {
		case i == m.cursor && it.disabled:
			rows = append(rows, curOff.Render(label+hint))
		case i == m.cursor:
			rows = append(rows, cur.Render(label+hint))
		case it.disabled:
			rows = append(rows, dim.Render(label+hint))
		default:
			rows = append(rows, txt.Render(label)+dim.Render(hint))
		}
	}
	vis := m.visible()
	top := min(max(m.top, 0), max(0, len(rows)-vis))
	if m.cursor < top {
		top = m.cursor
	}
	if m.cursor >= top+vis {
		top = m.cursor - vis + 1
	}
	rows = rows[top:min(len(rows), top+vis)]
	return drawPopupBox(popupLayerColor(m.layer), " "+m.glyph+" "+m.title+" ",
		legend, animRows(m.anim, capRows(rows, m.screenH)), innerW)
}

// menuKeys is every key the menu would fire, one per row, headers as "".
func (m spaceMenu) menuKeys() []string {
	keys := make([]string, len(m.items))
	for i, it := range m.items {
		if it.stop() {
			keys[i] = it.key
		}
	}
	return keys
}
