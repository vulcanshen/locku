package ui

import (
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
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
	if m.focus == panelSide {
		// Enter on any row of [1] is the same thing: over to [2], where
		// the row's fields are (user, 2026-09-24). The active saver is set
		// on preference › saver, not here; the dot only shows.
		edit := action{key: "enter", label: "[Enter] Edit", hint: "its rows, in [2]",
			run: func(a *AppModel) tea.Cmd { a.focus = panelDetail; return nil }}
		it := m.sideAt()
		if it.kind != sideSaver {
			return []action{edit}
		}
		del := action{key: "X", label: "Delete", hint: "this saver", run: (*AppModel).deleteSaver}
		switch {
		case len(m.cfg.Savers) == 1:
			del.disabled, del.hint = true, "cannot delete: last one"
		case m.cfg.Savers[it.saver].Name == m.cfg.Saver:
			del.disabled, del.hint = true, "cannot delete: active"
		}
		return []action{
			edit,
			{key: "p", label: "Preview", hint: "the lock, showing this saver", run: (*AppModel).previewSaver},
			{key: "D", label: "Duplicate", hint: "a copy, under a new name", run: (*AppModel).duplicateSaver},
			{key: "r", label: "Rename", hint: "this saver", run: (*AppModel).renameSaver},
			del,
		}
	}
	var out []action
	switch r := m.rowAt(); r.kind {
	case rowName:
		out = append(out, action{key: "enter", label: "[Enter] Rename", hint: "this saver", run: (*AppModel).renameSaver})
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
			out = append(out,
				action{key: "enter", label: "[Enter] Change PIN", hint: "after the current one", run: (*AppModel).changePIN},
				action{key: "x", label: "Clear PIN", hint: "back to: any key unlocks", run: (*AppModel).clearPIN})
		}
	case rowSaver:
		out = append(out, action{key: "enter", label: "[Enter] Choose", hint: "the saver the lock shows", run: (*AppModel).chooseSaver})
	case rowShowStatus:
		out = append(out, action{key: "enter", label: "[Enter] Toggle", hint: "user@host and the time, on the lock", run: (*AppModel).toggleStatus})
	case rowPromptTimeout:
		out = append(out, action{key: "enter", label: "[Enter] Edit", hint: "seconds until the prompt closes; 0 never", run: (*AppModel).editNumber})
	case rowLockoutAfter:
		out = append(out, action{key: "enter", label: "[Enter] Edit", hint: "wrong PINs before a cooldown; 0 off", run: (*AppModel).editNumber})
	case rowLockoutSeconds:
		out = append(out, action{key: "enter", label: "[Enter] Edit", hint: "the cooldown, in seconds", run: (*AppModel).editNumber})
	case rowTmuxConf:
		out = append(out, action{key: "enter", label: "[Enter] Edit", hint: "the file locku setup tmux writes", run: (*AppModel).editPath})
	case rowScreenConf:
		out = append(out, action{key: "enter", label: "[Enter] Edit", hint: "the file locku setup screen writes", run: (*AppModel).editPath})
	}
	if it := m.sideAt(); it.kind == sideSaver {
		save := action{key: "S", label: "Save", hint: "write the colour draft to config.yaml", panelOp: true, run: (*AppModel).saveColours}
		reset := action{key: "R", label: "Reset", hint: "drop the draft: the saved colours again", panelOp: true, run: (*AppModel).resetColours}
		if !m.dirtyOf(m.cfg.Savers[it.saver]) {
			save.disabled, save.hint = true, "nothing to save"
			reset.disabled, reset.hint = true, "nothing changed"
		}
		out = append(out,
			action{key: "P", label: "Preview", hint: "the lock, showing this saver with its draft", panelOp: true, run: (*AppModel).previewSaver},
			save, reset)
	}
	return out
}

// previewHere is the global P: on a saver's [2], that saver; anywhere
// else, the active one (user, 2026-09-24).
func (m *AppModel) previewHere() tea.Cmd {
	if m.focus == panelDetail && m.sideAt().kind == sideSaver {
		return m.previewSaver()
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
// write fails. The savers are cloned: the slice is what a rename edits.
func (m AppModel) snapshot() config.Config {
	before := m.cfg
	before.Savers = slices.Clone(m.cfg.Savers)
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

// previewCfg is the config a preview runs on: the file's, with every
// colour draft in place of the saved colours — a draft is what a preview
// is for.
func (m AppModel) previewCfg() config.Config {
	cfg := m.snapshot()
	for i, s := range cfg.Savers {
		d := m.draftOf(s)
		cfg.Savers[i].BG, cfg.Savers[i].FG = d.BG, d.FG
	}
	return cfg
}

// ---- panel [1]

// previewSaver is [p] on a saver: the lock as it would look with THIS one
// active, whether or not it is (user, 2026-09-24).
func (m *AppModel) previewSaver() tea.Cmd {
	cfg := m.previewCfg()
	cfg.Saver = m.cfg.Savers[m.sideAt().saver].Name
	return m.startPreview(cfg)
}

func (m *AppModel) duplicateSaver() tea.Cmd {
	m.editRef = m.sideAt().saver
	return m.input.ask(inputPopup{title: "name", prompt: "name of the copy",
		value: m.cfg.Savers[m.editRef].Name + "2", accept: "create", action: inputDuplicate}, m.layer())
}

func (m *AppModel) renameSaver() tea.Cmd {
	m.editRef = m.sideAt().saver
	return m.input.ask(inputPopup{title: "name", prompt: "name",
		value: m.cfg.Savers[m.editRef].Name, accept: "rename", action: inputRename}, m.layer())
}

func (m *AppModel) deleteSaver() tea.Cmd {
	ref := m.sideAt().saver
	return m.confirm.ask(confirmPopup{title: "Delete saver", accept: "delete",
		lines:  []string{"Delete " + m.cfg.Savers[ref].Name + "?", "the config is written at once"},
		action: confirmDeleteSaver, ref: ref}, m.layer())
}

// ---- panel [2]

func (m *AppModel) chooseLayout() tea.Cmd {
	s := m.cfg.Savers[m.sideAt().saver]
	return m.openOptions("layout", saver.Layouts, s.Layout, 0)
}

func (m *AppModel) chooseSize() tea.Cmd {
	s := m.cfg.Savers[m.sideAt().saver]
	return m.openOptions("size", saver.Sizes, s.Size, 0)
}

func (m *AppModel) chooseFont() tea.Cmd {
	s := m.cfg.Savers[m.sideAt().saver]
	return m.openOptions("font", saver.Fonts, s.Font, 0)
}

func (m *AppModel) chooseTime() tea.Cmd {
	s := m.cfg.Savers[m.sideAt().saver]
	return m.openOptions("time", saver.TimeFormats, s.Time, 0)
}

func (m *AppModel) chooseDate() tea.Cmd {
	s := m.cfg.Savers[m.sideAt().saver]
	return m.openOptions("date", saver.DateFormats, s.Date, 0)
}

func (m *AppModel) chooseSaver() tea.Cmd {
	names := make([]string, len(m.cfg.Savers))
	for i, s := range m.cfg.Savers {
		names[i] = s.Name
	}
	return m.openOptions("saver", names, m.cfg.Saver, 0)
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
// goes into the saver's draft; everything else goes into the file at once.
func (m *AppModel) commitOptions(key string) tea.Cmd {
	v := strings.TrimPrefix(key, "v:")
	before := m.snapshot()
	switch r := m.optionsFor; r.kind {
	case rowLayout:
		m.cfg.Savers[m.sideAt().saver].Layout = v
	case rowSize:
		m.cfg.Savers[m.sideAt().saver].Size = v
	case rowFont:
		m.cfg.Savers[m.sideAt().saver].Font = v
	case rowTime:
		m.cfg.Savers[m.sideAt().saver].Time = v
	case rowDate:
		m.cfg.Savers[m.sideAt().saver].Date = v
	case rowSaver:
		m.cfg.Saver = v
	case rowChannel:
		n, err := strconv.Atoi(v)
		if err != nil {
			return m.options.close()
		}
		s := m.cfg.Savers[m.sideAt().saver]
		d := m.draftOf(s)
		hex := &d.BG
		if r.which == 1 {
			hex = &d.FG
		}
		c := [3]int{}
		c[0], c[1], c[2] = config.RGB(*hex)
		c[r.ch] = n
		*hex = config.Hex(c[0], c[1], c[2])
		m.drafts[s.Name] = d
		return m.options.close()
	default:
		return m.options.close()
	}
	return tea.Batch(m.options.close(), m.save(before))
}

// saveColours is [S] on a saver: its draft becomes the file's.
func (m *AppModel) saveColours() tea.Cmd {
	before := m.snapshot()
	i := m.sideAt().saver
	d := m.draftOf(m.cfg.Savers[i])
	m.cfg.Savers[i].BG, m.cfg.Savers[i].FG = d.BG, d.FG
	cmd := m.save(before)
	if m.cfg.Savers[i].Colours() == d {
		delete(m.drafts, m.cfg.Savers[i].Name)
	}
	return cmd
}

// resetColours is [R]: the draft goes, and the file's colours show again.
func (m *AppModel) resetColours() tea.Cmd {
	delete(m.drafts, m.cfg.Savers[m.sideAt().saver].Name)
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
	case rowPromptTimeout:
		cur = itoa(m.cfg.PromptTimeout)
	case rowLockoutAfter:
		cur = itoa(m.cfg.LockoutAfter)
	case rowLockoutSeconds:
		cur = itoa(m.cfg.LockoutSeconds)
	}
	return m.input.ask(inputPopup{title: "number", prompt: r.label + " — empty for the default",
		value: cur, accept: "save", action: inputNumber}, m.layer())
}

// editPath is Enter on tmux_conf or screen_conf: webu's settings box.
// The current value — or the usual file when there is none — is an
// OFFER, shown dim: Tab takes it into the line to edit, Backspace
// declines it, typing starts fresh over it; Enter commits the line as
// typed, and an offer nobody took changes nothing (user, 2026-09-24:
// webu's way of taking a value).
func (m *AppModel) editPath() tea.Cmd {
	r := m.rowAt()
	m.editKind = r.kind
	offer, usual := m.cfg.TmuxConf, "~/.tmux.conf"
	if r.kind == rowScreenConf {
		offer, usual = m.cfg.ScreenConf, "~/.screenrc"
	}
	if offer == "" {
		offer = usual
	}
	return m.input.ask(inputPopup{title: "path", prompt: r.label + " — the file locku setup writes; Backspace then Enter to unset",
		placeholder: offer, accept: "save", action: inputPath}, m.layer())
}

// ---- the PIN (ux.md §2.2): one box at a time, one question each.

func (m *AppModel) setPIN() tea.Cmd {
	m.pinAfter = pinSet
	return m.askPIN("new PIN", inputPINNew)
}

func (m *AppModel) changePIN() tea.Cmd {
	m.pinAfter = pinChange
	return m.askPIN("current PIN", inputPINCurrent)
}

func (m *AppModel) clearPIN() tea.Cmd {
	m.pinAfter = pinClear
	return m.askPIN("current PIN", inputPINCurrent)
}

func (m *AppModel) askPIN(title string, action inputAction) tea.Cmd {
	return m.input.ask(inputPopup{title: title, prompt: "PIN", masked: true, accept: "next", action: action}, m.layer())
}

// ---- the input box's answer

func (m *AppModel) commitInput() tea.Cmd {
	v := m.input.value
	switch m.input.action {
	case inputRename, inputDuplicate:
		name := strings.TrimSpace(v)
		if name == "" {
			m.input.suffix = " · empty"
			return nil
		}
		if i := m.cfg.Index(name); i >= 0 && !(m.input.action == inputRename && i == m.editRef) {
			m.input.suffix = " · taken"
			return nil
		}
		before := m.snapshot()
		if m.input.action == inputRename {
			old := m.cfg.Savers[m.editRef].Name
			m.cfg.Savers[m.editRef].Name = name
			if m.cfg.Saver == old {
				m.cfg.Saver = name
			}
			// The draft follows the name.
			if d, ok := m.drafts[old]; ok {
				delete(m.drafts, old)
				m.drafts[name] = d
			}
		} else {
			s := m.cfg.Savers[m.editRef]
			s.Name = name
			m.cfg.Savers = append(m.cfg.Savers, s)
			m.cur1 = len(m.cfg.Savers) - 1
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
		case rowPromptTimeout:
			if v == "" {
				n = def.PromptTimeout
			}
			m.cfg.PromptTimeout = n
		case rowLockoutAfter:
			if v == "" {
				n = def.LockoutAfter
			}
			m.cfg.LockoutAfter = n
		case rowLockoutSeconds:
			if v == "" || n == 0 {
				n = def.LockoutSeconds
			}
			m.cfg.LockoutSeconds = n
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
		if m.editKind == rowScreenConf {
			m.cfg.ScreenConf = v
		} else {
			m.cfg.TmuxConf = v
		}
		return tea.Batch(m.input.close(), m.save(before))

	case inputPINCurrent:
		if !m.cfg.CheckPIN(v) {
			return m.input.freeze(" · wrong")
		}
		if m.pinAfter == pinClear {
			return tea.Batch(m.input.close(), m.confirm.ask(confirmPopup{title: "Clear PIN", accept: "clear",
				lines:  []string{"Clear the PIN?", "the lock will open on any key"},
				action: confirmClearPIN}, m.layer()))
		}
		return m.askPIN("new PIN", inputPINNew)

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
	case confirmDeleteSaver:
		if c.ref < 0 || c.ref >= len(m.cfg.Savers) {
			break
		}
		delete(m.drafts, m.cfg.Savers[c.ref].Name)
		m.cfg.Savers = slices.Delete(m.cfg.Savers, c.ref, c.ref+1)
		// The cursor stays among the savers: the next one, or the new last.
		m.cur1 = min(c.ref, len(m.cfg.Savers)-1)
		m.cur2 = 0
	case confirmClearPIN:
		m.cfg.ClearPIN()
	case confirmQuit:
		return tea.Quit
	}
	return tea.Batch(m.confirm.close(), m.menu.close(), m.save(before))
}
