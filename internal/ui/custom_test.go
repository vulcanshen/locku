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
	if v := m.View(); !strings.Contains(v, "your own program") || !strings.Contains(v, "command") || !strings.Contains(v, "not set") || strings.Contains(v, "layout") || strings.Contains(v, "runner") {
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
	if v := m.press("2").View(); !strings.Contains(v, "cmatrix -b") || strings.Contains(v, "layout") {
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
	if m.preview == nil || m.preview.word != saver.WordError || !strings.Contains(m.View(), "no command") {
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
	if note != "" || cfg.Profiles[0].Command != "cmatrix -b" || cfg.Profiles[0].Layout != "" || cfg.Profiles[1].Command != "" {
		t.Errorf("note %q, profiles %+v", note, cfg.Profiles)
	}
}

// A word on the board (user, 2026-09-25): ERROR or COMPLETED in the
// clock's face at the largest size that fits, the note on the status
// row in red; and the prompt-only lock, up from the first frame, ends
// with Back when the prompt closes and without it for the PIN.
func TestWordLockAndPromptOnly(t *testing.T) {
	cfg := config.Default()
	lk := NewLockWord(cfg, "", "ERROR", "custom saver: exit 3 · boom")
	lk, _ = lk.step(tea.WindowSizeMsg{Width: 120, Height: 40})
	if len(lk.layout.blocks) == 0 || lk.layout.blocks[0].lines[0] != "ERROR" || lk.layout.blocks[0].k != 3 {
		t.Fatalf("the board must spell ERROR at the largest size: %+v", lk.layout)
	}
	if v := lk.View(); !strings.Contains(v, "custom saver: exit 3 · boom") {
		t.Errorf("the note is missing:\n%s", v)
	}
	// Narrower, the word steps down; narrower still, it is drawn as text.
	lk, _ = NewLockWord(cfg, "", "COMPLETED", "n").step(tea.WindowSizeMsg{Width: 100, Height: 20})
	if len(lk.layout.blocks) == 0 || lk.layout.blocks[0].k != 1 {
		t.Errorf("at 100 columns COMPLETED fits small: %+v %v", lk.layout, lk.plain)
	}
	lk, _ = NewLockWord(cfg, "", "COMPLETED", "n").step(tea.WindowSizeMsg{Width: 20, Height: 8})
	if len(lk.layout.blocks) != 0 || len(lk.plain) != 1 || lk.plain[0] != "COMPLETED" {
		t.Errorf("at 20 columns COMPLETED is text: %+v %v", lk.layout, lk.plain)
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
	ground := plainRows(nil, lipgloss.Color(cfg.Profiles[0].BG), lipgloss.Color(cfg.Profiles[0].FG), 100, 29, true)[0]
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
