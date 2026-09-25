// Package custom runs the user's own program as the saver (function.md
// §5.6; user, 2026-09-25): a command of theirs, on a pty of locku's, its
// output passed on to the terminal as it comes. The program is a process
// of its own and nothing more: locku does not restart it, does not read
// its screen, and when the lock ends kills it — its whole process group,
// at once. Keys never reach it: the terminal's input is locku's, for the
// PIN. The one thing it is told is its size, as any terminal tells any
// program, and the one thing it is asked is to paint itself again, by
// the signal a resize sends, when the PIN prompt has had the screen. A
// program that ends is not restarted: the lock goes on, with the word
// for what happened on the board (Outcome).
//
// Nothing here may bring the lock down: a program that cannot even be
// started is an outcome like any other.
package custom

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/creack/pty"
	"github.com/muesli/cancelreader"

	"github.com/vulcanshen/locku/internal/saver"
)

// Outcome is how the program ended: the word for the board, and the
// note for the status row.
type Outcome struct {
	Word saver.Word // EXIT 3, EXIT 139, or NONE
	Note string     // custom saver: exit 127 · sh: x: command not found
}

// NoCommand is the outcome of a profile with no command set.
func NoCommand() Outcome { return Outcome{Word: saver.WordNone, Note: "custom saver: no command"} }

// Failed is the outcome of a program that could not be started at all.
func Failed(err error) Outcome {
	return Outcome{Word: saver.WordNone, Note: "custom saver: " + err.Error()}
}

// outcome reads the end of a program off its Wait: the code, as a shell
// would report it — 0 for an ending on its own, which it was not meant
// to have; 128 and the signal's number for a program killed — and the
// last thing it said on stderr, when it said anything (user,
// 2026-09-25: the code itself, not a word for it).
func outcome(err error, last string) Outcome {
	if err == nil {
		return Outcome{Word: saver.ExitWord(0), Note: "custom saver exited 0"}
	}
	o := Outcome{Word: saver.WordNone, Note: "custom saver: " + err.Error()}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			o = Outcome{Word: saver.ExitWord(128 + int(ws.Signal())), Note: "custom saver: killed: " + ws.Signal().String()}
		} else {
			o = Outcome{Word: saver.ExitWord(ee.ExitCode()), Note: fmt.Sprintf("custom saver: exit %d", ee.ExitCode())}
		}
	}
	if last != "" {
		o.Note += " · " + last
	}
	return o
}

// Proxy is the running program: its pty, and the tap that passes its
// output on, or holds it back while the PIN prompt has the screen.
type Proxy struct {
	cmd      *exec.Cmd
	ptmx     *os.File
	out      io.Writer
	mu       sync.Mutex
	on       bool
	tail     tail
	tailDone chan struct{}
	done     chan Outcome  // the outcome, once
	finished chan struct{} // closed once the program is reaped
}

// Start runs command through sh -c on a new pty, cols × rows, passing
// its output on to out from the first byte. The shell's stderr — and the
// program's — comes through a pipe of its own, so what was said on the
// way down can be read back (sh: x: command not found), while stdout
// keeps the pty: a terminal, to the program.
func Start(command string, out io.Writer, cols, rows int) (*Proxy, error) {
	if strings.TrimSpace(command) == "" {
		return nil, errors.New("no command")
	}
	cmd := exec.Command("sh", "-c", command)
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = pw
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(max(1, rows)), Cols: uint16(max(1, cols))})
	pw.Close()
	if err != nil {
		pr.Close()
		return nil, err
	}
	p := &Proxy{cmd: cmd, ptmx: ptmx, out: out, on: true,
		tailDone: make(chan struct{}), done: make(chan Outcome, 1), finished: make(chan struct{})}
	go func() {
		defer close(p.tailDone)
		p.tail.read(pr)
	}()
	go p.pump()
	return p, nil
}

// pump passes the program's output on while the tap is open, and drops
// it while it is not; when the pty ends — the program has gone — the
// program is reaped and its outcome delivered.
func (p *Proxy) pump() {
	buf := make([]byte, 32*1024)
	for {
		n, err := p.ptmx.Read(buf)
		if n > 0 {
			p.mu.Lock()
			on := p.on
			p.mu.Unlock()
			if on {
				p.out.Write(buf[:n])
			}
		}
		if err != nil {
			break
		}
	}
	err := p.cmd.Wait()
	// The last word on stderr may still be on its way; a grandchild
	// holding the pipe open is not waited for.
	select {
	case <-p.tailDone:
	case <-time.After(200 * time.Millisecond):
	}
	p.done <- outcome(err, p.tail.lastLine())
	close(p.finished)
}

// Done delivers the outcome once the program has ended, however it did.
func (p *Proxy) Done() <-chan Outcome { return p.done }

// Forward opens the tap, or closes it: while the PIN prompt has the
// screen the program's output is read and dropped, so it never blocks
// and never draws over the prompt.
func (p *Proxy) Forward(on bool) {
	p.mu.Lock()
	p.on = on
	p.mu.Unlock()
}

// Resize tells the pty its new size, which tells the program, as any
// terminal would.
func (p *Proxy) Resize(cols, rows int) {
	pty.Setsize(p.ptmx, &pty.Winsize{Rows: uint16(max(1, rows)), Cols: uint16(max(1, cols))})
}

// Redraw asks the program to paint itself whole, after the PIN prompt
// has had the screen: the pty is made a column narrower and then its
// size again — two real resizes, each a SIGWINCH to the program. A
// signal alone was not enough (measured 2026-09-25, cmatrix): a curses
// program told the size has not changed repaints only what it thinks
// changed, against a screen that no longer shows what it last drew —
// the frames it drew while the prompt was up were dropped — and the
// old frame shows through. A size that really changed makes it lay the
// screen out again from nothing.
func (p *Proxy) Redraw(cols, rows int) {
	p.Resize(cols-1, rows)
	time.Sleep(40 * time.Millisecond)
	p.Resize(cols, rows)
}

// Kill ends the program, its whole process group, at once, and waits
// for it to be reaped. It is the lock's end, or the program's preview's.
func (p *Proxy) Kill() {
	p.signal(syscall.SIGKILL)
	p.ptmx.Close()
	<-p.finished
}

// signal goes to the whole group: sh -c and whatever it started.
func (p *Proxy) signal(sig syscall.Signal) {
	if p.cmd.Process != nil {
		syscall.Kill(-p.cmd.Process.Pid, sig)
	}
}

// tail keeps the last line said on stderr, reading everything so the
// pipe never fills.
type tail struct {
	mu   sync.Mutex
	last string
}

func (t *tail) read(r io.ReadCloser) {
	defer r.Close()
	buf := make([]byte, 4096)
	var keep []byte
	for {
		n, err := r.Read(buf)
		if n > 0 {
			keep = append(keep, buf[:n]...)
			if len(keep) > 1024 {
				keep = keep[len(keep)-1024:]
			}
			lines := strings.Split(strings.TrimSpace(string(keep)), "\n")
			t.mu.Lock()
			t.last = strings.TrimSpace(lines[len(lines)-1])
			t.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (t *tail) lastLine() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.last
}

// Terminal is the real terminal while a program has the screen: raw, so
// every key is locku's and none is a signal; on the alternate screen,
// so the user's own screen is there again at the end; its keys read by
// one reader that can be called off, so a lock program can have the
// terminal next without a read of ours in the way.
type Terminal struct {
	tty   *os.File
	out   *os.File
	state *term.State
	keys  cancelreader.CancelReader
	// The overlay's rectangle, the largest drawn since it was last
	// cleared: what Clear has to cover.
	box struct{ top, left, w, h int }
}

// Overlay draws box — the PIN prompt's lines — over the middle of the
// screen, and nothing else: the program's picture stays around it,
// frozen (user, 2026-09-25: the prompt over the picture, not over a
// ground of locku's, and no switch of screens under it).
func (t *Terminal) Overlay(box string) {
	cols, rows := t.Size()
	top, left, w, h := paintBox(t.out, cols, rows, box)
	if t.box.h == 0 {
		t.box.top, t.box.left, t.box.w, t.box.h = top, left, w, h
		return
	}
	// The union with what was drawn before, so a box that grew and
	// shrank is cleared whole.
	right, bottom := max(t.box.left+t.box.w, left+w), max(t.box.top+t.box.h, top+h)
	t.box.top, t.box.left = min(t.box.top, top), min(t.box.left, left)
	t.box.w, t.box.h = right-t.box.left, bottom-t.box.top
}

// Clear blanks where the overlay was. What was under it is the
// program's to paint again — Redraw asks it to; a program that does not
// answer is left with a blank where the prompt was, not with the prompt.
func (t *Terminal) Clear() {
	if t.box.h > 0 {
		clearBox(t.out, t.box.top, t.box.left, t.box.w, t.box.h)
	}
	t.box.top, t.box.left, t.box.w, t.box.h = 0, 0, 0, 0
}

// paintBox writes box's lines centred on a cols × rows screen, each at
// its own position, and returns the rectangle they took. Nothing
// outside the lines is touched.
func paintBox(out io.Writer, cols, rows int, box string) (top, left, w, h int) {
	lines := strings.Split(strings.TrimRight(box, "\n"), "\n")
	if box == "" || len(lines) == 0 {
		return 0, 0, 0, 0
	}
	for _, l := range lines {
		w = max(w, ansi.StringWidth(l))
	}
	h = len(lines)
	top, left = max(0, (rows-h)/2), max(0, (cols-w)/2)
	var b strings.Builder
	for i, l := range lines {
		fmt.Fprintf(&b, "\x1b[%d;%dH%s\x1b[0m", top+i+1, left+1, l)
	}
	io.WriteString(out, b.String())
	return top, left, w, h
}

// clearBox blanks the rectangle, in the terminal's own colours.
func clearBox(out io.Writer, top, left, w, h int) {
	var b strings.Builder
	blank := strings.Repeat(" ", w)
	for i := 0; i < h; i++ {
		fmt.Fprintf(&b, "\x1b[0m\x1b[%d;%dH%s", top+i+1, left+1, blank)
	}
	io.WriteString(out, b.String())
}

// Take puts the terminal into locku's hands: raw, on the alternate
// screen, cleared, the cursor hidden.
func Take(tty, out *os.File) (*Terminal, error) {
	state, err := term.MakeRaw(tty.Fd())
	if err != nil {
		return nil, err
	}
	t := &Terminal{tty: tty, out: out, state: state}
	t.out.WriteString("\x1b[?1049h\x1b[?25l\x1b[2J\x1b[H")
	return t, nil
}

// Give hands the terminal back as it was: the main screen, the cursor,
// the attributes, and the modes it had.
func (t *Terminal) Give() {
	t.out.WriteString("\x1b[0m\x1b[2J\x1b[H\x1b[?25h\x1b[?1049l")
	term.Restore(t.tty.Fd(), t.state)
}

// Retake is Take again, after a lock program had the terminal.
func (t *Terminal) Retake() error {
	state, err := term.MakeRaw(t.tty.Fd())
	if err != nil {
		return err
	}
	t.state = state
	t.out.WriteString("\x1b[?1049h\x1b[?25l\x1b[2J\x1b[H")
	return nil
}

// Size is the terminal's, in cells.
func (t *Terminal) Size() (cols, rows int) {
	cols, rows, err := term.GetSize(t.tty.Fd())
	if err != nil || cols <= 0 || rows <= 0 {
		return 80, 24
	}
	return cols, rows
}

// Key starts one read of the terminal and delivers its end: nil for a
// key — whatever it was; a key is not input, it is the question — an
// error for a terminal that has gone, or for a read called off (Cancel).
func (t *Terminal) Key() <-chan error {
	ch := make(chan error, 1)
	cr, err := cancelreader.NewReader(t.tty)
	if err != nil {
		ch <- err
		return ch
	}
	t.keys = cr
	go func() {
		defer cr.Close()
		var b [64]byte
		_, err := cr.Read(b[:])
		ch <- err
	}()
	return ch
}

// Cancel calls the pending read off; the channel Key returned gets
// cancelreader.ErrCanceled.
func (t *Terminal) Cancel() {
	if t.keys != nil {
		t.keys.Cancel()
		t.keys = nil
	}
}

// Preview runs command on the terminal until a key, or until the
// program ends: a look at it from the settings screen (function.md
// §5.6). It returns the outcome of a program that ended, and nil for a
// key; either way the program is killed and the terminal given back.
func Preview(command string, tty, out *os.File) *Outcome {
	t, err := Take(tty, out)
	if err != nil {
		o := Failed(err)
		return &o
	}
	defer t.Give()
	cols, rows := t.Size()
	p, err := Start(command, out, cols, rows)
	if err != nil {
		o := Failed(err)
		return &o
	}
	defer p.Kill()
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	keys := t.Key()
	for {
		select {
		case <-winch:
			p.Resize(t.Size())
		case o := <-p.Done():
			t.Cancel()
			<-keys
			return &o
		case <-keys:
			return nil
		}
	}
}
