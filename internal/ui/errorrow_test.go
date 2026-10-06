package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// An input whose Enter can be refused keeps a row for why from the moment
// it opens: with the error or without, the box is as tall, and the error
// is in that row inside it, not on the border (tdp F7, K3, 2026-09-28).
// Every box's Enter can be: a command's too, since a line break or a tab
// is refused in all of them (2026-10-06; until then it kept none).
func TestInputErrorRow(t *testing.T) {
	open := popupAnimator{phase: animOpen}
	boxes := map[string]func(err string) string{
		"name": func(err string) string {
			return inputPopup{anim: open, title: "name", prompt: "name", action: inputRename, err: err, screenW: 100, screenH: 30}.view()
		},
		"new PIN": func(err string) string {
			return inputPopup{anim: open, title: "new PIN", masked: true, action: inputPINNew, err: err, screenW: 100, screenH: 30}.view()
		},
	}
	for name, box := range boxes {
		clean, refused := box(""), box("name is taken")
		if strings.Count(clean, "\n") != strings.Count(refused, "\n") {
			t.Errorf("%s: %d rows clean, %d with the error", name, strings.Count(clean, "\n"), strings.Count(refused, "\n"))
		}
		inside(t, name, refused, "name is taken")
	}
	name := inputPopup{anim: open, title: "name", prompt: "name", action: inputRename, screenW: 100, screenH: 30}.view()
	command := inputPopup{anim: open, title: "command", prompt: "command", action: inputCommand, screenW: 100, screenH: 30}.view()
	if strings.Count(command, "\n") != strings.Count(name, "\n") {
		t.Errorf("a command box keeps no error row:\n%s", command)
	}
}

// The lock's PIN prompt: wrong and the lockout are in its error row, the
// box as tall as when idle.
func TestPINPromptErrorRow(t *testing.T) {
	now := time.Now()
	p := pinPrompt{anim: popupAnimator{phase: animOpen}, screenW: 100, screenH: 30}
	idle := p.view(now)
	p.state = promptWrong
	inside(t, "wrong", p.view(now), "wrong PIN")
	if strings.Count(p.view(now), "\n") != strings.Count(idle, "\n") {
		t.Errorf("wrong changed the box's height")
	}
	p.state, p.until = promptLockout, now.Add(30*time.Second)
	inside(t, "lockout", p.view(now), "try again in 30 s")
}

// inside: want is on a row inside box, and not on its borders.
func inside(t *testing.T, name, box, want string) {
	t.Helper()
	lines := strings.Split(ansi.Strip(box), "\n")
	if strings.Contains(lines[0], want) || strings.Contains(lines[len(lines)-1], want) {
		t.Errorf("%s: %q is on the border:\n%s", name, want, ansi.Strip(box))
	}
	if !strings.Contains(strings.Join(lines[1:len(lines)-1], "\n"), want) {
		t.Errorf("%s: %q is not in the box:\n%s", name, want, ansi.Strip(box))
	}
}
