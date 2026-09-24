package ui

import (
	"math"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
)

// pinPrompt is the lock screen's one popup (ui.md §3.2): a masked line,
// 48 columns, centred, a row of air above and below, the dots growing
// out from the middle of the box (user, 2026-09-24: bigger, and the
// input starting from the centre). It has four looks and they differ
// only in the border and its title — the row inside never moves:
//
//	PIN                       idle: the layer colour, enter unlock · esc back
//	PIN · wrong               red for a second, the row cleared, every key swallowed
//	PIN · try again in 27 s   red, counting down, every key but Esc swallowed
//	PIN · closing             prompt_timeout ran out: the ordinary closing animation
//
// Red is an override colour (VTP §2.4): it says "wrong", not "deeper".
type promptState int

const (
	promptIdle promptState = iota
	promptWrong
	promptLockout
)

const (
	pinPromptW = 48 // the whole box, borders included (32 until 2026-09-24)
	wrongHold  = time.Second
)

type pinPrompt struct {
	anim  popupAnimator
	value []rune
	state promptState
	// until is when a lockout ends; the title counts down to it.
	until time.Time
	// timedOut marks a close that prompt_timeout caused, for the title.
	timedOut bool
	screenW  int
	screenH  int
}

func newPinPrompt() pinPrompt { return pinPrompt{anim: newPopupAnimator("pinprompt")} }

func (p *pinPrompt) setSize(w, h int) { p.screenW, p.screenH = w, h }

func (p *pinPrompt) open() tea.Cmd {
	p.value, p.timedOut = nil, false
	p.state = promptIdle
	return p.anim.open()
}

func (p *pinPrompt) close(timedOut bool) tea.Cmd {
	p.value, p.timedOut = nil, timedOut
	return p.anim.close()
}

// add appends one character, up to the PIN's longest allowed.
func (p *pinPrompt) add(r rune) {
	if len(p.value) < config.PINMax && unicode.IsPrint(r) {
		p.value = append(p.value, r)
	}
}

func (p *pinPrompt) backspace() {
	if n := len(p.value); n > 0 {
		p.value = p.value[:n-1]
	}
}

// remaining is the whole seconds left in a lockout, at least 1 while it
// is on.
func (p pinPrompt) remaining(now time.Time) int {
	return max(1, int(math.Ceil(p.until.Sub(now).Seconds())))
}

func (p pinPrompt) view(now time.Time) string {
	// 48 columns, or what a smaller terminal can hold.
	innerW := popupInnerW(p.screenW, pinPromptW-2)
	bc := popupLayerColor(1)
	title := " " + glyphLock + " PIN "
	hint := ""
	row := ""
	switch {
	case p.anim.phase == animClosing && p.timedOut:
		title += "· closing "
	case p.state == promptWrong:
		bc = warnColor
		title += "· wrong "
	case p.state == promptLockout:
		bc = warnColor
		title += "· try again in " + itoa(p.remaining(now)) + " s "
		hint = hintLegend([][2]string{{"Esc", "back"}})
	default:
		hint = hintLegend([][2]string{{"Enter", "unlock"}, {"Esc", "back"}})
		row = pinRow(len(p.value), innerW)
	}
	return drawPopupBox(bc, title, hint, animRows(p.anim, []string{row}), innerW)
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
