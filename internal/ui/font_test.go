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
		for y, row := range g {
			if len(row) != fontW {
				t.Errorf("%q row %d is %d wide", r, y, len(row))
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
	// Every glyph but the space and the punctuation lights something.
	for r, g := range font {
		if r == ' ' {
			continue
		}
		if strings.Count(strings.Join(g[:], ""), "#") == 0 {
			t.Errorf("%q is blank", r)
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
