package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dirs points the config and data directories at temporary ones.
func dirs(t *testing.T) {
	t.Helper()
	t.Setenv("LOCKU__CONFIG", t.TempDir())
	t.Setenv("LOCKU__DATA", t.TempDir())
}

// withProfiles is the defaults with the clock and a second one, clock2.
func withProfiles() Config {
	cfg := Default()
	second := DefaultProfile()
	second.Name = "clock2"
	cfg.Profiles = append(cfg.Profiles, second)
	return cfg
}

func names(cfg Config) string {
	var out []string
	for _, p := range cfg.Profiles {
		out = append(out, p.Name)
	}
	return strings.Join(out, ",")
}

func has(t *testing.T, path, s string) bool {
	t.Helper()
	b, _ := os.ReadFile(path)
	return strings.Contains(string(b), s)
}

// On a machine new to locku the profiles go to profiles.yaml, data, and
// config.yaml has none (tdp E3; user, 2026-10-20).
func TestProfilesAreData(t *testing.T) {
	dirs(t)
	if err := Save(withProfiles()); err != nil {
		t.Fatal(err)
	}
	if !has(t, ProfilesPath(), "name: clock2") || has(t, Path(), "profiles:") || !has(t, Path(), "profile: clock") {
		t.Fatal("the profiles in profiles.yaml, the one in use in config.yaml")
	}
	if cfg, note := Load(); note != "" || names(cfg) != "clock,clock2" {
		t.Errorf("read back: %q %q", names(cfg), note)
	}
}

// A config.yaml from before has its profiles: they are read and written
// there until the settings screen moves them, once, to profiles.yaml
// (tdp E3, when the place changes).
func TestProfilesStayUntilMoved(t *testing.T) {
	dirs(t)
	if err := SaveFile(Path(), withProfiles()); err != nil {
		t.Fatal(err)
	}
	cfg, _ := Load()
	if names(cfg) != "clock,clock2" {
		t.Fatalf("read from config.yaml: %q", names(cfg))
	}
	cfg.ShowStatus = false
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ProfilesPath()); err == nil || !has(t, Path(), "name: clock2") {
		t.Fatal("saved before the move: config.yaml keeps the profiles, no profiles.yaml")
	}
	if moved, err := MoveProfiles(&cfg); !moved || err != nil {
		t.Fatalf("moved %v: %v", moved, err)
	}
	if !has(t, ProfilesPath(), "name: clock2") || has(t, Path(), "profiles:") || !has(t, Path(), "show_status: false") {
		t.Fatal("moved: the profiles in profiles.yaml alone, the rest as it was")
	}
	if moved, _ := MoveProfiles(&cfg); moved {
		t.Error("moved twice")
	}
	if again, _ := Load(); names(again) != "clock,clock2" || again.Both() {
		t.Errorf("after the move: %q, both %v", names(again), again.Both())
	}
}

// Profiles in both places: profiles.yaml's are used, config.yaml's left
// where they are for the user to merge, written back as they were.
func TestProfilesInBothPlacesAreKept(t *testing.T) {
	dirs(t)
	if err := SaveFile(Path(), withProfiles()); err != nil {
		t.Fatal(err)
	}
	if err := saveProfiles(ProfilesPath(), []Profile{NewProfile("snake", "snake")}); err != nil {
		t.Fatal(err)
	}
	cfg, _ := Load()
	if names(cfg) != "snake" || !cfg.Both() {
		t.Fatalf("%q, both %v", names(cfg), cfg.Both())
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	if !has(t, Path(), "name: clock2") || !has(t, ProfilesPath(), "name: snake") {
		t.Error("a save lost the profiles left in config.yaml")
	}
}

// A profiles.yaml that cannot be read: the default profile, the PIN kept,
// and the file never written over (tdp E3: the settings screen moves it
// aside; the lock goes on).
func TestUnreadableProfiles(t *testing.T) {
	dirs(t)
	cfg := withProfiles()
	if err := cfg.SetPIN("1234"); err != nil {
		t.Fatal(err)
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	bad := "profiles: [not yaml\n"
	if err := os.WriteFile(ProfilesPath(), []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UnreadableProfiles(); err == nil {
		t.Fatal("no reason")
	}
	got, note := Load()
	if !strings.HasPrefix(note, "profiles.yaml:") || names(got) != "clock" || !got.CheckPIN("1234") {
		t.Fatalf("%q %q, PIN %v", names(got), note, got.CheckPIN("1234"))
	}
	got.ShowStatus = false
	if err := Save(got); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(ProfilesPath()); string(b) != bad {
		t.Errorf("written over:\n%s", b)
	}
}

// A move that cannot be made leaves the profiles where they were, whole
// (tdp E3: the old place goes on in use).
func TestAMoveThatFailsLeavesThem(t *testing.T) {
	dirs(t)
	if err := SaveFile(Path(), withProfiles()); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(t.TempDir(), "data")
	if err := os.Mkdir(data, 0o500); err != nil { // there, and not to be written in
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(data, 0o700) })
	t.Setenv("LOCKU__DATA", data)
	cfg, _ := Load()
	if moved, err := MoveProfiles(&cfg); moved || err == nil {
		t.Fatalf("moved %v: %v", moved, err)
	}
	if !has(t, Path(), "name: clock2") {
		t.Error("the profiles left config.yaml")
	}
	cfg.ShowStatus = false
	if err := Save(cfg); err != nil || !has(t, Path(), "name: clock2") {
		t.Errorf("saved after the failed move: %v", err)
	}
}
