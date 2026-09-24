package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// idle_lock is handed to both tmux and screen as it is, 0 included —
// which turns the idle lock off in both (user, 2026-09-24: one value
// for every tool that runs locku as its screensaver).
func TestIdleLockIsHandedOn(t *testing.T) {
	for _, c := range []struct {
		idle         int
		tmux, screen string
	}{{300, "set -g lock-after-time 300 ", "idle 300 lockscreen"}, {0, "set -g lock-after-time 0 ", "idle 0 lockscreen"}, {45, "set -g lock-after-time 45 ", "idle 45 lockscreen"}} {
		if l := strings.Join(tmuxLines(c.idle), "\n"); !strings.Contains(l, c.tmux) {
			t.Errorf("idle %d, tmux:\n%s", c.idle, l)
		}
		if l := screenLines(c.idle)[0]; !strings.HasPrefix(l, c.screen) {
			t.Errorf("idle %d, screen: %q", c.idle, l)
		}
		if s := tmuxSet(c.idle)[1]; s[2] != "lock-after-time" || s[3] != itoa(c.idle) {
			t.Errorf("idle %d, live: %v", c.idle, s)
		}
	}
}

// Every line locku writes says so at its end, so it reads as locku's
// when met on its own (user, 2026-09-24: it has to be easy to take out).
func TestEveryLineIsMarked(t *testing.T) {
	for _, l := range append(append([]string{}, tmuxLines(300)...), screenLines(300)...) {
		if !strings.Contains(l, "# locku") {
			t.Errorf("unmarked: %q", l)
		}
	}
	// No key is bound: the lock is a command alias, and the hooks and
	// the alias sit at locku's own index.
	// The lock command is this binary by its absolute path — not "locku",
	// which the client's shell may not find — told the socket.
	joined := strings.Join(tmuxLines(300), "\n")
	if !strings.Contains(joined, `set -gF lock-command "/`) || !strings.Contains(joined, ` lock -S '#{socket_path}'"`) ||
		strings.Contains(joined, `"locku lock`) {
		t.Errorf("the lock command must be absolute and told the socket:\n%s", joined)
	}
	if !strings.HasPrefix(tmuxSet(300)[0][3], "/") || !strings.HasSuffix(tmuxSet(300)[0][3], " lock -S '#{socket_path}'") {
		t.Errorf("live lock command: %q", tmuxSet(300)[0][3])
	}
	// The lock is the server's: lock-server, not lock-session.
	if strings.Contains(joined, "bind ") || !strings.Contains(joined, `command-alias[90]" "locku=lock-server"`) ||
		!strings.Contains(joined, `client-attached[90]`) || !strings.Contains(joined, `client-session-changed[90]`) ||
		!strings.Contains(joined, `#{@locked}`) || strings.Contains(joined, "lock-session") {
		t.Errorf("tmux block:\n%s", joined)
	}
	// What is set on a live server is what is unset, one for one — and
	// then the mark a lock may have left.
	if len(tmuxUnset) != len(tmuxSet(300))+1 || tmuxUnset[len(tmuxUnset)-1][2] != "@locked" {
		t.Errorf("%d set, %d unset: %v", len(tmuxSet(300)), len(tmuxUnset), tmuxUnset)
	}
	for i := range tmuxSet(300) {
		if tmuxSet(300)[i][2] != tmuxUnset[i][2] {
			t.Errorf("set %v, unset %v", tmuxSet(300)[i], tmuxUnset[i])
		}
	}
}

func TestTmuxWritesAndUndoesTheFile(t *testing.T) {
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("PATH", t.TempDir()) // no tmux
	path := filepath.Join(h, ".tmux.conf")
	os.WriteFile(path, []byte("set -g mouse on\n"), 0o644)
	var out bytes.Buffer
	if err := Tmux(&out, path, 300); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range tmuxLines(300) {
		if !strings.Contains(string(body), want) {
			t.Errorf("missing %q in\n%s", want, body)
		}
	}
	if !strings.Contains(out.String(), "wrote") || !strings.Contains(out.String(), "not on PATH") {
		t.Errorf("output:\n%s", out.String())
	}
	out.Reset()
	if err := Tmux(&out, path, 300); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "already up to date") {
		t.Errorf("second run:\n%s", out.String())
	}
	// -d: the block goes, the user's line stays; again, nothing to do.
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
	if err := Tmux(&out, "", 300); err == nil || !strings.Contains(err.Error(), "tmux: no file set") {
		t.Errorf("empty path: %v", err)
	}
	if err := Tmux(&out, "tmux.conf", 300); err == nil || !strings.Contains(err.Error(), "not an absolute path") {
		t.Errorf("relative path: %v", err)
	}
	h2 := t.TempDir()
	t.Setenv("HOME", h2)
	if err := Tmux(&out, "~/.config/tmux/tmux.conf", 300); err != nil {
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
		if err := Screen(&out, filepath.Join(h, ".screenrc"), 300); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(h, c.rc))
		if err != nil {
			t.Fatalf("%s: %v", c.shell, err)
		}
		if !strings.Contains(string(b), blockBegin+"\n"+c.prefix) || !strings.Contains(string(b), "# locku") {
			t.Errorf("%s:\n%s", c.shell, b)
		}
		if !strings.Contains(out.String(), "new shell") {
			t.Errorf("%s output:\n%s", c.shell, out.String())
		}
		out.Reset()
		if err := ScreenUndo(&out, filepath.Join(h, ".screenrc")); err != nil {
			t.Fatal(err)
		}
		if b, _ := os.ReadFile(filepath.Join(h, c.rc)); strings.Contains(string(b), "LOCKPRG") {
			t.Errorf("%s after undo:\n%s", c.shell, b)
		}
		if err := Screen(&out, filepath.Join(h, ".screenrc"), 300); err != nil {
			t.Fatal(err)
		}
	}
	b, _ := os.ReadFile(filepath.Join(h, ".screenrc"))
	if !strings.Contains(string(b), "idle 300 lockscreen") || strings.Contains(string(b), "setenv") {
		t.Errorf(".screenrc:\n%s", b)
	}
	if err := ScreenUndo(new(bytes.Buffer), filepath.Join(h, ".screenrc")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(h, ".screenrc")); strings.Contains(string(b), "lockscreen") {
		t.Errorf(".screenrc after undo:\n%s", b)
	}
	// Unset, nothing is written — not even the shell rc.
	h3 := t.TempDir()
	t.Setenv("HOME", h3)
	if err := Screen(new(bytes.Buffer), "", 300); err == nil || !strings.Contains(err.Error(), "screen: no file set") {
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
