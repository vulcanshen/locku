package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/crypto/bcrypt"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// A lock at a fixed moment, sized 80×24, with pin set (or none when "").
func testLock(t *testing.T, pin string, tweak func(*config.Config)) LockModel {
	t.Helper()
	cfg := config.Default()
	if pin != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.MinCost)
		if err != nil {
			t.Fatal(err)
		}
		cfg.PINHash = string(h)
	}
	if tweak != nil {
		tweak(&cfg)
	}
	m := NewLock(cfg, "")
	m.now = func() time.Time { return at }
	m, _ = m.step(tea.WindowSizeMsg{Width: 80, Height: 24})
	return m
}

func keyRunes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

var (
	keyEnter = tea.KeyMsg{Type: tea.KeyEnter}
	keyEsc   = tea.KeyMsg{Type: tea.KeyEsc}
)

// openPrompt presses a key and runs the opening animation through.
func openPrompt(t *testing.T, m LockModel) LockModel {
	t.Helper()
	m, _ = m.step(keyRunes("x"))
	for i := 0; i < animFrames+1 && !m.prompt.anim.isInteractive(); i++ {
		m, _ = m.step(AnimTickMsg{Target: "pinprompt"})
	}
	if !m.prompt.anim.isInteractive() {
		t.Fatal("prompt did not open")
	}
	if len(m.prompt.value) != 0 {
		t.Fatalf("the key that opened the prompt counted as input: %q", string(m.prompt.value))
	}
	return m
}

func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestNoPINAnyKeyEnds(t *testing.T) {
	m := testLock(t, "", nil)
	_, cmd := m.step(keyRunes("q"))
	if !quits(cmd) {
		t.Error("no PIN: any key must end the process")
	}
	if !strings.Contains(m.View(), "no PIN") {
		t.Error("the status row must say there is no PIN")
	}
}

func TestWrongThenRight(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", nil))
	m, _ = m.step(keyRunes("0000"))
	m, cmd := m.step(keyEnter)
	if quits(cmd) || m.prompt.state != promptWrong || len(m.prompt.value) != 0 {
		t.Fatalf("wrong PIN: quit=%v state=%v value=%q", quits(cmd), m.prompt.state, string(m.prompt.value))
	}
	// Everything is swallowed for the second, Esc included.
	m, _ = m.step(keyRunes("1"))
	m, _ = m.step(keyEsc)
	if len(m.prompt.value) != 0 || !m.prompt.anim.owns() {
		t.Fatal("input during the wrong second was not swallowed")
	}
	m, _ = m.step(wrongOverMsg{gen: m.promptGen})
	if m.prompt.state != promptIdle {
		t.Fatal("not back to idle after the second")
	}
	m, _ = m.step(keyRunes("1234"))
	_, cmd = m.step(keyEnter)
	if !quits(cmd) {
		t.Error("the right PIN must end the process")
	}
}

func TestEscBackToSaverLeavesTheBoardTicking(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", nil))
	gen := m.tickGen
	m, _ = m.step(keyRunes("12"))
	m, cmd := m.step(keyEsc)
	if m.prompt.anim.phase != animClosing || cmd == nil {
		t.Fatal("Esc did not close the prompt")
	}
	if m.tickGen != gen {
		t.Error("the board's ticks were restarted on close: they never stopped")
	}
	if len(m.prompt.value) != 0 {
		t.Error("input survived Esc")
	}
}

func TestLockout(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", func(c *config.Config) { c.LockoutAfter = 2; c.LockoutSeconds = 30 }))
	m, _ = m.step(keyRunes("0000"))
	m, _ = m.step(keyEnter)
	m, _ = m.step(wrongOverMsg{gen: m.promptGen})
	m, _ = m.step(keyRunes("0000"))
	m, _ = m.step(keyEnter)
	if m.prompt.state != promptLockout || !m.lockoutUntil.Equal(at.Add(30*time.Second)) {
		t.Fatalf("no lockout after 2 wrong: state=%v until=%v", m.prompt.state, m.lockoutUntil)
	}
	if !strings.Contains(m.View(), "try again in 30 s") {
		t.Errorf("countdown missing:\n%s", m.View())
	}
	// Keys are swallowed; Esc still goes back; the lockout survives a reopen.
	m, _ = m.step(keyRunes("1234"))
	m, cmd := m.step(keyEnter)
	if quits(cmd) || len(m.prompt.value) != 0 {
		t.Fatal("a PIN got through the lockout")
	}
	m, _ = m.step(keyEsc)
	for i := 0; i < animFrames+1 && m.prompt.anim.isActive(); i++ {
		m, _ = m.step(AnimTickMsg{Target: "pinprompt"})
	}
	m = openPrompt(t, m)
	if m.prompt.state != promptLockout {
		t.Fatal("reopening the prompt forgot the lockout")
	}
	// When the clock passes the deadline the count starts over.
	m.now = func() time.Time { return at.Add(31 * time.Second) }
	m, _ = m.step(lockoutTickMsg{gen: m.promptGen})
	if m.prompt.state != promptIdle || m.failures != 0 {
		t.Fatalf("lockout did not end: state=%v failures=%d", m.prompt.state, m.failures)
	}
}

func TestPromptTimeoutClosesAndKeysRearmIt(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", nil))
	gen := m.promptGen
	m, _ = m.step(keyRunes("1"))
	if m.promptGen == gen {
		t.Fatal("a key must restart the idle timer")
	}
	// The stale timer is ignored; the live one closes the prompt.
	m, _ = m.step(promptIdleMsg{gen: gen})
	if !m.prompt.anim.owns() {
		t.Fatal("a stale timeout closed the prompt")
	}
	m, _ = m.step(promptIdleMsg{gen: m.promptGen})
	if m.prompt.anim.phase != animClosing || !m.prompt.timedOut {
		t.Fatal("the live timeout did not close the prompt")
	}
	if !strings.Contains(m.View(), "closing") {
		t.Error("the closing title is missing")
	}
}

// The board goes on under the prompt, dimmed: a tick moves it and the
// prompt stays up (user, 2026-09-24: the colour changes, the clock does
// not stop).
func TestBoardKeepsTickingUnderThePrompt(t *testing.T) {
	m := testLock(t, "1234", nil)
	gen := m.tickGen
	m = openPrompt(t, m)
	if m.tickGen != gen {
		t.Fatal("opening the prompt stopped the board")
	}
	m.now = func() time.Time { return at.Add(time.Minute) }
	m, cmd := m.step(clockTickMsg{gen: gen})
	if cmd == nil || m.rev == nil {
		t.Fatal("the tick under the prompt did not move the board")
	}
	if !m.prompt.anim.owns() || !strings.Contains(m.View(), "PIN") {
		t.Error("the prompt went down with the tick")
	}
}

// A dino saver is the run: a frame every DinoFrame, the board replaced
// whole with no reveal, the world moved on.
func TestDinoLockRunsFrameByFrame(t *testing.T) {
	m := testLock(t, "", func(c *config.Config) { c.Profiles[0].Saver = saver.KindDino })
	if m.game == nil {
		t.Fatal("a dino saver must run the game")
	}
	m.game = saver.NewDino(7, saver.RunnerTRex, saver.SceneGrass)
	m, _ = m.step(tea.WindowSizeMsg{Width: 152, Height: 32})
	if m.shown.w != 76 || m.shown.h != 31 || m.shown.count() == 0 {
		t.Fatalf("board %dx%d, %d lit", m.shown.w, m.shown.h, m.shown.count())
	}
	before := m.shown.clone()
	m, cmd := m.step(clockTickMsg{gen: m.tickGen})
	if cmd == nil || m.rev != nil {
		t.Fatal("a frame must schedule the next and never reveal")
	}
	moved := false
	for i := range before.lit {
		if before.lit[i] != m.shown.lit[i] {
			moved = true
			break
		}
	}
	if !moved {
		t.Error("the frame did not move the world")
	}
	if len(strings.Split(m.View(), "\n")) != 32 {
		t.Error("the view is not the terminal")
	}
}

func TestPreviewUnlockHandsBack(t *testing.T) {
	m := newLock(config.Default(), "", true)
	m.now = func() time.Time { return at }
	m, _ = m.step(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, cmd := m.step(keyRunes("x"))
	if quits(cmd) || !m.unlocked {
		t.Error("a preview must hand back, not quit")
	}
}

func TestViewIsExactlyTheTerminal(t *testing.T) {
	for _, sz := range [][2]int{{80, 24}, {81, 25}, {40, 12}, {20, 5}} {
		m := testLock(t, "1234", nil)
		m, _ = m.step(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		check := func(label string) {
			lines := strings.Split(m.View(), "\n")
			if len(lines) != sz[1] {
				t.Errorf("%dx%d %s: %d lines", sz[0], sz[1], label, len(lines))
			}
			for i, l := range lines {
				if w := lipgloss.Width(l); w != sz[0] {
					t.Errorf("%dx%d %s: line %d is %d wide", sz[0], sz[1], label, i, w)
				}
			}
		}
		check("saver")
		m = openPrompt(t, m)
		check("prompt")
	}
}

func TestResizeRedrawsWhole(t *testing.T) {
	// The default saver asks for medium: HH MM at 2 is 36 px, which 80
	// columns (38 px) hold.
	scale := func(m LockModel) int {
		if len(m.layout.blocks) == 0 {
			return 0
		}
		return m.layout.blocks[0].k
	}
	m := testLock(t, "1234", nil)
	if scale(m) != 2 {
		t.Fatalf("80x24 k=%d", scale(m))
	}
	m, _ = m.step(tea.WindowSizeMsg{Width: 200, Height: 60})
	if scale(m) != 2 || m.rev != nil || m.shown.w != 100 || m.shown.h != 59 {
		t.Errorf("after resize: k=%d rev=%v board %dx%d", scale(m), m.rev != nil, m.shown.w, m.shown.h)
	}
	m, _ = m.step(tea.WindowSizeMsg{Width: 40, Height: 12})
	if scale(m) != 1 {
		t.Errorf("40x12 holds HH MM at 1: k=%d", scale(m))
	}
	m, _ = m.step(tea.WindowSizeMsg{Width: 30, Height: 8})
	if scale(m) != 0 || !strings.Contains(m.View(), "21 05") {
		t.Errorf("30x8 must fall back to plain text: k=%d", scale(m))
	}
	// A large saver gets 3 there.
	big := testLock(t, "1234", func(c *config.Config) { c.Profiles[0].Size = "large" })
	big, _ = big.step(tea.WindowSizeMsg{Width: 200, Height: 60})
	if scale(big) != 3 {
		t.Errorf("large at 200x60: k=%d", scale(big))
	}
}

func TestTickRevealsOnlyTheChange(t *testing.T) {
	m := testLock(t, "1234", nil)
	m, _ = m.step(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.now = func() time.Time { return at.Add(time.Minute) } // 21:06
	m, cmd := m.step(clockTickMsg{gen: m.tickGen})
	if m.rev == nil || cmd == nil {
		t.Fatal("a minute change must start a reveal")
	}
	for i := 0; i <= revealFrames && m.rev != nil; i++ {
		m, _ = m.step(revealTickMsg{gen: m.tickGen})
	}
	if m.rev != nil {
		t.Fatal("reveal never finished")
	}
	want := paint(faceTall, one([]string{"21 06"}, 2), 120, 39)
	for i := range want.lit {
		if want.lit[i] != m.shown.lit[i] {
			t.Fatal("the board does not show 21:06")
		}
	}
}
