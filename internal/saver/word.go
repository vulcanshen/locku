package saver

import "time"

// KindCustom is the third kind of saver: a program of the user's own,
// run on a terminal of its own, as the saver (user, 2026-09-25: they
// can write the animation themselves and let locku manage the rest —
// the lock, the PIN, the integration). What it draws is its business;
// the canvas never draws it. What the canvas draws FOR it is a Word: how
// the program ended, when it does.
const KindCustom = "custom"

// Word is one word on the board, in the clock's own face and sizes —
// ERROR, COMPLETED — for a custom saver's program that has ended (user,
// 2026-09-25: an ending is shown in locku's own format, the word stepped
// large to small as the terminal allows, the reason on the status row;
// the program is not restarted — it was meant to run for ever, and its
// ending is what the user has to know about). It never changes.
type Word string

// The two words: a program that ended on its own with 0, which it was
// not meant to do, and one that failed, could not be run, or was killed.
const (
	WordCompleted Word = "COMPLETED"
	WordError     Word = "ERROR"
)

func (w Word) Blocks(time.Time) []Block { return []Block{{Variants: [][]string{{string(w)}}}} }
func (w Word) Beside() bool             { return false }
func (w Word) Lines(time.Time) []string { return []string{string(w)} }

// Next is never: a day on, so the lock has something to sleep until.
func (w Word) Next(now time.Time) time.Time { return now.Add(24 * time.Hour) }
