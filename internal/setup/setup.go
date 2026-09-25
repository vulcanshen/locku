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
	"strconv"
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

// confPath is a tool's conf as a file to write: expanded, and refused
// when it is not set or not absolute (user, 2026-09-24: setup writes
// where the user said, and says so when they have not said).
func confPath(p, tool string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("%s: no file set — Integration › %s › conf", tool, tool)
	}
	abs, ok := config.AbsPath(p)
	if !ok {
		return "", fmt.Errorf("%s: %q is not an absolute path", tool, p)
	}
	return abs, nil
}

// Installed reports whether locku's block is in the file at p — "~/…"
// allowed; a file that is not there, or a path that is no path, has it
// not.
func Installed(p string) bool {
	abs, ok := config.AbsPath(p)
	if !ok {
		return false
	}
	b, err := os.ReadFile(abs)
	return err == nil && strings.Contains(string(b), blockBegin)
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

// The tmux side (function.md §6.2). The lock is the whole server's by
// default, as a screensaver is the whole machine's (user, 2026-09-24):
// `locku` locks every client on the server, and while locku has the
// server marked @locked — a global option every session sees — the
// hooks lock a client that attaches to, or switches into, any session,
// since tmux's own lock-server locks only the clients attached at that
// moment. Or, since 2026-09-25 (user; the study in
// .local/studies/lock.md), one session's: `locku` is lock-session, the
// mark is set on the session, and the same hooks read a client's
// session before the global (measured), so the other sessions are left
// as they are — a screensaver per workspace, not a wall between them.
// The idle lock stays tmux's per-session one either way: the screen
// that idles is the screen that locks. No key is bound unless the user
// names one: a user with a tmux.conf has keys of their own, so the lock
// is a command — `prefix :` then `locku` — through a command alias, and
// a bind-key of their choosing on top (empty binds nothing). The alias
// and the hooks sit at a high index in their arrays, so the user's own
// entries — at 0 — are untouched, and Remove can take exactly these out
// again.
const (
	tmuxIndex = "90"
	hookCmd   = `if -F "#{@locked}" lock-client`
)

// lockCmd is the lock command tmux runs: this binary by its absolute
// path — the client's shell may well not have locku on its PATH, and a
// command not found is a lock that flashes and is gone (user,
// 2026-09-24) — told the server's socket: set -F expands #{socket_path}
// when the line is read, since the lock runs in the client's process,
// where nothing else says which server it belongs to.
func lockCmd() string { return shellQuote(Binary()) + " lock -S '#{socket_path}'" }

// sessionLockCmd is the same told its session too, for lock-session:
// each session's OWN lock-command, set as the session is made
// (sessionHook) with its id expanded in. It has to be this way round —
// measured 2026-09-25: a locked client is no longer in list-clients, so
// the lock cannot look its session up by its tty; display-message -c
// falls back to the latest session, the wrong one. The id is quoted
// because the command runs through the shell, which would read $0 as
// its own name.
func sessionLockCmd() string { return lockCmd() + " -t '#{session_id}'" }

// sessionHook is what the session-created hook runs: the new session's
// lock-command, its id baked in.
func sessionHook() string { return `set -F lock-command "` + sessionLockCmd() + `"` }

// who is whose lock `locku` is, for the comments and the report.
func who(lock string) string {
	if lock == config.LockSession {
		return "this session's clients"
	}
	return "every client"
}

// noted is a line with its `# locku` comment at the column the others
// keep, or after one space when the line is longer than that.
func noted(line, comment string) string { return pad(line, 75) + " " + comment }

// tmuxLines is the block tmux.conf gets, from Integration › tmux: the
// idle time as it is, the lock `locku` runs, for lock-session the hook
// that tells each session's lock its session, and the bind-key when
// there is one.
func tmuxLines(t config.Tmux) []string {
	lines := []string{
		`set -gF lock-command "` + lockCmd() + `"  # locku`,
		noted(`set -g lock-after-time `+itoa(t.LockAfterTime), "# locku: 0 never"),
		noted(`set -s "command-alias[`+tmuxIndex+`]" "locku=`+t.Lock+`"`, "# locku: prefix : locku locks "+who(t.Lock)),
		noted(`set-hook -g "client-attached[`+tmuxIndex+`]" "if -F \"#{@locked}\" lock-client"`, "# locku: attaching while locked locks the client"),
		noted(`set-hook -g "client-session-changed[`+tmuxIndex+`]" "if -F \"#{@locked}\" lock-client"`, "# locku: so does switching sessions"),
	}
	if t.Lock == config.LockSession {
		lines = append(lines, noted(`set-hook -g "session-created[`+tmuxIndex+`]" "`+strings.ReplaceAll(sessionHook(), `"`, `\"`)+`"`, "# locku: a session's lock knows its session"))
	}
	if t.BindKey != "" {
		lines = append(lines, noted(`bind-key `+t.BindKey+` `+t.Lock, "# locku: prefix "+t.BindKey+" locks "+who(t.Lock)))
	}
	return lines
}

// bound is what the block in content binds — the key, and which lock —
// read off the file, so a key or a lock changed since the last write
// can be undone on the running server.
func bound(content string) (key, lock string) {
	i := strings.Index(content, blockBegin)
	if i < 0 {
		return "", ""
	}
	for _, l := range strings.Split(content[i:], "\n") {
		if l == blockEnd {
			break
		}
		f := strings.Fields(l)
		switch {
		case len(f) >= 3 && f[0] == "bind-key":
			key = f[1]
		case strings.Contains(l, `"locku=`+config.LockSession+`"`):
			lock = config.LockSession
		case strings.Contains(l, `"locku=`+config.LockServer+`"`):
			lock = config.LockServer
		}
	}
	return key, lock
}

// boundIn is bound, for the file at path; a file that is not there
// binds nothing.
func boundIn(path string) (key, lock string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	return bound(string(b))
}

// tmuxSet is the same on a running server; tmuxUnset its undoing, one
// for one.
func tmuxSet(t config.Tmux) [][]string {
	cmds := [][]string{
		{"set", "-gF", "lock-command", lockCmd()},
		{"set", "-g", "lock-after-time", itoa(t.LockAfterTime)},
		{"set", "-s", "command-alias[" + tmuxIndex + "]", "locku=" + t.Lock},
		{"set-hook", "-g", "client-attached[" + tmuxIndex + "]", hookCmd},
		{"set-hook", "-g", "client-session-changed[" + tmuxIndex + "]", hookCmd},
	}
	if t.Lock == config.LockSession {
		cmds = append(cmds, []string{"set-hook", "-g", "session-created[" + tmuxIndex + "]", sessionHook()})
	}
	if t.BindKey != "" {
		cmds = append(cmds, []string{"bind-key", t.BindKey, t.Lock})
	}
	return cmds
}

// tmuxUnset undoes tmuxSet one for one — then the session hook, the
// key that was bound, and the mark a lock may have left on the server;
// what each session holds is eachSession's to take off.
func tmuxUnset(key string) [][]string {
	cmds := [][]string{
		{"set", "-gu", "lock-command"},
		{"set", "-gu", "lock-after-time"},
		{"set", "-su", "command-alias[" + tmuxIndex + "]"},
		{"set-hook", "-gu", "client-attached[" + tmuxIndex + "]"},
		{"set-hook", "-gu", "client-session-changed[" + tmuxIndex + "]"},
		{"set-hook", "-gu", "session-created[" + tmuxIndex + "]"},
	}
	if key != "" {
		cmds = append(cmds, []string{"unbind-key", key})
	}
	return append(cmds, []string{"set", "-gu", "@locked"})
}

func itoa(n int) string { return strconv.Itoa(n) }

// pad is s followed by spaces to w, so the comments line up.
func pad(s string, w int) string {
	for len(s) < w {
		s += " "
	}
	return s
}

// Tmux writes the block into the file at t.Conf — "~/…" allowed — and,
// when a server is running, sets the same things on it now. A key the
// file bound before, and this write does not, comes off the server; a
// lock changed takes every mark off, the server's and each session's,
// since a mark left over reads as locked under the other rule (the
// study: a global mark after a switch to lock-session reads as every
// session locked); and each session gets its own lock-command, or
// loses it, as the lock says.
func Tmux(w io.Writer, t config.Tmux) error {
	path, err := confPath(t.Conf, "tmux")
	if err != nil {
		return err
	}
	wasKey, wasLock := boundIn(path)
	changed, err := write(path, tmuxLines(t))
	if err != nil {
		return err
	}
	report(w, path, changed, "wrote")
	cmds := tmuxSet(t)
	if wasKey != "" && wasKey != t.BindKey {
		cmds = append([][]string{{"unbind-key", wasKey}}, cmds...)
	}
	switched := wasLock != "" && wasLock != t.Lock
	if switched {
		cmds = append([][]string{{"set", "-gu", "@locked"}, {"set-hook", "-gu", "session-created[" + tmuxIndex + "]"}}, cmds...)
	}
	tmux, ok := tmuxLive(w, cmds)
	if !ok {
		return nil
	}
	eachSession(tmux, func(id string) {
		if t.Lock == config.LockSession {
			exec.Command(tmux, "set", "-t", id, "-F", "lock-command", sessionLockCmd()).Run()
		} else {
			exec.Command(tmux, "set", "-u", "-t", id, "lock-command").Run()
		}
		if switched {
			exec.Command(tmux, "set", "-u", "-t", id, "@locked").Run()
		}
	})
	fmt.Fprintf(w, "applied to the running tmux server: prefix : locku%s locks %s, %s, attaching while locked locks\n", keySays("prefix", t.BindKey), who(t.Lock), idleSays(t.LockAfterTime))
	return nil
}

// keySays is the bound key in words, after the tool's prefix, for what
// setup reports.
func keySays(prefix, key string) string {
	if key == "" {
		return ""
	}
	return " or " + prefix + " " + key
}

// idleSays is the idle time in words, for what setup reports.
func idleSays(idle int) string {
	if idle <= 0 {
		return "no idle lock"
	}
	return itoa(idle) + " idle seconds lock"
}

// TmuxUndo takes the block out of the file at path and, when a server is
// running, the same things off it — the key it bound, each session's
// own lock-command, and every mark included.
func TmuxUndo(w io.Writer, path string) error {
	path, err := confPath(path, "tmux")
	if err != nil {
		return err
	}
	wasKey, _ := boundIn(path)
	changed, err := erase(path)
	if err != nil {
		return err
	}
	report(w, path, changed, "removed locku's block from")
	tmux, ok := tmuxLive(w, tmuxUnset(wasKey))
	if !ok {
		return nil
	}
	eachSession(tmux, func(id string) {
		exec.Command(tmux, "set", "-u", "-t", id, "lock-command").Run()
		exec.Command(tmux, "set", "-u", "-t", id, "@locked").Run()
	})
	fmt.Fprintln(w, "taken off the running tmux server too")
	return nil
}

// tmuxLive runs each command on the running server, and reports the
// tmux it ran them with — or that there was no server to run them on.
func tmuxLive(w io.Writer, cmds [][]string) (string, bool) {
	tmux, err := exec.LookPath("tmux")
	if err != nil {
		fmt.Fprintln(w, "tmux is not on PATH: the file is done, nothing applied")
		return "", false
	}
	if exec.Command(tmux, "has-session").Run() != nil {
		fmt.Fprintln(w, "no tmux server is running: the file takes effect on the next one")
		return "", false
	}
	for _, args := range cmds {
		if out, err := exec.Command(tmux, args...).CombinedOutput(); err != nil {
			fmt.Fprintf(w, "tmux %s: %s\n", strings.Join(args, " "), strings.TrimSpace(string(out)))
		}
	}
	return tmux, true
}

// eachSession calls f with the id of every session on the running
// server.
func eachSession(tmux string, f func(id string)) {
	out, err := exec.Command(tmux, "list-sessions", "-F", "#{session_id}").Output()
	if err != nil {
		return
	}
	for _, id := range strings.Fields(string(out)) {
		f(id)
	}
}

// The screen side (function.md §6.2), the tmux side's shape under
// screen's own names (user, 2026-09-25). screen has no server: each
// screen is a process of its own, so there is no lock to scope, and no
// hooks, so a lock is not remembered for whoever attaches next. What it
// has is a lock program — LOCKPRG, read by the attacher, the front end
// on the terminal, from the environment of the shell that ran `screen`;
// .screenrc's own `setenv` never reaches that process (measured
// 2026-09-24) — which is why LOCKPRG goes into the shell's rc and not
// the screenrc; an idle timer, `idle N lockscreen`, under screen's own
// name for it; and its own key for the lock, C-a x, built in, which the
// user may double with a `bind <key> lockscreen` of their own (empty
// binds nothing: C-a x locks anyway). Every line is marked `# locku`
// at its end; screen reads a trailing comment fine (measured
// 2026-09-25, screen 4.00.03).
const screenLock = "lockscreen"

// screenLines is the block the screenrc gets, from Integration › screen:
// the idle time as screen's idle (0 turns it off there too), and the
// bind when there is one.
func screenLines(s config.Screen) []string {
	lines := []string{"idle " + itoa(s.Idle) + " " + screenLock + "   # locku: 0 never"}
	if s.Bind != "" {
		lines = append(lines, "bind "+s.Bind+" "+screenLock+"   # locku: C-a "+s.Bind+" locks, as C-a x does")
	}
	return lines
}

// screenBound is the key the block in content binds, read off the
// file, so a key changed since the last write can be unbound on the
// running screens; a key bound outside the block is not locku's.
func screenBound(content string) string {
	i := strings.Index(content, blockBegin)
	if i < 0 {
		return ""
	}
	for _, l := range strings.Split(content[i:], "\n") {
		if l == blockEnd {
			break
		}
		if f := strings.Fields(l); len(f) >= 3 && f[0] == "bind" && f[2] == screenLock {
			return f[1]
		}
	}
	return ""
}

// screenBoundIn is screenBound, for the file at path; a file that is
// not there binds nothing.
func screenBoundIn(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return screenBound(string(b))
}

// screenSet is the same on a running screen, sent with -X; screenUnset
// its undoing, one for one: idle off, and the key that was bound
// unbound — `bind <key>` with no command takes the binding off.
func screenSet(s config.Screen) [][]string {
	cmds := [][]string{{"idle", itoa(s.Idle), screenLock}}
	if s.Bind != "" {
		cmds = append(cmds, []string{"bind", s.Bind, screenLock})
	}
	return cmds
}

func screenUnset(key string) [][]string {
	cmds := [][]string{{"idle", "0"}}
	if key != "" {
		cmds = append(cmds, []string{"bind", key})
	}
	return cmds
}

// Screen writes the block into the file at s.Conf — Integration ›
// screen's conf, "~/…" allowed — LOCKPRG into the shell's rc file, and,
// when screens are running, sets the idle time and the key on each of
// them now (measured 2026-09-25, screen 4.00.03: `screen -S <session>
// -X idle` and `-X bind` reach a running session, attached or not). A
// key the file bound before, and this write does not, comes off them.
// What cannot be set on a running screen is LOCKPRG itself: it is the
// attacher's environment, fixed when `screen` or `screen -r` ran, so a
// screen attached from a shell without it locks with screen's own lock
// until it is detached and attached again from a new shell — the
// report says so.
func Screen(w io.Writer, s config.Screen) error {
	path, err := confPath(s.Conf, "screen")
	if err != nil {
		return err
	}
	wasKey := screenBoundIn(path)
	changed, err := write(path, screenLines(s))
	if err != nil {
		return err
	}
	report(w, path, changed, "wrote")

	exe := Binary()
	shellRC, line := shellRCLine(os.Getenv("SHELL"), exe)
	changed, err = write(shellRC, []string{line + "   # locku: screen's LOCKPRG"})
	if err != nil {
		return err
	}
	report(w, shellRC, changed, "wrote")
	fmt.Fprintf(w, "LOCKPRG=%s takes effect in a new shell; a screen session already running: detach, then `screen -r` from that shell\n", exe)
	cmds := screenSet(s)
	if wasKey != "" && wasKey != s.Bind {
		cmds = append([][]string{{"bind", wasKey}}, cmds...)
	}
	if n, ok := screenLive(w, cmds); ok {
		fmt.Fprintf(w, "applied to %d running screen session(s): C-a x%s locks, %s — with the LOCKPRG each was attached with\n", n, keySays("C-a", s.Bind), idleSays(s.Idle))
	}
	return nil
}

// ScreenUndo takes the blocks out of the file at path and the shell's rc
// and, when screens are running, the same things off them — the idle
// timer, and the key the file bound.
func ScreenUndo(w io.Writer, path string) error {
	path, err := confPath(path, "screen")
	if err != nil {
		return err
	}
	wasKey := screenBoundIn(path)
	changed, err := erase(path)
	if err != nil {
		return err
	}
	report(w, path, changed, "removed locku's block from")
	shellRC, _ := shellRCLine(os.Getenv("SHELL"), "")
	changed, err = erase(shellRC)
	if err != nil {
		return err
	}
	report(w, shellRC, changed, "removed locku's block from")
	fmt.Fprintln(w, "LOCKPRG is gone from new shells; a shell already open still has it")
	if n, ok := screenLive(w, screenUnset(wasKey)); ok {
		fmt.Fprintf(w, "taken off %d running screen session(s) too\n", n)
	}
	return nil
}

// screenLive sends each command to every running screen session, and
// reports how many there were — or that there was none to send them to.
// A session that will not take one is said, and the rest go on: best
// effort, as on tmux.
func screenLive(w io.Writer, cmds [][]string) (int, bool) {
	screen, err := exec.LookPath("screen")
	if err != nil {
		fmt.Fprintln(w, "screen is not on PATH: the file is done, nothing applied")
		return 0, false
	}
	sessions := screenSessions(screen)
	if len(sessions) == 0 {
		fmt.Fprintln(w, "no screen session is running: the file takes effect on the next one")
		return 0, false
	}
	for _, id := range sessions {
		for _, args := range cmds {
			argv := append([]string{"-S", id, "-X"}, args...)
			if out, err := exec.Command(screen, argv...).CombinedOutput(); err != nil {
				fmt.Fprintf(w, "screen %s: %s\n", strings.Join(argv, " "), strings.TrimSpace(string(out)))
			}
		}
	}
	return len(sessions), true
}

// screenSessions is every session `screen -ls` lists, as pid.name — the
// form -S takes without ambiguity. The listing is a line per session
// under a tab, `pid.name (state)`; its exit code says nothing (1 with
// no session, measured 2026-09-25), so only the lines are read.
func screenSessions(screen string) []string {
	out, _ := exec.Command(screen, "-ls").CombinedOutput()
	return screenSessionsOf(string(out))
}

// screenSessionsOf is screenSessions, off the listing.
func screenSessionsOf(ls string) []string {
	var ids []string
	for _, l := range strings.Split(ls, "\n") {
		if !strings.HasPrefix(l, "\t") {
			continue
		}
		if f := strings.Fields(l); len(f) >= 1 && strings.Contains(f[0], ".") {
			ids = append(ids, f[0])
		}
	}
	return ids
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
