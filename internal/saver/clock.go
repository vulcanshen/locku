// Package saver produces what the lock screen shows: a few lines of plain
// text, nothing else (function.md §5.1). The canvas decides how they are
// drawn; a saver decides only what they say. v1 has one type, the clock.
package saver

import (
	"strings"
	"time"
)

// The two time shapes and the five date shapes (function.md §5.2). They
// are the whole vocabulary: there is no free-form format, so every line a
// saver produces is made of the characters the pixel font has.
//
// The time has no colon: its groups are parted by a space, which on the
// board is a wider gap than the one between two digits, and that is
// grouping enough (user, 2026-09-24). The twelve-hour shapes went the
// same day: with the letters drawn as a seven-segment display draws them,
// AM and PM were not worth their pixels.
const (
	TimeHM  = "HH MM"
	TimeHMS = "HH MM SS"

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

	// The size is how many board cells one font pixel takes on a side:
	// small is one, medium two by two, large three by three (user,
	// 2026-09-24). It is the scale the canvas asks for; what does not fit
	// steps down (canvas.go fit).
	SizeSmall  = "small"
	SizeMedium = "medium"
	SizeLarge  = "large"
)

// TimeFormats, DateFormats, Layouts and Sizes are the options in the
// order the settings screen lists them.
var (
	TimeFormats = []string{TimeHM, TimeHMS}
	DateFormats = []string{DateOff, DateYMD, DateYMonD, DateMD, DateMonD}
	Layouts     = []string{LayoutRow, LayoutColumn}
	Sizes       = []string{SizeSmall, SizeMedium, SizeLarge}
)

var timeLayout = map[string]string{
	TimeHM:  "15 04",
	TimeHMS: "15 04 05",
}

// oldTime maps the shapes a file may still say — with colons, or in
// twelve hours — onto the two there are.
var oldTime = map[string]string{
	"HH:MM":          TimeHM,
	"HH:MM:SS":       TimeHMS,
	"HH:MM AM/PM":    TimeHM,
	"HH:MM:SS AM/PM": TimeHMS,
}

var dateLayout = map[string]string{
	DateYMD:   "2006-01-02",
	DateYMonD: "2006-Jan-02",
	DateMD:    "01-02",
	DateMonD:  "Jan-02",
}

// Scale is the whole factor a size names; anything else is medium.
func Scale(size string) int {
	switch size {
	case SizeSmall:
		return 1
	case SizeLarge:
		return 3
	}
	return 2
}

// Saver is what the canvas asks of any saver type.
type Saver interface {
	// Lines is the content at now: the time, then the date when one is
	// shown, on one line each or broken into their parts.
	Lines(now time.Time) []string
	// Next is when the lines will next change, so the lock can sleep until
	// then rather than poll (function.md §5.2 "tick").
	Next(now time.Time) time.Time
	// Steps is the content in order of preference, the whole of it first,
	// then with less and less on it (function.md §5.3): the canvas draws
	// the first step that fits. It is a list and not a chain, because
	// what does not fit may be the height as well as the width, and a
	// chain that drops the seconds on its way to dropping the date would
	// lose them even when the date alone was the problem (user,
	// 2026-09-24).
	Steps() []Saver
}

// Clock is the one saver type: a time with or without seconds, a date in
// one of four shapes or none, laid out in a row or a column.
type Clock struct {
	Time   string
	Date   string
	Layout string
}

// ValidTime, ValidDate, ValidLayout and ValidSize say whether s is one of
// the shapes.
func ValidTime(s string) bool   { _, ok := timeLayout[s]; return ok }
func ValidDate(s string) bool   { return s == DateOff || dateLayout[s] != "" }
func ValidLayout(s string) bool { return s == LayoutRow || s == LayoutColumn }
func ValidSize(s string) bool   { return s == SizeSmall || s == SizeMedium || s == SizeLarge }

// NormalTime is the time shape a file's value means: itself, an old
// spelling's new one, or the default.
func NormalTime(s string) string {
	if ValidTime(s) {
		return s
	}
	if t, ok := oldTime[s]; ok {
		return t
	}
	return TimeHM
}

// Normalized is c with anything that is not a shape replaced by the
// default: a hand-edited file is not a reason to draw nothing.
func (c Clock) Normalized() Clock {
	c.Time = NormalTime(c.Time)
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

// parts breaks "21 05 09" or "2026-SEP-24" at its separators.
func parts(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ':' || r == ' ' || r == '-' })
}

// Seconds reports whether the time shape shows seconds.
func (c Clock) Seconds() bool { return NormalTime(c.Time) == TimeHMS }

// Next is the next second boundary with seconds on the clock, the next
// minute boundary without. A date changes at midnight, which is also a
// minute boundary.
func (c Clock) Next(now time.Time) time.Time {
	if c.Seconds() {
		return now.Truncate(time.Second).Add(time.Second)
	}
	return now.Truncate(time.Minute).Add(time.Minute)
}

// Steps is the clock with less and less on it (function.md §5.3): the
// year goes first, then the seconds — the date is worth more than they
// are — then the date, at which point the seconds come back, and last
// the seconds alone. The time is the last thing to go, and it never goes
// here: after these the canvas falls back to plain text.
func (c Clock) Steps() []Saver {
	c = c.Normalized()
	short := c.Date
	switch c.Date {
	case DateYMD:
		short = DateMD
	case DateYMonD:
		short = DateMonD
	}
	var out []Saver
	add := func(t, d string) { out = append(out, Clock{Time: t, Date: d, Layout: c.Layout}) }
	add(c.Time, c.Date) // everything
	if short != c.Date {
		add(c.Time, short) // the year goes
	}
	if c.Seconds() && c.Date != DateOff {
		add(TimeHM, short) // the seconds go, the date stays
	}
	if c.Date != DateOff {
		add(c.Time, DateOff) // the date goes, the seconds are back
	}
	if c.Seconds() {
		add(TimeHM, DateOff) // the seconds go too
	}
	return out
}
