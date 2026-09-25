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

// SetLocked marks the server at socket — or, given a session id, that
// session alone — locked or not: the @locked option the hooks look at,
// global or the session's. The hooks read a client's session before the
// global (measured 2026-09-25), so where the mark is IS the lock's
// scope; the session's id comes in on -t, baked into that session's own
// lock-command by setup, since a locked client cannot look its session
// up (setup.sessionLockCmd). Nothing happens without a socket — a bare
// tty, screen, or a lock command from before the socket was passed.
func SetLocked(socket, session string, on bool) {
	if socket == "" {
		return
	}
	switch {
	case on && session != "":
		run("tmux", "-S", socket, "set-option", "-t", session, "@locked", "1")
	case on:
		run("tmux", "-S", socket, "set-option", "-g", "@locked", "1")
	case session != "":
		run("tmux", "-S", socket, "set-option", "-u", "-t", session, "@locked")
	default:
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
