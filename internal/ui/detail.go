package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
)

// Panel [2] (ui.md §1.1): the detail of whatever [1]'s cursor is on, shown
// at once — there is no Enter to open it. A saver is its four fields;
// config is the PIN and the four settings; style is the two colours, each
// a swatch and three channel sliders. Every key config.yaml has is a row
// here, except `saver` and `savers`, which ARE panel [1].

type rowKind int

const (
	rowName rowKind = iota
	rowType
	rowTime
	rowDate
	rowPIN
	rowShowStatus
	rowPromptTimeout
	rowLockoutAfter
	rowLockoutSeconds
	rowSwatch
	rowChannel
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
	hex            string
}

// labelW is the label column; sliders and values start after it. The
// widest label, lockout_seconds, is fifteen, and a value needs a gap.
const labelW = 17

// rows is panel [2] for the current selection.
func (m AppModel) rows() []row {
	value := valueColor
	switch it := m.sideAt(); it.kind {
	case sideSaver:
		s := m.cfg.Savers[it.saver]
		return []row{
			{kind: rowName, label: "name", value: s.Name, color: value, stop: true},
			{kind: rowType, label: "type", value: s.Type, color: dimColor},
			{kind: rowTime, label: "time", value: s.Time, color: value, stop: true},
			{kind: rowDate, label: "date", value: s.Date, color: value, stop: true},
		}
	case sideConfig:
		pin := row{kind: rowPIN, label: "PIN", value: "not set", color: yellowColor, stop: true}
		if m.cfg.HasPIN() {
			pin.value, pin.color = "set", liveColor
		}
		status := row{kind: rowShowStatus, label: "show_status", value: "off", color: value, stop: true}
		if m.cfg.ShowStatus {
			status.value, status.color = "on", liveColor
		}
		after := itoa(m.cfg.LockoutAfter)
		if m.cfg.LockoutAfter == 0 {
			after = "0 (off)"
		}
		return []row{
			pin,
			status,
			{kind: rowPromptTimeout, label: "prompt_timeout", value: itoa(m.cfg.PromptTimeout), color: value, stop: true},
			{kind: rowLockoutAfter, label: "lockout_after", value: after, color: value, stop: true},
			{kind: rowLockoutSeconds, label: "lockout_seconds", value: itoa(m.cfg.LockoutSeconds), color: value, stop: true},
		}
	default:
		var out []row
		for which, c := range []struct {
			label, hex string
		}{{"bg", m.cfg.Style.BG}, {"fg", m.cfg.Style.FG}} {
			r, g, b := config.RGB(c.hex)
			out = append(out, row{kind: rowSwatch, label: c.label, value: c.hex, color: dimColor, which: which, hex: c.hex})
			for ch, v := range []int{r, g, b} {
				out = append(out, row{kind: rowChannel, label: "  " + string("RGB"[ch]), value: itoa(v),
					color: value, stop: true, which: which, ch: ch, num: v})
			}
		}
		return out
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

// detailTitle is panel [2]'s chip: the saver's name, or config, or style.
func (m AppModel) detailTitle() string {
	switch it := m.sideAt(); it.kind {
	case sideSaver:
		return "[2] " + m.cfg.Savers[it.saver].Name
	case sideConfig:
		return "[2] config"
	default:
		return "[2] style"
	}
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

	lw := min(labelW, max(4, innerW/3))
	out := make([]string, 0, len(rows))
	for i, r := range rows {
		label := padRight(" "+r.label, lw)
		var plain, styled string
		switch r.kind {
		case rowSwatch:
			sw := lipgloss.NewStyle().Foreground(lipgloss.Color(r.hex)).Render(pixelCell + pixelGlyph)
			plain = padRight(label+"   "+r.hex, innerW)
			styled = dim.Render(label) + sw + " " + dim.Render(padRight(r.hex, innerW-lw-4))
		case rowChannel:
			bar := sliderBar(r.num)
			plain = padRight(label+bar+" "+r.value, innerW)
			styled = txt.Render(label) + dim.Render(bar) + " " +
				lipgloss.NewStyle().Foreground(r.color).Render(padRight(r.value, innerW-lw-sliderW-1))
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
