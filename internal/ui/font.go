package ui

// The pixel font: 7 rows high, one glyph for each of the 39 characters a
// saver can produce — the digits, the colon, the hyphen, the space, and the
// capitals (function.md §5.2 / §10.16). There is no second font and no
// lower case: the content is made of fixed choices, so this is the whole
// alphabet.
//
// Every stroke is horizontal or vertical — the look of a seven-segment
// display drawn in pixels: the zero has no slash, the seven no hook, the
// letters no diagonals (user, 2026-09-24; TestNoDiagonals keeps it so).
// And, being right-angled, the glyphs are narrow: a digit is three cells
// wide, as on a seven-segment display, so a clock is half again as
// compact as it was at five (user, 2026-09-24: squeeze them, fit more).
// Letters are three wide too, but for M and W, which have no shape at
// three; where a letter has no right-angled shape of its own it takes the
// blocky one pixel fonts use — N is a bridge, V a U with a point — and S,
// O and I read as 5, 0 and 1, the way a seven-segment clock reads them.
// Only the month names ever put a letter on the board.
//
// The punctuation is as wide as it needs to be — the colon one cell, the
// space two, the hyphen three. A row is '#' lit and '.' dark, top to
// bottom; every row of a glyph is the same width.

const (
	fontW = 3 // a digit, and most letters
	fontH = 7
)

// Charset is every rune the font can draw, which is every rune a saver may
// emit; TestSaverStaysInTheFont holds the two together.
const Charset = "0123456789:- ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// glyphW is a glyph's width in font pixels.
func glyphW(r rune) int {
	if g, ok := font[r]; ok {
		return len(g[0])
	}
	return fontW
}

var font = map[rune][fontH]string{
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
	' ': {"..", "..", "..", "..", "..", "..", ".."},
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
}
