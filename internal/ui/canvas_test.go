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
		// HH MM is 18 px: 80 columns hold 38, so large (54) steps down to 2 (36).
		{"80x24 HH MM large steps down to 2", hm, 80, 23, 3, 2, []string{"21 05"}},
		{"120x40 HH MM large", hm, 120, 39, 3, 3, []string{"21 05"}},
		{"200x60 HH MM large", hm, 200, 59, 3, 3, []string{"21 05"}},
		{"200x60 HH MM medium stays 2", hm, 200, 59, 2, 2, []string{"21 05"}},
		{"200x60 HH MM small stays 1", hm, 200, 59, 1, 1, []string{"21 05"}},
		// The full clock is 39 px wide (the date); without the year 29.
		{"80x24 full clock small: the year goes", full, 80, 23, 1, 1, []string{"21 05 09", "09-24"}},
		// At medium on 80 columns: the year (58) and the seconds (38 — but
		// 15 rows at 2 is 30, over 21) go, then the date; HH MM at 2 fits.
		{"80x24 full clock medium: HH MM alone, at 2", full, 80, 23, 2, 2, []string{"21 05"}},
		{"200x60 full clock medium: all of it", full, 200, 59, 2, 2, []string{"21 05 09", "2026-09-24"}},
		// Too short for two lines, wide enough for the seconds: the date
		// goes and the seconds stay (user, 2026-09-24 — a chain that
		// dropped the seconds on the way to the date showed HH MM here).
		{"130x24 full clock medium: the date goes, not the seconds", full, 130, 23, 2, 2, []string{"21 05 09"}},
		{"130x24 HH MM SS + MM-DD medium: the same", saver.Clock{Time: saver.TimeHMS, Date: saver.DateMD}, 130, 23, 2, 2, []string{"21 05 09"}},
		// And where both fit, the date is worth more than the seconds.
		{"130x40 HH MM SS + MM-DD medium: both", saver.Clock{Time: saver.TimeHMS, Date: saver.DateMD}, 130, 39, 2, 2, []string{"21 05 09", "09-24"}},
		{"84x40 HH MM SS + MM-DD medium: the seconds go, the date stays", saver.Clock{Time: saver.TimeHMS, Date: saver.DateMD}, 84, 39, 2, 2, []string{"21 05", "09-24"}},
		// 40 columns hold 18 px: exactly HH MM at 1.
		{"40x12 HH MM is 1", hm, 40, 11, 2, 1, []string{"21 05"}},
		{"40x12 full clock degrades to HH MM at 1", full, 40, 11, 3, 1, []string{"21 05"}},
		{"30x8 HH MM is plain text", hm, 30, 7, 2, 0, []string{"21 05"}},
		{"30x8 full clock degrades all the way to plain HH MM", full, 30, 7, 3, 0, []string{"21 05"}},
		// A column is two digits wide: 7 px; three lines are 23 px tall.
		{"120x40 column HH MM SS medium: 39 rows hold 16 px at 2, not 23", col, 120, 39, 2, 2, []string{"21", "05"}},
		{"120x80 column HH MM SS large", col, 120, 79, 3, 3, []string{"21", "05", "09"}},
	}
	for _, c := range cases {
		lines, k := fit(faceTall, c.s, at, c.cols, c.rows, c.size)
		if k != c.k || !reflect.DeepEqual(lines, c.lines) {
			t.Errorf("%s: got k=%d %q, want k=%d %q", c.name, k, lines, c.k, c.lines)
		}
	}
}

// The widths the sizes are reasoned from (function.md §5.3): digits and
// most letters three, M and W five, the space two, the hyphen three, a gap
// of one between.
func TestLineWidths(t *testing.T) {
	for line, want := range map[string]int{
		"21 05":      18, // 3+1+3 +1+2+1+ 3+1+3
		"21 05 09":   29,
		"2026-09-24": 39, // 8 digits, 2 hyphens, 9 gaps
		"09-24":      19,
		"SEP-24":     23,
		"MAR-24":     25,
		"21":         7,
		"2026":       15,
	} {
		if got := lineW(faceTall, line); got != want {
			t.Errorf("%q: %d px, want %d", line, got, want)
		}
	}
	// What the large size needs, in columns: HH MM 108, the seconds 174.
	if w, _ := pixelSize(faceTall, []string{"21 05"}); w*3*2 != 108 {
		t.Errorf("HH MM at 3 is %d columns", w*3*2)
	}
	if w, _ := pixelSize(faceTall, []string{"21 05 09"}); w*3*2 != 174 {
		t.Errorf("HH MM SS at 3 is %d columns", w*3*2)
	}
	// A column of HH / MM at 3 is 45 rows; with SS 69.
	if _, h := pixelSize(faceTall, []string{"21", "05"}); h*3 != 45 {
		t.Errorf("HH/MM at 3 is %d rows", h*3)
	}
	if _, h := pixelSize(faceTall, []string{"21", "05", "09"}); h*3 != 69 {
		t.Errorf("HH/MM/SS at 3 is %d rows", h*3)
	}
	// The short face: the same widths, five rows a line — a column of
	// HH / MM / SS at 3 is 51 rows.
	if w, h := pixelSize(faceShort, []string{"21 05 09"}); w != 29 || h != 5 {
		t.Errorf("short HH MM SS is %d×%d", w, h)
	}
	if _, h := pixelSize(faceShort, []string{"21", "05", "09"}); h*3 != 51 {
		t.Errorf("short HH/MM/SS at 3 is %d rows", h*3)
	}
	// A column of HH / MM / SS at large needs 54 rows in the short face
	// and 72 in the tall one; 43 rows hold HH / MM at large in the short
	// face and only at medium in the tall.
	col := saver.Clock{Time: saver.TimeHMS, Layout: saver.LayoutColumn}
	if lines, k := fit(faceShort, col, at, 160, 53, 3); k != 3 || len(lines) != 3 {
		t.Errorf("short column at 160x54: k=%d %q", k, lines)
	}
	if lines, k := fit(faceShort, col, at, 160, 42, 3); k != 3 || len(lines) != 2 {
		t.Errorf("short column at 160x43: k=%d %q", k, lines)
	}
	if lines, k := fit(faceTall, col, at, 160, 42, 3); k != 2 || len(lines) != 2 {
		t.Errorf("tall column at 160x43: k=%d %q", k, lines)
	}
}

func TestPaintCentresTheBlock(t *testing.T) {
	// 120×40 at 2: "21 05" is 18 × 7 font pixels → 36 × 14 on a 60 × 39 board.
	b := paint(faceTall, []string{"21 05"}, 2, 120, 39)
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
	if minX != 12 || maxX != 47 || minY != 12 || maxY != 25 {
		t.Errorf("lit box x %d..%d y %d..%d; want 12..47, 12..25", minX, maxX, minY, maxY)
	}
	// Scaling multiplies the lit count by k².
	if one := paint(faceTall, []string{"21 05"}, 1, 120, 39).count(); b.count() != one*4 {
		t.Errorf("k=2 lights %d, k=1 lights %d", b.count(), one)
	}
}

func TestRowsAreExactlyTheTerminalWide(t *testing.T) {
	bg, fg := lipgloss.Color("#313244"), lipgloss.Color("#f2b753")
	for _, cols := range []int{80, 81, 40, 7} {
		rows := 23
		lines, k := fit(faceTall, saver.Clock{Time: saver.TimeHM, Date: saver.DateYMD}, at, cols, rows, 2)
		var out []string
		if k >= 1 {
			out = boardRows(paint(faceTall, lines, k, cols, rows), bg, fg, cols, false)
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
	out := plainRows([]string{"21 05", "09-24"}, lipgloss.Color("#000000"), lipgloss.Color("#ffffff"), 40, 11, false)
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "21 05") || !strings.Contains(joined, "09-24") {
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
