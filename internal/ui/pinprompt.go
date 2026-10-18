package ui

import (
	"math"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/locku/internal/config"
)

// pinPrompt is the lock screen's one popup (ui.md §3.2): a password box in
// its unlock look, as the lib draws one (components/input/password,
// input/README, layout/popup) — the title's capsule, a prompt, the dots a
// cell apart in the middle and growing out both ways, the error row under
// them, the hint in the bottom border — centred over the board, which goes
// on underneath it, dimmed. The settings screen's PIN boxes are the lib's
// own, and this one looks as they do (user, 2026-09-24: a PIN looks the
// same wherever it is typed).
//
// TODO(lib LK1): the lib's input boxes open only in its App, which draws a
// footer, and have nothing for what the lock adds — a second of "wrong" that
// swallows keys, the cooling-off, the timeout — so the box is drawn here, the
// lib's way.
//
// TODO(tdp LK6): the box was seven rows, the dots in its very middle, no
// prompt (user, 2026-10-07); the one-line input's skeleton has a prompt, so
// nine.
//
// Its states, the box never changing size:
//
//	idle        the prompt, the dots, Enter:unlock Backspace:clear Esc:cancel
//	refused     the value has a control character pasted into it: not
//	            checked, the error row says so, the dots kept
//	wrong PIN   a second of every key swallowed, the dots cleared; the error
//	            row says so till the next Enter
//	try again   `try again in 27 s` counting down, every key but Esc swallowed
type promptState int

const (
	promptIdle promptState = iota
	promptWrong
	promptLockout
)

const wrongHold = time.Second

// The words on the box.
const (
	pinTitle      = "Unlock"
	pinPromptText = "The PIN"
	pinWrong      = "wrong PIN"
	pinControlErr = "PIN can't have control characters"
)

type pinPrompt struct {
	anim  popupAnimator
	value []rune
	state promptState
	// until is when a lockout ends; the error row counts down to it.
	until time.Time
	// err is the error row: what the last Enter found (input/README: it
	// changes when the value is checked, and only then).
	err     string
	screenW int
	screenH int
}

func newPinPrompt() pinPrompt { return pinPrompt{} }

func (p *pinPrompt) setSize(w, h int) { p.screenW, p.screenH = w, h }

func (p *pinPrompt) open() tea.Cmd {
	p.value, p.err = nil, ""
	p.state = promptIdle
	return p.anim.open()
}

func (p *pinPrompt) close() tea.Cmd {
	p.value = nil
	return p.anim.close()
}

// add appends what was typed, up to the PIN's longest: a key's printable
// characters, and every character of a paste, control characters too —
// drawn as dots like the rest, and refused on Enter (input/README, pasted
// control characters).
func (p *pinPrompt) add(rs []rune, paste bool) {
	for _, r := range rs {
		if !paste && unicode.IsControl(r) {
			continue
		}
		if len(p.value) < config.PINMax {
			p.value = append(p.value, r)
		}
	}
}

// clear is Backspace: the whole value goes, not a character of it — a
// value one cannot see is typed again faster than mended
// (components/input/password).
func (p *pinPrompt) clear() { p.value = nil }

// hasControl reports a control character in the value.
func (p pinPrompt) hasControl() bool {
	for _, r := range p.value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// remaining is the whole seconds left in a lockout, at least 1 while it
// is on.
func (p pinPrompt) remaining(now time.Time) int {
	return max(1, int(math.Ceil(p.until.Sub(now).Seconds())))
}

// idleHint is the box's hint while it takes keys: the box has a typing
// row only, so the hint is every key there is (Rules M3).
var idleHint = []terminu.KeyHint{{Key: "Enter", Desc: "unlock"}, {Key: "Backspace", Desc: "clear"}, {Key: "Esc", Desc: "cancel"}}

// hint is what can be pressed now (Rules M6: only those, in a hint):
// nothing for the second of "wrong", Esc in a lockout.
func (p pinPrompt) hint() []terminu.KeyHint {
	switch p.state {
	case promptWrong:
		return nil
	case promptLockout:
		return idleHint[2:]
	}
	return idleHint
}

// size is the box on the screen: as wide as the longest PIN's dots ask, the
// prompt, the title and the whole hint (layout/popup, width: known when it
// opens), no wider than the screen allows; nine rows with its borders.
func (p pinPrompt) size() (w, h int) {
	most := max(min(p.screenW-2, 120), 1)
	need := max(2*config.PINMax-1, terminu.Width(pinPromptText)) + 4
	need = max(need, terminu.Width(titleCapsule(1<<20))+4)
	need = max(need, hintWidth(idleHint)+6)
	return min(need, most), min(9, max(p.screenH-2, 3))
}

// place is where the box's top left corner goes: the middle of the screen.
func (p pinPrompt) place() (x, y int) {
	w, h := p.size()
	return (p.screenW - w) / 2, (p.screenH - h) / 2
}

// full is the open box, every row w cells.
func (p pinPrompt) full(now time.Time) []string {
	w, h := p.size()
	inner := w - 2
	cw := inner - 2
	layer := terminu.LayerColor(1)
	err := p.err
	if p.state == promptLockout {
		err = "try again in " + itoa(p.remaining(now)) + " s"
	}
	body := []string{
		padRow(terminu.Fg(terminu.Overlay0).Render(pinPromptText), inner),
		strings.Repeat(" ", inner),
		" " + terminu.Center(p.dots(cw), cw) + " ",
		strings.Repeat(" ", inner),
		" " + terminu.Center(terminu.Fg(terminu.Red).Render(terminu.Truncate(err, cw)), cw) + " ",
	}
	body = terminu.Pad2(body, inner, max(h-4, 0))
	return frameBox(w, layer, titleCapsule(inner-2), p.hint(), body)
}

// dots is the value in the unlock look: a ● a cell apart for each
// character, the cursor where the next one goes; once all of them are typed
// the cursor is on the last ● (components/input/password). Too wide, it
// keeps the cursor's end.
func (p pinPrompt) dots(w int) string {
	if p.state != promptIdle {
		return ""
	}
	n := len(p.value)
	plain := strings.Repeat("● ", n) + " "
	if n >= config.PINMax {
		plain = strings.TrimSuffix(plain, "  ")
	}
	cells := []rune(plain)
	if len(cells) > w {
		cells = append([]rune{'…'}, cells[len(cells)-max(w-1, 1):]...)
	}
	last := len(cells) - 1
	value := terminu.Fg(terminu.Lavender)
	cursor := terminu.Style{Fg: terminu.Base, Bg: terminu.Lavender}
	return value.Render(string(cells[:last])) + cursor.Render(string(cells[last]))
}

// view is the box as its animation frame has it, and where it goes: rows
// to put over the screen with their top left at x, y; none while closed.
func (p pinPrompt) view(now time.Time) (rows []string, x, y int) {
	if !p.anim.isActive() {
		return nil, 0, 0
	}
	full := p.full(now)
	w, h := p.size()
	bx, by := p.place()
	rows, first, dx := p.anim.frameOf(full, w, h, terminu.LayerColor(1))
	return rows, bx + dx, by + first
}

// ---- the box, drawn the lib's way (components/layout/popup)

const (
	glyphRoundLeft  = "\U0000E0B6" // nf-ple-left_half_circle_thick: a popup title's left end
	glyphRoundRight = "\U0000E0B4" // nf-ple-right_half_circle_thick: its right end
)

// titleCapsule is the box's title: the lock glyph and UNLOCK in a capsule
// rounded at both ends, filled with the layer colour, at most maxW cells.
func titleCapsule(maxW int) string {
	layer := terminu.LayerColor(1)
	ends := terminu.Width(glyphRoundLeft) + terminu.Width(glyphRoundRight)
	text := terminu.Truncate(glyphLock+" "+terminu.Upper(pinTitle), maxW-ends)
	if text == "" {
		return ""
	}
	return terminu.Fg(layer).Render(glyphRoundLeft) + terminu.On(layer).Render(text) + terminu.Fg(layer).Render(glyphRoundRight)
}

// padRow is one row of the body: one cell from the frame on either side.
func padRow(s string, inner int) string { return " " + terminu.Fit(s, inner-2) + " " }

func hintWidth(pairs []terminu.KeyHint) int {
	w := 0
	for i, p := range pairs {
		if i > 0 {
			w++
		}
		w += terminu.Width(p.Key) + 1 + terminu.Width(p.Desc)
	}
	return w
}

// renderHint writes the pairs as Key:desc, keys in Blue and the colon and
// words in Overlay0, dropping whole pairs from the end until they fit in w.
func renderHint(pairs []terminu.KeyHint, w int) string {
	for n := len(pairs); n > 0; n-- {
		if hintWidth(pairs[:n]) > w {
			continue
		}
		parts := make([]string, n)
		for i, p := range pairs[:n] {
			parts[i] = terminu.Fg(terminu.Blue).Render(p.Key) + terminu.Fg(terminu.Overlay0).Render(":"+p.Desc)
		}
		return strings.Join(parts, " ")
	}
	return ""
}

// frameBox frames body — rows as wide as the inside — in the heavy line of
// color: the title one segment after the top left corner, a blank row above
// and below the body, the hint one segment after the bottom left corner.
func frameBox(w int, color terminu.Color, title string, hint []terminu.KeyHint, body []string) []string {
	inner := w - 2
	if inner < 0 {
		return nil
	}
	line := terminu.Fg(color)
	top := ""
	if title != "" && inner > 1 {
		top = line.Render("━") + terminu.Cut(title, 0, inner-1)
	}
	out := []string{line.Render("┏") + top + line.Render(strings.Repeat("━", max(inner-terminu.Width(top), 0))) + line.Render("┓")}
	rows := append([]string{strings.Repeat(" ", inner)}, body...)
	rows = append(rows, strings.Repeat(" ", inner))
	for _, r := range rows {
		out = append(out, line.Render("┃")+terminu.Pad(r, inner)+line.Render("┃"))
	}
	h := ""
	if len(hint) > 0 && inner-4 > 0 {
		if s := renderHint(hint, inner-4); s != "" {
			h = " " + s + " "
		}
	}
	fill := max(inner-1-terminu.Width(h), 0)
	return append(out, line.Render("┗━")+h+line.Render(strings.Repeat("━", fill))+line.Render("┛"))
}

// ---- the animation (Rules F2, components/layout/popup)

// A popup opens in three steps — a line drawn out from the middle (four
// frames), held (one), then opened up and down (four) — and closes the same
// nine frames backwards, 32ms a frame; it takes no keys till it is open.
const (
	openFrames = 9
	frameTime  = 32 * time.Millisecond
)

// AnimTickMsg is the prompt's animation's next frame.
type AnimTickMsg struct{}

type animPhase int

const (
	animClosed animPhase = iota
	animOpening
	animOpen
	animClosing
)

type popupAnimator struct {
	phase animPhase
	frame int // 1–9 while opening, 9 when open, counting down while closing
}

func (a popupAnimator) isActive() bool { return a.phase != animClosed }

// owns reports whether the box still has the keys: opening or open. A
// closing one has not (Rules F3), and an opening one swallows them
// (isInteractive), so none lands on a half-drawn box.
func (a popupAnimator) owns() bool { return a.phase == animOpening || a.phase == animOpen }

func (a popupAnimator) isInteractive() bool { return a.phase == animOpen }

func (a *popupAnimator) open() tea.Cmd {
	a.phase, a.frame = animOpening, 1
	return animTick()
}

func (a *popupAnimator) close() tea.Cmd {
	if a.phase == animClosed {
		return nil
	}
	a.phase = animClosing
	return animTick()
}

func (a *popupAnimator) tick() tea.Cmd {
	switch a.phase {
	case animOpening:
		if a.frame++; a.frame >= openFrames {
			a.frame, a.phase = openFrames, animOpen
			return nil
		}
	case animClosing:
		if a.frame--; a.frame <= 0 {
			a.frame, a.phase = 0, animClosed
			return nil
		}
	default:
		return nil
	}
	return animTick()
}

func animTick() tea.Cmd {
	return tea.Tick(frameTime, func(time.Time) tea.Msg { return AnimTickMsg{} })
}

// frameOf is the w × h box full as this frame shows it: the rows, the row
// of the box they start at, and how far in from its left: the line — a
// quarter of the width a frame, from the middle out — for frames 1–5, the
// rows around the middle — a quarter of the height a frame — for 6–8.
func (a popupAnimator) frameOf(full []string, w, h int, color terminu.Color) (rows []string, first, x int) {
	f := a.frame
	switch {
	case f <= 0:
		return nil, 0, 0
	case f <= 5:
		n := min(f, 4)
		width := max((w*n+3)/4, 1)
		if width%2 != w%2 && width < w {
			width++
		}
		return []string{terminu.Fg(color).Render(strings.Repeat("━", width))}, h / 2, (w - width) / 2
	case f < openFrames:
		n := max((h*(f-5)+3)/4, 1)
		start := (h - n) / 2
		return full[start : start+n], start, 0
	}
	return full, 0, 0
}
