package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
)

// Panel [2] (ui.md §1.1): the detail of whatever [1]'s cursor is on, shown
// at once — there is no Enter to open it. A saver is its fields and its
// two colours, each a swatch and three channel sliders; preference is the
// PIN, the active saver and the four settings. Every key config.yaml has
// is a row here, except `savers`, which IS panel [1].
//
// A saver's colours edit a DRAFT (user, 2026-09-24): the sliders move a
// copy, the swatch row shows the saved colour and, when it differs, the
// draft beside it, and nothing reaches the file until [S] Save; [R] Reset
// drops the draft. Every other row writes at once.

type rowKind int

const (
	rowName rowKind = iota
	rowType
	rowLayout
	rowSize
	rowFont
	rowTime
	rowDate
	rowSwatch
	rowChannel
	rowPIN
	rowSaver
	rowShowStatus
	rowPromptTimeout
	rowLockoutAfter
	rowLockoutSeconds
	rowTmuxConf
	rowScreenConf
)

// row is one line of panel [2].
type row struct {
	kind  rowKind
	label string
	value string         // plain
	color lipgloss.Color // the value's ink
	stop  bool           // the cursor can rest here
	// which colour (0 bg, 1 fg) and which channel (0 r, 1 g, 2 b) a
	// swatch or channel row is about; num is the channel's value.
	which, ch, num int
	// a swatch row: the saved colour, and the draft when it differs.
	hex, draft string
}

// labelW is the label column; sliders and values start after it. The
// widest label, lockout_seconds, is fifteen, and a value wants air.
const labelW = 18

// draftOf is a saver's colours as the sliders have them: the draft when
// one is up, the file's otherwise.
func (m AppModel) draftOf(s config.Saver) config.Style {
	if d, ok := m.drafts[s.Name]; ok {
		return d
	}
	return s.Colours()
}

// dirtyOf reports whether a saver has a draft that differs from the file.
func (m AppModel) dirtyOf(s config.Saver) bool {
	d, ok := m.drafts[s.Name]
	return ok && d != s.Colours()
}

// anyDirty reports whether any saver has unsaved colours.
func (m AppModel) anyDirty() bool {
	for _, s := range m.cfg.Savers {
		if m.dirtyOf(s) {
			return true
		}
	}
	return false
}

// rows is panel [2] for the current selection.
func (m AppModel) rows() []row {
	value := valueColor
	switch it := m.sideAt(); it.kind {
	case sideSaver:
		s := m.cfg.Savers[it.saver]
		out := []row{
			{kind: rowName, label: "name", value: s.Name, color: value, stop: true},
			{kind: rowType, label: "type", value: s.Type, color: dimColor},
			{kind: rowLayout, label: "layout", value: s.Layout, color: value, stop: true},
			{kind: rowSize, label: "size", value: s.Size, color: value, stop: true},
			{kind: rowFont, label: "font", value: s.Font, color: value, stop: true},
			{kind: rowTime, label: "time", value: s.Time, color: value, stop: true},
			{kind: rowDate, label: "date", value: s.Date, color: value, stop: true},
		}
		saved, draft := s.Colours(), m.draftOf(s)
		for which, c := range []struct {
			label, saved, draft string
		}{{"bg", saved.BG, draft.BG}, {"fg", saved.FG, draft.FG}} {
			sw := row{kind: rowSwatch, label: c.label, value: c.saved, color: dimColor, which: which, hex: c.saved}
			if c.draft != c.saved {
				sw.draft = c.draft
				sw.value += "  →  " + c.draft
			}
			out = append(out, sw)
			r, g, b := config.RGB(c.draft)
			for ch, v := range []int{r, g, b} {
				out = append(out, row{kind: rowChannel, label: "  " + string("RGB"[ch]), value: itoa(v),
					color: value, stop: true, which: which, ch: ch, num: v})
			}
		}
		return out
	default:
		pin := row{kind: rowPIN, label: "PIN", value: "not set", color: yellowColor, stop: true}
		if m.cfg.HasPIN() {
			pin.value, pin.color = "set", liveColor
		}
		active := row{kind: rowSaver, label: "saver", value: m.cfg.Saver, color: value, stop: true}
		if _, ok := m.cfg.Active(); !ok {
			active.value, active.color = m.cfg.Saver+" (missing)", yellowColor
		}
		status := row{kind: rowShowStatus, label: "show_status", value: "off", color: value, stop: true}
		if m.cfg.ShowStatus {
			status.value, status.color = "on", liveColor
		}
		after := itoa(m.cfg.LockoutAfter)
		if m.cfg.LockoutAfter == 0 {
			after = "0 (off)"
		}
		// The files `locku setup` writes: unset is said, in yellow, since
		// setup refuses without them (user, 2026-09-24).
		pathRow := func(kind rowKind, label, p string) row {
			r := row{kind: kind, label: label, value: p, color: value, stop: true}
			if p == "" {
				r.value, r.color = "not set", yellowColor
			}
			return r
		}
		return []row{
			pin,
			active,
			status,
			{kind: rowPromptTimeout, label: "prompt_timeout", value: itoa(m.cfg.PromptTimeout), color: value, stop: true},
			{kind: rowLockoutAfter, label: "lockout_after", value: after, color: value, stop: true},
			{kind: rowLockoutSeconds, label: "lockout_seconds", value: itoa(m.cfg.LockoutSeconds), color: value, stop: true},
			pathRow(rowTmuxConf, "tmux_conf", m.cfg.TmuxConf),
			pathRow(rowScreenConf, "screen_conf", m.cfg.ScreenConf),
		}
	}
}

// stops is the indices of the rows the cursor can rest on.
func (m AppModel) stops() []int {
	var out []int
	for i, r := range m.rows() {
		if r.stop {
			out = append(out, i)
		}
	}
	return out
}

// rowAt is the row under [2]'s cursor.
func (m AppModel) rowAt() row {
	rows, stops := m.rows(), m.stops()
	if len(stops) == 0 {
		return row{}
	}
	return rows[stops[clamp(m.cur2, 0, len(stops)-1)]]
}

// sliderW is the track's width in cells: enough to see where the thumb
// stands, not so much that the bar is the row (webu's).
const sliderW = 12

// sliderBar draws a channel's track with the thumb where its value stands
// in 0–255.
func sliderBar(v int) string {
	at := (clamp(v, 0, 255)*(sliderW-1) + 127) / 255
	return strings.Repeat("─", at) + "●" + strings.Repeat("─", sliderW-1-at)
}

// detailTitle is panel [2]'s chip: the saver's name — with ` · unsaved`
// while its colour draft differs (ui.md §B) — or preference.
func (m AppModel) detailTitle() string {
	it := m.sideAt()
	if it.kind != sideSaver {
		return "[2] preference"
	}
	s := m.cfg.Savers[it.saver]
	if m.dirtyOf(s) {
		return "[2] " + s.Name + " · unsaved"
	}
	return "[2] " + s.Name
}

// detailBody draws panel [2]'s rows at innerW × innerH.
func (m AppModel) detailBody(innerW, innerH int) []string {
	rows, stops := m.rows(), m.stops()
	curRow := -1
	if len(stops) > 0 {
		curRow = stops[clamp(m.cur2, 0, len(stops)-1)]
	}
	cur := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(handColor)
	curOff := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(borderDim)
	dim := lipgloss.NewStyle().Foreground(dimColor)
	txt := lipgloss.NewStyle().Foreground(textColor)
	swatch := func(hex string) string {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render(pixelGlyph)
	}

	// The label column gives way only when the panel is too narrow to hold
	// it and a value; a third of the panel was too little on an ordinary
	// terminal (user, 2026-09-24: lockout_seconds touched its value).
	lw := min(labelW, max(4, innerW-12))
	out := make([]string, 0, len(rows))
	for i, r := range rows {
		label := padRight(" "+r.label, lw)
		var plain, styled string
		switch r.kind {
		case rowSwatch:
			// The saved colour, and the draft after an arrow when it differs.
			plain = padRight(label+"  "+r.value, innerW)
			styled = dim.Render(label) + swatch(r.hex) + " " + dim.Render(r.hex)
			used := lw + 2 + dispW(r.hex)
			if r.draft != "" {
				styled += dim.Render("  →  ") + swatch(r.draft) + " " + dim.Render(r.draft)
				used += 5 + 2 + dispW(r.draft)
			}
			styled = clipANSI(styled, innerW) + spaces(innerW-min(used, innerW))
		case rowChannel:
			// The track and the number wear the channel's own colour at
			// its value — the R slider is #RR0000, the G one #00GG00, the
			// B one #0000BB — so the slider shows what it is setting
			// (user, 2026-09-24).
			// On a ground that runs the other way — white at 0, black at 255
			// — so a dark value is still a visible bar (user, 2026-09-24).
			bar := sliderBar(r.num)
			ch := [3]int{}
			ch[r.ch] = r.num
			tint := lipgloss.NewStyle().
				Foreground(lipgloss.Color(config.Hex(ch[0], ch[1], ch[2]))).
				Background(lipgloss.Color(config.Hex(255-r.num, 255-r.num, 255-r.num)))
			// The ground is the track's alone; the number reads as a value
			// like every other row's, in the value colour, on nothing.
			plain = padRight(label+bar+" "+r.value, innerW)
			styled = txt.Render(label) + tint.Render(bar) + " " +
				lipgloss.NewStyle().Foreground(r.color).Render(r.value) +
				spaces(innerW-lw-sliderW-1-dispW(r.value))
		default:
			plain = padRight(label+r.value, innerW)
			ls := txt
			if !r.stop {
				ls = dim
			}
			styled = ls.Render(label) + lipgloss.NewStyle().Foreground(r.color).Render(padRight(r.value, innerW-lw))
		}
		switch {
		case i == curRow && m.focus == panelDetail:
			out = append(out, cur.Render(plain))
		case i == curRow:
			out = append(out, curOff.Render(plain))
		default:
			out = append(out, styled)
		}
	}
	return fitLines(out, innerW, innerH)
}
