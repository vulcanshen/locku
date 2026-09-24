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

// Saver is one named instance (function.md §5.2): what it shows, how it
// is laid out, and the two colours its board is drawn in — the colours
// are the saver's own, not a global setting (user, 2026-09-24). Type is
// kept for the day a second one exists; v1 draws every instance as a
// clock.
type Saver struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`
	Layout string `yaml:"layout"`
	Size   string `yaml:"size"`
	Font   string `yaml:"font"`
	Time   string `yaml:"time"`
	Date   string `yaml:"date"`
	BG     string `yaml:"bg"`
	FG     string `yaml:"fg"`
}

// Style is a pair of board colours as "#rrggbb": a saver's, or a draft of
// them on the settings screen.
type Style struct {
	BG string
	FG string
}

// Colours is the saver's pair.
func (s Saver) Colours() Style { return Style{BG: s.BG, FG: s.FG} }

// Config is config.yaml, one field per key.
type Config struct {
	Auth           string  `yaml:"auth"`
	PINHash        string  `yaml:"pin_hash"`
	Saver          string  `yaml:"saver"`
	Savers         []Saver `yaml:"savers"`
	ShowStatus     bool    `yaml:"show_status"`
	PromptTimeout  int     `yaml:"prompt_timeout"`
	LockoutAfter   int     `yaml:"lockout_after"`
	LockoutSeconds int     `yaml:"lockout_seconds"`
}

// DefaultSaver is the instance a fresh install has, and the one drawn when
// the file names none that exists.
func DefaultSaver() Saver {
	return Saver{Name: "clock", Type: "clock", Layout: "row", Size: "medium", Font: "3x7", Time: "HH MM", Date: "off", BG: DefaultBG, FG: DefaultFG}
}

// Default is the file as it would be with every key left out.
func Default() Config {
	return Config{
		Auth:           AuthPIN,
		Saver:          "clock",
		Savers:         []Saver{DefaultSaver()},
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
	// keeps its default.
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
	return cfg.sanitized()
}

// sanitized brings a parsed file to something the rest of locku can rely
// on, and says what it had to correct. Only the saver reference is worth
// a note (function.md §7): a colour or a number out of range is quietly
// its default (ui.md §1.1).
func (cfg Config) sanitized() (Config, string) {
	note := ""
	if cfg.Auth == "" {
		cfg.Auth = AuthPIN
	}
	var savers []Saver
	seen := map[string]bool{}
	for _, s := range cfg.Savers {
		s.Name = strings.TrimSpace(s.Name)
		if s.Name == "" || seen[s.Name] {
			continue
		}
		seen[s.Name] = true
		if s.Type == "" {
			s.Type = "clock"
		}
		if s.Layout == "" {
			s.Layout = DefaultSaver().Layout
		}
		if s.Size == "" {
			s.Size = DefaultSaver().Size
		}
		if s.Font == "" {
			s.Font = DefaultSaver().Font
		}
		if s.Time == "" {
			s.Time = DefaultSaver().Time
		}
		if s.Date == "" {
			s.Date = DefaultSaver().Date
		}
		// A colour that is not "#rrggbb" is quietly its default (ui.md §1.1).
		if !ValidHex(s.BG) {
			s.BG = DefaultBG
		}
		if !ValidHex(s.FG) {
			s.FG = DefaultFG
		}
		s.BG, s.FG = strings.ToLower(s.BG), strings.ToLower(s.FG)
		savers = append(savers, s)
	}
	cfg.Savers = savers
	switch {
	case len(cfg.Savers) == 0:
		note = "no savers"
		cfg.Savers = []Saver{DefaultSaver()}
		cfg.Saver = DefaultSaver().Name
	case !seen[cfg.Saver]:
		note = fmt.Sprintf("saver %q not found", cfg.Saver)
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
	return cfg, note
}

// Active is the saver the file points at, and whether it exists. When it
// does not, the caller draws DefaultSaver and the status row says so.
func (cfg Config) Active() (Saver, bool) {
	for _, s := range cfg.Savers {
		if s.Name == cfg.Saver {
			return s, true
		}
	}
	return DefaultSaver(), false
}

// Index is the position of the saver called name, or -1.
func (cfg Config) Index(name string) int {
	for i, s := range cfg.Savers {
		if s.Name == name {
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
