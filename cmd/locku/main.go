// Command locku is a screensaver with a PIN for the terminal: tmux's
// lock-command, screen's LOCKPRG, or a bare `locku lock` on any tty.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/custom"
	"github.com/vulcanshen/locku/internal/login"
	"github.com/vulcanshen/locku/internal/saver"
	"github.com/vulcanshen/locku/internal/tmux"
	"github.com/vulcanshen/locku/internal/ui"
	"github.com/vulcanshen/locku/internal/version"
)

const usage = `locku — a screensaver with a PIN, for the terminal

  locku                      settings: the PIN, the savers, the colours
  locku lock                 lock this terminal: what tmux's lock-command
                             and screen's LOCKPRG run, and what a bare tty
                             runs by hand. tmux's lock-command, written by
                             the settings screen, adds -S <socket> — the
                             server to mark locked — and, for lock-session,
                             -t <session> — the session to mark; you never
                             type them. screen runs it as SCREEN-LOCK with
                             no arguments at all.
  locku pin reset            a new PIN, made and shown once, after y/N and
                             your login password — the way back from a PIN
                             forgotten; a lock already up takes it at its
                             next key
  locku version              the version
  locku help                 this
`

func main() {
	// screen runs LOCKPRG by execl with argv[0] set to SCREEN-LOCK and no
	// arguments: that name is the whole message (function.md §6).
	if filepath.Base(os.Args[0]) == "SCREEN-LOCK" {
		os.Exit(runLock("", ""))
	}
	args := os.Args[1:]
	switch {
	case len(args) == 0:
		os.Exit(runSettings())
	case args[0] == "lock":
		os.Exit(runLock(lockArgs(args[1:])))
	case args[0] == "pin" && len(args) > 1 && args[1] == "reset":
		os.Exit(pinReset(os.Stdin, os.Stdout, login.Verify))
	case args[0] == "version":
		fmt.Println("locku " + version.Display())
	case args[0] == "help" || args[0] == "-h" || args[0] == "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "locku: unknown command %q\n\n%s", args[0], usage)
		os.Exit(2)
	}
}

// lockArgs is what follows `lock`: -S, the tmux server's socket, and -t,
// the session whose lock this is, either "" when not given.
func lockArgs(args []string) (socket, session string) {
	for i := 0; i+1 < len(args); i++ {
		switch args[i] {
		case "-S":
			socket = args[i+1]
		case "-t":
			session = args[i+1]
		}
	}
	return socket, session
}

// runLock is `locku lock`. The process is the lock (function.md §1.2): it
// ends for the right PIN, for any key when there is no PIN, and for a
// terminal that has gone away — and for nothing else. Signals are ignored
// rather than handled (§2.2), and a program that comes down for any other
// reason, a panic included, goes straight back up.
func runLock(socket, session string) int {
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP,
		syscall.SIGTSTP, syscall.SIGTTIN, syscall.SIGTTOU)
	cfg, problem := config.Load()
	// Under tmux the server — or, for lock-session, this session — is
	// marked locked for as long as this runs, so a client attaching to it
	// meanwhile is locked too (function.md §6.2); a terminal that goes
	// away leaves the mark, and the next client in meets the lock.
	tmux.SetLocked(socket, session, true)
	var code int
	var unlocked bool
	if p, ok := cfg.Active(); ok && p.Saver == saver.KindCustom {
		code, unlocked = runCustom(cfg, problem, p.Command)
	} else {
		code, unlocked = runBoard(func() ui.LockModel { return ui.NewLock(cfg, problem) })
	}
	if unlocked {
		tmux.SetLocked(socket, session, false)
	}
	return code
}

// runBoard runs a lock program on the terminal until it ends, and puts
// it back up when it comes down for any other reason; it reports
// whether the lock was unlocked, as against the terminal going away.
func runBoard(model func() ui.LockModel) (code int, unlocked bool) {
	quick := 0
	for {
		started := time.Now()
		var p *tea.Program
		in := ui.LockInput(os.Stdin, func() { p.Send(ui.TTYGoneMsg{}) })
		p = tea.NewProgram(model(),
			tea.WithAltScreen(),
			tea.WithoutSignalHandler(),
			tea.WithInput(in),
		)
		if m, err := p.Run(); err == nil {
			lm, ok := m.(ui.LockModel)
			return 0, !ok || !lm.TTYGone()
		}
		// Bubble Tea has restored the terminal; the lock goes back up. A
		// program that cannot even start — three failures inside a second
		// each — is a terminal nothing can be drawn on, and there is
		// nothing to protect.
		if time.Since(started) < time.Second {
			if quick++; quick >= 3 {
				return 1, false
			}
		} else {
			quick = 0
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// runCustom is the lock with the user's own program for a saver
// (function.md §5.5; user, 2026-09-25): the terminal is locku's — raw,
// on the alternate screen — and the program's, on a pty, is passed on to
// it as it comes. A key holds the program's picture back and puts the
// PIN prompt up on a clear screen, a lock program of its own; Esc or
// the timeout brings the picture back, the right PIN ends everything,
// the program killed. The program ending is not the lock ending: the
// board comes up with the word for it, and stays until the PIN.
func runCustom(cfg config.Config, problem, command string) (int, bool) {
	board := func(o custom.Outcome) (int, bool) {
		return runBoard(func() ui.LockModel { return ui.NewLockWord(cfg, problem, o.Word, o.Note) })
	}
	if strings.TrimSpace(command) == "" {
		return board(custom.NoCommand())
	}
	t, err := custom.Take(os.Stdin, os.Stdout)
	if err != nil {
		return board(custom.Failed(err))
	}
	cols, rows := t.Size()
	p, err := custom.Start(command, t.Writer(), cols, rows)
	if err != nil {
		t.Give()
		return board(custom.Failed(err))
	}
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	keys := t.Key()
	for {
		select {
		case <-winch:
			c, r := t.Size()
			t.Resized(c, r)
			p.Resize(c, r)
		case o := <-p.Done():
			// Ended on its own: the word on locku's board from here on.
			t.Cancel()
			<-keys
			p.Kill()
			t.Give()
			return board(o)
		case err := <-keys:
			if err != nil {
				// The terminal went away: nothing left to protect, and
				// nothing was unlocked.
				p.Kill()
				t.Give()
				return 0, false
			}
			// The PIN as the file has it now — as the board lock reads it
			// at every key (function.md §4.5): none, and this key ends it.
			noPIN := !cfg.HasPIN()
			if h, ok := config.LoadPINHash(); ok {
				noPIN = h == ""
			}
			if noPIN {
				p.Kill()
				t.Give()
				return 0, true
			}
			// The picture goes on; the prompt rides over it. When it
			// closes its place is blanked, and a program that sits still
			// is asked to paint itself; one that draws fills the place by
			// itself, and is not asked — asked, a curses program starts
			// its picture over.
			unlocked, back := runPrompt(cfg, problem, t)
			if !back {
				p.Kill()
				t.Give()
				return 0, unlocked
			}
			t.Clear()
			if p.Idle(500 * time.Millisecond) {
				p.Redraw()
			}
			keys = t.Key()
		}
	}
}

// runPrompt is the PIN prompt on its own, a lock program with no
// renderer that draws its box through t.Overlay, over the program's
// frozen picture, and nothing else (user, 2026-09-25: a ground of
// locku's hid the picture, and a switch of screens under the prompt
// left it to be painted again). It reports whether the PIN unlocked,
// and whether the prompt closed instead — Esc, the timeout — so the
// picture is to come back. A prompt that cannot even run leaves the
// lock as it is, the program back on the screen.
func runPrompt(cfg config.Config, problem string, t *custom.Terminal) (unlocked, back bool) {
	var p *tea.Program
	in := ui.LockInput(os.Stdin, func() { p.Send(ui.TTYGoneMsg{}) })
	cols, rows := t.Size()
	p = tea.NewProgram(ui.NewLockPrompt(cfg, problem, cols, rows, t.Overlay),
		tea.WithoutRenderer(),
		tea.WithoutSignalHandler(),
		tea.WithInput(in),
	)
	m, err := p.Run()
	if err != nil {
		return false, true
	}
	lm, ok := m.(ui.LockModel)
	switch {
	case !ok:
		return true, false
	case lm.TTYGone():
		return false, false
	case lm.Back():
		return false, true
	}
	return true, false
}

// runSettings is the bare `locku`: the settings screen, where the tmux
// and screen integration is set up and removed too (user, 2026-09-25:
// buttons, in place of a `locku setup` command).
func runSettings() int {
	cfg, problem := config.Load()
	p := tea.NewProgram(ui.NewApp(cfg, problem), tea.WithAltScreen())
	_, err := p.Run()
	switch {
	case errors.Is(err, tea.ErrInterrupted):
		return 130 // the conventional 128+SIGINT
	case err != nil:
		fmt.Fprintln(os.Stderr, "locku:", err)
		return 1
	}
	return 0
}
