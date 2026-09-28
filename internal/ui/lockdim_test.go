package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// Under the PIN prompt the lock steps back whole: the lit pixels and the
// status row, a config error's red among it, in the one dim colour
// (tdp F8; user, 2026-09-28: Overlay0, as in the settings screen).
func TestPromptDimsTheLock(t *testing.T) {
	colours(t)
	m := testLock(t, "1234", nil)
	m.problem = "config.yaml: yaml: line 1"
	lines := func(m LockModel) []string { return strings.Split(m.View(), "\n") }
	status := func(m LockModel) string { l := lines(m); return l[len(l)-1] }
	fg := m.style.FG

	if l := status(m); !has(l, warnColor) {
		t.Fatalf("no prompt: the config error is red: %q", l)
	}
	m = openPrompt(t, m)
	if l := status(m); has(l, warnColor) || !has(l, dimColor) {
		t.Errorf("under the prompt the status row must be dim, red and all: %q", l)
	}
	board := strings.Join(lines(m)[:len(lines(m))-1], "\n")
	// Overlay0, not the Surface2 it was: the prompt's own hint is Overlay0
	// too, so it is Surface2's absence that says which the pixels are.
	if has(board, lipgloss.Color(fg)) || has(board, borderDim) || !has(board, dimColor) {
		t.Errorf("under the prompt the lit pixels must be Overlay0, not %s or Surface2", fg)
	}

	// Esc: the prompt goes, and the lock is itself again.
	m, _ = m.step(keyEsc)
	for i := 0; i < animFrames+1 && m.prompt.anim.isActive(); i++ {
		m, _ = m.step(AnimTickMsg{Target: "pinprompt"})
	}
	if l := status(m); !has(l, warnColor) {
		t.Errorf("the prompt gone, the config error is red again: %q", l)
	}
}
