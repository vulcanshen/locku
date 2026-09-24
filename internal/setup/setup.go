// Package setup is `locku setup`: it writes the lock into tmux's and
// screen's configuration (function.md §6.2). Only a managed block is ever
// touched — between two marker lines — so running it again replaces the
// block and nothing else, and a hand-written file keeps every other line.
package setup

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// write puts the block into path, creating the file when it is not
// there, and reports whether anything changed.
func write(path string, lines []string) (changed bool, err error) {
	old, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	next := Apply(string(old), lines)
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

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func report(w io.Writer, path string, changed bool) {
	if changed {
		fmt.Fprintf(w, "wrote %s\n", path)
	} else {
		fmt.Fprintf(w, "%s already up to date\n", path)
	}
}

// tmuxLines is the block tmux gets (function.md §8).
var tmuxLines = []string{
	`set -g lock-command "locku lock"`,
	`set -g lock-after-time 300`,
	`bind L lock-session`,
}

// Tmux writes the block into ~/.tmux.conf — or ~/.config/tmux/tmux.conf
// when only that exists — and, when a server is running, sets the same
// three things on it now.
func Tmux(w io.Writer) error {
	path := filepath.Join(home(), ".tmux.conf")
	if alt := filepath.Join(home(), ".config", "tmux", "tmux.conf"); !exists(path) && exists(alt) {
		path = alt
	}
	changed, err := write(path, tmuxLines)
	if err != nil {
		return err
	}
	report(w, path, changed)

	tmux, err := exec.LookPath("tmux")
	if err != nil {
		fmt.Fprintln(w, "tmux is not on PATH: the file is written, nothing applied")
		return nil
	}
	if exec.Command(tmux, "has-session").Run() != nil {
		fmt.Fprintln(w, "no tmux server is running: the file takes effect on the next one")
		return nil
	}
	for _, args := range [][]string{
		{"set", "-g", "lock-command", "locku lock"},
		{"set", "-g", "lock-after-time", "300"},
		{"bind", "L", "lock-session"},
	} {
		if out, err := exec.Command(tmux, args...).CombinedOutput(); err != nil {
			fmt.Fprintf(w, "tmux %s: %s\n", strings.Join(args, " "), strings.TrimSpace(string(out)))
			continue
		}
	}
	fmt.Fprintln(w, "applied to the running tmux server: prefix L locks, 5 idle minutes lock")
	return nil
}

// Screen writes `idle 300 lockscreen` into ~/.screenrc and LOCKPRG into
// the shell's rc file. LOCKPRG has to be in the environment of the shell
// that runs `screen` — screen's front end reads it, and .screenrc's own
// `setenv` never reaches that process (function.md §6.2, measured
// 2026-09-24) — so the rc file it is.
func Screen(w io.Writer) error {
	rc := filepath.Join(home(), ".screenrc")
	changed, err := write(rc, []string{"idle 300 lockscreen"})
	if err != nil {
		return err
	}
	report(w, rc, changed)

	exe := Binary()
	shellRC, line := shellRCLine(os.Getenv("SHELL"), exe)
	changed, err = write(shellRC, []string{line})
	if err != nil {
		return err
	}
	report(w, shellRC, changed)
	fmt.Fprintf(w, "LOCKPRG=%s takes effect in a new shell; a screen session already running: detach, then `screen -r` from that shell\n", exe)
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
