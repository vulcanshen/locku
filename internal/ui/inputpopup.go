package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// inputAction says what to do with the answer. It exists so the popup itself
// stays a text box and nothing else — the same reason confirmPopup carries an
// action rather than a closure.
type inputAction int

const (
	inputNone       inputAction = iota
	inputRename                 // a saver's new name
	inputDuplicate              // a copy's name
	inputNumber                 // prompt_timeout / lockout_after / lockout_seconds
	inputPath                   // tmux_conf / screen_conf, on an offer
	inputPINCurrent             // the PIN in force, before a change or a clear
	inputPINNew                 // the new PIN
	inputPINConfirm             // the new PIN again
)

// inputPopup is one line of text (ui.md §3.1). The BORDER says what kind of
// box this is — `name`, `number`, `new PIN` — and a suffix says how the
// box feels about what is in it: ` · taken`, ` · invalid`, ` · wrong`. The
// row inside is the field's name and the value being typed.
//
// While it is up every printable key is a character: Space is a space and
// ? is a question mark (§4.5).
type inputPopup struct {
	anim   popupAnimator
	title  string // the type: "name", "number", "current PIN", …
	suffix string // " · taken", " · invalid", …; cleared by the next key
	prompt string // the field
	value  string
	action inputAction
	accept string // the verb on the Enter hint
	masked bool   // a PIN: dots, never the characters
	// placeholder is shown dim in the empty box: an offer Tab takes and
	// Backspace declines (webu's settings box; ux.md §2.1). An offer
	// nobody took is not a value: the caller sees "" and does nothing.
	placeholder string
	// frozen swallows every key: the second after a wrong current PIN
	// (ux.md §2.2), the same beat the lock screen keeps.
	frozen    bool
	frozenGen int
	layer     int
	screenW   int
	screenH   int
}

// inputThawMsg ends the frozen second.
type inputThawMsg struct{ gen int }

const inputFreeze = time.Second

func newInputPopup() inputPopup { return inputPopup{anim: newPopupAnimator("input")} }

func (m inputPopup) isActive() bool      { return m.anim.isActive() }
func (m inputPopup) isInteractive() bool { return m.anim.isInteractive() }
func (m *inputPopup) close() tea.Cmd     { return m.anim.close() }
func (m *inputPopup) setSize(w, h int)   { m.screenW, m.screenH = w, h }

// ask opens the box as p describes it, the value pre-filled and the
// cursor at its end. Pre-filling matters for an edit: most edits change
// part of a value, and starting from empty makes the common case retype
// the whole thing.
func (m *inputPopup) ask(p inputPopup, layer int) tea.Cmd {
	p.anim, p.layer = m.anim, layer
	p.screenW, p.screenH, p.frozenGen = m.screenW, m.screenH, m.frozenGen
	*m = p
	return m.anim.open()
}

// freeze puts the box on hold for a second with the given verdict in its
// border, the value cleared, every key swallowed (ux.md §2.2).
func (m *inputPopup) freeze(suffix string) tea.Cmd {
	m.suffix, m.value, m.frozen = suffix, "", true
	m.frozenGen++
	gen := m.frozenGen
	return tea.Tick(inputFreeze, func(time.Time) tea.Msg { return inputThawMsg{gen} })
}

func (m *inputPopup) thaw(msg inputThawMsg) {
	if msg.gen == m.frozenGen {
		m.frozen, m.suffix = false, ""
	}
}

// update edits the line; Enter and Esc are resolved by the caller, since
// commit and cancel are each one role app-wide (§4.3).
func (m *inputPopup) update(msg tea.KeyMsg) {
	if !m.anim.isInteractive() || m.frozen {
		return
	}
	m.suffix = ""
	switch msg.Type {
	case tea.KeyTab:
		if m.value == "" && m.placeholder != "" {
			m.value, m.placeholder = m.placeholder, ""
		}
	case tea.KeyBackspace:
		if r := []rune(m.value); len(r) > 0 {
			m.value = string(r[:len(r)-1])
		} else {
			m.placeholder = ""
		}
	case tea.KeyCtrlU:
		m.value = ""
	case tea.KeySpace:
		m.value += " "
	case tea.KeyRunes:
		m.value += string(msg.Runes)
	}
}

func (m inputPopup) view() string {
	innerW := popupInnerW(m.screenW, max(40, dispW(m.value)+8, dispW(m.placeholder)+8, dispW(m.prompt)+3))
	dim := lipgloss.NewStyle().Foreground(dimColor)
	edit := lipgloss.NewStyle().Foreground(editColor)
	cur := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(editColor)

	shown := m.value
	if m.masked {
		shown = strings.Repeat("●", len([]rune(m.value)))
	}
	value := truncateHead(shown, innerW-3)
	line := " " + edit.Render(value) + cur.Render(" ") + spaces(max(0, innerW-2-dispW(value)))
	offered := m.value == "" && m.placeholder != ""
	if offered {
		// The offer, dim, after the cursor: not typed, so not lavender.
		ph := truncate(m.placeholder, innerW-3)
		line = " " + cur.Render(" ") + dim.Render(ph) + spaces(max(0, innerW-2-dispW(ph)))
	}
	rows := []string{
		dim.Render(padRight(" "+m.prompt, innerW)),
		spaces(innerW),
		line,
	}
	bc := popupLayerColor(m.layer)
	if m.suffix != "" {
		bc = warnColor
	}
	pairs := [][2]string{{"Enter", m.accept}}
	if offered {
		pairs = append(pairs, [2]string{"Tab", "edit it"}, [2]string{"Bksp", "clear"})
	}
	hint := hintLegend(append(pairs, [2]string{"Esc", "cancel"}))
	if m.frozen {
		hint = ""
	}
	return drawPopupBox(bc, " "+glyphInput+" "+m.title+m.suffix+" ", hint,
		animRows(m.anim, rows), innerW)
}
