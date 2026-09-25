package ui

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	overlay "github.com/rmhubbert/bubbletea-overlay"

	"github.com/vulcanshen/locku/internal/config"
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
	help    helpPopup
	input   inputPopup
	confirm confirmPopup
	toast   toastModel
	splash  splashModel

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
		cfg:     cfg,
		problem: problem,
		drafts:  map[draftKey]config.Style{},
		focus:   panelSide,
		cur1:    profileItem(max(0, cfg.Index(cfg.Profile))),
		menu:    newSpaceMenu(),
		options: newOptionsMenu(),
		help:    newHelpPopup(),
		input:   newInputPopup(),
		confirm: newConfirmPopup(),
		toast:   newToast(),
		splash:  newSplashModel(),
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
		for _, p := range []interface{ setSize(int, int) }{&m.menu, &m.options, &m.help, &m.input, &m.confirm, &m.toast} {
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
			m.menu.anim.tick(msg), m.options.anim.tick(msg), m.help.anim.tick(msg),
			m.input.anim.tick(msg), m.confirm.anim.tick(msg), m.toast.anim.tick(msg))
	case toastExpireMsg:
		return m, m.toast.expire(msg)
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

// key routes one keystroke: the splash, then Esc (one place, §4.3), then
// whichever float owns the keyboard, then the globals, then the panel.
func (m AppModel) key(msg tea.KeyMsg) (AppModel, tea.Cmd) {
	k := msg.String()
	if k == "ctrl+c" {
		return m, tea.Quit
	}
	if m.splash.isActive() {
		var cmd tea.Cmd
		m.splash, cmd = m.splash.update(msg)
		return m, cmd
	}
	if k == "esc" {
		return m, m.closeTop()
	}
	// Typing: every printable key is a character (§4.5).
	if m.input.anim.owns() {
		if !m.input.isInteractive() || m.input.frozen {
			return m, nil
		}
		if k == "enter" {
			return m, m.commitInput()
		}
		m.input.update(msg)
		return m, nil
	}
	// Help sits on top of anything.
	if m.help.anim.owns() {
		if k == "?" {
			return m, m.help.close()
		}
		m.help.update(msg)
		return m, nil
	}
	if k == "?" {
		return m, m.help.open(m.layer(), m.helpEntries())
	}
	if m.confirm.anim.owns() {
		switch k {
		case "enter":
			if m.confirm.isInteractive() {
				return m, m.commitConfirm()
			}
		case " ":
			return m, m.confirm.close()
		}
		return m, nil
	}
	if m.options.anim.owns() {
		if k == " " {
			return m, m.options.close()
		}
		var key string
		m.options, key = m.options.update(msg)
		if key != "" {
			return m, m.commitOptions(key)
		}
		return m, nil
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
		// The menu is the slow path to the same table the letter is the
		// fast path to; either way the row runs and the menu goes.
		closeCmd := m.menu.close()
		m, cmd := m.dispatch(key)
		return m, tea.Batch(closeCmd, cmd)
	}
	// No float: the globals (ux.md §A.2).
	switch k {
	case "q":
		if m.anyDirty() {
			return m, m.confirm.ask(confirmPopup{title: "Unsaved colours", accept: "quit anyway",
				lines:  []string{"Quit without saving the colours?", "S on the profile saves them, R drops them"},
				action: confirmQuit}, m.layer())
		}
		return m, tea.Quit
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

// openMenu is Space: the actions for the cursor, as rows, in two regions
// when the panel has actions of its own — item operation first, then
// panel operation — and flat when it has one kind (VTP §A.1.1).
func (m *AppModel) openMenu() tea.Cmd {
	var item, panel []menuItem
	for _, a := range m.actions() {
		mi := menuItem{label: a.label, key: a.key, hint: a.hint, disabled: a.disabled}
		if a.panelOp {
			panel = append(panel, mi)
		} else {
			item = append(item, mi)
		}
	}
	var items []menuItem
	switch {
	case len(item) > 0 && len(panel) > 0:
		items = append(items, menuItem{label: "item operation", header: true})
		items = append(items, item...)
		items = append(items, menuItem{label: "panel operation", header: true})
		items = append(items, panel...)
	default:
		items = append(item, panel...)
	}
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

// closeTop is Esc: the topmost float goes, and only it. With nothing up it
// does nothing — Esc never leaves the app (ux.md §A.0.K).
func (m *AppModel) closeTop() tea.Cmd {
	switch {
	case m.toast.isActive():
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
	}
	return nil
}

// popupDepth counts the floats up, for the layer colour of the next.
func (m AppModel) popupDepth() int {
	n := 0
	for _, up := range []bool{m.menu.anim.owns(), m.options.anim.owns(), m.help.anim.owns(), m.input.anim.owns(), m.confirm.anim.owns()} {
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
	hint := foldHome(config.Path())
	var panels string
	switch {
	case m.width < narrowW && m.focus == panelSide:
		panels = panelFrame(m.width-2, m.sidebarBody(m.width-2, innerH), sideChips, "", true)
	case m.width < narrowW:
		panels = panelFrame(m.width-2, m.detailBody(m.width-2, innerH), m.detailChips(), hint, true)
	default:
		w2 := m.width - sideW - 2
		panels = joinHorizontal(
			panelFrame(sideW-2, m.sidebarBody(sideW-2, innerH), sideChips, "", m.focus == panelSide),
			panelFrame(w2, m.detailBody(w2, innerH), m.detailChips(), hint, m.focus == panelDetail))
	}
	footer := keyLegend([][2]string{{"space", "menu"}, {"?", "help"}, {"tab/1-2", "panels"}, {"q", "quit"}}, m.width)
	out := panels + "\n" + footer

	for _, f := range []struct {
		up   bool
		view func() string
	}{
		{m.menu.isActive(), m.menu.view},
		{m.options.isActive(), m.options.view},
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

// foldHome writes the home directory as ~.
func foldHome(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

// helpEntries is what ? shows here: on [2] of a panel with a glossary —
// preference, a tool — that glossary alone, what each row means; anywhere
// else the keys (user, 2026-09-25: "only the preference items").
func (m AppModel) helpEntries() []helpEntry {
	if m.focus == panelDetail {
		switch it := m.sideAt(); it.kind {
		case sidePreference:
			return helpPreference
		case sideTool:
			return helpTool(tools[it.ref])
		}
	}
	return helpKeys
}
