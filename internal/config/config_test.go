package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMissingFileIsTheDefaults(t *testing.T) {
	cfg, note := LoadFile(filepath.Join(t.TempDir(), "config.yaml"))
	if note != "" {
		t.Errorf("note %q for a missing file", note)
	}
	if cfg.HasPIN() || cfg.Profile != "clock" || !cfg.ShowStatus || cfg.PromptTimeout != 30 {
		t.Errorf("not the defaults: %+v", cfg)
	}
}

func TestAbsentKeysKeepTheirDefaults(t *testing.T) {
	cfg, note := LoadFile(write(t, "prompt_timeout: 5\nprofiles:\n  - name: clock\n    bg: \"#000000\"\n"))
	if note != "" {
		t.Errorf("note %q", note)
	}
	if cfg.PromptTimeout != 5 || !cfg.ShowStatus {
		t.Errorf("%+v", cfg)
	}
	s, ok := cfg.Active()
	if !ok || s.Name != "clock" {
		t.Errorf("active %+v %v", s, ok)
	}
	// A profile's absent keys are the defaults too.
	if s.Saver != "clock" || s.BG != "#000000" || s.FG != DefaultFG || s.Layout != "row" || s.Size != "medium" || s.Font != "3x7" || s.Time != "HH MM" || s.Date != "off" {
		t.Errorf("profile %+v", s)
	}
}

func TestMalformedFileFailsOpen(t *testing.T) {
	cfg, note := LoadFile(write(t, "savers: [\n"))
	if !strings.HasPrefix(note, "config.yaml:") {
		t.Errorf("note %q", note)
	}
	if cfg.HasPIN() {
		t.Error("a broken file must not lock")
	}
}

func TestBadHashFailsOpen(t *testing.T) {
	cfg, note := LoadFile(write(t, "pin_hash: nonsense\n"))
	if note != "pin_hash is not a bcrypt hash" {
		t.Errorf("note %q", note)
	}
	if cfg.HasPIN() {
		t.Error("an unusable hash must not lock")
	}
}

func TestProfileNotFoundIsNoted(t *testing.T) {
	cfg, note := LoadFile(write(t, "profile: nope\nprofiles:\n  - name: a\n    saver: clock\n"))
	if note != `profile "nope" not found` {
		t.Errorf("note %q", note)
	}
	if s, ok := cfg.Active(); ok || s.Name != "clock" {
		t.Errorf("active %+v %v", s, ok)
	}
	if len(cfg.Profiles) != 1 || cfg.Profiles[0].Time != "HH MM" || cfg.Profiles[0].Date != "off" {
		t.Errorf("profiles %+v", cfg.Profiles)
	}
}

func TestEmptyProfilesAreTheDefault(t *testing.T) {
	cfg, note := LoadFile(write(t, "profiles: []\n"))
	if note != "no profiles" {
		t.Errorf("note %q", note)
	}
	if len(cfg.Profiles) != 1 || cfg.Profile != "clock" {
		t.Errorf("%+v", cfg)
	}
}

// The keys before 2026-09-24 — saver, savers, a saver's type — are read
// into profile, profiles and a profile's saver, and the next save writes
// only the new names.
func TestOldKeysAreCarriedOver(t *testing.T) {
	p := write(t, "saver: run\nsavers:\n  - name: run\n    type: dino\n  - name: clock\n")
	cfg, note := LoadFile(p)
	if note != "" {
		t.Errorf("note %q", note)
	}
	if cfg.Profile != "run" || len(cfg.Profiles) != 2 || cfg.Profiles[0].Saver != "dino" || cfg.Profiles[0].Runner != "trex" || cfg.Profiles[1].Saver != "clock" {
		t.Errorf("%+v", cfg)
	}
	if err := SaveFile(p, cfg); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(p)
	if s := string(body); !strings.Contains(s, "profile: run") || !strings.Contains(s, "profiles:") || !strings.Contains(s, "saver: dino") ||
		strings.Contains(s, "savers:") || strings.Contains(s, "type:") {
		t.Errorf("saved with the old keys:\n%s", s)
	}
}

func TestBadValuesAreDefaults(t *testing.T) {
	cfg, _ := LoadFile(write(t, "profiles:\n  - name: clock\n    bg: red\n    fg: \"#ABCDEF\"\nprompt_timeout: -1\nlockout_seconds: 0\n"))
	if s := cfg.Profiles[0]; s.BG != DefaultBG || s.FG != "#abcdef" {
		t.Errorf("colours %+v", s.Colours())
	}
	if cfg.PromptTimeout != 30 || cfg.LockoutSeconds != 30 {
		t.Errorf("%+v", cfg)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "deep", "config.yaml")
	cfg := Default()
	cfg.Profiles = append(cfg.Profiles, Profile{Name: "big", Saver: "clock", Time: "HH MM SS", Date: "YYYY-MM-DD"})
	cfg.Profile = "big"
	if err := cfg.SetPIN("1234"); err != nil {
		t.Fatal(err)
	}
	if err := SaveFile(p, cfg); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("mode %v", st.Mode().Perm())
	}
	back, note := LoadFile(p)
	if note != "" {
		t.Errorf("note %q", note)
	}
	if !back.CheckPIN("1234") || back.CheckPIN("0000") {
		t.Error("PIN did not survive the round trip")
	}
	if s, ok := back.Active(); !ok || s.Name != "big" || s.Date != "YYYY-MM-DD" {
		t.Errorf("active %+v %v", s, ok)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(p), ".config.yaml.*")); len(left) != 0 {
		t.Errorf("temp files left: %v", left)
	}
}

func TestPINLength(t *testing.T) {
	cfg := Default()
	if err := cfg.SetPIN("123"); err == nil {
		t.Error("3 chars accepted")
	}
	if err := cfg.SetPIN(strings.Repeat("x", 65)); err == nil {
		t.Error("65 chars accepted")
	}
	if err := cfg.SetPIN("with space ?"); err != nil {
		t.Errorf("printable PIN refused: %v", err)
	}
	if !cfg.CheckPIN("with space ?") {
		t.Error("set PIN does not check")
	}
	cfg.ClearPIN()
	if cfg.HasPIN() || cfg.CheckPIN("") {
		t.Error("cleared PIN still there")
	}
}

func TestCheckPINAcceptsAnyCost(t *testing.T) {
	h, _ := bcrypt.GenerateFromPassword([]byte("abcd"), bcrypt.MinCost)
	cfg := Config{PINHash: string(h)}
	if !cfg.CheckPIN("abcd") {
		t.Error("min-cost hash rejected")
	}
}

func TestHexHelpers(t *testing.T) {
	r, g, b := RGB("#f2b753")
	if r != 242 || g != 183 || b != 83 {
		t.Errorf("%d %d %d", r, g, b)
	}
	if Hex(242, 183, 83) != "#f2b753" || Hex(-1, 300, 0) != "#00ff00" {
		t.Error(Hex(242, 183, 83), Hex(-1, 300, 0))
	}
	if ValidHex("#12345") || ValidHex("123456") || !ValidHex("#ABCdef") {
		t.Error("ValidHex")
	}
}
