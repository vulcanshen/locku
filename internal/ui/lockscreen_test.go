package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/crypto/bcrypt"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// A lock at a fixed moment, sized 80×24, with pin set (or none when ""),
// showing a plain medium clock — HH MM in the tall face, no date —
// whatever the saver's defaults are today; the box does not time out but
// where a test says so.
// The status row's user@host is pinned for every test here: the row is
// clipped to the terminal's width, and the machine's own name decided
// whether the words a test looks for were still on it (GitHub's macOS
// runner, 2026-09-25).
func TestMain(m *testing.M) {
	whoami = func() (string, string) { return "user", "host" }
	os.Exit(m.Run())
}

func testLock(t *testing.T, pin string, tweak func(*config.Config)) *LockModel {
	t.Helper()
	// No file: the lock keeps the hash it was made with (the file is
	// read at every key, when there is one).
	t.Setenv("LOCKU__CONFIG", t.TempDir())
	cfg := config.Default()
	cfg.Profiles[0].Size, cfg.Profiles[0].Font, cfg.Profiles[0].Time, cfg.Profiles[0].Date = "medium", "3x7", "HH MM", "off"
	cfg.PINPromptTimeout = 0
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

// drive hands msg to m as Bubble Tea would, through Update, and feeds the
// messages its commands give back in again — the box's animation among
// them — leaving a timer slower than a frame's; it reports whether the
// lock quit.
func drive(m *LockModel, msg tea.Msg) bool {
	_, cmd := m.Update(msg)
	return runCmd(m, cmd)
}

func runCmd(m *LockModel, cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		switch msg := msg.(type) {
		case nil:
		case tea.QuitMsg:
			return true
		case tea.BatchMsg:
			quit := false
			for _, c := range msg {
				quit = runCmd(m, c) || quit
			}
			return quit
		default:
			return drive(m, msg)
		}
	case <-time.After(100 * time.Millisecond):
	}
	return false
}

// press types keys, each a message: "Enter", "Esc", "Backspace" by name,
// anything else as its characters; it reports whether the lock quit.
func press(m *LockModel, keys ...string) bool {
	quit := false
	for _, k := range keys {
		msg := keyRunes(k)
		switch k {
		case "Enter":
			msg = keyEnter
		case "Esc":
			msg = keyEsc
		case "Backspace":
			msg = tea.KeyMsg{Type: tea.KeyBackspace}
		}
		quit = drive(m, msg) || quit
	}
	return quit
}

// typed types pin a character at a time, then Enter.
func typed(m *LockModel, pin string) bool {
	for _, r := range pin {
		press(m, string(r))
	}
	return press(m, "Enter")
}

// screen is the lock as the terminal shows it, without its colours.
func screen(m *LockModel) string { return ansi.Strip(m.View()) }

// dots is how many characters the box holds.
func dots(m *LockModel) int { return strings.Count(screen(m), "●") }

// pastHold lets the second of "wrong" go by on the lock's clock.
func pastHold(m *LockModel) {
	now := m.now()
	m.now = func() time.Time { return now.Add(wrongHold) }
	drive(m, heldOverMsg{})
}

// openPrompt presses a key and lets the box open.
func openPrompt(t *testing.T, m *LockModel) *LockModel {
	t.Helper()
	press(m, "x")
	if v := screen(m); !m.boxUp || !strings.Contains(v, "┏") {
		t.Fatalf("the box did not open:\n%s", v)
	}
	if dots(m) != 0 {
		t.Fatal("the key that opened the box counted as input")
	}
	return m
}

// quits: the command leaves, on its own or as one of a batch — a menu row
// that quits closes the menu in the same batch.
func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	switch msg := cmd().(type) {
	case tea.QuitMsg:
		return true
	case tea.BatchMsg:
		for _, c := range msg {
			if quits(c) {
				return true
			}
		}
	}
	return false
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

// The box is the lib's password box in its unlock look (tdp password.2):
// UNLOCK, no prompt row, seven rows with the frame, as wide as the longest
// PIN's dots and the hint ask, in the middle of the board.
func TestTheBoxIsTheLibs(t *testing.T) {
	for _, icons := range []int{1, 2} {
		withIcons(t, icons)
		m := openPrompt(t, testLock(t, "1234", nil))
		var box []string
		for _, l := range strings.Split(m.View(), "\n") {
			if p := ansi.Strip(l); strings.ContainsAny(p, "┏┃┗") {
				box = append(box, l)
			}
		}
		if len(box) != 7 || !strings.Contains(ansi.Strip(box[6]), "Enter:unlock Backspace:clear Esc:cancel") {
			t.Fatalf("icons %d: the box:\n%s", icons, screen(m))
		}
		at := (80 - pinBoxW) / 2
		for _, r := range []string{"┏", "┃", "┗"} {
			if l := line(m.View(), r); dispW(l[:strings.Index(l, r)]) != at {
				t.Errorf("icons %d: the box's %s is at %d, want %d", icons, r, dispW(l[:strings.Index(l, r)]), at)
			}
		}
	}
}

// pinBoxW is the box's width: the hint, Enter:unlock Backspace:clear
// Esc:cancel, and three cells on either side.
const pinBoxW = 45

func TestWrongThenRight(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", nil))
	if quit := typed(m, "0000"); quit || !strings.Contains(screen(m), "wrong PIN") || dots(m) != 0 || m.failures != 1 {
		t.Fatalf("wrong PIN: quit=%v dots=%d failures=%d\n%s", quit, dots(m), m.failures, screen(m))
	}
	// Everything is swallowed for the second, Esc included.
	press(m, "1", "Esc")
	if dots(m) != 0 || !m.boxUp {
		t.Fatal("input during the wrong second was not swallowed")
	}
	pastHold(m)
	if !typed(m, "1234") {
		t.Error("the right PIN must end the process")
	}
}

// Backspace clears the whole PIN (tdp password.4); a pasted
// control character is refused, and is not a wrong PIN; an Alt
// combination types nothing (tdp input.4).
func TestTheBoxsKeys(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", nil))
	press(m, "1", "2", "Backspace")
	if dots(m) != 0 {
		t.Errorf("Backspace left %d", dots(m))
	}
	drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("12\n34"), Paste: true})
	press(m, "Enter")
	if v := screen(m); !strings.Contains(v, "PIN can't have control characters") || m.failures != 0 {
		t.Errorf("a pasted line break: failures %d\n%s", m.failures, v)
	}
	press(m, "Backspace")
	drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1"), Alt: true})
	if dots(m) != 0 {
		t.Error("Alt-1 typed")
	}
}

// Ctrl-C does not leave the lock, on the board, in the box, or before the
// terminal's size is known: the lock ends by the PIN alone (§1.2). Nor does
// it ask Quit locku?, which the lib asks with a popup open (tdp K9): the
// lock's QuitFlow does nothing.
func TestCtrlCDoesNotLeave(t *testing.T) {
	ctrlC := tea.KeyMsg{Type: tea.KeyCtrlC}
	m := testLock(t, "1234", nil)
	if drive(m, ctrlC) || !m.boxUp {
		t.Fatal("Ctrl-C on the board must open the box, as any key")
	}
	if drive(m, ctrlC) || drive(m, ctrlC) || !m.boxUp {
		t.Error("Ctrl-C in the box left it")
	}
	if strings.Contains(screen(m), "Quit") {
		t.Errorf("Ctrl-C in the box asked to quit:\n%s", screen(m))
	}
	cold := NewLock(m.cfg, "")
	if drive(cold, ctrlC) || drive(cold, ctrlC) {
		t.Error("Ctrl-C before the size left")
	}
}

func TestEscBackToSaverLeavesTheBoardTicking(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", nil))
	gen := m.tickGen
	press(m, "1", "2", "Esc")
	if m.boxUp || strings.Contains(screen(m), "UNLOCK") {
		t.Fatal("Esc did not close the box")
	}
	if m.tickGen != gen {
		t.Error("the board's ticks were restarted on close: they never stopped")
	}
	if openPrompt(t, m); dots(m) != 0 {
		t.Error("input survived Esc")
	}
}

// The cooling-off (§4.4): the box counts it down on its error row, every
// key but Esc swallowed (tdp popup.7); it is there again when the box opens
// again, and once it is over the count starts again, the box taking a PIN.
// On the clock: the lib counts on it.
func TestLockout(t *testing.T) {
	m := testLock(t, "1234", func(c *config.Config) { c.WrongPINAttempts = 2; c.WrongPINCooldown = 1 })
	m.now = time.Now
	openPrompt(t, m)
	typed(m, "0000")
	m.held = time.Time{}
	typed(m, "0000")
	if !m.lockedOut() || !strings.Contains(screen(m), "try again in 1 s") {
		t.Fatalf("no cooling-off after 2 wrong:\n%s", screen(m))
	}
	if typed(m, "1234") || dots(m) == 0 && !m.lockedOut() {
		t.Fatal("a PIN got through the cooling-off")
	}
	press(m, "Esc")
	if openPrompt(t, m); !strings.Contains(screen(m), "try again in") {
		t.Fatalf("opening the box again forgot the cooling-off:\n%s", screen(m))
	}
	time.Sleep(1100 * time.Millisecond)
	drive(m, lockoutOverMsg{})
	if m.failures != 0 || m.lockedOut() || dots(m) != 0 {
		t.Fatalf("the cooling-off did not end: failures %d, dots %d", m.failures, dots(m))
	}
	if !typed(m, "1234") {
		t.Error("the right PIN after the cooling-off must end the process")
	}
}

func TestPromptTimeoutClosesAndKeysRearmIt(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", func(c *config.Config) { c.PINPromptTimeout = 30 }))
	gen := m.idleGen
	press(m, "1")
	if m.idleGen == gen {
		t.Fatal("a key must restart the idle timer")
	}
	// The stale timer is ignored; the live one closes the box.
	drive(m, boxIdleMsg{gen: gen})
	if !m.boxUp {
		t.Fatal("a stale timeout closed the box")
	}
	drive(m, boxIdleMsg{gen: m.idleGen})
	if m.boxUp || strings.Contains(screen(m), "UNLOCK") {
		t.Fatal("the live timeout did not close the box")
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
	if !m.boxUp || !strings.Contains(screen(m), "UNLOCK") {
		t.Error("the prompt went down with the tick")
	}
}

// A dino saver is the run: a frame every DinoFrame, the board replaced
// whole with no reveal, the world moved on.
func TestDinoLockRunsFrameByFrame(t *testing.T) {
	m := testLock(t, "", func(c *config.Config) { c.Profiles[0].Saver = saver.KindRunner })
	if m.game == nil {
		t.Fatal("a dino saver must run the game")
	}
	m.game = saver.NewDino(7, saver.RunnerBig, saver.CharacterTRex, saver.SceneGrass, saver.BackgroundNight, time.Now)
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
	for i := range before.ink {
		if before.ink[i] != m.shown.ink[i] {
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

// A preview is a program of its own, run while the settings screen has
// handed the terminal over: any key ends it, and the settings screen is
// back (ui.md §2.1).
func TestPreviewUnlockHandsBack(t *testing.T) {
	m := newLock(config.Default(), "", true)
	m.now = func() time.Time { return at }
	m, _ = m.step(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, cmd := m.step(keyRunes("x"))
	if !quits(cmd) || !m.unlocked || m.over {
		t.Error("a key must end the preview, and only the preview")
	}
}

func TestViewIsExactlyTheTerminal(t *testing.T) {
	// Once more with icons two cells wide, and with powerline symbols two
	// as well: the board's pixels and the prompt's title, whose capsule's
	// ends are powerline symbols (tdp E5, L4).
	for _, w := range [][2]int{{1, 1}, {2, 1}, {2, 2}} {
		cells := w[0]
		withWidths(t, w[0], w[1])
		for _, sz := range [][2]int{{80, 24}, {81, 25}, {40, 12}, {20, 5}} {
			m := testLock(t, "1234", nil)
			m, _ = m.step(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
			check := func(label string) {
				lines := strings.Split(m.View(), "\n")
				if len(lines) != sz[1] {
					t.Errorf("icons %d, %dx%d %s: %d lines", cells, sz[0], sz[1], label, len(lines))
				}
				for i, l := range lines {
					if w := dispW(l); w != sz[0] {
						t.Errorf("icons %d, %dx%d %s: line %d is %d wide", cells, sz[0], sz[1], label, i, w)
					}
				}
			}
			check("saver")
			m = openPrompt(t, m)
			check("prompt")
		}
	}
}

func TestResizeRedrawsWhole(t *testing.T) {
	// The default saver asks for medium: HH MM at 2 is 36 px, which 80
	// columns (38 px) hold.
	scale := func(m *LockModel) int {
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
	for i := range want.ink {
		if want.ink[i] != m.shown.ink[i] {
			t.Fatal("the board does not show 21:06")
		}
	}
}

// A PIN reset under a lock already up (user, 2026-09-25): the hash is
// read off the file at every key, so the new PIN opens and the old one
// no longer does; a file gone bad meanwhile changes nothing; a file
// with no PIN opens at any key.
func TestLockReadsTheFilesPINAtEveryKey(t *testing.T) {
	cfg := config.Default()
	if err := cfg.SetPIN("9999"); err != nil {
		t.Fatal(err)
	}
	// lock is a lock made with 1234, its file then written by write.
	lock := func(write func(file string) error) *LockModel {
		m := testLock(t, "1234", nil)
		if err := write(filepath.Join(os.Getenv("LOCKU__CONFIG"), "config.yaml")); err != nil {
			t.Fatal(err)
		}
		return m
	}
	reset := func(file string) error { return config.SaveFile(file, cfg) }
	if m := openPrompt(t, lock(reset)); typed(m, "1234") || !strings.Contains(screen(m), "wrong PIN") {
		t.Fatal("the old PIN must not open the lock once the file holds a new one")
	}
	if !typed(openPrompt(t, lock(reset)), "9999") {
		t.Fatal("the new PIN must open the lock")
	}
	// A file gone bad keeps the hash the lock has.
	bad := func(file string) error { return os.WriteFile(file, []byte("pin_hash: [not a hash\n"), 0o600) }
	if !typed(openPrompt(t, lock(bad)), "1234") {
		t.Fatal("a file gone bad must change nothing")
	}
	// A file with no PIN: any key.
	none := func(file string) error { return config.SaveFile(file, config.Default()) }
	if !press(lock(none), "x") {
		t.Fatal("a file with no PIN opens at any key")
	}
}

// The runner's sky is drawn down the whole board (user, 2026-10-07): the
// ground a gradient, the top row the sky's top, the bottom row its
// bottom, each row its own shade, and the runner in its colour; a clock
// keeps its one ground.
func TestTheLockDrawsTheRunnersSky(t *testing.T) {
	colours(t)
	m := testLock(t, "", func(c *config.Config) {
		p := config.NewProfile("r", saver.KindRunner)
		p.Background = saver.BackgroundDay
		c.Profiles, c.Profile = []config.Profile{p}, "r"
	})
	m, _ = m.step(tea.WindowSizeMsg{Width: 152, Height: 32})
	sh := m.shading()
	if sh == nil {
		t.Fatal("no shading")
	}
	ground := sh.Looks[1].Ground
	if len(ground) != 31 || ground[0] == ground[15] || ground[15] == ground[30] {
		t.Fatalf("the ground %v", ground)
	}
	lines := strings.Split(m.View(), "\n")
	for _, y := range []int{0, 15, 30} {
		if !near(fgRe, lines[y], hexRGB(ground[y])) {
			t.Errorf("row %d is not %s: %q", y, ground[y], lines[y])
		}
	}
	if !has(strings.Join(lines[:31], "\n"), "#313244") {
		t.Error("the runner is not in its colour by day")
	}
	if c := testLock(t, "", nil); c.shading() != nil {
		t.Errorf("a clock's shading %v", c.shading())
	}
}
