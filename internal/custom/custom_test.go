package custom

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
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
	if o := ended(t, p); o.Word != "ERROR" || !strings.Contains(o.Note, "killed") {
		t.Errorf("killed: %+v", o)
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("the kill took %v", time.Since(start))
	}
}

// How a program ends is its word and its note: 0 is COMPLETED; a code,
// a missing command with what the shell said, or a signal, ERROR.
func TestOutcomes(t *testing.T) {
	for _, c := range []struct{ cmd, word, note string }{
		{"exit 0", "COMPLETED", "custom saver exited 0"},
		{"echo boom >&2; exit 3", "ERROR", "custom saver: exit 3 · boom"},
		{"no-such-program-locku", "ERROR", "custom saver: exit 127 · "},
		{"kill -SEGV $$", "ERROR", "custom saver: killed: segmentation fault"},
	} {
		p, err := Start(c.cmd, io.Discard, 80, 24)
		if err != nil {
			t.Fatal(err)
		}
		o := ended(t, p)
		if o.Word != c.word || !strings.HasPrefix(o.Note, c.note) {
			t.Errorf("%q: %+v, want %s %q", c.cmd, o, c.word, c.note)
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
	if o := NoCommand(); o.Word != "ERROR" || o.Note != "custom saver: no command" {
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
