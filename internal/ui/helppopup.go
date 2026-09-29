package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpPopup is ?, the key reference of whatever is in front, to read and
// scroll, not to run (tdp K6, M4): on a popup that popup's own keys and
// nothing of the app's; on a panel its keys over the core keys
// (AppModel.panelKeys); on [2] of preference, or of a tool, the glossary of
// THAT panel's settings instead — what each row means, wrapped, in place of
// the note that used to sit under each row (user, 2026-09-25; kept there in
// place of the key reference, 2026-09-26 and 2026-09-27 — a deviation,
// dev-remarks).
type helpPopup struct {
	anim    popupAnimator
	entries []helpEntry
	top     int
	layer   int
	screenW int
	screenH int
	// glossary: the left column is the settings' names, not keys, so it
	// is not drawn as keys are (openGlossary).
	glossary bool
}

func newHelpPopup() helpPopup { return helpPopup{anim: newPopupAnimator("help")} }

func (m helpPopup) isActive() bool      { return m.anim.isActive() }
func (m helpPopup) isInteractive() bool { return m.anim.isInteractive() }

// open shows a key reference: a popup's keys, or a panel's.
func (m *helpPopup) open(layer int, entries []helpEntry) tea.Cmd {
	m.layer, m.top, m.entries, m.glossary = layer, 0, entries, false
	return m.anim.open()
}

// openGlossary shows one panel's glossary: what each of its rows means.
func (m *helpPopup) openGlossary(layer int, entries []helpEntry) tea.Cmd {
	cmd := m.open(layer, entries)
	m.glossary = true
	return cmd
}

func (m *helpPopup) close() tea.Cmd   { return m.anim.close() }
func (m *helpPopup) setSize(w, h int) { m.screenW, m.screenH = w, h }

// helpEntry is one line: a section header (key == "") or a key/description pair.
type helpEntry struct {
	key, desc string
	// disabled: what the key acts on is here, but it cannot run now —
	// listed, dimmed, as its Space menu row is (tdp M6, 2026-09-29).
	disabled bool
}

// keyReference is the foot of a panel's key reference (tdp M4): the core
// keys and the walking keys, which work on every panel. The panel's own
// keys come first, off its actions. Keys that do one thing are joined with
// /, a range with – (tdp M5, 2026-09-29; with " · " and "-" until then).
var keyReference = []helpEntry{
	{key: "Tab/1–2", desc: "next panel / this panel"},
	{key: "Enter", desc: "[1]: the row's fields, in [2]; [2]: edit, choose, toggle, pick"},
	{key: "Esc", desc: "close the top popup"},
	{key: "Space", desc: "what can be done here"},
	{key: "?", desc: "the keys here; on a popup, that popup's keys"},
	{key: "q/Ctrl-C", desc: "quit; asks first when colours are unsaved"},
	{key: "j/k", desc: "next / previous row"},
	{key: "u/d", desc: "half a page"},
	{key: "gg/G", desc: "first / last"},
}

// menuHelp is ? on the Space menu: that box's keys (tdp K6).
var menuHelp = []helpEntry{
	{desc: "Space menu"},
	{key: "j/k", desc: "next / previous row"},
	{key: "u/d", desc: "half a page"},
	{key: "gg/G", desc: "first / last"},
	{key: "Enter", desc: "run the row; a dimmed one cannot run now"},
	{key: "[x]", desc: "the letter in a row's brackets runs it"},
	{key: "Space/Esc", desc: "close"},
}

// globalMenuHelp is ? on the global operation popup: that box's keys.
var globalMenuHelp = []helpEntry{
	{desc: "Global operation"},
	{key: "j/k", desc: "next / previous row"},
	{key: "Enter", desc: "run the row"},
	{key: "[x]", desc: "the letter in a row's brackets runs it"},
	{key: "Esc", desc: "back to the Space menu"},
}

// optionsHelp is ? on an options list.
var optionsHelp = []helpEntry{
	{desc: "Choose one"},
	{key: "j/k", desc: "next / previous"},
	{key: "u/d", desc: "half a page"},
	{key: "gg/G", desc: "first / last"},
	{key: "Enter", desc: "take this one"},
	{key: "Esc", desc: "close, nothing changed"},
}

// confirmHelp is ? on a confirm: what Enter does there, and the way back.
func confirmHelp(accept string) []helpEntry {
	return []helpEntry{
		{desc: "Confirm"},
		{key: "Enter", desc: accept},
		{key: "Esc", desc: "cancel"},
	}
}

// helpPreference is what each of preference's rows means; ? on a [2]
// without a glossary — a profile's, a saver's — is the keys.
var helpPreference = []helpEntry{
	{desc: "[2] preference — what each row is"},
	{key: "PIN", desc: "what the lock asks for; with none, any key unlocks"},
	{key: "profile", desc: "the profile the lock shows"},
	{key: "show_status", desc: "user@host and the time, on the lock's last row"},
	{key: "pin_prompt_timeout", desc: "seconds without a key before the PIN box closes; 0 never"},
	{key: "wrong_pin_attempts", desc: "wrong PINs in a row before a cooldown; 0 off"},
	{key: "wrong_pin_attempt_cooldown", desc: "seconds the cooldown lasts"},
}

// helpTool is what each of a tool's rows means, the idle time under the
// tool's own name for it.
func helpTool(name string) []helpEntry {
	out := []helpEntry{
		{desc: "[2] " + name + " — what each row is"},
		{key: "activate", desc: "on: locku's block is in the file, and every row here is written into it the moment it changes; off: it is not. [Enter] turns it, after a confirm"},
		{key: "config file path", desc: "the file locku's block is written into; ~/ allowed"},
		{desc: "under the line: " + name + "'s own settings"},
	}
	if name == tools[toolTmux] {
		// The lock row is about how much a lock covers — not about what
		// runs it, which is every trigger alike (user, 2026-09-25: a
		// description that named the bind-key misled).
		out = append(out, helpEntry{key: "lock", desc: "how much a lock covers. lock-server: the whole server — every client, and whoever attaches to any session while it is locked. lock-session: only the session that locked — its clients, and whoever attaches to it; the other sessions go on as they were"})
	}
	out = append(out, helpEntry{key: toolIdle[name], desc: "idle seconds before " + name + " locks by itself; 0 never"})
	if name == tools[toolTmux] {
		out = append(out, helpEntry{key: "bind-key", desc: "the key after prefix that runs the lock, as tmux spells it — l, C-l, F12; empty binds none"})
	} else {
		// screen has its own key for the lock, and no server: the lock
		// is LOCKPRG in the shell's environment, so it lives in the
		// shell rc, not the screenrc (measured 2026-09-24).
		out = append(out, helpEntry{key: "bind", desc: "the key after C-a that runs the lock, as screen's bind spells it — l, ^L; empty binds none, and C-a x — screen's own key for it — locks anyway"})
		out = append(out, helpEntry{key: "LOCKPRG", desc: "the lock itself, locku by its absolute path: not a row, and not in this file — screen reads it from the environment of the shell that started it, so activate puts it into the shell rc, and a new shell has it; a screen already running takes idle and the key at once, but its lock is the shell's it was attached from until it is detached and attached again from a new shell"})
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
// line. Keys are Blue and what they do Text (tdp D2, 2026-09-29; keys
// were Subtext1 until then, which a glossary's names still are); a key
// that cannot run now is dimmed, key and words, as its Space menu row is
// (tdp M6, 2026-09-29; as bright as the rest until then). The box
// is as wide as every popup (tdp F7; as wide as its longest
// description until 2026-09-28, the old D4); a description longer than
// its column wraps under itself, with the key on its first line only
// (user, 2026-09-25: the glossary must wrap, not be cut), a column clear
// of the border.
func (m helpPopup) layout() (keyW, innerW int, lines []string) {
	for _, e := range m.entries {
		keyW = max(keyW, dispW(e.key))
	}
	innerW = popupInnerW(m.screenW)
	descW := max(1, innerW-keyW-5)

	dim := lipgloss.NewStyle().Foreground(dimColor)
	key := lipgloss.NewStyle().Foreground(focusColor)
	if m.glossary {
		key = key.Foreground(handColor)
	}
	txt := lipgloss.NewStyle().Foreground(textColor)

	for _, e := range m.entries {
		if e.key == "" {
			lines = append(lines, dim.Render(padRight(" "+e.desc, innerW)))
			continue
		}
		ks, ds := key, txt
		if e.disabled {
			ks, ds = dim, dim
		}
		k := "  " + e.key
		for _, l := range wrap(e.desc, descW) {
			lines = append(lines, ks.Render(padRight(k, keyW+4))+ds.Render(padRight(l, descW)))
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
	return drawPopupBox(popupLayerColor(m.layer), " "+glyphHelp+" Help ", pairs,
		animRows(m.anim, capRows(rows, m.screenH)), innerW)
}
