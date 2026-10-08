package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// A one-line value — a name, a number, a path, a key, a command, a PIN —
// and what comes into it, typed or pasted (terminu, 2026-10-06; to be
// tdp's components/input). A line break or a tab stays in the value as
// it came, and is drawn as a red `\n` or `\t`: two cells, never cut, told
// apart from a `\` and an `n` typed. Any other control character is
// dropped. A value holding a line break or a tab is not taken: its Enter
// says why in the error row.
//
// Until then the settings screen took a pasted line break as it came,
// and drew it, breaking the box's row; a new PIN pasted as `12\n34` was
// saved, and the lock's prompt, which took printable characters only,
// could never be given it. The prompt dropped it: a pasted `x\ny` was
// `xy`, one character short, and nothing said so (user: show it as `\n`
// or `\t`, plainly).

// takeText is what of rs goes into a one-line value: a line break or a
// tab as it is, any other control character — the rest of C0, DEL, C1 —
// dropped.
func takeText(rs []rune) string {
	var b strings.Builder
	for _, r := range rs {
		if r == '\n' || r == '\r' || r == '\t' || !(r < 0x20 || r >= 0x7f && r <= 0x9f) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// valueUnits is v as it is drawn, counted and deleted: a character each,
// and "\r\n" one line break.
func valueUnits(v string) []string {
	var us []string
	r := []rune(v)
	for i := 0; i < len(r); i++ {
		if r[i] == '\r' && i+1 < len(r) && r[i+1] == '\n' {
			us = append(us, "\r\n")
			i++
			continue
		}
		us = append(us, string(r[i]))
	}
	return us
}

// valueLen is how many units v holds: a PIN's dots.
func valueLen(v string) int { return len(valueUnits(v)) }

// dropLast is v less its last unit: Backspace.
func dropLast(v string) string {
	us := valueUnits(v)
	if len(us) == 0 {
		return v
	}
	return strings.Join(us[:len(us)-1], "")
}

// hasBreak reports whether v holds a line break or a tab.
func hasBreak(v string) bool { return strings.ContainsAny(v, "\r\n\t") }

// breakErr is the error row's reason for such a value; field is what the
// box takes.
func breakErr(field string) string { return field + " can't have line breaks or tabs" }

// pinBreakErr is a PIN box's: the box is too narrow for breakErr's 34
// columns (pinInnerW), and its title says PIN already (user, 2026-10-08;
// breakErr("PIN") until then).
const pinBreakErr = "no line breaks or tabs"

// unitShown is how a unit is drawn, and whether it is a control one.
func unitShown(u string) (string, bool) {
	switch u {
	case "\r\n", "\n", "\r":
		return `\n`, true
	case "\t":
		return `\t`, true
	}
	return u, false
}

// valueView draws v in at most w cells, styled: the value lavender, a
// line break or a tab red. What does not fit goes from the FRONT, a `…`
// in its place, keeping the tail, where the cursor is; a `\n` goes whole
// or not at all. It returns the width drawn.
func valueView(v string, w int) (string, int) {
	if w <= 0 {
		return "", 0
	}
	us := valueUnits(v)
	total := 0
	for _, u := range us {
		s, _ := unitShown(u)
		total += dispW(s)
	}
	edit := lipgloss.NewStyle().Foreground(editColor)
	warn := lipgloss.NewStyle().Foreground(warnColor)
	var b, run strings.Builder
	flush := func() {
		if run.Len() > 0 {
			b.WriteString(edit.Render(run.String()))
			run.Reset()
		}
	}
	from, used := 0, total
	if total > w {
		// One cell for the `…`; whole units from the end in the rest.
		used = 0
		for from = len(us); from > 0; from-- {
			s, _ := unitShown(us[from-1])
			if used+dispW(s) > w-1 {
				break
			}
			used += dispW(s)
		}
		run.WriteString("…" + spaces(w-1-used))
		used = w
	}
	for _, u := range us[from:] {
		s, ctl := unitShown(u)
		if ctl {
			flush()
			b.WriteString(warn.Render(s))
			continue
		}
		run.WriteString(s)
	}
	flush()
	return b.String(), used
}
