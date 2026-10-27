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
		row := terminu.Row{ID: profileID(p.Name), Cells: []string{p.Name}, Group: "Profiles"}
		if p.Name == s.cfg.Profile {
			// The dot in Green, the app's colour, in the mark column: the two
			// cells the group's rows are further in, the names lined up with
			// the other groups' (tdp panel.7, panel.11, panel-154; LK22, user
			// 2026-10-27).
			row.Flag = terminu.Span{Text: "●", Style: terminu.Fg(terminu.Green)}
		}
		rows = append(rows, row)
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
	// Over to [2], the preview and Delete clear the Space menu they run from:
	// a drill-down, the terminal handed over, the item gone (tdp T1).
	edit := terminu.Action{Key: "Enter", Label: "Edit", Desc: "its values, in [2]", Scope: terminu.ItemScope, Hint: "edit", Clear: true,
		ToDetails: func() bool { return true }}
	switch s.cur.kind {
	case kindSaver:
		kind := saver.Kinds[s.cur.ref]
		edit.Desc = "its defaults, in [2]"
		return []terminu.Action{
			edit,
			{Key: "p", Label: "preview", Desc: "the lock, showing this saver with its defaults", Scope: terminu.ItemScope, Clear: true,
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
			Do: func() tea.Cmd {
				return s.change("The lock shows "+name+" from now on", func() { s.cfg.Profile = name })
			}},
		{Key: "p", Label: "preview", Desc: "the lock, showing this profile", Scope: terminu.ItemScope, Clear: true,
			Do: func() tea.Cmd { return s.previewProfile(ref) }},
		// A box for a name, from the Space menu, opens over it, and Esc goes
		// back to it (tdp F4, T1: a value written back keeps its source).
		{Key: "D", Label: "Duplicate", Desc: "a copy, under a new name", Scope: terminu.ItemScope,
			Do: func() tea.Cmd { return s.duplicate(ref) }},
		{Key: "r", Label: "rename", Desc: "this profile", Scope: terminu.ItemScope,
			Do: func() tea.Cmd { return s.rename(ref) }},
		// The last one, or the one the lock shows: there, but not now
		// (tdp M6). It asks first, over the menu it was run from (tdp F6, F4).
		{Key: "X", Label: "Delete", Desc: "this profile", Scope: terminu.ItemScope, Clear: true,
			Disabled: func() bool { return active || len(s.cfg.Profiles) == 1 },
			Ask: &terminu.Ask{Title: "Delete", Subject: name, Question: "Delete " + name + "?", Verb: "delete",
				Details: []string{"config.yaml is written at once"}},
			Do: func() tea.Cmd { return s.delete(name) }},
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

// newProfile asks a name for a profile of kind, made as its defaults say:
// the free name is offered in grey, Enter on it taking it (tdp input.6). Made,
// the cursor is on it and the focus in [2], to go on with its values
// (ui.md §3.1; tdp K3, S19: the user expects to be taken to it).
func (s *Settings) newProfile(kind string) tea.Cmd {
	return s.app.Text(terminu.TextInput{Title: "New profile", Subject: kind, Glyph: glyphPencil, Name: "Name",
		Prompt: "A name for the new " + kind + " profile", Verb: "create", Suggest: s.freeName(kind), Check: s.nameCheck(-1),
		Commit: func(name string) error {
			return s.commit("Profile "+name+" made, of "+kind, func() { s.cfg.Profiles = append(s.cfg.Profiles, s.cfg.NewProfile(name, kind)) })
		},
		Done: s.made})
}

func (s *Settings) duplicate(ref int) tea.Cmd {
	p := s.cfg.Profiles[ref]
	return s.app.Text(terminu.TextInput{Title: "Duplicate", Subject: p.Name, Glyph: glyphPencil, Name: "Name",
		Prompt: "A name for the copy of " + p.Name, Verb: "create", Suggest: s.freeName(p.Name), Check: s.nameCheck(-1),
		Commit: func(name string) error {
			return s.commit("Profile "+name+" made, a copy of "+p.Name, func() {
				c := p
				c.Name = name
				s.cfg.Profiles = append(s.cfg.Profiles, c)
			})
		},
		Done: s.made})
}

// made is a new profile written: the cursor on it, and the user taken to
// it, the focus in [2] and the Space menu it was made from closed (tdp T1:
// Esc on the box comes back to the menu, the profile made does not).
func (s *Settings) made(name string) tea.Cmd {
	s.selectProfile(name)
	return tea.Batch(s.app.CloseAll(), s.app.Focus(s.p2))
}

func (s *Settings) rename(ref int) tea.Cmd {
	old := s.cfg.Profiles[ref].Name
	return s.app.Text(terminu.TextInput{Title: "Rename", Subject: old, Glyph: glyphPencil, Name: "Name",
		Prompt: "New name for " + old, Verb: "rename", Old: old, Check: s.nameCheck(ref),
		Commit: func(name string) error {
			return s.commit("Profile "+old+" renamed "+name, func() {
				s.cfg.Profiles[ref].Name = name
				if s.cfg.Profile == old {
					s.cfg.Profile = name
				}
			})
		},
		Done: func(name string) tea.Cmd {
			s.selectProfile(name)
			return nil
		}})
}

// delete takes the profile called name away, once its confirm is
// answered (the action's Ask).
func (s *Settings) delete(name string) tea.Cmd {
	i := s.cfg.Index(name)
	if i < 0 {
		return nil
	}
	cmd := s.change("Profile "+name+" deleted", func() { s.cfg.Profiles = append(s.cfg.Profiles[:i:i], s.cfg.Profiles[i+1:]...) })
	// The cursor stays among the profiles: the next one, or the new last.
	s.selectProfile(s.cfg.Profiles[min(i, len(s.cfg.Profiles)-1)].Name)
	return cmd
}

// selectProfile puts [1]'s cursor on the profile called name, and [2] on it.
func (s *Settings) selectProfile(name string) {
	s.side.Select(profileID(name))
	s.refresh()
}

// alter changes what [2] is on — the profile, or the saver's defaults —
// through f, in the config; writing it is the caller's.
func (s *Settings) alter(f func(*config.Profile)) {
	switch s.cur.kind {
	case kindProfile:
		f(&s.cfg.Profiles[s.cur.ref])
	case kindSaver:
		kind := saver.Kinds[s.cur.ref]
		p := s.cfg.Saver(kind)
		f(&p)
		s.cfg.SetSaver(kind, p)
	}
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
