// Package settings is the bare `locku`: the settings screen, built on
// terminu-go, the family's lib (tdp E8) — the lib draws the screen, the
// popups and the core keys from what is registered here, and this package
// says what there is and what each thing does (ui.md §1.1).
//
// One screen of two panels. [1] is the items, in four groups: the
// profiles — named savers, the objects — then the savers, the kinds there
// are, then Integration's tmux and screen, then preference (user,
// 2026-09-24 and 2026-09-25). [2] is the details of the item under [1]'s
// cursor, one value a row, each written to config.yaml the moment it is
// set (components/layout/panel, one value per row).
package settings

import (
	_ "embed"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
	"github.com/vulcanshen/locku/internal/version"
)

// icon is docs/icon.svg, which go:embed cannot reach: the splash draws it
// cell for cell (tdp E6, S5), and TestIconIsTheDocsIcon holds the two the same.
//
//go:embed icon.svg
var icon []byte

// intro is the splash's line under the name and the version (tdp S5).
const intro = "A screensaver with a PIN, for the terminal"

type kind int

const (
	kindProfile    kind = iota // a profile, by its index in cfg.Profiles
	kindSaver                  // a kind of saver, by its index in saver.Kinds
	kindTool                   // a tool, by its index in tools
	kindPreference             // the one settings row
)

// tools is the Integration group, in order.
var tools = []string{"tmux", "screen"}

const (
	toolTmux = iota
	toolScreen
)

// item is what [1]'s cursor can be on.
type item struct {
	kind kind
	ref  int
}

// Settings is the settings screen: the config it edits and the lib's app
// that shows it.
type Settings struct {
	cfg     config.Config
	problem string // Load's note: told once, as a toast
	app     *terminu.App
	side    *terminu.List
	vals    *terminu.Values
	p1, p2  *terminu.Panel
	cur     item
	shown   item // the item [2] was last filled for
	// pinNew is the new PIN between its two boxes: typed, then typed again.
	pinNew string
}

// New is the settings screen on cfg; problem is Load's note.
func New(cfg config.Config, problem string) (*Settings, error) {
	s := &Settings{cfg: cfg, problem: problem, shown: item{kind: -1}}
	a := terminu.New("locku")
	a.QuitDesc = "leave locku"
	a.MinWidth, a.MinHeight = minW, minH
	splash, err := terminu.NewSplash(icon, "locku", version.Display(), intro)
	if err != nil {
		return nil, err
	}
	a.Splash = splash
	a.OnMsg = s.onMsg
	a.Start = s.start
	s.app = a
	s.side = &terminu.List{OnMove: func(r terminu.Row) tea.Cmd { s.moved(r.ID); return nil }}
	s.vals = &terminu.Values{}
	s.p2 = &terminu.Panel{Body: s.vals}
	s.p1 = &terminu.Panel{Label: "locku", Body: s.side, Details: s.p2}
	a.AddScreen(&terminu.Screen{Name: "Settings", Panels: []*terminu.Panel{s.p1, s.p2}, Layout: layout, NarrowBelow: narrowBelow})
	s.side.SetRows(s.sideRows())
	// The cursor starts on the active profile: what one most often came
	// to change (ui.md §1.1).
	if i := cfg.Index(cfg.Profile); i >= 0 {
		s.side.Select(profileID(cfg.Profiles[i].Name))
	}
	if r, ok := s.side.Current(); ok {
		s.moved(r.ID)
	}
	return s, nil
}

// start is what the settings screen sees to as it starts, in order. A file
// of locku's that cannot be read — config.yaml, profiles.yaml — is not
// gone on with: the next write would put the defaults over it. Moved aside
// it is kept whole, as <name>.bak; Esc leaves, the reason on stderr (tdp
// E3, when it cannot be read; until 2026-10-20 an error popup over the
// defaults). Then the profiles move to their own file, the first time, and
// what is wrong in the files is said.
func (s *Settings) start() tea.Cmd {
	if err := config.Unreadable(); err != nil {
		return s.app.MoveAside(config.Path(), err, s.fresh("config.yaml"))
	}
	if err := config.UnreadableProfiles(); err != nil {
		return s.app.MoveAside(config.ProfilesPath(), err, s.fresh("profiles.yaml"))
	}
	var said tea.Cmd
	if s.problem != "" {
		said = s.app.Error("config.yaml", s.problem)
	}
	return tea.Batch(s.moveProfiles(), said)
}

// fresh is name made anew once the one that could not be read is moved
// aside: the files read again, the defaults in its place, written now —
// the other file as it was — and the rest of the start (tdp E3).
func (s *Settings) fresh(name string) func() tea.Cmd {
	return func() tea.Cmd {
		s.cfg, s.problem = config.Load()
		if err := config.Save(s.cfg); err != nil {
			return s.app.Error("Write failed", err.Error())
		}
		s.app.Log(terminu.Info, name+" made anew, with the defaults")
		s.refresh()
		return s.start()
	}
}

// moveProfiles moves the profiles out of config.yaml into their own file,
// the first time (tdp E3, when the place changes): a toast says from where
// to where. One that cannot be made, and profiles in both places, open an
// error popup with both paths.
func (s *Settings) moveProfiles() tea.Cmd {
	from, to := tilde(config.Path()), tilde(config.ProfilesPath())
	if s.cfg.Both() {
		return s.app.Error("Profiles", "The profiles are in "+to+", and in "+from+" too: "+to+"'s are used, "+from+"'s left there to be merged by hand.")
	}
	moved, err := config.MoveProfiles(&s.cfg)
	switch {
	case err != nil:
		return s.app.Error("Profiles", "The profiles could not be moved from "+from+" to "+to+": "+err.Error()+". They stay in "+from+".")
	case moved:
		return s.app.Toast(terminu.Info, "Profiles moved from "+from+" to "+to)
	}
	return nil
}

// tilde is path with the home directory written ~.
func tilde(path string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~" + path[len(home):]
	}
	return path
}

// App is the lib's app: what Run runs, and what the tests drive.
func (s *Settings) App() *terminu.App { return s.app }

// Run runs the settings screen until it is left.
func (s *Settings) Run() error { return s.app.Run() }

// sideW is [1]'s width, webu's (ui.md §1.3); narrowBelow the width under
// which only the focused panel is drawn. minW × minH is the least size
// (tdp L1), lower than the family's 80 × 40 so a tmux pane will do: the
// colour picker, the largest box here at 36 × 11, fits whole inside [2]'s
// frame, under its header and over the footer (2026-10-19).
const (
	sideW       = 24
	narrowBelow = 60
	minW, minH  = 40, 15
)

func layout(w, h int) []terminu.Rect {
	side := min(sideW, w/2)
	return []terminu.Rect{{X: 0, Y: 0, W: side, H: h}, {X: side, Y: 0, W: w - side, H: h}}
}

// moved is [1]'s cursor coming to the row id: [2] follows it at once
// (components/layout/panel, details).
func (s *Settings) moved(id string) {
	s.cur = s.itemOf(id)
	s.fill()
}

// refresh is after a change: [1]'s rows again, the cursor kept on its
// item, and [2] filled again.
func (s *Settings) refresh() {
	s.side.SetRows(s.sideRows())
	if r, ok := s.side.Current(); ok {
		s.cur = s.itemOf(r.ID)
	}
	s.fill()
}

// fill puts [2] on the item under [1]'s cursor. Another item starts [2]
// over at its first row; the same one keeps its place.
func (s *Settings) fill() {
	if s.cur != s.shown {
		s.vals = &terminu.Values{Header: [2]string{"Property", "Value"}}
		s.vals.OnMove = func(*terminu.Field, int) tea.Cmd { s.p2.Actions = s.detailActions(); return nil }
		s.p2.Body = s.vals
		s.shown = s.cur
	}
	s.vals.Fields = s.fields()
	s.p2.Label = s.detailLabel()
	s.p1.Actions = s.sideActions()
	s.p2.Actions = s.detailActions()
}

// snapshot is a copy of the config a change can be undone to when the
// write fails: the profiles and the savers cloned, what a change edits.
func (s *Settings) snapshot() config.Config {
	before := s.cfg
	before.Profiles = slices.Clone(s.cfg.Profiles)
	before.Savers = map[string]config.Profile{}
	for k, v := range s.cfg.Savers {
		before.Savers[k] = v
	}
	return before
}

// save writes config.yaml now — every change is saved the moment it is
// made — and, when the write fails, puts the config back as it was and
// says so (tdp F5): an action that failed, an error popup, the whole of it
// there (components/dialog/toast, dialog/note).
func (s *Settings) save(before config.Config) tea.Cmd {
	defer s.refresh()
	if err := config.Save(s.cfg); err != nil {
		s.cfg = before
		return s.app.Error("Write failed", err.Error())
	}
	return nil
}

// commit is change for a value typed or chosen in a box, its Commit: a
// write that fails puts the config back and is the box's, which stays open
// with the value in it under the error popup (components/layout/popup, the
// action failing).
func (s *Settings) commit(what string, f func()) error {
	before := s.snapshot()
	f()
	defer s.refresh()
	if err := config.Save(s.cfg); err != nil {
		s.cfg = before
		return err
	}
	s.done(what)
	return nil
}

// done puts what was done, once it is written, in the App Log (tdp F5:
// the app's actions and their results); "" when a toast says it already.
func (s *Settings) done(what string) {
	if what != "" {
		s.app.Log(terminu.Success, what)
	}
}

// wrote is a value set, as the App Log says it: "profile clock › bg:
// #F2B753".
func (s *Settings) wrote(label, value string) string {
	where := "preference"
	switch s.cur.kind {
	case kindProfile:
		where = "profile " + s.cfg.Profiles[s.cur.ref].Name
	case kindSaver:
		where = saver.Kinds[s.cur.ref] + " defaults"
	case kindTool:
		where = tools[s.cur.ref]
	}
	if value == "" {
		value = "none"
	}
	return where + " › " + label + ": " + value
}

// tell says what was done in a Success toast when it fits in three rows,
// and in a note to read when it does not: setup's report of a long path
// is read, not glimpsed (components/dialog/toast: how many rows is the app's).
func (s *Settings) tell(title string, lines []string) tea.Cmd {
	text := strings.Join(lines, " · ")
	w, _ := s.app.Size()
	if len(terminu.Wrap(text, max(min(w-2, 120)-4, 1))) <= 3 {
		return s.app.Toast(terminu.Success, text)
	}
	s.done(title + ": " + text) // a toast logs itself, a note does not
	return s.app.Viewer(&terminu.Viewer{Title: title, Glyph: glyphFile, Lines: lines})
}

// change runs f on the config and saves it; what is what was done, for
// the App Log once it is written.
func (s *Settings) change(what string, f func()) tea.Cmd {
	before := s.snapshot()
	f()
	cmd := s.save(before)
	if cmd == nil {
		s.done(what)
	}
	return cmd
}

func (s *Settings) onMsg(msg tea.Msg) tea.Cmd {
	if m, ok := msg.(previewDone); ok && m.err != nil {
		return s.app.Error("Preview", m.err.Error())
	}
	return nil
}

func itoa(n int) string { return strconv.Itoa(n) }

// The title glyphs of the popups locku opens (components/layout/popup: the
// app's choice for an input).
const (
	glyphPencil  = "\U000F03EB" // nf-md-pencil
	glyphLock    = "\U000F033E" // nf-md-lock
	glyphTimer   = "\U000F13AB" // nf-md-timer_outline
	glyphPalette = "\U000F03D8" // nf-md-palette
	glyphTag     = "\U000F04F9" // nf-md-tag
	glyphFile    = "\U000F0214" // nf-md-file
	glyphConsole = "\U000F018D" // nf-md-console
	glyphKey     = "\U000F030C" // nf-md-keyboard
)
