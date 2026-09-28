package custom

import (
	"os"
	"testing"
	"time"
)

// Key is a key's, not the terminal's answer's: answers are read past,
// and the key after them ends it (2026-09-28).
func TestKeyReadsPastAnswers(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	term := &Terminal{tty: r}
	keys := term.Key()

	w.Write([]byte("\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\\x1b[?62;22c\x1b[>84;0;0c\x1bP>|tmux 3.7c\x1b\\"))
	select {
	case err := <-keys:
		t.Fatalf("tmux's answers ended Key: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	w.Write([]byte("x"))
	select {
	case err := <-keys:
		if err != nil {
			t.Errorf("x: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("x did not end Key")
	}
}
