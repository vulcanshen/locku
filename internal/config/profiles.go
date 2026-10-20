package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProfilesPath is where the profiles are: data, under the data directory —
// a list the user writes for locku, even one of how the lock runs (tdp E3,
// v0.9.1; user, 2026-10-20). Until that day they were config.yaml's
// `profiles:`; `profile:`, the one in use, and `savers:`, each kind's
// defaults, are single settings, and stay there.
func ProfilesPath() string { return filepath.Join(DataDir(), "profiles.yaml") }

// ProfilesVersion is profiles.yaml's shape: ProfilesVersion when locku
// writes it.
const ProfilesVersion = 1

// profilesFile is profiles.yaml.
type profilesFile struct {
	Version  int       `yaml:"version"`
	Profiles []Profile `yaml:"profiles"`
}

// readProfiles is profiles.yaml at path: os.ErrNotExist when it is not
// there, an unreadable when it cannot be read.
func readProfiles(path string) ([]Profile, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err != nil {
		return nil, unreadable{"profiles unreadable: " + err.Error(), err.Error()}
	}
	var f profilesFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, unreadable{"profiles.yaml: " + why(err), why(err)}
	}
	return f.Profiles, nil
}

// UnreadableProfiles is why profiles.yaml cannot be read, nil when it can,
// or is not there (tdp E3, when it cannot be read).
func UnreadableProfiles() error {
	_, err := readProfiles(ProfilesPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// Load reads config.yaml and the profiles: profiles.yaml's, or, before
// they are moved there (MoveProfiles), config.yaml's own. The note is what
// the status row says when a file could not be honoured — "" when it was,
// or when there is none, the ordinary state of a machine that has never
// run `locku`. The Config returned is always usable: profiles that cannot
// be read are the default one, the PIN kept.
func Load() (Config, string) {
	cfg, note, had, ok := loadFile(Path())
	if !ok {
		return cfg, note
	}
	ps, err := readProfiles(ProfilesPath())
	var u unreadable
	switch {
	case errors.Is(err, os.ErrNotExist):
		cfg.legacy = had
		return cfg, note
	case errors.As(err, &u):
		cfg.lost = true
		cfg.Profiles = Default().Profiles
		cfg, _ = cfg.sanitized()
		return cfg, u.note
	}
	if had {
		cfg.left = cfg.Profiles
	}
	cfg.Profiles = ps
	return cfg.sanitized()
}

// Save writes cfg: the profiles to profiles.yaml, the rest to config.yaml;
// before the profiles are moved, all of it to config.yaml, as it was.
// Profiles that could not be read are not written over, and config.yaml
// keeps the profiles left in it (Load).
func Save(cfg Config) error {
	if cfg.legacy {
		return SaveFile(Path(), cfg)
	}
	if !cfg.lost {
		if err := saveProfiles(ProfilesPath(), cfg.Profiles); err != nil {
			return err
		}
	}
	c := cfg
	c.Profiles = cfg.left
	return SaveFile(Path(), c)
}

// saveProfiles writes ps to profiles.yaml at path.
func saveProfiles(path string, ps []Profile) error {
	data, err := yaml.Marshal(profilesFile{Version: ProfilesVersion, Profiles: ps})
	if err != nil {
		return err
	}
	return writeAtomic(path, data)
}

// MoveProfiles moves the profiles out of config.yaml into profiles.yaml,
// once, the first time the settings screen starts after 2026-10-20 (tdp
// E3, when the place changes): profiles.yaml written, then config.yaml
// without them. moved reports it did; err is why it could not, and the
// profiles stay where they were, in config.yaml.
func MoveProfiles(cfg *Config) (moved bool, err error) {
	if !cfg.legacy {
		return false, nil
	}
	if err := saveProfiles(ProfilesPath(), cfg.Profiles); err != nil {
		return false, err
	}
	c := *cfg
	c.legacy = false
	if err := Save(c); err != nil {
		os.Remove(ProfilesPath()) // the old place, whole, is the one in use
		return false, err
	}
	cfg.legacy = false
	return true, nil
}

// Both reports profiles left in config.yaml beside profiles.yaml's: those
// are used, config.yaml's are the user's to merge (tdp E3).
func (cfg Config) Both() bool { return cfg.left != nil }
