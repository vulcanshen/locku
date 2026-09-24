// Package saver produces what the lock screen shows, with no style of its
// own (function.md §5.1): the clock, a few lines of plain text, and the
// dino run, a bitmap in its own pixels (dino.go). The canvas decides how
// either is drawn; a saver decides only what it says.
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

	// The font is the glyphs' height: seven rows, or five (user,
	// 2026-09-24: try the short one). Both are three wide.
	FontTall  = "3x7"
	FontShort = "3x5"
)

// TimeFormats, DateFormats, Layouts, Sizes and Fonts are the options in
// the order the settings screen lists them.
var (
	TimeFormats = []string{TimeHM, TimeHMS}
	DateFormats = []string{DateOff, DateYMD, DateYMonD, DateMD, DateMonD}
	Layouts     = []string{LayoutRow, LayoutColumn}
	Sizes       = []string{SizeSmall, SizeMedium, SizeLarge}
	Fonts       = []string{FontTall, FontShort}
)

// ValidFont says whether s is one of the fonts.
func ValidFont(s string) bool { return s == FontTall || s == FontShort }

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

// Block is one group of lines the canvas lays out on its own — the time,
// or the date — with its content in order of preference. The canvas
// takes the first variant that fits at the largest size it can, every
// size tried before the next variant (user, 2026-09-24: the size goes
// before the units, and the date is the date's business, the time the
// time's — so a date that is too wide shrinks while the time keeps its
// size).
type Block struct {
	Variants [][]string
}

// Saver is what the canvas asks of any saver type.
type Saver interface {
	// Blocks is the content at now: the time first, then the date when
	// one is shown. The first block is never dropped; a later one is,
	// when nothing of it fits in what the ones before leave.
	Blocks(now time.Time) []Block
	// Beside says how the blocks sit: one under the other, the time above
	// (false), or side by side, the time on the right and the date on the
	// left (true) — the column layout (user, 2026-09-24).
	Beside() bool
	// Lines is the whole content at now, one line each or broken into
	// parts: what the fonts have to be able to draw.
	Lines(now time.Time) []string
	// Next is when the lines will next change, so the lock can sleep until
	// then rather than poll (function.md §5.2 "tick").
	Next(now time.Time) time.Time
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

// lines is one string as the layout shows it: on one line, or in a column
// with the separators gone and each part a line of its own.
func (c Clock) lines(s string) []string {
	if c.Layout == LayoutColumn {
		return parts(s)
	}
	return []string{s}
}

// shortDate is the date shape without its year, or the shape itself.
func shortDate(d string) string {
	switch d {
	case DateYMD:
		return DateMD
	case DateYMonD:
		return DateMonD
	}
	return d
}

// Blocks is the time — with its seconds, then without — and, when one is
// shown, the date — with its year, then without. Month names are upper
// case (JAN..DEC): the font has no lower case.
func (c Clock) Blocks(now time.Time) []Block {
	c = c.Normalized()
	t := Block{Variants: [][]string{c.lines(now.Format(timeLayout[c.Time]))}}
	if c.Seconds() {
		t.Variants = append(t.Variants, c.lines(now.Format(timeLayout[TimeHM])))
	}
	out := []Block{t}
	if c.Date != DateOff {
		d := Block{Variants: [][]string{c.lines(strings.ToUpper(now.Format(dateLayout[c.Date])))}}
		if short := shortDate(c.Date); short != c.Date {
			d.Variants = append(d.Variants, c.lines(strings.ToUpper(now.Format(dateLayout[short]))))
		}
		out = append(out, d)
	}
	return out
}

// Beside is true in the column layout: the time's parts stacked in one
// column on the right, the date's in another on the left.
func (c Clock) Beside() bool { return c.Normalized().Layout == LayoutColumn }

// Lines is the whole content: every block's first variant.
func (c Clock) Lines(now time.Time) []string {
	var out []string
	for _, b := range c.Blocks(now) {
		out = append(out, b.Variants[0]...)
	}
	return out
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
