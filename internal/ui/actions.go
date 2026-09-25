package ui

import (
	"bytes"
	"errors"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/custom"
	"github.com/vulcanshen/locku/internal/saver"
	"github.com/vulcanshen/locku/internal/setup"
)

// action is one thing the user can do to what the cursor is on, or to the
// panel. The Space menu lists actions and the letter hotkeys dispatch
// them, from ONE table, so a letter hotkey that is not a menu row cannot
// exist (§4.2) — and a row that is disabled still answers, with why.
type action struct {
	key      string // "enter" for the core-key action, or one letter
	label    string
	hint     string
	disabled bool
	// panelOp: the action is about the panel, not the row — the menu's
	// second region (VTP §A.1.1).
	panelOp bool
	run     func(*AppModel) tea.Cmd
}

// actions is the table for the current focus and cursor (ux.md §A.1).
func (m AppModel) actions() []action {
	it := m.sideAt()
	if m.focus == panelSide {
		// Enter on any row of [1] is the same thing: over to [2], where
		// the row's rows are (user, 2026-09-24). The active profile is set
		// on preference › profile, not here; the dot only shows.
		edit := action{key: "enter", label: "[Enter] Edit", hint: "its rows, in [2]",
			run: func(a *AppModel) tea.Cmd { a.focus = panelDetail; return nil }}
		switch it.kind {
		case sideSaver:
			edit.hint = "its defaults, in [2]"
			return []action{edit,
				{key: "p", label: "Preview", hint: "the lock, showing this saver with its defaults", run: (*AppModel).previewThis},
				m.newProfileAction()}
		case sideTool, sidePreference:
			return []action{edit}
		}
		del := action{key: "X", label: "Delete", hint: "this profile", run: (*AppModel).deleteProfile}
		switch {
		case len(m.cfg.Profiles) == 1:
			del.disabled, del.hint = true, "cannot delete: last one"
		case m.cfg.Profiles[it.ref].Name == m.cfg.Profile:
			del.disabled, del.hint = true, "cannot delete: active"
		}
		return []action{
			edit,
			{key: "p", label: "Preview", hint: "the lock, showing this profile", run: (*AppModel).previewThis},
			{key: "D", label: "Duplicate", hint: "a copy, under a new name", run: (*AppModel).duplicateProfile},
			{key: "r", label: "Rename", hint: "this profile", run: (*AppModel).renameProfile},
			del,
		}
	}
	var out []action
	switch r := m.rowAt(); r.kind {
	case rowName:
		out = append(out, action{key: "enter", label: "[Enter] Rename", hint: "this profile", run: (*AppModel).renameProfile})
	case rowRunner:
		out = append(out, action{key: "enter", label: "[Enter] Choose", hint: "who runs", run: (*AppModel).chooseRunner})
	case rowScene:
		out = append(out, action{key: "enter", label: "[Enter] Choose", hint: "where it runs", run: (*AppModel).chooseScene})
	case rowCommand:
		out = append(out, action{key: "enter", label: "[Enter] Edit", hint: "the program that draws, as sh -c runs it; empty for none", run: (*AppModel).editCommand})
	case rowLayout:
		out = append(out, action{key: "enter", label: "[Enter] Choose", hint: "a row, or a column of parts", run: (*AppModel).chooseLayout})
	case rowSize:
		out = append(out, action{key: "enter", label: "[Enter] Choose", hint: "small, medium or large digits", run: (*AppModel).chooseSize})
	case rowFont:
		out = append(out, action{key: "enter", label: "[Enter] Choose", hint: "3x7, or the shorter 3x5", run: (*AppModel).chooseFont})
	case rowTime:
		out = append(out, action{key: "enter", label: "[Enter] Choose", hint: "one of four shapes", run: (*AppModel).chooseTime})
	case rowDate:
		out = append(out, action{key: "enter", label: "[Enter] Choose", hint: "off, or one of four shapes", run: (*AppModel).chooseDate})
	case rowChannel:
		out = append(out, action{key: "enter", label: "[Enter] Pick", hint: "0 to 255, into the draft", run: (*AppModel).pickChannel})
	case rowPIN:
		if !m.cfg.HasPIN() {
			out = append(out, action{key: "enter", label: "[Enter] Set PIN", hint: "the lock will ask for it", run: (*AppModel).setPIN})
		} else {
			out = append(out, action{key: "enter", label: "[Enter] Change PIN", hint: "after the current one: a new one, or none", run: (*AppModel).changePIN})
		}
	case rowProfile:
		out = append(out, action{key: "enter", label: "[Enter] Choose", hint: "the profile the lock shows", run: (*AppModel).chooseProfile})
	case rowShowStatus:
		out = append(out, action{key: "enter", label: "[Enter] Toggle", hint: "user@host and the time, on the lock", run: (*AppModel).toggleStatus})
	case rowPINPromptTimeout:
		out = append(out, action{key: "enter", label: "[Enter] Edit", hint: "seconds until the PIN box closes; 0 never", run: (*AppModel).editNumber})
	case rowWrongPINAttempts:
		out = append(out, action{key: "enter", label: "[Enter] Edit", hint: "wrong PINs before a cooldown; 0 off", run: (*AppModel).editNumber})
	case rowWrongPINCooldown:
		out = append(out, action{key: "enter", label: "[Enter] Edit", hint: "the cooldown, in seconds", run: (*AppModel).editNumber})
	case rowConf:
		out = append(out, action{key: "enter", label: "[Enter] Edit", hint: "the file the block goes into", run: (*AppModel).editPath})
	case rowIdle:
		out = append(out, action{key: "enter", label: "[Enter] Edit", hint: "idle seconds before it locks; 0 never", run: (*AppModel).editNumber})
	case rowBindKey:
		out = append(out, action{key: "enter", label: "[Enter] Edit", hint: "the key after prefix that locks; empty binds none", run: (*AppModel).editBindKey})
	case rowActivate:
		out = append(out, m.activateAction())
	case rowLock:
		out = append(out, action{key: "enter", label: "[Enter] Choose", hint: "how much a lock covers: the whole server, or one session", run: (*AppModel).chooseLock})
	}
	if it.kind == sideTool {
		return out
	}
	if p, key, ok := m.subject(); ok {
		what := "this profile"
		if it.kind == sideSaver {
			what = "this saver with its defaults"
			out = append(out, m.newProfileAction())
		}
		save := action{key: "S", label: "Save", hint: "write the colour draft to config.yaml", panelOp: true, run: (*AppModel).saveColours}
		reset := action{key: "R", label: "Reset", hint: "drop the draft: the saved colours again", panelOp: true, run: (*AppModel).resetColours}
		if !m.dirtyOf(key, p) {
			save.disabled, save.hint = true, "nothing to save"
			reset.disabled, reset.hint = true, "nothing changed"
		}
		preview := action{key: "P", label: "Preview", hint: "the lock, showing " + what + " and its draft", panelOp: true, run: (*AppModel).previewThis}
		if p.Saver == saver.KindCustom {
			// No colours, no draft: nothing to save or reset.
			preview.hint = "the program, on the terminal, until a key"
			return append(out, preview)
		}
		out = append(out, preview, save, reset)
	}
	return out
}

// activateAction is Enter on a tool's activate row (user, 2026-09-25:
// property and value like every row — on while locku's block is in the
// file, off while it is not — in place of a button, of the S and X
// hotkeys before it, and of the `locku setup` command before those):
// on writes the block into the file, and onto a running tmux server,
// after a confirm; off takes it out. It says why it cannot when the
// file is not set.
func (m AppModel) activateAction() action {
	name, t := m.tool()
	a := action{key: "enter", label: "[Enter] Activate", hint: "write locku's block into the file", run: (*AppModel).activateTool}
	if setup.Installed(t.Conf) {
		a.label, a.hint, a.run = "[Enter] Deactivate", "take locku's block out of the file", (*AppModel).deactivateTool
	}
	if name == tools[toolTmux] {
		a.hint += ", and a running server"
	} else {
		a.hint += ", and LOCKPRG in the shell rc"
	}
	if t.Conf == "" {
		a.disabled, a.hint = true, "set the config file path first"
	}
	return a
}

// newProfileAction is [n] on a saver: a profile of it (user, 2026-09-24:
// a class has no name, an object does; what is made is a profile, and it
// is made from a saver, as the saver's defaults say).
func (m AppModel) newProfileAction() action {
	return action{key: "n", label: "New", hint: "a profile of this saver, under a name", panelOp: m.focus == panelDetail, run: (*AppModel).newProfile}
}

// previewHere is the global P: on a profile's or a saver's [2], that
// one; anywhere else, the active profile (user, 2026-09-24).
func (m *AppModel) previewHere() tea.Cmd {
	if _, _, ok := m.subject(); ok && m.focus == panelDetail {
		return m.previewThis()
	}
	return m.startPreview(m.previewCfg())
}

// dispatch runs the action bound to key, or says why it cannot.
func (m AppModel) dispatch(key string) (AppModel, tea.Cmd) {
	for _, a := range m.actions() {
		if a.key != key {
			continue
		}
		if a.disabled {
			return m, m.toast.show(a.hint, toastError)
		}
		return m, a.run(&m)
	}
	return m, nil
}

// snapshot is a copy of the config a change can be undone to when the
// write fails. The profiles and the savers are cloned: they are what a
// change edits.
func (m AppModel) snapshot() config.Config {
	before := m.cfg
	before.Profiles = slices.Clone(m.cfg.Profiles)
	before.Savers = map[string]config.Profile{}
	for k, v := range m.cfg.Savers {
		before.Savers[k] = v
	}
	return before
}

// save writes config.yaml now — every change but the colours is saved the
// moment it is made (ui.md §1.1) — and, when the write fails, puts the
// config back as it was and says so.
func (m *AppModel) save(before config.Config) tea.Cmd {
	if err := config.Save(m.cfg); err != nil {
		m.cfg = before
		m.cur1 = clamp(m.cur1, 0, len(m.sideItems())-1)
		return m.toast.show("write failed: "+err.Error(), toastError)
	}
	return nil
}

// edit changes what [2] is on — the profile, or the saver's defaults —
// through f.
func (m *AppModel) edit(f func(*config.Profile)) {
	switch it := m.sideAt(); it.kind {
	case sideProfile:
		f(&m.cfg.Profiles[it.ref])
	case sideSaver:
		kind := saver.Kinds[it.ref]
		p := m.cfg.Saver(kind)
		f(&p)
		m.cfg.SetSaver(kind, p)
	}
}

// editTool changes the tool under the cursor through f.
func (m *AppModel) editTool(f func(*config.Tool)) {
	name, t := m.tool()
	f(&t)
	m.cfg.SetTool(name, t)
}

// previewCfg is the config a preview runs on: the file's, with every
// colour draft in place of the saved colours — a draft is what a preview
// is for.
func (m AppModel) previewCfg() config.Config {
	cfg := m.snapshot()
	for i, p := range cfg.Profiles {
		d := m.draftOf(profileKey(p.Name), p)
		cfg.Profiles[i].BG, cfg.Profiles[i].FG = d.BG, d.FG
	}
	return cfg
}

// ---- panel [1]

// previewThis is [p] on a profile or a saver: the lock as it would look
// with THIS profile active, whether or not it is — or, for a saver, with
// a profile made of its defaults, draft included (user, 2026-09-24).
func (m *AppModel) previewThis() tea.Cmd {
	cfg := m.previewCfg()
	p, key, ok := m.subject()
	if !ok {
		p, _ = cfg.Active()
		return m.previewOf(cfg, p)
	}
	if m.sideAt().kind == sideSaver {
		d := m.draftOf(key, p)
		p.BG, p.FG = d.BG, d.FG
		p.Name = p.Saver + " (defaults)"
		cfg.Profiles = append(cfg.Profiles, p)
	}
	cfg.Profile = p.Name
	return m.previewOf(cfg, p)
}

// previewOf shows cfg's active profile p: the lock inside the settings
// screen — or, for a custom saver, p's program on a terminal of its own,
// the screen given up to it until a key (function.md §5.6); a program
// that ends, or none set, is the word on the board, a preview as any
// other.
func (m *AppModel) previewOf(cfg config.Config, p config.Profile) tea.Cmd {
	if p.Saver != saver.KindCustom {
		return m.startPreview(cfg)
	}
	if strings.TrimSpace(p.Command) == "" {
		return m.startPreviewWord(cfg, custom.NoCommand())
	}
	return tea.Exec(&customPreview{command: p.Command}, func(err error) tea.Msg {
		var e ended
		if errors.As(err, &e) {
			return customPreviewEndMsg{cfg: cfg, outcome: &e.o}
		}
		return customPreviewEndMsg{cfg: cfg}
	})
}

// customPreview is the program's preview as a command Bubble Tea hands
// the terminal to: it runs until a key, or until the program ends, and
// carries an ending back as its error.
type customPreview struct{ command string }

func (c *customPreview) SetStdin(io.Reader)  {}
func (c *customPreview) SetStdout(io.Writer) {}
func (c *customPreview) SetStderr(io.Writer) {}
func (c *customPreview) Run() error {
	if o := custom.Preview(c.command, os.Stdin, os.Stdout); o != nil {
		return ended{*o}
	}
	return nil
}

// ended is a program's outcome, as the error a preview ends with.
type ended struct{ o custom.Outcome }

func (e ended) Error() string { return e.o.Note }

// customPreviewEndMsg says the program's preview is over — with the
// outcome of a program that ended, or nothing for a key.
type customPreviewEndMsg struct {
	cfg     config.Config
	outcome *custom.Outcome
}

// newProfile is [n] on a saver: a name for the profile to make of it,
// offered as the saver's own name while that is free, then numbered.
func (m *AppModel) newProfile() tea.Cmd {
	m.editRef = m.sideAt().ref
	kind := saver.Kinds[m.editRef]
	name := kind
	for i := 2; m.cfg.Index(name) >= 0; i++ {
		name = kind + itoa(i)
	}
	return m.input.ask(inputPopup{title: "name", prompt: "name of the new " + kind + " profile",
		value: name, accept: "create", action: inputNew}, m.layer())
}

func (m *AppModel) duplicateProfile() tea.Cmd {
	m.editRef = m.sideAt().ref
	return m.input.ask(inputPopup{title: "name", prompt: "name of the copy",
		value: m.cfg.Profiles[m.editRef].Name + "2", accept: "create", action: inputDuplicate}, m.layer())
}

func (m *AppModel) renameProfile() tea.Cmd {
	m.editRef = m.sideAt().ref
	return m.input.ask(inputPopup{title: "name", prompt: "name",
		value: m.cfg.Profiles[m.editRef].Name, accept: "rename", action: inputRename}, m.layer())
}

func (m *AppModel) deleteProfile() tea.Cmd {
	ref := m.sideAt().ref
	return m.confirm.ask(confirmPopup{title: "Delete profile", accept: "delete",
		lines:  []string{"Delete " + m.cfg.Profiles[ref].Name + "?", "the config is written at once"},
		action: confirmDeleteProfile, ref: ref}, m.layer())
}

// ---- the tools (ux.md §A.1): the block written, or taken out.

// activateTool is Enter on activate while off: a confirm, then the
// block in. From then on the rows are live (syncTool), and the confirm
// says so.
func (m *AppModel) activateTool() tea.Cmd {
	name, t := m.tool()
	then := "a running server takes it at once"
	if name != tools[toolTmux] {
		then = "LOCKPRG goes into the shell rc too"
	}
	return m.confirm.ask(confirmPopup{title: "Activate " + name + " integration", accept: "activate",
		lines:  []string{"Write locku's block into " + t.Conf + "?", then + "; from then on a change here is written at once"},
		action: confirmActivate, ref: m.sideAt().ref}, m.layer())
}

// deactivateTool is Enter on activate while on: a confirm, then the
// block out.
func (m *AppModel) deactivateTool() tea.Cmd {
	name, t := m.tool()
	return m.confirm.ask(confirmPopup{title: "Deactivate " + name + " integration", accept: "deactivate",
		lines:  []string{"Take locku's block out of " + t.Conf + "?", "the file is rewritten at once"},
		action: confirmDeactivate, ref: m.sideAt().ref}, m.layer())
}

// install writes the tool's block as its rows say — for tmux onto a
// running server too — reporting into out.
func (m AppModel) install(out *bytes.Buffer) error {
	name, t := m.tool()
	if name == tools[toolTmux] {
		return setup.Tmux(out, m.cfg.Tmux)
	}
	return setup.Screen(out, t.Conf, t.Idle)
}

// uninstall takes the tool's block out of the file at conf, and for tmux
// off a running server, reporting into out.
func (m AppModel) uninstall(out *bytes.Buffer, conf string) error {
	if name, _ := m.tool(); name == tools[toolTmux] {
		return setup.TmuxUndo(out, conf)
	}
	return setup.ScreenUndo(out, conf)
}

// reported is what setup printed, as the toast: the error, or the lines.
func (m *AppModel) reported(out bytes.Buffer, err error) tea.Cmd {
	if err != nil {
		return m.toast.show(err.Error(), toastError)
	}
	return m.toast.show(said(out), toastInfo)
}

// syncTool is the other half of activate (user, 2026-09-25: turn it on
// once, then what is set is what is in): a tool's row having changed
// while locku's block is in the file — the file at oldConf, the one
// before this change — the block is written again as the rows now say,
// and for tmux applied to the running server, with no activate to turn
// again; a conf that moved takes the block out of the old file first,
// and one cleared leaves it out. With the block not in, config.yaml
// alone changed, and activate stays the user's to turn.
func (m *AppModel) syncTool(oldConf string) tea.Cmd {
	if !setup.Installed(oldConf) {
		return nil
	}
	_, t := m.tool()
	var out bytes.Buffer
	if oldConf != t.Conf {
		if err := m.uninstall(&out, oldConf); err != nil || t.Conf == "" {
			return m.reported(out, err)
		}
	}
	return m.reported(out, m.install(&out))
}

// said is setup's report as one toast line: its lines, parted by dots.
func said(out bytes.Buffer) string {
	return strings.Join(strings.Split(strings.TrimSpace(out.String()), "\n"), " · ")
}

// ---- panel [2]

// choose opens the options for one of the subject's fields.
func (m *AppModel) choose(title string, values []string, current func(config.Profile) string) tea.Cmd {
	p, _, _ := m.subject()
	return m.openOptions(title, values, current(p), 0)
}

func (m *AppModel) chooseRunner() tea.Cmd {
	return m.choose("runner", saver.Runners, func(p config.Profile) string { return p.Runner })
}

// chooseLock is Enter on tmux's lock: lock-server, or lock-session.
func (m *AppModel) chooseLock() tea.Cmd {
	return m.openOptions("lock", config.TmuxLocks, m.cfg.Tmux.Lock, 0)
}

func (m *AppModel) chooseScene() tea.Cmd {
	return m.choose("scene", saver.Scenes, func(p config.Profile) string { return p.Scene })
}

func (m *AppModel) chooseLayout() tea.Cmd {
	return m.choose("layout", saver.Layouts, func(p config.Profile) string { return p.Layout })
}

func (m *AppModel) chooseSize() tea.Cmd {
	return m.choose("size", saver.Sizes, func(p config.Profile) string { return p.Size })
}

func (m *AppModel) chooseFont() tea.Cmd {
	return m.choose("font", saver.Fonts, func(p config.Profile) string { return p.Font })
}

func (m *AppModel) chooseTime() tea.Cmd {
	return m.choose("time", saver.TimeFormats, func(p config.Profile) string { return p.Time })
}

func (m *AppModel) chooseDate() tea.Cmd {
	return m.choose("date", saver.DateFormats, func(p config.Profile) string { return p.Date })
}

func (m *AppModel) chooseProfile() tea.Cmd {
	names := make([]string, len(m.cfg.Profiles))
	for i, p := range m.cfg.Profiles {
		names[i] = p.Name
	}
	return m.openOptions("profile", names, m.cfg.Profile, 0)
}

func (m *AppModel) pickChannel() tea.Cmd {
	r := m.rowAt()
	values := make([]string, 256)
	for i := range values {
		values[i] = itoa(i)
	}
	title := []string{"bg", "fg"}[r.which] + " · " + string("RGB"[r.ch])
	return m.openOptions(title, values, itoa(r.num), 10)
}

// openOptions lists values with the cursor on current; rows > 0 windows the
// list to that many rows, the cursor centred (ux.md §2.1).
func (m *AppModel) openOptions(title string, values []string, current string, rows int) tea.Cmd {
	items := make([]menuItem, len(values))
	at := 0
	for i, v := range values {
		items[i] = menuItem{label: v, key: "v:" + v}
		if v == current {
			at, items[i].hint = i, "current"
		}
	}
	m.optionsFor = m.rowAt()
	m.options.setItems(items, title, m.layer())
	m.options.rows = rows
	m.options.cursor = at
	m.options.center()
	return m.options.open()
}

// commitOptions is a value chosen from the options list. A colour channel
// goes into the draft; everything else goes into the file at once — into
// the profile, or the saver's defaults, whichever [2] is on.
func (m *AppModel) commitOptions(key string) tea.Cmd {
	v := strings.TrimPrefix(key, "v:")
	before := m.snapshot()
	switch r := m.optionsFor; r.kind {
	case rowRunner:
		m.edit(func(p *config.Profile) { p.Runner = v })
	case rowScene:
		m.edit(func(p *config.Profile) { p.Scene = v })
	case rowLayout:
		m.edit(func(p *config.Profile) { p.Layout = v })
	case rowSize:
		m.edit(func(p *config.Profile) { p.Size = v })
	case rowFont:
		m.edit(func(p *config.Profile) { p.Font = v })
	case rowTime:
		m.edit(func(p *config.Profile) { p.Time = v })
	case rowDate:
		m.edit(func(p *config.Profile) { p.Date = v })
	case rowProfile:
		m.cfg.Profile = v
	case rowLock:
		m.cfg.Tmux.Lock = v
		return tea.Batch(m.options.close(), m.save(before), m.syncTool(m.cfg.Tmux.Conf))
	case rowPIN:
		// After the current PIN (ux.md §2.2): Remove is done on Enter, no
		// confirm; New goes on to the new PIN and its confirmation.
		if v == pinRemove {
			m.cfg.ClearPIN()
			return tea.Batch(m.options.close(), m.save(before), m.toast.show("PIN removed", toastInfo))
		}
		return tea.Batch(m.options.close(), m.askPIN("new PIN", inputPINNew))
	case rowChannel:
		n, err := strconv.Atoi(v)
		if err != nil {
			return m.options.close()
		}
		p, dk, _ := m.subject()
		d := m.draftOf(dk, p)
		hex := &d.BG
		if r.which == 1 {
			hex = &d.FG
		}
		c := [3]int{}
		c[0], c[1], c[2] = config.RGB(*hex)
		c[r.ch] = n
		*hex = config.Hex(c[0], c[1], c[2])
		m.drafts[dk] = d
		return m.options.close()
	default:
		return m.options.close()
	}
	return tea.Batch(m.options.close(), m.save(before))
}

// saveColours is [S]: the draft becomes the file's.
func (m *AppModel) saveColours() tea.Cmd {
	before := m.snapshot()
	p, key, ok := m.subject()
	if !ok {
		return nil
	}
	d := m.draftOf(key, p)
	m.edit(func(p *config.Profile) { p.BG, p.FG = d.BG, d.FG })
	cmd := m.save(before)
	if p, _, _ := m.subject(); p.Colours() == d {
		delete(m.drafts, key)
	}
	return cmd
}

// resetColours is [R]: the draft goes, and the file's colours show again.
func (m *AppModel) resetColours() tea.Cmd {
	if _, key, ok := m.subject(); ok {
		delete(m.drafts, key)
	}
	return nil
}

func (m *AppModel) toggleStatus() tea.Cmd {
	before := m.snapshot()
	m.cfg.ShowStatus = !m.cfg.ShowStatus
	return m.save(before)
}

func (m *AppModel) editNumber() tea.Cmd {
	r := m.rowAt()
	m.editKind = r.kind
	cur := ""
	switch r.kind {
	case rowPINPromptTimeout:
		cur = itoa(m.cfg.PINPromptTimeout)
	case rowWrongPINAttempts:
		cur = itoa(m.cfg.WrongPINAttempts)
	case rowWrongPINCooldown:
		cur = itoa(m.cfg.WrongPINCooldown)
	case rowIdle:
		_, t := m.tool()
		cur = itoa(t.Idle)
	}
	return m.input.ask(inputPopup{title: "number", prompt: r.label + " — empty for the default",
		value: cur, accept: "save", action: inputNumber}, m.layer())
}

// editPath is Enter on a tool's conf: webu's settings box. The current
// value — or the usual file when there is none — is an OFFER, shown dim:
// Tab takes it into the line to edit, Backspace declines it, typing
// starts fresh over it; Enter commits the line as typed, and an offer
// nobody took changes nothing (user, 2026-09-24: webu's way of taking a
// value).
func (m *AppModel) editPath() tea.Cmd {
	name, t := m.tool()
	offer := t.Conf
	if offer == "" {
		offer = usualConf[name]
	}
	return m.input.ask(inputPopup{title: "path", prompt: "config file path — the file " + name + "'s block goes into; Backspace then Enter to unset",
		placeholder: offer, accept: "save", action: inputPath}, m.layer())
}

// editBindKey is Enter on tmux's bind-key: the key as tmux spells it —
// l, C-l, F12 — or nothing, which binds none (user, 2026-09-25).
func (m *AppModel) editBindKey() tea.Cmd {
	return m.input.ask(inputPopup{title: "key", prompt: "bind-key — the key after prefix that locks every client; empty binds none",
		value: m.cfg.Tmux.BindKey, accept: "save", action: inputBindKey}, m.layer())
}

// editCommand is Enter on a custom saver's command: the line as it is,
// to edit; empty is none.
func (m *AppModel) editCommand() tea.Cmd {
	p, _, _ := m.subject()
	return m.input.ask(inputPopup{title: "command", prompt: "command — the program that draws, as sh -c runs it; empty for none",
		value: p.Command, accept: "save", action: inputCommand}, m.layer())
}

// ---- the PIN (ux.md §2.2): one box at a time, one question each.

func (m *AppModel) setPIN() tea.Cmd { return m.askPIN("new PIN", inputPINNew) }

// changePIN is Enter on a PIN that is set: the current one first, then
// the choice — a new PIN, or none (user, 2026-09-24: a PIN must be
// removable, and Remove takes effect on Enter, with no confirm).
func (m *AppModel) changePIN() tea.Cmd { return m.askPIN("current PIN", inputPINCurrent) }

func (m *AppModel) askPIN(title string, action inputAction) tea.Cmd {
	return m.input.ask(inputPopup{title: title, prompt: "PIN", masked: true, accept: "next", action: action}, m.layer())
}

// The two things to do once the current PIN is given.
const (
	pinNew    = "New PIN"
	pinRemove = "Remove PIN"
)

// askPINAction is the choice after the current PIN: the options list,
// on the PIN row, so commitOptions knows whose answer it is.
func (m *AppModel) askPINAction() tea.Cmd {
	m.optionsFor = m.rowAt()
	m.options.setItems([]menuItem{
		{label: pinNew, key: "v:" + pinNew, hint: "then confirm it"},
		{label: pinRemove, key: "v:" + pinRemove, hint: "any key unlocks, from now"},
	}, "PIN", m.layer())
	m.options.rows = 0
	m.options.cursor = 0
	m.options.center()
	return m.options.open()
}

// ---- the input box's answer

// takeName is a name typed for a profile — new, copied or renamed — or
// why it will not do: the box says ` · empty` or ` · taken` and stays.
func (m *AppModel) takeName(v string, self int) (string, bool) {
	name := strings.TrimSpace(v)
	if name == "" {
		m.input.suffix = " · empty"
		return "", false
	}
	if i := m.cfg.Index(name); i >= 0 && i != self {
		m.input.suffix = " · taken"
		return "", false
	}
	return name, true
}

func (m *AppModel) commitInput() tea.Cmd {
	v := m.input.value
	switch m.input.action {
	case inputNew:
		name, ok := m.takeName(v, -1)
		if !ok {
			return nil
		}
		// A profile of the saver, as its defaults say.
		before := m.snapshot()
		m.cfg.Profiles = append(m.cfg.Profiles, m.cfg.NewProfile(name, saver.Kinds[m.editRef]))
		m.cur1 = profileItem(len(m.cfg.Profiles) - 1)
		m.cur2 = 0
		m.focus = panelDetail
		return tea.Batch(m.input.close(), m.save(before))

	case inputRename, inputDuplicate:
		self := -1
		if m.input.action == inputRename {
			self = m.editRef
		}
		name, ok := m.takeName(v, self)
		if !ok {
			return nil
		}
		before := m.snapshot()
		if m.input.action == inputRename {
			old := m.cfg.Profiles[m.editRef].Name
			m.cfg.Profiles[m.editRef].Name = name
			if m.cfg.Profile == old {
				m.cfg.Profile = name
			}
			// The draft follows the name.
			if d, ok := m.drafts[profileKey(old)]; ok {
				delete(m.drafts, profileKey(old))
				m.drafts[profileKey(name)] = d
			}
		} else {
			p := m.cfg.Profiles[m.editRef]
			p.Name = name
			m.cfg.Profiles = append(m.cfg.Profiles, p)
			m.cur1 = profileItem(len(m.cfg.Profiles) - 1)
			m.cur2 = 0
		}
		return tea.Batch(m.input.close(), m.save(before))

	case inputNumber:
		v = strings.TrimSpace(v)
		def := config.Default()
		n := 0
		if v != "" {
			var err error
			if n, err = strconv.Atoi(v); err != nil || n < 0 {
				m.input.suffix = " · invalid"
				return nil
			}
		}
		before := m.snapshot()
		switch m.editKind {
		case rowPINPromptTimeout:
			if v == "" {
				n = def.PINPromptTimeout
			}
			m.cfg.PINPromptTimeout = n
		case rowWrongPINAttempts:
			if v == "" {
				n = def.WrongPINAttempts
			}
			m.cfg.WrongPINAttempts = n
		case rowWrongPINCooldown:
			if v == "" || n == 0 {
				n = def.WrongPINCooldown
			}
			m.cfg.WrongPINCooldown = n
		case rowIdle:
			if v == "" {
				n = config.DefaultIdle
			}
			m.editTool(func(t *config.Tool) { t.Idle = n })
			_, t := m.tool()
			return tea.Batch(m.input.close(), m.save(before), m.syncTool(t.Conf))
		}
		return tea.Batch(m.input.close(), m.save(before))

	case inputPath:
		if v == "" && m.input.placeholder != "" {
			return m.input.close() // the offer was neither taken nor declined
		}
		v = strings.TrimSpace(v)
		if _, ok := config.AbsPath(v); v != "" && !ok {
			m.input.suffix = " · absolute or ~/ path"
			return nil
		}
		before := m.snapshot()
		_, was := m.tool()
		m.editTool(func(t *config.Tool) { t.Conf = v })
		return tea.Batch(m.input.close(), m.save(before), m.syncTool(was.Conf))

	case inputCommand:
		v = strings.TrimSpace(v)
		before := m.snapshot()
		m.edit(func(p *config.Profile) { p.Command = v })
		return tea.Batch(m.input.close(), m.save(before))

	case inputBindKey:
		// One key as tmux names it: a space would make it two words on
		// the line, a # the rest of the line a comment.
		v = strings.TrimSpace(v)
		if strings.ContainsAny(v, " \t#") {
			m.input.suffix = " · one key, e.g. l or C-l"
			return nil
		}
		before := m.snapshot()
		m.cfg.Tmux.BindKey = v
		return tea.Batch(m.input.close(), m.save(before), m.syncTool(m.cfg.Tmux.Conf))

	case inputPINCurrent:
		if !m.cfg.CheckPIN(v) {
			return m.input.freeze(" · wrong")
		}
		return tea.Batch(m.input.close(), m.askPINAction())

	case inputPINNew:
		if err := config.CheckPINLength(v); err != nil {
			m.input.suffix = " · " + err.Error()
			return nil
		}
		m.pinNew = v
		return m.askPIN("confirm PIN", inputPINConfirm)

	case inputPINConfirm:
		if v != m.pinNew {
			m.pinNew = ""
			return tea.Batch(m.toast.show("PIN mismatch", toastError), m.askPIN("new PIN", inputPINNew))
		}
		before := m.snapshot()
		if err := m.cfg.SetPIN(v); err != nil {
			m.input.suffix = " · " + err.Error()
			return nil
		}
		m.pinNew = ""
		return tea.Batch(m.input.close(), m.save(before), m.toast.show("PIN set", toastInfo))
	}
	return m.input.close()
}

// commitConfirm is Enter on a confirm.
func (m *AppModel) commitConfirm() tea.Cmd {
	c := m.confirm
	before := m.snapshot()
	switch c.action {
	case confirmDeleteProfile:
		if c.ref < 0 || c.ref >= len(m.cfg.Profiles) {
			break
		}
		delete(m.drafts, profileKey(m.cfg.Profiles[c.ref].Name))
		m.cfg.Profiles = slices.Delete(m.cfg.Profiles, c.ref, c.ref+1)
		// The cursor stays among the profiles: the next one, or the new last.
		m.cur1 = profileItem(min(c.ref, len(m.cfg.Profiles)-1))
		m.cur2 = 0
	case confirmActivate, confirmDeactivate:
		// The block into, or out of, the tool's file: nothing of
		// config.yaml changes.
		var out bytes.Buffer
		var err error
		if c.action == confirmActivate {
			err = m.install(&out)
		} else {
			_, t := m.tool()
			err = m.uninstall(&out, t.Conf)
		}
		return tea.Batch(m.confirm.close(), m.menu.close(), m.reported(out, err))
	case confirmQuit:
		return tea.Quit
	}
	return tea.Batch(m.confirm.close(), m.menu.close(), m.save(before))
}
