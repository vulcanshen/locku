package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// The lock keeps aside what did not fit as any start does (tdp E3): the file
// copied whole to .bak and written again with the default; the lock is no
// TUI screen (tdp K1, P5), so the sentence goes on its status row, after
// what is wrong already.
func TestTheLockKeepsAsideWhatDidNotFit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOCKU__CONFIG", dir)
	t.Setenv("LOCKU__DATA", filepath.Join(dir, "data"))
	cfg := config.Default()
	cfg.Profiles = append(cfg.Profiles, cfg.NewProfile("mine", saver.KindSnake))
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(config.ProfilesPath())
	was := strings.Replace(string(b), "speed: normal", "speed: warp", 1)
	if err := os.WriteFile(config.ProfilesPath(), []byte(was), 0o600); err != nil {
		t.Fatal(err)
	}
	_, note := lockStart(config.Load())
	if !strings.Contains(note, "did not fit") || !strings.Contains(note, "profiles.yaml.bak") {
		t.Errorf("the status row's note: %q", note)
	}
	if bak, _ := os.ReadFile(config.ProfilesPath() + ".bak"); string(bak) != was {
		t.Errorf("the copy:\n%s", bak)
	}
	if now, _ := os.ReadFile(config.ProfilesPath()); strings.Contains(string(now), "warp") {
		t.Errorf("not written again:\n%s", now)
	}
	if _, note := lockStart(config.Load()); note != "" {
		t.Errorf("a file that fits says nothing: %q", note)
	}
}

// When a file cannot be read the lock touches none (tdp E3, a screen outside
// the TUI): a value that did not fit in the other is not kept aside or
// written again, the next settings screen's start doing it; the status row
// says which file cannot be read and why.
func TestTheLockTouchesNoFileWhenOneCannotBeRead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOCKU__CONFIG", dir)
	t.Setenv("LOCKU__DATA", filepath.Join(dir, "data"))
	conf := "version: 1\nprofile: clock\nsavers:\n  runner: {saver: runner, background: dusk}\n"
	if err := os.WriteFile(config.Path(), []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(config.ProfilesPath()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.ProfilesPath(), []byte("profiles: [not yaml\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, note := lockStart(config.Load())
	if now, _ := os.ReadFile(config.Path()); string(now) != conf {
		t.Errorf("config.yaml was written:\n%s", now)
	}
	if _, err := os.Stat(config.Path() + ".bak"); !os.IsNotExist(err) {
		t.Errorf("a .bak was made: %v", err)
	}
	if !strings.Contains(note, "profiles.yaml") {
		t.Errorf("the status row does not say which file: %q", note)
	}
}

// The lock of a random profile is one of the profiles it lists, picked as
// it starts (user, 2026-11-18).
func TestTheLockPicksOneOfARandomProfiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOCKU__CONFIG", dir)
	t.Setenv("LOCKU__DATA", filepath.Join(dir, "data"))
	cfg := config.Default()
	cfg.Profiles = append(cfg.Profiles, cfg.NewProfile("snake", saver.KindSnake),
		config.Profile{Name: "mix", Saver: saver.KindRandom, Profiles: []string{"snake"}})
	cfg.Profile = "mix"
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	if got, note := lockStart(config.Load()); got.Profile != "snake" || note != "" {
		t.Errorf("the lock shows %q, %q", got.Profile, note)
	}
}
