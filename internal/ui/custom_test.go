package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// A profile's command is its own: the file keeps it, and only for the
// custom kind.
func TestCommandIsTheCustomKindsAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("profile: c\nprofiles:\n  - name: c\n    saver: custom\n    command: \"  cmatrix -b  \"\n  - name: k\n    saver: clock\n    command: nope\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, note := config.LoadFile(path)
	if note != "" || cfg.Profiles[0].Command != "cmatrix -b" || cfg.Profiles[0].Layout != "" || cfg.Profiles[0].BG != "" || cfg.Profiles[1].Command != "" {
		t.Errorf("note %q, profiles %+v", note, cfg.Profiles)
	}
	// Saved, a custom profile has no colours (user, 2026-09-25).
	if err := config.SaveFile(path, cfg); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	s := string(b)
	i, j := strings.Index(s, "saver: custom"), strings.Index(s, "- name: k")
	if i < 0 || j < i || strings.Contains(s[i:j], "bg:") || strings.Contains(s[i:j], "fg:") || !strings.Contains(s[i:j], "command: cmatrix -b") {
		t.Errorf("saved:\n%s", s)
	}
}

// A word on the board (user, 2026-09-25): EXIT and the code in the
// clock's face at the largest size that fits — EXIT in the gold, the
// code green for 0 and peach otherwise, NONE red — the note on the status row in
// red; and the prompt-only lock, up from the first frame, ends with
// Back when the prompt closes and without it for the PIN.
func TestWordLockAndPromptOnly(t *testing.T) {
	t.Setenv("LOCKU__CONFIG", t.TempDir()) // no file: the lock keeps its hash
	cfg := config.Default()
	lk := NewLockWord(cfg, "", saver.ExitWord(3), "custom saver: exit 3 · boom")
	lk, _ = lk.step(tea.WindowSizeMsg{Width: 160, Height: 40})
	if len(lk.layout.blocks) == 0 || lk.layout.blocks[0].lines[0] != "EXIT 3" || lk.layout.blocks[0].k != 3 || lk.style.FG != config.DefaultFG || lk.style.BG != config.DefaultBG || lk.accent != peachColor {
		t.Fatalf("the board must spell EXIT 3 at the largest size, gold with a peach code: %+v %+v %v", lk.layout, lk.style, lk.accent)
	}
	// The letters wear the gold, the code the accent: every accented
	// pixel lies right of every plain lit one.
	plainMax, accMin := -1, lk.shown.w
	for y := 0; y < lk.shown.h; y++ {
		for x := 0; x < lk.shown.w; x++ {
			i := y*lk.shown.w + x
			switch {
			case lk.shown.ink[i] == inkOff:
			case lk.shown.ink[i] == inkAccent:
				accMin = min(accMin, x)
			default:
				plainMax = max(plainMax, x)
			}
		}
	}
	if plainMax < 0 || accMin == lk.shown.w || accMin <= plainMax {
		t.Errorf("the code must be the accented part, right of EXIT: plain to %d, accent from %d", plainMax, accMin)
	}
	if v := lk.View(); !strings.Contains(v, "custom saver: exit 3 · boom") {
		t.Errorf("the note is missing:\n%s", v)
	}
	// EXIT 0's code is green; narrower, the word steps down; narrower
	// still, it is drawn as text. NONE is red, all of it.
	lk, _ = NewLockWord(cfg, "", saver.ExitWord(0), "n").step(tea.WindowSizeMsg{Width: 60, Height: 20})
	if len(lk.layout.blocks) == 0 || lk.layout.blocks[0].k != 1 || lk.accent != liveColor || lk.style.FG != config.DefaultFG {
		t.Errorf("at 60 columns EXIT 0 fits small, gold with a green code: %+v %v %+v", lk.layout, lk.plain, lk.style)
	}
	lk, _ = NewLockWord(cfg, "", saver.ExitWord(0), "n").step(tea.WindowSizeMsg{Width: 20, Height: 8})
	if len(lk.layout.blocks) != 0 || len(lk.plain) != 1 || lk.plain[0] != "EXIT 0" {
		t.Errorf("at 20 columns EXIT 0 is text: %+v %v", lk.layout, lk.plain)
	}
	if lk := NewLockWord(cfg, "", saver.WordNone, "custom saver: no command"); lk.style.FG != string(warnColor) || lk.accentFrom != 0 {
		t.Errorf("NONE is red, all of it: %+v %d", lk.style, lk.accentFrom)
	}

	if err := cfg.SetPIN("1234"); err != nil {
		t.Fatal(err)
	}
	var painted []string
	prompt := func() *LockModel {
		painted = nil
		return NewLockPrompt(cfg, "", 100, 30, func(box string, _ int) { painted = append(painted, box) }, Tries{})
	}
	p := prompt()
	if p.initCmd == nil || !p.boxUp || p.width != 100 {
		t.Fatal("the box must be up from the first frame, at the size given")
	}
	// Every step paints the box, through the callback: the animation's
	// frames, and the keys; the box alone, nothing of the lock under it.
	runCmd(p, p.Init())
	last := func() string { return ansi.Strip(painted[len(painted)-1]) }
	if len(painted) == 0 || !strings.Contains(painted[len(painted)-1], glyphLock) || !strings.Contains(last(), "UNLOCK") {
		t.Fatalf("the box must be painted through the callback: %d paints", len(painted))
	}
	if b := strings.Split(last(), "\n"); len(b) != 7 || !strings.HasPrefix(b[0], "┏") || !strings.HasPrefix(b[6], "┗") {
		t.Errorf("the box, whole and alone:\n%s", last())
	}
	if v := ansi.Strip(p.View()); strings.Contains(v, "user@host") || strings.Contains(v, "no PIN") {
		t.Errorf("a prompt-only lock draws no lock under its box:\n%s", v)
	}
	n := len(painted)
	if !press(p, "Esc") || len(painted) != n || !p.Back() {
		t.Errorf("Esc ends the prompt program with Back, without painting again: %d paints, back %v", len(painted)-n, p.Back())
	}
	pin := prompt()
	runCmd(pin, pin.Init())
	if !typed(pin, "1234") || pin.Back() {
		t.Errorf("the PIN must end the prompt program without Back: %v", pin.Back())
	}
}
