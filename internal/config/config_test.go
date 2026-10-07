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
	if cfg.HasPIN() || cfg.Profile != "clock" || !cfg.ShowStatus || cfg.PINPromptTimeout != 30 || cfg.Tmux.LockAfterTime != 300 || cfg.Tmux.Lock != LockServer || cfg.Screen.Idle != 300 {
		t.Errorf("not the defaults: %+v", cfg)
	}
}

// tmux's lock is one of two: lock-session as written, anything else is
// lock-server (2026-09-25).
func TestTmuxLockIsOneOfTwo(t *testing.T) {
	for _, c := range []struct{ in, want string }{{"lock-session", LockSession}, {"lock-server", LockServer}, {"nonsense", LockServer}, {"", LockServer}} {
		cfg, _ := LoadFile(write(t, "tmux:\n  lock: "+c.in+"\n"))
		if cfg.Tmux.Lock != c.want {
			t.Errorf("lock %q read as %q", c.in, cfg.Tmux.Lock)
		}
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
	if len(cfg.Savers) != 5 || cfg.Saver("clock") != NewProfile("", "clock") || cfg.Saver("runner").Participants != "big" || cfg.Saver("custom") != NewProfile("", "custom") ||
		cfg.Saver("bounce") != NewProfile("", "bounce") || cfg.Saver("snake") != NewProfile("", "snake") {
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
	if c.Size != "medium" || c.FG != "#ffffff" || c.Font != "3x5" || c.Participants != "" || c.Name != "" || c.Saver != "clock" {
		t.Errorf("clock defaults %+v", c)
	}
	if cfg.Saver("runner").Scene != "grassland" {
		t.Errorf("runner defaults %+v", cfg.Saver("runner"))
	}
	if n := cfg.NewProfile("x", "clock"); n.Name != "x" || n.Size != "medium" || n.FG != "#ffffff" {
		t.Errorf("new profile %+v", n)
	}
	if err := SaveFile(p, cfg); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(p)
	if s := string(body); !strings.Contains(s, "savers:\n") || !strings.Contains(s, "\n    clock:\n        saver: clock\n") || !strings.Contains(s, "        size: medium") || !strings.Contains(s, "    runner:") {
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
	if cfg.Profile != "run" || len(cfg.Profiles) != 2 || cfg.Profiles[0].Saver != "runner" || cfg.Profiles[0].Participants != "big" || cfg.Profiles[1].Saver != "clock" {
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
	if d, c := cfg.Profiles[0], cfg.Profiles[1]; d.Size != "" || d.Layout != "" || c.Participants != "" || c.Size != "large" {
		t.Errorf("dino %+v clock %+v", d, c)
	}
	if err := SaveFile(p, cfg); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(p)
	if s := string(body); !strings.Contains(s, "profile: run") || !strings.Contains(s, "profiles:") || !strings.Contains(s, "saver: runner") ||
		!strings.Contains(s, "\n    clock:\n        saver: clock\n") || strings.Contains(s, "type:") || strings.Contains(s, "old_savers") ||
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

// The runner names before 2026-09-25 — trex, two-trex — are read as
// big and big-small, in a profile and in the dino's defaults alike, as
// the runner's participants since 2026-10-07; and the next save writes
// only the new names.
func TestOldRunnerNamesAreCarriedOver(t *testing.T) {
	p := write(t, "profile: one\nprofiles:\n  - name: one\n    saver: dino\n    runner: trex\n  - name: two\n    saver: dino\n    runner: two-trex\nsavers:\n  dino:\n    runner: two-trex\n")
	cfg, note := LoadFile(p)
	if note != "" {
		t.Errorf("note %q", note)
	}
	if cfg.Profiles[0].Participants != "big" || cfg.Profiles[1].Participants != "big-small" || cfg.Saver("runner").Participants != "big-small" || cfg.Profiles[0].Saver != "runner" {
		t.Errorf("profiles %+v, runner defaults %+v", cfg.Profiles, cfg.Saver("runner"))
	}
	if err := SaveFile(p, cfg); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(p)
	if s := string(body); strings.Contains(s, "trex") || strings.Contains(s, "dino") || !strings.Contains(s, "participants: big\n") || !strings.Contains(s, "participants: big-small\n") ||
		!strings.Contains(s, "    runner:\n        saver: runner\n") {
		t.Errorf("saved with the old names:\n%s", s)
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
	cfg.Screen.Bind = "^L"
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
	// screen's key is under screen's own name for binding one, and stays
	// where it is when the tool's file or idle time is stored.
	if b, _ := os.ReadFile(p); back.Screen.Bind != "^L" || !strings.Contains(string(b), "    bind: ^L\n") {
		t.Errorf("screen's bind: %q in\n%s", back.Screen.Bind, b)
	}
	back.SetTool("screen", Tool{Conf: "~/.screenrc", Idle: 45})
	if back.Screen.Bind != "^L" || back.Screen.Conf != "~/.screenrc" || back.Screen.Idle != 45 {
		t.Errorf("SetTool touched the bind: %+v", back.Screen)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(p), ".config.yaml.*")); len(left) != 0 {
		t.Errorf("temp files left: %v", left)
	}
}

// A PIN made for the user is eight digits, and a new one each time.
func TestNewPINIsEightDigits(t *testing.T) {
	a, err := NewPIN()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := NewPIN()
	if len(a) != 8 || strings.Trim(a, "0123456789") != "" || a == b {
		t.Errorf("%q %q", a, b)
	}
	if err := CheckPINLength(a); err != nil {
		t.Errorf("a made PIN must be a PIN: %v", err)
	}
}

// The file's pin_hash, read on its own for a lock already up: the hash,
// none, or not readable.
func TestLoadPINHash(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOCKU__CONFIG", dir)
	if _, ok := LoadPINHash(); ok {
		t.Error("no file must not read as anything")
	}
	cfg := Default()
	cfg.SetPIN("1234")
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	if h, ok := LoadPINHash(); !ok || h != cfg.PINHash {
		t.Errorf("%q %v", h, ok)
	}
	os.WriteFile(Path(), []byte("pin_hash: nope\n"), 0o600)
	if _, ok := LoadPINHash(); ok {
		t.Error("a hash that is no hash must not read")
	}
	os.WriteFile(Path(), []byte("auth: pin\n"), 0o600)
	if h, ok := LoadPINHash(); !ok || h != "" {
		t.Errorf("no pin_hash is none: %q %v", h, ok)
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

// The variables are the family's, APP__NAME, and the old names are not
// read (tdp D6, 2026-09-29).
func TestOldVariableNamesAreNotRead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOCKU__CONFIG", "")
	t.Setenv("LOCKU__DATA", "")
	t.Setenv("LOCKU_CONFIG", dir)
	t.Setenv("LOCKU_DATA", dir)
	if Dir() == dir || DataDir() == dir {
		t.Errorf("an old name was read: %q %q", Dir(), DataDir())
	}
	t.Setenv("LOCKU__CONFIG", dir)
	t.Setenv("LOCKU__DATA", dir)
	if Dir() != dir || DataDir() != dir {
		t.Errorf("the new names: %q %q", Dir(), DataDir())
	}
}

// The bouncing box has no settings and colours of its own (user,
// 2026-10-06): whatever else a bounce profile carries in the file goes,
// and it is written back as its name and its saver.
func TestABounceProfileIsItsNameAndSaver(t *testing.T) {
	p := write(t, "profile: box\nprofiles:\n  - name: box\n    saver: bounce\n    bg: \"#000000\"\n    fg: \"#ffffff\"\n    size: large\n    runner: big\n    command: cmatrix\n")
	cfg, note := LoadFile(p)
	if note != "" {
		t.Errorf("note %q", note)
	}
	if s, ok := cfg.Active(); !ok || s != (Profile{Name: "box", Saver: "bounce"}) {
		t.Errorf("the bounce profile %+v", s)
	}
	if n := cfg.NewProfile("b", "bounce"); n != (Profile{Name: "b", Saver: "bounce"}) {
		t.Errorf("a new one %+v", n)
	}
	if err := SaveFile(p, cfg); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(p)
	if s := string(body); !strings.Contains(s, "  - name: box\n      saver: bounce\n") || strings.Contains(s, "#000000") || strings.Contains(s, "cmatrix") ||
		!strings.Contains(s, "    bounce:\n        saver: bounce\n    clock:") {
		t.Errorf("saved:\n%s", s)
	}
}

// The snake changes colour at every apple, so its colours are its own
// (user, 2026-10-06), as the bouncing box's; its one setting is its
// speed, by name (2026-10-07), normal unless it says — or says one it
// does not have, cells a second as it was among them. Whatever else the
// file has goes, and no other saver keeps a speed.
func TestASnakeProfileIsItsSpeed(t *testing.T) {
	p := write(t, "profile: s\nprofiles:\n  - name: s\n    saver: snake\n    speed: very-fast\n    size: large\n    runner: big\n    command: cmatrix\n    bg: \"#43523d\"\n    fg: \"#ABCDEF\"\n"+
		"  - name: fast\n    saver: snake\n    speed: 12\n  - name: none\n    saver: snake\n  - name: c\n    saver: clock\n    speed: fast\n  - name: d\n    saver: dino\n    speed: fast\n")
	cfg, _ := LoadFile(p)
	if s, _ := cfg.Active(); s != (Profile{Name: "s", Saver: "snake", Speed: "very-fast"}) {
		t.Errorf("the snake profile %+v", s)
	}
	if cfg.Profiles[1].Speed != "normal" || cfg.Profiles[2].Speed != "normal" || cfg.Profiles[3].Speed != "" || cfg.Profiles[4].Speed != "" {
		t.Errorf("speeds %+v", cfg.Profiles)
	}
	if n := cfg.NewProfile("n", "snake"); n != (Profile{Name: "n", Saver: "snake", Speed: "normal"}) {
		t.Errorf("a new one %+v", n)
	}
	if err := SaveFile(p, cfg); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(p)
	if s := string(body); !strings.Contains(s, "  - name: s\n      saver: snake\n      speed: very-fast\n") || !strings.Contains(s, "    snake:\n        saver: snake\n        speed: normal\n") {
		t.Errorf("saved:\n%s", s)
	}
}

// A dino has a character, the T-Rex unless it says (user, 2026-10-06);
// no other saver has one.
func TestADinoHasACharacter(t *testing.T) {
	p := write(t, "profile: d\nprofiles:\n  - name: d\n    saver: dino\n  - name: c\n    saver: dino\n    character: cat\n  - name: k\n    saver: clock\n    character: cat\n  - name: m\n    saver: custom\n    character: cat\n")
	cfg, _ := LoadFile(p)
	if c := cfg.Profiles[0].Character; c != "t-rex" {
		t.Errorf("a dino without one: %q", c)
	}
	if c := cfg.Profiles[1].Character; c != "cat" {
		t.Errorf("a dino with one: %q", c)
	}
	if cfg.Profiles[2].Character != "" || cfg.Profiles[3].Character != "" {
		t.Errorf("not a dino: %+v %+v", cfg.Profiles[2], cfg.Profiles[3])
	}
	if cfg.Saver("runner").Character != "t-rex" || NewProfile("x", "runner").Character != "t-rex" {
		t.Errorf("the runner's defaults %+v", cfg.Saver("runner"))
	}
}

// The runner's background (user, 2026-10-07): day, night or
// time-shifting, time-shifting unless it says — or says one it does not
// have; its bg / fg go, as the others' have, and no other saver keeps a
// background.
func TestARunnerHasABackground(t *testing.T) {
	p := write(t, "profile: r\nprofiles:\n  - name: r\n    saver: runner\n    background: day\n    bg: \"#000000\"\n    fg: \"#ffffff\"\n"+
		"  - name: n\n    saver: runner\n  - name: x\n    saver: runner\n    background: noon\n  - name: c\n    saver: clock\n    background: day\n")
	cfg, _ := LoadFile(p)
	if r := cfg.Profiles[0]; r.Background != "day" || r.BG != "" || r.FG != "" {
		t.Errorf("a runner by day %+v", r)
	}
	if cfg.Profiles[1].Background != "time-shifting" || cfg.Profiles[2].Background != "time-shifting" || cfg.Profiles[3].Background != "" {
		t.Errorf("backgrounds %+v", cfg.Profiles)
	}
	if d := cfg.Saver("runner"); d.Background != "time-shifting" || d.BG != "" || d.FG != "" {
		t.Errorf("the runner's defaults %+v", d)
	}
}

// The file says its version (user, 2026-10-07). One that says none —
// every one from before — is version 0: read with its old names, then
// written over in version 1's and saying so, and read again it is left
// as it is. One at version 1 is read as it is, a name from before 1 no
// longer read as the new, and is not written over; one from a later
// locku neither — saved, it says 1. A file that cannot be honoured is
// not written over.
func TestTheConfigHasAVersion(t *testing.T) {
	p := write(t, "profile: d\nprofiles:\n  - name: d\n    saver: dino\n    runner: small\n    bg: \"#000000\"\n")
	cfg, note := LoadFile(p)
	if note != "" || cfg.Version != 1 || cfg.Profiles[0].Saver != "runner" || cfg.Profiles[0].Participants != "small" {
		t.Errorf("version 0: note %q, %+v", note, cfg)
	}
	body, _ := os.ReadFile(p)
	if s := string(body); !strings.HasPrefix(s, "version: 1\n") || strings.Contains(s, "dino") || strings.Contains(s, "#000000") ||
		!strings.Contains(s, "participants: small\n") || !strings.Contains(s, "background: time-shifting\n") {
		t.Errorf("version 0, written over:\n%s", s)
	}
	if again, _ := LoadFile(p); again.Version != 1 || again.Profiles[0].Participants != "small" {
		t.Errorf("read again %+v", again)
	}
	if after, _ := os.ReadFile(p); string(after) != string(body) {
		t.Errorf("version 1 written over:\n%s", after)
	}

	one := "version: 1\nprompt_timeout: 5\nprofile: d\nprofiles:\n  - name: d\n    saver: runner\n    runner: small\n"
	p = write(t, one)
	if cfg, _ := LoadFile(p); cfg.Version != 1 || cfg.Profiles[0].Participants != "big" || cfg.PINPromptTimeout != 30 {
		t.Errorf("version 1 with names from before: %+v", cfg)
	}
	if after, _ := os.ReadFile(p); string(after) != one {
		t.Errorf("version 1 written over:\n%s", after)
	}

	later := "version: 2\nprofile: c\nprofiles:\n  - name: c\n    saver: clock\n    shade: blue\n"
	p = write(t, later)
	cfg, note = LoadFile(p)
	if after, _ := os.ReadFile(p); note != "" || cfg.Version != 2 || cfg.Profile != "c" || string(after) != later {
		t.Errorf("version 2: note %q, %+v, the file:\n%s", note, cfg, after)
	}
	if err := SaveFile(p, cfg); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(p); !strings.HasPrefix(string(after), "version: 1\n") {
		t.Errorf("saved by this locku:\n%s", after)
	}

	for _, bad := range []string{"profile: [\n", "pin_hash: nope\nprofile: c\n", "version: x\nprofile: c\n"} {
		p = write(t, bad)
		if _, note := LoadFile(p); note == "" {
			t.Errorf("%q: no note", bad)
		}
		if after, _ := os.ReadFile(p); string(after) != bad {
			t.Errorf("%q written over:\n%s", bad, after)
		}
	}
}
