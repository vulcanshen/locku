package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"

	"github.com/vulcanshen/locku/internal/config"
)

// A terminal for the command to ask on, and what is typed at it.
func typing(t *testing.T, lines ...string) *os.File {
	t.Helper()
	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ptmx.Close(); tty.Close() })
	go func() {
		for _, l := range lines {
			time.Sleep(250 * time.Millisecond)
			ptmx.Write([]byte(l + "\n"))
		}
		// Whatever the command echoes is read and dropped.
		buf := make([]byte, 1024)
		for {
			if _, err := ptmx.Read(buf); err != nil {
				return
			}
		}
	}()
	return tty
}

// `locku pin reset` asks twice — y/N, then the login password — makes a
// new PIN, writes its hash over the old one, shows it once and logs the
// reset without it; a wrong password, or a no, changes nothing (user,
// 2026-09-25: as elasticsearch resets a password).
func TestPINResetAsksTwiceAndRotates(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOCKU_CONFIG", dir)
	t.Setenv("LOCKU_DATA", filepath.Join(dir, "data"))
	cfg := config.Default()
	if err := cfg.SetPIN("1234"); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	verify := func(_, pw string) bool { return pw == "s3cret" }

	var out bytes.Buffer
	if code := pinReset(typing(t, "y", "s3cret"), &out, verify); code != 0 {
		t.Fatalf("code %d:\n%s", code, out.String())
	}
	m := regexp.MustCompile(`New PIN: (\d{8})`).FindStringSubmatch(out.String())
	if m == nil || !strings.Contains(out.String(), "Password for ") || !strings.Contains(out.String(), "[y/N]") {
		t.Fatalf("the new PIN must be shown once, after both questions:\n%s", out.String())
	}
	got, _ := config.Load()
	if !got.CheckPIN(m[1]) || got.CheckPIN("1234") {
		t.Errorf("the file must hold the new PIN and not the old: %+v", got.PINHash)
	}
	log, err := os.ReadFile(filepath.Join(dir, "data", "pin-resets.log"))
	if err != nil || !strings.Contains(string(log), "pin reset by ") || !strings.Contains(string(log), "a new PIN written") || strings.Contains(string(log), m[1]) {
		t.Errorf("the log must say what happened, and never the PIN: %v\n%s", err, log)
	}

	// The wrong password: refused, logged, nothing changed.
	out.Reset()
	if code := pinReset(typing(t, "y", "nope"), &out, verify); code == 0 || !strings.Contains(out.String(), "not your login password") {
		t.Errorf("code %d:\n%s", code, out.String())
	}
	if again, _ := config.Load(); !again.CheckPIN(m[1]) {
		t.Error("a refused reset must change nothing")
	}
	if log, _ := os.ReadFile(filepath.Join(dir, "data", "pin-resets.log")); !strings.Contains(string(log), "refused") {
		t.Errorf("the refusal must be logged:\n%s", log)
	}

	// No: nothing asked further, nothing changed.
	out.Reset()
	if code := pinReset(typing(t, "n"), &out, verify); code == 0 || strings.Contains(out.String(), "Password for") {
		t.Errorf("code %d:\n%s", code, out.String())
	}
}

// Without a terminal there is nothing to ask on.
func TestPINResetNeedsATerminal(t *testing.T) {
	t.Setenv("LOCKU_CONFIG", t.TempDir())
	f, err := os.CreateTemp(t.TempDir(), "in")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out bytes.Buffer
	if code := pinReset(f, &out, func(string, string) bool { return true }); code != 2 || !strings.Contains(out.String(), "needs a terminal") {
		t.Errorf("code %d:\n%s", code, out.String())
	}
}
