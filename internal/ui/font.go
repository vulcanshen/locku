package ui

import "github.com/vulcanshen/locku/internal/saver"

// The pixel fonts: one glyph for each of the 39 characters a saver can
// produce — the digits, the colon, the hyphen, the space, and the capitals
// (function.md §5.2 / §10.16) — in two faces, 3 × 7 and 3 × 5, a saver's
// choice (user, 2026-09-24: try the short one). There is no lower case:
// the content is made of fixed choices, so this is the whole alphabet.
//
// Every stroke is horizontal or vertical — the look of a seven-segment
// display drawn in pixels: the zero has no slash, the seven no hook, the
// letters no diagonals (user, 2026-09-24; TestNoDiagonals keeps it so).
// And, being right-angled, the glyphs are narrow: a digit is three cells
// wide, as on a seven-segment display. Letters are three wide too, but
// for M and W, which have no shape at three; where a letter has no
// right-angled shape of its own it takes the blocky one pixel fonts use —
// N is a bridge, V a U with a point — and S, O and I read as 5, 0 and 1,
// the way a seven-segment clock reads them. Only the month names ever put
// a letter on the board.
//
// The punctuation is as wide as it needs to be — the colon one cell, the
// hyphen three. The space is one cell here and, on the board, one gap
// unit: it is dark, so it is a gap, not a glyph (canvas.go). A row is
// '#' lit and '.' dark, top to bottom; every row of a glyph is the same
// width, every glyph of a face the same height.

// fontW is a digit's width, and most letters'.
const fontW = 3

// Charset is every rune the fonts can draw, which is every rune a saver
// may emit; TestSaverStaysInTheFont holds the two together.
const Charset = "0123456789:- ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// face is one font: its height and its glyphs.
type face struct {
	name string
	h    int
	g    map[rune][]string
}

// faceOf is the face a saver's font names; anything else is the tall one.
func faceOf(name string) face {
	if name == saver.FontShort {
		return faceShort
	}
	return faceTall
}

// glyphW is a glyph's width in font pixels.
func (f face) glyphW(r rune) int {
	if g, ok := f.g[r]; ok {
		return len(g[0])
	}
	return fontW
}

var faceTall = face{name: saver.FontTall, h: 7, g: map[rune][]string{
	'0': {"###", "#.#", "#.#", "#.#", "#.#", "#.#", "###"},
	'1': {".#.", "##.", ".#.", ".#.", ".#.", ".#.", "###"},
	'2': {"###", "..#", "..#", "###", "#..", "#..", "###"},
	'3': {"###", "..#", "..#", "###", "..#", "..#", "###"},
	'4': {"#.#", "#.#", "#.#", "###", "..#", "..#", "..#"},
	'5': {"###", "#..", "#..", "###", "..#", "..#", "###"},
	'6': {"###", "#..", "#..", "###", "#.#", "#.#", "###"},
	'7': {"###", "..#", "..#", "..#", "..#", "..#", "..#"},
	'8': {"###", "#.#", "#.#", "###", "#.#", "#.#", "###"},
	'9': {"###", "#.#", "#.#", "###", "..#", "..#", "###"},
	':': {".", "#", "#", ".", "#", "#", "."},
	'-': {"...", "...", "...", "###", "...", "...", "..."},
	' ': {".", ".", ".", ".", ".", ".", "."},
	'A': {"###", "#.#", "#.#", "###", "#.#", "#.#", "#.#"},
	'B': {"##.", "#.#", "#.#", "##.", "#.#", "#.#", "##."},
	'C': {"###", "#..", "#..", "#..", "#..", "#..", "###"},
	'D': {"##.", "#.#", "#.#", "#.#", "#.#", "#.#", "##."},
	'E': {"###", "#..", "#..", "##.", "#..", "#..", "###"},
	'F': {"###", "#..", "#..", "##.", "#..", "#..", "#.."},
	'G': {"###", "#..", "#..", "#..", "#.#", "#.#", "###"},
	'H': {"#.#", "#.#", "#.#", "###", "#.#", "#.#", "#.#"},
	'I': {"###", ".#.", ".#.", ".#.", ".#.", ".#.", "###"},
	'J': {"###", "..#", "..#", "..#", "..#", "#.#", "###"},
	'K': {"#.#", "#.#", "#.#", "##.", "#.#", "#.#", "#.#"},
	'L': {"#..", "#..", "#..", "#..", "#..", "#..", "###"},
	'M': {"#...#", "##.##", "#.#.#", "#.#.#", "#...#", "#...#", "#...#"},
	'N': {"###", "#.#", "#.#", "#.#", "#.#", "#.#", "#.#"},
	'O': {"###", "#.#", "#.#", "#.#", "#.#", "#.#", "###"},
	'P': {"###", "#.#", "#.#", "###", "#..", "#..", "#.."},
	'Q': {"###", "#.#", "#.#", "#.#", "#.#", "###", "..#"},
	'R': {"###", "#.#", "#.#", "###", "#..", "#.#", "#.#"},
	'S': {"###", "#..", "#..", "###", "..#", "..#", "###"},
	'T': {"###", ".#.", ".#.", ".#.", ".#.", ".#.", ".#."},
	'U': {"#.#", "#.#", "#.#", "#.#", "#.#", "#.#", "###"},
	'V': {"#.#", "#.#", "#.#", "#.#", "#.#", "###", ".#."},
	'W': {"#...#", "#...#", "#...#", "#.#.#", "#.#.#", "#.#.#", "#####"},
	'X': {"#.#", "#.#", "###", ".#.", "###", "#.#", "#.#"},
	'Y': {"#.#", "#.#", "#.#", "###", ".#.", ".#.", ".#."},
	'Z': {"###", "..#", "..#", "###", "#..", "#..", "###"},
}}

// faceShort is the same alphabet in five rows: the middle bar sits on the
// third row, so a digit is a seven-segment digit with the verticals one
// pixel long. Two lines of it are 11 pixels tall where the tall face
// needs 15, and three 17 where it needs 23 — a column of HH MM SS at the
// large size fits 54 rows instead of 72.
var faceShort = face{name: saver.FontShort, h: 5, g: map[rune][]string{
	'0': {"###", "#.#", "#.#", "#.#", "###"},
	'1': {".#.", "##.", ".#.", ".#.", "###"},
	'2': {"###", "..#", "###", "#..", "###"},
	'3': {"###", "..#", "###", "..#", "###"},
	'4': {"#.#", "#.#", "###", "..#", "..#"},
	'5': {"###", "#..", "###", "..#", "###"},
	'6': {"###", "#..", "###", "#.#", "###"},
	'7': {"###", "..#", "..#", "..#", "..#"},
	'8': {"###", "#.#", "###", "#.#", "###"},
	'9': {"###", "#.#", "###", "..#", "###"},
	':': {".", "#", ".", "#", "."},
	'-': {"...", "...", "###", "...", "..."},
	' ': {".", ".", ".", ".", "."},
	'A': {"###", "#.#", "###", "#.#", "#.#"},
	'B': {"###", "#.#", "###", "#.#", "###"}, // as 8, the way a seven-segment display shows it
	'C': {"###", "#..", "#..", "#..", "###"},
	'D': {"##.", "#.#", "#.#", "#.#", "##."},
	'E': {"###", "#..", "##.", "#..", "###"},
	'F': {"###", "#..", "##.", "#..", "#.."},
	'G': {"###", "#..", "#.#", "#.#", "###"},
	'H': {"#.#", "#.#", "###", "#.#", "#.#"},
	'I': {"###", ".#.", ".#.", ".#.", "###"},
	'J': {"###", "..#", "..#", "#.#", "###"},
	'K': {"#.#", "#.#", "##.", "#.#", "#.#"},
	'L': {"#..", "#..", "#..", "#..", "###"},
	'M': {"#...#", "##.##", "#.#.#", "#.#.#", "#...#"},
	'N': {"###", "#.#", "#.#", "#.#", "#.#"},
	'O': {"###", "#.#", "#.#", "#.#", "###"},
	'P': {"###", "#.#", "###", "#..", "#.."},
	'Q': {"###", "#.#", "#.#", "###", "..#"},
	'R': {"###", "#.#", "##.", "#.#", "#.#"},
	'S': {"###", "#..", "###", "..#", "###"},
	'T': {"###", ".#.", ".#.", ".#.", ".#."},
	'U': {"#.#", "#.#", "#.#", "#.#", "###"},
	'V': {"#.#", "#.#", "#.#", "###", ".#."},
	'W': {"#...#", "#...#", "#.#.#", "#.#.#", "#####"},
	'X': {"#.#", "###", ".#.", "###", "#.#"},
	'Y': {"#.#", "#.#", "###", ".#.", ".#."},
	'Z': {"###", "..#", "###", "#..", "###"},
}}
