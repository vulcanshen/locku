package ui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/saver"
)

var at = time.Date(2026, time.September, 24, 21, 5, 9, 0, time.UTC)

// The scales function.md §12 expects, with rows = terminal height − 1 for
// the status row. The size asked for is kept while some content fits at
// it; then it steps down.
func TestFitPicksTheScale(t *testing.T) {
	hm := saver.Clock{Time: saver.TimeHM, Date: saver.DateOff, Layout: saver.LayoutRow}
	full := saver.Clock{Time: saver.TimeHMS, Date: saver.DateYMD, Layout: saver.LayoutRow}
	col := saver.Clock{Time: saver.TimeHMS, Date: saver.DateOff, Layout: saver.LayoutColumn}
	cases := []struct {
		name       string
		s          saver.Clock
		cols, rows int
		size       int
		k          int
		lines      []string
	}{
		{"80x24 HH:MM large steps down to 1", hm, 80, 23, 3, 1, []string{"21:05"}},
		{"120x40 HH:MM large steps down to 2", hm, 120, 39, 3, 2, []string{"21:05"}},
		{"200x60 HH:MM large", hm, 200, 59, 3, 3, []string{"21:05"}},
		{"200x60 HH:MM medium stays 2", hm, 200, 59, 2, 2, []string{"21:05"}},
		{"200x60 HH:MM small stays 1", hm, 200, 59, 1, 1, []string{"21:05"}},
		{"80x24 HH:MM:SS + YYYY-MM-DD small degrades to HH:MM + MM-DD", full, 80, 23, 1, 1, []string{"21:05", "09-24"}},
		// At medium the content gives way before the size: 80 columns hold
		// 38 pixels, HH:MM at 2 is 58, so it steps to small — and at small
		// the full content is back on the ladder from the top.
		{"80x24 full clock medium: small, HH:MM + MM-DD", full, 80, 23, 2, 1, []string{"21:05", "09-24"}},
		// 200 columns hold 98 pixels: the full clock at 2 is 118, without
		// the year 94 — the year goes and the seconds stay.
		{"200x60 full clock medium: the year goes first", full, 200, 59, 2, 2, []string{"21:05:09", "09-24"}},
		{"40x12 HH:MM is plain text", hm, 40, 11, 2, 0, []string{"21:05"}},
		{"40x12 full clock degrades all the way to plain HH:MM", full, 40, 11, 3, 0, []string{"21:05"}},
		// A column is two characters wide: 11 px; three lines are 25 px tall.
		{"120x40 column HH:MM:SS medium: 39 rows hold 16 px at 2, not 25", col, 120, 39, 2, 2, []string{"21", "05"}},
		{"120x80 column HH:MM:SS large", col, 120, 79, 3, 3, []string{"21", "05", "09"}},
	}
	for _, c := range cases {
		lines, k := fit(c.s, at, c.cols, c.rows, c.size)
		if k != c.k || !reflect.DeepEqual(lines, c.lines) {
			t.Errorf("%s: got k=%d %q, want k=%d %q", c.name, k, lines, c.k, c.lines)
		}
	}
}

func TestPaintCentresTheBlock(t *testing.T) {
	// 120×40 at 2: "21:05" is 29 × 7 font pixels → 58 × 14 on a 60 × 39 board.
	b := paint([]string{"21:05"}, 2, 120, 39)
	if b.w != 60 || b.h != 39 {
		t.Fatalf("board %d×%d", b.w, b.h)
	}
	minX, minY, maxX, maxY := b.w, b.h, -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.at(x, y) {
				minX, maxX = min(minX, x), max(maxX, x)
				minY, maxY = min(minY, y), max(maxY, y)
			}
		}
	}
	if minX != 1 || maxX != 58 || minY != 12 || maxY != 25 {
		t.Errorf("lit box x %d..%d y %d..%d; want 1..58, 12..25", minX, maxX, minY, maxY)
	}
	// Scaling multiplies the lit count by k².
	if one := paint([]string{"21:05"}, 1, 120, 39).count(); b.count() != one*4 {
		t.Errorf("k=2 lights %d, k=1 lights %d", b.count(), one)
	}
}

func TestRowsAreExactlyTheTerminalWide(t *testing.T) {
	bg, fg := lipgloss.Color("#313244"), lipgloss.Color("#f2b753")
	for _, cols := range []int{80, 81, 40, 7} {
		rows := 23
		lines, k := fit(saver.Clock{Time: saver.TimeHM, Date: saver.DateYMD}, at, cols, rows, 2)
		var out []string
		if k >= 1 {
			out = boardRows(paint(lines, k, cols, rows), bg, fg, cols, false)
		} else {
			out = plainRows(lines, bg, fg, cols, rows, true)
		}
		if len(out) != rows {
			t.Fatalf("cols %d: %d rows, want %d", cols, len(out), rows)
		}
		for i, r := range out {
			if w := lipgloss.Width(r); w != cols {
				t.Errorf("cols %d row %d is %d wide", cols, i, w)
			}
		}
		if s := statusRow(cols, true, "vulcan", "prod-db-01", at, true, ""); lipgloss.Width(s) != cols {
			t.Errorf("cols %d status is %d wide", cols, lipgloss.Width(s))
		}
	}
}

func TestPlainTextCarriesTheLines(t *testing.T) {
	out := plainRows([]string{"21:05", "09-24"}, lipgloss.Color("#000000"), lipgloss.Color("#ffffff"), 40, 11, false)
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "21:05") || !strings.Contains(joined, "09-24") {
		t.Errorf("text missing:\n%s", joined)
	}
}

func TestStatusRowSaysWhatMatters(t *testing.T) {
	s := statusRow(80, true, "vulcan", "host", at, false, "")
	if !strings.Contains(s, "vulcan@host · locked since 21:05") {
		t.Errorf("%q", s)
	}
	s = statusRow(80, false, "vulcan", "host", at, true, "")
	if strings.Contains(s, "vulcan") || !strings.Contains(s, "no PIN · any key unlocks") {
		t.Errorf("status off must still say there is no PIN: %q", s)
	}
	s = statusRow(80, true, "v", "h", at, true, "config.yaml: yaml: line 1")
	if !strings.Contains(s, "config error: config.yaml") || !strings.Contains(s, "no PIN") {
		t.Errorf("%q", s)
	}
}
