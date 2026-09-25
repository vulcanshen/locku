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
	if cfg.HasPIN() || cfg.Profile != "clock" || !cfg.ShowStatus || cfg.PINPromptTimeout != 30 || cfg.Tmux.LockAfterTime != 300 || cfg.Screen.Idle != 300 {
		t.Errorf("not the defaults: %+v", cfg)
	}
}

func TestAbsentKeysKeepTheirDefaults(t *testing.T) {
	cfg, note := LoadFile(write(t, "prompt_timeout: 5\nprofiles:\n  - name: clock\n    bg: \"#000000\"\n"))
	if note != "" {
		t.Errorf("note %q", note)
	}
	if cfg.PINPromptTimeout != 5 || !cfg.ShowStatus {
		t.Errorf("%+v", cfg)
	}
	s, ok := cfg.Active()
	if !ok || s.Name != "clock" {
		t.Errorf("active %+v %v", s, ok)
	}
	// A profile's absent keys are the built-in defaults too.
	if s.Saver != "clock" || s.BG != "#000000" || s.FG != DefaultFG || s.Layout != "row" || s.Size != "large" || s.Font != "3x5" || s.Time != "HH MM SS" || s.Date != "YYYY-MM-DD" {
		t.Errorf("profile %+v", s)
	}
	// And every saver has its defaults, whole, the built-in ones here.
	if len(cfg.Savers) != 2 || cfg.Saver("clock") != NewProfile("", "clock") || cfg.Saver("dino").Runner != "trex" {
		t.Errorf("savers %+v", cfg.Savers)
	}
}

// The file's own defaults for a saver come first — a new profile is made
// of them — and go back whole; a saver the file says nothing about has
// the built-in ones.
func TestSaverDefaultsAreTheFilesOwn(t *testing.T) {
	p := write(t, "savers:\n  clock:\n    size: medium\n    fg: \"#ffffff\"\n    runner: cat\n")
	cfg, note := LoadFile(p)
	if note != "" {
		t.Errorf("note %q", note)
	}
	c := cfg.Saver("clock")
	if c.Size != "medium" || c.FG != "#ffffff" || c.Font != "3x5" || c.Runner != "" || c.Name != "" || c.Saver != "clock" {
		t.Errorf("clock defaults %+v", c)
	}
	if cfg.Saver("dino").Scene != "grassland" {
		t.Errorf("dino defaults %+v", cfg.Saver("dino"))
	}
	if n := cfg.NewProfile("x", "clock"); n.Name != "x" || n.Size != "medium" || n.FG != "#ffffff" {
		t.Errorf("new profile %+v", n)
	}
	if err := SaveFile(p, cfg); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(p)
	if s := string(body); !strings.Contains(s, "savers:\n    clock:") || !strings.Contains(s, "        size: medium") || !strings.Contains(s, "    dino:") {
		t.Errorf("saved:\n%s", s)
	}
	back, _ := LoadFile(p)
	if back.Saver("clock") != c {
		t.Errorf("round trip %+v", back.Saver("clock"))
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
	if len(cfg.Profiles) != 1 || cfg.Profiles[0].Time != "HH MM SS" || cfg.Profiles[0].Date != "YYYY-MM-DD" {
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

// The keys before 2026-09-24 — saver, savers, a saver's type, the
// prompt and lockout settings — are read as the new names, and the next
// save writes only those.
func TestOldKeysAreCarriedOver(t *testing.T) {
	p := write(t, "saver: run\nsavers:\n  - name: run\n    type: dino\n  - name: clock\nprompt_timeout: 5\nlockout_after: 3\nlockout_seconds: 9\ntmux_conf: ~/.tmux.conf\nidle_lock: 45\n")
	cfg, note := LoadFile(p)
	if note != "" {
		t.Errorf("note %q", note)
	}
	if cfg.Profile != "run" || len(cfg.Profiles) != 2 || cfg.Profiles[0].Saver != "dino" || cfg.Profiles[0].Runner != "trex" || cfg.Profiles[1].Saver != "clock" {
		t.Errorf("%+v", cfg)
	}
	if cfg.PINPromptTimeout != 5 || cfg.WrongPINAttempts != 3 || cfg.WrongPINCooldown != 9 {
		t.Errorf("the settings under their old names: %+v", cfg)
	}
	// The tools' keys moved under each tool; the one idle time became
	// both tools' own (2026-09-25).
	if cfg.Tmux.Conf != "~/.tmux.conf" || cfg.Tmux.LockAfterTime != 45 || cfg.Screen.Conf != "" || cfg.Screen.Idle != 45 {
		t.Errorf("the tools under their old keys: tmux %+v screen %+v", cfg.Tmux, cfg.Screen)
	}
	// Each saver's keys are its own: a dino has no size or shapes, a
	// clock no runner.
	if d, c := cfg.Profiles[0], cfg.Profiles[1]; d.Size != "" || d.Layout != "" || c.Runner != "" || c.Size != "large" {
		t.Errorf("dino %+v clock %+v", d, c)
	}
	if err := SaveFile(p, cfg); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(p)
	if s := string(body); !strings.Contains(s, "profile: run") || !strings.Contains(s, "profiles:") || !strings.Contains(s, "saver: dino") ||
		!strings.Contains(s, "savers:\n    clock:") || strings.Contains(s, "type:") || strings.Contains(s, "old_savers") ||
		!strings.Contains(s, "wrong_pin_attempts: 3") || strings.Contains(s, "lockout_") || strings.Contains(s, "\nprompt_timeout") ||
		!strings.Contains(s, "tmux:\n    conf: ~/.tmux.conf\n    lock-after-time: 45") || !strings.Contains(s, "screen:\n    conf: \"\"\n    idle: 45") ||
		strings.Contains(s, "tmux_conf") || strings.Contains(s, "\nidle_lock") {
		t.Errorf("saved with the old keys:\n%s", s)
	}
}

// A tool's idle time written as idle_lock — the shape of 2026-09-25's
// morning — is read under the tool's own name for it.
func TestAToolsIdleLockIsCarriedOver(t *testing.T) {
	cfg, note := LoadFile(write(t, "tmux:\n  conf: ~/.tmux.conf\n  idle_lock: 45\nscreen:\n  idle_lock: 0\n"))
	if note != "" || cfg.Tmux.Conf != "~/.tmux.conf" || cfg.Tmux.LockAfterTime != 45 || cfg.Screen.Idle != 0 {
		t.Errorf("note %q, tmux %+v, screen %+v", note, cfg.Tmux, cfg.Screen)
	}
}

func TestBadValuesAreDefaults(t *testing.T) {
	cfg, _ := LoadFile(write(t, "profiles:\n  - name: clock\n    bg: red\n    fg: \"#ABCDEF\"\nprompt_timeout: -1\nlockout_seconds: 0\n"))
	if s := cfg.Profiles[0]; s.BG != DefaultBG || s.FG != "#abcdef" {
		t.Errorf("colours %+v", s.Colours())
	}
	if cfg.PINPromptTimeout != 30 || cfg.WrongPINCooldown != 30 {
		t.Errorf("%+v", cfg)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "deep", "config.yaml")
	cfg := Default()
	cfg.Profiles = append(cfg.Profiles, Profile{Name: "big", Saver: "clock", Time: "HH MM SS", Date: "YYYY-MM-DD"})
	cfg.Profile = "big"
	cfg.Tmux.BindKey = "C-l"
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
	if b, _ := os.ReadFile(p); back.Tmux.BindKey != "C-l" || !strings.Contains(string(b), "    bind-key: C-l\n") {
		t.Errorf("tmux's bind-key: %q in\n%s", back.Tmux.BindKey, b)
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
