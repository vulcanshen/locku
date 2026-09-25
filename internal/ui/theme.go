package ui

import "github.com/charmbracelet/lipgloss"

// Anchors are catppuccin-mocha, like the rest of the family (ui.md §4):
// assigned once, derived everywhere. One band, one meaning (VTP §B).
//
// The board's two colours are NOT here: they are the user's data
// (Settings › style, config.Style), read at draw time. Their defaults —
// surface0 and the splash gold — are config's business.
var (
	// structural — panel chrome, and the KEY half of every legend (§4.4).
	focusColor = lipgloss.Color("#89b4fa") // blue
	borderDim  = lipgloss.Color("#585b70") // surface2: unfocused border
	// the cursor row: "the current hand".
	handColor = lipgloss.Color("#bac2de") // subtext1
	// the field under edit in an input popup.
	editColor = lipgloss.Color("#b4befe") // lavender
	// neutral text.
	textColor = lipgloss.Color("#cdd6f4") // text
	dimColor  = lipgloss.Color("#6c7086") // overlay0: hints, read-only rows, section labels
	// a value that can be changed (ui.md §4).
	valueColor = lipgloss.Color("#cba6f7") // mauve
	// the user's footprint: the active saver's dot, PIN "set", a toggle "on".
	liveColor = lipgloss.Color("#a6e3a1") // green
	// overrides, outside the brightness hierarchy (§2.4): red is "wrong",
	// yellow is "not set yet".
	warnColor   = lipgloss.Color("#f38ba8") // red
	yellowColor = lipgloss.Color("#f9e2af") // yellow
	// the lit cells while the PIN prompt is up: the board steps back to be
	// the prompt's backdrop (ui.md §2.3).
	backdropColor = borderDim
)

const baseHex = "#1e1e2e" // canvas; also dark text on a bright chip

// Nerd Font glyphs. Never a PUA literal in source — built from the rune so
// the codepoint stays greppable and the file stays editor-safe (family
// rule). Every codepoint below was read out of the installed font's cmap
// (JetBrainsMono Nerd Font Mono, 2026-09-24), not remembered.
var (
	// pixelGlyph is one cell of the board: nf-fa-square, the same glyph the
	// family's splashes are drawn with.
	pixelGlyph = string(rune(0xf0c8)) // nf-fa-square
	glyphLock  = string(rune(0xf023)) // nf-fa-lock             — the PIN prompt
	glyphMenu  = string(rune(0xf0c9)) // nf-fa-bars (reorder)   — Space menu
	glyphHelp  = string(rune(0xf059)) // nf-fa-question_circle  — help
	glyphWarn  = string(rune(0xf071)) // nf-fa-warning          — confirm, error toast
	glyphInfo  = string(rune(0xf05a)) // nf-fa-info_circle      — toast
	glyphInput = string(rune(0xf040)) // nf-fa-pencil           — input popup
	glyphList  = string(rune(0xf0c9)) // the options list is a menu too

	// A title chip's ends and seams, as webu's and sshu's (chrome.go):
	// round caps, and between two chips of one chain a slanted seam —
	// filled where the fills differ, a thin slash where they do not.
	capLeft     = string(rune(0xe0b6)) // powerline round-left
	capRight    = string(rune(0xe0b4)) // powerline round-right
	dividerHard = string(rune(0xe0bc)) // ple-upper_left_triangle
	dividerSoft = string(rune(0xe0bb)) // ple-forwardslash_separator
)
