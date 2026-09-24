package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
)

// newTestApp is the settings screen over a two-saver config, writing to a
// temporary directory, sized 100×30.
func newTestApp(t *testing.T) AppModel {
	t.Helper()
	t.Setenv("LOCKU_CONFIG", t.TempDir())
	cfg := config.Default()
	second := config.DefaultSaver()
	second.Name, second.Time, second.Date = "clock2", "HH:MM:SS", "YYYY-MM-DD"
	cfg.Savers = append(cfg.Savers, second)
	m := NewApp(cfg, "")
	return m.size(100, 30)
}

func (m AppModel) size(w, h int) AppModel {
	mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return mm.(AppModel)
}

// press sends keys, running every animator through between them so a
// float opened by one key is listening for the next.
func (m AppModel) press(keys ...string) AppModel {
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		case "backspace":
			msg = tea.KeyMsg{Type: tea.KeyBackspace}
		case "ctrl+u":
			msg = tea.KeyMsg{Type: tea.KeyCtrlU}
		case " ":
			msg = tea.KeyMsg{Type: tea.KeySpace}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		mm, _ := m.Update(msg)
		m = mm.(AppModel)
		m = m.settle()
	}
	return m
}

// settle runs every popup animation to its end.
func (m AppModel) settle() AppModel {
	for i := 0; i < animFrames+1; i++ {
		for _, t := range []string{"spacemenu", "options", "help", "input", "confirm", "toast"} {
			mm, _ := m.Update(AnimTickMsg{Target: t})
			m = mm.(AppModel)
		}
	}
	return m
}

// expireToast is the toast's own timer firing.
func (m AppModel) expireToast() AppModel {
	mm, _ := m.Update(toastExpireMsg{gen: m.toast.gen})
	return mm.(AppModel).settle()
}

func (m AppModel) typed(s string) AppModel {
	for _, r := range s {
		m = m.press(string(r))
	}
	return m
}

func saved(t *testing.T) config.Config {
	t.Helper()
	cfg, note := config.Load()
	if note != "" {
		t.Fatalf("saved file: %s", note)
	}
	return cfg
}

// The saver detail's stops: name, layout, time, date, then bg R G B and
// fg R G B — ten rows.
const (
	stopName = iota
	stopLayout
	stopTime
	stopDate
	stopBgR
	stopBgG
	stopBgB
	stopFgR
	stopFgG
	stopFgB
)

func TestSidebarEnterOpensTheDetail(t *testing.T) {
	m := newTestApp(t).press("j", "enter")
	if m.focus != panelDetail || m.cfg.Saver != "clock" {
		t.Errorf("Enter on a saver: focus %d, active %q", m.focus, m.cfg.Saver)
	}
	m = m.press("1", "j", "enter") // preference
	if m.focus != panelDetail || m.sideAt().kind != sidePreference {
		t.Error("Enter on preference must go to [2]")
	}
	if !strings.Contains(m.View(), "[2] preference") || strings.Contains(m.View(), "style") {
		t.Errorf("preference, not config or style:\n%s", m.View())
	}
}

func TestPreferenceChoosesTheActiveSaver(t *testing.T) {
	m := newTestApp(t).press("G", "2", "j") // preference › saver
	if m.rowAt().kind != rowSaver {
		t.Fatalf("row %v", m.rowAt().kind)
	}
	m = m.press("enter")
	if !m.options.isInteractive() || m.options.items[m.options.cursor].label != "clock" {
		t.Fatal("the options must open on the active saver")
	}
	m = m.press("j", "enter")
	if m.cfg.Saver != "clock2" || saved(t).Saver != "clock2" {
		t.Errorf("active %q", m.cfg.Saver)
	}
	if !strings.Contains(m.View(), "● clock2") {
		t.Error("the dot did not move")
	}
}

func TestRenameFollowsTheActiveNameAndTheDraft(t *testing.T) {
	m := newTestApp(t).press("r")
	if !m.input.isInteractive() || m.input.value != "clock" || m.input.title != "name" {
		t.Fatalf("rename box: %+v", m.input)
	}
	m = m.press("ctrl+u").typed("clock2").press("enter")
	if m.input.suffix != " · taken" || !m.input.anim.owns() {
		t.Fatalf("a taken name must be refused in the box: %q", m.input.suffix)
	}
	m = m.press("ctrl+u").press("enter")
	if m.input.suffix != " · empty" {
		t.Fatalf("an empty name must be refused: %q", m.input.suffix)
	}
	m = m.typed("main").press("enter")
	if m.cfg.Savers[0].Name != "main" || m.cfg.Saver != "main" {
		t.Errorf("rename: %+v active %q", m.cfg.Savers, m.cfg.Saver)
	}
	if s := saved(t); s.Saver != "main" {
		t.Error("not written")
	}
	// A colour draft follows the rename.
	m = m.press("2").typed(strings.Repeat("j", stopBgR)).press("enter", "G", "enter")
	if !m.dirtyOf(m.cfg.Savers[0]) {
		t.Fatal("no draft")
	}
	m = m.press("1", "r", "ctrl+u").typed("renamed").press("enter")
	if !m.dirtyOf(m.cfg.Savers[0]) || m.drafts["renamed"].BG != "#ff3244" {
		t.Errorf("the draft did not follow the name: %v", m.drafts)
	}
}

func TestDuplicateLandsOnTheCopy(t *testing.T) {
	m := newTestApp(t).press("D")
	if m.input.value != "clock2" {
		t.Fatalf("offer %q", m.input.value)
	}
	m = m.press("enter")
	if m.input.suffix != " · taken" {
		t.Fatalf("clock2 exists: %q", m.input.suffix)
	}
	m = m.press("ctrl+u").typed("third").press("enter")
	if len(m.cfg.Savers) != 3 || m.cfg.Savers[2].Name != "third" || m.cfg.Savers[2].Time != "HH:MM" || m.cur1 != 2 {
		t.Errorf("savers %+v cur1 %d", m.cfg.Savers, m.cur1)
	}
}

func TestDeleteRules(t *testing.T) {
	m := newTestApp(t).press("X")
	if m.confirm.isActive() || !strings.Contains(m.toast.msg, "active") {
		t.Fatal("the active saver must not be deletable")
	}
	m = m.press("esc", "j", "X")
	if !m.confirm.isInteractive() {
		t.Fatal("no confirm for a deletable saver")
	}
	m = m.press("enter")
	if len(m.cfg.Savers) != 1 || len(saved(t).Savers) != 1 {
		t.Errorf("not deleted: %+v", m.cfg.Savers)
	}
	m = m.press("X")
	if !strings.Contains(m.toast.msg, "last") {
		t.Error("the last saver must not be deletable")
	}
}

func TestSidebarPreviewsThatSaver(t *testing.T) {
	m := newTestApp(t).press("j", "p")
	if m.preview == nil {
		t.Fatal("p did not start a preview")
	}
	if m.preview.clock.Time != "HH:MM:SS" {
		t.Errorf("the preview shows %q, not the saver under the cursor", m.preview.clock.Time)
	}
	if m.cfg.Saver != "clock" {
		t.Error("previewing must not change the active saver")
	}
	m = m.press("x") // no PIN: any key unlocks
	if m.preview != nil {
		t.Error("the preview did not hand back")
	}
}

func TestDetailChoosesAndToggles(t *testing.T) {
	m := newTestApp(t).press("2", "j", "j") // [2] on time (name, [type], layout, time)
	if m.rowAt().kind != rowTime {
		t.Fatalf("row %v", m.rowAt().kind)
	}
	m = m.press("enter")
	if !m.options.isInteractive() || m.options.items[m.options.cursor].label != "HH:MM" {
		t.Fatal("options must open on the current value")
	}
	m = m.press("j", "j", "enter")
	if m.cfg.Savers[0].Time != "HH:MM:SS" || saved(t).Savers[0].Time != "HH:MM:SS" {
		t.Errorf("time %q", m.cfg.Savers[0].Time)
	}
	// layout: a column.
	m = m.press("k", "enter")
	if !m.options.isInteractive() || m.options.items[m.options.cursor].label != "row" {
		t.Fatal("layout options must open on row")
	}
	m = m.press("j", "enter")
	if m.cfg.Savers[0].Layout != "column" || saved(t).Savers[0].Layout != "column" {
		t.Errorf("layout %q", m.cfg.Savers[0].Layout)
	}
	// preference › show_status flips in place (PIN, saver, show_status).
	m = m.press("1", "G", "2", "j", "j", "enter")
	if m.cfg.ShowStatus || saved(t).ShowStatus {
		t.Error("show_status did not toggle off")
	}
	// prompt_timeout: a box, refusing nonsense, empty for the default.
	m = m.press("j", "enter")
	if m.input.title != "number" || m.input.value != "30" {
		t.Fatalf("box %+v", m.input)
	}
	m = m.press("ctrl+u").typed("abc").press("enter")
	if m.input.suffix != " · invalid" {
		t.Fatalf("suffix %q", m.input.suffix)
	}
	m = m.press("ctrl+u").typed("12").press("enter")
	if m.cfg.PromptTimeout != 12 || saved(t).PromptTimeout != 12 {
		t.Errorf("prompt_timeout %d", m.cfg.PromptTimeout)
	}
	m = m.press("enter", "ctrl+u", "enter")
	if m.cfg.PromptTimeout != 30 {
		t.Errorf("empty must mean the default, got %d", m.cfg.PromptTimeout)
	}
}

func TestPINSetChangeClear(t *testing.T) {
	m := newTestApp(t).press("G", "2") // preference, PIN row
	if m.rowAt().kind != rowPIN {
		t.Fatal("not on the PIN row")
	}
	m = m.press("enter")
	if m.input.title != "new PIN" || !m.input.masked {
		t.Fatalf("box %+v", m.input)
	}
	m = m.typed("123").press("enter")
	if !strings.Contains(m.input.suffix, "4-64") {
		t.Fatalf("a short PIN must be refused: %q", m.input.suffix)
	}
	m = m.press("ctrl+u").typed("1234").press("enter")
	if m.input.title != "confirm PIN" {
		t.Fatalf("expected the confirm box, got %q", m.input.title)
	}
	m = m.typed("9999").press("enter")
	if m.input.title != "new PIN" || !strings.Contains(m.toast.msg, "mismatch") {
		t.Fatalf("a mismatch goes back to new PIN with a toast: %q %q", m.input.title, m.toast.msg)
	}
	m = m.typed("1234").press("enter").typed("1234").press("enter")
	if !m.cfg.HasPIN() || !saved(t).CheckPIN("1234") || m.toast.msg != "PIN set" {
		t.Fatal("PIN not set")
	}
	// Change: the current one first, and a wrong one freezes the box.
	m = m.press("enter")
	if m.input.title != "current PIN" {
		t.Fatalf("change must ask the current PIN, got %q", m.input.title)
	}
	m = m.typed("0000").press("enter")
	if m.input.suffix != " · wrong" || !m.input.frozen {
		t.Fatalf("wrong current: %+v", m.input)
	}
	m = m.typed("1")
	if m.input.value != "" {
		t.Fatal("frozen box took a key")
	}
	mm, _ := m.Update(inputThawMsg{gen: m.input.frozenGen})
	m = mm.(AppModel)
	m = m.typed("1234").press("enter").typed("5678").press("enter").typed("5678").press("enter")
	if !m.cfg.CheckPIN("5678") {
		t.Fatal("PIN not changed")
	}
	// Clear: current PIN, then a confirm.
	m = m.press("x").typed("5678").press("enter")
	if !m.confirm.isInteractive() || m.confirm.action != confirmClearPIN {
		t.Fatal("clear must confirm")
	}
	m = m.press("enter")
	if m.cfg.HasPIN() || saved(t).HasPIN() {
		t.Error("PIN not cleared")
	}
	// Esc anywhere in a chain cancels all of it. (The toast goes first —
	// Esc takes the topmost thing down, and a toast is a thing — so it is
	// expired here as its timer would have.)
	m = m.expireToast().press("enter").typed("1234").press("enter", "esc")
	if m.input.anim.owns() || m.pinNew != "" || m.cfg.HasPIN() {
		t.Error("Esc did not cancel the chain")
	}
}

func TestColourDraftSaveReset(t *testing.T) {
	m := newTestApp(t).press("2").typed(strings.Repeat("j", stopBgR)) // clock › bg › R
	r := m.rowAt()
	if r.kind != rowChannel || r.which != 0 || r.ch != 0 || r.num != 0x31 {
		t.Fatalf("row %+v", r)
	}
	m = m.press("enter")
	if !m.options.isInteractive() || m.options.rows != 10 || m.options.items[m.options.cursor].label != "49" {
		t.Fatalf("options: rows=%d cursor=%q", m.options.rows, m.options.items[m.options.cursor].label)
	}
	if m.options.top > m.options.cursor || m.options.cursor >= m.options.top+10 {
		t.Errorf("cursor %d not in window at %d", m.options.cursor, m.options.top)
	}
	m = m.press("G", "enter")
	// The draft moved; the file and the config did not.
	clock := m.cfg.Savers[0]
	if m.draftOf(clock).BG != "#ff3244" || clock.BG != config.DefaultBG || saved(t).Savers[0].BG != config.DefaultBG {
		t.Errorf("draft %q cfg %q", m.draftOf(clock).BG, clock.BG)
	}
	if v := m.View(); !m.dirtyOf(clock) || !strings.Contains(v, "· unsaved") ||
		!strings.Contains(v, "#313244") || !strings.Contains(v, "→") || !strings.Contains(v, "#ff3244") {
		t.Errorf("the draft is not shown:\n%s", v)
	}
	// The other saver is untouched, and its own panel says so.
	if m.dirtyOf(m.cfg.Savers[1]) || strings.Contains(m.press("1", "j").View(), "unsaved") {
		t.Error("the draft leaked to another saver")
	}
	// A preview runs on the draft.
	m = m.press("P")
	if m.preview == nil || m.preview.style.BG != "#ff3244" {
		t.Error("the preview must use the draft colours")
	}
	m = m.press("x")
	// Reset drops it.
	m = m.press("1", "k", "2", "R")
	if m.dirtyOf(m.cfg.Savers[0]) || m.draftOf(m.cfg.Savers[0]).BG != config.DefaultBG {
		t.Errorf("reset: draft %q", m.draftOf(m.cfg.Savers[0]).BG)
	}
	m = m.press("R")
	if !strings.Contains(m.toast.msg, "nothing changed") {
		t.Error("R with nothing to reset must say so")
	}
	// Pick again, then Save writes it.
	m = m.press("esc", "enter", "G", "enter", "S")
	if m.dirtyOf(m.cfg.Savers[0]) || m.cfg.Savers[0].BG != "#ff3244" || saved(t).Savers[0].BG != "#ff3244" {
		t.Errorf("save: dirty=%v bg %q", m.dirtyOf(m.cfg.Savers[0]), m.cfg.Savers[0].BG)
	}
	// fg › B is the last stop; the swatch row shows one colour when clean.
	m = m.press("G", "enter", "g", "g", "enter", "S")
	if m.cfg.Savers[0].FG != "#f2b700" {
		t.Errorf("fg %q", m.cfg.Savers[0].FG)
	}
	if strings.Contains(m.View(), "→") || strings.Contains(m.View(), "unsaved") {
		t.Error("a clean saver must show no arrow and no unsaved")
	}
}

func TestQuitAsksWhenColoursUnsaved(t *testing.T) {
	m := newTestApp(t).press("2").typed(strings.Repeat("j", stopFgB)).press("enter", "G", "enter")
	if !m.anyDirty() {
		t.Fatal("not dirty")
	}
	m = m.press("q")
	if !m.confirm.isInteractive() || m.confirm.action != confirmQuit {
		t.Fatal("q with a dirty draft must ask")
	}
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}); !quits(cmd) {
		t.Error("Enter on the confirm must quit")
	}
	m = m.press("esc", "R")
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}); !quits(cmd) {
		t.Error("q with a clean draft must quit at once")
	}
}

// Every letter hotkey the panel answers is a row of the Space menu, and
// no action claims a navigation letter (§4.2, ux.md §4). Where the panel
// has actions of its own the menu has two regions.
func TestMenuCoversEveryHotkey(t *testing.T) {
	m := newTestApp(t)
	positions := 0
	for _, focus := range []panel{panelSide, panelDetail} {
		m.focus = focus
		for c1 := range m.sideItems() {
			m.cur1, m.cur2 = c1, 0
			n := 1
			if focus == panelDetail {
				n = len(m.stops())
			}
			for c2 := 0; c2 < n; c2++ {
				m.cur2 = c2
				positions++
				acts := m.actions()
				if len(acts) == 0 {
					t.Errorf("focus %d cur %d/%d: nothing to do, Space would show an empty menu", focus, c1, c2)
				}
				opened := m.press(" ")
				if !opened.menu.isInteractive() {
					t.Fatalf("focus %d cur %d/%d: Space did not open the menu", focus, c1, c2)
				}
				keys := opened.menu.menuKeys()
				for _, a := range acts {
					if navKeys[a.key] {
						t.Errorf("action %q claims the navigation key %q", a.label, a.key)
					}
					if hotkeyIndex(keys, a.key) < 0 {
						t.Errorf("action %q (%s) is not a menu row", a.label, a.key)
					}
					if len(a.key) == 1 && !strings.Contains(bracketHotkey(a.label, a.key), "["+a.key+"]") {
						t.Errorf("row %q does not show its key %q", a.label, a.key)
					}
				}
				hasPanel := false
				for _, a := range acts {
					hasPanel = hasPanel || a.panelOp
				}
				headers := 0
				for _, it := range opened.menu.items {
					if it.header {
						headers++
					}
				}
				if hasPanel && headers != 2 || !hasPanel && headers != 0 {
					t.Errorf("focus %d cur %d/%d: %d headers with panel ops %v", focus, c1, c2, headers, hasPanel)
				}
				if opened.menu.items[opened.menu.cursor].header {
					t.Errorf("focus %d cur %d/%d: the cursor opened on a header", focus, c1, c2)
				}
			}
		}
	}
	if positions < 10 {
		t.Errorf("only %d positions checked", positions)
	}
}

func TestViewFitsTheTerminal(t *testing.T) {
	check := func(t *testing.T, m AppModel, label string) {
		t.Helper()
		lines := strings.Split(m.View(), "\n")
		if len(lines) != m.height {
			t.Errorf("%s: %d lines for height %d", label, len(lines), m.height)
		}
		for i, l := range lines {
			if w := lipgloss.Width(l); w != m.width {
				t.Errorf("%s: line %d is %d wide, want %d", label, i, w, m.width)
			}
		}
	}
	for _, sz := range [][2]int{{100, 30}, {80, 24}, {59, 20}, {40, 12}} {
		m := newTestApp(t).size(sz[0], sz[1])
		check(t, m, "plain")
		check(t, m.press(" "), "menu")
		check(t, m.press("?"), "help")
		check(t, m.press("r"), "input")
		check(t, m.press("2", "j", "enter"), "options")
		check(t, m.press("2"), "detail focused")
		check(t, m.press("G", "2"), "preference")
		check(t, m.press("2", "G", "enter", "G", "enter"), "saver with a draft")
		check(t, m.press("2", " "), "saver menu with regions")
	}
}

func TestPreviewComesBack(t *testing.T) {
	m := newTestApp(t).press("P")
	if m.preview == nil {
		t.Fatal("P did not start the preview")
	}
	if !strings.Contains(m.View(), "no PIN") {
		t.Error("the preview must draw the lock")
	}
	m = m.press("x") // no PIN: any key unlocks
	if m.preview != nil {
		t.Error("the preview did not hand back")
	}
}

func TestEscClosesOnlyTheTop(t *testing.T) {
	m := newTestApp(t).press("j", "X") // menu-less confirm via the hotkey
	if !m.confirm.isInteractive() {
		t.Fatal("no confirm")
	}
	m = m.press("?")
	if !m.help.isInteractive() {
		t.Fatal("help must open over a confirm")
	}
	m = m.press("esc")
	if m.help.anim.owns() || !m.confirm.anim.owns() {
		t.Error("Esc must close the help and leave the confirm")
	}
	m = m.press("esc")
	if m.confirm.anim.owns() {
		t.Error("Esc must close the confirm")
	}
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc}); cmd != nil {
		t.Error("Esc with nothing up must do nothing")
	}
}

func TestMenuRunsTheRow(t *testing.T) {
	m := newTestApp(t).press("j", " ")
	if !m.menu.isInteractive() {
		t.Fatal("no menu")
	}
	m = m.press("enter") // [Enter] Edit is the first row
	if m.focus != panelDetail || m.menu.anim.owns() {
		t.Errorf("menu commit: focus %d menu up %v", m.focus, m.menu.anim.owns())
	}
	// A hotkey inside the menu runs its row too.
	m = m.press("1", " ", "p")
	if m.preview == nil {
		t.Error("p inside the menu must preview")
	}
}

func TestQuitAndSpaceInsideFloats(t *testing.T) {
	m := newTestApp(t).press(" ")
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}); cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Error("q inside a float must not quit")
		}
	}
	m = m.press(" ")
	if m.menu.anim.owns() {
		t.Error("Space must close the menu it opened")
	}
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}); cmd == nil || !quits(cmd) {
		t.Error("q on the panel must quit")
	}
}
