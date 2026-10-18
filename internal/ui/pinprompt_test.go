package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// boxText is the prompt as its frame draws it now, its rows joined.
func boxText(p pinPrompt, now time.Time) string {
	rows, _, _ := p.view(now)
	return strings.Join(rows, "\n")
}

// pinBoxW is the lock's PIN box on a screen wide enough: the whole of its
// hint, Enter:unlock Backspace:clear Esc:cancel, and the frame's six cells
// round it (components/layout/popup, width).
const pinBoxW = 39 + 6

// The PIN box is the lib's password box in its unlock look: the title's
// capsule, the prompt, the dots in the middle, the error row, the whole
// hint, nine rows (components/input/password, input/README).
func TestPINBoxIsTheLibs(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", nil))
	for _, r := range "12" {
		m, _ = m.step(keyRunes(string(r)))
	}
	box := strings.Split(ansi.Strip(boxText(m.prompt, m.now())), "\n")
	if len(box) != 9 {
		t.Fatalf("%d rows:\n%s", len(box), strings.Join(box, "\n"))
	}
	for r, l := range box {
		if w := dispW(l); w != pinBoxW {
			t.Errorf("row %d is %d wide, want %d: %q", r, w, pinBoxW, l)
		}
	}
	if !strings.Contains(box[0], "UNLOCK") || !strings.Contains(box[2], "The PIN") {
		t.Errorf("the title and the prompt:\n%s", strings.Join(box, "\n"))
	}
	if got := strings.Count(box[4], "●"); got != 2 {
		t.Errorf("%d dots for two characters: %q", got, box[4])
	}
	if !strings.Contains(box[8], "Enter:unlock Backspace:clear Esc:cancel") {
		t.Errorf("the hint: %q", box[8])
	}
	// The dots sit in the middle, the cursor counted: "● ● █" is five
	// cells, as far from either side.
	row := strings.TrimSuffix(strings.TrimPrefix(box[4], "┃"), "┃")
	left := len(row) - len(strings.TrimLeft(row, " "))
	right := len([]rune(row)) - len([]rune(strings.TrimRight(row, " "))) - 1 // the cursor is a space
	if d := left - right; d < -1 || d > 1 {
		t.Errorf("the dots are %d left, %d right: %q", left, right, box[4])
	}
}

// Backspace clears the whole PIN, not a character of it
// (components/input/password).
func TestBackspaceClearsThePIN(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", nil))
	for _, r := range "123" {
		m, _ = m.step(keyRunes(string(r)))
	}
	m, _ = m.step(tea.KeyMsg{Type: tea.KeyBackspace})
	if len(m.prompt.value) != 0 {
		t.Errorf("Backspace left %q", string(m.prompt.value))
	}
}

// A PIN with a control character pasted into it is not checked, and not
// a wrong one: the error row names the value (components/input/README),
// the dots stay, and the error stays till the next Enter.
func TestLockPINWithAControlCharacter(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", nil))
	m, _ = m.step(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("12\n34"), Paste: true})
	at := m.now()
	if n := strings.Count(ansi.Strip(boxText(m.prompt, at)), "●"); n != 5 {
		t.Fatalf("%d dots for a five-character paste", n)
	}
	m, _ = m.step(tea.KeyMsg{Type: tea.KeyEnter})
	if m.failures != 0 || m.prompt.state != promptIdle {
		t.Errorf("a control character counted as a wrong PIN: %d, state %v", m.failures, m.prompt.state)
	}
	if v := ansi.Strip(boxText(m.prompt, at)); !strings.Contains(v, "PIN can't have control characters") {
		t.Errorf("the error row:\n%s", v)
	}
	m, _ = m.step(keyRunes("5"))
	if v := ansi.Strip(boxText(m.prompt, at)); !strings.Contains(v, "can't have control characters") {
		t.Errorf("the error went before the next Enter:\n%s", v)
	}
	// A key typed with Ctrl puts nothing in.
	m, _ = m.step(tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = m.step(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\x07'}})
	if len(m.prompt.value) != 0 {
		t.Errorf("a typed control character went in: %q", string(m.prompt.value))
	}
}

// The box opens as the family's popups do: a line from the middle out,
// held, then opened up and down, nine frames (tdp F2).
func TestPromptOpensWithTheFamilyAnimation(t *testing.T) {
	m := testLock(t, "1234", nil)
	m, _ = m.step(keyRunes("x"))
	var heights []int
	for range openFrames {
		rows, _, _ := m.prompt.view(m.now())
		heights = append(heights, len(rows))
		m, _ = m.step(AnimTickMsg{})
	}
	want := []int{1, 1, 1, 1, 1, 3, 5, 7, 9}
	for i := range want {
		if heights[i] != want[i] {
			t.Fatalf("the rows frame by frame: %v, want %v", heights, want)
		}
	}
	if !m.prompt.anim.isInteractive() {
		t.Error("open after nine frames")
	}
}

// An Alt combination types nothing on the PIN box, as on every typing row
// (components/input/README).
func TestAltTypesNothing(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", nil))
	m, _ = m.step(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b"), Alt: true})
	if len(m.prompt.value) != 0 {
		t.Errorf("Alt-b typed %q", string(m.prompt.value))
	}
}

// The board is dimmed while the box opens and is open, and comes back the
// moment it starts to close: a box closing is not the top any more
// (components/layout/popup).
func TestBoardBrightensAsTheBoxCloses(t *testing.T) {
	m := openPrompt(t, testLock(t, "1234", nil))
	open := m.View()
	m, _ = m.step(tea.KeyMsg{Type: tea.KeyEsc})
	if m.prompt.anim.phase != animClosing {
		t.Fatal("Esc did not start the close")
	}
	closing := m.View()
	m2 := testLock(t, "1234", nil)
	plain := m2.View()
	last := func(v string) string { lines := strings.Split(v, "\n"); return lines[len(lines)-1] }
	if last(open) == last(plain) {
		t.Fatal("the status row is not dimmed under the open box")
	}
	if last(closing) != last(plain) {
		t.Errorf("the status row is still dimmed while the box closes:\n%q\n%q", last(closing), last(plain))
	}
}
