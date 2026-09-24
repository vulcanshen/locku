// Package setup is `locku setup`: it writes the lock into tmux's and
// screen's configuration (function.md §6.2), and takes it out again
// with -d. Only a managed block is ever touched — between two marker
// lines, every line of it marked `# locku` so it reads as locku's when
// met on its own — so running it again replaces the block and nothing
// else, and a hand-written file keeps every other line.
package setup

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vulcanshen/locku/internal/config"
)

const (
	blockBegin = "# >>> locku >>>"
	blockEnd   = "# <<< locku <<<"
)

// Apply returns content with the managed block set to lines: replaced
// where one is, appended where there is none. It is idempotent.
func Apply(content string, lines []string) string {
	block := blockBegin + "\n" + strings.Join(lines, "\n") + "\n" + blockEnd + "\n"
	if i := strings.Index(content, blockBegin); i >= 0 {
		if j := strings.Index(content[i:], blockEnd); j >= 0 {
			after := content[i+j+len(blockEnd):]
			after = strings.TrimPrefix(after, "\n")
			return content[:i] + block + after
		}
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if content != "" {
		content += "\n"
	}
	return content + block
}

// Remove returns content without the managed block, and without the
// blank line that was put before it; content with no block comes back
// as it is.
func Remove(content string) string {
	i := strings.Index(content, blockBegin)
	if i < 0 {
		return content
	}
	j := strings.Index(content[i:], blockEnd)
	if j < 0 {
		return content
	}
	after := strings.TrimPrefix(content[i+j+len(blockEnd):], "\n")
	before := content[:i]
	if strings.HasSuffix(before, "\n\n") {
		before = before[:len(before)-1]
	}
	return before + after
}

// write puts the block into path, creating the file when it is not
// there, and reports whether anything changed.
func write(path string, lines []string) (changed bool, err error) {
	return rewrite(path, func(old string) string { return Apply(old, lines) }, true)
}

// erase takes the block out of path, when the file is there, and reports
// whether anything changed.
func erase(path string) (changed bool, err error) {
	return rewrite(path, Remove, false)
}

// rewrite reads path, puts it through edit, and writes it back when that
// changed anything — creating it when create says so, else leaving a
// missing file missing.
func rewrite(path string, edit func(string) string, create bool) (bool, error) {
	old, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err != nil && !create {
		return false, nil
	}
	next := edit(string(old))
	if string(old) == next {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	mode := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	return true, os.WriteFile(path, []byte(next), mode)
}

func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

// confPath is a preference's path as a file to write: expanded, and
// refused when it is not set or not absolute (user, 2026-09-24: setup
// writes where the user said, and says so when they have not said).
func confPath(p, key string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("%s is not set: run locku, preference › %s", key, key)
	}
	abs, ok := config.AbsPath(p)
	if !ok {
		return "", fmt.Errorf("%s is %q, not an absolute path", key, p)
	}
	return abs, nil
}

func report(w io.Writer, path string, changed bool, verb string) {
	if changed {
		fmt.Fprintf(w, "%s %s\n", verb, path)
	} else if verb == "wrote" {
		fmt.Fprintf(w, "%s already up to date\n", path)
	} else {
		fmt.Fprintf(w, "%s has nothing of locku's\n", path)
	}
}

// The tmux side (function.md §6.2). No key is bound: a user with a
// tmux.conf has keys of their own, so the lock is a command — `prefix :`
// then `locku` — through a command alias (user, 2026-09-24). The hooks
// lock a client that attaches to, or switches into, a session locku has
// marked @locked (tmux package): tmux's own lock-session locks only the
// clients attached at that moment. The alias and the hooks sit at a
// high index in their arrays, so the user's own entries — at 0 — are
// untouched, and -d can take exactly these out again.
const (
	tmuxIndex = "90"
	hookCmd   = `if -F "#{@locked}" lock-client`
)

// tmuxLines is the block tmux.conf gets. The lock command is told the
// server's socket — set -F expands #{socket_path} when the file is read
// — since the lock runs in the client's process, where nothing else says
// which server it belongs to.
var tmuxLines = []string{
	`set -gF lock-command "locku lock -S '#{socket_path}'"                       # locku`,
	`set -g lock-after-time 300                                                  # locku: seconds idle before the lock; 0 never`,
	`set -s "command-alias[` + tmuxIndex + `]" "locku=lock-session"                             # locku: prefix : locku`,
	`set-hook -g "client-attached[` + tmuxIndex + `]" "if -F \"#{@locked}\" lock-client"        # locku: attaching to a locked session locks the client`,
	`set-hook -g "client-session-changed[` + tmuxIndex + `]" "if -F \"#{@locked}\" lock-client" # locku: so does switching into one`,
}

// tmuxSet and tmuxUnset are the same on a running server, and its undoing.
var (
	tmuxSet = [][]string{
		{"set", "-gF", "lock-command", "locku lock -S '#{socket_path}'"},
		{"set", "-g", "lock-after-time", "300"},
		{"set", "-s", "command-alias[" + tmuxIndex + "]", "locku=lock-session"},
		{"set-hook", "-g", "client-attached[" + tmuxIndex + "]", hookCmd},
		{"set-hook", "-g", "client-session-changed[" + tmuxIndex + "]", hookCmd},
	}
	tmuxUnset = [][]string{
		{"set", "-gu", "lock-command"},
		{"set", "-gu", "lock-after-time"},
		{"set", "-su", "command-alias[" + tmuxIndex + "]"},
		{"set-hook", "-gu", "client-attached[" + tmuxIndex + "]"},
		{"set-hook", "-gu", "client-session-changed[" + tmuxIndex + "]"},
	}
)

// Tmux writes the block into the file at path — preference's tmux_conf,
// "~/…" allowed — and, when a server is running, sets the same things on
// it now.
func Tmux(w io.Writer, path string) error {
	path, err := confPath(path, "tmux_conf")
	if err != nil {
		return err
	}
	changed, err := write(path, tmuxLines)
	if err != nil {
		return err
	}
	report(w, path, changed, "wrote")
	if !tmuxLive(w, tmuxSet) {
		return nil
	}
	fmt.Fprintln(w, "applied to the running tmux server: prefix : locku locks, 5 idle minutes lock, attaching to a locked session locks")
	return nil
}

// TmuxUndo takes the block out of the file at path and, when a server is
// running, the same things off it.
func TmuxUndo(w io.Writer, path string) error {
	path, err := confPath(path, "tmux_conf")
	if err != nil {
		return err
	}
	changed, err := erase(path)
	if err != nil {
		return err
	}
	report(w, path, changed, "removed locku's block from")
	if !tmuxLive(w, tmuxUnset) {
		return nil
	}
	fmt.Fprintln(w, "taken off the running tmux server too")
	return nil
}

// tmuxLive runs each command on the running server, and reports whether
// there was one to run them on.
func tmuxLive(w io.Writer, cmds [][]string) bool {
	tmux, err := exec.LookPath("tmux")
	if err != nil {
		fmt.Fprintln(w, "tmux is not on PATH: the file is done, nothing applied")
		return false
	}
	if exec.Command(tmux, "has-session").Run() != nil {
		fmt.Fprintln(w, "no tmux server is running: the file takes effect on the next one")
		return false
	}
	for _, args := range cmds {
		if out, err := exec.Command(tmux, args...).CombinedOutput(); err != nil {
			fmt.Fprintf(w, "tmux %s: %s\n", strings.Join(args, " "), strings.TrimSpace(string(out)))
		}
	}
	return true
}

// screenLines is the block .screenrc gets.
var screenLines = []string{"idle 300 lockscreen   # locku: seconds idle before the lock"}

// Screen writes `idle 300 lockscreen` into the file at rc — preference's
// screen_conf — and LOCKPRG into the shell's rc file. LOCKPRG has to be
// in the environment of the shell that runs `screen` — screen's front
// end reads it, and .screenrc's own `setenv` never reaches that process
// (function.md §6.2, measured 2026-09-24) — so the rc file it is.
func Screen(w io.Writer, rc string) error {
	rc, err := confPath(rc, "screen_conf")
	if err != nil {
		return err
	}
	changed, err := write(rc, screenLines)
	if err != nil {
		return err
	}
	report(w, rc, changed, "wrote")

	exe := Binary()
	shellRC, line := shellRCLine(os.Getenv("SHELL"), exe)
	changed, err = write(shellRC, []string{line + "   # locku: screen's LOCKPRG"})
	if err != nil {
		return err
	}
	report(w, shellRC, changed, "wrote")
	fmt.Fprintf(w, "LOCKPRG=%s takes effect in a new shell; a screen session already running: detach, then `screen -r` from that shell\n", exe)
	return nil
}

// ScreenUndo takes the blocks out of the file at rc and the shell's rc.
func ScreenUndo(w io.Writer, rc string) error {
	rc, err := confPath(rc, "screen_conf")
	if err != nil {
		return err
	}
	changed, err := erase(rc)
	if err != nil {
		return err
	}
	report(w, rc, changed, "removed locku's block from")
	shellRC, _ := shellRCLine(os.Getenv("SHELL"), "")
	changed, err = erase(shellRC)
	if err != nil {
		return err
	}
	report(w, shellRC, changed, "removed locku's block from")
	fmt.Fprintln(w, "LOCKPRG is gone from new shells; a shell already open still has it")
	return nil
}

// shellRCLine is the rc file and the line for the shell at shellPath.
// fish sets variables its own way; anything that is not fish or bash is
// treated as zsh's family — zsh is what a Mac has, and bash has its own
// file — and a shell nobody recognised gets ~/.profile, which most of
// them read.
func shellRCLine(shellPath, exe string) (string, string) {
	switch filepath.Base(shellPath) {
	case "fish":
		return filepath.Join(home(), ".config", "fish", "config.fish"), "set -gx LOCKPRG " + shellQuote(exe)
	case "bash":
		return filepath.Join(home(), ".bashrc"), "export LOCKPRG=" + shellQuote(exe)
	case "zsh":
		return filepath.Join(home(), ".zshrc"), "export LOCKPRG=" + shellQuote(exe)
	}
	return filepath.Join(home(), ".profile"), "export LOCKPRG=" + shellQuote(exe)
}

// shellQuote wraps a path for a shell when it needs it.
func shellQuote(s string) string {
	if strings.ContainsAny(s, " \t'\"$`\\") {
		return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
	}
	return s
}

// Binary is the absolute path LOCKPRG points at: locku as PATH finds it —
// a brew symlink survives an upgrade where the Cellar path underneath it
// does not — or, when it is not on PATH, this very executable.
func Binary() string {
	if p, err := exec.LookPath("locku"); err == nil {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
		return p
	}
	if p, err := os.Executable(); err == nil {
		return p
	}
	return "locku"
}
