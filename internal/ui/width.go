package ui

import (
	"strconv"
	"strings"

	terminu "github.com/vulcanshen/terminu-go"
)

// One width for everything the lock draws (tdp E5): the lib's. How many
// cells an icon and a powerline symbol take is measured when the lock
// starts (terminu.Probe, in DetectIconWidth) or, for a preview, by the
// settings screen's lib as it started; every width here — padding, cuts,
// centring, the box over the board — goes through terminu.Width (until
// 2026-10-18, locku's own copy of filu's width.go).

// IconCells is how many cells a Nerd Font icon takes: what a custom saver's
// program is told (tdp E5).
func IconCells() int { return terminu.IconWidth() }

// dispW is the terminal cell width of s.
func dispW(s string) int { return terminu.Width(s) }

// truncate clips s to at most w cells, marking the cut with "…".
func truncate(s string, w int) string { return terminu.Truncate(s, w) }

// clipANSI cuts a styled line to at most w cells, its styles kept.
func clipANSI(s string, w int) string {
	if dispW(s) <= w {
		return s
	}
	return terminu.Cut(s, 0, max(w, 0))
}

// blockWidth is the widest of lines.
func blockWidth(lines []string) int {
	w := 0
	for _, l := range lines {
		w = max(w, dispW(l))
	}
	return w
}

// spaces is n spaces, none for n <= 0.
func spaces(n int) string { return strings.Repeat(" ", max(n, 0)) }

func itoa(n int) string { return strconv.Itoa(n) }
