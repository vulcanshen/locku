package ui

import (
	"io"
	"os"
	"sync"

	"github.com/vulcanshen/locku/internal/termreply"
)

// lockInput is the terminal, watched. Bubble Tea's own read loop treats
// end-of-file as nothing to report and keeps the program up, which for a
// lock would mean a process guarding a terminal that no longer exists
// (function.md §1.2 case three). This wrapper turns the first read error —
// EOF on macOS, EIO on Linux when the pty's master side closes — into a
// call the lock answers with TTYGoneMsg, and hands Bubble Tea a clean EOF.
//
// It implements the file methods Bubble Tea and its cancel reader look
// for, so the terminal is still put into raw mode and reads can still be
// cancelled on the way out; without them the input would be read as a
// pipe, and the last read would hang the exit for half a second.
//
// And it hands on keys only: the terminal's answers to what it was asked
// — tmux asks a few things as a client attaches — are taken out, since
// Bubble Tea reads an OSC answer as Alt-] and a run of characters, and
// any key opens the prompt, or with no PIN unlocks (2026-09-28).
type lockInput struct {
	tty   *os.File
	gone  func()
	once  sync.Once
	reply termreply.Filter
}

// LockInput wraps tty; gone is called once, from the reading goroutine,
// the first time a read fails.
func LockInput(tty *os.File, gone func()) io.Reader {
	return &lockInput{tty: tty, gone: gone}
}

// Read reads less than p holds by what the filter holds back, so that the
// keys it gives out, those bytes on top, fit in p. A read that was all
// answer gives nothing: Bubble Tea reads again.
func (l *lockInput) Read(p []byte) (int, error) {
	n, err := l.tty.Read(p[:max(1, len(p)-l.reply.Held())])
	if err != nil {
		l.once.Do(l.gone)
		return n, io.EOF
	}
	return copy(p, l.reply.Keys(p[:n])), nil
}

func (l *lockInput) Write(p []byte) (int, error) { return l.tty.Write(p) }
func (l *lockInput) Close() error                { return l.tty.Close() }
func (l *lockInput) Fd() uintptr                 { return l.tty.Fd() }
func (l *lockInput) Name() string                { return l.tty.Name() }
