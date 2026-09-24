// Package tmux is the little locku says to tmux while it locks: that the
// server is locked — so a client that attaches to any session meanwhile
// is locked too, by the hooks `locku setup tmux` writes (function.md
// §6.2; user, 2026-09-24: the lock is the whole server's, as a
// screensaver is the whole machine's, since a lock on one session is
// walked round by attaching to another). tmux itself has no such state:
// lock-server and lock-session lock the clients attached at that moment.
//
// Nothing here may hold the lock up or bring it down: every call is a
// short command with a timeout, and every failure is silence.
package tmux

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// wait is the most any one call to tmux may take.
const wait = 2 * time.Second

// SetLocked marks the server at socket locked or not: the global @locked
// option, which every session sees and the hooks look at. Nothing
// happens without a socket — a bare tty, screen, or a lock command from
// before the socket was passed.
func SetLocked(socket string, on bool) {
	if socket == "" {
		return
	}
	if on {
		run("tmux", "-S", socket, "set-option", "-g", "@locked", "1")
	} else {
		run("tmux", "-S", socket, "set-option", "-gu", "@locked")
	}
}

// run is a command's trimmed output, or "" for any trouble at all.
func run(name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
