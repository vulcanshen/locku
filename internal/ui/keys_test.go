package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// fgBefore is the last foreground l sets before s, "" if s is not in l
// as one run of text.
func fgBefore(l, s string) string {
	i := strings.Index(l, s)
	if i < 0 {
		return ""
	}
	ms := fgRe.FindAllString(l[:i], -1)
	if len(ms) == 0 {
		return ""
	}
	return ms[len(ms)-1]
}

// footer is the view's last row.
func footer(m AppModel) string {
	ls := strings.Split(m.View(), "\n")
	return ls[len(ls)-1]
}

// A hint and the footer are Key:what, one space apart, keys named as
// the keycap names them (tdp M5, D1, 2026-09-29).
func TestHintsAndFooterAreKeyColonWhat(t *testing.T) {
	m := newTestApp(t)
	if got := strings.TrimSpace(ansi.Strip(footer(m))); got != "Space:menu ?:help Tab/1–2:panels q:quit" {
		t.Errorf("footer: %q", got)
	}
	// Too narrow for all four: whole pairs go, from the right.
	if got := strings.TrimSpace(ansi.Strip(footer(m.size(30, 12)))); got != "Space:menu ?:help" {
		t.Errorf("footer at 30 columns: %q", got)
	}
	if v := ansi.Strip(m.press(" ").View()); !strings.Contains(v, "j/k:move Enter:run Esc:close") {
		t.Errorf("the Space menu's hint:\n%s", v)
	}
	// tmux's config file path, with an offer to take.
	v := ansi.Strip(m.press("G", "k", "k", "2", "j", "enter").View())
	if !strings.Contains(v, "Enter:save Tab:edit it Backspace:clear Esc:cancel") || strings.Contains(v, "Bksp") {
		t.Errorf("the path box's hint:\n%s", v)
	}
}

// The key Blue, the colon and what it does Overlay0 (tdp D2).
func TestHintAndFooterColours(t *testing.T) {
	colours(t)
	m := newTestApp(t)
	f := footer(m)
	if !has(fgBefore(f, "Space"), focusColor) || !has(fgBefore(f, ":menu"), dimColor) {
		t.Errorf("footer: Space Blue, :menu Overlay0: %q", f)
	}
	h := line(m.press(" ").View(), "j/k:move")
	if !has(fgBefore(h, "j/k"), focusColor) || !has(fgBefore(h, ":move"), dimColor) {
		t.Errorf("hint: j/k Blue, :move Overlay0: %q", h)
	}
}

// helpLines is the ? box's every line, colours and all.
func helpLines(m AppModel) string {
	_, _, lines := m.help.layout()
	return strings.Join(lines, "\n")
}

// In ?, keys are Blue and what they do Text; keys that do one thing are
// joined with /, a range with – (tdp M5, D2, 2026-09-29). A glossary's
// left column is the settings' names, not keys: it stays as it was.
func TestKeyReferenceKeys(t *testing.T) {
	colours(t)
	m := newTestApp(t)
	ref := helpLines(m.press("?"))
	if s := ansi.Strip(ref); !strings.Contains(s, "q/Ctrl-C") || !strings.Contains(s, "Tab/1–2") || strings.Contains(s, "q · Ctrl-C") {
		t.Errorf("keys doing one thing, with / and –:\n%s", s)
	}
	l := line(ref, "close the top popup")
	if !has(fgBefore(l, "Esc"), focusColor) || has(l, handColor) || !has(fgBefore(l, "close the top popup"), textColor) {
		t.Errorf("Esc Blue, what it does Text: %q", l)
	}
	if s := ansi.Strip(helpLines(m.press(" ", "?"))); !strings.Contains(s, "Space/Esc") {
		t.Errorf("the Space menu's ?:\n%s", s)
	}
	tmux := m.press("G", "k", "k", "2", "?")
	g := line(helpLines(tmux), "locku's block is in the file")
	if !has(fgBefore(g, "activate"), handColor) || has(g, focusColor) {
		t.Errorf("tmux's glossary: the name is not a key: %q", g)
	}
	// The same box, a key reference again after the glossary.
	l = line(helpLines(tmux.press("esc", "1", "?")), "close the top popup")
	if !has(fgBefore(l, "Esc"), focusColor) {
		t.Errorf("? on [1] after the glossary: Esc Blue: %q", l)
	}
}

// A key named in a sentence is in square brackets: a confirm's lines, a
// box's prompt, a glossary's words, the splash (tdp M5, 2026-09-29).
func TestKeysInSentencesAreBracketed(t *testing.T) {
	m := newTestApp(t)
	q := m.press("2").typed(strings.Repeat("j", stopBgR)).press("enter", "G", "enter", "q")
	if v := ansi.Strip(q.View()); !strings.Contains(v, "[S] on the profile saves them, [R] drops them") {
		t.Errorf("the quit confirm:\n%s", v)
	}
	if v := ansi.Strip(m.press("G", "k", "k", "2", "j", "enter").View()); !strings.Contains(v, "[Backspace] then [Enter] to unset") {
		t.Errorf("the path box's prompt:\n%s", v)
	}
	activate := ""
	for _, e := range m.press("G", "k", "k", "2", "?").help.entries {
		if e.key == "activate" {
			activate = e.desc
		}
	}
	if !strings.Contains(activate, "[Enter] turns it") {
		t.Errorf("tmux's glossary, activate: %q", activate)
	}
	s := newSplashModel()
	s.show()
	s, _ = s.update(splashHintMsg{})
	if v := ansi.Strip(s.render(100, 40)); !strings.Contains(v, "Press [Esc] to close") {
		t.Errorf("the splash:\n%s", v)
	}
}
