package custom

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vulcanshen/locku/internal/saver"
)

// sink is a writer the pump can write to while the test reads it.
type sink struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *sink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *sink) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		if ok() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("waited in vain for %s", what)
}

func ended(t *testing.T, p *Proxy) Outcome {
	t.Helper()
	select {
	case o := <-p.Done():
		return o
	case <-time.After(5 * time.Second):
		t.Fatal("the program did not end")
		return Outcome{}
	}
}

// The program's output is passed on as it comes, held back while the
// tap is closed — and not replayed — and passed on again; it is killed
// as a group, at once, and that is an ending too.
func TestOutputIsPassedOnAndHeldBack(t *testing.T) {
	out := &sink{}
	p, err := Start("printf ONE; sleep 0.3; printf TWO; sleep 0.4; printf THREE; sleep 30", out, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "ONE", func() bool { return strings.Contains(out.String(), "ONE") })
	p.Forward(false)
	time.Sleep(500 * time.Millisecond)
	if strings.Contains(out.String(), "TWO") {
		t.Fatalf("held back, yet passed on:\n%s", out.String())
	}
	p.Forward(true)
	waitFor(t, "THREE", func() bool { return strings.Contains(out.String(), "THREE") })
	if strings.Contains(out.String(), "TWO") {
		t.Errorf("what was held back must not come later:\n%s", out.String())
	}
	select {
	case <-p.Done():
		t.Fatal("the program ended on its own")
	default:
	}
	start := time.Now()
	p.Kill()
	if o := ended(t, p); o.Word != saver.ExitWord(137) || !strings.Contains(o.Note, "killed") {
		t.Errorf("killed: %+v", o)
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("the kill took %v", time.Since(start))
	}
}

// How a program ends is its word and its note: EXIT and the code as a
// shell would report it — a signal 128 and its number — and what the
// shell said for a missing command.
func TestOutcomes(t *testing.T) {
	for _, c := range []struct {
		cmd  string
		word saver.Word
		note string
	}{
		{"exit 0", saver.ExitWord(0), "custom saver exited 0"},
		{"echo boom >&2; exit 3", saver.ExitWord(3), "custom saver: exit 3 · boom"},
		{"no-such-program-locku", saver.ExitWord(127), "custom saver: exit 127 · "},
		{"kill -SEGV $$", saver.ExitWord(139), "custom saver: killed: segmentation fault"},
	} {
		p, err := Start(c.cmd, io.Discard, 80, 24)
		if err != nil {
			t.Fatal(err)
		}
		o := ended(t, p)
		if o.Word != c.word || !strings.HasPrefix(o.Note, c.note) {
			t.Errorf("%q: %+v, want %q %q", c.cmd, o, c.word, c.note)
		}
		if strings.HasPrefix(c.cmd, "no-such") && !strings.Contains(o.Note, "not found") {
			t.Errorf("%q: the shell's word is missing: %q", c.cmd, o.Note)
		}
		p.Kill()
	}
}

// No command is no program: Start refuses it, and NoCommand is its word.
func TestNoCommand(t *testing.T) {
	if _, err := Start("  ", io.Discard, 80, 24); err == nil {
		t.Error("an empty command started")
	}
	if o := NoCommand(); o.Word != saver.WordNone || o.Note != "custom saver: no command" {
		t.Errorf("%+v", o)
	}
}

// The size given is the pty's, and a resize reaches the program as any
// terminal's would.
func TestSizeReachesTheProgram(t *testing.T) {
	out := &sink{}
	p, err := Start("stty size; trap 'stty size' WINCH; while :; do sleep 0.1; done", out, 100, 40)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Kill()
	waitFor(t, "the first size", func() bool { return strings.Contains(out.String(), "40 100") })
	p.Resize(50, 20)
	waitFor(t, "the new size", func() bool { return strings.Contains(out.String(), "20 50") })
}

// The prompt's box goes over the screen's middle, each line at its own
// place and nothing else touched; Clear blanks exactly that rectangle.
func TestOverlayTouchesOnlyItsBox(t *testing.T) {
	var out bytes.Buffer
	top, left, w, h := paintBox(&out, 80, 24, "+----+\n|\x1b[31m ab \x1b[0m|\n+----+")
	if top != 10 || left != 37 || w != 6 || h != 3 {
		t.Errorf("box at %d,%d %dx%d", top, left, w, h)
	}
	s := out.String()
	for i, want := range []string{"\x1b[11;38H+----+", "\x1b[12;38H|\x1b[31m ab \x1b[0m|", "\x1b[13;38H+----+"} {
		if !strings.Contains(s, want) {
			t.Errorf("line %d not placed: %q", i, s)
		}
	}
	if strings.Contains(s, "\x1b[2J") || strings.Contains(s, "\x1b[?1049") {
		t.Errorf("the overlay must not clear or switch the screen: %q", s)
	}
	out.Reset()
	clearBox(&out, top, left, w, h)
	if c := out.String(); strings.Count(c, "\x1b[0m\x1b[") != 3 || !strings.Contains(c, "\x1b[11;38H      ") || !strings.Contains(c, "\x1b[13;38H      ") {
		t.Errorf("clear must blank the three rows and nothing more: %q", c)
	}
	if top, left, w, h := paintBox(&out, 80, 24, ""); w != 0 || h != 0 || top != 0 || left != 0 {
		t.Errorf("nothing to paint: %d,%d %dx%d", top, left, w, h)
	}
}
