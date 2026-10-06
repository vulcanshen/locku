package ui

import (
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
	inputNew                    // a new profile's name, of the saver at editRef
	inputRename                 // a profile's new name
	inputDuplicate              // a copy's name
	inputNumber                 // pin_prompt_timeout, wrong_pin_attempts, wrong_pin_attempt_cooldown, a tool's idle time
	inputPath                   // tmux_conf / screen_conf, on an offer
	inputBindKey                // tmux's bind-key: one key, or nothing
	inputCommand                // a custom saver's command
	inputPINCurrent             // the PIN in force, before a change or a clear
	inputPINNew                 // the new PIN
	inputPINConfirm             // the new PIN again
)

// inputPopup is one line of text (ui.md §3.1). The BORDER says what kind of
// box this is — `name`, `number`, `new PIN`. Inside are the field's name
// and the value being typed, and under them, in a box whose Enter can be
// refused, a row kept for why: `name is taken`, `wrong PIN` (tdp F7, K3,
// 2026-09-28; it used to be a suffix on the border, ` · taken`).
//
// While it is up every printable key is a character: Space is a space and
// ? is a question mark (tdp K8). A line break or a tab pasted in stays, and
// is drawn as a red `\n` / `\t`; the box will not take the value with one
// (oneline.go).
type inputPopup struct {
	anim   popupAnimator
	title  string // the type: "name", "number", "current PIN", …
	err    string // why the last Enter was refused; cleared by the next key
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

// inputThawMsg ends the frozen second of the box whose animator is target:
// three boxes can be up at once (the PIN chain), each counting its own.
type inputThawMsg struct {
	target string
	gen    int
}

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
// error row, the value cleared, every key swallowed (ux.md §2.2).
func (m *inputPopup) freeze(err string) tea.Cmd {
	m.err, m.value, m.frozen = err, "", true
	m.frozenGen++
	target, gen := m.anim.target, m.frozenGen
	return tea.Tick(inputFreeze, func(time.Time) tea.Msg { return inputThawMsg{target, gen} })
}

func (m *inputPopup) thaw(msg inputThawMsg) {
	if msg.target == m.anim.target && msg.gen == m.frozenGen {
		m.frozen, m.err = false, ""
	}
}

// update edits the line; Enter and Esc are resolved by the caller, since
// commit and cancel are each one role app-wide (tdp K1).
func (m *inputPopup) update(msg tea.KeyMsg) {
	if !m.anim.isInteractive() || m.frozen {
		return
	}
	m.err = ""
	switch msg.Type {
	case tea.KeyTab:
		if m.value == "" && m.placeholder != "" {
			m.value, m.placeholder = m.placeholder, ""
		}
	case tea.KeyBackspace:
		if m.value != "" {
			m.value = dropLast(m.value)
		} else {
			m.placeholder = ""
		}
	case tea.KeyCtrlU:
		m.value = ""
	case tea.KeySpace:
		m.value += " "
	case tea.KeyRunes:
		m.value += takeText(msg.Runes)
	}
}

// canFail: an Enter here can be refused, so the box keeps a row for why
// from the moment it opens (tdp F7). Every box's can: a line break or a
// tab in the value is refused in all of them, a command's too, which was
// taken as it was typed until 2026-10-06.
func (m inputPopup) canFail() bool { return m.action != inputNone }

// errorRow is an input's row for why its Enter was refused (tdp F7, K3):
// blank until then, red with the reason after. A PIN box's is centred
// under its dots, the same in all three and on the lock (ui.md §3.2).
func errorRow(err string, innerW int, centre bool) string {
	if err == "" {
		return spaces(innerW)
	}
	err = truncate(err, innerW-2)
	lead := 1
	if centre {
		lead = (innerW - dispW(err)) / 2
	}
	return lipgloss.NewStyle().Foreground(warnColor).Render(padRight(spaces(lead)+err, innerW))
}

func (m inputPopup) view() string {
	bc := popupLayerColor(m.layer)
	if m.err != "" {
		bc = warnColor
	}
	if m.masked {
		// A PIN box IS the lock's PIN prompt (pinprompt.go): the same
		// width, the same air, the same spaced dots from the middle — one
		// look for a PIN wherever it is typed (user, 2026-09-24).
		innerW := popupInnerW(m.screenW)
		hint := [][2]string{{"Enter", m.accept}, {"Esc", "cancel"}}
		if m.frozen {
			hint = nil
		}
		return drawPopupBox(bc, " "+glyphLock+" "+m.title+" ", hint,
			animRows(m.anim, []string{pinRow(valueLen(m.value), innerW), errorRow(m.err, innerW, true)}), innerW)
	}

	innerW := popupInnerW(m.screenW)
	dim := lipgloss.NewStyle().Foreground(dimColor)
	cur := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(editColor)

	value, vw := valueView(m.value, innerW-3)
	line := " " + value + cur.Render(" ") + spaces(max(0, innerW-2-vw))
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
	if m.canFail() {
		rows = append(rows, errorRow(m.err, innerW, false))
	}
	pairs := [][2]string{{"Enter", m.accept}}
	if offered {
		pairs = append(pairs, [2]string{"Tab", "edit it"}, [2]string{"Backspace", "clear"})
	}
	hint := append(pairs, [2]string{"Esc", "cancel"})
	if m.frozen {
		hint = nil
	}
	return drawPopupBox(bc, " "+glyphInput+" "+m.title+" ", hint,
		animRows(m.anim, rows), innerW)
}
