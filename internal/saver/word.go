package saver

import (
	"strconv"
	"time"
)

// KindCustom is the third kind of saver: a program of the user's own,
// run on a terminal of its own, as the saver (user, 2026-09-25: they
// can write the animation themselves and let locku manage the rest —
// the lock, the PIN, the integration). What it draws is its business;
// the canvas never draws it. What the canvas draws FOR it is a Word: how
// the program ended, when it does.
const KindCustom = "custom"

// Word is what the board says of a custom saver's program that has
// ended, in the clock's own face and sizes: EXIT and its code, as a
// shell would report it, or NONE when nothing ran (user, 2026-09-25: an
// ending is shown in locku's own format, the word stepped large to
// small as the terminal allows, the reason on the status row; the
// program is not restarted — it was meant to run for ever, and its
// ending is what the user has to know about. The code itself is the
// truest word for it: COMPLETED / ERROR, then DONE / ERROR, were tried
// the same day and dropped). It never changes.
type Word string

// WordNone is the board's word when no program ran: none set, or one
// that could not be started.
const WordNone Word = "NONE"

// ExitWord is the board's word for a program that ended with code — a
// signal counted as a shell counts it, 128 and the signal's number.
func ExitWord(code int) Word { return Word("EXIT " + strconv.Itoa(code)) }

// Fine reports whether w is the one ending a program means to have:
// EXIT 0, its code green on the board; another code is peach, and NONE
// red.
func (w Word) Fine() bool { return w == ExitWord(0) }

func (w Word) Blocks(time.Time) []Block { return []Block{{Variants: [][]string{{string(w)}}}} }
func (w Word) Beside() bool             { return false }
func (w Word) Lines(time.Time) []string { return []string{string(w)} }

// Next is never: a day on, so the lock has something to sleep until.
func (w Word) Next(now time.Time) time.Time { return now.Add(24 * time.Hour) }
