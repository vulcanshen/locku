package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
)

// newTestApp is the settings screen over a two-profile config, writing to
// a temporary directory, sized 100×30: clock is the default profile —
// large, 3x5, HH MM SS, the full date — and clock2 a plain HH MM with no
// date.
func newTestApp(t *testing.T) AppModel {
	t.Helper()
	t.Setenv("LOCKU_CONFIG", t.TempDir())
	cfg := config.Default()
	second := config.DefaultProfile()
	second.Name, second.Time, second.Date = "clock2", "HH MM", "off"
	cfg.Profiles = append(cfg.Profiles, second)
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

// A clock profile's stops in [2]: name, layout, size, font, time, date,
// then bg R G B and fg R G B — twelve rows; the saver row is read-only.
const (
	stopName = iota
	stopLayout
	stopSize
	stopFont
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
	if m.focus != panelDetail || m.cfg.Profile != "clock" {
		t.Errorf("Enter on a saver: focus %d, active %q", m.focus, m.cfg.Profile)
	}
	m = m.press("1", "G", "enter") // preference, the last row
	if m.focus != panelDetail || m.sideAt().kind != sidePreference {
		t.Error("Enter on preference must go to [2]")
	}
	if !strings.Contains(m.View(), "[2] preference") || strings.Contains(m.View(), "style") {
		t.Errorf("preference, not config or style:\n%s", m.View())
	}
}

func TestPreferenceChoosesTheActiveSaver(t *testing.T) {
	m := newTestApp(t).press("G", "2", "j") // preference › saver
	if m.rowAt().kind != rowProfile {
		t.Fatalf("row %v", m.rowAt().kind)
	}
	m = m.press("enter")
	if !m.options.isInteractive() || m.options.items[m.options.cursor].label != "clock" {
		t.Fatal("the options must open on the active saver")
	}
	m = m.press("j", "enter")
	if m.cfg.Profile != "clock2" || saved(t).Profile != "clock2" {
		t.Errorf("active %q", m.cfg.Profile)
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
	if m.cfg.Profiles[0].Name != "main" || m.cfg.Profile != "main" {
		t.Errorf("rename: %+v active %q", m.cfg.Profiles, m.cfg.Profile)
	}
	if s := saved(t); s.Profile != "main" {
		t.Error("not written")
	}
	// A colour draft follows the rename.
	m = m.press("2").typed(strings.Repeat("j", stopBgR)).press("enter", "G", "enter")
	if !m.dirtyOf(profileKey("main"), m.cfg.Profiles[0]) {
		t.Fatal("no draft")
	}
	m = m.press("1", "r", "ctrl+u").typed("renamed").press("enter")
	if !m.dirtyOf(profileKey("renamed"), m.cfg.Profiles[0]) || m.drafts[profileKey("renamed")].BG != "#ff3244" {
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
	if len(m.cfg.Profiles) != 3 || m.cfg.Profiles[2].Name != "third" || m.cfg.Profiles[2].Time != "HH MM SS" || m.cur1 != profileItem(2) {
		t.Errorf("profiles %+v cur1 %d", m.cfg.Profiles, m.cur1)
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
	if len(m.cfg.Profiles) != 1 || len(saved(t).Profiles) != 1 {
		t.Errorf("not deleted: %+v", m.cfg.Profiles)
	}
	m = m.press("X")
	if !strings.Contains(m.toast.msg, "last") {
		t.Error("the last saver must not be deletable")
	}
}

func TestDetailPreviewsThatSaver(t *testing.T) {
	// P on [2] of clock2 previews clock2 although clock is active; P on
	// preference previews the active one.
	m := newTestApp(t).press("j", "2", "P")
	if m.preview == nil || m.preview.clock.Time != "HH MM" {
		t.Fatal("P on a profile's [2] must preview that profile")
	}
	m = m.press("x", "1", "G", "2", "P")
	if m.preview == nil || m.preview.clock.Time != "HH MM SS" {
		t.Fatal("P on preference must preview the active profile")
	}
	m = m.press("x", "1", "g", "g", "j", "2", " ") // back to clock2, the second profile
	if hotkeyIndex(m.menu.menuKeys(), "P") < 0 {
		t.Error("[P] Preview must be a row of the profile's [2] menu")
	}
}

func TestSidebarPreviewsThatSaver(t *testing.T) {
	m := newTestApp(t).press("j", "p")
	if m.preview == nil {
		t.Fatal("p did not start a preview")
	}
	if m.preview.clock.Time != "HH MM" {
		t.Errorf("the preview shows %q, not the profile under the cursor", m.preview.clock.Time)
	}
	if m.cfg.Profile != "clock" {
		t.Error("previewing must not change the active saver")
	}
	m = m.press("x") // no PIN: any key unlocks
	if m.preview != nil {
		t.Error("the preview did not hand back")
	}
}

func TestDetailChoosesAndToggles(t *testing.T) {
	m := newTestApp(t).press("2", "j", "j", "j", "j") // [2] on time (name, layout, size, font, time; saver is not a stop)
	if m.rowAt().kind != rowTime {
		t.Fatalf("row %v", m.rowAt().kind)
	}
	m = m.press("enter")
	if !m.options.isInteractive() || m.options.items[m.options.cursor].label != "HH MM SS" {
		t.Fatal("options must open on the current value")
	}
	m = m.press("k", "enter")
	if m.cfg.Profiles[0].Time != "HH MM" || saved(t).Profiles[0].Time != "HH MM" {
		t.Errorf("time %q", m.cfg.Profiles[0].Time)
	}
	// font: the tall one.
	m = m.press("k", "enter")
	if !m.options.isInteractive() || m.options.items[m.options.cursor].label != "3x5" {
		t.Fatal("font options must open on 3x5")
	}
	m = m.press("k", "enter")
	if m.cfg.Profiles[0].Font != "3x7" || saved(t).Profiles[0].Font != "3x7" {
		t.Errorf("font %q", m.cfg.Profiles[0].Font)
	}
	// size: medium.
	m = m.press("k", "enter")
	if !m.options.isInteractive() || m.options.items[m.options.cursor].label != "large" {
		t.Fatal("size options must open on large")
	}
	m = m.press("k", "enter")
	if m.cfg.Profiles[0].Size != "medium" || saved(t).Profiles[0].Size != "medium" {
		t.Errorf("size %q", m.cfg.Profiles[0].Size)
	}
	// layout: a column.
	m = m.press("k", "enter")
	if !m.options.isInteractive() || m.options.items[m.options.cursor].label != "row" {
		t.Fatal("layout options must open on row")
	}
	m = m.press("j", "enter")
	if m.cfg.Profiles[0].Layout != "column" || saved(t).Profiles[0].Layout != "column" {
		t.Errorf("layout %q", m.cfg.Profiles[0].Layout)
	}
	// preference › show_status flips in place (PIN, saver, show_status).
	m = m.press("1", "G", "2", "j", "j", "enter")
	if m.cfg.ShowStatus || saved(t).ShowStatus {
		t.Error("show_status did not toggle off")
	}
	// pin_prompt_timeout: a box, refusing nonsense, empty for the default.
	m = m.press("j", "enter")
	if m.input.title != "number" || m.input.value != "30" {
		t.Fatalf("box %+v", m.input)
	}
	m = m.press("ctrl+u").typed("abc").press("enter")
	if m.input.suffix != " · invalid" {
		t.Fatalf("suffix %q", m.input.suffix)
	}
	m = m.press("ctrl+u").typed("12").press("enter")
	if m.cfg.PINPromptTimeout != 12 || saved(t).PINPromptTimeout != 12 {
		t.Errorf("pin_prompt_timeout %d", m.cfg.PINPromptTimeout)
	}
	m = m.press("enter", "ctrl+u", "enter")
	if m.cfg.PINPromptTimeout != 30 {
		t.Errorf("empty must mean the default, got %d", m.cfg.PINPromptTimeout)
	}
}

// Integration holds tmux and screen, each with its file and its idle
// time (user, 2026-09-25). The file is typed webu's way: the box opens
// on an OFFER — the value, or the usual file when nothing is set — that
// Tab takes and Backspace declines; Enter on an untouched offer changes
// nothing; a relative path is refused (user, 2026-09-24).
func TestToolsHaveTheirFileAndIdleTime(t *testing.T) {
	m := newTestApp(t).press("G", "k") // under preference: screen, then tmux
	if it := m.sideAt(); it.kind != sideTool || it.ref != toolScreen {
		t.Fatalf("above preference sits screen, not %+v", it)
	}
	m = m.press("k", "2")
	if it := m.sideAt(); it.kind != sideTool || it.ref != toolTmux || m.rowAt().kind != rowActivate {
		t.Fatalf("tmux, activate row: %+v %+v", it, m.rowAt())
	}
	if v := m.View(); !strings.Contains(v, capLeft+"[2] tmux"+capRight) || strings.Contains(v, "integration") || strings.Contains(v, "config.yaml") || !strings.Contains(v, "activate") || !strings.Contains(v, "off") || !strings.Contains(v, "config file path") || !strings.Contains(v, "not set") || !strings.Contains(v, "────") || !strings.Contains(v, " lock ") || !strings.Contains(v, "lock-server") || !strings.Contains(v, "lock-after-time") || !strings.Contains(v, "bind-key") || !strings.Contains(v, "none") || strings.Contains(v, "tool ") || strings.Contains(v, "status") || strings.Contains(v, "Install") || strings.Contains(v, "installed") {
		t.Errorf("tmux's rows are activate, config file path, a rule, lock, lock-after-time, bind-key:\n%s", v)
	}
	m = m.press("j", "enter")
	if m.input.title != "path" || m.input.placeholder != "~/.tmux.conf" || m.input.value != "" {
		t.Fatalf("box %+v", m.input)
	}
	if !strings.Contains(m.View(), "~/.tmux.conf") || !strings.Contains(m.View(), "Tab") {
		t.Errorf("the offer and the Tab hint must show:\n%s", m.View())
	}
	// Enter on the untouched offer: the box closes, nothing changes.
	m = m.press("enter")
	if m.input.isActive() || m.cfg.Tmux.Conf != "" {
		t.Errorf("an offer nobody took must change nothing: %q", m.cfg.Tmux.Conf)
	}
	// Tab takes it; Enter saves it.
	m = m.press("enter", "tab")
	if m.input.value != "~/.tmux.conf" || m.input.placeholder != "" {
		t.Fatalf("Tab: %+v", m.input)
	}
	m = m.press("enter")
	if m.cfg.Tmux.Conf != "~/.tmux.conf" || saved(t).Tmux.Conf != "~/.tmux.conf" {
		t.Errorf("tmux conf %q", m.cfg.Tmux.Conf)
	}
	// The value is now the offer; typing starts fresh over it, a relative
	// path is refused, an absolute one saved.
	m = m.press("enter")
	if m.input.placeholder != "~/.tmux.conf" {
		t.Fatalf("offer %q", m.input.placeholder)
	}
	m = m.typed("tmux.conf").press("enter")
	if m.input.suffix != " · absolute or ~/ path" {
		t.Fatalf("suffix %q", m.input.suffix)
	}
	m = m.press("ctrl+u").typed("/etc/tmux.conf").press("enter")
	if m.cfg.Tmux.Conf != "/etc/tmux.conf" || saved(t).Tmux.Conf != "/etc/tmux.conf" {
		t.Errorf("tmux conf %q", m.cfg.Tmux.Conf)
	}
	// Backspace declines the offer; Enter on the empty line unsets.
	m = m.press("enter", "backspace")
	if m.input.placeholder != "" || m.input.value != "" {
		t.Fatalf("Backspace: %+v", m.input)
	}
	m = m.press("enter")
	if m.cfg.Tmux.Conf != "" || saved(t).Tmux.Conf != "" {
		t.Errorf("unset: %q", m.cfg.Tmux.Conf)
	}
	// Under the rule, tmux's lock: the server's, or one session's
	// (user, 2026-09-25).
	m = m.press("j", "enter")
	if m.rowAt().kind != rowLock || !m.options.isInteractive() {
		t.Fatalf("lock: %+v %v", m.rowAt(), m.options.isInteractive())
	}
	m = m.press("j", "enter")
	if m.cfg.Tmux.Lock != config.LockSession || saved(t).Tmux.Lock != config.LockSession || !strings.Contains(m.View(), "lock-session") {
		t.Errorf("lock %q:\n%s", m.cfg.Tmux.Lock, m.View())
	}
	// The idle time is the tool's own, under its own name: tmux's
	// lock-after-time changes, screen's idle does not.
	m = m.press("j", "enter")
	if m.input.title != "number" || m.input.value != "300" {
		t.Fatalf("idle box %+v", m.input)
	}
	m = m.press("ctrl+u").typed("45").press("enter")
	if m.cfg.Tmux.LockAfterTime != 45 || saved(t).Tmux.LockAfterTime != 45 || m.cfg.Screen.Idle != 300 {
		t.Errorf("tmux idle %d, screen idle %d", m.cfg.Tmux.LockAfterTime, m.cfg.Screen.Idle)
	}
	// tmux's bind-key: one key as tmux spells it, empty for
	// none; two words are refused (user, 2026-09-25).
	m = m.press("j", "enter")
	if m.rowAt().kind != rowBindKey || m.input.title != "key" || m.input.value != "" {
		t.Fatalf("bind-key box %+v, row %+v", m.input, m.rowAt())
	}
	m = m.typed("C l").press("enter")
	if m.input.suffix != " · one key, e.g. l or C-l" || m.cfg.Tmux.BindKey != "" {
		t.Fatalf("two words: suffix %q, key %q", m.input.suffix, m.cfg.Tmux.BindKey)
	}
	m = m.press("ctrl+u").typed("C-l").press("enter")
	if m.cfg.Tmux.BindKey != "C-l" || saved(t).Tmux.BindKey != "C-l" || !strings.Contains(m.View(), "C-l") {
		t.Errorf("bind-key %q:\n%s", m.cfg.Tmux.BindKey, m.View())
	}
	m = m.press("enter", "ctrl+u", "enter")
	if m.cfg.Tmux.BindKey != "" || saved(t).Tmux.BindKey != "" || !strings.Contains(m.View(), "none") {
		t.Errorf("emptied: %q:\n%s", m.cfg.Tmux.BindKey, m.View())
	}
	// screen, with its own usual file.
	m = m.press("1", "j", "2", "j", "enter")
	if m.input.placeholder != "~/.screenrc" {
		t.Fatalf("offer %q", m.input.placeholder)
	}
	m = m.press("tab", "enter")
	if m.cfg.Screen.Conf != "~/.screenrc" || saved(t).Screen.Conf != "~/.screenrc" {
		t.Errorf("screen conf %q", m.cfg.Screen.Conf)
	}
	// screen's rows are the tmux side's under screen's names (user,
	// 2026-09-25): idle, then bind — the key after C-a, as screen spells
	// it — and no lock to choose, since a screen is a process of its own.
	var kinds []rowKind
	for _, r := range m.rows() {
		if r.stop || r.kind == rowRule {
			kinds = append(kinds, r.kind)
		}
	}
	if want := []rowKind{rowActivate, rowConf, rowRule, rowIdle, rowBindKey}; !slices.Equal(kinds, want) {
		t.Errorf("screen's rows: %v, want %v", kinds, want)
	}
	if v := m.View(); !strings.Contains(v, " bind ") || !strings.Contains(v, " idle ") || strings.Contains(v, "bind-key") || strings.Contains(v, " lock ") {
		t.Errorf("screen's rows are activate, config file path, a rule, idle, bind:\n%s", v)
	}
	m = m.press("j", "j", "enter")
	if m.rowAt().kind != rowBindKey || m.input.title != "key" || m.input.value != "" || !strings.Contains(m.input.prompt, "bind — the key after C-a") || !strings.Contains(m.input.prompt, "C-a x locks anyway") {
		t.Fatalf("bind box %+v, row %+v", m.input, m.rowAt())
	}
	m = m.typed("^ L").press("enter")
	if m.input.suffix != " · one key, e.g. l or ^L" || m.cfg.Screen.Bind != "" {
		t.Fatalf("two words: suffix %q, key %q", m.input.suffix, m.cfg.Screen.Bind)
	}
	m = m.press("ctrl+u").typed("^L").press("enter")
	if m.cfg.Screen.Bind != "^L" || saved(t).Screen.Bind != "^L" || m.cfg.Tmux.BindKey != "" || !strings.Contains(m.View(), "^L") {
		t.Errorf("bind %q, tmux's %q:\n%s", m.cfg.Screen.Bind, m.cfg.Tmux.BindKey, m.View())
	}
	m = m.press("enter", "ctrl+u", "enter")
	if m.cfg.Screen.Bind != "" || saved(t).Screen.Bind != "" || !strings.Contains(m.View(), "none") {
		t.Errorf("emptied: %q:\n%s", m.cfg.Screen.Bind, m.View())
	}
}

// screen's activate is the tmux side's: the block into the screenrc,
// LOCKPRG into the shell rc, and once it is on a row changed — the idle
// time, the key — is written at once; off takes both out (user,
// 2026-09-25). No screen on PATH: the files only.
func TestActivateScreenFromTheScreen(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/sh")            // the shell rc is ~/.profile
	m := newTestApp(t).press("G", "k", "2") // screen, [2]: activate
	rc := filepath.Join(home, ".screenrc")
	profile := filepath.Join(home, ".profile")
	m = m.press("j", "enter", "ctrl+u").typed(rc).press("enter")
	if m.cfg.Screen.Conf != rc || m.rowAt().kind != rowConf {
		t.Fatalf("conf %q:\n%s", m.cfg.Screen.Conf, m.View())
	}
	// Off: the key is config.yaml's alone.
	m = m.press("j", "j", "enter").typed("l").press("enter") // bind l
	if _, err := os.ReadFile(rc); err == nil || saved(t).Screen.Bind != "l" {
		t.Fatalf("while off the file must not exist: %v", err)
	}
	m = m.press("g", "g", "enter")
	if !m.confirm.isInteractive() || m.confirm.action != confirmActivate || !strings.Contains(m.View(), "LOCKPRG") {
		t.Fatalf("activate must confirm, and say where LOCKPRG goes:\n%s", m.View())
	}
	m = m.press("enter")
	b, err := os.ReadFile(rc)
	if err != nil || !strings.Contains(string(b), "# >>> locku >>>") || !strings.Contains(string(b), "idle 300 lockscreen") || !strings.Contains(string(b), "bind l lockscreen") {
		t.Fatalf("after activate: %v\n%s", err, b)
	}
	if p, err := os.ReadFile(profile); err != nil || !strings.Contains(string(p), "export LOCKPRG=") || !strings.Contains(string(p), "# locku") {
		t.Fatalf("the shell rc: %v\n%s", err, p)
	}
	if !strings.Contains(m.toast.msg, "wrote") || m.rowAt().value != "on" {
		t.Errorf("toast %q, row %+v", m.toast.msg, m.rowAt())
	}
	// On: the idle time changed is in the file at once, and so is the
	// key emptied.
	m = m.expireToast().press("j", "j", "enter", "ctrl+u").typed("45").press("enter")
	if b, _ := os.ReadFile(rc); !strings.Contains(string(b), "idle 45 lockscreen") || !strings.Contains(m.toast.msg, "wrote") {
		t.Errorf("idle changed while on, toast %q:\n%s", m.toast.msg, b)
	}
	m = m.expireToast().press("j", "enter", "ctrl+u", "enter")
	if b, _ := os.ReadFile(rc); strings.Contains(string(b), "bind") || !strings.Contains(string(b), "idle 45 lockscreen") {
		t.Errorf("key emptied while on:\n%s", b)
	}
	// Off: both files lose the block.
	m = m.expireToast().press("g", "g", "enter", "enter")
	if b, _ := os.ReadFile(rc); strings.Contains(string(b), "locku") || m.rowAt().value != "off" {
		t.Errorf("after deactivate:\n%s\n%s", b, m.View())
	}
	if p, _ := os.ReadFile(profile); strings.Contains(string(p), "LOCKPRG") {
		t.Errorf("the shell rc after deactivate:\n%s", p)
	}
}

// A tool's activate row is on while locku's block is in the file and
// off while it is not: Enter turns it, after a confirm, and once it is
// on a row changed is written at once, a config file path moved takes
// the block along, and off takes it out (user, 2026-09-25: property and
// value throughout, in place of a button; turn it on once, then what is
// set is what is in).
func TestActivateFromTheScreen(t *testing.T) {
	t.Setenv("PATH", t.TempDir())                // no tmux: the file only
	m := newTestApp(t).press("G", "k", "k", "2") // tmux, [2]: activate
	if hotkeyIndex(m.press("1", " ").menu.menuKeys(), "S") >= 0 || hotkeyIndex(m.press(" ").menu.menuKeys(), "S") >= 0 {
		t.Fatal("S is no longer a key")
	}
	if m.rowAt().kind != rowActivate || m.rowAt().value != "off" {
		t.Fatalf("the first stop is activate, off: %+v", m.rowAt())
	}
	m = m.press("enter")
	if !strings.Contains(m.toast.msg, "config file path first") || m.confirm.isActive() {
		t.Fatalf("activate without a file: %q", m.toast.msg)
	}
	conf := filepath.Join(t.TempDir(), "tmux.conf")
	m = m.expireToast().press("j", "enter", "ctrl+u").typed(conf).press("enter")
	if m.cfg.Tmux.Conf != conf || m.rowAt().kind != rowConf {
		t.Fatalf("conf %q:\n%s", m.cfg.Tmux.Conf, m.View())
	}
	// Off: a row changed is config.yaml's alone.
	m = m.press("j", "j", "j", "enter").typed("l").press("enter") // bind-key l
	if _, err := os.ReadFile(conf); err == nil || saved(t).Tmux.BindKey != "l" {
		t.Fatalf("while off the file must not exist: %v", err)
	}
	// On: a confirm, then the block, and the row says so.
	m = m.press("g", "g", "enter")
	if !m.confirm.isInteractive() || m.confirm.action != confirmActivate || !strings.Contains(m.View(), "Write locku's block into") {
		t.Fatalf("activate must confirm:\n%s", m.View())
	}
	m = m.press("enter")
	b, err := os.ReadFile(conf)
	if err != nil || !strings.Contains(string(b), "# >>> locku >>>") || !strings.Contains(string(b), "lock-after-time 300") || !strings.Contains(string(b), "bind-key l lock-server") {
		t.Fatalf("after activate: %v\n%s", err, b)
	}
	if !strings.Contains(m.toast.msg, "wrote") || m.rowAt().value != "on" || strings.Contains(m.View(), "installed") {
		t.Errorf("toast %q, row %+v:\n%s", m.toast.msg, m.rowAt(), m.View())
	}
	// On: the idle time changed is in the file at once, and so is the
	// key emptied.
	m = m.expireToast().press("j", "j", "j", "enter", "ctrl+u").typed("45").press("enter")
	if b, _ := os.ReadFile(conf); !strings.Contains(string(b), "lock-after-time 45") || !strings.Contains(m.toast.msg, "wrote") {
		t.Errorf("idle changed while on, toast %q:\n%s", m.toast.msg, b)
	}
	// So is the lock chosen: lock-session, and the hook that tells each
	// session its lock.
	m = m.expireToast().press("k", "enter", "j", "enter")
	if b, _ := os.ReadFile(conf); m.cfg.Tmux.Lock != config.LockSession || !strings.Contains(string(b), `"locku=lock-session"`) || !strings.Contains(string(b), "session-created[90]") || !strings.Contains(string(b), "bind-key l lock-session") {
		t.Errorf("lock-session while on:\n%s", b)
	}
	m = m.expireToast().press("j")
	m = m.expireToast().press("j", "enter", "ctrl+u", "enter")
	if b, _ := os.ReadFile(conf); strings.Contains(string(b), "bind-key") {
		t.Errorf("key emptied while on:\n%s", b)
	}
	// The file moved: the block goes with it, out of the old file.
	conf2 := filepath.Join(t.TempDir(), "tmux.conf")
	m = m.expireToast().press("g", "g", "j", "enter", "ctrl+u").typed(conf2).press("enter")
	if b, _ := os.ReadFile(conf); strings.Contains(string(b), "locku") {
		t.Errorf("the old file keeps the block:\n%s", b)
	}
	if b, _ := os.ReadFile(conf2); !strings.Contains(string(b), "lock-after-time 45") || m.press("k").rowAt().value != "on" {
		t.Errorf("the new file:\n%s\n%s", b, m.View())
	}
	// The menu on activate says what Enter does.
	m = m.expireToast().press("k", " ")
	if !strings.Contains(m.View(), "Deactivate") {
		t.Errorf("activate's menu:\n%s", m.View())
	}
	// Off asks, then takes the block out.
	m = m.press("esc", "enter")
	if !m.confirm.isInteractive() || m.confirm.action != confirmDeactivate {
		t.Fatal("deactivate must confirm")
	}
	m = m.press("enter")
	if b, _ := os.ReadFile(conf2); strings.Contains(string(b), "locku") {
		t.Errorf("after deactivate:\n%s", b)
	}
	if !strings.Contains(m.toast.msg, "removed") || m.rowAt().value != "off" {
		t.Errorf("toast %q, row %+v", m.toast.msg, m.rowAt())
	}
	// Off again: a row changed stays out of the file.
	m = m.expireToast().press("j", "j", "j", "enter", "ctrl+u").typed("60").press("enter")
	if b, _ := os.ReadFile(conf2); strings.Contains(string(b), "locku") {
		t.Errorf("a change while off reached the file:\n%s", b)
	}
}

// The title's chips: one grey strip on a panel without the keys; with
// them the border's colour on the name, and unsaved in its own (user,
// 2026-09-25).
func TestChipFill(t *testing.T) {
	name, hot, cold := chip{text: "[2] clock", border: true}, chip{text: "unsaved", fill: yellowColor}, chip{text: "plain"}
	for _, c := range []chip{name, hot, cold} {
		if f := chipFill(c, borderDim, false); f != borderDim {
			t.Errorf("unfocused, %q wears %v", c.text, f)
		}
	}
	if chipFill(name, focusColor, true) != focusColor || chipFill(hot, focusColor, true) != yellowColor || chipFill(cold, focusColor, true) != borderDim {
		t.Errorf("focused: name %v, unsaved %v, plain %v",
			chipFill(name, focusColor, true), chipFill(hot, focusColor, true), chipFill(cold, focusColor, true))
	}
}

// A preview is a look at the saver: any key hands back, PIN or no PIN
// (user, 2026-09-25: the PIN is the lock's business).
func TestPreviewNeedsNoPIN(t *testing.T) {
	m := newTestApp(t)
	if err := m.cfg.SetPIN("1234"); err != nil {
		t.Fatal(err)
	}
	m = m.press("2", "P")
	if m.preview == nil || strings.Contains(m.View(), "any key unlocks") {
		t.Fatalf("P must preview, as the lock would look:\n%s", m.View())
	}
	if m = m.press("x"); m.preview != nil {
		t.Error("a key must hand back without a PIN")
	}
}

// What each preference setting means is in the help, not on the panel
// (user, 2026-09-25).
// ? on [2] of preference is the preference glossary and nothing else —
// what each row means, in place of the note that sat under each row —
// on [2] of a tool that tool's; on [1], and on a profile's [2], the keys
// (user, 2026-09-25). A description longer than its column wraps.
func TestHelpIsThePanelsGlossaryOnItsDetail(t *testing.T) {
	has := func(v string, want ...string) []string {
		var missing []string
		for _, w := range want {
			if !strings.Contains(v, w) {
				missing = append(missing, w)
			}
		}
		return missing
	}
	m := newTestApp(t)
	if v := m.press("?").View() + m.press("?", "G").View(); len(has(v, "Core keys", "Integration", "duplicate")) != 0 || strings.Contains(v, "what each row is") {
		t.Errorf("on a profile: the keys, top and bottom:\n%s", v)
	}
	if v := m.press("G", "?").View(); !strings.Contains(v, "Core keys") || strings.Contains(v, "any key unlocks") { // [1] on preference
		t.Errorf("on [1], preference: still the keys:\n%s", v)
	}
	if v := m.press("G", "2", "?").View(); len(has(v, "[2] preference", "any key unlocks", "wrong_pin_attempt_cooldown")) != 0 || strings.Contains(v, "Core keys") || strings.Contains(v, "idle_lock") {
		t.Errorf("on [2], preference: its glossary only:\n%s", v)
	}
	if pv := m.press("G", "2").View(); strings.Contains(pv, "any key unlocks") {
		t.Errorf("preference's [2] must not carry the notes:\n%s", pv)
	}
	if v := m.press("G", "k", "k", "2", "?").View(); len(has(v, "[2] tmux", "activate", "config file path", "lock-server", "lock-after-time", "bind-key")) != 0 || strings.Contains(v, "idle_lock") || strings.Contains(v, "Core keys") || strings.Contains(v, "any key unlocks") {
		t.Errorf("on [2], tmux: its glossary only:\n%s", v)
	}
	if v := m.press("G", "k", "2", "?").View(); len(has(v, "[2] screen", "idle", "activate", "bind", "C-a x", "LOCKPRG", "shell rc")) != 0 || strings.Contains(v, "bind-key") || strings.Contains(v, "lock-server") {
		t.Errorf("on [2], screen: its glossary only, its key under screen's name, and where LOCKPRG lives:\n%s", v)
	}
	// Narrow: the PIN's line does not fit beside a 26-column key and is
	// not cut — its end goes on under itself.
	v := m.size(60, 30).press("G", "2", "?").View()
	if !strings.Contains(v, "what the lock asks for") || !strings.Contains(v, "unlocks") {
		t.Fatalf("the description is cut:\n%s", v)
	}
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(l, "what the lock asks for") && strings.Contains(l, "unlocks") {
			t.Errorf("the description did not wrap:\n%s", v)
		}
	}
}

// Every [2] is a table under a header row, Property and Value, which
// the cursor skips (user, 2026-09-25).
func TestEveryDetailHasAHeader(t *testing.T) {
	m := newTestApp(t)
	for _, keys := range [][]string{{}, {"G", "k", "k", "k", "k"}, {"G", "k", "k"}, {"G"}} { // a profile, a saver, tmux, preference
		mm := m.press(keys...)
		if r := mm.rows(); r[0].kind != rowHead || r[0].label != "Property" || r[0].value != "Value" || r[0].stop {
			t.Errorf("%v: first row %+v", keys, r[0])
		}
		mm = mm.press("2")
		if v := mm.View(); !strings.Contains(v, "Property ") || !strings.Contains(v, "Value") || strings.Contains(v, "Properties") || mm.rowAt().kind == rowHead {
			t.Errorf("%v: the header, and the cursor not on it:\n%s", keys, v)
		}
	}
}

// wrap breaks at spaces, never past w, and cuts a word longer than w.
func TestWrap(t *testing.T) {
	got := wrap("what the lock asks for; with none, any key unlocks", 26)
	if len(got) != 2 || got[0] != "what the lock asks for;" || got[1] != "with none, any key unlocks" {
		t.Errorf("%q", got)
	}
	for _, l := range wrap("seconds without a key before the PIN box closes; 0 never", 10) {
		if dispW(l) > 10 {
			t.Errorf("%q is wider than 10", l)
		}
	}
	if got := wrap("abcdefghij", 4); len(got) != 3 || got[0] != "abcd" || got[2] != "ij" {
		t.Errorf("a long word: %q", got)
	}
	if wrap("", 10) != nil || wrap("x", 0) != nil {
		t.Error("nothing to wrap")
	}
}

// The sidebar starts on the active profile; under the profiles sit the
// savers — the kinds — whose [2] is what it is, which profiles are of
// it, and its DEFAULTS: the same rows a profile has, which a profile
// made of it from now on starts as, and which [p] previews; changing
// them changes no profile already made (user, 2026-09-24).
func TestSaverDefaultsAreEditedAndPreviewed(t *testing.T) {
	m := newTestApp(t)
	if it := m.sideAt(); it.kind != sideProfile || it.ref != 0 || m.cfg.Profiles[0].Name != "clock" {
		t.Fatalf("the cursor must start on the active profile, not %+v", it)
	}
	m = m.press("G", "k", "k", "k", "k", "k") // under the profiles: the savers, the tools, then preference
	if it := m.sideAt(); it.kind != sideSaver || it.ref != 0 {
		t.Fatalf("the first saver sits under the profiles, not %+v", it)
	}
	v := m.View()
	if !strings.Contains(v, "Savers") || !strings.Contains(v, "Profiles") || !strings.Contains(v, capLeft+"[2] clock"+capRight) ||
		!strings.Contains(v, "clock, clock2") || !strings.Contains(v, "defaults") || strings.Contains(v, "unsaved") {
		t.Errorf("a saver's screen:\n%s", v)
	}
	// The built-in clock defaults, as the user set them.
	d := m.cfg.Saver("clock")
	if d.Layout != "row" || d.Size != "large" || d.Font != "3x5" || d.Time != "HH MM SS" || d.Date != "YYYY-MM-DD" || d.BG != config.DefaultBG || d.FG != config.DefaultFG {
		t.Fatalf("clock defaults %+v", d)
	}
	if got := len(m.stops()); got != 11 { // layout, size, font, time, date, six channels — no name
		t.Errorf("%d stops", got)
	}
	// The size row: large to medium, written to the file's savers, and
	// no profile moves.
	m = m.press("2", "j", "enter")
	if !m.options.isInteractive() || m.options.items[m.options.cursor].label != "large" {
		t.Fatalf("size options: %+v", m.options.items)
	}
	m = m.press("k", "enter")
	if m.cfg.Saver("clock").Size != "medium" || saved(t).Saver("clock").Size != "medium" {
		t.Errorf("default size %q", m.cfg.Saver("clock").Size)
	}
	if m.cfg.Profiles[0].Size != "large" || saved(t).Profiles[0].Size != "large" {
		t.Error("changing a saver's defaults must not touch a profile")
	}
	// A colour draft on the saver: the chip says so, S writes it, and
	// the profiles keep their colours.
	m = m.press("G", "enter", "G", "enter")
	if m.cfg.Saver("clock").FG != config.DefaultFG || !strings.Contains(m.View(), "[2] clock") || !strings.Contains(m.View(), "unsaved") {
		t.Errorf("the saver's draft:\n%s", m.View())
	}
	m = m.press("S")
	if m.cfg.Saver("clock").FG != "#f2b7ff" || saved(t).Saver("clock").FG != "#f2b7ff" || m.cfg.Profiles[0].FG != config.DefaultFG {
		t.Errorf("saved defaults %+v, profile %+v", m.cfg.Saver("clock"), m.cfg.Profiles[0])
	}
	// p previews a profile made of the defaults; n makes one.
	m = m.press("1", "p")
	if m.preview == nil || m.preview.scale != 2 || m.preview.style.FG != "#f2b7ff" || m.preview.clock.Time != "HH MM SS" {
		t.Fatalf("the saver's preview: %+v", m.preview)
	}
	m = m.press("x", "n", "enter")
	if p := m.cfg.Profiles[2]; p.Name != "clock3" || p.Size != "medium" || p.FG != "#f2b7ff" {
		t.Errorf("a new profile must be the defaults: %+v", p)
	}
	// Enter goes to [2] as on every row; the menu there has New and the
	// panel operations, and no Enter row on the description.
	m = m.press("1", "G", "k", "k", "k", "k", "k", "enter", " ")
	if m.focus != panelDetail || hotkeyIndex(m.menu.menuKeys(), "n") < 0 || hotkeyIndex(m.menu.menuKeys(), "S") < 0 {
		t.Errorf("a saver's [2] menu: %v", m.menu.menuKeys())
	}
}

// [n] on a saver makes a profile of it: the name box offers the saver's
// own name while it is free, then a numbered one; the new profile has
// the saver's rows — a dino's size, runner and scene — lands under the
// cursor, and previews as the run.
func TestNewProfileOfASaver(t *testing.T) {
	m := newTestApp(t).press("G", "k", "k", "k", "k", "n") // the dino saver, above the tools
	if !m.input.isInteractive() || m.input.title != "name" || m.input.value != "dino" {
		t.Fatalf("new box: %+v", m.input)
	}
	m = m.press("ctrl+u").typed("clock").press("enter")
	if m.input.suffix != " · taken" {
		t.Fatalf("a taken name must be refused: %q", m.input.suffix)
	}
	m = m.press("ctrl+u").typed("dino").press("enter")
	p := m.cfg.Profiles[2]
	if len(m.cfg.Profiles) != 3 || p.Name != "dino" || p.Saver != "dino" || p.Runner != "big" || p.Scene != "grassland" ||
		len(saved(t).Profiles) != 3 || saved(t).Profiles[2].Saver != "dino" {
		t.Fatalf("profiles %+v", m.cfg.Profiles)
	}
	if m.cur1 != profileItem(2) || m.focus != panelDetail || m.sideAt().kind != sideProfile {
		t.Errorf("the cursor must land on the new profile's [2]: cur1 %d focus %d", m.cur1, m.focus)
	}
	v := m.View()
	if strings.Contains(v, "layout") || strings.Contains(v, "HH MM") || !strings.Contains(v, "runner") || !strings.Contains(v, "grassland") {
		t.Errorf("a dino's rows:\n%s", v)
	}
	if got := len(m.stops()); got != 9 { // name, runner, scene, six channels — no size
		t.Errorf("%d stops", got)
	}
	if p.Size != "" || p.Layout != "" || strings.Contains(v, "size") {
		t.Errorf("a dino has no size or shapes: %+v", p)
	}
	m = m.press("P")
	if m.preview == nil || m.preview.game == nil {
		t.Fatal("the preview must run the dino")
	}
	m = m.press("x") // no PIN: any key hands back
	// A second one of the same saver is offered the next free name, and
	// the saver's [2] lists both.
	m = m.press("1", "G", "k", "k", "k", "k", "n")
	if m.input.value != "dino2" {
		t.Errorf("offer %q", m.input.value)
	}
	m = m.press("esc")
	if rows := m.rows(); rows[3].label != "profiles" || rows[3].value != "dino" {
		t.Errorf("the saver's profiles row: %+v", rows[3])
	}
}

// The panels loop like the menus do (user, 2026-09-24): k on the first
// row is the last, j on the last is the first; u and d still stop.
func TestPanelsLoop(t *testing.T) {
	m := newTestApp(t).press("g", "g", "k")
	if m.sideAt().kind != sidePreference {
		t.Errorf("k on the first row must reach the last, not %+v", m.sideAt())
	}
	m = m.press("j")
	if it := m.sideAt(); it.kind != sideProfile || it.ref != 0 {
		t.Errorf("j on the last row must reach the first, not %+v", it)
	}
	m = m.press("d", "d", "d")
	if m.sideAt().kind != sidePreference {
		t.Error("d stops at the end")
	}
	m = m.press("2", "k")
	if m.rowAt().kind != rowWrongPINCooldown {
		t.Errorf("[2] loops too: k on the first row is %v", m.rowAt().kind)
	}
}

// [2] follows the cursor when its rows outgrow the panel.
func TestDetailScrollsToTheCursor(t *testing.T) {
	m := newTestApp(t).size(60, 12).press("2", "G")
	if v := m.View(); !strings.Contains(v, "  B ") || strings.Contains(v, "name ") {
		t.Errorf("the last row must be on screen, the first rows off it:\n%s", v)
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
	// The right current PIN opens the choice: a new PIN, or none.
	m = m.typed("1234").press("enter")
	if !m.options.isInteractive() || len(m.options.items) != 2 || m.options.items[0].label != "New PIN" || m.options.items[1].label != "Remove PIN" {
		t.Fatalf("after the current PIN: %+v", m.options.items)
	}
	m = m.press("enter").typed("5678").press("enter").typed("5678").press("enter")
	if !m.cfg.CheckPIN("5678") {
		t.Fatal("PIN not changed")
	}
	// Remove: current PIN, the choice, Enter — done, no confirm.
	m = m.expireToast().press("enter").typed("5678").press("enter", "j", "enter")
	if m.confirm.isActive() || m.cfg.HasPIN() || saved(t).HasPIN() || m.toast.msg != "PIN removed" {
		t.Errorf("PIN not removed: hasPIN=%v toast=%q", m.cfg.HasPIN(), m.toast.msg)
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
	clock := m.cfg.Profiles[0]
	key := profileKey("clock")
	if m.draftOf(key, clock).BG != "#ff3244" || clock.BG != config.DefaultBG || saved(t).Profiles[0].BG != config.DefaultBG {
		t.Errorf("draft %q cfg %q", m.draftOf(key, clock).BG, clock.BG)
	}
	if v := m.View(); !m.dirtyOf(key, clock) || !strings.Contains(v, " unsaved"+capRight) ||
		!strings.Contains(v, "#313244") || !strings.Contains(v, "→") || !strings.Contains(v, "#ff3244") {
		t.Errorf("the draft is not shown:\n%s", v)
	}
	// The other saver is untouched, and its own panel says so.
	if m.dirtyOf(profileKey("clock2"), m.cfg.Profiles[1]) || strings.Contains(m.press("1", "j").View(), "unsaved") {
		t.Error("the draft leaked to another saver")
	}
	// A preview runs on the draft.
	m = m.press("P")
	if m.preview == nil || m.preview.style.BG != "#ff3244" {
		t.Error("the preview must use the draft colours")
	}
	m = m.press("x")
	// Reset drops it.
	m = m.press("1", "2", "R")
	if m.dirtyOf(key, m.cfg.Profiles[0]) || m.draftOf(key, m.cfg.Profiles[0]).BG != config.DefaultBG {
		t.Errorf("reset: draft %q", m.draftOf(key, m.cfg.Profiles[0]).BG)
	}
	m = m.press("R")
	if !strings.Contains(m.toast.msg, "nothing changed") {
		t.Error("R with nothing to reset must say so")
	}
	// Pick again, then Save writes it.
	m = m.press("esc", "enter", "G", "enter", "S")
	if m.dirtyOf(key, m.cfg.Profiles[0]) || m.cfg.Profiles[0].BG != "#ff3244" || saved(t).Profiles[0].BG != "#ff3244" {
		t.Errorf("save: dirty=%v bg %q", m.dirtyOf(key, m.cfg.Profiles[0]), m.cfg.Profiles[0].BG)
	}
	// fg › B is the last stop; the swatch row shows one colour when clean.
	m = m.press("G", "enter", "g", "g", "enter", "S")
	if m.cfg.Profiles[0].FG != "#f2b700" {
		t.Errorf("fg %q", m.cfg.Profiles[0].FG)
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
		check(t, m.press("2", "G", "enter", "G", "enter"), "profile with a draft")
		check(t, m.press("2", " "), "profile menu with regions")
		check(t, m.press("G", "k", "k", "k", "2"), "a saver detail")
		check(t, m.press("G", "k", "2"), "a tool detail")
	}
}

func TestPreviewComesBack(t *testing.T) {
	m := newTestApp(t).press("2", "P") // on [2]: P does nothing on [1]
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
