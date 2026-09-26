package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	overlay "github.com/rmhubbert/bubbletea-overlay"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/custom"
)

// AppModel is the settings screen, the bare `locku` (ui.md §1.1): panel
// [1], the savers, the profiles and preference; panel [2], the detail of
// whatever [1]'s cursor is on. Every change is written to config.yaml the
// moment it is made, except a profile's colours: those move a draft until
// [S] Save.
type AppModel struct {
	cfg     config.Config
	problem string // Load's note: told once, as a toast
	// drafts holds a profile's — or a saver's defaults' — colours as its
	// sliders have them, while they differ from the file's (user,
	// 2026-09-24: a slider that wrote at once could not be put back). [S]
	// Save writes one, [R] Reset drops it, and a rename carries it along.
	drafts map[draftKey]config.Style

	width, height int
	focus         panel
	cur1, top1    int // panel [1]'s cursor and window
	cur2          int // panel [2]'s cursor, into stops()

	menu    spaceMenu
	options spaceMenu
	// helpMenu is the ? menu on a panel (tdp M4): the global operations,
	// run from it, over the key reference.
	helpMenu spaceMenu
	help     helpPopup
	input    inputPopup
	confirm  confirmPopup
	toast    toastModel
	splash   splashModel

	// preview is the lock, running inside this process (ui.md §2.1); nil
	// when the settings are showing.
	preview *LockModel

	pendingG bool // the first half of gg

	// what an open box is about
	editRef    int     // the profile a name box edits, or the saver a new one is of
	editKind   rowKind // the setting a number or path box edits
	optionsFor row     // the row an options list is for
	pinNew     string  // the new PIN, awaiting its confirmation
}

type panel int

const (
	panelSide   panel = 1
	panelDetail panel = 2
)

// NewApp is the settings screen over cfg, its cursor on the active
// profile — what one most often came to change. problem is Load's note.
func NewApp(cfg config.Config, problem string) AppModel {
	return AppModel{
		cfg:      cfg,
		problem:  problem,
		drafts:   map[draftKey]config.Style{},
		focus:    panelSide,
		cur1:     profileItem(max(0, cfg.Index(cfg.Profile))),
		menu:     newSpaceMenu(),
		options:  newOptionsMenu(),
		helpMenu: newHelpMenu(),
		help:     newHelpPopup(),
		input:    newInputPopup(),
		confirm:  newConfirmPopup(),
		toast:    newToast(),
		splash:   newSplashModel(),
	}
}

func (m AppModel) Init() tea.Cmd {
	if m.problem != "" {
		// A file that could not be honoured is news, not a reason to
		// withhold the screen: the defaults are up, nothing is written
		// until the first change (ux.md §6).
		return m.toast.show("config.yaml ignored: "+m.problem, toastError)
	}
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		for _, p := range []interface{ setSize(int, int) }{&m.menu, &m.options, &m.helpMenu, &m.help, &m.input, &m.confirm, &m.toast} {
			p.setSize(msg.Width, msg.Height)
		}
		if m.preview != nil {
			lk, cmd := m.preview.step(msg)
			m.preview = &lk
			return m, cmd
		}
		return m, nil
	}
	// The preview is the lock: it takes every message until it opens.
	if m.preview != nil {
		lk, cmd := m.preview.step(msg)
		if lk.unlocked {
			m.preview = nil
			return m, nil
		}
		m.preview = &lk
		return m, cmd
	}
	switch msg := msg.(type) {
	case AnimTickMsg:
		return m, tea.Batch(
			m.menu.anim.tick(msg), m.options.anim.tick(msg), m.helpMenu.anim.tick(msg), m.help.anim.tick(msg),
			m.input.anim.tick(msg), m.confirm.anim.tick(msg), m.toast.anim.tick(msg))
	case toastExpireMsg:
		return m, m.toast.expire(msg)
	case customPreviewEndMsg:
		// The program's preview is over: a program that ended, or could
		// not run, is the word on the board, a preview as any other.
		if msg.outcome != nil {
			return m, m.startPreviewWord(msg.cfg, *msg.outcome)
		}
		return m, nil
	case inputThawMsg:
		m.input.thaw(msg)
		return m, nil
	case splashTickMsg, splashIdentityMsg, splashHintMsg:
		var cmd tea.Cmd
		m.splash, cmd = m.splash.update(msg)
		return m, cmd
	case tea.KeyMsg:
		return m.key(msg)
	}
	return m, nil
}

// key routes one keystroke: the splash, then Ctrl-C, then Esc (one place,
// tdp K4), then a box being typed in, then q, then whichever float owns
// the keyboard, then the globals, then the panel.
func (m AppModel) key(msg tea.KeyMsg) (AppModel, tea.Cmd) {
	k := msg.String()
	// The splash takes every key, Ctrl-C and q among them: any key only
	// closes it (tdp S3).
	if m.splash.isActive() {
		var cmd tea.Cmd
		m.splash, cmd = m.splash.update(msg)
		return m, cmd
	}
	// Ctrl-C is the way out that q is, typing or not; pressed again while
	// that way out asks about the colours, it leaves at once (tdp K9).
	if k == "ctrl+c" {
		if m.asksToQuit() {
			return m, tea.Quit
		}
		return m, m.quit()
	}
	if k == "esc" {
		return m, m.closeTop()
	}
	// Typing: every printable key is a character (tdp K8).
	if m.input.anim.owns() {
		if !m.input.isInteractive() || m.input.frozen {
			return m, nil
		}
		if k == "enter" {
			return m, m.done(m.commitInput())
		}
		m.input.update(msg)
		return m, nil
	}
	// q leaves from any surface but a box being typed in (tdp K1, K9).
	if k == "q" {
		return m, m.quit()
	}
	// A popup's own help, or a panel's glossary, sits on top of anything.
	if m.help.anim.owns() {
		if k == "?" {
			return m, m.help.close()
		}
		m.help.update(msg)
		return m, nil
	}
	// ? closes the ? menu when that is the top; on anything else it is
	// that surface's help (tdp K6).
	if k == "?" {
		if m.helpMenu.anim.owns() && !m.boxUp() {
			return m, m.helpMenu.close()
		}
		return m, m.openHelp()
	}
	// A box opened from a menu sits on it and takes the keys first (tdp
	// F4). Space opens and closes the Space menu and nothing else: on a
	// confirm or an options list it does nothing (tdp K5).
	if m.confirm.anim.owns() {
		if k == "enter" && m.confirm.isInteractive() {
			return m, m.done(m.commitConfirm())
		}
		return m, nil
	}
	if m.options.anim.owns() {
		var key string
		m.options, key = m.options.update(msg)
		if key != "" {
			return m, m.done(m.commitOptions(key))
		}
		return m, nil
	}
	// The ? menu: the global operations, run from here, over the key
	// reference (tdp M4).
	if m.helpMenu.anim.owns() {
		var key string
		m.helpMenu, key = m.helpMenu.update(msg)
		if key == "" {
			return m, nil
		}
		return m.runRow(true, key)
	}
	if m.menu.anim.owns() {
		if k == " " {
			return m, m.menu.close()
		}
		var key string
		m.menu, key = m.menu.update(msg)
		if key == "" {
			return m, nil
		}
		return m.runRow(false, key)
	}
	// No float: the globals (ux.md §A.2).
	switch k {
	case " ":
		return m, m.openMenu()
	case "P":
		return m, m.previewHere()
	case "V":
		return m, m.splash.show()
	case "tab":
		if m.focus == panelSide {
			m.focus = panelDetail
		} else {
			m.focus = panelSide
		}
		return m, nil
	case "1":
		m.focus = panelSide
		return m, nil
	case "2":
		m.focus = panelDetail
		return m, nil
	}
	// The gg chord, then navigation, then the panel's actions.
	if m.pendingG {
		m.pendingG = false
		if k == "g" {
			k = "gg"
		}
	} else if k == "g" {
		m.pendingG = true
		return m, nil
	}
	if navKeys[k] {
		return m.move(k), nil
	}
	return m.dispatch(k)
}

// move walks the focused panel's cursor: j and k round the ends (user,
// 2026-09-24: a panel loops like a menu does), u and d by half a page,
// gg and G to the ends. Moving [1] resets [2] to its first row: the
// detail has changed under it.
func (m AppModel) move(k string) AppModel {
	page := max(1, m.height-3)
	if m.focus == panelSide {
		was := m.cur1
		m.cur1 = moveCursor(m.cur1, len(m.sideItems()), k, page, true)
		if m.cur1 != was {
			m.cur2 = 0
		}
		return m
	}
	m.cur2 = moveCursor(m.cur2, len(m.stops()), k, page, true)
	return m
}

// openMenu is Space: everything that can be done here, in three regions —
// item operation, panel operation, global operation — a region with
// nothing in it left out, a rule between two, and no titles when only one
// is left (tdp M2).
func (m *AppModel) openMenu() tea.Cmd {
	var item, panel []menuItem
	for _, a := range m.actions() {
		if a.panelOp {
			panel = append(panel, a.row())
		} else {
			item = append(item, a.row())
		}
	}
	var global []menuItem
	for _, a := range m.globalActions() {
		global = append(global, a.row())
	}
	items := regions([]string{"item operation", "panel operation", "global operation"}, item, panel, global)
	title := "[1] locku"
	if m.focus == panelDetail {
		title = m.detailTitle()
	}
	m.menu.setItems(items, title, m.layer())
	return m.menu.open()
}

// startPreview: the whole screen becomes the lock on cfg — the file's
// config with every colour draft in place, and, from [p] on a profile,
// that profile active — and comes back when it opens (ui.md §2.1).
func (m *AppModel) startPreview(cfg config.Config) tea.Cmd {
	lk := newLock(cfg, "", true)
	lk, cmd := lk.step(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	m.preview = &lk
	return cmd
}

// startPreviewWord is a preview of a custom saver's ending: the word on
// the board, the note on the status row.
func (m *AppModel) startPreviewWord(cfg config.Config, o custom.Outcome) tea.Cmd {
	lk := newLock(cfg, "", true).withWord(o.Word, o.Note)
	lk, cmd := lk.step(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	m.preview = &lk
	return cmd
}

// closeTop is Esc: the topmost float goes, and only it. With nothing up it
// does nothing — Esc never leaves the app (ux.md §A.0.K). A float already
// closing is past it, a toast too: the Esc goes to the one under it (tdp F3).
func (m *AppModel) closeTop() tea.Cmd {
	switch {
	case m.toast.anim.owns():
		return m.toast.close()
	case m.help.anim.owns():
		return m.help.close()
	case m.confirm.anim.owns():
		return m.confirm.close()
	case m.input.anim.owns():
		// Any step of a PIN chain cancels the whole chain (ux.md §2.2).
		m.pinNew = ""
		return m.input.close()
	case m.options.anim.owns():
		return m.options.close()
	case m.menu.anim.owns():
		return m.menu.close()
	case m.helpMenu.anim.owns():
		return m.helpMenu.close()
	}
	return nil
}

// popupDepth counts the floats up, for the layer colour of the next.
func (m AppModel) popupDepth() int {
	n := 0
	for _, up := range []bool{m.menu.anim.owns(), m.options.anim.owns(), m.helpMenu.anim.owns(), m.help.anim.owns(), m.input.anim.owns(), m.confirm.anim.owns()} {
		if up {
			n++
		}
	}
	return n
}

func (m AppModel) layer() int { return m.popupDepth() + 1 }

func (m AppModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	if m.preview != nil {
		return m.preview.View()
	}
	if m.splash.isActive() {
		return m.splash.render(m.width, m.height)
	}
	innerH := max(0, m.height-3)
	var panels string
	switch {
	case m.width < narrowW && m.focus == panelSide:
		panels = panelFrame(m.width-2, m.sidebarBody(m.width-2, innerH), sideChips, true)
	case m.width < narrowW:
		panels = panelFrame(m.width-2, m.detailBody(m.width-2, innerH), m.detailChips(), true)
	default:
		w2 := m.width - sideW - 2
		panels = joinHorizontal(
			panelFrame(sideW-2, m.sidebarBody(sideW-2, innerH), sideChips, m.focus == panelSide),
			panelFrame(w2, m.detailBody(w2, innerH), m.detailChips(), m.focus == panelDetail))
	}
	footer := keyLegend([][2]string{{"space", "menu"}, {"?", "help"}, {"tab/1-2", "panels"}, {"q", "quit"}}, m.width)
	out := panels + "\n" + footer

	for _, f := range []struct {
		up   bool
		view func() string
	}{
		{m.menu.isActive(), m.menu.view},
		{m.options.isActive(), m.options.view},
		{m.helpMenu.isActive(), m.helpMenu.view},
		{m.input.isActive(), m.input.view},
		{m.confirm.isActive(), m.confirm.view},
		{m.help.isActive(), m.help.view},
	} {
		if f.up {
			out = overlay.Composite(f.view(), out, overlay.Center, overlay.Center, 0, 0)
		}
	}
	if m.toast.isActive() {
		out = overlay.Composite(m.toast.view(), out, overlay.Center, overlay.Bottom, 0, -2)
	}
	return out
}

// openHelp is ?. On a popup it is that popup's own help: the keys of that
// box, nothing of the app's (tdp K6). On [2] of preference or a tool it is
// the glossary alone, what each row means (user, 2026-09-25: "only the
// preference items"; kept in place of the ? menu there, 2026-09-26 — a
// deviation, dev-remarks). On any other panel it is the ? menu (tdp M4).
func (m *AppModel) openHelp() tea.Cmd {
	switch {
	case m.confirm.anim.owns():
		return m.help.open(m.layer(), confirmHelp(m.confirm.accept))
	case m.options.anim.owns():
		return m.help.open(m.layer(), optionsHelp)
	case m.menu.anim.owns():
		return m.help.open(m.layer(), menuHelp)
	}
	if m.focus == panelDetail {
		switch it := m.sideAt(); it.kind {
		case sidePreference:
			return m.help.open(m.layer(), helpPreference)
		case sideTool:
			return m.help.open(m.layer(), helpTool(tools[it.ref]))
		}
	}
	var global []menuItem
	for _, a := range m.globalActions() {
		global = append(global, a.row())
	}
	var ref []menuItem
	for _, e := range keyReference {
		ref = append(ref, menuItem{label: e.key, hint: e.desc, ref: true})
	}
	// Both regions are always there, so both keep their titles.
	items := append([]menuItem{{label: "global operation", header: true}}, global...)
	items = append(items, menuItem{rule: true}, menuItem{label: "key reference", header: true})
	m.helpMenu.setItems(append(items, ref...), "help", m.layer())
	return m.helpMenu.open()
}

// runRow runs a menu's row — the slow path to the same table the letter
// is the fast path to. A row that opens the next box, a confirm, a name
// or a list, leaves the menu under it, so cancelling the box comes back
// to the menu (tdp F4); any other row is done, and the menu goes — a
// preview among them, which replaces the whole screen (tdp T1).
func (m AppModel) runRow(fromHelp bool, key string) (AppModel, tea.Cmd) {
	m, cmd := m.dispatch(key)
	if m.boxUp() {
		return m, cmd
	}
	if fromHelp {
		return m, tea.Batch(m.helpMenu.close(), cmd)
	}
	return m, tea.Batch(m.menu.close(), cmd)
}

// boxUp: a confirm, an input or an options list owns the keyboard.
func (m AppModel) boxUp() bool {
	return m.confirm.anim.owns() || m.input.anim.owns() || m.options.anim.owns()
}

// done follows a commit: one that opened no next box finished what a menu
// began, so the menus under it go too (tdp T1, D3); one that opened the
// next box of a chain — the PIN's next question — keeps them.
func (m *AppModel) done(cmd tea.Cmd) tea.Cmd {
	if m.boxUp() {
		return cmd
	}
	cmds := []tea.Cmd{cmd}
	for _, menu := range []*spaceMenu{&m.menu, &m.helpMenu} {
		if menu.anim.owns() {
			cmds = append(cmds, menu.close())
		}
	}
	return tea.Batch(cmds...)
}

// asksToQuit: the way out is asking about unsaved colours right now.
func (m AppModel) asksToQuit() bool {
	return m.confirm.anim.owns() && m.confirm.action == confirmQuit
}

// quit is q, Ctrl-C and the global Quit row alike (tdp K9): with a colour
// draft unsaved it asks first, otherwise it leaves. Asked already, it asks
// nothing more: a second Ctrl-C is the way out at once.
func (m *AppModel) quit() tea.Cmd {
	if m.asksToQuit() {
		return nil
	}
	if m.anyDirty() {
		return m.confirm.ask(confirmPopup{title: "Unsaved colours", accept: "quit anyway",
			lines:  []string{"Quit without saving the colours?", "S on the profile saves them, R drops them"},
			action: confirmQuit}, m.layer())
	}
	return tea.Quit
}
