// Package login checks a password against the user's own login. The
// account is the one boundary locku has (function.md §0.1): whoever can
// run commands as the user can end a lock anyway, so a PIN reset asks
// for the account's password as its confirmation and nothing weaker
// (user, 2026-09-25). A static binary has no PAM, so the check is su's:
// `su <user> -c true` on a pty of ours, the password typed at its
// prompt, its exit status the answer. su takes its time over a wrong
// password; that is as it should be.
package login

import (
	"bytes"
	"os/exec"
	"time"

	"github.com/creack/pty"
)

// Verify reports whether password is user's login password.
func Verify(user, password string) bool {
	cmd := exec.Command("su", user, "-c", "true")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return false
	}
	defer ptmx.Close()
	// su flushes what was typed before its prompt: the password goes in
	// once the prompt is there, and everything su says is read and
	// dropped until it ends.
	prompt := make(chan struct{}, 1)
	go func() {
		buf := make([]byte, 512)
		seen := false
		for {
			n, err := ptmx.Read(buf)
			if n > 0 && !seen && bytes.Contains(bytes.ToLower(buf[:n]), []byte("assword")) {
				seen = true
				prompt <- struct{}{}
			}
			if err != nil {
				return
			}
		}
	}()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-prompt:
		ptmx.Write([]byte(password + "\n"))
	case err := <-done:
		// su ended before asking: an unknown user, or no su at all.
		_ = err
		return false
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		return false
	}
	select {
	case err := <-done:
		return err == nil
	case <-time.After(15 * time.Second):
		cmd.Process.Kill()
		return false
	}
}
