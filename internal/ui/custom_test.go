package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// The custom saver on the settings screen (user, 2026-09-25): a third
// kind, whose profile has a command and no shapes; the command is typed
// in a box, empty is none and said in yellow; a profile made of it has
// the command for its default.
func TestCustomSaverHasACommand(t *testing.T) {
	m := newTestApp(t).press("G", "k", "k", "k") // above the tools: the custom saver
	if it := m.sideAt(); it.kind != sideSaver || saver.Kinds[it.ref] != saver.KindCustom {
		t.Fatalf("the custom saver sits above the tools, not %+v", it)
	}
	if v := m.View(); !strings.Contains(v, "your own program") || !strings.Contains(v, "command") || !strings.Contains(v, "not set") || strings.Contains(v, "layout") || strings.Contains(v, "runner") || strings.Contains(v, " bg ") || strings.Contains(v, " fg ") {
		t.Errorf("the custom saver's [2]:\n%s", v)
	}
	m = m.press("2")
	if m.rowAt().kind != rowCommand {
		t.Fatalf("the first stop is the command, not %+v", m.rowAt())
	}
	m = m.press("enter")
	if m.input.title != "command" || m.input.value != "" {
		t.Fatalf("box %+v", m.input)
	}
	m = m.typed("cmatrix -b").press("enter")
	if m.cfg.Saver(saver.KindCustom).Command != "cmatrix -b" || saved(t).Saver(saver.KindCustom).Command != "cmatrix -b" || !strings.Contains(m.View(), "cmatrix -b") {
		t.Errorf("command %q:\n%s", m.cfg.Saver(saver.KindCustom).Command, m.View())
	}
	// n makes a profile of it, the command its default.
	m = m.press("1", "n", "enter")
	p, ok := m.cfg.Active()
	if it := m.sideAt(); it.kind != sideProfile || m.cfg.Profiles[it.ref].Saver != saver.KindCustom || m.cfg.Profiles[it.ref].Command != "cmatrix -b" {
		t.Fatalf("the new profile: %+v", m.cfg.Profiles)
	}
	_ = p
	_ = ok
	if v := m.press("2").View(); !strings.Contains(v, "cmatrix -b") || strings.Contains(v, "layout") || strings.Contains(v, " bg ") || hotkeyIndex(m.press("2", " ").menu.menuKeys(), "S") >= 0 {
		t.Errorf("the custom profile's rows:\n%s", v)
	}
	// Emptied again, the command is none.
	m = m.press("2", "j", "enter", "ctrl+u", "enter") // past the name row
	if it := m.sideAt(); m.cfg.Profiles[it.ref].Command != "" || !strings.Contains(m.View(), "not set") {
		t.Errorf("emptied: %+v", m.cfg.Profiles[it.ref])
	}
	// A preview of a profile with no command is the word on the board,
	// inside the screen; one with a command hands the terminal to the
	// program — a command for Bubble Tea, not a lock inside the screen.
	m = m.press("P")
	if m.preview == nil || m.preview.word != saver.WordNone || !strings.Contains(m.View(), "no command") {
		t.Fatalf("no command must preview as the word:\n%s", m.View())
	}
	m = m.press("x", "enter").typed("cmatrix").press("enter", "P")
	if m.preview != nil {
		t.Errorf("a command previews outside the screen, not inside it")
	}
}

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
			case !lk.shown.lit[i]:
			case lk.shown.tone[i]:
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
	p := NewLockPrompt(cfg, "")
	if p.initCmd == nil || !p.prompt.anim.isActive() {
		t.Fatal("the prompt must be up from the first frame")
	}
	p, _ = p.step(tea.WindowSizeMsg{Width: 100, Height: 30})
	for i := 0; i < 20 && !p.prompt.anim.isInteractive(); i++ {
		p, _ = p.step(AnimTickMsg{Target: "pinprompt"})
	}
	// The ground is the profile's colour alone, no clock on it: the top
	// row is a bare, dimmed ground row.
	ground := plainRows(nil, lipgloss.Color(config.DefaultBG), lipgloss.Color(config.DefaultFG), 100, 29, true)[0]
	if v := p.View(); !strings.HasPrefix(v, ground) {
		t.Errorf("a prompt-only lock draws no board:\n%s", v)
	}
	esc, cmd := p.step(tea.KeyMsg{Type: tea.KeyEsc})
	if !esc.Back() || !quits(cmd) {
		t.Errorf("Esc must end the prompt program with Back: %v %v", esc.Back(), quits(cmd))
	}
	pin := p
	for _, r := range "1234" {
		pin, _ = pin.step(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	pin, cmd = pin.step(tea.KeyMsg{Type: tea.KeyEnter})
	if pin.Back() || !quits(cmd) {
		t.Errorf("the PIN must end the prompt program without Back: %v %v", pin.Back(), quits(cmd))
	}
}
