package settings

import (
	"path/filepath"
	"regexp"
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
// to it, no colour of locku's own drawn as text (tdp L4, L1, E5).
func TestTDP(t *testing.T) {
	tdptest.Check(t, tdptest.App{Name: "locku", New: newApp})
}

// colourCode is a value that is a colour code: drawn by its name, as it
// must be, it is what Check takes for a colour of locku's own left in a
// value (LK21), so the walk below has the rest of Tricky and the test after
// it the colour codes.
var colourCode = regexp.MustCompile("\x1b\\[[0-9;]*m|\u009b[0-9;]*m")

// trickyConfig is a config.yaml of tricky values: every one a profile's
// name, every other one a custom saver whose command is another, the active
// one active, and tmux's file and key (components, control characters).
func trickyConfig(values []string, active string) config.Config {
	cfg := config.Default()
	cfg.Profiles = nil
	for i, v := range values {
		p := config.NewProfile(v, saver.KindClock)
		if i%2 == 1 {
			p = config.NewProfile(v, saver.KindCustom)
			p.Command = values[(i+7)%len(values)]
		}
		cfg.Profiles = append(cfg.Profiles, p)
	}
	cfg.Profile = active
	cfg.Tmux.Conf, cfg.Tmux.BindKey = "/tmp/tab\there", "C-\x1b"
	return cfg
}

// The walk over a config.yaml full of values that have broken screens.
func TestTDPWithTrickyValues(t *testing.T) {
	var values []string
	for _, v := range tdptest.Tricky {
		if !colourCode.MatchString(v) {
			values = append(values, v)
		}
	}
	dir := func(home string) string { return filepath.Join(home, "lockucfg") }
	tdptest.Check(t, tdptest.App{Name: "locku", New: newApp,
		Env: func(home string) map[string]string { return map[string]string{"LOCKU__CONFIG": dir(home)} },
		Seed: func(home string) {
			if err := config.SaveFile(filepath.Join(dir(home), "config.yaml"), trickyConfig(values, "tab\there")); err != nil {
				panic(err)
			}
		}})
}

// Every Tricky value, the colour codes too, on the screen and in what opens
// over it: a colour code in a name is drawn by its name, never obeyed
// (components, control characters); every frame each row the terminal's
// width, nothing sent to the terminal as it is.
func TestTrickyValuesAreDrawnByName(t *testing.T) {
	t.Setenv("LOCKU__CONFIG", t.TempDir())
	t.Setenv("LOCKU__DATA", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	if err := config.Save(trickyConfig(tdptest.Tricky, "\x1b[31mred\x1b[0m")); err != nil {
		t.Fatal(err)
	}
	for _, sz := range [][2]int{{minW, minH}, {80, 24}, {120, 40}} {
		d := tdptest.Start(t, newApp(), sz[0], sz[1])
		if !d.Has(`\x1b[31mred`) {
			t.Errorf("%v: the active profile's name is not drawn by name", sz)
		}
		d.Press("Space", "Esc", "?", "Esc", "!", "Esc", "Tab", "Space", "Esc", "?", "Esc", "j", "Enter", "Esc", "1", "G", "Space", "Esc")
	}
}
