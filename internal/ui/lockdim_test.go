package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// Under the PIN prompt the lock fades whole, each colour its own way: the
// lit pixels, the ground, the status row with a config error's red — the
// same fading as the settings screen's (tdp F8, D2; user, 2026-09-28).
func TestPromptDimsTheLock(t *testing.T) {
	colours(t)
	m := testLock(t, "1234", nil)
	m.problem = "config.yaml: yaml: line 1"
	lines := func(m LockModel) []string { return strings.Split(m.View(), "\n") }
	status := func(m LockModel) string { l := lines(m); return l[len(l)-1] }
	fg, bg := lipgloss.Color(m.style.FG), lipgloss.Color(m.style.BG)

	if l := status(m); !has(l, warnColor) {
		t.Fatalf("no prompt: the config error is red: %q", l)
	}
	m = openPrompt(t, m)
	if l := status(m); has(l, warnColor) || !has(l, dimmed(warnColor)) {
		t.Errorf("under the prompt the config error must be its red faded: %q", l)
	}
	board := strings.Join(lines(m)[:len(lines(m))-1], "\n")
	if has(board, fg) || !has(board, dimmed(fg)) {
		t.Errorf("under the prompt the lit pixels must be %s faded, not %s", fg, fg)
	}
	if !has(board, dimmed(bg)) {
		t.Errorf("under the prompt the ground must be %s faded", bg)
	}
	// Not one grey for all: the lit pixels are not the ground.
	if dimmed(fg) == dimmed(bg) || has(board, dimColor) && !has(board, dimmed(fg)) {
		t.Errorf("the lit pixels and the ground faded into one colour")
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
