package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Everything drawn into a fixed slot goes through these. A panel is a
// fixed-width box (§1.2), so a field that miscounts its own width does not
// merely look off — it pushes the right border out and breaks the frame.
//
// Rule: measure and pad PLAIN text, then apply the style. lipgloss.Width knows
// how to skip ANSI, but padding a styled string means the pad lands inside the
// styled span and picks up its background.

// dispW is the terminal cell width of s.
func dispW(s string) int { return lipgloss.Width(s) }

// truncate clips s to at most w cells, marking the cut with a single-cell "…".
// w <= 0 yields "". A string that already fits is returned untouched.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if dispW(s) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := dispW(string(r))
		if used+rw > w-1 { // leave one cell for the ellipsis
			break
		}
		b.WriteRune(r)
		used += rw
	}
	return b.String() + strings.Repeat(" ", w-1-used) + "…"
}

// truncateHead cuts from the FRONT, keeping the tail: where the cursor is.
func truncateHead(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if dispW(s) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	r := []rune(s)
	used, i := 0, len(r)
	for ; i > 0; i-- {
		rw := dispW(string(r[i-1]))
		if used+rw > w-1 { // leave one cell for the ellipsis
			break
		}
		used += rw
	}
	return "…" + strings.Repeat(" ", w-1-used) + string(r[i:])
}

// clipANSI cuts a possibly-styled string to w cells without severing an escape
// sequence. truncate() is for plain text; using it on styled output would cut
// mid-ANSI and bleed the style into everything after it.
func clipANSI(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if dispW(s) <= w {
		return s
	}
	return ansi.Truncate(s, w, "")
}

// padRight fits s into exactly w cells, truncating or right-padding as needed.
func padRight(s string, w int) string {
	s = truncate(s, w)
	return s + strings.Repeat(" ", max(0, w-dispW(s)))
}

// padLeft fits s into exactly w cells, right-aligned.
func padLeft(s string, w int) string {
	s = truncate(s, w)
	return strings.Repeat(" ", max(0, w-dispW(s))) + s
}

// spaces is n blanks.
func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

// fitLines forces body to exactly h lines, each padded to w cells — the
// invariant panelChrome relies on to keep its right border straight.
func fitLines(body []string, w, h int) []string {
	if len(body) > h {
		body = body[:max(0, h)]
	}
	out := make([]string, 0, max(0, h))
	for _, l := range body {
		out = append(out, l+strings.Repeat(" ", max(0, w-dispW(l))))
	}
	for len(out) < h {
		out = append(out, strings.Repeat(" ", max(0, w)))
	}
	return out
}

func clamp(v, lo, hi int) int { return min(max(v, lo), hi) }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// wrap breaks s into lines no wider than w, at spaces; a word wider than
// w is cut. Nothing comes back for an empty s or a w under one.
func wrap(s string, w int) []string {
	if w < 1 {
		return nil
	}
	var out []string
	line := ""
	for _, word := range strings.Fields(s) {
		for dispW(word) > w {
			if line != "" {
				out = append(out, line)
				line = ""
			}
			r := []rune(word)
			out = append(out, string(r[:w]))
			word = string(r[w:])
		}
		switch {
		case line == "":
			line = word
		case dispW(line)+1+dispW(word) <= w:
			line += " " + word
		default:
			out = append(out, line)
			line = word
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}
