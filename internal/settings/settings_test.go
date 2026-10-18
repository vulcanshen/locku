package settings

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// driver runs the settings screen as Bubble Tea would: a message in, the
// commands it gives back run and their messages fed in again — the
// popups' animations among them — but a timer longer than an animation's
// (a toast's) is left.
type driver struct {
	t    *testing.T
	s    *Settings
	app  *terminu.App
	quit bool
}

func newDriver(t *testing.T, cfg config.Config, w, h int) *driver {
	t.Helper()
	t.Setenv("LOCKU__CONFIG", t.TempDir())
	t.Setenv("PATH", t.TempDir()) // no tmux, no screen: the files only
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	s, err := New(cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	d := &driver{t: t, s: s, app: s.App()}
	d.send(tea.WindowSizeMsg{Width: w, Height: h})
	return d
}

// testConfig is the default config with a second clock and a profile of
// every other saver.
func testConfig() config.Config {
	cfg := config.Default()
	second := config.DefaultProfile()
	second.Name, second.Time, second.Date = "clock2", "HH MM", "off"
	cfg.Profiles = append(cfg.Profiles, second)
	for _, k := range saver.Kinds {
		if k != saver.KindClock {
			cfg.Profiles = append(cfg.Profiles, config.NewProfile(k, k))
		}
	}
	return cfg
}

func (d *driver) send(msg tea.Msg) {
	_, cmd := d.app.Update(msg)
	d.run(cmd)
}

func (d *driver) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		switch m := msg.(type) {
		case nil:
		case tea.BatchMsg:
			for _, c := range m {
				d.run(c)
			}
		case tea.QuitMsg:
			d.quit = true
		default:
			d.send(m)
		}
	case <-time.After(150 * time.Millisecond):
	}
}

// press sends keys by the lib's names: "Enter", "Tab", "Esc", "Space",
// "Backspace", or the characters themselves.
func (d *driver) press(keys ...string) *driver {
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "Enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "Tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		case "Esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "Space":
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		case "Backspace":
			msg = tea.KeyMsg{Type: tea.KeyBackspace}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		d.send(msg)
	}
	return d
}

// typeText types s a character at a time.
func (d *driver) typeText(s string) *driver {
	for _, r := range s {
		d.press(string(r))
	}
	return d
}

// on puts [1]'s cursor on the row id, [2] following it.
func (d *driver) on(id string) *driver {
	d.s.side.Select(id)
	d.s.moved(id)
	return d
}

func (d *driver) screen() string { return ansi.Strip(strings.Join(d.app.Frame(), "\n")) }

// saved is config.yaml as it is on disk now.
func (d *driver) saved() config.Config {
	d.t.Helper()
	cfg, problem := config.Load()
	if problem != "" {
		d.t.Fatalf("config.yaml: %s", problem)
	}
	return cfg
}

// labels are [2]'s rows' labels.
func (d *driver) labels() []string {
	var out []string
	for _, f := range d.s.vals.Fields {
		out = append(out, f.Label)
	}
	return out
}

// toField moves [2]'s cursor onto the field labelled label, option k of
// it for a field drawn in place, focusing [2] first.
func (d *driver) toField(label string, k int) *driver {
	d.t.Helper()
	d.press("2")
	for range 100 {
		if f := d.s.vals.Current(); f != nil && f.Label == label {
			if k == 0 {
				return d
			}
			k--
		}
		d.press("j")
	}
	d.t.Fatalf("no %q in [2]: %v", label, d.labels())
	return d
}

// withIcons draws with icons n cells wide for the rest of the test.
func withIcons(t *testing.T, n int) {
	icon, powerline := terminu.IconWidth(), terminu.PowerlineWidth()
	terminu.SetIconWidth(n, powerline)
	t.Cleanup(func() { terminu.SetIconWidth(icon, powerline) })
}

// The splash draws docs/icon.svg, which go:embed cannot reach: the copy
// here is the same, byte for byte (tdp E6).
func TestIconIsTheDocsIcon(t *testing.T) {
	docs, err := os.ReadFile("../../docs/icon.svg")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(docs, icon) {
		t.Error("internal/settings/icon.svg is not docs/icon.svg: copy it over")
	}
}

// Every row is the terminal's width, at every size, with icons one cell
// and two, with each kind of popup up (tdp L4, E5).
func TestEveryRowIsTheTerminal(t *testing.T) {
	steps := map[string]func(d *driver){
		"the screen":    func(d *driver) {},
		"a Space menu":  func(d *driver) { d.press("Space") },
		"the keys":      func(d *driver) { d.press("?") },
		"a [2]'s keys":  func(d *driver) { d.on(preferenceID).press("2", "?") },
		"a PIN box":     func(d *driver) { d.on(preferenceID).press("2", "Enter") },
		"a colour":      func(d *driver) { d.on(profileID("clock")).toField("bg", 0).press("Enter") },
		"a select":      func(d *driver) { d.on(profileID("runner")).toField("participants", 0).press("Enter") },
		"a confirm":     func(d *driver) { d.on(profileID("clock2")).press("X") },
		"a text box":    func(d *driver) { d.on(profileID("clock2")).press("r") },
		"a number box":  func(d *driver) { d.on(preferenceID).toField("pin_prompt_timeout", 0).press("Enter") },
		"tmux":          func(d *driver) { d.on(toolID("tmux")).press("2") },
		"narrow [2]":    func(d *driver) { d.press("Tab") },
		"global popup":  func(d *driver) { d.press("Space", "G", "Enter") },
		"the quit flow": func(d *driver) { d.press("q") },
	}
	for _, icons := range []int{1, 2} {
		withIcons(t, icons)
		for _, sz := range [][2]int{{100, 30}, {80, 40}, {59, 20}, {40, 12}, {24, 8}} {
			for name, step := range steps {
				d := newDriver(t, testConfig(), sz[0], sz[1])
				step(d)
				frame := d.app.Frame()
				if len(frame) != sz[1] {
					t.Errorf("icons %d, %v, %s: %d rows", icons, sz, name, len(frame))
				}
				for r, l := range frame {
					if w := terminu.Width(l); w != sz[0] {
						t.Errorf("icons %d, %v, %s: row %d is %d wide", icons, sz, name, r, w)
						break
					}
				}
			}
		}
	}
}

// [1] is the profiles, the savers, the tools and preference, in groups,
// the active profile with its dot; the cursor starts on the active profile
// (ui.md §1.1).
func TestSideIsGroupedAndStartsOnTheActive(t *testing.T) {
	cfg := testConfig()
	cfg.Profile = "clock2"
	d := newDriver(t, cfg, 100, 30)
	v := d.screen()
	for _, g := range []string{"PROFILES", "SAVERS", "INTEGRATION", "SETTINGS"} {
		if !strings.Contains(v, g) {
			t.Errorf("no %s group:\n%s", g, v)
		}
	}
	if !strings.Contains(v, "● clock2") || strings.Contains(v, "● clock ") {
		t.Errorf("the dot is on the active profile alone:\n%s", v)
	}
	if d.s.cur.kind != kindProfile || d.s.cfg.Profiles[d.s.cur.ref].Name != "clock2" || !strings.Contains(v, "[2] CLOCK2") {
		t.Errorf("the cursor starts on the active profile: %+v\n%s", d.s.cur, v)
	}
}

// Each saver's [2] is its own settings — on its profiles and on its
// defaults alike — and the colours only where it has them (ui.md §1.1).
func TestEachSaverHasItsRows(t *testing.T) {
	want := map[string][]string{
		saver.KindClock:     {"layout", "size", "font", "time", "date", "bg", "fg"},
		saver.KindRunner:    {"participants", "character", "scene", "background"},
		saver.KindBounce:    {"speed", "time"},
		saver.KindSnake:     {"speed"},
		saver.KindTetromino: {"speed"},
		saver.KindPets:      {"animals", "count", "scene"},
		saver.KindCustom:    {"command"},
	}
	d := newDriver(t, testConfig(), 100, 30)
	for kind, rows := range want {
		d.on(profileID(kind))
		if got := d.labels(); !slices.Equal(got, append([]string{"name", "saver"}, rows...)) {
			t.Errorf("%s's profile: %v", kind, got)
		}
		d.on(saverID(kind))
		if got := d.labels(); !slices.Equal(got, append([]string{"saver", "what", "profiles", "defaults"}, rows...)) {
			t.Errorf("%s's defaults: %v", kind, got)
		}
	}
}

// A radio drawn in place takes effect on Enter and is in the file at once;
// a choice of more than five is a select (components/dialog/form).
func TestChoicesAreWrittenAtOnce(t *testing.T) {
	d := newDriver(t, testConfig(), 100, 30)
	d.on(profileID("clock")).toField("layout", 1).press("Enter")
	if got := d.saved().Profiles[0].Layout; got != "column" {
		t.Errorf("the layout written: %q", got)
	}
	d.on(profileID("runner")).toField("participants", 0).press("Enter")
	if !strings.Contains(d.screen(), "PARTICIPANTS: runner") {
		t.Fatalf("six participants are a select:\n%s", d.screen())
	}
	d.press("j", "Enter")
	i := d.saved().Index("runner")
	if got := d.saved().Profiles[i].Participants; got == config.NewProfile("runner", saver.KindRunner).Participants {
		t.Errorf("the select wrote nothing: %q", got)
	}
	// The saver's defaults change, and no profile already made.
	d.on(saverID(saver.KindClock)).toField("size", 0).press("Enter")
	if got, p := d.saved().Saver(saver.KindClock).Size, d.saved().Profiles[0].Size; got != "small" || p == "small" {
		t.Errorf("the defaults' size %q, the profile's %q", got, p)
	}
}

// Rename follows the active profile; a taken name is refused in the box.
func TestRenameDuplicateNewDelete(t *testing.T) {
	d := newDriver(t, testConfig(), 100, 30)
	d.on(profileID("clock")).press("r", "Tab").typeText("x").press("Enter")
	if cfg := d.saved(); cfg.Index("clockx") != 0 || cfg.Profile != "clockx" {
		t.Errorf("renamed: %v, active %q", cfg.Index("clockx"), cfg.Profile)
	}
	d.press("D").typeText("clock2").press("Enter")
	if !strings.Contains(d.screen(), "Name is taken") {
		t.Errorf("a taken name is refused in the box:\n%s", d.screen())
	}
	d.press("Backspace").typeText("9").press("Enter")
	if d.saved().Index("clock9") < 0 {
		t.Error("the copy was not made")
	}
	if r, _ := d.s.side.Current(); r.ID != profileID("clock9") {
		t.Errorf("the cursor goes to the copy: %q", r.ID)
	}
	d.on(saverID(saver.KindSnake)).press("n").typeText("s2").press("Enter")
	if i := d.saved().Index("s2"); i < 0 || d.saved().Profiles[i].Saver != saver.KindSnake {
		t.Error("the new snake profile was not made")
	}
	d.on(profileID("s2")).press("X")
	if !strings.Contains(d.screen(), "Delete s2?") {
		t.Fatalf("Delete asks first:\n%s", d.screen())
	}
	d.press("Enter")
	if d.saved().Index("s2") >= 0 {
		t.Error("s2 was not deleted")
	}
	// The active one cannot be deleted: there, dimmed, and X does nothing.
	d.on(profileID("clockx")).press("X")
	if strings.Contains(d.screen(), "Delete clockx?") {
		t.Error("the active profile asked to be deleted")
	}
}

// The PIN: set with a confirmation, changed after the current one, and
// removed with Backspace on its grey dots (components/input/password).
func TestPINSetChangeRemove(t *testing.T) {
	d := newDriver(t, testConfig(), 100, 30)
	d.on(preferenceID).toField("PIN", 0).press("Enter").typeText("12").press("Enter")
	if !strings.Contains(d.screen(), "PIN must be 4-12 characters") {
		t.Errorf("a short PIN is refused in the box:\n%s", d.screen())
	}
	d.typeText("34").press("Enter").typeText("1235").press("Enter")
	if !strings.Contains(d.screen(), "not the new PIN") {
		t.Errorf("a different confirmation is refused:\n%s", d.screen())
	}
	d.press("Backspace").typeText("1234").press("Enter")
	if !d.saved().CheckPIN("1234") {
		t.Fatal("the PIN was not set")
	}
	d.press("Enter").typeText("0000").press("Enter")
	if !strings.Contains(d.screen(), "wrong PIN") {
		t.Errorf("a wrong current PIN is refused:\n%s", d.screen())
	}
	d.press("Backspace").typeText("1234").press("Enter", "Enter")
	if !d.saved().CheckPIN("1234") {
		t.Error("Enter on the grey dots must keep the PIN")
	}
	d.press("Enter").typeText("1234").press("Enter", "Backspace", "Enter")
	if d.saved().HasPIN() {
		t.Error("Backspace, then Enter, must leave no PIN")
	}
}

// A number box writes its number; a switch flips in place (preference).
func TestNumbersAndSwitches(t *testing.T) {
	d := newDriver(t, testConfig(), 100, 30)
	d.on(preferenceID).toField("pin_prompt_timeout", 0).press("Enter", "Backspace").typeText("45").press("Enter")
	if got := d.saved().PINPromptTimeout; got != 45 {
		t.Errorf("pin_prompt_timeout %d", got)
	}
	was := d.saved().ShowStatus
	d.toField("show_status", 0).press("Enter")
	if d.saved().ShowStatus == was {
		t.Error("show_status did not flip")
	}
}

// activate writes locku's line into the tool's file after a confirm, and
// the rows after it are written into locku's own (user, 2026-09-25).
func TestActivateTmux(t *testing.T) {
	cfg := testConfig()
	conf := filepath.Join(t.TempDir(), "tmux.conf")
	cfg.Tmux.Conf = conf
	d := newDriver(t, cfg, 100, 30)
	d.on(toolID("tmux")).toField("activate", 0).press("Enter")
	if !strings.Contains(d.screen(), "Write locku's line into") {
		t.Fatalf("activate asks first:\n%s", d.screen())
	}
	// What setup did is told: in a toast, or in a note when it is longer
	// than a toast's three rows (the temporary paths here are long). Esc
	// takes either away.
	d.press("Enter", "Esc")
	b, err := os.ReadFile(conf)
	if err != nil || !strings.Contains(string(b), "source-file") {
		t.Fatalf("the line is not in the file: %q %v", b, err)
	}
	d.toField(toolIdle["tmux"], 0).press("Enter", "Backspace").typeText("60").press("Enter")
	own, _ := os.ReadFile(filepath.Join(config.Dir(), "locku.tmux.conf"))
	if !strings.Contains(string(own), "lock-after-time 60") {
		t.Errorf("a row changed while on is written at once:\n%s", own)
	}
}

// The keys of [2] are its Enter and its panel's, and preference's and a
// tool's then say what each row is (user, 2026-09-25).
func TestDetailKeysAndGlossary(t *testing.T) {
	d := newDriver(t, testConfig(), 100, 40)
	d.on(preferenceID).press("2", "?")
	v := d.screen()
	for _, s := range []string{"Edit — what the lock asks for", "Preview — the lock, showing the active profile", "WHAT EACH ROW IS", "wrong_pin_attempt_cooldown"} {
		if !strings.Contains(v, s) {
			t.Errorf("no %q in preference's keys:\n%s", s, v)
		}
	}
	d.press("Esc").on(profileID("clock")).press("?")
	if strings.Contains(d.screen(), "WHAT EACH ROW IS") {
		t.Error("a profile's keys have no glossary")
	}
}

// A number out of its range is said on the error row, the value named,
// and the box stays (components/input/number).
func TestNumberOutOfRangeIsSaid(t *testing.T) {
	d := newDriver(t, testConfig(), 100, 30)
	d.on(preferenceID).toField("wrong_pin_attempt_cooldown", 0).press("Enter", "Backspace").typeText("0").press("Enter")
	if !strings.Contains(d.screen(), "wrong_pin_attempt_cooldown must be 1 or more") {
		t.Errorf("0 is refused in the box:\n%s", d.screen())
	}
	d.press("Backspace").typeText("99999999999999999999").press("Enter")
	if !strings.Contains(d.screen(), "must be a whole number") {
		t.Errorf("a number past an int is refused in the box:\n%s", d.screen())
	}
	if got := d.saved().WrongPINCooldown; got != testConfig().WrongPINCooldown {
		t.Errorf("a refused number was written: %d", got)
	}
}

// What the Space menu opens — a box, a confirm — opens over it, and Esc
// goes back to it (tdp F4, T1).
func TestMenuStaysUnderWhatItOpens(t *testing.T) {
	d := newDriver(t, testConfig(), 100, 30)
	d.on(profileID("clock2")).press("Space", "r")
	if v := d.screen(); !strings.Contains(v, "RENAME: clock2") || !strings.Contains(v, "Enter:run Esc:close") {
		t.Fatalf("the box over the menu:\n%s", v)
	}
	d.press("Esc")
	if v := d.screen(); strings.Contains(v, "RENAME") || !strings.Contains(v, "Enter:run Esc:close") {
		t.Errorf("Esc goes back to the menu:\n%s", v)
	}
	d.press("Esc", "Space", "X")
	if v := d.screen(); !strings.Contains(v, "Delete clock2?") || !strings.Contains(v, "Enter:run Esc:close") {
		t.Fatalf("the confirm over the menu:\n%s", v)
	}
	d.press("Esc")
	if v := d.screen(); strings.Contains(v, "Delete clock2?") || !strings.Contains(v, "Enter:run Esc:close") {
		t.Errorf("Esc on the confirm goes back to the menu:\n%s", v)
	}
}

// A write that fails opens an error popup with the whole of it, and the
// config is as it was (components/dialog/toast, dialog/note).
func TestWriteFailureIsAnErrorPopup(t *testing.T) {
	d := newDriver(t, testConfig(), 100, 30)
	dir := config.Dir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })
	d.on(profileID("clock2")).press("r", "Tab").typeText("x").press("Enter")
	if v := d.screen(); !strings.Contains(v, "WRITE FAILED") {
		t.Errorf("an error popup:\n%s", v)
	}
	if d.s.cfg.Index("clock2x") >= 0 {
		t.Error("the change stayed after the write failed")
	}
}

// A profile the file names but does not have is a value gone wrong: the
// profile row's label is Red (components/layout/panel).
func TestMissingProfileIsRed(t *testing.T) {
	cfg := testConfig()
	cfg.Profile = "ghost-profile"
	d := newDriver(t, cfg, 100, 30)
	d.on(preferenceID)
	for _, f := range d.s.vals.Fields {
		if f.Label == "profile" && (f.Err == nil || !f.Err()) {
			t.Error("the profile row is not marked wrong")
		}
	}
}
