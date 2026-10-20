package settings

import (
	"path/filepath"
	"testing"

	terminu "github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/tdptest"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// newApp is the settings screen as `locku` makes it, all but Run.
func newApp() *terminu.App {
	s, err := New(config.Load())
	if err != nil {
		panic(err)
	}
	return s.App()
}

// The family's walk (terminu-go's tdptest): the settings screen launched as
// for the first time on many terminal sizes and icon widths, on each panel
// the Space menu, the global operation popup, the App Log and the key
// reference opened and closed, then a terminal under the least size; every
// frame each row exactly the terminal's width, no control character sent
// to it (tdp L4, L1, E5).
func TestTDP(t *testing.T) {
	tdptest.Check(t, tdptest.App{Name: "locku", New: newApp})
}

// The same walk over a config.yaml full of values that have broken screens:
// every tricky one a profile's name, the active one a custom saver whose
// command is tricky too, and tmux's file and key (components, control
// characters).
func TestTDPWithTrickyValues(t *testing.T) {
	dir := func(home string) string { return filepath.Join(home, "lockucfg") }
	tdptest.Check(t, tdptest.App{Name: "locku", New: newApp,
		Env: func(home string) map[string]string { return map[string]string{"LOCKU__CONFIG": dir(home)} },
		Seed: func(home string) {
			cfg := config.Default()
			cfg.Profiles = nil
			for i, v := range tdptest.Tricky {
				p := config.NewProfile(v, saver.KindClock)
				if i%2 == 1 {
					p = config.NewProfile(v, saver.KindCustom)
					p.Command = tdptest.Tricky[(i+7)%len(tdptest.Tricky)]
				}
				cfg.Profiles = append(cfg.Profiles, p)
			}
			cfg.Profile = "\x1b[31mred\x1b[0m"
			cfg.Tmux.Conf, cfg.Tmux.BindKey = "/tmp/tab\there", "C-\x1b"
			if err := config.SaveFile(filepath.Join(dir(home), "config.yaml"), cfg); err != nil {
				panic(err)
			}
		}})
}
