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

	"github.com/charmbracelet/x/term"
	"github.com/creack/pty"
	"github.com/muesli/cancelreader"
)

// Outcome is how the program ended: the word for the board, and the
// note for the status row.
type Outcome struct {
	Word string // ERROR, or COMPLETED
	Note string // custom saver: exit 127 · sh: x: command not found
}

// NoCommand is the outcome of a profile with no command set.
func NoCommand() Outcome { return Outcome{Word: "ERROR", Note: "custom saver: no command"} }

// Failed is the outcome of a program that could not be started at all.
func Failed(err error) Outcome { return Outcome{Word: "ERROR", Note: "custom saver: " + err.Error()} }

// outcome reads the end of a program off its Wait: 0 is COMPLETED —
// it was not meant to complete, but it did, on its own; anything else
// is ERROR, with the code or the signal, and the last thing it said on
// stderr when it said anything.
func outcome(err error, last string) Outcome {
	if err == nil {
		return Outcome{Word: "COMPLETED", Note: "custom saver exited 0"}
	}
	note := "custom saver: " + err.Error()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			note = "custom saver: killed: " + ws.Signal().String()
		} else {
			note = fmt.Sprintf("custom saver: exit %d", ee.ExitCode())
		}
	}
	if last != "" {
		note += " · " + last
	}
	return Outcome{Word: "ERROR", Note: note}
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

// Redraw asks the program to paint itself again — the signal a resize
// sends, which every full-screen program answers with a full repaint —
// after the PIN prompt has had the screen.
func (p *Proxy) Redraw() { p.signal(syscall.SIGWINCH) }

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

// Canceled reports whether err is a read called off, not a key and not
// a terminal gone.
func Canceled(err error) bool { return errors.Is(err, cancelreader.ErrCanceled) }

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
