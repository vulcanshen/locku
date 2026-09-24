package ui

import (
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// LockModel is `locku lock` (function.md §3): the board, the status row and
// the PIN prompt. The process IS the lock — while it runs the terminal is
// locked — and there are exactly three ways out: the right PIN, any key
// when no PIN is set, and the terminal going away (§1.2). Nothing here
// returns tea.Quit for any other reason.
type LockModel struct {
	cfg     config.Config
	problem string
	clock   saver.Clock
	noPIN   bool

	width, height int
	lines         []string // what the board spells now
	k             int      // its scale; 0 draws the lines as text
	shown         board    // the board on screen
	rev           *reveal  // the change in progress, if one is
	tickGen       int      // a clock or reveal tick from before a change carries an older gen

	prompt       pinPrompt
	promptGen    int // likewise for the prompt's timers
	failures     int // consecutive wrong PINs; process-lifetime (§4.4)
	lockoutUntil time.Time

	lockedAt   time.Time
	user, host string

	// preview: this lock runs inside the settings screen (ui.md §2.1), so
	// an unlock hands control back rather than ending the process.
	preview  bool
	unlocked bool

	now func() time.Time
}

// Messages the lock sends itself. Each carries the generation it was
// scheduled under, so a timer from before a state change is ignored when
// it fires.
type (
	clockTickMsg   struct{ gen int }
	revealTickMsg  struct{ gen int }
	promptIdleMsg  struct{ gen int } // prompt_timeout ran out
	wrongOverMsg   struct{ gen int } // the second of "wrong" is over
	lockoutTickMsg struct{ gen int } // once a second while locked out
)

// TTYGoneMsg says the terminal went away under the lock: read returned
// EOF or EIO. There is nothing left to protect, and the process ends.
type TTYGoneMsg struct{}

// NewLock is the model for `locku lock`.
func NewLock(cfg config.Config, problem string) LockModel { return newLock(cfg, problem, false) }

func newLock(cfg config.Config, problem string, preview bool) LockModel {
	s, ok := cfg.Active()
	if !ok && problem == "" {
		problem = fmt.Sprintf("saver %q not found", cfg.Saver)
	}
	m := LockModel{
		cfg:     cfg,
		problem: problem,
		clock:   saver.Clock{Time: s.Time, Date: s.Date}.Normalized(),
		noPIN:   !cfg.HasPIN(),
		prompt:  newPinPrompt(),
		preview: preview,
		now:     time.Now,
	}
	m.lockedAt = m.now()
	m.user, m.host = whoami()
	return m
}

// whoami is the status row's "user@host": the user running the lock and
// the machine's short name.
func whoami() (string, string) {
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

func (m LockModel) Init() tea.Cmd { return m.clockTick() }

func (m LockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.step(msg)
}

// step is Update with the concrete type, so the settings screen can host a
// preview without a type assertion on every message.
func (m LockModel) step(msg tea.Msg) (LockModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.prompt.setSize(msg.Width, msg.Height)
		// A resize redraws whole; nothing animates (function.md §5.3).
		m.tickGen++
		m.rev = nil
		m.shown = m.refit()
		return m, m.clockTick()
	case TTYGoneMsg:
		return m, tea.Quit
	case clockTickMsg:
		if msg.gen != m.tickGen {
			return m, nil
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
	case AnimTickMsg:
		return m, m.prompt.anim.tick(msg)
	case promptIdleMsg:
		if msg.gen != m.promptGen || !m.prompt.anim.owns() || m.prompt.state != promptIdle {
			return m, nil
		}
		return m, m.closePrompt(true)
	case wrongOverMsg:
		if msg.gen != m.promptGen || m.prompt.state != promptWrong {
			return m, nil
		}
		m.prompt.state = promptIdle
		return m, m.armTimeout()
	case lockoutTickMsg:
		if msg.gen != m.promptGen || m.prompt.state != promptLockout {
			return m, nil
		}
		if !m.now().Before(m.lockoutUntil) {
			// The cooling-off is over and the count starts again (§4.4).
			m.failures = 0
			m.prompt.state = promptIdle
			return m, m.armTimeout()
		}
		return m, m.lockoutTick()
	case tea.KeyMsg:
		return m.key(msg)
	}
	return m, nil
}

// key is every keystroke (ux.md §2.3). On the saver any key opens the
// prompt and is not input. In the prompt: Enter checks, Esc goes back,
// Backspace deletes, a printable character is appended — and while the
// prompt says "wrong" or is locked out, keys are swallowed, Esc excepted
// during a lockout.
func (m LockModel) key(msg tea.KeyMsg) (LockModel, tea.Cmd) {
	if m.noPIN {
		return m.unlock()
	}
	if !m.prompt.anim.owns() {
		return m, m.openPrompt()
	}
	if !m.prompt.anim.isInteractive() {
		return m, nil
	}
	switch m.prompt.state {
	case promptWrong:
		return m, nil
	case promptLockout:
		if msg.Type == tea.KeyEsc {
			return m, m.closePrompt(false)
		}
		return m, nil
	}
	switch msg.Type {
	case tea.KeyEsc:
		return m, m.closePrompt(false)
	case tea.KeyEnter:
		return m.check()
	case tea.KeyBackspace:
		m.prompt.backspace()
	case tea.KeySpace:
		m.prompt.add(' ')
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			m.prompt.add(r)
		}
	}
	// Every key restarts the idle timer, so a PIN typed slowly does not
	// vanish halfway (function.md §3).
	return m, m.armTimeout()
}

// openPrompt puts the prompt up. The board holds still underneath it: its
// ticks stop, and a reveal in flight is completed on the spot.
func (m *LockModel) openPrompt() tea.Cmd {
	m.tickGen++
	if m.rev != nil {
		m.shown, m.rev = m.rev.to, nil
	}
	m.promptGen++
	cmd := m.prompt.open()
	if m.now().Before(m.lockoutUntil) {
		m.prompt.state = promptLockout
		return tea.Batch(cmd, m.lockoutTick())
	}
	return tea.Batch(cmd, m.armTimeout())
}

// closePrompt takes it down and brings the board up to date: it was not
// ticking while the prompt was up (ux.md §6).
func (m *LockModel) closePrompt(timedOut bool) tea.Cmd {
	m.promptGen++
	cmd := m.prompt.close(timedOut)
	m.tickGen++
	return tea.Batch(cmd, m.redraw(), m.clockTick())
}

// check is Enter in the prompt: the right PIN ends the lock at once — no
// closing animation, the terminal is wanted back (ux.md §2.3). A wrong one
// is a second of "wrong" with every key swallowed, or, at the configured
// count, a lockout.
func (m LockModel) check() (LockModel, tea.Cmd) {
	pin := string(m.prompt.value)
	m.prompt.value = nil
	if m.cfg.CheckPIN(pin) {
		return m.unlock()
	}
	m.failures++
	m.promptGen++
	if m.cfg.LockoutAfter > 0 && m.failures >= m.cfg.LockoutAfter {
		m.lockoutUntil = m.now().Add(time.Duration(m.cfg.LockoutSeconds) * time.Second)
		m.prompt.until = m.lockoutUntil
		m.prompt.state = promptLockout
		return m, m.lockoutTick()
	}
	m.prompt.state = promptWrong
	gen := m.promptGen
	return m, tea.Tick(wrongHold, func(time.Time) tea.Msg { return wrongOverMsg{gen} })
}

// unlock is the way out: the process ends, or, in a preview, the settings
// screen takes over again.
func (m LockModel) unlock() (LockModel, tea.Cmd) {
	if m.preview {
		m.unlocked = true
		return m, nil
	}
	return m, tea.Quit
}

// refit lays the current time out for the current size and returns the
// board it makes, setting lines and k on the way.
func (m *LockModel) refit() board {
	rows := m.height - 1
	m.lines, m.k = fit(m.clock, m.now(), m.width, rows)
	return paint(m.lines, m.k, m.width, rows)
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

func (m LockModel) clockTick() tea.Cmd {
	gen := m.tickGen
	now := m.now()
	d := m.clock.Next(now).Sub(now)
	return tea.Tick(d, func(time.Time) tea.Msg { return clockTickMsg{gen} })
}

func (m LockModel) revealTick() tea.Cmd {
	gen := m.tickGen
	return tea.Tick(revealStep, func(time.Time) tea.Msg { return revealTickMsg{gen} })
}

// armTimeout starts prompt_timeout over; 0 means the prompt stays.
func (m *LockModel) armTimeout() tea.Cmd {
	m.promptGen++
	if m.cfg.PromptTimeout <= 0 {
		return nil
	}
	gen := m.promptGen
	d := time.Duration(m.cfg.PromptTimeout) * time.Second
	return tea.Tick(d, func(time.Time) tea.Msg { return promptIdleMsg{gen} })
}

func (m LockModel) lockoutTick() tea.Cmd {
	gen := m.promptGen
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return lockoutTickMsg{gen} })
}

func (m LockModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	bg, fg := lipgloss.Color(m.cfg.Style.BG), lipgloss.Color(m.cfg.Style.FG)
	dimmed := m.prompt.anim.isActive()
	rows := m.height - 1
	var out []string
	if rows > 0 {
		if m.k >= 1 {
			out = boardRows(m.shown, bg, fg, m.width, dimmed)
		} else {
			out = plainRows(m.lines, bg, fg, m.width, rows, dimmed)
		}
	}
	out = append(out, statusRow(m.width, m.cfg.ShowStatus, m.user, m.host, m.lockedAt, m.noPIN, m.problem))
	view := strings.Join(out, "\n")
	if m.prompt.anim.isActive() {
		view = overlay.Composite(m.prompt.view(m.now()), view, overlay.Center, overlay.Center, 0, 0)
	}
	return view
}
