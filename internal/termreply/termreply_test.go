package termreply

import (
	"strings"
	"testing"
)

// keys runs reads through one Filter and joins what came out.
func keys(reads ...string) string {
	var f Filter
	var out []byte
	for _, r := range reads {
		out = append(out, f.Keys([]byte(r))...)
	}
	return string(out)
}

// The answers a terminal gives, each dropped whole, alone or with a key
// on either side of it.
func TestAnswersAreDropped(t *testing.T) {
	answers := map[string]string{
		"background colour, BEL":     "\x1b]11;rgb:1e1e/1e1e/2e2e\x07",
		"background colour, ST":      "\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\",
		"foreground colour":          "\x1b]10;rgb:cdcd/d6d6/f4f4\x1b\\",
		"terminal name (XTVERSION)":  "\x1bP>|iTerm2 3.5.0\x1b\\",
		"DA3":                        "\x1bP!|00000000\x1b\\",
		"a setting (DECRQSS)":        "\x1bP1$r0m\x1b\\",
		"cursor position":            "\x1b[12;40R",
		"device attributes (DA1)":    "\x1b[?62;22;52c",
		"secondary attributes (DA2)": "\x1b[>84;0;0c",
		"a mode's state (DECRPM)":    "\x1b[?2026;2$y",
		"keyboard flags (kitty)":     "\x1b[?0u",
		"status (DSR)":               "\x1b[0n",
		"window size":                "\x1b[8;24;80t",
	}
	for name, a := range answers {
		if got := keys(a); got != "" {
			t.Errorf("%s: %q let through as %q", name, a, got)
		}
		if got := keys("a" + a + "b"); got != "ab" {
			t.Errorf("%s: keys around it came out as %q", name, got)
		}
	}
}

// What tmux 3.7c's attach asks, answered the way the e2e's terminal and a
// real one do, in one read: the bug of 2026-09-28 was this reaching the
// lock as a key.
func TestTmuxAttachAnswersAreDropped(t *testing.T) {
	all := "\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\\x1b[?62;22c\x1b[>84;0;0c\x1bP>|WezTerm 20240203\x1b\\"
	if got := keys(all); got != "" {
		t.Errorf("tmux's answers let through as %q", got)
	}
}

// An answer split anywhere over two reads is still dropped whole.
func TestAnAnswerSplitOverReads(t *testing.T) {
	for _, a := range []string{"\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\", "\x1b[?62;22c", "\x1bP>|tmux 3.7c\x1b\\", "\x1b[12;40R"} {
		// From after the ESC's next byte: a lone ESC at the end of a
		// read is Esc, pressed, and goes out.
		for i := 2; i < len(a); i++ {
			if got := keys(a[:i], a[i:]+"x"); got != "x" {
				t.Errorf("%q split at %d: %q", a, i, got)
			}
		}
	}
}

// Keys come out as they went in.
func TestKeysGoThrough(t *testing.T) {
	for _, k := range []string{
		"x", "1234", "\r", "\x7f", "\t", " ", "你好",
		"\x1b",                // Esc
		"\x1b\x1b",            // Esc, Esc
		"\x1b[A",              // up
		"\x1bOA",              // up, application mode
		"\x1b[1;5C",           // Ctrl-right
		"\x1b[15~",            // F5
		"\x1bOP",              // F1
		"\x1b[Z",              // Shift-Tab
		"\x1ba",               // Alt-a
		"\x1b]a",              // Alt-], then a
		"\x1bPx",              // Alt-P, then x
		"\x1b[200~p\x1b[201~", // a paste
		"\x03",                // Ctrl-C
	} {
		if got := keys(k); got != k {
			t.Errorf("%q came out as %q", k, got)
		}
	}
}

// A key sequence split over two reads comes out whole with the second;
// Esc at the end of a read goes out at once, not with the next key.
func TestKeysOverReads(t *testing.T) {
	var f Filter
	if got := string(f.Keys([]byte("\x1b[1;"))); got != "" || f.Held() != 4 {
		t.Errorf("half a CSI: out %q, held %d", got, f.Held())
	}
	if got := string(f.Keys([]byte("5C"))); got != "\x1b[1;5C" {
		t.Errorf("its second half: %q", got)
	}
	if got := string(f.Keys([]byte("\x1b"))); got != "\x1b" || f.Held() != 0 {
		t.Errorf("Esc at the end of a read: out %q, held %d", got, f.Held())
	}
	// Alt-] and Alt-P at the end of a read wait for the next, which may
	// be the rest of an answer; a key after them lets them out.
	for _, alt := range []string{"\x1b]", "\x1bP"} {
		if got := string(f.Keys([]byte(alt))); got != "" || f.Held() != 2 {
			t.Errorf("%q at the end of a read: out %q, held %d", alt, got, f.Held())
		}
		if got := string(f.Keys([]byte("z"))); got != alt+"z" {
			t.Errorf("%q, then z: %q", alt, got)
		}
	}
}

// An answer that never ends is not a wall: a control key ends it, and
// so does its length.
func TestAnUnendedAnswerGivesWay(t *testing.T) {
	if got := keys("\x1b]1", "\r"); got != "\r" {
		t.Errorf("Enter after an unended OSC: %q", got)
	}
	if got := keys("\x1b]1" + strings.Repeat("a", maxAnswer+10)); !strings.HasSuffix(got, "a") {
		t.Errorf("an OSC past maxAnswer never gave way")
	}
}
