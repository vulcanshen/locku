package termreply

import "os"

// DropPending throws away what is waiting to be read on tty, before the
// lock reads it: the lock counts the keys pressed once it is up. What is
// waiting then is keys pressed before, or what is left of the terminal's
// answers — and a piece of one gets past a Filter, which knows answers by
// their whole shape. Bubble Tea asks the terminal its background colour
// as the program starts, before main, and the reader it asks through
// (termenv) takes an answer's ESC \ to end at the ESC: when an answer to
// tmux's attach came in first, two answers are read as one each and the
// last \ is left, a key (measured 2026-09-28: `\` then the cursor
// report, and the lock unlocked).
func DropPending(tty *os.File) error { return dropPending(int(tty.Fd())) }
