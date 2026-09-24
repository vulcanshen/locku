package tmux

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fake puts a tmux on PATH that logs what it was asked, and returns the log.
func fake(t *testing.T) string {
	t.Helper()
	bin := t.TempDir()
	log := filepath.Join(bin, "log")
	if err := os.WriteFile(filepath.Join(bin, "tmux"), []byte("#!/bin/sh\necho \"$@\" >> "+log+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	return log
}

// The mark is the server's — global, on the socket the lock was told —
// set and unset; without a socket nothing is said at all.
func TestSetLockedIsGlobalOnTheSocket(t *testing.T) {
	log := fake(t)
	SetLocked("/tmp/tmux-1/e2e", true)
	SetLocked("/tmp/tmux-1/e2e", false)
	SetLocked("", true)
	SetLocked("", false)
	b, _ := os.ReadFile(log)
	want := "-S /tmp/tmux-1/e2e set-option -g @locked 1\n-S /tmp/tmux-1/e2e set-option -gu @locked\n"
	if string(b) != want {
		t.Errorf("tmux was asked:\n%s", b)
	}
}

// No tmux on PATH is no trouble: silence.
func TestNoTmuxIsSilence(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	SetLocked("/tmp/tmux-1/e2e", true)
	if got := run("tmux", "list-sessions"); !strings.HasPrefix(got, "") || got != "" {
		t.Errorf("run without tmux: %q", got)
	}
}
