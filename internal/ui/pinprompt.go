package ui

import (
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
)

// pinPrompt is the lock screen's one popup (ui.md §3.2): a masked line,
// as wide as every popup (tdp F7; 48 columns until 2026-09-28), centred,
// the dots in its middle row growing out from the middle of the box (user,
// 2026-09-24: bigger, and the input starting from the centre). A
// 64-character PIN is 129 columns of dots, wider than a popup gets: the
// dots it has no room for go from the front (user, 2026-09-28: the widest
// PIN against 120 settles the width). Two rows under the dots is its error
// row, blank until an Enter is refused (tdp F7, K3; 2026-09-28 — the error
// used to be in the title; pinBox lays the rows out). It has five looks,
// and the box never changes size:
//
//	idle        the layer colour, enter unlock · esc back
//	not taken   red, why in the error row — a line break or a tab in it
//	            (oneline.go) — the dots kept, until the next key
//	wrong PIN   red for a second, the dots cleared, every key swallowed
//	try again   red, `try again in 27 s` counting down, every key but Esc swallowed
//	closing     `PIN · closing` in the title: pin_prompt_timeout ran out, the
//	            ordinary closing animation
//
// Red is an override colour (tdp D2): it says "wrong", not "deeper".
type promptState int

const (
	promptIdle promptState = iota
	promptWrong
	promptLockout
)

const wrongHold = time.Second

type pinPrompt struct {
	anim  popupAnimator
	value []rune
	state promptState
	// until is when a lockout ends; the title counts down to it.
	until time.Time
	// timedOut marks a close that pin_prompt_timeout caused, for the title.
	timedOut bool
	// err is why the last Enter was not even checked; the next key clears it.
	err     string
	screenW int
	screenH int
}

func newPinPrompt() pinPrompt { return pinPrompt{anim: newPopupAnimator("pinprompt")} }

func (p *pinPrompt) setSize(w, h int) { p.screenW, p.screenH = w, h }

func (p *pinPrompt) open() tea.Cmd {
	p.value, p.timedOut, p.err = nil, false, ""
	p.state = promptIdle
	return p.anim.open()
}

func (p *pinPrompt) close(timedOut bool) tea.Cmd {
	p.value, p.timedOut, p.err = nil, timedOut, ""
	return p.anim.close()
}

// add appends what of rs goes into a value (oneline.go: a line break or a
// tab too; until 2026-10-06 printable characters only), up to the PIN's
// longest allowed.
func (p *pinPrompt) add(rs []rune) {
	for _, r := range takeText(rs) {
		if len(p.value) < config.PINMax {
			p.value = append(p.value, r)
		}
	}
}

// backspace takes the last unit: a "\r\n" whole.
func (p *pinPrompt) backspace() { p.value = []rune(dropLast(string(p.value))) }

// remaining is the whole seconds left in a lockout, at least 1 while it
// is on.
func (p pinPrompt) remaining(now time.Time) int {
	return max(1, int(math.Ceil(p.until.Sub(now).Seconds())))
}

func (p pinPrompt) view(now time.Time) string {
	innerW := popupInnerW(p.screenW)
	bc := popupLayerColor(1)
	title := " " + glyphLock + " PIN "
	var hint [][2]string
	row, err := spaces(innerW), ""
	switch {
	case p.anim.phase == animClosing && p.timedOut:
		title += "· closing "
	case p.state == promptWrong:
		bc, err = warnColor, "wrong PIN"
	case p.state == promptLockout:
		bc, err = warnColor, "try again in "+itoa(p.remaining(now))+" s"
		hint = [][2]string{{"Esc", "back"}}
	default:
		hint = [][2]string{{"Enter", "unlock"}, {"Esc", "back"}}
		row = pinRow(valueLen(string(p.value)), innerW)
		if p.err != "" {
			bc, err = warnColor, p.err
		}
	}
	return pinBox(bc, title, hint, p.anim, row, errorRow(err, innerW, true), innerW)
}

// pinBox frames every PIN box — the lock's prompt and the settings
// screen's current / new / confirm boxes: seven rows with the borders,
// two rows of air, the dots in the middle row, a row of air, the error
// row against the bottom border (user, 2026-10-07: the error row under the
// dots, with a row of air under it too, had left them a row above the
// middle).
func pinBox(bc lipgloss.Color, title string, hint [][2]string, a popupAnimator, row, err string, innerW int) string {
	air := spaces(innerW)
	return drawPopupBoxPad(bc, title, hint, animRows(a, []string{air, air, row, air, err}), innerW, false)
}

// pinRow is the masked line every PIN box shares — the lock's prompt and
// the settings screen's current / new / confirm boxes (ui.md §3.2): one
// dot a character with a space between, the cursor after, the whole of
// it in the middle of the box and growing out both ways as the PIN is
// typed (user, 2026-09-24: a PIN looks the same wherever it is typed).
func pinRow(n, innerW int) string {
	edit := lipgloss.NewStyle().Foreground(editColor)
	cur := lipgloss.NewStyle().Foreground(lipgloss.Color(baseHex)).Background(editColor)
	dots := strings.Repeat("● ", n)
	shown := truncateHead(dots, innerW-2)
	return spaces((innerW-dispW(shown)-1)/2) + edit.Render(shown) + cur.Render(" ")
}
