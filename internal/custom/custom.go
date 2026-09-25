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
	"bytes"
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

// Proxy is the running program: its pty, and the pump that passes its
// output on to the screen as it comes.
type Proxy struct {
	cmd      *exec.Cmd
	ptmx     *os.File
	out      io.Writer
	mu       sync.Mutex
	last     time.Time // when the program last wrote anything
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
	p := &Proxy{cmd: cmd, ptmx: ptmx, out: out, last: time.Now(),
		tailDone: make(chan struct{}), done: make(chan Outcome, 1), finished: make(chan struct{})}
	go func() {
		defer close(p.tailDone)
		p.tail.read(pr)
	}()
	go p.pump()
	return p, nil
}

// pump passes the program's output on, every byte, as it comes; when
// the pty ends — the program has gone — the program is reaped and its
// outcome delivered.
func (p *Proxy) pump() {
	buf := make([]byte, 32*1024)
	for {
		n, err := p.ptmx.Read(buf)
		if n > 0 {
			p.out.Write(buf[:n])
			p.mu.Lock()
			p.last = time.Now()
			p.mu.Unlock()
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

// Idle reports whether the program has drawn nothing for d: a program
// that sits still, which will not fill the prompt's place by itself.
func (p *Proxy) Idle(d time.Duration) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return time.Since(p.last) > d
}

// Resize tells the pty its new size, which tells the program, as any
// terminal would.
func (p *Proxy) Resize(cols, rows int) {
	pty.Setsize(p.ptmx, &pty.Winsize{Rows: uint16(max(1, rows)), Cols: uint16(max(1, cols))})
}

// Redraw asks the program to paint itself again — the signal a resize
// sends, which a curses program answers with a whole repaint — for a
// program that sits still after the prompt's place was blanked. One
// that draws is not asked: nothing of what it drew was dropped, it
// fills the place by itself, and a curses program asked starts its
// picture over (measured 2026-09-25, cmatrix: a flash and the rain
// from the top).
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

// screen is the program's way to the terminal, and the prompt's. The
// program's output goes through as it comes; while the prompt's box is
// up, every chunk is followed by the box again — the cursor and its
// attributes saved and restored around it, so the program never notices
// — inside one synchronised update, so the terminal shows the frame
// and the box as one (user, 2026-09-25: the picture goes on under the
// prompt, as the board does under it on the other savers; and nothing
// the program draws is dropped, so the screen never falls out of step
// with it). Clear blanks the box's place; what the program draws next
// fills it.
type screen struct {
	mu         sync.Mutex
	out        io.Writer
	cols, rows int
	box        string
	rect       struct{ top, left, w, h int }
}

const (
	syncBegin  = "\x1b[?2026h" // a synchronised update: what follows shows as one
	syncEnd    = "\x1b[?2026l"
	saveCur    = "\x1b7" // DECSC: the cursor, its attributes
	restoreCur = "\x1b8" // DECRC
)

func (s *screen) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.box == "" {
		return s.out.Write(p)
	}
	var b bytes.Buffer
	b.WriteString(syncBegin)
	b.Write(p)
	b.WriteString(saveCur)
	s.paint(&b)
	b.WriteString(restoreCur)
	b.WriteString(syncEnd)
	if _, err := s.out.Write(b.Bytes()); err != nil {
		return 0, err
	}
	return len(p), nil
}

// paint writes the box into w and widens the rectangle it has taken.
func (s *screen) paint(w io.Writer) {
	top, left, wd, h := paintBox(w, s.cols, s.rows, s.box)
	if h == 0 {
		return
	}
	if s.rect.h == 0 {
		s.rect.top, s.rect.left, s.rect.w, s.rect.h = top, left, wd, h
		return
	}
	right, bottom := max(s.rect.left+s.rect.w, left+wd), max(s.rect.top+s.rect.h, top+h)
	s.rect.top, s.rect.left = min(s.rect.top, top), min(s.rect.left, left)
	s.rect.w, s.rect.h = right-s.rect.left, bottom-s.rect.top
}

// Overlay puts box — the PIN prompt's lines — over the middle of the
// screen, and keeps it there over whatever the program draws.
func (s *screen) Overlay(box string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.box = box
	var b bytes.Buffer
	b.WriteString(syncBegin + saveCur)
	s.paint(&b)
	b.WriteString(restoreCur + syncEnd)
	s.out.Write(b.Bytes())
}

// Clear takes the box away and blanks its place, the largest it was.
func (s *screen) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.box = ""
	if s.rect.h > 0 {
		var b bytes.Buffer
		b.WriteString(syncBegin + saveCur)
		clearBox(&b, s.rect.top, s.rect.left, s.rect.w, s.rect.h)
		b.WriteString(restoreCur + syncEnd)
		s.out.Write(b.Bytes())
	}
	s.rect.top, s.rect.left, s.rect.w, s.rect.h = 0, 0, 0, 0
}

// Resized tells the screen its new size, for where the box goes.
func (s *screen) Resized(cols, rows int) {
	s.mu.Lock()
	s.cols, s.rows = cols, rows
	s.mu.Unlock()
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

// Terminal is the real terminal while a program has the screen: raw, so
// every key is locku's and none is a signal; on the alternate screen,
// so the user's own screen is there again at the end; its keys read by
// one reader that can be called off, so a lock program can have the
// terminal next without a read of ours in the way; and a screen the
// program's output and the prompt's box share.
type Terminal struct {
	tty   *os.File
	out   *os.File
	state *term.State
	keys  cancelreader.CancelReader
	scr   *screen
}

// Take puts the terminal into locku's hands: raw, on the alternate
// screen, cleared, the cursor hidden.
func Take(tty, out *os.File) (*Terminal, error) {
	state, err := term.MakeRaw(tty.Fd())
	if err != nil {
		return nil, err
	}
	t := &Terminal{tty: tty, out: out, state: state}
	cols, rows := t.Size()
	t.scr = &screen{out: out, cols: cols, rows: rows}
	t.out.WriteString("\x1b[?1049h\x1b[?25l\x1b[2J\x1b[H")
	return t, nil
}

// Give hands the terminal back as it was: the main screen, the cursor,
// the attributes, and the modes it had.
func (t *Terminal) Give() {
	t.out.WriteString("\x1b[0m\x1b[2J\x1b[H\x1b[?25h\x1b[?1049l")
	term.Restore(t.tty.Fd(), t.state)
}

// Writer is where the program's output goes: through the screen.
func (t *Terminal) Writer() io.Writer { return t.scr }

// Overlay puts the PIN prompt's box over the program's picture, and
// keeps it there; Clear takes it away; Resized follows the terminal.
func (t *Terminal) Overlay(box string)     { t.scr.Overlay(box) }
func (t *Terminal) Clear()                 { t.scr.Clear() }
func (t *Terminal) Resized(cols, rows int) { t.scr.Resized(cols, rows) }

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
	p, err := Start(command, t.Writer(), cols, rows)
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
