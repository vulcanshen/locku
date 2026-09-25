package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpPopup is the §A.2 non-contextual entry point: every global action the
// app has, reachable from any surface and on top of any float. Its
// completeness is the promise — a user who never read a README finds the
// whole vocabulary here.
//
// With one exception, the user's (2026-09-25): on [2] of preference, or of
// a tool, ? is the glossary of THAT panel's settings and nothing else —
// what each row means, wrapped, in place of the note that used to sit
// under each row. The keys are a panel away, on [1].
type helpPopup struct {
	anim    popupAnimator
	entries []helpEntry
	top     int
	layer   int
	screenW int
	screenH int
}

func newHelpPopup() helpPopup { return helpPopup{anim: newPopupAnimator("help"), entries: helpKeys} }

func (m helpPopup) isActive() bool      { return m.anim.isActive() }
func (m helpPopup) isInteractive() bool { return m.anim.isInteractive() }

// open shows entries: the keys, or one panel's glossary (AppModel.helpEntries).
func (m *helpPopup) open(layer int, entries []helpEntry) tea.Cmd {
	m.layer, m.top, m.entries = layer, 0, entries
	return m.anim.open()
}
func (m *helpPopup) close() tea.Cmd   { return m.anim.close() }
func (m *helpPopup) setSize(w, h int) { m.screenW, m.screenH = w, h }

// helpEntry is one line: a section header (key == "") or a key/description pair.
type helpEntry struct{ key, desc string }

// helpKeys is the whole vocabulary (ux.md §A.2 and the appendix). The
// core keys come first because they are the five a user has to hold to
// walk the app (§A.0.K).
var helpKeys = []helpEntry{
	{"", "Core keys"},
	{"Tab · 1-2", "next panel / this panel"},
	{"Enter", "[1]: the row's fields, in [2]; [2]: edit, choose, toggle, pick"},
	{"Esc", "close the top float"},
	{"Space", "what can I do here: the item, and the panel"},
	{"?", "this help — on [2] of preference, or of tmux / screen, what each row means"},
	{"", "Global"},
	{"P", "preview: the lock with the active profile — on a profile's [2], that profile — drafts included; any key comes back"},
	{"q", "quit — asks first when colours are unsaved"},
	{"Ctrl+C", "force quit"},
	{"", "[1] Savers — the kinds: clock, dino"},
	{"n", "new profile of this saver, under a name"},
	{"", "[1] Profiles — the ones set up"},
	{"p", "preview the lock showing this profile"},
	{"D", "duplicate it under a new name"},
	{"r", "rename it"},
	{"X", "delete it (not the active one, not the last one)"},
	{"", "[2] a profile, or a saver's defaults"},
	{"P", "preview the lock showing this profile"},
	{"S", "save its colour draft to config.yaml"},
	{"R", "reset the draft to the saved colours"},
	{"", "[2] tmux, screen"},
	{"Enter", "on activate: on writes locku's block into the file (tmux: and onto a running server), after a confirm — and from then on a row changed is written at once; off takes it out"},
	{"", "Navigate"},
	{"j · k", "next / previous row"},
	{"u · d", "half a page"},
	{"gg · G", "first / last"},
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
		{toolIdle[name], "idle seconds before " + name + " locks by itself; 0 never"},
	}
	if name == tools[toolTmux] {
		out = append(out, helpEntry{"bind-key", "the key after prefix that locks every client, as tmux spells it — l, C-l, F12; empty binds none"})
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
