package ui

import terminu "github.com/vulcanshen/terminu-go"

// DetectIconWidth asks the terminal, before the lock takes it, how many
// cells a Nerd Font icon and a powerline symbol take, and what its sixteen
// colours are, for the dimming (tdp E5, components/color) — the lib does
// it: LOCKU__ICON_WIDTH / LOCKU__POWERLINE_WIDTH first, then the family's
// TERMINU__ ones, then the terminal itself (until 2026-10-18, locku's own
// probe, the icon alone). Call it once, before the program starts; on a
// lock, before termreply.DropPending, which takes an answer that came too
// late.
func DetectIconWidth() { terminu.Probe("locku") }
