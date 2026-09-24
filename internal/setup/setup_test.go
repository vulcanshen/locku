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

func TestTmuxWritesTheFile(t *testing.T) {
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("PATH", t.TempDir()) // no tmux
	var out bytes.Buffer
	if err := Tmux(&out, filepath.Join(h, ".tmux.conf")); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(h, ".tmux.conf"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range tmuxLines {
		if !strings.Contains(string(body), want) {
			t.Errorf("missing %q in\n%s", want, body)
		}
	}
	if !strings.Contains(out.String(), "wrote") || !strings.Contains(out.String(), "not on PATH") {
		t.Errorf("output:\n%s", out.String())
	}
	out.Reset()
	if err := Tmux(&out, filepath.Join(h, ".tmux.conf")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "already up to date") {
		t.Errorf("second run:\n%s", out.String())
	}
	// No path is a refusal, not a guess; a relative one too. A path
	// under ~ is expanded, the directory made.
	if err := Tmux(&out, ""); err == nil || !strings.Contains(err.Error(), "tmux_conf is not set") {
		t.Errorf("empty path: %v", err)
	}
	if err := Tmux(&out, "tmux.conf"); err == nil || !strings.Contains(err.Error(), "not an absolute path") {
		t.Errorf("relative path: %v", err)
	}
	h2 := t.TempDir()
	t.Setenv("HOME", h2)
	if err := Tmux(&out, "~/.config/tmux/tmux.conf"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(h2, ".config", "tmux", "tmux.conf")); !strings.Contains(string(b), blockBegin) {
		t.Errorf("~ was not expanded:\n%s", b)
	}
}

func TestScreenWritesRCAndShellRC(t *testing.T) {
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
		if err := Screen(&out, filepath.Join(h, ".screenrc")); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(h, c.rc))
		if err != nil {
			t.Fatalf("%s: %v", c.shell, err)
		}
		if !strings.Contains(string(b), blockBegin+"\n"+c.prefix) {
			t.Errorf("%s:\n%s", c.shell, b)
		}
		if !strings.Contains(out.String(), "new shell") {
			t.Errorf("%s output:\n%s", c.shell, out.String())
		}
	}
	b, _ := os.ReadFile(filepath.Join(h, ".screenrc"))
	if !strings.Contains(string(b), "idle 300 lockscreen") || strings.Contains(string(b), "setenv") {
		t.Errorf(".screenrc:\n%s", b)
	}
	// Unset, nothing is written — not even the shell rc.
	h3 := t.TempDir()
	t.Setenv("HOME", h3)
	if err := Screen(new(bytes.Buffer), ""); err == nil || !strings.Contains(err.Error(), "screen_conf is not set") {
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
