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
	if k := screenBound(blockOf(Apply("bind l redisplay\n", screenLines(keyed)))); k != "^L" {
		t.Errorf("read off the file: %q", k)
	}
	if k := screenBound(blockOf(Apply("bind l lockscreen\n", screenLines(plain)))); k != "" {
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
	lines := append(append(tmuxLines(server), tmuxLines(session)...), screenLines(config.Screen{Idle: 300, Bind: "l"})...)
	for _, l := range append(lines, tmuxInclude("/x/locku.tmux.conf"), screenInclude("/x/locku.screenrc")) {
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
	// The lock is the server's by default: lock-server, and no hook that
	// tells a session its lock.
	if strings.Contains(joined, "bind-key") || !strings.Contains(joined, `command-alias[90]" "locku=lock-server"`) ||
		!strings.Contains(joined, `client-attached[90]`) || !strings.Contains(joined, `client-session-changed[90]`) ||
		!strings.Contains(joined, `#{@locked}`) || strings.Contains(joined, "lock-session") || strings.Contains(joined, "session-created") {
		t.Errorf("tmux block:\n%s", joined)
	}
	// Both hooks lock the client that set them off, by name: a bare
	// lock-client took the other client when two attached at once
	// (measured 2026-09-28).
	want := `"run -C \"#{?#{@locked},lock-client -t #{hook_client},}\""`
	for _, h := range []string{"client-attached[90]", "client-session-changed[90]"} {
		if !strings.Contains(joined, `set-hook -g "`+h+`" `+want) {
			t.Errorf("the %s hook must lock #{hook_client}:\n%s", h, joined)
		}
	}
	if strings.Contains(joined, `" lock-client`) || strings.Contains(joined, "lock-client\"") {
		t.Errorf("a bare lock-client is left:\n%s", joined)
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
	// The server takes the block itself, sourced, so there is no second
	// list to keep in step; what is unset is everything the block sets
	// — each option once — then the session hook, the key that was
	// bound, and the mark a lock may have left.
	unset := tmuxUnset("")
	names := []string{}
	for _, u := range unset {
		names = append(names, u[len(u)-1])
	}
	if strings.Join(names, " ") != "lock-command lock-after-time command-alias[90] client-attached[90] client-session-changed[90] session-created[90] @locked" {
		t.Errorf("unset: %v", names)
	}
	for _, l := range tmuxLines(session) {
		if strings.HasPrefix(l, "bind-key") {
			continue
		}
		f := strings.Fields(l)
		opt := strings.Trim(f[2], `"`)
		if f[0] == "set-hook" {
			opt = strings.Trim(f[2], `"`)
		}
		found := false
		for _, n := range names {
			if n == opt {
				found = true
			}
		}
		if !found {
			t.Errorf("the block sets %q, which nothing unsets", opt)
		}
	}
	unset = tmuxUnset("C-l")
	if u := unset[len(unset)-2]; strings.Join(u, " ") != "unbind-key C-l" || unset[len(unset)-1][2] != "@locked" {
		t.Errorf("live unbind: %v", unset)
	}
	// What the file binds is read back off it; a key bound outside the
	// block is not locku's.
	if k, l := bound(blockOf(Apply("set -g mouse on\n", tmuxLines(session)))); k != "C-l" || l != config.LockSession {
		t.Errorf("read off the file: %q %q", k, l)
	}
	if k, l := bound(blockOf(Apply("bind-key l last-window\n", tmuxLines(server)))); k != "" || l != config.LockServer {
		t.Errorf("read off the file: %q %q", k, l)
	}
}

func TestTmuxWritesAndUndoesTheFile(t *testing.T) {
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("PATH", t.TempDir()) // no tmux
	t.Setenv("LOCKU__CONFIG", filepath.Join(h, ".config", "locku"))
	path := filepath.Join(h, ".tmux.conf")
	own := filepath.Join(h, ".config", "locku", "locku.tmux.conf")
	os.WriteFile(path, []byte("set -g mouse on\n"), 0o644)
	at := func(conf, lock, key string) config.Tmux {
		return config.Tmux{Conf: conf, LockAfterTime: 300, Lock: lock, BindKey: key}
	}
	var out bytes.Buffer
	if err := Tmux(&out, at(path, config.LockServer, "")); err != nil {
		t.Fatal(err)
	}
	// The user's file gets the block with one line in it, which reads
	// locku's own file by ~, the same on every machine (user, 2026-10-08);
	// the lines are in locku's own file.
	user, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(user) != "set -g mouse on\n\n"+blockBegin+"\nsource-file -q ~/.config/locku/locku.tmux.conf  # locku: the lock, as locku's settings screen sets it\n"+blockEnd+"\n" {
		t.Errorf("the user's file:\n%s", user)
	}
	body, err := os.ReadFile(own)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range tmuxLines(at(path, config.LockServer, "")) {
		if !strings.Contains(string(body), want) {
			t.Errorf("missing %q in\n%s", want, body)
		}
	}
	if !strings.Contains(string(body), "~/.tmux.conf reads this file") {
		t.Errorf("locku's own file says nothing of who reads it:\n%s", body)
	}
	if !strings.Contains(out.String(), "wrote "+own) || !strings.Contains(out.String(), "wrote "+path) || !strings.Contains(out.String(), "not on PATH") {
		t.Errorf("output:\n%s", out.String())
	}
	if !Installed(path) {
		t.Error("not installed after a write")
	}
	out.Reset()
	if err := Tmux(&out, at(path, config.LockServer, "")); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "already up to date") != 2 {
		t.Errorf("second run:\n%s", out.String())
	}
	// A row changed is locku's own file's alone: the user's is not
	// touched again (user, 2026-10-08).
	mine := func(label string, lines int) string {
		t.Helper()
		if b, _ := os.ReadFile(path); string(b) != string(user) {
			t.Errorf("%s: the user's file changed:\n%s", label, b)
		}
		b, _ := os.ReadFile(own)
		if n := strings.Count(string(b), " # locku"); n != lines {
			t.Errorf("%s: %d lines of locku's, want %d:\n%s", label, n, lines, b)
		}
		return string(b)
	}
	// A key: one bind-key line; emptied again, the line goes.
	if err := Tmux(&out, at(path, config.LockServer, "l")); err != nil {
		t.Fatal(err)
	}
	if b := mine("with a key", 6); !strings.Contains(b, "bind-key l lock-server") {
		t.Errorf("with a key:\n%s", b)
	}
	if err := Tmux(&out, at(path, config.LockServer, "")); err != nil {
		t.Fatal(err)
	}
	if b := mine("key emptied", 5); strings.Contains(b, "bind-key") {
		t.Errorf("key emptied:\n%s", b)
	}
	// lock-session: the alias, and the session hook; lock-server again,
	// and both go.
	if err := Tmux(&out, at(path, config.LockSession, "")); err != nil {
		t.Fatal(err)
	}
	if b := mine("lock-session", 6); !strings.Contains(b, `"locku=lock-session"`) || !strings.Contains(b, "session-created[90]") {
		t.Errorf("lock-session:\n%s", b)
	}
	if err := Tmux(&out, at(path, config.LockServer, "")); err != nil {
		t.Fatal(err)
	}
	if b := mine("lock-server again", 5); strings.Contains(b, "lock-session") || strings.Contains(b, "session-created") {
		t.Errorf("lock-server again:\n%s", b)
	}
	// locku's own file gone, the line reads nothing: not installed.
	os.Rename(own, own+".away")
	if Installed(path) {
		t.Error("installed with locku's own file gone")
	}
	os.Rename(own+".away", own)
	// Undo: the block goes, the user's line stays, locku's own file goes;
	// again, nothing to do.
	out.Reset()
	if err := TmuxUndo(&out, path); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != "set -g mouse on\n" {
		t.Errorf("after undo:\n%s", b)
	}
	if _, err := os.Stat(own); err == nil {
		t.Error("locku's own file is left after undo")
	}
	if !strings.Contains(out.String(), "removed locku's block from") || !strings.Contains(out.String(), "removed "+own) {
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

// A block an earlier locku wrote whole into the user's file — the lines
// themselves, before 2026-10-08 — is installed as it is, the key and the
// lock it set are read off it to be undone, and the next write puts the
// one line in its place.
func TestAnEarlierBlockGivesWayToTheLine(t *testing.T) {
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("LOCKU__CONFIG", filepath.Join(h, ".config", "locku"))
	tc := filepath.Join(h, ".tmux.conf")
	os.WriteFile(tc, []byte(Apply("set -g mouse on\n", tmuxLines(config.Tmux{LockAfterTime: 300, Lock: config.LockSession, BindKey: "l"}))), 0o644)
	if !Installed(tc) {
		t.Error("an earlier tmux block is not installed")
	}
	if k, l := bound(ours(tc, ownPath(tmuxFile))); k != "l" || l != config.LockSession {
		t.Errorf("read off an earlier block: %q %q", k, l)
	}
	if err := Tmux(new(bytes.Buffer), config.Tmux{Conf: tc, LockAfterTime: 300, Lock: config.LockServer}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(tc); !strings.HasPrefix(string(b), "set -g mouse on\n\n"+blockBegin+"\nsource-file -q ~/") || strings.Contains(string(b), "lock-command") {
		t.Errorf("tmux, after a write:\n%s", b)
	}
	rc := filepath.Join(h, ".screenrc")
	os.WriteFile(rc, []byte(Apply("", screenLines(config.Screen{Idle: 300, Bind: "^L"}))), 0o644)
	if !Installed(rc) || screenBound(ours(rc, ownPath(screenFile))) != "^L" {
		t.Errorf("an earlier screen block: installed %v, key %q", Installed(rc), screenBound(ours(rc, ownPath(screenFile))))
	}
	if err := Screen(new(bytes.Buffer), config.Screen{Conf: rc, Idle: 300}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(rc); !strings.HasPrefix(string(b), blockBegin+"\nsource $HOME/.config/locku/locku.screenrc ") || strings.Contains(string(b), "idle") {
		t.Errorf("screen, after a write:\n%s", b)
	}
}

// The line names locku's own file from home as each tool writes it —
// tmux ~, screen $HOME (measured 2026-10-08) — and, outside home or with
// a character a config line reads otherwise, by the whole path, quoted.
func TestTheLineNamesLockusOwnFile(t *testing.T) {
	h := t.TempDir()
	t.Setenv("HOME", h)
	for _, c := range []struct{ own, tmux, screen string }{
		{filepath.Join(h, ".config", "locku", "locku.tmux.conf"), "~/.config/locku/locku.tmux.conf", "$HOME/.config/locku/locku.tmux.conf"},
		{filepath.Join(h, "My Config", "locku.tmux.conf"), "'" + h + "/My Config/locku.tmux.conf'", "'" + h + "/My Config/locku.tmux.conf'"},
		{"/etc/locku/locku.tmux.conf", "'/etc/locku/locku.tmux.conf'", "'/etc/locku/locku.tmux.conf'"},
	} {
		if got := includeArg(c.own, "~/"); got != c.tmux {
			t.Errorf("tmux, %s: %s", c.own, got)
		}
		if got := includeArg(c.own, "$HOME/"); got != c.screen {
			t.Errorf("screen, %s: %s", c.own, got)
		}
	}
	if tilde(filepath.Join(h, "a", "b")) != "~/a/b" || tilde("/etc/a") != "/etc/a" || tilde(h) != h {
		t.Errorf("tilde: %s %s %s", tilde(filepath.Join(h, "a", "b")), tilde("/etc/a"), tilde(h))
	}
}

func TestScreenWritesAndUndoesRCAndShellRC(t *testing.T) {
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LOCKU__CONFIG", filepath.Join(h, ".config", "locku"))
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
	// The screenrc gets one line, which reads locku's own file by $HOME;
	// the lines are in that file, and a row changed is that file's alone.
	rc := filepath.Join(h, ".screenrc")
	own := filepath.Join(h, ".config", "locku", "locku.screenrc")
	user, _ := os.ReadFile(rc)
	if string(user) != blockBegin+"\nsource $HOME/.config/locku/locku.screenrc   # locku: the lock, as locku's settings screen sets it\n"+blockEnd+"\n" {
		t.Errorf(".screenrc:\n%s", user)
	}
	mine := func(label string, lines int) string {
		t.Helper()
		if b, _ := os.ReadFile(rc); string(b) != string(user) {
			t.Errorf("%s: the user's file changed:\n%s", label, b)
		}
		b, _ := os.ReadFile(own)
		if n := strings.Count(string(b), " # locku"); n != lines {
			t.Errorf("%s: %d lines of locku's, want %d:\n%s", label, n, lines, b)
		}
		return string(b)
	}
	if b := mine("idle", 1); !strings.Contains(b, "idle 300 lockscreen") || strings.Contains(b, "setenv") || strings.Contains(b, "bind") {
		t.Errorf("locku.screenrc:\n%s", b)
	}
	// A key: one bind line more; emptied again, the line goes.
	if err := Screen(new(bytes.Buffer), config.Screen{Conf: rc, Idle: 300, Bind: "l"}); err != nil {
		t.Fatal(err)
	}
	if b := mine("with a key", 2); !strings.Contains(b, "bind l lockscreen") {
		t.Errorf("with a key:\n%s", b)
	}
	if err := Screen(new(bytes.Buffer), config.Screen{Conf: rc, Idle: 300}); err != nil {
		t.Fatal(err)
	}
	if b := mine("key emptied", 1); strings.Contains(b, "bind") {
		t.Errorf("key emptied:\n%s", b)
	}
	if err := ScreenUndo(new(bytes.Buffer), rc); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(rc); strings.Contains(string(b), "locku") {
		t.Errorf(".screenrc after undo:\n%s", b)
	}
	if _, err := os.Stat(own); err == nil {
		t.Error("locku's own screenrc is left after undo")
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
