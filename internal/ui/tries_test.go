package ui

import (
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/vulcanshen/locku/internal/config"
)

// customPrompt is a custom saver's prompt as runPrompt starts it, with the
// tries the ones before it handed on, open and taking keys.
func customPrompt(t *testing.T, tries Tries) *LockModel {
	t.Helper()
	t.Setenv("LOCKU__CONFIG", t.TempDir())
	cfg := config.Default()
	cfg.WrongPINAttempts, cfg.WrongPINCooldown, cfg.PINPromptTimeout = 2, 30, 0
	h, err := bcrypt.GenerateFromPassword([]byte("1234"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	cfg.PINHash = string(h)
	m := NewLockPrompt(cfg, "", 80, 24, func(string, int) {}, tries)
	runCmd(m, m.Init())
	return m
}

// wrong types a wrong PIN and Enter, and lets the second of "wrong" pass.
func wrong(m *LockModel) *LockModel {
	typed(m, "0000")
	if m.holding() {
		pastHold(m)
	}
	return m
}

// A custom saver's prompts are one lock: the wrong PINs and the
// cooling-off go on from one to the next, as they do on the board, and
// Esc does not start them over (function.md §4.4; terminu, 2026-10-06 —
// each prompt used to start from nothing).
func TestCustomPromptsShareTheTries(t *testing.T) {
	m := wrong(customPrompt(t, Tries{}))
	press(m, "Esc")
	if !m.Back() || m.Tries().failures != 1 {
		t.Fatalf("Esc after one wrong: back %v, tries %+v", m.Back(), m.Tries())
	}
	m = wrong(customPrompt(t, m.Tries()))
	if !m.lockedOut() {
		t.Fatalf("the second wrong, in the next prompt, is the second: tries %+v", m.Tries())
	}
	press(m, "Esc")
	m = customPrompt(t, m.Tries())
	if v := screen(m); !m.lockedOut() || !strings.Contains(v, "try again in 30 s") && !strings.Contains(v, "try again in 29 s") {
		t.Fatalf("Esc and back: the cooling-off goes on:\n%s", v)
	}
	// Run out while no prompt was up: the count starts again.
	tries := m.Tries()
	tries.until = time.Now().Add(-time.Second)
	m = wrong(customPrompt(t, tries))
	if m.lockedOut() || m.Tries().failures != 1 {
		t.Errorf("after the cooling-off one wrong is one: tries %+v", m.Tries())
	}
}

// On the board too: a cooling-off that runs out while the box is down
// starts the count again, as one the box sees out does (function.md
// §4.4). It used to leave the count where it was, so the next wrong PIN
// was a cooling-off again.
func TestCooldownRunsOutWhileThePromptIsDown(t *testing.T) {
	cool := func(c *config.Config) { c.WrongPINAttempts = 2; c.WrongPINCooldown = 30 }
	m := openPrompt(t, testLock(t, "1234", cool))
	m = wrong(wrong(m))
	if !m.lockedOut() {
		t.Fatalf("no cooling-off: failures %d", m.failures)
	}
	press(m, "Esc")
	m.now = func() time.Time { return at.Add(31 * time.Second) }
	m = wrong(openPrompt(t, m))
	if m.lockedOut() || m.failures != 1 {
		t.Errorf("after the cooling-off one wrong is one: failures %d", m.failures)
	}
	// And a count started after one the box saw out is not lost to it.
	m = openPrompt(t, testLock(t, "1234", cool))
	m = wrong(wrong(m))
	m.now = func() time.Time { return at.Add(31 * time.Second) }
	drive(m, lockoutOverMsg{})
	press(m, "Esc")
	m = wrong(openPrompt(t, m))
	press(m, "Esc")
	if m = wrong(openPrompt(t, m)); !m.lockedOut() {
		t.Errorf("two wrong since the last cooling-off: failures %d", m.failures)
	}
}
