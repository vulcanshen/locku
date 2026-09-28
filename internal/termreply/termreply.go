// Package termreply takes a terminal's answers out of what the lock reads
// from it, so that only keys are left (function.md §2.1). A terminal
// answers what it is asked — its background colour, where the cursor is,
// what it is — on the same input the keys come in by, and the lock reads
// any key as a key: an answer that reaches it would open the PIN prompt,
// or, with no PIN, unlock. They do reach it: tmux asks the terminal a
// few things the moment a client attaches, and the hook that locks a
// client attaching to a locked tmux runs just after, so the answers come
// back to the lock (measured 2026-09-28, tmux 3.7c: the lock gone in
// 17 ms, every time).
//
// An answer is told from a key by its shape, which no key has:
//
//   - OSC, ESC ] and a digit, to BEL or ESC \ — a colour, say.
//   - DCS, ESC P and a digit, > or !, to ESC \ — the terminal's name.
//   - CSI with a private marker, ESC [ and ? > or =, or ending in R, n,
//     t, c, or $y — the cursor's position, the device attributes, a
//     mode's state. Shift-F3 is ESC [ 1 ; 2 R in some terminals, and is
//     dropped with the cursor reports; another key does what it would.
//
// Everything else is a key: a character, Enter, Backspace, Esc on its
// own, the arrows and the function keys, Alt with a key — Alt-] and
// Alt-P too, when what follows is not the rest of an answer.
//
// Esc at the end of a read goes out at once: it is a key people press,
// and it must not wait for the next one. ESC ] and ESC P at the end of a
// read wait for the next: an answer cut there would otherwise get in,
// and an answer let in unlocks, where Alt-] or Alt-P kept back only
// waits for the next key — nobody presses them on a lock.
package termreply

import "bytes"

// A Filter reads a terminal's input a read at a time and keeps what is
// between reads: an answer split over two reads is dropped whole, and a
// key sequence split over two comes out whole with the second.
type Filter struct {
	state state
	seq   []byte // an escape sequence not yet known to be a key or an answer
	n     int    // bytes of the answer being dropped
}

type state int

const (
	ground  state = iota
	escape        // ESC
	intro         // ESC ] or ESC P: an answer if what follows says so
	csi           // ESC [ and what has come of it
	answer        // inside an OSC or DCS answer
	answerE       // ESC inside it: the first half of ESC \
)

const (
	esc = 0x1b
	bel = 0x07
	// maxHeld is the most a Filter holds back between reads: a CSI longer
	// than this is let through as it is.
	maxHeld = 32
	// maxAnswer is the most dropped as one answer: past it, the terminal
	// is taken to have said something else, and the bytes are keys again.
	maxAnswer = 4096
)

// Held is how many bytes the Filter holds back from earlier reads, which
// the next call to Keys can give out on top of its own.
func (f *Filter) Held() int { return len(f.seq) }

// Keys is what of b is keys, with what an earlier read left pending in
// front when it turns out to be a key.
func (f *Filter) Keys(b []byte) []byte {
	var out []byte
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch f.state {
		case ground:
			if c == esc {
				f.state, f.seq = escape, append(f.seq[:0], c)
				continue
			}
			out = append(out, c)
		case escape:
			switch c {
			case '[':
				f.state, f.seq = csi, append(f.seq, c)
			case ']', 'P':
				f.state, f.seq = intro, append(f.seq, c)
			case esc:
				out = append(out, esc) // Esc, then maybe another sequence
			default:
				out = append(out, esc, c)
				f.state, f.seq = ground, f.seq[:0]
			}
		case intro:
			if isDigit(c) || (f.seq[1] == 'P' && (c == '>' || c == '!')) {
				f.state, f.seq, f.n = answer, f.seq[:0], 0
				continue
			}
			out = append(out, f.seq...) // Alt-] or Alt-P
			f.state, f.seq = ground, f.seq[:0]
			i-- // c again, from the ground
		case csi:
			switch {
			case c >= 0x40 && c <= 0x7e: // the final byte
				f.seq = append(f.seq, c)
				if !isAnswer(f.seq) {
					out = append(out, f.seq...)
				}
				f.state, f.seq = ground, f.seq[:0]
			case c >= 0x20 && c <= 0x3f && len(f.seq) < maxHeld:
				f.seq = append(f.seq, c)
			default: // not a CSI after all
				out = append(out, f.seq...)
				f.state, f.seq = ground, f.seq[:0]
				i--
			}
		case answer:
			f.n++
			switch {
			case c == bel:
				f.state = ground
			case c == esc:
				f.state = answerE
			case c < 0x20 || f.n > maxAnswer: // no answer has these: a key
				f.state = ground
				i--
			}
		case answerE:
			if c == '\\' { // ESC \, the end
				f.state = ground
				continue
			}
			f.state, f.seq = escape, append(f.seq[:0], esc) // cut short; ESC begins again
			i--
		}
	}
	// Esc goes out now; ESC ] or ESC P, a CSI begun, and an answer being
	// dropped carry on into the next read.
	if f.state == escape {
		out = append(out, f.seq...)
		f.state, f.seq = ground, f.seq[:0]
	}
	return out
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// isAnswer reports whether a whole CSI sequence is a terminal's answer.
func isAnswer(seq []byte) bool {
	if len(seq) > 2 && (seq[2] == '?' || seq[2] == '>' || seq[2] == '=') {
		return true
	}
	switch final := seq[len(seq)-1]; final {
	case 'R', 'n', 't', 'c':
		return true
	case 'y':
		return bytes.IndexByte(seq[2:len(seq)-1], '$') >= 0
	}
	return false
}
