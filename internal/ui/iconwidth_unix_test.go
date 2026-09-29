//go:build darwin || linux

package ui

import "testing"

// LOCKU__ICON_WIDTH decides when set to 1 or 2; anything else is asked of
// the terminal, which a test has none of.
func TestIconWidthOverride(t *testing.T) {
	withIcons(t, 1)
	t.Setenv("LOCKU__ICON_WIDTH", "2")
	DetectIconWidth()
	if iconCells != 2 {
		t.Errorf("LOCKU__ICON_WIDTH=2: %d", iconCells)
	}
	iconCells = 1
	t.Setenv("LOCKU__ICON_WIDTH", "3")
	DetectIconWidth()
	if iconCells != 1 {
		t.Errorf("LOCKU__ICON_WIDTH=3 is not a width: %d", iconCells)
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
