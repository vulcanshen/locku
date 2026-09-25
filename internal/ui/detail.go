package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
	"github.com/vulcanshen/locku/internal/setup"
)

// Panel [2] (ui.md §1.1): the detail of whatever [1]'s cursor is on, shown
// at once — there is no Enter to open it. A profile is its fields and its
// two colours, each a swatch and three channel sliders. A saver is what
// it is, which profiles are of it, and then its DEFAULTS — the same
// fields and colours a profile has, which a profile made of it from now
// on starts as, and which [p] previews; changing them changes no profile
// already made (user, 2026-09-24). A tool — tmux, screen — is its file
// and its idle time — tmux's the key after prefix that locks, too —
// under locku's own two rows: activate, on while locku's block is in
// the file and off while it is not, and the file's path; a rule parts
// the two from the tool's own keys (user, 2026-09-25: property and
// value throughout — the button, and the S and X hotkeys before it,
// are gone). Preference is the PIN, the active profile and the lock's
// settings. What each setting means is in the ? help, on that panel.
// Every key config.yaml has is a row here, except `profiles` and
// `savers`, which ARE panel [1]; a tool's idle time under the tool's
// own name for it. The first row of every [2] is the table's header,
// Property and Value (user, 2026-09-25).
//
// Colours edit a DRAFT (user, 2026-09-24): the sliders move a copy, the
// swatch row shows the saved colour and, when it differs, the draft
// beside it, and nothing reaches the file until [S] Save; [R] Reset
// drops the draft. Every other row writes at once.

type rowKind int

const (
	rowNone rowKind = iota // no row: a panel with nothing to stop on
	rowHead                // the table's header: Property, Value
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
	rowConf     // a tool's file
	rowIdle     // a tool's idle time, under the tool's own name for it
	rowBindKey  // tmux's key after prefix that locks, or none
	rowActivate // locku's block in the tool's file: on, or off
	rowRule     // the line between locku's rows and the tool's own
	rowLock     // tmux's lock: lock-server, or lock-session
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

// usualConf is where a tool's file usually is: the offer in the conf
// box when nothing is set.
var usualConf = map[string]string{"tmux": "~/.tmux.conf", "screen": "~/.screenrc"}

// toolIdle is each tool's own name for its idle time: the row's label,
// and the file's key (user, 2026-09-25: tmux's row says lock-after-time,
// as tmux does).
var toolIdle = map[string]string{"tmux": "lock-after-time", "screen": "idle"}

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
// defaults — with its draft key; ok is false elsewhere.
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

// tool is the tool under the cursor — tmux or screen — and its settings.
func (m AppModel) tool() (name string, t config.Tool) {
	name = tools[m.sideAt().ref]
	return name, m.cfg.Tool(name)
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

// offOr is n, or "0 (off)" for none.
func offOr(n int) string {
	if n == 0 {
		return "0 (off)"
	}
	return itoa(n)
}

// rows is panel [2] for the current selection.
func (m AppModel) rows() []row {
	value := valueColor
	head := row{kind: rowHead, label: "Property", value: "Value"}
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
			head,
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
			head,
			{kind: rowName, label: "name", value: p.Name, color: value, stop: true},
			{kind: rowSaver, label: "saver", value: p.Saver, color: dimColor},
		}
		out = append(out, fieldRows(p)...)
		return append(out, m.colourRows(key, p)...)
	case sideTool:
		name, t := m.tool()
		// locku's own two rows first — whether the block is in the file,
		// read off the file each time, and the file — then a rule, then
		// the tool's own keys under the tool's own names (user,
		// 2026-09-25: property and value throughout, the rule parting
		// what is locku's from what is the tool's).
		active := row{kind: rowActivate, label: "activate", value: "off", color: value, stop: true}
		if setup.Installed(t.Conf) {
			active.value, active.color = "on", liveColor
		}
		conf := row{kind: rowConf, label: "config file path", value: t.Conf, color: value, stop: true}
		if t.Conf == "" {
			conf.value, conf.color = "not set", yellowColor
		}
		out := []row{
			head,
			active,
			conf,
			{kind: rowRule},
		}
		// tmux alone chooses its lock: the server's, or this session's
		// (user, 2026-09-25).
		if name == tools[toolTmux] {
			out = append(out, row{kind: rowLock, label: "lock", value: m.cfg.Tmux.Lock, color: value, stop: true})
		}
		out = append(out, row{kind: rowIdle, label: toolIdle[name], value: offOr(t.Idle), color: value, stop: true})
		// tmux alone binds a key: the one after prefix that locks every
		// client, as tmux spells it; none is none (user, 2026-09-25).
		if name == tools[toolTmux] {
			bind := row{kind: rowBindKey, label: "bind-key", value: m.cfg.Tmux.BindKey, color: value, stop: true}
			if bind.value == "" {
				bind.value = "none"
			}
			out = append(out, bind)
		}
		return out
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
		return []row{
			head,
			pin,
			active,
			status,
			{kind: rowPINPromptTimeout, label: "pin_prompt_timeout", value: itoa(m.cfg.PINPromptTimeout), color: value, stop: true},
			{kind: rowWrongPINAttempts, label: "wrong_pin_attempts", value: offOr(m.cfg.WrongPINAttempts), color: value, stop: true},
			{kind: rowWrongPINCooldown, label: "wrong_pin_attempt_cooldown", value: itoa(m.cfg.WrongPINCooldown), color: value, stop: true},
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

// sideChips is panel [1]'s title: one chip, in the border's colour.
var sideChips = []chip{{text: "[1] locku", border: true}}

// detailChips is panel [2]'s title as a chain (ui.md §5, 2026-09-25):
// the panel and what it shows, in the border's colour, and, while a
// colour draft differs, unsaved — lit yellow while the panel has the
// keys (chipFill). What kind of thing it shows — profile, saver,
// integration, settings — is nowhere: it was a link of this chain,
// then a capsule of its own in the border's other corner, and the user
// found it told them nothing the sidebar's cursor did not (2026-09-25).
func (m AppModel) detailChips() []chip {
	switch it := m.sideAt(); it.kind {
	case sideSaver, sideProfile:
		p, key, _ := m.subject()
		name := p.Name
		if it.kind == sideSaver {
			name = saver.Kinds[it.ref]
		}
		out := []chip{{text: "[2] " + name, border: true}}
		if m.dirtyOf(key, p) {
			out = append(out, chip{text: "unsaved", fill: yellowColor})
		}
		return out
	case sideTool:
		name, _ := m.tool()
		return []chip{{text: "[2] " + name, border: true}}
	}
	return []chip{{text: "[2] preference", border: true}}
}

// detailTitle is the same, in words, for the Space menu's own box.
func (m AppModel) detailTitle() string {
	var parts []string
	for _, c := range m.detailChips() {
		parts = append(parts, c.text)
	}
	return strings.Join(parts, " · ")
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
	head := lipgloss.NewStyle().Foreground(focusColor)
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
	keep := 0
	for i, r := range rows {
		label := padRight(" "+r.label, lw)
		var plain, styled string
		switch r.kind {
		case rowRule:
			// The line between locku's rows and the tool's own.
			plain = strings.Repeat("─", innerW)
			styled = lipgloss.NewStyle().Foreground(borderDim).Render(plain)
		case rowHead:
			// The table's header, in the colour of the sidebar's group
			// titles, so both panels are read the same way.
			plain = padRight(label+r.value, innerW)
			styled = head.Render(plain)
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
		if i == curRow {
			keep = len(out) - 1
		}
	}
	// The lines follow the cursor when they outgrow the panel.
	top := scrollTo(0, min(len(out)-1, keep), innerH)
	return fitLines(out[min(top, len(out)):], innerW, innerH)
}
