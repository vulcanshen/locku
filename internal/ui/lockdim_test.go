package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// Under the PIN box the lock fades whole, each colour its own way: the lit
// pixels, the ground, the status row with a config error's red — the lib's
// fading, as on the settings screen (tdp F8, D2; user, 2026-09-28). As the
// box starts to close the lock is itself again (components/layout/popup).
func TestPromptDimsTheLock(t *testing.T) {
	colours(t)
	m := testLock(t, "1234", nil)
	m.problem = "config.yaml: yaml: line 1"
	lines := func() []string { return strings.Split(m.View(), "\n") }
	status := func() string { l := lines(); return l[len(l)-1] }
	fg, bg := lipgloss.Color(m.style.FG), lipgloss.Color(m.style.BG)

	if l := status(); !has(l, warnColor) {
		t.Fatalf("no box: the config error is red: %q", l)
	}
	openPrompt(t, m)
	if l := status(); has(l, warnColor) || !has(l, dimmed(warnColor)) {
		t.Errorf("under the box the config error must be its red faded: %q", l)
	}
	board := strings.Join(lines()[:len(lines())-1], "\n")
	if has(board, fg) || !has(board, dimmed(fg)) {
		t.Errorf("under the box the lit pixels must be %s faded, not %s", fg, fg)
	}
	if !has(board, dimmed(bg)) {
		t.Errorf("under the box the ground must be %s faded", bg)
	}
	// Not one grey for all: the lit pixels are not the ground.
	if dimmed(fg) == dimmed(bg) || has(board, dimColor) && !has(board, dimmed(fg)) {
		t.Errorf("the lit pixels and the ground faded into one colour")
	}

	// Esc: the box starts to close, and the lock is itself again at once.
	m.step(keyEsc)
	if l := status(); !has(l, warnColor) {
		t.Errorf("the box closing, the config error is red again: %q", l)
	}
}
