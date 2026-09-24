package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// Panel [2] (ui.md §1.1): the detail of whatever [1]'s cursor is on, shown
// at once — there is no Enter to open it. A profile is its fields and its
// two colours, each a swatch and three channel sliders. A saver is what
// it is, which profiles are of it, and then its DEFAULTS — the same
// fields and colours a profile has, which a profile made of it from now
// on starts as, and which [p] previews; changing them changes no profile
// already made (user, 2026-09-24). Preference is the PIN, the active
// profile and the settings. Every key config.yaml has is a row here,
// except `profiles`, which IS panel [1].
//
// Colours edit a DRAFT (user, 2026-09-24): the sliders move a copy, the
// swatch row shows the saved colour and, when it differs, the draft
// beside it, and nothing reaches the file until [S] Save; [R] Reset
// drops the draft. Every other row writes at once.

type rowKind int

const (
	rowNone rowKind = iota // no row: a panel with nothing to stop on
	rowName
	rowSaver // a profile's saver: the class, read-only
	rowLayout
	rowSize
	rowFont
	rowTime
	rowDate
	rowRunner
	rowScene
	rowSwatch
	rowChannel
	rowAbout // a saver's description, read-only
	rowPIN
	rowProfile // preference's active profile
	rowShowStatus
	rowPINPromptTimeout
	rowWrongPINAttempts
	rowWrongPINCooldown
	rowIdleLock
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
// widest label, wrong_pin_attempt_cooldown, is twenty-six, and a value
// wants air.
const labelW = 28

// about is what [2] says of a saver.
var about = map[string]string{
	saver.KindClock: "the time and the date, on the LED board",
	saver.KindDino:  "the offline dino run, jumping by itself, for ever",
}

// draftKey names whose colours a draft holds: a profile's, by name, or
// a saver's defaults, by kind. The two namespaces never meet — a profile
// may well be called clock.
type draftKey struct {
	saver bool
	name  string
}

func profileKey(name string) draftKey { return draftKey{name: name} }
func saverKey(kind string) draftKey   { return draftKey{saver: true, name: kind} }

// subject is what [2] edits under the cursor — a profile, or a saver's
// defaults — with its draft key; ok is false on preference.
func (m AppModel) subject() (p config.Profile, key draftKey, ok bool) {
	switch it := m.sideAt(); it.kind {
	case sideProfile:
		p = m.cfg.Profiles[it.ref]
		return p, profileKey(p.Name), true
	case sideSaver:
		kind := saver.Kinds[it.ref]
		return m.cfg.Saver(kind), saverKey(kind), true
	}
	return config.Profile{}, draftKey{}, false
}

// draftOf is p's colours as the sliders have them: the draft under key
// when one is up, p's own otherwise.
func (m AppModel) draftOf(key draftKey, p config.Profile) config.Style {
	if d, ok := m.drafts[key]; ok {
		return d
	}
	return p.Colours()
}

// dirtyOf reports whether the draft under key differs from p's colours.
func (m AppModel) dirtyOf(key draftKey, p config.Profile) bool {
	d, ok := m.drafts[key]
	return ok && d != p.Colours()
}

// anyDirty reports whether any profile or saver has unsaved colours.
func (m AppModel) anyDirty() bool {
	for _, p := range m.cfg.Profiles {
		if m.dirtyOf(profileKey(p.Name), p) {
			return true
		}
	}
	for _, k := range saver.Kinds {
		if m.dirtyOf(saverKey(k), m.cfg.Saver(k)) {
			return true
		}
	}
	return false
}

// fieldRows is a saver's own settings for p: the clock's shapes and
// size, or the run's runner and scene.
func fieldRows(p config.Profile) []row {
	value := valueColor
	if p.Saver == saver.KindDino {
		// No size: the run is drawn as large as the terminal allows
		// (user, 2026-09-24).
		return []row{
			{kind: rowRunner, label: "runner", value: p.Runner, color: value, stop: true},
			{kind: rowScene, label: "scene", value: p.Scene, color: value, stop: true},
		}
	}
	return []row{
		{kind: rowLayout, label: "layout", value: p.Layout, color: value, stop: true},
		{kind: rowSize, label: "size", value: p.Size, color: value, stop: true},
		{kind: rowFont, label: "font", value: p.Font, color: value, stop: true},
		{kind: rowTime, label: "time", value: p.Time, color: value, stop: true},
		{kind: rowDate, label: "date", value: p.Date, color: value, stop: true},
	}
}

// colourRows is the two colours of p: each a swatch row — the saved
// colour, and the draft under key when it differs — and three channels.
func (m AppModel) colourRows(key draftKey, p config.Profile) []row {
	var out []row
	saved, draft := p.Colours(), m.draftOf(key, p)
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
				color: valueColor, stop: true, which: which, ch: ch, num: v})
		}
	}
	return out
}

// rows is panel [2] for the current selection.
func (m AppModel) rows() []row {
	value := valueColor
	switch it := m.sideAt(); it.kind {
	case sideSaver:
		kind := saver.Kinds[it.ref]
		var names []string
		for _, p := range m.cfg.Profiles {
			if p.Saver == kind {
				names = append(names, p.Name)
			}
		}
		used := strings.Join(names, ", ")
		if used == "" {
			used = "none yet"
		}
		p, key, _ := m.subject()
		out := []row{
			{kind: rowAbout, label: "saver", value: kind, color: value},
			{kind: rowAbout, label: "what", value: about[kind], color: value},
			{kind: rowAbout, label: "profiles", value: used, color: value},
			{kind: rowAbout, label: "defaults", value: "for profiles made of it from now on", color: dimColor},
		}
		out = append(out, fieldRows(p)...)
		return append(out, m.colourRows(key, p)...)
	case sideProfile:
		p, key, _ := m.subject()
		// The saver is the profile's class: shown, not changed — a profile
		// of another saver is a new profile (user, 2026-09-24). The rows
		// under it are the saver's own.
		out := []row{
			{kind: rowName, label: "name", value: p.Name, color: value, stop: true},
			{kind: rowSaver, label: "saver", value: p.Saver, color: dimColor},
		}
		out = append(out, fieldRows(p)...)
		return append(out, m.colourRows(key, p)...)
	default:
		pin := row{kind: rowPIN, label: "PIN", value: "not set", color: yellowColor, stop: true}
		if m.cfg.HasPIN() {
			pin.value, pin.color = "set", liveColor
		}
		active := row{kind: rowProfile, label: "profile", value: m.cfg.Profile, color: value, stop: true}
		if _, ok := m.cfg.Active(); !ok {
			active.value, active.color = m.cfg.Profile+" (missing)", yellowColor
		}
		status := row{kind: rowShowStatus, label: "show_status", value: "off", color: value, stop: true}
		if m.cfg.ShowStatus {
			status.value, status.color = "on", liveColor
		}
		after := itoa(m.cfg.WrongPINAttempts)
		if m.cfg.WrongPINAttempts == 0 {
			after = "0 (off)"
		}
		idle := itoa(m.cfg.IdleLock)
		if m.cfg.IdleLock == 0 {
			idle = "0 (off)"
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
			{kind: rowPINPromptTimeout, label: "pin_prompt_timeout", value: itoa(m.cfg.PINPromptTimeout), color: value, stop: true},
			{kind: rowWrongPINAttempts, label: "wrong_pin_attempts", value: after, color: value, stop: true},
			{kind: rowWrongPINCooldown, label: "wrong_pin_attempt_cooldown", value: itoa(m.cfg.WrongPINCooldown), color: value, stop: true},
			{kind: rowIdleLock, label: "idle_lock", value: idle, color: value, stop: true},
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

// rowAt is the row under [2]'s cursor, or a rowNone row when the panel
// has no stops.
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

// detailTitle is panel [2]'s chip: a saver's name with ` · saver`, a
// profile's name, or preference — with ` · unsaved` while a colour
// draft differs (ui.md §B).
func (m AppModel) detailTitle() string {
	switch it := m.sideAt(); it.kind {
	case sideSaver, sideProfile:
		p, key, _ := m.subject()
		title := "[2] " + p.Name
		if it.kind == sideSaver {
			title = "[2] " + saver.Kinds[it.ref] + " · saver"
		}
		if m.dirtyOf(key, p) {
			title += " · unsaved"
		}
		return title
	}
	return "[2] preference"
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
	// terminal (user, 2026-09-24: the widest label touched its value).
	// A slider row needs the track, a space and three digits after the
	// label, so that is the least the label column leaves.
	lw := min(labelW, max(4, innerW-sliderW-5))
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
			styled = ls.Render(label) + lipgloss.NewStyle().Foreground(r.color).Render(padRight(truncate(r.value, innerW-lw), innerW-lw))
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
