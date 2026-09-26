package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpPopup is ? where ? is not the ? menu: on a popup, that popup's own
// help — the keys of that box and nothing of the app's (tdp K6); on [2] of
// preference, or of a tool, the glossary of THAT panel's settings — what
// each row means, wrapped, in place of the note that used to sit under
// each row (user, 2026-09-25; kept there in place of the ? menu,
// 2026-09-26 — a deviation, dev-remarks). On any other panel ? is the ?
// menu (AppModel.openHelp).
type helpPopup struct {
	anim    popupAnimator
	entries []helpEntry
	top     int
	layer   int
	screenW int
	screenH int
}

func newHelpPopup() helpPopup { return helpPopup{anim: newPopupAnimator("help")} }

func (m helpPopup) isActive() bool      { return m.anim.isActive() }
func (m helpPopup) isInteractive() bool { return m.anim.isInteractive() }

// open shows entries: a popup's keys, or one panel's glossary.
func (m *helpPopup) open(layer int, entries []helpEntry) tea.Cmd {
	m.layer, m.top, m.entries = layer, 0, entries
	return m.anim.open()
}
func (m *helpPopup) close() tea.Cmd   { return m.anim.close() }
func (m *helpPopup) setSize(w, h int) { m.screenW, m.screenH = w, h }

// helpEntry is one line: a section header (key == "") or a key/description pair.
type helpEntry struct{ key, desc string }

// keyReference is the ? menu's second region (tdp M4): the core keys and
// the walking keys, to read. Every other key is a row of a Space menu.
var keyReference = []helpEntry{
	{"Tab · 1-2", "next panel / this panel"},
	{"Enter", "[1]: the row's fields, in [2]; [2]: edit, choose, toggle, pick"},
	{"Esc", "close the top popup"},
	{"Space", "what can be done here"},
	{"?", "this menu; on a popup, that popup's keys"},
	{"q · Ctrl-C", "quit; asks first when colours are unsaved"},
	{"j · k", "next / previous row"},
	{"u · d", "half a page"},
	{"gg · G", "first / last"},
}

// menuHelp is ? on the Space menu: that box's keys (tdp K6).
var menuHelp = []helpEntry{
	{"", "Space menu"},
	{"j · k", "next / previous row"},
	{"u · d", "half a page"},
	{"gg · G", "first / last"},
	{"Enter", "run the row; a dimmed one cannot run now"},
	{"[x]", "the letter in a row's brackets runs it"},
	{"Space · Esc", "close"},
}

// optionsHelp is ? on an options list.
var optionsHelp = []helpEntry{
	{"", "Choose one"},
	{"j · k", "next / previous"},
	{"u · d", "half a page"},
	{"gg · G", "first / last"},
	{"Enter", "take this one"},
	{"Esc", "close, nothing changed"},
}

// confirmHelp is ? on a confirm: what Enter does there, and the way back.
func confirmHelp(accept string) []helpEntry {
	return []helpEntry{
		{"", "Confirm"},
		{"Enter", accept},
		{"Esc", "cancel"},
	}
}

// helpPreference is what each of preference's rows means; ? on a [2]
// without a glossary — a profile's, a saver's — is the keys.
var helpPreference = []helpEntry{
	{"", "[2] preference — what each row is"},
	{"PIN", "what the lock asks for; with none, any key unlocks"},
	{"profile", "the profile the lock shows"},
	{"show_status", "user@host and the time, on the lock's last row"},
	{"pin_prompt_timeout", "seconds without a key before the PIN box closes; 0 never"},
	{"wrong_pin_attempts", "wrong PINs in a row before a cooldown; 0 off"},
	{"wrong_pin_attempt_cooldown", "seconds the cooldown lasts"},
}

// helpTool is what each of a tool's rows means, the idle time under the
// tool's own name for it.
func helpTool(name string) []helpEntry {
	out := []helpEntry{
		{"", "[2] " + name + " — what each row is"},
		{"activate", "on: locku's block is in the file, and every row here is written into it the moment it changes; off: it is not. Enter turns it, after a confirm"},
		{"config file path", "the file locku's block is written into; ~/ allowed"},
		{"", "under the line: " + name + "'s own settings"},
	}
	if name == tools[toolTmux] {
		// The lock row is about how much a lock covers — not about what
		// runs it, which is every trigger alike (user, 2026-09-25: a
		// description that named the bind-key misled).
		out = append(out, helpEntry{"lock", "how much a lock covers. lock-server: the whole server — every client, and whoever attaches to any session while it is locked. lock-session: only the session that locked — its clients, and whoever attaches to it; the other sessions go on as they were"})
	}
	out = append(out, helpEntry{toolIdle[name], "idle seconds before " + name + " locks by itself; 0 never"})
	if name == tools[toolTmux] {
		out = append(out, helpEntry{"bind-key", "the key after prefix that runs the lock, as tmux spells it — l, C-l, F12; empty binds none"})
	} else {
		// screen has its own key for the lock, and no server: the lock
		// is LOCKPRG in the shell's environment, so it lives in the
		// shell rc, not the screenrc (measured 2026-09-24).
		out = append(out, helpEntry{"bind", "the key after C-a that runs the lock, as screen's bind spells it — l, ^L; empty binds none, and C-a x — screen's own key for it — locks anyway"})
		out = append(out, helpEntry{"LOCKPRG", "the lock itself, locku by its absolute path: not a row, and not in this file — screen reads it from the environment of the shell that started it, so activate puts it into the shell rc, and a new shell has it; a screen already running takes idle and the key at once, but its lock is the shell's it was attached from until it is detached and attached again from a new shell"})
	}
	return out
}

func (m *helpPopup) update(msg tea.KeyMsg) {
	if !m.anim.isInteractive() {
		return
	}
	_, _, lines := m.layout()
	vis := m.visible(len(lines))
	m.top = moveScroll(m.top, max(0, len(lines)-vis), msg.String(), vis)
}

// visible is how many of n lines fit; the box costs 6 rows of chrome.
func (m helpPopup) visible(n int) int { return max(1, min(n, m.screenH-6)) }

// layout is the key column's width, the box's inner width, and every
// line: a description longer than its column wraps under itself, with
// the key on its first line only (user, 2026-09-25: the glossary must
// wrap, not be cut).
func (m helpPopup) layout() (keyW, innerW int, lines []string) {
	for _, e := range m.entries {
		keyW = max(keyW, dispW(e.key))
	}
	innerW = popupInnerW(m.screenW, keyW+64)
	descW := max(1, innerW-keyW-4)

	dim := lipgloss.NewStyle().Foreground(dimColor)
	key := lipgloss.NewStyle().Foreground(handColor)
	txt := lipgloss.NewStyle().Foreground(textColor)

	for _, e := range m.entries {
		if e.key == "" {
			lines = append(lines, dim.Render(padRight(" "+e.desc, innerW)))
			continue
		}
		k := "  " + e.key
		for _, l := range wrap(e.desc, descW) {
			lines = append(lines, key.Render(padRight(k, keyW+4))+txt.Render(padRight(l, descW)))
			k = ""
		}
	}
	return keyW, innerW, lines
}

func (m helpPopup) view() string {
	_, innerW, lines := m.layout()
	vis := m.visible(len(lines))
	end := min(len(lines), m.top+vis)
	rows := lines[m.top:end]

	pairs := [][2]string{{"Esc", "close"}}
	if len(lines) > vis {
		pairs = append([][2]string{{"j/k", "scroll"}}, pairs...)
	}
	return drawPopupBox(popupLayerColor(m.layer), " "+glyphHelp+" Help ", hintLegend(pairs),
		animRows(m.anim, capRows(rows, m.screenH)), innerW)
}
