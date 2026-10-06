package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/vulcanshen/locku/internal/config"
)

// paste is what Bubble Tea makes of a bracketed paste: every character
// in one run, control characters and all.
func paste(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s), Paste: true}
}

// A line break or a tab is kept as it came; any other control character
// is dropped (terminu, 2026-10-06).
func TestTakeText(t *testing.T) {
	for in, want := range map[string]string{
		"abc":              "abc",
		"a\nb":             "a\nb",
		"a\r\nb":           "a\r\nb",
		"a\rb\tc":          "a\rb\tc",
		"a\x1b[31mb":       "a[31mb",
		"\x00a\x07b\x7fc":  "abc",
		"a\u0085b\u009bc":  "abc",
		"時鐘 ✓\u200b":       "時鐘 ✓\u200b",
		"\x1b\x01\x02\x03": "",
	} {
		if got := takeText([]rune(in)); got != want {
			t.Errorf("takeText(%q) = %q, want %q", in, got, want)
		}
	}
}

// "\r\n" is one line break: one dot, one `\n`, one Backspace.
func TestValueUnits(t *testing.T) {
	for in, want := range map[string]int{"": 0, "ab": 2, "a\r\nb": 3, "a\n\rb": 4, "\r\r\n": 2, "時\t": 2} {
		if got := valueLen(in); got != want {
			t.Errorf("valueLen(%q) = %d, want %d", in, got, want)
		}
	}
	for in, want := range map[string]string{"": "", "ab": "a", "a\r\n": "a", "a\n": "a", "a\r": "a", "a\t": "a", "a時": "a"} {
		if got := dropLast(in); got != want {
			t.Errorf("dropLast(%q) = %q, want %q", in, got, want)
		}
	}
}

// A line break or a tab is drawn `\n` / `\t` in red, two cells that a
// cut never splits; the rest in the value's colour.
func TestValueView(t *testing.T) {
	colours(t)
	for _, c := range []struct {
		in    string
		w     int
		shown string
	}{
		{"a\tb", 10, `a\tb`},
		{"a\r\nb", 10, `a\nb`},
		{"a\rb\nc", 10, `a\nb\nc`},
		{"x\ny", 3, "… y"},
		{"xy\n", 3, `…\n`},
		{"xy\n", 4, `xy\n`},
		{"abcdef", 4, "…def"},
	} {
		got, w := valueView(c.in, c.w)
		if ansi.Strip(got) != c.shown || w != dispW(c.shown) || w > c.w {
			t.Errorf("valueView(%q, %d) = %q (%d wide), want %q", c.in, c.w, ansi.Strip(got), w, c.shown)
		}
	}
	got, _ := valueView("ab\tc", 20)
	if !has(fgBefore(got, `\t`), warnColor) || !has(fgBefore(got, "ab"), editColor) || !has(fgBefore(got, "c"), editColor) {
		t.Errorf("`\\t` red, the rest lavender: %q", got)
	}
}

// Settings: a pasted line break stays in the value and is drawn, the row
// unbroken; Enter is refused, saying so, and nothing is written.
func TestSettingsBoxTakesALineBreakButNotTheValue(t *testing.T) {
	m := newTestApp(t).press("r", "ctrl+u")
	rows := strings.Count(m.View(), "\n")
	mm, _ := m.Update(paste("ma\x1bin\r\nx"))
	m = mm.(AppModel)
	if m.input.value != "main\r\nx" {
		t.Fatalf("value %q", m.input.value)
	}
	if v := ansi.Strip(m.View()); !strings.Contains(v, `main\nx`) || strings.Count(m.View(), "\n") != rows {
		t.Errorf("the line break drawn in the row, the box as tall:\n%s", v)
	}
	m = m.press("enter")
	if m.input.err != "name can't have line breaks or tabs" || !m.input.anim.owns() || m.cfg.Profiles[0].Name != "clock" {
		t.Fatalf("Enter with a line break: err %q, open %v, name %q", m.input.err, m.input.anim.owns(), m.cfg.Profiles[0].Name)
	}
	// Backspace takes the x, then the line break whole.
	m = m.press("backspace", "backspace")
	if m.input.value != "main" || m.input.err != "" {
		t.Fatalf("after two Backspaces: %q, err %q", m.input.value, m.input.err)
	}
	if m = m.press("enter"); m.cfg.Profiles[0].Name != "main" {
		t.Errorf("the name without it: %q", m.cfg.Profiles[0].Name)
	}
}

// Every box that takes a value refuses one with a line break or a tab,
// before anything else: a current PIN is not checked, so not frozen.
func TestEveryBoxRefusesALineBreakOrATab(t *testing.T) {
	for _, c := range []struct {
		action inputAction
		title  string
		masked bool
		err    string
	}{
		{inputNew, "name", false, "name"},
		{inputRename, "name", false, "name"},
		{inputDuplicate, "name", false, "name"},
		{inputNumber, "number", false, "number"},
		{inputPath, "path", false, "path"},
		{inputBindKey, "key", false, "key"},
		{inputCommand, "command", false, "command"},
		{inputPINCurrent, "current PIN", true, "PIN"},
		{inputPINNew, "new PIN", true, "PIN"},
		{inputPINConfirm, "confirm PIN", true, "PIN"},
	} {
		for _, v := range []string{"12\n34", "12\t34", "1234\r"} {
			m := newTestApp(t)
			before := m.cfg
			m.editRef = 0
			m.pinNew = v
			m.input = inputPopup{anim: popupAnimator{phase: animOpen, target: "input"}, title: c.title, masked: c.masked, action: c.action, value: v}
			m.commitInput(&m.input)
			if want := c.err + " can't have line breaks or tabs"; m.input.err != want {
				t.Errorf("%s %q: err %q, want %q", c.title, v, m.input.err, want)
			}
			if m.input.frozen || m.pinConfirm.anim.owns() || m.cfg.PINHash != before.PINHash || m.cfg.Profiles[0].Name != before.Profiles[0].Name {
				t.Errorf("%s %q: taken anyway", c.title, v)
			}
		}
	}
}

// new PIN, pasted `12\r\n34`: five dots, refused, never saved — a PIN the
// lock could not take (terminu, 2026-10-06).
func TestNewPINWithALineBreak(t *testing.T) {
	m := newTestApp(t).press("G", "2", "enter")
	mm, _ := m.Update(paste("12\r\n34"))
	m = mm.(AppModel)
	if n := strings.Count(ansi.Strip(m.input.view()), "●"); n != 5 {
		t.Errorf("%d dots for 12\\r\\n34, want 5", n)
	}
	if v := ansi.Strip(m.input.view()); strings.Contains(v, `\n`) {
		t.Errorf("a PIN box shows the line break:\n%s", v)
	}
	m = m.press("enter")
	if m.input.err != "PIN can't have line breaks or tabs" || m.pinConfirm.anim.owns() || m.cfg.HasPIN() {
		t.Errorf("new PIN 12\\r\\n34: err %q, confirm %v, set %v", m.input.err, m.pinConfirm.anim.owns(), m.cfg.HasPIN())
	}
}

// The lock: a pasted line break goes into the PIN as one dot, and Enter
// says why it is not taken, without checking it or counting a wrong one;
// another control character is dropped.
func TestLockPINWithALineBreak(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", func(c *config.Config) { c.WrongPINAttempts = 1 }))
	m, _ = m.step(paste("12\r\n3\x1b4"))
	if string(m.prompt.value) != "12\r\n34" {
		t.Fatalf("value %q", string(m.prompt.value))
	}
	if n := strings.Count(ansi.Strip(m.prompt.view(at)), "●"); n != 5 {
		t.Errorf("%d dots, want 5", n)
	}
	m, _ = m.step(keyEnter)
	if m.over || m.failures != 0 || m.prompt.state != promptIdle || m.prompt.err != "PIN can't have line breaks or tabs" {
		t.Fatalf("Enter: over %v, failures %d, state %v, err %q", m.over, m.failures, m.prompt.state, m.prompt.err)
	}
	inside(t, "line break", m.prompt.view(at), "PIN can't have line breaks or tabs")
	// The next key clears it; Backspace takes 4, 3, then the line break.
	m, _ = m.step(tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = m.step(tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = m.step(tea.KeyMsg{Type: tea.KeyBackspace})
	if string(m.prompt.value) != "12" || m.prompt.err != "" {
		t.Fatalf("after three Backspaces: %q, err %q", string(m.prompt.value), m.prompt.err)
	}
	m, _ = m.step(keyRunes("34"))
	if _, cmd := m.step(keyEnter); !quits(cmd) {
		t.Error("the PIN without it must unlock")
	}
}
