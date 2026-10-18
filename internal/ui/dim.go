package ui

import (
	"fmt"
	"strings"

	terminu "github.com/vulcanshen/terminu-go"
)

// Dimming is fading what was drawn, not drawing it again (tdp F8,
// components/color): every foreground and background colour in the SGR
// sequences of an already-styled screen goes toward the base, never
// lighter, and the text, the layout and bold all stay. It is the lib's
// (until 2026-10-18 locku's own, filu's reference copied): the same
// formula, and the sixteen colours the terminal itself reports, asked when
// the lock starts (DetectIconWidth), where locku's took xterm's.

// dimANSI fades a drawn screen, every line of it.
func dimANSI(s string) string { return strings.Join(terminu.DimLines(strings.Split(s, "\n")), "\n") }

// hexRGB is a #rrggbb colour's three channels.
func hexRGB(hex string) [3]int {
	var c [3]int
	fmt.Sscanf(hex, "#%02x%02x%02x", &c[0], &c[1], &c[2])
	return c
}
