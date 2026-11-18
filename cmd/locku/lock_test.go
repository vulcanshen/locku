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
