// Package saver produces what the lock screen shows: a few lines of plain
// text, nothing else (function.md §5.1). The canvas decides how they are
// drawn; a saver decides only what they say. v1 has one type, the clock.
package saver

import (
	"strings"
	"time"
)

// The four time shapes and the five date shapes (function.md §5.2). They
// are the whole vocabulary: there is no free-form format, so every line a
// saver produces is made of the 39 characters the pixel font has.
const (
	TimeHM    = "HH:MM"
	TimeHM12  = "HH:MM AM/PM"
	TimeHMS   = "HH:MM:SS"
	TimeHMS12 = "HH:MM:SS AM/PM"

	DateOff   = "off"
	DateYMD   = "YYYY-MM-DD"
	DateYMonD = "YYYY-MMM-DD"
	DateMD    = "MM-DD"
	DateMonD  = "MMM-DD"

	// LayoutRow is the time on one line and the date on the next;
	// LayoutColumn breaks each at its separators — HH over MM over SS,
	// then the date's parts — so the lines are two or four characters and
	// the digits come out several times bigger (user, 2026-09-24).
	LayoutRow    = "row"
	LayoutColumn = "column"
)

// TimeFormats, DateFormats and Layouts are the options in the order the
// settings screen lists them.
var (
	TimeFormats = []string{TimeHM, TimeHM12, TimeHMS, TimeHMS12}
	DateFormats = []string{DateOff, DateYMD, DateYMonD, DateMD, DateMonD}
	Layouts     = []string{LayoutRow, LayoutColumn}
)

var timeLayout = map[string]string{
	TimeHM:    "15:04",
	TimeHM12:  "03:04 PM",
	TimeHMS:   "15:04:05",
	TimeHMS12: "03:04:05 PM",
}

var dateLayout = map[string]string{
	DateYMD:   "2006-01-02",
	DateYMonD: "2006-Jan-02",
	DateMD:    "01-02",
	DateMonD:  "Jan-02",
}

// Saver is what the canvas asks of any saver type.
type Saver interface {
	// Lines is the content at now: the time, then the date when one is
	// shown, on one line each or broken into their parts.
	Lines(now time.Time) []string
	// Next is when the lines will next change, so the lock can sleep until
	// then rather than poll (function.md §5.2 "tick").
	Next(now time.Time) time.Time
	// Degrade is the next step down when the lines do not fit
	// (function.md §5.3): the same saver with less on it, or false when
	// there is nothing left to drop.
	Degrade() (Saver, bool)
}

// Clock is the one saver type: a time in one of four shapes, a date in
// one of four or none, laid out in a row or a column.
type Clock struct {
	Time   string
	Date   string
	Layout string
}

// ValidTime, ValidDate and ValidLayout say whether s is one of the shapes.
func ValidTime(s string) bool   { _, ok := timeLayout[s]; return ok }
func ValidDate(s string) bool   { return s == DateOff || dateLayout[s] != "" }
func ValidLayout(s string) bool { return s == LayoutRow || s == LayoutColumn }

// Normalized is c with anything that is not a shape replaced by the
// default: a hand-edited file is not a reason to draw nothing.
func (c Clock) Normalized() Clock {
	if !ValidTime(c.Time) {
		c.Time = TimeHM
	}
	if !ValidDate(c.Date) {
		c.Date = DateOff
	}
	if !ValidLayout(c.Layout) {
		c.Layout = LayoutRow
	}
	return c
}

// Lines is the time, then the date when one is shown. Month names are
// upper case (JAN..DEC): the font has no lower case. In a column the
// separators go and each part is a line of its own.
func (c Clock) Lines(now time.Time) []string {
	c = c.Normalized()
	var lines []string
	add := func(s string) {
		if c.Layout == LayoutColumn {
			lines = append(lines, parts(s)...)
			return
		}
		lines = append(lines, s)
	}
	add(now.Format(timeLayout[c.Time]))
	if c.Date != DateOff {
		add(strings.ToUpper(now.Format(dateLayout[c.Date])))
	}
	return lines
}

// parts breaks "21:05:09", "09:05 PM" or "2026-SEP-24" at its separators.
func parts(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ':' || r == ' ' || r == '-' })
}

// Seconds reports whether the time shape shows seconds.
func (c Clock) Seconds() bool { return c.Time == TimeHMS || c.Time == TimeHMS12 }

// Next is the next second boundary with seconds on the clock, the next
// minute boundary without. A date changes at midnight, which is also a
// minute boundary.
func (c Clock) Next(now time.Time) time.Time {
	if c.Normalized().Seconds() {
		return now.Truncate(time.Second).Add(time.Second)
	}
	return now.Truncate(time.Minute).Add(time.Minute)
}

// Degrade drops content in the order of function.md §5.3: the year, then
// the seconds, then the date. The time is the last thing to go, and it
// never goes here — after this ladder the canvas falls back to plain text.
func (c Clock) Degrade() (Saver, bool) {
	c = c.Normalized()
	switch {
	case c.Date == DateYMD:
		c.Date = DateMD
	case c.Date == DateYMonD:
		c.Date = DateMonD
	case c.Time == TimeHMS:
		c.Time = TimeHM
	case c.Time == TimeHMS12:
		c.Time = TimeHM12
	case c.Date != DateOff:
		c.Date = DateOff
	default:
		return c, false
	}
	return c, true
}
