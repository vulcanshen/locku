package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"golang.org/x/crypto/bcrypt"

	"github.com/vulcanshen/locku/internal/config"
)

// customPrompt is a custom saver's prompt as runPrompt starts it, with the
// tries the ones before it handed on, open and taking keys.
func customPrompt(t *testing.T, tries Tries) LockModel {
	t.Helper()
	t.Setenv("LOCKU__CONFIG", t.TempDir())
	cfg := config.Default()
	cfg.WrongPINAttempts, cfg.WrongPINCooldown = 2, 30
	h, err := bcrypt.GenerateFromPassword([]byte("1234"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	cfg.PINHash = string(h)
	m := NewLockPrompt(cfg, "", 80, 24, func(string, int) {}, tries)
	for i := 0; i < animFrames+1 && !m.prompt.anim.isInteractive(); i++ {
		m, _ = m.step(AnimTickMsg{Target: "pinprompt"})
	}
	return m
}

// wrong types a wrong PIN and Enter, and lets the second of "wrong" pass.
func wrong(m LockModel) LockModel {
	m, _ = m.step(keyRunes("0000"))
	m, _ = m.step(keyEnter)
	if m.prompt.state == promptWrong {
		m, _ = m.step(wrongOverMsg{gen: m.promptGen})
	}
	return m
}

// A custom saver's prompts are one lock: the wrong PINs and the
// cooling-off go on from one to the next, as they do on the board, and
// Esc does not start them over (function.md §4.4; terminu, 2026-10-06 —
// each prompt used to start from nothing).
func TestCustomPromptsShareTheTries(t *testing.T) {
	m := wrong(customPrompt(t, Tries{}))
	m, _ = m.step(keyEsc)
	if !m.Back() || m.Tries().failures != 1 {
		t.Fatalf("Esc after one wrong: back %v, tries %+v", m.Back(), m.Tries())
	}
	m = wrong(customPrompt(t, m.Tries()))
	if m.prompt.state != promptLockout {
		t.Fatalf("the second wrong, in the next prompt, is the second: state %v", m.prompt.state)
	}
	m, _ = m.step(keyEsc)
	m = customPrompt(t, m.Tries())
	if v := ansi.Strip(m.prompt.view(m.now())); m.prompt.state != promptLockout || !strings.Contains(v, "try again in 30 s") && !strings.Contains(v, "try again in 29 s") {
		t.Fatalf("Esc and back: the cooling-off goes on:\n%s", v)
	}
	// Run out while no prompt was up: the count starts again.
	tries := m.Tries()
	tries.until = time.Now().Add(-time.Second)
	m = wrong(customPrompt(t, tries))
	if m.prompt.state == promptLockout || m.Tries().failures != 1 {
		t.Errorf("after the cooling-off one wrong is one: state %v, tries %+v", m.prompt.state, m.Tries())
	}
}

// On the board too: a cooling-off that runs out while the prompt is down
// starts the count again, as one the prompt sees out does (function.md
// §4.4). It used to leave the count where it was, so the next wrong PIN
// was a cooling-off again.
func TestCooldownRunsOutWhileThePromptIsDown(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", func(c *config.Config) { c.WrongPINAttempts = 2; c.WrongPINCooldown = 30 }))
	m = wrong(wrong(m))
	if m.prompt.state != promptLockout {
		t.Fatalf("no lockout: %v", m.prompt.state)
	}
	m, _ = m.step(keyEsc)
	for i := 0; i < animFrames+1 && m.prompt.anim.isActive(); i++ {
		m, _ = m.step(AnimTickMsg{Target: "pinprompt"})
	}
	m.now = func() time.Time { return at.Add(31 * time.Second) }
	m = wrong(openPrompt(t, m))
	if m.prompt.state == promptLockout || m.failures != 1 {
		t.Errorf("after the cooling-off one wrong is one: state %v, failures %d", m.prompt.state, m.failures)
	}
	// And a count started after one the prompt saw out is not lost to it.
	m = openPrompt(t, testLock(t, "1234", func(c *config.Config) { c.WrongPINAttempts = 2; c.WrongPINCooldown = 30 }))
	m = wrong(wrong(m))
	m.now = func() time.Time { return at.Add(31 * time.Second) }
	m, _ = m.step(lockoutTickMsg{gen: m.promptGen})
	m = wrong(m)
	m, _ = m.step(keyEsc)
	for i := 0; i < animFrames+1 && m.prompt.anim.isActive(); i++ {
		m, _ = m.step(AnimTickMsg{Target: "pinprompt"})
	}
	if m = wrong(openPrompt(t, m)); m.prompt.state != promptLockout {
		t.Errorf("two wrong since the last cooling-off: state %v, failures %d", m.prompt.state, m.failures)
	}
}
