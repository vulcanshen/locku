//go:build darwin || linux

package ui

import "testing"

// LOCKU__ICON_WIDTH decides, then TERMINU__ICON_WIDTH, when set to 1 or 2;
// with neither the terminal is asked, which a test has none of (tdp D6,
// v0.1.22).
func TestIconWidthOverride(t *testing.T) {
	withIcons(t, 1)
	for _, c := range []struct {
		own, family string
		want        int
	}{
		{"2", "", 2},
		{"", "2", 2},
		{"1", "2", 1}, // the user's own over the family's
		{"2", "1", 2},
		{"3", "2", 2}, // not a width: as if unset
		{"x", "0", 1}, // neither: asked, and no terminal answers
	} {
		iconCells = 1
		t.Setenv("LOCKU__ICON_WIDTH", c.own)
		t.Setenv("TERMINU__ICON_WIDTH", c.family)
		DetectIconWidth()
		if iconCells != c.want {
			t.Errorf("LOCKU__ICON_WIDTH=%q TERMINU__ICON_WIDTH=%q: %d, want %d", c.own, c.family, iconCells, c.want)
		}
	}
}

func TestParseCPRColumn(t *testing.T) {
	for in, want := range map[string]int{"\x1b[12;3R": 3, "\x1b[1;2R": 2} {
		if got, ok := parseCPRColumn([]byte(in)); !ok || got != want {
			t.Errorf("%q: %d %v", in, got, ok)
		}
	}
	for _, in := range []string{"\x1b[12R", "garbage", "\x1b[1;xR"} {
		if _, ok := parseCPRColumn([]byte(in)); ok {
			t.Errorf("%q is no answer", in)
		}
	}
}
