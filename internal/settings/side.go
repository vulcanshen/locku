package settings

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// Panel [1] (ui.md §1.1): four groups, one under the other. Profiles
// first: the ones the user has set up — named, configured savers: the
// objects. Then Savers, the kinds there are — the classes, which have no
// name but their own and are not made or deleted (user, 2026-09-24, who
// drew the line and put the profiles on top). Then Integration: the tools
// that run locku as their screensaver — tmux, screen (user, 2026-09-25).
// Settings is one row, preference. The active profile — the one
// preference › profile names — carries a green dot, and the dot is all it
// is: it shows, it does not set.

// The rows' IDs: what the cursor is kept by when the rows change.
func profileID(name string) string { return "profile:" + name }
func saverID(kind string) string   { return "saver:" + kind }
func toolID(name string) string    { return "tool:" + name }

const preferenceID = "preference"

// sideRows is [1]'s rows, in their groups.
func (s *Settings) sideRows() []terminu.Row {
	var rows []terminu.Row
	for _, p := range s.cfg.Profiles {
		mark := "  "
		if p.Name == s.cfg.Profile {
			mark = terminu.Fg(terminu.Green).Render("●") + " "
		}
		rows = append(rows, terminu.Row{ID: profileID(p.Name), Cells: []string{mark + p.Name}, Group: "Profiles"})
	}
	for _, k := range saver.Kinds {
		rows = append(rows, terminu.Row{ID: saverID(k), Cells: []string{k}, Group: "Savers"})
	}
	for _, t := range tools {
		rows = append(rows, terminu.Row{ID: toolID(t), Cells: []string{t}, Group: "Integration"})
	}
	return append(rows, terminu.Row{ID: preferenceID, Cells: []string{"preference"}, Group: "Settings"})
}

// itemOf is the item a row's id names.
func (s *Settings) itemOf(id string) item {
	switch {
	case strings.HasPrefix(id, "profile:"):
		if i := s.cfg.Index(strings.TrimPrefix(id, "profile:")); i >= 0 {
			return item{kind: kindProfile, ref: i}
		}
	case strings.HasPrefix(id, "saver:"):
		for i, k := range saver.Kinds {
			if saverID(k) == id {
				return item{kind: kindSaver, ref: i}
			}
		}
	case strings.HasPrefix(id, "tool:"):
		for i, t := range tools {
			if toolID(t) == id {
				return item{kind: kindTool, ref: i}
			}
		}
	}
	return item{kind: kindPreference}
}

// sideActions is what can be done to the row under [1]'s cursor (ux.md
// §A.1). Enter on any row is the same thing: over to [2], where the row's
// values are (user, 2026-09-24) — the active profile is set on
// preference › profile, or with [a], not with Enter.
func (s *Settings) sideActions() []terminu.Action {
	edit := terminu.Action{Key: "Enter", Label: "Edit", Desc: "its values, in [2]", Scope: terminu.ItemScope, Hint: "edit",
		ToDetails: func() bool { return true }}
	switch s.cur.kind {
	case kindSaver:
		kind := saver.Kinds[s.cur.ref]
		edit.Desc = "its defaults, in [2]"
		return []terminu.Action{
			edit,
			{Key: "p", Label: "preview", Desc: "the lock, showing this saver with its defaults", Scope: terminu.ItemScope,
				Do: func() tea.Cmd { return s.previewSaver(kind) }},
			s.newAction(kind, terminu.ItemScope),
		}
	case kindTool, kindPreference:
		return []terminu.Action{edit}
	}
	ref := s.cur.ref
	name := s.cfg.Profiles[ref].Name
	active := name == s.cfg.Profile
	return []terminu.Action{
		edit,
		// [a]: the lock shows this one from now on (user, 2026-09-25),
		// without the walk to preference › profile, which stays the other
		// way to the same setting.
		{Key: "a", Label: "activate", Desc: "the lock shows this profile from now on", Scope: terminu.ItemScope,
			Disabled: func() bool { return active },
			Do:       func() tea.Cmd { return s.change(func() { s.cfg.Profile = name }) }},
		{Key: "p", Label: "preview", Desc: "the lock, showing this profile", Scope: terminu.ItemScope,
			Do: func() tea.Cmd { return s.previewProfile(ref) }},
		{Key: "D", Label: "Duplicate", Desc: "a copy, under a new name", Scope: terminu.ItemScope,
			Do: func() tea.Cmd { return s.duplicate(ref) }},
		{Key: "r", Label: "rename", Desc: "this profile", Scope: terminu.ItemScope,
			Do: func() tea.Cmd { return s.rename(ref) }},
		// The last one, or the one the lock shows: there, but not now
		// (tdp M6).
		{Key: "X", Label: "Delete", Desc: "this profile", Scope: terminu.ItemScope,
			Disabled: func() bool { return active || len(s.cfg.Profiles) == 1 },
			Do:       func() tea.Cmd { return s.delete(ref) }},
	}
}

// newAction is [n] on a saver: a profile of it (user, 2026-09-24: a class
// has no name, an object does; what is made is a profile, made as the
// saver's defaults say).
func (s *Settings) newAction(kind string, scope terminu.Scope) terminu.Action {
	return terminu.Action{Key: "n", Label: "new", Desc: "a profile of this saver, under a name", Scope: scope,
		Do: func() tea.Cmd { return s.newProfile(kind) }}
}

// freeName is base while no profile has it, else base numbered from 2.
func (s *Settings) freeName(base string) string {
	name := base
	for i := 2; s.cfg.Index(name) >= 0; i++ {
		name = base + itoa(i)
	}
	return name
}

// nameCheck is a profile's name: not empty, and no other profile's; self
// is the profile being renamed, -1 for a new one.
func (s *Settings) nameCheck(self int) func(string) error {
	return func(v string) error {
		if v == "" {
			return errNameEmpty
		}
		if i := s.cfg.Index(v); i >= 0 && i != self {
			return errNameTaken
		}
		return nil
	}
}

// newProfile asks a name for a profile of kind, made as its defaults say.
// The free name is in the prompt: the grey suggestion is for an old value
// only, and a new profile has none (components/input/README).
func (s *Settings) newProfile(kind string) tea.Cmd {
	free := s.freeName(kind)
	return s.app.Text(terminu.TextInput{Title: "New profile", Subject: kind, Glyph: glyphPencil, Name: "Name",
		Prompt: "A name for the new " + kind + " profile; " + free + " is free", Verb: "create", Check: s.nameCheck(-1),
		Done: func(name string) tea.Cmd {
			cmd := s.change(func() { s.cfg.Profiles = append(s.cfg.Profiles, s.cfg.NewProfile(name, kind)) })
			s.selectProfile(name)
			return cmd
		}})
}

func (s *Settings) duplicate(ref int) tea.Cmd {
	p := s.cfg.Profiles[ref]
	free := s.freeName(p.Name)
	return s.app.Text(terminu.TextInput{Title: "Duplicate", Subject: p.Name, Glyph: glyphPencil, Name: "Name",
		Prompt: "A name for the copy of " + p.Name + "; " + free + " is free", Verb: "create", Check: s.nameCheck(-1),
		Done: func(name string) tea.Cmd {
			cmd := s.change(func() {
				c := p
				c.Name = name
				s.cfg.Profiles = append(s.cfg.Profiles, c)
			})
			s.selectProfile(name)
			return cmd
		}})
}

func (s *Settings) rename(ref int) tea.Cmd {
	old := s.cfg.Profiles[ref].Name
	return s.app.Text(terminu.TextInput{Title: "Rename", Subject: old, Glyph: glyphPencil, Name: "Name",
		Prompt: "New name for " + old, Verb: "rename", Old: old, Check: s.nameCheck(ref),
		Done: func(name string) tea.Cmd {
			cmd := s.change(func() {
				s.cfg.Profiles[ref].Name = name
				if s.cfg.Profile == old {
					s.cfg.Profile = name
				}
			})
			s.selectProfile(name)
			return cmd
		}})
}

func (s *Settings) delete(ref int) tea.Cmd {
	name := s.cfg.Profiles[ref].Name
	return s.app.Confirm(terminu.Ask{Title: "Delete", Subject: name, Question: "Delete " + name + "?", Verb: "delete",
		Details: []string{"config.yaml is written at once"}},
		func() tea.Cmd {
			i := s.cfg.Index(name)
			if i < 0 {
				return nil
			}
			cmd := s.change(func() { s.cfg.Profiles = append(s.cfg.Profiles[:i:i], s.cfg.Profiles[i+1:]...) })
			// The cursor stays among the profiles: the next one, or the new last.
			s.selectProfile(s.cfg.Profiles[min(i, len(s.cfg.Profiles)-1)].Name)
			return cmd
		})
}

// selectProfile puts [1]'s cursor on the profile called name, and [2] on it.
func (s *Settings) selectProfile(name string) {
	s.side.Select(profileID(name))
	s.refresh()
}

// edit changes what [2] is on — the profile, or the saver's defaults —
// through f.
func (s *Settings) edit(f func(*config.Profile)) tea.Cmd {
	return s.change(func() {
		switch s.cur.kind {
		case kindProfile:
			f(&s.cfg.Profiles[s.cur.ref])
		case kindSaver:
			kind := saver.Kinds[s.cur.ref]
			p := s.cfg.Saver(kind)
			f(&p)
			s.cfg.SetSaver(kind, p)
		}
	})
}

// subject is what [2] edits — a profile, or a saver's defaults; ok is
// false elsewhere.
func (s *Settings) subject() (p config.Profile, ok bool) {
	switch s.cur.kind {
	case kindProfile:
		return s.cfg.Profiles[s.cur.ref], true
	case kindSaver:
		return s.cfg.Saver(saver.Kinds[s.cur.ref]), true
	}
	return config.Profile{}, false
}
