package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/vulcanshen/locku/internal/saver"
)

var faces = []face{faceTall, faceShort}

func TestFontIsWellFormed(t *testing.T) {
	if len([]rune(Charset)) != 39 {
		t.Fatalf("charset %d runes; want 39", len([]rune(Charset)))
	}
	for _, f := range faces {
		if len(f.g) != 39 {
			t.Errorf("%s has %d glyphs; want 39", f.name, len(f.g))
		}
		for _, r := range Charset {
			g, ok := f.g[r]
			if !ok {
				t.Errorf("%s: no glyph for %q", f.name, r)
				continue
			}
			if len(g) != f.h {
				t.Errorf("%s: %q is %d rows, the face is %d", f.name, r, len(g), f.h)
				continue
			}
			w := len(g[0])
			if w < 1 || w > 5 {
				t.Errorf("%s: %q is %d wide", f.name, r, w)
			}
			if strings.ContainsRune("0123456789ABCDEFGHIJKLNOPQRSTUVXYZ", r) && w != fontW {
				t.Errorf("%s: %q is %d wide; digits and letters are %d", f.name, r, w, fontW)
			}
			if (r == 'M' || r == 'W') && w != 5 {
				t.Errorf("%s: %q is %d wide; M and W need 5", f.name, r, w)
			}
			for y, row := range g {
				if len(row) != w {
					t.Errorf("%s: %q row %d is %d wide, row 0 is %d", f.name, r, y, len(row), w)
				}
				if strings.Trim(row, "#.") != "" {
					t.Errorf("%s: %q row %d has a stray cell: %q", f.name, r, y, row)
				}
			}
			if r != ' ' && strings.Count(strings.Join(g, ""), "#") == 0 {
				t.Errorf("%s: %q is blank", f.name, r)
			}
		}
		for r := range f.g {
			if !strings.ContainsRune(Charset, r) {
				t.Errorf("%s: glyph %q is not in the charset", f.name, r)
			}
		}
		// The punctuation is narrow: that is what keeps a clock within
		// reach of the large size.
		if f.glyphW(':') != 1 || f.glyphW(' ') != 2 || f.glyphW('-') != 3 || f.glyphW('7') != 3 || f.glyphW('?') != fontW {
			t.Errorf("%s widths: : %d, space %d, - %d, 7 %d, unknown %d", f.name,
				f.glyphW(':'), f.glyphW(' '), f.glyphW('-'), f.glyphW('7'), f.glyphW('?'))
		}
	}
	if faceOf("nonsense").name != saver.FontTall || faceOf(saver.FontShort).h != 5 {
		t.Error("faceOf")
	}
}

// Every stroke is horizontal or vertical: a lit pixel always has a lit
// neighbour straight above, below, left or right, so no glyph is drawn
// with a stair of single pixels — which is what a diagonal is on a grid
// (user, 2026-09-24). The colon is two dots, and a dot stands alone.
func TestNoDiagonals(t *testing.T) {
	for _, f := range faces {
		for r, g := range f.g {
			if r == ':' {
				continue
			}
			w := len(g[0])
			lit := func(x, y int) bool {
				return x >= 0 && x < w && y >= 0 && y < f.h && g[y][x] == '#'
			}
			for y := 0; y < f.h; y++ {
				for x := 0; x < w; x++ {
					if !lit(x, y) {
						continue
					}
					if !lit(x-1, y) && !lit(x+1, y) && !lit(x, y-1) && !lit(x, y+1) {
						t.Errorf("%s: %q: the pixel at %d,%d stands alone — a diagonal, or a stray", f.name, r, x, y)
					}
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
							for _, f := range faces {
								if _, ok := f.g[r]; !ok {
									t.Errorf("%s / %s at %v: %q has no glyph in %s", tf, df, at, r, f.name)
								}
							}
						}
					}
				}
			}
		}
	}
}
