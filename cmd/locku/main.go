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
	"github.com/vulcanshen/locku/internal/ui"
	"github.com/vulcanshen/locku/internal/version"
)

const usage = `locku — a screensaver with a PIN, for the terminal

  locku                      settings: the PIN, the savers, the colours
  locku lock                 lock this terminal — what tmux and screen run
  locku setup [tmux|screen]  write the lock into ~/.tmux.conf, ~/.screenrc and
                             the shell rc; both without an argument
  locku version              the version
  locku help                 this
`

func main() {
	// screen runs LOCKPRG by execl with argv[0] set to SCREEN-LOCK and no
	// arguments: that name is the whole message (function.md §6).
	if filepath.Base(os.Args[0]) == "SCREEN-LOCK" {
		os.Exit(runLock())
	}
	args := os.Args[1:]
	switch {
	case len(args) == 0:
		os.Exit(runSettings())
	case args[0] == "lock":
		os.Exit(runLock())
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

// runLock is `locku lock`. The process is the lock (function.md §1.2): it
// ends for the right PIN, for any key when there is no PIN, and for a
// terminal that has gone away — and for nothing else. Signals are ignored
// rather than handled (§2.2), and a program that comes down for any other
// reason, a panic included, goes straight back up.
func runLock() int {
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP,
		syscall.SIGTSTP, syscall.SIGTTIN, syscall.SIGTTOU)
	cfg, problem := config.Load()
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
		if _, err := p.Run(); err == nil {
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

// runSetup is `locku setup [tmux|screen]`: both without an argument.
func runSetup(args []string) int {
	targets := args
	if len(targets) == 0 {
		targets = []string{"tmux", "screen"}
	}
	code := 0
	for _, t := range targets {
		var err error
		switch t {
		case "tmux":
			err = setup.Tmux(os.Stdout)
		case "screen":
			err = setup.Screen(os.Stdout)
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
