package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vulcanshen/locku/internal/config"
)

func TestApplyIsIdempotentAndKeepsTheRest(t *testing.T) {
	lines := []string{"a", "b"}
	once := Apply("set -g mouse on\n", lines)
	if !strings.HasPrefix(once, "set -g mouse on\n\n# >>> locku >>>\na\nb\n# <<< locku <<<\n") {
		t.Fatalf("appended wrong:\n%s", once)
	}
	if again := Apply(once, lines); again != once {
		t.Errorf("not idempotent:\n%s\n---\n%s", once, again)
	}
	// A block in the middle is replaced in place, its neighbours kept.
	mid := "before\n# >>> locku >>>\nold\n# <<< locku <<<\nafter\n"
	got := Apply(mid, []string{"new"})
	if got != "before\n# >>> locku >>>\nnew\n# <<< locku <<<\nafter\n" {
		t.Errorf("replace:\n%s", got)
	}
	// A file without a trailing newline gets one before the block.
	if got := Apply("x", lines); !strings.HasPrefix(got, "x\n\n# >>> locku >>>") {
		t.Errorf("no newline:\n%s", got)
	}
	if got := Apply("", lines); !strings.HasPrefix(got, "# >>> locku >>>") {
		t.Errorf("empty:\n%s", got)
	}
}

// Remove is Apply's undo: the block goes, and the blank line put before
// it, and nothing else; a file with no block is left alone.
func TestRemoveUndoesApply(t *testing.T) {
	for _, before := range []string{"set -g mouse on\n", "", "x", "before\nafter\n"} {
		with := Apply(before, []string{"a", "b"})
		want := before
		if before == "x" {
			want = "x\n" // the newline Apply had to add stays: the file is still whole
		}
		if got := Remove(with); got != want {
			t.Errorf("%q: after apply and remove %q", before, got)
		}
	}
	if got := Remove("before\n# >>> locku >>>\nold\n# <<< locku <<<\nafter\n"); got != "before\nafter\n" {
		t.Errorf("middle: %q", got)
	}
	if got := Remove("nothing of ours\n"); got != "nothing of ours\n" {
		t.Errorf("no block: %q", got)
	}
}

// A tool's idle time — tmux's lock-after-time, screen's idle — is
// handed on as it is, 0 included, which turns the idle lock off.
func TestIdleTimeIsHandedOn(t *testing.T) {
	for _, c := range []struct {
		idle         int
		tmux, screen string
	}{{300, "set -g lock-after-time 300 ", "idle 300 lockscreen"}, {0, "set -g lock-after-time 0 ", "idle 0 lockscreen"}, {45, "set -g lock-after-time 45 ", "idle 45 lockscreen"}} {
		tm := config.Tmux{LockAfterTime: c.idle, Lock: config.LockServer}
		if l := strings.Join(tmuxLines(tm), "\n"); !strings.Contains(l, c.tmux) {
			t.Errorf("idle %d, tmux:\n%s", c.idle, l)
		}
		sc := config.Screen{Idle: c.idle}
		if l := screenLines(sc)[0]; !strings.HasPrefix(l, c.screen) {
			t.Errorf("idle %d, screen: %q", c.idle, l)
		}
		if s := tmuxSet(tm)[1]; s[2] != "lock-after-time" || s[3] != itoa(c.idle) {
			t.Errorf("idle %d, live: %v", c.idle, s)
		}
		if s := screenSet(sc)[0]; strings.Join(s, " ") != c.screen {
			t.Errorf("idle %d, screen live: %v", c.idle, s)
		}
	}
}

// The screen block is the tmux block's shape under screen's own names
// (user, 2026-09-25): the idle time, and a `bind <key> lockscreen` when
// a key is named — none by default, since C-a x is screen's own; what
// is set on a running screen is what is unset, one for one, and the key
// the file binds is read back off it.
func TestScreenBindsAKeyWhenNamed(t *testing.T) {
	plain, keyed := config.Screen{Idle: 300}, config.Screen{Idle: 300, Bind: "^L"}
	if l := screenLines(plain); len(l) != 1 || strings.Contains(l[0], "bind") {
		t.Errorf("no key: %v", l)
	}
	if l := screenLines(keyed); len(l) != 2 || !strings.HasPrefix(l[1], "bind ^L lockscreen ") || !strings.Contains(l[1], "# locku: C-a ^L locks, as C-a x does") {
		t.Errorf("a key: %v", l)
	}
	set, unset := screenSet(keyed), screenUnset("^L")
	if len(set) != 2 || strings.Join(set[1], " ") != "bind ^L lockscreen" || len(unset) != 2 || strings.Join(unset[0], " ") != "idle 0" || strings.Join(unset[1], " ") != "bind ^L" {
		t.Errorf("live: set %v, unset %v", set, unset)
	}
	if u := screenUnset(""); len(u) != 1 {
		t.Errorf("no key to unbind: %v", u)
	}
	if k := screenBound(Apply("bind l redisplay\n", screenLines(keyed))); k != "^L" {
		t.Errorf("read off the file: %q", k)
	}
	if k := screenBound(Apply("bind l lockscreen\n", screenLines(plain))); k != "" {
		t.Errorf("a key bound outside the block is not locku's: %q", k)
	}
	// A screen -ls listing is a line per session under a tab; its
	// other lines are not sessions.
	ls := "There are screens on:\n\t9092.probe\t(Attached)\n\t9100.other\t(Detached)\n2 Sockets in /tmp/screens.\n"
	if got := screenSessionsOf(ls); len(got) != 2 || got[0] != "9092.probe" || got[1] != "9100.other" {
		t.Errorf("sessions: %v", got)
	}
	if got := screenSessionsOf("No Sockets found in /tmp/screens.\n"); len(got) != 0 {
		t.Errorf("no sessions: %v", got)
	}
}

// Every line locku writes says so at its end, so it reads as locku's
// when met on its own (user, 2026-09-24: it has to be easy to take out).
func TestEveryLineIsMarked(t *testing.T) {
	server := config.Tmux{LockAfterTime: 300, Lock: config.LockServer}
	session := config.Tmux{LockAfterTime: 300, Lock: config.LockSession, BindKey: "C-l"}
	for _, l := range append(append(tmuxLines(server), tmuxLines(session)...), screenLines(config.Screen{Idle: 300, Bind: "l"})...) {
		if !strings.Contains(l, " # locku") {
			t.Errorf("unmarked: %q", l)
		}
	}
	// No key is bound unless the user names one: the lock is a command
	// alias, and the hooks and the alias sit at locku's own index.
	// The lock command is this binary by its absolute path — not "locku",
	// which the client's shell may not find — told the socket.
	joined := strings.Join(tmuxLines(server), "\n")
	if !strings.Contains(joined, `set -gF lock-command "/`) || !strings.Contains(joined, ` lock -S '#{socket_path}'"`) ||
		strings.Contains(joined, `"locku lock`) {
		t.Errorf("the lock command must be absolute and told the socket:\n%s", joined)
	}
	if !strings.HasPrefix(tmuxSet(server)[0][3], "/") || !strings.HasSuffix(tmuxSet(server)[0][3], " lock -S '#{socket_path}'") {
		t.Errorf("live lock command: %q", tmuxSet(server)[0][3])
	}
	// The lock is the server's by default: lock-server, and no hook that
	// tells a session its lock.
	if strings.Contains(joined, "bind-key") || !strings.Contains(joined, `command-alias[90]" "locku=lock-server"`) ||
		!strings.Contains(joined, `client-attached[90]`) || !strings.Contains(joined, `client-session-changed[90]`) ||
		!strings.Contains(joined, `#{@locked}`) || strings.Contains(joined, "lock-session") || strings.Contains(joined, "session-created") {
		t.Errorf("tmux block:\n%s", joined)
	}
	// lock-session: the alias and the key run it, and a session-created
	// hook gives each session its own lock-command with its id in it,
	// quoted from the shell (user, 2026-09-25; measured: a locked client
	// cannot find its session by its tty).
	sj := strings.Join(tmuxLines(session), "\n")
	if !strings.Contains(sj, `"locku=lock-session"`) || !strings.Contains(sj, "bind-key C-l lock-session") ||
		!strings.Contains(sj, `set-hook -g "session-created[90]" "set -F lock-command \"/`) || !strings.Contains(sj, `-t '#{session_id}'\""`) ||
		strings.Contains(sj, "lock-server") {
		t.Errorf("tmux block, lock-session:\n%s", sj)
	}
	// What is set on a live server is what is unset, one for one — then
	// the session hook, and the mark a lock may have left.
	set, unset := tmuxSet(server), tmuxUnset("")
	if len(unset) != len(set)+2 || unset[len(unset)-2][2] != "session-created[90]" || unset[len(unset)-1][2] != "@locked" {
		t.Errorf("%d set, %d unset: %v", len(set), len(unset), unset)
	}
	for i := range set {
		if set[i][2] != unset[i][2] {
			t.Errorf("set %v, unset %v", set[i], unset[i])
		}
	}
	set, unset = tmuxSet(session), tmuxUnset("C-l")
	if hook := set[5]; strings.Join(hook[:3], " ") != "set-hook -g session-created[90]" || !strings.HasSuffix(hook[3], `-t '#{session_id}'"`) {
		t.Errorf("live session hook: %v", hook)
	}
	if last := set[len(set)-1]; strings.Join(last, " ") != "bind-key C-l lock-session" {
		t.Errorf("live bind: %v", last)
	}
	if u := unset[len(unset)-2]; strings.Join(u, " ") != "unbind-key C-l" || unset[len(unset)-1][2] != "@locked" {
		t.Errorf("live unbind: %v", unset)
	}
	// What the file binds is read back off it; a key bound outside the
	// block is not locku's.
	if k, l := bound(Apply("set -g mouse on\n", tmuxLines(session))); k != "C-l" || l != config.LockSession {
		t.Errorf("read off the file: %q %q", k, l)
	}
	if k, l := bound(Apply("bind-key l last-window\n", tmuxLines(server))); k != "" || l != config.LockServer {
		t.Errorf("read off the file: %q %q", k, l)
	}
}

func TestTmuxWritesAndUndoesTheFile(t *testing.T) {
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("PATH", t.TempDir()) // no tmux
	path := filepath.Join(h, ".tmux.conf")
	os.WriteFile(path, []byte("set -g mouse on\n"), 0o644)
	at := func(conf, lock, key string) config.Tmux {
		return config.Tmux{Conf: conf, LockAfterTime: 300, Lock: lock, BindKey: key}
	}
	var out bytes.Buffer
	if err := Tmux(&out, at(path, config.LockServer, "")); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range tmuxLines(at(path, config.LockServer, "")) {
		if !strings.Contains(string(body), want) {
			t.Errorf("missing %q in\n%s", want, body)
		}
	}
	if !strings.Contains(out.String(), "wrote") || !strings.Contains(out.String(), "not on PATH") {
		t.Errorf("output:\n%s", out.String())
	}
	out.Reset()
	if err := Tmux(&out, at(path, config.LockServer, "")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "already up to date") {
		t.Errorf("second run:\n%s", out.String())
	}
	// A key: one bind-key line; emptied again, the line goes.
	if err := Tmux(&out, at(path, config.LockServer, "l")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), "bind-key l lock-server") || strings.Count(string(b), "# locku") != 6 {
		t.Errorf("with a key:\n%s", b)
	}
	if err := Tmux(&out, at(path, config.LockServer, "")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); strings.Contains(string(b), "bind-key") || strings.Count(string(b), "# locku") != 5 {
		t.Errorf("key emptied:\n%s", b)
	}
	// lock-session: the alias, and the session hook; lock-server again,
	// and both go.
	if err := Tmux(&out, at(path, config.LockSession, "")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), `"locku=lock-session"`) || !strings.Contains(string(b), "session-created[90]") || strings.Count(string(b), "# locku") != 6 {
		t.Errorf("lock-session:\n%s", b)
	}
	if err := Tmux(&out, at(path, config.LockServer, "")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); strings.Contains(string(b), "lock-session") || strings.Contains(string(b), "session-created") || strings.Count(string(b), "# locku") != 5 {
		t.Errorf("lock-server again:\n%s", b)
	}
	// Undo: the block goes, the user's line stays; again, nothing to do.
	out.Reset()
	if err := TmuxUndo(&out, path); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != "set -g mouse on\n" {
		t.Errorf("after undo:\n%s", b)
	}
	if !strings.Contains(out.String(), "removed") {
		t.Errorf("undo output:\n%s", out.String())
	}
	out.Reset()
	if err := TmuxUndo(&out, path); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "nothing of locku's") {
		t.Errorf("second undo:\n%s", out.String())
	}
	// Undo on a file that is not there makes none.
	if err := TmuxUndo(&out, filepath.Join(h, "none.conf")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(h, "none.conf")); err == nil {
		t.Error("undo created the file")
	}
	// No path is a refusal, not a guess; a relative one too. A path
	// under ~ is expanded, the directory made.
	if err := Tmux(&out, at("", config.LockServer, "")); err == nil || !strings.Contains(err.Error(), "tmux: no file set") {
		t.Errorf("empty path: %v", err)
	}
	if err := Tmux(&out, at("tmux.conf", config.LockServer, "")); err == nil || !strings.Contains(err.Error(), "not an absolute path") {
		t.Errorf("relative path: %v", err)
	}
	h2 := t.TempDir()
	t.Setenv("HOME", h2)
	if err := Tmux(&out, at("~/.config/tmux/tmux.conf", config.LockServer, "")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(h2, ".config", "tmux", "tmux.conf")); !strings.Contains(string(b), blockBegin) {
		t.Errorf("~ was not expanded:\n%s", b)
	}
}

func TestScreenWritesAndUndoesRCAndShellRC(t *testing.T) {
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("PATH", t.TempDir())
	for _, c := range []struct{ shell, rc, prefix string }{
		{"/bin/zsh", ".zshrc", "export LOCKPRG="},
		{"/bin/bash", ".bashrc", "export LOCKPRG="},
		{"/opt/homebrew/bin/fish", ".config/fish/config.fish", "set -gx LOCKPRG "},
		{"/bin/weird", ".profile", "export LOCKPRG="},
	} {
		t.Setenv("SHELL", c.shell)
		var out bytes.Buffer
		if err := Screen(&out, config.Screen{Conf: filepath.Join(h, ".screenrc"), Idle: 300}); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(h, c.rc))
		if err != nil {
			t.Fatalf("%s: %v", c.shell, err)
		}
		if !strings.Contains(string(b), blockBegin+"\n"+c.prefix) || !strings.Contains(string(b), "# locku") {
			t.Errorf("%s:\n%s", c.shell, b)
		}
		if !strings.Contains(out.String(), "new shell") || !strings.Contains(out.String(), "not on PATH") {
			t.Errorf("%s output:\n%s", c.shell, out.String())
		}
		out.Reset()
		if err := ScreenUndo(&out, filepath.Join(h, ".screenrc")); err != nil {
			t.Fatal(err)
		}
		if b, _ := os.ReadFile(filepath.Join(h, c.rc)); strings.Contains(string(b), "LOCKPRG") {
			t.Errorf("%s after undo:\n%s", c.shell, b)
		}
		if err := Screen(&out, config.Screen{Conf: filepath.Join(h, ".screenrc"), Idle: 300}); err != nil {
			t.Fatal(err)
		}
	}
	rc := filepath.Join(h, ".screenrc")
	b, _ := os.ReadFile(rc)
	if !strings.Contains(string(b), "idle 300 lockscreen") || strings.Contains(string(b), "setenv") || strings.Contains(string(b), "bind") || strings.Count(string(b), "# locku") != 1 {
		t.Errorf(".screenrc:\n%s", b)
	}
	// A key: one bind line more; emptied again, the line goes.
	if err := Screen(new(bytes.Buffer), config.Screen{Conf: rc, Idle: 300, Bind: "l"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(rc); !strings.Contains(string(b), "bind l lockscreen") || strings.Count(string(b), "# locku") != 2 {
		t.Errorf("with a key:\n%s", b)
	}
	if err := Screen(new(bytes.Buffer), config.Screen{Conf: rc, Idle: 300}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(rc); strings.Contains(string(b), "bind") || strings.Count(string(b), "# locku") != 1 {
		t.Errorf("key emptied:\n%s", b)
	}
	if err := ScreenUndo(new(bytes.Buffer), rc); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(rc); strings.Contains(string(b), "lockscreen") {
		t.Errorf(".screenrc after undo:\n%s", b)
	}
	// Unset, nothing is written — not even the shell rc.
	h3 := t.TempDir()
	t.Setenv("HOME", h3)
	if err := Screen(new(bytes.Buffer), config.Screen{Idle: 300}); err == nil || !strings.Contains(err.Error(), "screen: no file set") {
		t.Errorf("empty path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(h3, ".profile")); err == nil {
		t.Error("the shell rc was written although screen_conf is not set")
	}
}

func TestShellQuote(t *testing.T) {
	if shellQuote("/usr/local/bin/locku") != "/usr/local/bin/locku" {
		t.Error("plain path quoted")
	}
	if shellQuote("/Users/a b/locku") != "'/Users/a b/locku'" {
		t.Error(shellQuote("/Users/a b/locku"))
	}
}
