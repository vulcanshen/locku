package ui

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// LockModel is `locku lock` (function.md §3): the board, the status row and
// the PIN box. The process IS the lock — while it runs the terminal is
// locked — and there are exactly three ways out: the right PIN, any key
// when no PIN is set, and the terminal going away (§1.2). Nothing here
// returns tea.Quit for any other reason.
//
// It runs in the lib's App, which draws the board as its canvas, whole, and
// the PIN box over it — the lib's password box in its unlock look, the
// board dimmed under it (tdp E8; LK1, 2026-10-19: until then the box was
// drawn here, the lib's way). The lock is not a surface (tdp K1): no key of
// the lib's acts on it, and what the lock adds to the box — a second of
// "wrong" that swallows every key, the timeout, the PIN read at every key —
// is done here, around the App.
type LockModel struct {
	cfg     config.Config
	problem string
	clock   saver.Clock
	game    saver.Game // the dino run or another game, when that is the profile's saver; the clock is idle then
	word    saver.Word // a custom saver's ending on the board; the clock is idle then
	note    string     // the custom saver's ending, in red on the status row
	// The accent (2026-09-25): the colour the code after EXIT wears, and
	// the rune it starts at — green for 0, red for the rest, while EXIT
	// keeps the gold; 0 accents nothing.
	accent     lipgloss.Color
	accentFrom int
	style      config.Style // the active profile's colours
	noPIN      bool

	width, height int
	layout        layout   // what the board spells now, block by block
	plain         []string // when nothing fits: the lines drawn as text
	scale         int      // the scale the saver asks for: 1, 2 or 3
	face          face     // the font the saver asks for
	shown         board    // the board on screen
	rev           *reveal  // the change in progress, if one is
	tickGen       int      // a clock or reveal tick from before a change carries an older gen

	app     *terminu.App
	boxUp   bool      // the PIN box is open, or opening
	idleGen int       // a timeout from before a key carries an older gen
	held    time.Time // the second of "wrong": every key swallowed till then
	wrong   bool      // the Enter just handled found the PIN wrong
	// failures is the wrong PINs in a row, process-lifetime (§4.4), and
	// lockoutUntil the end of the cooling-off they led to.
	failures     int
	lockoutUntil time.Time

	lockedAt   time.Time
	user, host string

	// preview: this lock runs inside the settings screen (ui.md §2.1), so
	// an unlock hands control back rather than ending the process.
	preview  bool
	unlocked bool
	// gone: the terminal went away under the lock; the process ends, but
	// nothing was unlocked — a tmux session stays marked locked.
	gone bool
	// promptOnly: the PIN box alone, for the custom saver (function.md
	// §5.5) — up from the first frame, and when it closes the program is
	// put back (back), not a board.
	promptOnly bool
	back       bool
	over       bool // ended: unlocked, gone, or back — nothing more to paint
	initCmd    tea.Cmd
	// paint draws the box for a prompt-only lock that has no renderer of
	// its own: the custom saver's, where the box goes over the program's
	// picture and nothing else may be drawn (user, 2026-09-25). It is given
	// the box's width too: the program's side cannot tell how wide an icon
	// is (tdp D6).
	paint func(box string, w int)

	now func() time.Time
}

// Messages the lock sends itself. A tick carries the generation it was
// scheduled under, so one from before a state change is ignored when it
// fires.
type (
	clockTickMsg   struct{ gen int }
	revealTickMsg  struct{ gen int }
	boxIdleMsg     struct{ gen int } // pin_prompt_timeout ran out
	heldOverMsg    struct{}          // the second of "wrong" is over
	lockoutOverMsg struct{}          // the cooling-off is over
)

// TTYGoneMsg says the terminal went away under the lock: read returned
// EOF or EIO. There is nothing left to protect, and the process ends.
type TTYGoneMsg struct{}

// The box: its title says what to type (tdp S36), and it says wrong for a
// wrong PIN, which is a second of every key swallowed, the dots cleared
// (function.md §4.4).
const (
	pinTitle  = "Unlock"
	wrongHold = time.Second
)

var errWrongPIN = errors.New("wrong PIN")

// NewLock is the model for `locku lock`.
func NewLock(cfg config.Config, problem string) *LockModel { return newLock(cfg, problem, false) }

// NewLockWord is the lock for a custom saver's program that has ended
// (function.md §5.5): the word on the board — EXIT and its code, or
// NONE — in the clock's face at the largest size that fits, and the
// note in red on the status row.
func NewLockWord(cfg config.Config, problem string, word saver.Word, note string) *LockModel {
	return newLock(cfg, problem, false).withWord(word, note)
}

// withWord puts word on the board in place of whatever the profile
// draws, and note on the status row.
func (m *LockModel) withWord(word saver.Word, note string) *LockModel {
	m.word, m.note, m.game = word, note, nil
	m.scale, m.face = 3, faceOf("3x7")
	// A custom profile has no colours of its own (user, 2026-09-25): the
	// board's ground is the default, and EXIT the default gold; the code
	// after it wears the green of a thing that is on for 0, and peach for
	// anything else. NONE — nothing ran — wears the red a wrong PIN
	// wears, all of it (user: red is NONE's alone).
	m.style = config.Style{BG: config.DefaultBG, FG: config.DefaultFG}
	m.accent, m.accentFrom = peachColor, len("EXIT ")
	if word.Fine() {
		m.accent = liveColor
	}
	if word == saver.WordNone {
		m.style.FG, m.accentFrom = string(warnColor), 0
	}
	return m
}

// Tries is a lock's wrong PINs in a row and the end of its cooling-off
// (function.md §4.4): the whole lock's, which a custom saver's prompts —
// a program each — hand on from one to the next.
type Tries struct {
	failures int
	until    time.Time
}

// Tries is what this lock leaves the next prompt.
func (m *LockModel) Tries() Tries { return Tries{m.failures, m.lockoutUntil} }

// NewLockPrompt is the PIN box alone, for the custom saver: up from the
// first frame, painted by paint over the program's picture on a cols ×
// rows screen; Esc or the timeout ends the program with Back set, the
// right PIN with it clear. tries is where the prompt before it left off.
func NewLockPrompt(cfg config.Config, problem string, cols, rows int, paint func(box string, w int), tries Tries) *LockModel {
	m := newLock(cfg, problem, false)
	m.promptOnly, m.game, m.paint = true, nil, paint
	m.failures, m.lockoutUntil = tries.failures, tries.until
	m.width, m.height = cols, rows
	m.app.Update(tea.WindowSizeMsg{Width: cols, Height: rows})
	m.initCmd = m.openBox()
	return m
}

// Back reports whether a prompt-only lock ended because the box closed —
// the program is to be put back — rather than by the PIN.
func (m *LockModel) Back() bool { return m.back }

func newLock(cfg config.Config, problem string, preview bool) *LockModel {
	s, ok := cfg.Active()
	if !ok && problem == "" {
		problem = fmt.Sprintf("profile %q not found", cfg.Profile)
	}
	m := &LockModel{
		cfg:     cfg,
		problem: problem,
		clock:   saver.Clock{Time: s.Time, Date: s.Date, Layout: s.Layout}.Normalized(),
		scale:   saver.Scale(s.Size),
		face:    faceOf(s.Font),
		style:   s.Colours(),
		noPIN:   !cfg.HasPIN(),
		preview: preview,
		now:     time.Now,
	}
	seed := uint64(time.Now().UnixNano())
	switch s.Saver {
	case saver.KindRunner:
		m.game = saver.NewDino(seed, s.Participants, s.Character, s.Scene, s.Background, time.Now)
	case saver.KindBounce:
		m.game = saver.NewBounce(seed, time.Now, func(l string) []string { return spell(faceShort, l) }, s.Speed, s.Time)
	case saver.KindSnake:
		m.game = saver.NewSnake(seed, s.Speed)
	case saver.KindTetromino:
		m.game = saver.NewTetromino(seed, s.Speed, func(l string) []string { return spell(faceShort, l) })
	case saver.KindPets:
		m.game = saver.NewPets(seed, s.Count)
	}
	if !saver.Coloured(s.Saver) {
		// No colours of its own: what the lock draws in the profile's
		// wears the defaults, as a custom saver's prompt does.
		m.style = config.Style{BG: config.DefaultBG, FG: config.DefaultFG}
	}
	m.lockedAt = m.now()
	m.user, m.host = whoami()
	m.app = m.newApp()
	return m
}

// newApp is the lib's app the lock runs in. It locks a terminal of any size
// (tdp L1: the least size is the app's, and this is not a surface), and
// Ctrl-C does not leave it: the lock ends by the PIN alone (§1.2). A key on
// the board is the lock's (CanvasKey).
func (m *LockModel) newApp() *terminu.App {
	a := terminu.New("locku")
	a.MinWidth, a.MinHeight = 1, 1
	a.QuitFlow = func() tea.Cmd { return nil }
	a.Canvas = m.canvas
	a.CanvasKey = func(string, tea.KeyMsg) (bool, tea.Cmd) { return true, m.openBox() }
	return a
}

// whoami is the status row's "user@host": the user running the lock and
// the machine's short name. A variable so a test can pin it: the row is
// clipped to the terminal, and a long hostname (GitHub's macOS runner,
// 62 characters without a dot, 2026-09-25) pushed the words a test looks
// for off the end.
var whoami = func() (string, string) {
	name := os.Getenv("USER")
	if u, err := user.Current(); err == nil && u.Username != "" {
		name = u.Username
	}
	host, _ := os.Hostname()
	if i := strings.IndexByte(host, '.'); i > 0 {
		host = host[:i]
	}
	return name, host
}

func (m *LockModel) Init() tea.Cmd { return tea.Batch(m.app.Init(), m.clockTick(), m.initCmd) }

func (m *LockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.step(msg)
	if m.promptOnly && m.paint != nil && !m.over {
		// The box as the lib draws it over a blank screen, taken out of it:
		// the program's picture is under it, and nothing else is drawn.
		box := boxOf(m.app.Frame())
		m.paint(strings.Join(box, "\n"), blockWidth(box))
	}
	return m, cmd
}

// step is Update for the tests and the preview, the model itself back so a
// test reads as a run of steps.
func (m *LockModel) step(msg tea.Msg) (*LockModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// A resize redraws whole; nothing animates (function.md §5.3).
		m.tickGen++
		m.rev = nil
		m.shown = m.refit()
		return m, tea.Batch(m.toApp(msg), m.clockTick())
	case TTYGoneMsg:
		m.gone, m.over = true, true
		return m, tea.Quit
	case clockTickMsg:
		if msg.gen != m.tickGen {
			return m, nil
		}
		if m.game != nil {
			// A frame of the run: the board is replaced whole, no reveal —
			// the world moves, it does not change.
			m.game.Step()
			m.shown, m.rev = m.refit(), nil
			return m, m.clockTick()
		}
		return m, tea.Batch(m.redraw(), m.clockTick())
	case revealTickMsg:
		if msg.gen != m.tickGen || m.rev == nil {
			return m, nil
		}
		if m.rev.advance() {
			m.shown, m.rev = m.rev.to, nil
			return m, nil
		}
		m.shown = m.rev.cur
		return m, m.revealTick()
	case boxIdleMsg:
		// Not while a wrong PIN is said or a cooling-off counts down: each
		// starts the timer again when it is over.
		if msg.gen != m.idleGen || !m.boxUp || m.holding() || m.lockedOut() {
			return m, nil
		}
		return m, m.app.CloseAll()
	case heldOverMsg:
		if !m.boxUp || m.holding() {
			return m, nil
		}
		return m, m.armTimeout()
	case lockoutOverMsg:
		if m.lockoutUntil.IsZero() || m.lockedOut() {
			return m, nil
		}
		// The cooling-off is over and the count starts again (§4.4); the
		// wrong PIN that led to it goes from the box.
		m.endLockout()
		if !m.boxUp {
			return m, nil
		}
		return m, tea.Batch(m.toApp(tea.KeyMsg{Type: tea.KeyBackspace}), m.armTimeout())
	case tea.KeyMsg:
		return m, m.key(msg)
	}
	// The lib's own: the box's animation, the cooling-off's count.
	return m, m.toApp(msg)
}

// toApp hands msg to the lib's app.
func (m *LockModel) toApp(msg tea.Msg) tea.Cmd {
	_, cmd := m.app.Update(msg)
	return cmd
}

// key is every keystroke (ux.md §2.3). On the board any key opens the box
// and is not input; in the box it is the box's, and starts the timeout
// again — but for the second of "wrong", when every key is swallowed.
func (m *LockModel) key(msg tea.KeyMsg) tea.Cmd {
	// Nothing is drawn before the terminal's size is known, and a key
	// there is not one the user saw anything for.
	if m.width <= 0 || m.height <= 0 {
		return nil
	}
	// A preview is a look at the saver, not the lock: any key hands back,
	// PIN or no PIN — the PIN is `locku lock`'s business (user,
	// 2026-09-25). The status row still reads as the lock's would.
	if m.preview {
		return m.unlock()
	}
	// The PIN is read off the file at every key, not once at the start,
	// so a reset from another shell — `locku pin reset`, a new hash
	// written under a lock already up — takes at the next key (user,
	// 2026-09-25); a file that cannot be read keeps the hash the lock
	// has.
	if h, ok := config.LoadPINHash(); ok {
		m.cfg.PINHash, m.noPIN = h, h == ""
	}
	if m.noPIN {
		return m.unlock()
	}
	if !m.boxUp {
		return m.toApp(msg)
	}
	if m.holding() {
		return nil
	}
	cmd := m.toApp(msg)
	if !m.wrong {
		// Every key restarts the idle timer, so a PIN typed slowly does
		// not vanish halfway (function.md §3).
		return tea.Batch(cmd, m.armTimeout())
	}
	m.wrong = false
	if m.lockedOut() {
		// The box counts the cooling-off down (tdp S37), every key but Esc
		// swallowed; when it is over the count starts again.
		return tea.Batch(cmd, m.app.Tick(m.lockoutUntil.Sub(m.now()), func(time.Time) tea.Msg { return lockoutOverMsg{} }))
	}
	// A second of "wrong": the dots go — Backspace clears a password whole
	// — and every key is swallowed, so Enter pressed again and again does
	// not skip the word (function.md §4.4).
	m.held = m.now().Add(wrongHold)
	return tea.Batch(cmd, m.toApp(tea.KeyMsg{Type: tea.KeyBackspace}),
		m.app.Tick(wrongHold, func(time.Time) tea.Msg { return heldOverMsg{} }))
}

func (m *LockModel) holding() bool   { return m.now().Before(m.held) }
func (m *LockModel) lockedOut() bool { return m.now().Before(m.lockoutUntil) }

// openBox puts the PIN box up. The board goes on underneath it — ticking,
// revealing — dimmed to a backdrop (user, 2026-09-24: the colour changes,
// the clock does not stop).
func (m *LockModel) openBox() tea.Cmd {
	if !m.lockoutUntil.IsZero() && !m.lockedOut() {
		// It ran out while the box was down: the count starts again all the
		// same (2026-10-06; it used to stay, and the next wrong PIN was a
		// cooling-off again).
		m.endLockout()
	}
	m.boxUp = true
	cmd := m.app.Password(terminu.PasswordInput{Title: pinTitle, Glyph: glyphLock, Name: "PIN", Verb: "unlock",
		Unlock: true, MaxLen: config.PINMax,
		Check: m.check,
		// The lib counts on the clock, the lock on its own (a test's).
		PausedUntil: func() time.Time { return time.Now().Add(m.lockoutUntil.Sub(m.now())) },
		Done:        func(string, bool) tea.Cmd { return m.unlock() },
		OnCancel:    m.boxClosed})
	if m.lockedOut() {
		cmd = tea.Batch(cmd, m.app.Tick(m.lockoutUntil.Sub(m.now()), func(time.Time) tea.Msg { return lockoutOverMsg{} }))
	}
	return tea.Batch(cmd, m.armTimeout())
}

// endLockout starts the count again once a cooling-off is over.
func (m *LockModel) endLockout() { m.failures, m.lockoutUntil = 0, time.Time{} }

// boxClosed is the box gone without the PIN — Esc, or the timeout; the
// board never stopped. Prompt-only, the program is waiting for the
// screen: this lock ends, with Back.
func (m *LockModel) boxClosed() tea.Cmd {
	m.boxUp = false
	m.idleGen++
	m.held = time.Time{}
	if m.promptOnly {
		m.back, m.over = true, true
		return tea.Quit
	}
	return nil
}

// check is Enter in the box: the right PIN ends the lock at once — no
// closing animation, the terminal is wanted back (ux.md §2.3); a wrong one
// is a second of "wrong" (key), or, at the configured count, a cooling-off.
// A PIN with a control character in it — pasted — the lib refuses before
// it gets here: it is not a wrong one.
func (m *LockModel) check(pin string) error {
	if m.cfg.CheckPIN(pin) {
		return nil
	}
	m.failures++
	m.wrong = true
	if m.cfg.WrongPINAttempts > 0 && m.failures >= m.cfg.WrongPINAttempts {
		m.lockoutUntil = m.now().Add(time.Duration(m.cfg.WrongPINCooldown) * time.Second)
	}
	return errWrongPIN
}

// TTYGone reports whether the lock ended because the terminal went away
// rather than because it was unlocked.
func (m *LockModel) TTYGone() bool { return m.gone }

// unlock is the way out: the process ends, or, in a preview, the settings
// screen takes over again.
func (m *LockModel) unlock() tea.Cmd {
	if m.preview {
		m.unlocked = true
		return tea.Quit
	}
	m.over = true
	return tea.Quit
}

// refit lays the current time out for the current size and returns the
// board it makes, setting lines and k on the way.
func (m *LockModel) refit() board {
	rows := m.height - 1
	if m.game != nil {
		// A game has no size of its own: as large as the terminal allows.
		k, w, h := fitScene(m.game.Room(), m.width, rows)
		return paintScene(m.game.Draw(w, h), k, m.width, rows)
	}
	var s saver.Saver = m.clock
	if m.word != "" {
		s = m.word
	}
	m.layout, m.plain = fit(m.face, s, m.now(), m.width, rows, m.scale)
	m.layout.accentFrom = m.accentFrom
	return paint(m.face, m.layout, m.width, rows)
}

// redraw moves the board to now: by a reveal when only some pixels change,
// at once when the board is a different size or the same.
func (m *LockModel) redraw() tea.Cmd {
	target := m.refit()
	from := m.shown
	if m.rev != nil {
		from = m.rev.cur
	}
	if r := newReveal(from, target); r != nil {
		m.rev = r
		return m.revealTick()
	}
	m.shown, m.rev = target, nil
	return nil
}

// clockTick is the next change of content: the clock's next minute or
// second, or the run's next frame.
func (m *LockModel) clockTick() tea.Cmd {
	gen := m.tickGen
	now := m.now()
	next := m.clock.Next(now)
	if m.game != nil {
		next = m.game.Next(now)
	}
	if m.word != "" {
		next = m.word.Next(now)
	}
	return m.app.Tick(next.Sub(now), func(time.Time) tea.Msg { return clockTickMsg{gen} })
}

func (m *LockModel) revealTick() tea.Cmd {
	gen := m.tickGen
	return m.app.Tick(revealStep, func(time.Time) tea.Msg { return revealTickMsg{gen} })
}

// armTimeout starts pin_prompt_timeout over; 0 means the box stays.
func (m *LockModel) armTimeout() tea.Cmd {
	m.idleGen++
	if m.cfg.PINPromptTimeout <= 0 {
		return nil
	}
	gen := m.idleGen
	d := time.Duration(m.cfg.PINPromptTimeout) * time.Second
	return m.app.Tick(d, func(time.Time) tea.Msg { return boxIdleMsg{gen} })
}

// inks are the board's colours: a game's own when it has them (user,
// 2026-10-06), else the profile's ground and fg, and the accent.
func (m *LockModel) inks() []lipgloss.Color {
	if g, ok := m.game.(saver.Inked); ok {
		var out []lipgloss.Color
		for _, c := range g.Inks() {
			out = append(out, lipgloss.Color(c))
		}
		return out
	}
	return []lipgloss.Color{lipgloss.Color(m.style.BG), lipgloss.Color(m.style.FG), m.accent}
}

// shading is the board's colours square by square, when the game's
// are not one set (user, 2026-10-07: the runner's sky); else nil, every
// square in its ink.
func (m *LockModel) shading() *saver.Shading {
	g, ok := m.game.(saver.Graded)
	if !ok {
		return nil
	}
	s := g.Shade(m.shown.w, m.shown.h)
	return &s
}

// canvas is the lock as the App draws it under the box: the board and the
// status row. Under the box the lib fades it whole, each colour its own way
// (tdp F8, D2, 2026-09-28). A prompt-only lock draws nothing: the program's
// picture is under its box.
func (m *LockModel) canvas(w, h int) []string {
	if m.promptOnly {
		return nil
	}
	bg, fg := lipgloss.Color(m.style.BG), lipgloss.Color(m.style.FG)
	rows := m.height - 1
	var out []string
	if rows > 0 {
		switch {
		case len(m.layout.blocks) > 0 || m.game != nil:
			out = boardRows(m.shown, m.inks(), m.shading(), m.width)
		default:
			out = plainRows(m.plain, bg, fg, m.width, rows)
		}
	}
	return append(out, statusRow(m.width, m.cfg.ShowStatus, m.user, m.host, m.lockedAt, m.noPIN, m.problem, m.note))
}

func (m *LockModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	return m.app.View()
}

// boxOf is what the lib drew over a blank canvas, cut out of its frame: the
// rows and the columns that are not blank.
func boxOf(frame []string) []string {
	top, bottom, left, right := -1, -1, 1<<30, 0
	for i, l := range frame {
		plain := terminu.StripStyles(l)
		if strings.TrimSpace(plain) == "" {
			continue
		}
		if top < 0 {
			top = i
		}
		bottom = i
		left = min(left, terminu.Width(plain)-terminu.Width(strings.TrimLeft(plain, " ")))
		right = max(right, terminu.Width(strings.TrimRight(plain, " ")))
	}
	if top < 0 {
		return nil
	}
	out := make([]string, 0, bottom-top+1)
	for _, l := range frame[top : bottom+1] {
		out = append(out, terminu.Cut(l, left, right))
	}
	return out
}
