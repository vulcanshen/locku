package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
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
	t.Setenv("LOCKU__CONFIG", dir)
	t.Setenv("LOCKU__DATA", filepath.Join(dir, "data"))
	cfg := config.Default()
	if err := cfg.SetPIN("1234"); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	verify := func(_, pw string) bool { return pw == "s3cret" }

	var out bytes.Buffer
	if code := pinReset(typing(t, "y", "s3cret"), &out, io.Discard, verify); code != 0 {
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
	if code := pinReset(typing(t, "y", "nope"), &out, io.Discard, verify); code == 0 || !strings.Contains(out.String(), "not your login password") {
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
	if code := pinReset(typing(t, "n"), &out, io.Discard, verify); code == 0 || strings.Contains(out.String(), "Password for") {
		t.Errorf("code %d:\n%s", code, out.String())
	}
}

// A file of locku's that cannot be read stops the reset before it asks:
// nothing written, nothing printed, the path and the reason on stderr and a
// non-zero exit (tdp E3, with no screen). A config.yaml whose pin_hash is
// fine and whose rest is not was once read as the defaults, and the reset
// wrote them over it, and the default profile over profiles.yaml.
func TestPINResetTouchesNothingItCannotRead(t *testing.T) {
	for _, tc := range []struct {
		name  string
		spoil func(cfg config.Config) (path, data string)
	}{
		{"config.yaml", func(cfg config.Config) (string, string) {
			return config.Path(), "version: 1\npin_hash: " + cfg.PINHash + "\nsavers: 3\n"
		}},
		{"profiles.yaml", func(config.Config) (string, string) {
			return config.ProfilesPath(), "profiles: 3\n"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("LOCKU__CONFIG", dir)
			t.Setenv("LOCKU__DATA", filepath.Join(dir, "data"))
			cfg := config.Default()
			if err := cfg.SetPIN("1234"); err != nil {
				t.Fatal(err)
			}
			cfg.Profiles = append(cfg.Profiles, cfg.NewProfile("mine", saver.KindSnake))
			if err := config.Save(cfg); err != nil {
				t.Fatal(err)
			}
			path, data := tc.spoil(cfg)
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			before := map[string][]byte{}
			for _, p := range []string{config.Path(), config.ProfilesPath()} {
				before[p], _ = os.ReadFile(p)
			}

			var out, errs bytes.Buffer
			code := pinReset(typing(t, "y", "s3cret"), &out, &errs, func(_, pw string) bool { return pw == "s3cret" })
			if code == 0 || out.Len() != 0 {
				t.Errorf("code %d, printed:\n%s", code, out.String())
			}
			if want := "locku pin reset: " + foldHome(path) + " can't be read: "; !strings.HasPrefix(errs.String(), want) || strings.Count(errs.String(), "\n") != 1 {
				t.Errorf("stderr must be one line, the path and the reason:\n%s\nwant it to start with %q", errs.String(), want)
			}
			for p, b := range before {
				if now, _ := os.ReadFile(p); !bytes.Equal(now, b) {
					t.Errorf("%s was written:\n%s\nwas:\n%s", filepath.Base(p), now, b)
				}
			}
		})
	}
}

// A value that did not fit is kept aside with no screen too (tdp E3): the
// reset copies the file to .bak, writes it again with the default, says so
// on stderr, and goes on.
func TestPINResetKeepsAsideWhatDidNotFit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOCKU__CONFIG", dir)
	t.Setenv("LOCKU__DATA", filepath.Join(dir, "data"))
	cfg := config.Default()
	if err := cfg.SetPIN("1234"); err != nil {
		t.Fatal(err)
	}
	cfg.Profiles = append(cfg.Profiles, cfg.NewProfile("mine", saver.KindSnake))
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(config.ProfilesPath())
	was := strings.Replace(string(b), "speed: normal", "speed: warp", 1)
	if err := os.WriteFile(config.ProfilesPath(), []byte(was), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if code := pinReset(typing(t, "y", "s3cret"), &out, &errs, func(_, pw string) bool { return pw == "s3cret" }); code != 0 {
		t.Fatalf("code %d:\n%s\n%s", code, out.String(), errs.String())
	}
	if !strings.Contains(errs.String(), "did not fit") || !strings.Contains(errs.String(), "profiles.yaml.bak") {
		t.Errorf("stderr:\n%s", errs.String())
	}
	if bak, _ := os.ReadFile(config.ProfilesPath() + ".bak"); string(bak) != was {
		t.Errorf("the copy:\n%s", bak)
	}
	if now, _ := os.ReadFile(config.ProfilesPath()); strings.Contains(string(now), "warp") {
		t.Errorf("not written again:\n%s", now)
	}
}

// A profile config.yaml names that profiles.yaml does not have is said on
// stderr with no screen (tdp E3), the file left as it is, and the reset
// goes on.
func TestPINResetSaysAProfileThatIsNotThere(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOCKU__CONFIG", dir)
	t.Setenv("LOCKU__DATA", filepath.Join(dir, "data"))
	cfg := config.Default()
	if err := cfg.SetPIN("1234"); err != nil {
		t.Fatal(err)
	}
	cfg.Profile = "ghost"
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if code := pinReset(typing(t, "y", "s3cret"), &out, &errs, func(_, pw string) bool { return pw == "s3cret" }); code != 0 {
		t.Fatalf("code %d:\n%s\n%s", code, out.String(), errs.String())
	}
	if !strings.Contains(errs.String(), `locku pin reset: Profile "ghost" in config.yaml is not in profiles.yaml: using the default`) {
		t.Errorf("stderr:\n%s", errs.String())
	}
	if again, _ := config.Load(); again.Profile != "ghost" {
		t.Errorf("the profile named was changed to %q", again.Profile)
	}
}

// Without a terminal there is nothing to ask on.
func TestPINResetNeedsATerminal(t *testing.T) {
	t.Setenv("LOCKU__CONFIG", t.TempDir())
	f, err := os.CreateTemp(t.TempDir(), "in")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out bytes.Buffer
	if code := pinReset(f, &out, io.Discard, func(string, string) bool { return true }); code != 2 || !strings.Contains(out.String(), "needs a terminal") {
		t.Errorf("code %d:\n%s", code, out.String())
	}
}
