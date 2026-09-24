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
// One part of it is contextual after all: the glossary — what each setting
// of preference, or of a tool, means. It comes at the end, and only on the
// panel it is about (user, 2026-09-25: on preference, only preference's).
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

// open shows the keys, then the glossary of where the cursor is, if it has one.
func (m *helpPopup) open(layer int, here sideKind) tea.Cmd {
	m.layer, m.top = layer, 0
	m.entries = helpKeys
	if g := helpGlossary[here]; g != nil {
		m.entries = append(append([]helpEntry{}, helpKeys...), g...)
	}
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
	{"?", "this help"},
	{"", "Global"},
	{"P", "preview: the lock with the active profile — on a profile's [2], that profile — drafts included; unlock to come back"},
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
	{"", "[1] Integration — tmux, screen"},
	{"S", "setup: locku's block into the tool's file (tmux: and onto a running server)"},
	{"X", "remove: the block out again"},
	{"", "Navigate"},
	{"j · k", "next / previous row"},
	{"u · d", "half a page"},
	{"gg · G", "first / last"},
}

// helpGlossary is what each setting means, by the sidebar item whose [2]
// shows it; the items without one have none.
var helpGlossary = map[sideKind][]helpEntry{
	sideTool: {
		{"", "[2] tmux / screen — what each is"},
		{"conf", "the file the block goes into; ~/ allowed"},
		{"idle_lock", "idle seconds before the tool locks by itself; 0 never — setup again after a change"},
		{"status", "whether the block is in the file now"},
	},
	sidePreference: {
		{"", "[2] preference — what each is"},
		{"PIN", "what the lock asks for; with none, any key unlocks"},
		{"profile", "the profile the lock shows"},
		{"show_status", "user@host and the time, on the lock's last row"},
		{"pin_prompt_timeout", "seconds without a key before the PIN box closes; 0 never"},
		{"wrong_pin_attempts", "wrong PINs in a row before a cooldown; 0 off"},
		{"wrong_pin_attempt_cooldown", "seconds the cooldown lasts"},
	},
}

func (m *helpPopup) update(msg tea.KeyMsg) {
	if !m.anim.isInteractive() {
		return
	}
	m.top = moveScroll(m.top, max(0, len(m.entries)-m.visible()), msg.String(), m.visible())
}

// visible is how many content lines fit; the box costs 6 rows of chrome.
func (m helpPopup) visible() int { return max(1, min(len(m.entries), m.screenH-6)) }

func (m helpPopup) view() string {
	keyW := 0
	for _, e := range m.entries {
		keyW = max(keyW, dispW(e.key))
	}
	innerW := popupInnerW(m.screenW, keyW+64)

	dim := lipgloss.NewStyle().Foreground(dimColor)
	key := lipgloss.NewStyle().Foreground(handColor)
	txt := lipgloss.NewStyle().Foreground(textColor)

	vis := m.visible()
	end := min(len(m.entries), m.top+vis)
	rows := make([]string, 0, vis)
	for _, e := range m.entries[m.top:end] {
		if e.key == "" {
			rows = append(rows, dim.Render(padRight(" "+e.desc, innerW)))
			continue
		}
		rows = append(rows, key.Render(padRight("  "+e.key, keyW+4))+
			txt.Render(padRight(e.desc, innerW-keyW-4)))
	}

	pairs := [][2]string{{"Esc", "close"}}
	if len(m.entries) > vis {
		pairs = append([][2]string{{"j/k", "scroll"}}, pairs...)
	}
	return drawPopupBox(popupLayerColor(m.layer), " "+glyphHelp+" Help ", hintLegend(pairs),
		animRows(m.anim, capRows(rows, m.screenH)), innerW)
}
