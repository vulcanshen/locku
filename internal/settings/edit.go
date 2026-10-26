package settings

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/setup"
)

// The values' errors, on the box's error row, the value named
// (tdp popup.7).
var (
	errNameEmpty = errors.New("Name can't be empty")
	errNameTaken = errors.New("Name is taken")
	errNotAbs    = errors.New("config file path must be an absolute or ~/ path")
	errWrongPIN  = errors.New("wrong PIN")
	errNotNewPIN = errors.New("not the new PIN")
	errPINLength = fmt.Errorf("PIN must be %d-%d characters", config.PINMin, config.PINMax)
)

func oneKeyErr(name, example string) error { return errors.New(name + " must be one key, " + example) }

// ---- the PIN (ux.md §2.2): one box a step, one question each.

// editPIN is Enter on the PIN row. With none set, a new one and its
// confirmation; with one set, the current one first, then a new one —
// or none: the box shows the PIN that is there as eight grey dots, and
// Backspace on them, then Enter, leaves no PIN (tdp password.4;
// until 2026-10-18 a menu after the current PIN offered New PIN or Remove PIN).
// Each step is its own box over the one before, Esc one step back, and the
// last one done closes them all (tdp F1, F4, T1; 2026-10-18 to 10-19 a step
// closed as the next opened). An unlock box has no prompt row: the titles
// say what to type, CURRENT PIN, NEW PIN, CONFIRM PIN (tdp password.2).
func (s *Settings) editPIN() tea.Cmd {
	if !s.cfg.HasPIN() {
		return s.newPIN(false, 1)
	}
	return s.app.Password(terminu.PasswordInput{Title: "Current PIN", Glyph: glyphLock, Name: "PIN",
		Verb: "next", Unlock: true, MaxLen: config.PINMax, Stay: true,
		Check: func(v string) error {
			switch {
			case v == "":
				return terminu.ErrEmpty
			case !s.cfg.CheckPIN(v):
				return errWrongPIN
			}
			return nil
		},
		Done: func(string, bool) tea.Cmd { return s.newPIN(true, 2) }})
}

// newPIN asks the new PIN, the flow's step-th box; set says one is there
// now, which Enter on nothing keeps and Backspace removes.
func (s *Settings) newPIN(set bool, step int) tea.Cmd {
	return s.app.Password(terminu.PasswordInput{Title: "New PIN", Glyph: glyphLock, Name: "PIN",
		Verb: "next", Set: set, Unlock: true, MaxLen: config.PINMax, Stay: true,
		// Empty is no PIN: the one there removed, or none kept.
		Check: func(v string) error {
			if v != "" && config.CheckPINLength(v) != nil {
				return errPINLength
			}
			return nil
		},
		Commit: func(_ string, clear bool) error {
			if !clear {
				return nil // written once it is typed again
			}
			return s.commit("", s.cfg.ClearPIN) // the toast says it
		},
		// Enter on the grey dots keeps the PIN: this box is closing, and the
		// steps under it go with it.
		Unchanged: func() tea.Cmd { return s.app.CloseTop(step - 1) },
		Done: func(v string, clear bool) tea.Cmd {
			if clear {
				return tea.Batch(s.app.CloseTop(step), s.app.Toast(terminu.Success, "PIN removed"))
			}
			s.pinNew = v
			return s.confirmPIN(step + 1)
		}})
}

// confirmPIN asks the new PIN again, the flow's last box; a different one
// is refused in this box, and is one Esc from typing it over.
func (s *Settings) confirmPIN(step int) tea.Cmd {
	return s.app.Password(terminu.PasswordInput{Title: "Confirm PIN", Glyph: glyphLock, Name: "PIN",
		Verb: "set", Unlock: true, MaxLen: config.PINMax, Stay: true,
		Check: func(v string) error {
			if v != s.pinNew {
				return errNotNewPIN
			}
			return nil
		},
		Commit: func(v string, _ bool) error {
			var err error
			if cerr := s.commit("", func() { err = s.cfg.SetPIN(v) }); cerr != nil {
				return cerr
			}
			return err
		},
		Done: func(string, bool) tea.Cmd {
			s.pinNew = ""
			return tea.Batch(s.app.CloseTop(step), s.app.Toast(terminu.Success, "PIN set"))
		}})
}

// ---- the tools (ux.md §A.1): locku's line written, or taken out.

// askActivate is activate turned: a confirm, then the line in — or out.
// From then on the rows are live (resync), and the confirm says so.
func (s *Settings) askActivate(name string, on bool) tea.Cmd {
	t := s.cfg.Tool(name)
	own := setup.Own(name)
	if on {
		then := "a running server takes it at once"
		if name != tools[toolTmux] {
			then = "LOCKPRG goes into the shell rc too, running screens take the rest at once"
		}
		return s.app.Confirm(terminu.Ask{Title: "Activate", Subject: name, Verb: "activate",
			Details:  []string{"The rows here go into " + own + ", which it reads; " + then + ". From then on a change here is written at once."},
			Question: "Write locku's line into " + t.Conf + "?"},
			func() tea.Cmd {
				var out setup.Report
				err := s.install(name, &out)
				s.refresh()
				return s.reported("Activate", name, &out, err)
			})
	}
	return s.app.Confirm(terminu.Ask{Title: "Deactivate", Subject: name, Verb: "deactivate",
		Details:  []string{"The file is rewritten at once, and " + own + " removed."},
		Question: "Take locku's line out of " + t.Conf + "?"},
		func() tea.Cmd {
			var out setup.Report
			err := s.uninstall(name, &out, t.Conf)
			s.refresh()
			return s.reported("Deactivate", name, &out, err)
		})
}

// install writes the tool's settings as its rows say — onto a running
// tmux server, or running screens, too — reporting into out.
func (s *Settings) install(name string, out *setup.Report) error {
	if name == tools[toolTmux] {
		return setup.Tmux(out, s.cfg.Tmux)
	}
	return setup.Screen(out, s.cfg.Screen)
}

// uninstall takes the tool's line out of the file at conf, and off a
// running tmux server or running screens, reporting into out.
func (s *Settings) uninstall(name string, out *setup.Report, conf string) error {
	if name == tools[toolTmux] {
		return setup.TmuxUndo(out, conf)
	}
	return setup.ScreenUndo(out, conf)
}

// reported is what setup reported about the tool called name, verb done to
// it: a failure in an error popup, ACTIVATE FAILED: tmux, what was done
// before it first, a row each (tdp components.5: the app's own text has no
// line break; until 2026-10-25 one message joined with \n); else the lines,
// in a toast when they fit, in a note when not (tdp toast.2).
// A program's output of several lines, of which the report has the last,
// goes whole into the App Log after them (tdp components.5). out is a
// pointer, read once setup has written it.
func (s *Settings) reported(verb, name string, out *setup.Report, err error) tea.Cmd {
	var lines []string
	if text := strings.TrimSpace(out.String()); text != "" {
		lines = strings.Split(text, "\n")
	}
	var cmd tea.Cmd
	if err != nil {
		cmd = s.app.ErrorAbout(verb+" failed", name, append(lines, err.Error())...)
	} else {
		cmd = s.tell(name, lines)
	}
	for _, all := range out.Whole {
		s.app.Log(terminu.Info, "Output of "+all)
	}
	return cmd
}

// resync is the other half of activate (user, 2026-09-25: turn it on
// once, then what is set is what is in): a row of the tool's changed while
// locku's line was in the file, so its settings are written again as the
// rows now say, and for tmux applied to the running server. With the line
// not in, config.yaml alone changed, and activate stays the user's to turn.
func (s *Settings) resync(name string, installed bool) tea.Cmd {
	if !installed {
		return nil
	}
	var out setup.Report
	return s.reported("Write", name, &out, s.install(name, &out))
}

// moveConf is the tool's file changed from oldConf while locku's line was
// in it: the line out of the old file, then into the new one — and left
// out when the path was cleared.
func (s *Settings) moveConf(name, oldConf string, installed bool) tea.Cmd {
	if !installed {
		return nil
	}
	t := s.cfg.Tool(name)
	var out setup.Report
	if oldConf != t.Conf {
		if err := s.uninstall(name, &out, oldConf); err != nil || t.Conf == "" {
			return s.reported("Move", name, &out, err)
		}
	}
	return s.reported("Move", name, &out, s.install(name, &out))
}
