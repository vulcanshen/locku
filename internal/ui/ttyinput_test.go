package ui

import (
	"os"
	"testing"
)

// The lock's input hands on keys only: what tmux's attach asks, answered
// by the terminal, reads as nothing, and the key after it as itself
// (2026-09-28: the answer used to reach the lock as a key, and unlock).
func TestLockInputReadsPastAnswers(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	in := LockInput(r, func() {})
	buf := make([]byte, 256)

	w.Write([]byte("\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\\x1b[?62;22c\x1b[>84;0;0c"))
	if n, err := in.Read(buf); n != 0 || err != nil {
		t.Errorf("tmux's answers read as %q, %v", buf[:n], err)
	}
	w.Write([]byte("\x1bP>|tmux 3.7c\x1b\\x"))
	if n, err := in.Read(buf); string(buf[:n]) != "x" || err != nil {
		t.Errorf("an answer, then x: %q, %v", buf[:n], err)
	}
	// A CSI split over two reads comes out whole, and fits.
	w.Write([]byte("\x1b[1;"))
	if n, _ := in.Read(buf); n != 0 {
		t.Errorf("half a CSI read as %q", buf[:n])
	}
	w.Write([]byte("5C"))
	if n, _ := in.Read(buf); string(buf[:n]) != "\x1b[1;5C" {
		t.Errorf("its second half: %q", buf[:n])
	}
}
