package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/vulcanshen/locku/internal/saver"
)

func TestFontIsWellFormed(t *testing.T) {
	if len(font) != 39 || len([]rune(Charset)) != 39 {
		t.Fatalf("font has %d glyphs, charset %d runes; want 39", len(font), len([]rune(Charset)))
	}
	for _, r := range Charset {
		g, ok := font[r]
		if !ok {
			t.Errorf("no glyph for %q", r)
			continue
		}
		w := len(g[0])
		if w < 1 || w > 5 {
			t.Errorf("%q is %d wide", r, w)
		}
		if strings.ContainsRune("0123456789ABCDEFGHIJKLNOPQRSTUVXYZ", r) && w != fontW {
			t.Errorf("%q is %d wide; digits and letters are %d", r, w, fontW)
		}
		if (r == 'M' || r == 'W') && w != 5 {
			t.Errorf("%q is %d wide; M and W need 5", r, w)
		}
		for y, row := range g {
			if len(row) != w {
				t.Errorf("%q row %d is %d wide, row 0 is %d", r, y, len(row), w)
			}
			if strings.Trim(row, "#.") != "" {
				t.Errorf("%q row %d has a stray cell: %q", r, y, row)
			}
		}
	}
	for r := range font {
		if !strings.ContainsRune(Charset, r) {
			t.Errorf("glyph %q is not in the charset", r)
		}
	}
	// Every glyph but the space lights something.
	for r, g := range font {
		if r == ' ' {
			continue
		}
		if strings.Count(strings.Join(g[:], ""), "#") == 0 {
			t.Errorf("%q is blank", r)
		}
	}
	// The punctuation is narrow: that is what keeps a clock within reach
	// of the large size.
	if glyphW(':') != 1 || glyphW(' ') != 2 || glyphW('-') != 3 || glyphW('7') != 3 || glyphW('?') != fontW {
		t.Errorf("widths: : %d, space %d, - %d, 7 %d, unknown %d", glyphW(':'), glyphW(' '), glyphW('-'), glyphW('7'), glyphW('?'))
	}
}

// Every stroke is horizontal or vertical: a lit pixel always has a lit
// neighbour straight above, below, left or right, so no glyph is drawn
// with a stair of single pixels — which is what a diagonal is on a grid
// (user, 2026-09-24).
func TestNoDiagonals(t *testing.T) {
	for r, g := range font {
		w := len(g[0])
		lit := func(x, y int) bool {
			return x >= 0 && x < w && y >= 0 && y < fontH && g[y][x] == '#'
		}
		for y := 0; y < fontH; y++ {
			for x := 0; x < w; x++ {
				if !lit(x, y) {
					continue
				}
				if !lit(x-1, y) && !lit(x+1, y) && !lit(x, y-1) && !lit(x, y+1) {
					t.Errorf("%q: the pixel at %d,%d stands alone — a diagonal, or a stray", r, x, y)
				}
			}
		}
	}
}

// Every line a saver can produce, in every shape, across a year, is drawn
// from the charset — the promise of function.md §5.2 that the content is
// made of fixed choices.
func TestSaverStaysInTheFont(t *testing.T) {
	for _, tf := range saver.TimeFormats {
		for _, df := range saver.DateFormats {
			c := saver.Clock{Time: tf, Date: df}
			for month := 1; month <= 12; month++ {
				for _, h := range []int{0, 9, 12, 23} {
					at := time.Date(2026, time.Month(month), 28, h, 59, 59, 0, time.UTC)
					for _, line := range c.Lines(at) {
						for _, r := range line {
							if _, ok := font[r]; !ok {
								t.Errorf("%s / %s at %v: %q has no glyph", tf, df, at, r)
							}
						}
					}
				}
			}
		}
	}
}
