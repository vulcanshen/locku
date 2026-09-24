// Package config reads and writes locku's one file, config.yaml
// (function.md §7). Reading never fails the caller: a missing, unreadable
// or malformed file yields the defaults and a note for the status row,
// because a lock that cannot read its settings must open, not shut —
// locku is not a security boundary (function.md §0.1, §4.3).
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"

	"github.com/vulcanshen/locku/internal/saver"
)

const (
	// DefaultBG is the board's dark cell: surface0. DefaultFG is the lit
	// cell: the family splash's gold (ui.md §4).
	DefaultBG = "#313244"
	DefaultFG = "#f2b753"

	// AuthPIN is the one auth method v1 knows. The key is in the file so a
	// later `auth: pam` changes nothing about its shape (function.md §4.1).
	AuthPIN = "pin"

	// PINMin and PINMax bound a PIN's length in characters (function.md §4.2).
	PINMin = 4
	PINMax = 64

	bcryptCost = 10
)

// Profile is one named, configured saver (function.md §5.2; user,
// 2026-09-24: a saver is the class — the clock, the dino — and a profile
// is the object, the one thing that has a name and can be made). Saver
// says which; the rest is how it shows, and the two colours its board is
// drawn in — the colours are the profile's own, not a global setting.
type Profile struct {
	Name  string `yaml:"name"`
	Saver string `yaml:"saver"`
	// The clock's own: its shapes and its size. A dino leaves them out
	// of the file — it has no size; the canvas draws it as large as the
	// terminal allows (user, 2026-09-24).
	Layout string `yaml:"layout,omitempty"`
	Size   string `yaml:"size,omitempty"`
	Font   string `yaml:"font,omitempty"`
	Time   string `yaml:"time,omitempty"`
	Date   string `yaml:"date,omitempty"`
	BG     string `yaml:"bg"`
	FG     string `yaml:"fg"`
	// The dino run's own (2026-09-24): who runs, and where. A clock
	// leaves them out of the file.
	Runner string `yaml:"runner,omitempty"`
	Scene  string `yaml:"scene,omitempty"`

	// OldType is the key before 2026-09-24, when a profile was a "saver"
	// and its saver a "type": read, carried into Saver, never written.
	OldType string `yaml:"type,omitempty"`
}

// Style is a pair of board colours as "#rrggbb": a profile's, or a draft
// of them on the settings screen.
type Style struct {
	BG string
	FG string
}

// Colours is the profile's pair.
func (p Profile) Colours() Style { return Style{BG: p.BG, FG: p.FG} }

// Config is config.yaml, one field per key.
type Config struct {
	Auth           string    `yaml:"auth"`
	PINHash        string    `yaml:"pin_hash"`
	Profile        string    `yaml:"profile"`
	Profiles       []Profile `yaml:"profiles"`
	ShowStatus     bool      `yaml:"show_status"`
	PromptTimeout  int       `yaml:"prompt_timeout"`
	LockoutAfter   int       `yaml:"lockout_after"`
	LockoutSeconds int       `yaml:"lockout_seconds"`
	// TmuxConf and ScreenConf are the files `locku setup` writes into, as
	// the user typed them — "~/…" allowed. Empty is not set, and setup
	// refuses rather than guesses (user, 2026-09-24).
	TmuxConf   string `yaml:"tmux_conf"`
	ScreenConf string `yaml:"screen_conf"`

	// The keys before 2026-09-24 — `saver` named the active profile and
	// `savers` listed them: read, carried over, never written.
	OldSaver  string    `yaml:"saver,omitempty"`
	OldSavers []Profile `yaml:"savers,omitempty"`
}

// DefaultProfile is the profile a fresh install has, and the one drawn
// when the file names none that exists.
func DefaultProfile() Profile { return NewProfile("clock", saver.KindClock) }

// NewProfile is a profile called name of the saver kind, with that
// saver's defaults and nothing of the other's.
func NewProfile(name, kind string) Profile {
	if kind == saver.KindDino {
		return Profile{Name: name, Saver: kind, Runner: saver.Runners[0], Scene: saver.Scenes[0], BG: DefaultBG, FG: DefaultFG}
	}
	return Profile{Name: name, Saver: saver.KindClock, Layout: "row", Size: "medium", Font: "3x7", Time: "HH MM", Date: "off", BG: DefaultBG, FG: DefaultFG}
}

// Default is the file as it would be with every key left out.
func Default() Config {
	return Config{
		Auth:           AuthPIN,
		Profile:        "clock",
		Profiles:       []Profile{DefaultProfile()},
		ShowStatus:     true,
		PromptTimeout:  30,
		LockoutAfter:   0,
		LockoutSeconds: 30,
	}
}

// Dir is where config.yaml lives: $LOCKU_CONFIG, else $XDG_CONFIG_HOME/locku,
// else ~/.config/locku (ui.md §6).
func Dir() string {
	if d := os.Getenv("LOCKU_CONFIG"); d != "" {
		return d
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "locku")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "locku")
}

// Path is the file itself.
func Path() string { return filepath.Join(Dir(), "config.yaml") }

// Load reads config.yaml. The note is what the status row says when the
// file could not be honoured — "" when it was, or when there is no file,
// which is the ordinary state of a machine that has never run `locku`.
// The Config returned is always usable.
func Load() (Config, string) { return LoadFile(Path()) }

// LoadFile is Load on a given path.
func LoadFile(path string) (Config, string) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, ""
	}
	if err != nil {
		return Default(), "config unreadable: " + err.Error()
	}
	// Unmarshal only touches the keys the file has, so what it leaves out
	// keeps its default — except the profile keys, cleared first so the
	// old names can stand in for them when the file still has those.
	cfg.Profile, cfg.Profiles = "", nil
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Default(), "config.yaml: " + firstLine(err.Error())
	}
	// A pin_hash that is not bcrypt could never match anything: that is a
	// lock with no key, and the file is treated as absent (function.md §4.3).
	if cfg.PINHash != "" {
		if _, err := bcrypt.Cost([]byte(cfg.PINHash)); err != nil {
			return Default(), "pin_hash is not a bcrypt hash"
		}
	}
	cfg.carryOver()
	return cfg.sanitized()
}

// carryOver takes the old keys into the new ones — `saver` into profile,
// `savers` into profiles, a profile's `type` into its saver — and drops
// them, so the next save writes only the new names. A file with neither
// key has the default profile, as before.
func (cfg *Config) carryOver() {
	if cfg.Profiles == nil && cfg.OldSavers != nil {
		cfg.Profiles = cfg.OldSavers
	}
	if cfg.Profiles == nil {
		cfg.Profiles = Default().Profiles
	}
	if cfg.Profile == "" {
		cfg.Profile = cfg.OldSaver
	}
	if cfg.Profile == "" {
		cfg.Profile = Default().Profile
	}
	for i := range cfg.Profiles {
		if cfg.Profiles[i].Saver == "" {
			cfg.Profiles[i].Saver = cfg.Profiles[i].OldType
		}
		cfg.Profiles[i].OldType = ""
	}
	cfg.OldSaver, cfg.OldSavers = "", nil
}

// sanitized brings a parsed file to something the rest of locku can rely
// on, and says what it had to correct. Only the profile reference is
// worth a note (function.md §7): a colour or a number out of range is
// quietly its default (ui.md §1.1).
func (cfg Config) sanitized() (Config, string) {
	note := ""
	if cfg.Auth == "" {
		cfg.Auth = AuthPIN
	}
	var profiles []Profile
	seen := map[string]bool{}
	for _, p := range cfg.Profiles {
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" || seen[p.Name] {
			continue
		}
		seen[p.Name] = true
		if p.Saver == "" {
			p.Saver = saver.KindClock
		}
		// A saver's absent keys are its defaults; the other saver's keys
		// are not its business and are dropped.
		d := NewProfile(p.Name, p.Saver)
		if p.Saver == saver.KindDino {
			if p.Runner == "" {
				p.Runner = d.Runner
			}
			if p.Scene == "" {
				p.Scene = d.Scene
			}
			p.Layout, p.Size, p.Font, p.Time, p.Date = "", "", "", "", ""
		} else {
			if p.Layout == "" {
				p.Layout = d.Layout
			}
			if p.Size == "" {
				p.Size = d.Size
			}
			if p.Font == "" {
				p.Font = d.Font
			}
			if p.Time == "" {
				p.Time = d.Time
			}
			if p.Date == "" {
				p.Date = d.Date
			}
			p.Runner, p.Scene = "", ""
		}
		// A colour that is not "#rrggbb" is quietly its default (ui.md §1.1).
		if !ValidHex(p.BG) {
			p.BG = DefaultBG
		}
		if !ValidHex(p.FG) {
			p.FG = DefaultFG
		}
		p.BG, p.FG = strings.ToLower(p.BG), strings.ToLower(p.FG)
		profiles = append(profiles, p)
	}
	cfg.Profiles = profiles
	switch {
	case len(cfg.Profiles) == 0:
		note = "no profiles"
		cfg.Profiles = []Profile{DefaultProfile()}
		cfg.Profile = DefaultProfile().Name
	case !seen[cfg.Profile]:
		note = fmt.Sprintf("profile %q not found", cfg.Profile)
	}
	if cfg.PromptTimeout < 0 {
		cfg.PromptTimeout = Default().PromptTimeout
	}
	if cfg.LockoutAfter < 0 {
		cfg.LockoutAfter = 0
	}
	if cfg.LockoutSeconds <= 0 {
		cfg.LockoutSeconds = Default().LockoutSeconds
	}
	cfg.TmuxConf = strings.TrimSpace(cfg.TmuxConf)
	cfg.ScreenConf = strings.TrimSpace(cfg.ScreenConf)
	return cfg, note
}

// AbsPath is p with a leading ~/ expanded to the home directory, and
// whether p is a path setup can take: absolute, or under ~. A relative
// path would land wherever setup happened to run.
func AbsPath(p string) (string, bool) {
	switch {
	case strings.HasPrefix(p, "~/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return p, false
		}
		return filepath.Join(home, p[2:]), true
	case filepath.IsAbs(p):
		return filepath.Clean(p), true
	}
	return p, false
}

// Active is the profile the file points at, and whether it exists. When
// it does not, the caller draws DefaultProfile and the status row says so.
func (cfg Config) Active() (Profile, bool) {
	for _, p := range cfg.Profiles {
		if p.Name == cfg.Profile {
			return p, true
		}
	}
	return DefaultProfile(), false
}

// Index is the position of the profile called name, or -1.
func (cfg Config) Index(name string) int {
	for i, p := range cfg.Profiles {
		if p.Name == name {
			return i
		}
	}
	return -1
}

// HasPIN reports whether a PIN is set: the difference between a lock and a
// screensaver (function.md §4.3).
func (cfg Config) HasPIN() bool { return cfg.PINHash != "" }

// CheckPIN reports whether pin is the one set. With no PIN set nothing
// matches — callers never ask, but the answer is still the safe one.
func (cfg Config) CheckPIN(pin string) bool {
	if cfg.PINHash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(cfg.PINHash), []byte(pin)) == nil
}

// SetPIN stores pin, hashed. The length bounds are the file's, not a
// strength policy: locku is not a security boundary.
func (cfg *Config) SetPIN(pin string) error {
	if err := CheckPINLength(pin); err != nil {
		return err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(pin), bcryptCost)
	if err != nil {
		return err
	}
	cfg.PINHash = string(h)
	return nil
}

// CheckPINLength is the one rule a PIN has (function.md §4.2).
func CheckPINLength(pin string) error {
	n := len([]rune(pin))
	if n < PINMin || n > PINMax {
		return fmt.Errorf("%d-%d chars", PINMin, PINMax)
	}
	return nil
}

// ClearPIN goes back to a screensaver.
func (cfg *Config) ClearPIN() { cfg.PINHash = "" }

// Save writes cfg to Path.
func Save(cfg Config) error { return SaveFile(Path(), cfg) }

// SaveFile writes cfg to path atomically — a temp file beside it, then a
// rename — with mode 0600, since the file carries the PIN's hash (ui.md
// §6). The directory is made when it is not there.
func SaveFile(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".config.yaml.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

var hexRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// ValidHex reports whether s is a "#rrggbb" colour.
func ValidHex(s string) bool { return hexRe.MatchString(s) }

// RGB splits a "#rrggbb" colour into its channels; an invalid one is black.
func RGB(hex string) (r, g, b int) {
	if !ValidHex(hex) {
		return 0, 0, 0
	}
	var v int
	fmt.Sscanf(hex[1:], "%06x", &v)
	return v >> 16 & 0xff, v >> 8 & 0xff, v & 0xff
}

// Hex joins three channels into "#rrggbb", each clamped to 0–255.
func Hex(r, g, b int) string {
	c := func(v int) int { return min(255, max(0, v)) }
	return fmt.Sprintf("#%02x%02x%02x", c(r), c(g), c(b))
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
