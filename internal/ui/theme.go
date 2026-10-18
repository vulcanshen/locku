package ui

import "github.com/charmbracelet/lipgloss"

// The lock's colours are catppuccin-mocha, like the rest of the family
// (ui.md §4); the PIN box's are the lib's (components/color).
//
// The board's two colours are NOT here: they are the user's data
// (Settings › style, config.Style), read at draw time. Their defaults —
// surface0 and the splash gold — are config's business.
var (
	dimColor = lipgloss.Color("#6c7086") // overlay0: the status row
	// a thing that is on: a custom saver's EXIT 0.
	liveColor = lipgloss.Color("#a6e3a1") // green
	// red is "wrong", yellow is "not set yet".
	warnColor   = lipgloss.Color("#f38ba8") // red
	yellowColor = lipgloss.Color("#f9e2af") // yellow
	// a custom saver's program that ended with a code other than 0: the
	// code on the board (user, 2026-09-25: red is NONE's, nothing ran).
	peachColor = lipgloss.Color("#fab387") // peach
)

// Nerd Font glyphs, as code points (tdp E4). Every one was read out of the
// installed font's cmap (JetBrainsMono Nerd Font Mono, 2026-09-24).
var (
	// pixelGlyph is one cell of the board: nf-fa-square, the same glyph the
	// family's splashes are drawn with.
	pixelGlyph = string(rune(0xf0c8)) // nf-fa-square
	glyphLock  = string(rune(0xf023)) // nf-fa-lock: the PIN box's title
)
