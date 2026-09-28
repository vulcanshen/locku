package termreply

import (
	"testing"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/creack/pty"
)

// What was typed before the lock is gone; what is typed after is read.
func TestDropPending(t *testing.T) {
	master, tty, err := pty.Open()
	if err != nil {
		t.Skip("no pty:", err)
	}
	defer master.Close()
	defer tty.Close()
	// Raw, as the lock has it: cooked, a line waits for its newline.
	if _, err := term.MakeRaw(tty.Fd()); err != nil {
		t.Fatal(err)
	}

	// termenv's leftover when an answer to tmux came first: the \ of an
	// ESC \, and the cursor report.
	master.Write([]byte("\\\x1b[1;1R"))
	time.Sleep(50 * time.Millisecond)
	if err := DropPending(tty); err != nil {
		t.Fatal(err)
	}
	master.Write([]byte("x"))
	got := make(chan string, 1)
	go func() {
		buf := make([]byte, 64)
		n, _ := tty.Read(buf)
		got <- string(buf[:n])
	}()
	select {
	case s := <-got:
		if s != "x" {
			t.Errorf("after DropPending read %q; want only x", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("nothing read after DropPending")
	}
}
