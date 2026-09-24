// Package tmux is the little locku says to tmux while it locks: which
// session its client is on, and that the session is locked — so a client
// that attaches meanwhile is locked too, by the hooks `locku setup tmux`
// writes (function.md §6.2; user, 2026-09-24). tmux itself has no such
// state: lock-session only locks the clients attached at that moment.
//
// Nothing here may hold the lock up or bring it down: every call is a
// short command with a timeout, and every failure is silence.
package tmux

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// wait is the most any one call to tmux or tty may take.
const wait = 2 * time.Second

// Session is the id of the session the terminal on stdin is a client of
// — on the server at socket, or the default one for "" — or "" when it
// is not a tmux client: a bare tty, screen, no tmux to ask. The lock
// command runs in the client's process with no TMUX in its environment,
// and a locked client is not in list-clients; the client's tty is what
// tmux resolves (measured 2026-09-24 on tmux 3.7c). The id, not the
// name: a name with a colon would be read as a window.
func Session(socket string) string {
	tty := run(os.Stdin, "tty")
	if !strings.HasPrefix(tty, "/dev/") {
		return ""
	}
	id := run(nil, "tmux", args(socket, "display-message", "-p", "-t", tty, "#{session_id}")...)
	if !strings.HasPrefix(id, "$") {
		return ""
	}
	return id
}

// SetLocked marks the session locked or not: the @locked option the
// hooks look at. Nothing happens for "".
func SetLocked(socket, session string, on bool) {
	if session == "" {
		return
	}
	if on {
		run(nil, "tmux", args(socket, "set-option", "-t", session, "@locked", "1")...)
	} else {
		run(nil, "tmux", args(socket, "set-option", "-t", session, "-u", "@locked")...)
	}
}

// args is a tmux command line, on the server at socket when there is one.
func args(socket string, rest ...string) []string {
	if socket == "" {
		return rest
	}
	return append([]string{"-S", socket}, rest...)
}

// run is a command's trimmed output, or "" for any trouble at all.
func run(stdin *os.File, name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
