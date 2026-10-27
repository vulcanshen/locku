package settings

import (
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/custom"
	"github.com/vulcanshen/locku/internal/saver"
	"github.com/vulcanshen/locku/internal/ui"
)

// Preview (ui.md §2.1): the whole terminal becomes the lock, as a desktop
// screensaver's preview does, and any key hands it back to the settings
// screen as it was — no PIN asked (user, 2026-09-25). The lib's screen is
// handed over with tea.Exec, the lock run on the terminal meanwhile; a
// custom saver's program is given the terminal itself, and one that ends,
// or none set, is the word on the board (function.md §5.5). One that is
// over goes in the App Log, an operation that only looks (tdp note.7,
// 2026-10-27).

// previewProfile is [p] on a profile, or [P] on its [2]: the lock as it
// would look with this profile active, whether or not it is.
func (s *Settings) previewProfile(ref int) tea.Cmd {
	cfg := s.snapshot()
	p := cfg.Profiles[ref]
	cfg.Profile = p.Name
	return preview(cfg, p, "profile "+p.Name)
}

// previewSaver is [p] on a saver, or [P] on its [2]: a profile made of
// its defaults, shown (user, 2026-09-24).
func (s *Settings) previewSaver(kind string) tea.Cmd {
	cfg := s.snapshot()
	p := cfg.Saver(kind)
	p.Name = kind + " (defaults)"
	cfg.Profiles = append(cfg.Profiles, p)
	cfg.Profile = p.Name
	return preview(cfg, p, kind+" defaults")
}

// previewActive is [P] on preference's or a tool's [2]: the lock as it
// is, the active profile.
func (s *Settings) previewActive() tea.Cmd {
	cfg := s.snapshot()
	p, _ := cfg.Active()
	return preview(cfg, p, "profile "+p.Name)
}

// preview shows p; what is what is shown, as the App Log says it, the
// words of Settings.wrote: "profile clock2", "clock defaults".
func preview(cfg config.Config, p config.Profile, what string) tea.Cmd {
	return tea.Exec(&previewRun{cfg: cfg, p: p}, func(err error) tea.Msg { return previewDone{what, err} })
}

// previewDone says the preview of what is over, with what went wrong, if
// anything.
type previewDone struct {
	what string
	err  error
}

// previewRun is the preview as a command Bubble Tea hands the terminal to.
type previewRun struct {
	cfg config.Config
	p   config.Profile
}

func (r *previewRun) SetStdin(io.Reader)  {}
func (r *previewRun) SetStdout(io.Writer) {}
func (r *previewRun) SetStderr(io.Writer) {}

func (r *previewRun) Run() error {
	if r.p.Saver != saver.KindCustom {
		return ui.RunPreview(r.cfg)
	}
	if strings.TrimSpace(r.p.Command) == "" {
		return ui.RunPreviewWord(r.cfg, custom.NoCommand())
	}
	if o := custom.Preview(r.p.Command, terminu.IconWidth(), os.Stdin, os.Stdout); o != nil {
		return ui.RunPreviewWord(r.cfg, *o)
	}
	return nil
}
