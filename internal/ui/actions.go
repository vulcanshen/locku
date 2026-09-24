package ui

import (
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// action is one thing the user can do to what the cursor is on. The Space
// menu lists actions and the letter hotkeys dispatch them, from ONE table,
// so a letter hotkey that is not a menu row cannot exist (§4.2) — and a
// row that is disabled still answers, with why.
type action struct {
	key      string // "enter" for the core-key action, or one letter
	label    string
	hint     string
	disabled bool
	run      func(*AppModel) tea.Cmd
}

// actions is the table for the current focus and cursor (ux.md §A.1).
func (m AppModel) actions() []action {
	if m.focus == panelSide {
		it := m.sideAt()
		if it.kind != sideSaver {
			return []action{{key: "enter", label: "[Enter] Edit", hint: "its rows, in [2]",
				run: func(a *AppModel) tea.Cmd { a.focus = panelDetail; return nil }}}
		}
		s := m.cfg.Savers[it.saver]
		active := s.Name == m.cfg.Saver
		del := action{key: "x", label: "Delete", hint: "this saver", run: (*AppModel).deleteSaver}
		switch {
		case len(m.cfg.Savers) == 1:
			del.disabled, del.hint = true, "cannot delete: last one"
		case active:
			del.disabled, del.hint = true, "cannot delete: active"
		}
		set := action{key: "enter", label: "[Enter] Set active", hint: "the lock shows this one", run: (*AppModel).setActive}
		if active {
			set.disabled, set.hint = true, "it is the active one"
		}
		return []action{
			set,
			{key: "c", label: "Duplicate", hint: "a copy, under a new name", run: (*AppModel).duplicateSaver},
			{key: "r", label: "Rename", hint: "this saver", run: (*AppModel).renameSaver},
			del,
		}
	}
	r := m.rowAt()
	switch r.kind {
	case rowName:
		return []action{{key: "enter", label: "[Enter] Rename", hint: "this saver", run: (*AppModel).renameSaver}}
	case rowTime:
		return []action{{key: "enter", label: "[Enter] Choose", hint: "one of four shapes", run: (*AppModel).chooseTime}}
	case rowDate:
		return []action{{key: "enter", label: "[Enter] Choose", hint: "off, or one of four shapes", run: (*AppModel).chooseDate}}
	case rowPIN:
		if !m.cfg.HasPIN() {
			return []action{{key: "enter", label: "[Enter] Set PIN", hint: "the lock will ask for it", run: (*AppModel).setPIN}}
		}
		return []action{
			{key: "enter", label: "[Enter] Change PIN", hint: "after the current one", run: (*AppModel).changePIN},
			{key: "x", label: "Clear PIN", hint: "back to: any key unlocks", run: (*AppModel).clearPIN},
		}
	case rowShowStatus:
		return []action{{key: "enter", label: "[Enter] Toggle", hint: "user@host and the time, on the lock", run: (*AppModel).toggleStatus}}
	case rowPromptTimeout:
		return []action{{key: "enter", label: "[Enter] Edit", hint: "seconds until the prompt closes; 0 never", run: (*AppModel).editNumber}}
	case rowLockoutAfter:
		return []action{{key: "enter", label: "[Enter] Edit", hint: "wrong PINs before a cooldown; 0 off", run: (*AppModel).editNumber}}
	case rowLockoutSeconds:
		return []action{{key: "enter", label: "[Enter] Edit", hint: "the cooldown, in seconds", run: (*AppModel).editNumber}}
	case rowChannel:
		return []action{{key: "enter", label: "[Enter] Pick", hint: "0 to 255", run: (*AppModel).pickChannel}}
	}
	return nil
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

// save writes config.yaml now — every change is saved the moment it is
// made (ui.md §1.1) — and, when the write fails, puts the config back as
// it was and says so.
func (m *AppModel) save(before config.Config) tea.Cmd {
	if err := config.Save(m.cfg); err != nil {
		m.cfg = before
		m.cur1 = clamp(m.cur1, 0, len(m.sideItems())-1)
		return m.toast.show("write failed: "+err.Error(), toastError)
	}
	return nil
}

// ---- panel [1]

func (m *AppModel) setActive() tea.Cmd {
	before := m.snapshot()
	m.cfg.Saver = m.cfg.Savers[m.sideAt().saver].Name
	return m.save(before)
}

func (m *AppModel) duplicateSaver() tea.Cmd {
	m.editRef = m.sideAt().saver
	return m.input.ask(inputPopup{title: "name", prompt: "name of the copy",
		value: m.cfg.Savers[m.editRef].Name + "2", accept: "create", action: inputDuplicate}, m.layer())
}

func (m *AppModel) renameSaver() tea.Cmd {
	if m.focus == panelSide {
		m.editRef = m.sideAt().saver
	} else {
		m.editRef = m.sideAt().saver
	}
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

func (m *AppModel) chooseTime() tea.Cmd {
	s := m.cfg.Savers[m.sideAt().saver]
	return m.openOptions("time", saver.TimeFormats, s.Time, 0)
}

func (m *AppModel) chooseDate() tea.Cmd {
	s := m.cfg.Savers[m.sideAt().saver]
	return m.openOptions("date", saver.DateFormats, s.Date, 0)
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

// commitOptions is a value chosen from the options list.
func (m *AppModel) commitOptions(key string) tea.Cmd {
	v := strings.TrimPrefix(key, "v:")
	before := m.snapshot()
	switch r := m.optionsFor; r.kind {
	case rowTime:
		m.cfg.Savers[m.sideAt().saver].Time = v
	case rowDate:
		m.cfg.Savers[m.sideAt().saver].Date = v
	case rowChannel:
		n, err := strconv.Atoi(v)
		if err != nil {
			return m.options.close()
		}
		hex := &m.cfg.Style.BG
		if r.which == 1 {
			hex = &m.cfg.Style.FG
		}
		c := [3]int{}
		c[0], c[1], c[2] = config.RGB(*hex)
		c[r.ch] = n
		*hex = config.Hex(c[0], c[1], c[2])
	}
	return tea.Batch(m.options.close(), m.save(before))
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
		m.cfg.Savers = slices.Delete(m.cfg.Savers, c.ref, c.ref+1)
		// The cursor stays among the savers: the next one, or the new last.
		m.cur1 = min(c.ref, len(m.cfg.Savers)-1)
		m.cur2 = 0
	case confirmClearPIN:
		m.cfg.ClearPIN()
	}
	return tea.Batch(m.confirm.close(), m.menu.close(), m.save(before))
}
