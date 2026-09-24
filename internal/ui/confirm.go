package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// confirmAction is what the caller does when the user commits (ux.md §5).
type confirmAction int

const (
	confirmNone        confirmAction = iota
	confirmDeleteSaver               // remove the saver at ref
	confirmQuit                      // leave with the colour draft unsaved
)

// confirmPopup is the message class (§6.1): a short question with one yes
// and one no. It is NOT a menu — a menu is "pick one of N", and blurring
// the two would make Enter mean different things on different floats.
type confirmPopup struct {
	anim    popupAnimator
	title   string
	lines   []string
	accept  string // what Enter does, shown in the hint
	action  confirmAction
	ref     int // the saver the action is about
	layer   int
	screenW int
	screenH int
}

func newConfirmPopup() confirmPopup { return confirmPopup{anim: newPopupAnimator("confirm")} }

func (m confirmPopup) isActive() bool      { return m.anim.isActive() }
func (m confirmPopup) isInteractive() bool { return m.anim.isInteractive() }
func (m *confirmPopup) close() tea.Cmd     { return m.anim.close() }
func (m *confirmPopup) setSize(w, h int)   { m.screenW, m.screenH = w, h }

func (m *confirmPopup) ask(c confirmPopup, layer int) tea.Cmd {
	anim, w, h := m.anim, m.screenW, m.screenH
	*m = c
	m.anim, m.screenW, m.screenH = anim, w, h
	m.layer = layer
	return m.anim.open()
}

func (m confirmPopup) view() string {
	w := dispW(m.title) + 6
	for _, l := range m.lines {
		w = max(w, dispW(l)+4)
	}
	innerW := popupInnerW(m.screenW, w)
	txt := lipgloss.NewStyle().Foreground(textColor)
	dim := lipgloss.NewStyle().Foreground(dimColor)
	rows := make([]string, 0, len(m.lines))
	for i, l := range m.lines {
		style := dim
		if i == 0 {
			style = txt
		}
		rows = append(rows, style.Render(padRight("  "+l, innerW)))
	}
	hint := hintLegend([][2]string{{"Enter", m.accept}, {"Esc", "cancel"}})
	return drawPopupBox(popupLayerColor(m.layer), " "+glyphWarn+" "+m.title+" ", hint,
		animRows(m.anim, capRows(rows, m.screenH)), innerW)
}
