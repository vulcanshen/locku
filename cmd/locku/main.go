// Command locku is a screensaver with a PIN for the terminal: tmux's
// lock-command, screen's LOCKPRG, or a bare `locku lock` on any tty.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/setup"
	"github.com/vulcanshen/locku/internal/tmux"
	"github.com/vulcanshen/locku/internal/ui"
	"github.com/vulcanshen/locku/internal/version"
)

const usage = `locku — a screensaver with a PIN, for the terminal

  locku                      settings: the PIN, the savers, the colours
  locku lock [-S socket]     lock this terminal — what tmux and screen run;
                             -S is the tmux server's socket, which setup's
                             lock-command passes
  locku setup [tmux|screen]  write the lock into the files preference names as
                             tmux_conf and screen_conf, and the shell rc; both
                             without a name. -d takes it out again
  locku setup -d [tmux|screen]
  locku version              the version
  locku help                 this
`

func main() {
	// screen runs LOCKPRG by execl with argv[0] set to SCREEN-LOCK and no
	// arguments: that name is the whole message (function.md §6).
	if filepath.Base(os.Args[0]) == "SCREEN-LOCK" {
		os.Exit(runLock(""))
	}
	args := os.Args[1:]
	switch {
	case len(args) == 0:
		os.Exit(runSettings())
	case args[0] == "lock":
		os.Exit(runLock(socketArg(args[1:])))
	case args[0] == "setup":
		os.Exit(runSetup(args[1:]))
	case args[0] == "version":
		fmt.Println("locku " + version.Display())
	case args[0] == "help" || args[0] == "-h" || args[0] == "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "locku: unknown command %q\n\n%s", args[0], usage)
		os.Exit(2)
	}
}

// socketArg is the -S after `lock`: the tmux server's socket, or "".
func socketArg(args []string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-S" {
			return args[i+1]
		}
	}
	return ""
}

// runLock is `locku lock`. The process is the lock (function.md §1.2): it
// ends for the right PIN, for any key when there is no PIN, and for a
// terminal that has gone away — and for nothing else. Signals are ignored
// rather than handled (§2.2), and a program that comes down for any other
// reason, a panic included, goes straight back up.
func runLock(socket string) int {
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP,
		syscall.SIGTSTP, syscall.SIGTTIN, syscall.SIGTTOU)
	cfg, problem := config.Load()
	// Under tmux the session is marked locked for as long as this runs,
	// so a client attaching meanwhile is locked too (function.md §6.2);
	// a terminal that goes away leaves the mark, and the next client in
	// meets the lock.
	session := tmux.Session(socket)
	tmux.SetLocked(socket, session, true)
	quick := 0
	for {
		started := time.Now()
		var p *tea.Program
		in := ui.LockInput(os.Stdin, func() { p.Send(ui.TTYGoneMsg{}) })
		p = tea.NewProgram(ui.NewLock(cfg, problem),
			tea.WithAltScreen(),
			tea.WithoutSignalHandler(),
			tea.WithInput(in),
		)
		if m, err := p.Run(); err == nil {
			if lm, ok := m.(ui.LockModel); !ok || !lm.TTYGone() {
				tmux.SetLocked(socket, session, false)
			}
			return 0
		}
		// Bubble Tea has restored the terminal; the lock goes back up. A
		// program that cannot even start — three failures inside a second
		// each — is a terminal nothing can be drawn on, and there is
		// nothing to protect.
		if time.Since(started) < time.Second {
			if quick++; quick >= 3 {
				return 1
			}
		} else {
			quick = 0
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// runSetup is `locku setup [-d] [tmux|screen]`: both without a name; -d
// takes locku out again (user, 2026-09-24). The files are the ones
// preference names; setup with one unset says so and writes nothing.
func runSetup(args []string) int {
	cfg, problem := config.Load()
	if problem != "" {
		fmt.Fprintf(os.Stderr, "locku: %s\n", problem)
	}
	undo := false
	if len(args) > 0 && args[0] == "-d" {
		undo, args = true, args[1:]
	}
	targets := args
	if len(targets) == 0 {
		targets = []string{"tmux", "screen"}
	}
	code := 0
	for _, t := range targets {
		var err error
		switch {
		case t == "tmux" && undo:
			err = setup.TmuxUndo(os.Stdout, cfg.TmuxConf)
		case t == "tmux":
			err = setup.Tmux(os.Stdout, cfg.TmuxConf)
		case t == "screen" && undo:
			err = setup.ScreenUndo(os.Stdout, cfg.ScreenConf)
		case t == "screen":
			err = setup.Screen(os.Stdout, cfg.ScreenConf)
		default:
			fmt.Fprintf(os.Stderr, "locku: setup takes tmux or screen, not %q\n", t)
			return 2
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "locku: setup %s: %v\n", t, err)
			code = 1
		}
	}
	return code
}

// runSettings is the bare `locku`: the settings screen.
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
