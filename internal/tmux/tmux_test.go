package tmux

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fake puts a tty and a tmux on PATH that answer as the real ones do on
// a client's tty — tmux logging what it was asked — and returns the log.
func fake(t *testing.T, ttyOut, sessionID string) string {
	t.Helper()
	bin := t.TempDir()
	log := filepath.Join(bin, "log")
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write("tty", "echo "+ttyOut+"\n")
	write("tmux", "echo \"$@\" >> "+log+"\ncase \"$*\" in *display-message*) echo '"+sessionID+"';; esac\n")
	t.Setenv("PATH", bin)
	return log
}

func TestSessionIsTheClientsByTTY(t *testing.T) {
	log := fake(t, "/dev/ttys004", "$3")
	if got := Session(""); got != "$3" {
		t.Errorf("session %q", got)
	}
	SetLocked("", "$3", true)
	SetLocked("", "$3", false)
	SetLocked("", "", true)
	b, _ := os.ReadFile(log)
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	want := []string{
		"display-message -p -t /dev/ttys004 #{session_id}",
		"set-option -t $3 @locked 1",
		"set-option -t $3 -u @locked",
	}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Errorf("tmux was asked:\n%s", b)
	}
}

// A socket names the server every call goes to: the one the lock
// command was told, since the client's tmux may not be the default one.
func TestSocketGoesOnEveryCall(t *testing.T) {
	log := fake(t, "/dev/ttys004", "$0")
	if got := Session("/tmp/tmux-1/e2e"); got != "$0" {
		t.Errorf("session %q", got)
	}
	SetLocked("/tmp/tmux-1/e2e", "$0", true)
	b, _ := os.ReadFile(log)
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if !strings.HasPrefix(l, "-S /tmp/tmux-1/e2e ") {
			t.Errorf("without the socket: %q", l)
		}
	}
}

func TestNoTmuxIsNoSession(t *testing.T) {
	// Not a tty at all: tty says "not a tty".
	fake(t, "not a tty", "$3")
	if got := Session(""); got != "" {
		t.Errorf("session %q on no tty", got)
	}
	// A tty, but tmux knows no such client: nothing that looks like an id.
	fake(t, "/dev/ttys004", "no current client")
	if got := Session(""); got != "" {
		t.Errorf("session %q with no client", got)
	}
	// Nothing on PATH.
	t.Setenv("PATH", t.TempDir())
	if got := Session(""); got != "" {
		t.Errorf("session %q without tmux", got)
	}
}
