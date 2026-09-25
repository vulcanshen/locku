package ui

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestSplashIsTheIcon holds the splash to docs/icon.svg cell for cell: the
// icon's 25 × 25 grid, rows 2 to 22, is the splash's 21 rows. The splash
// was drawn a day before the icon existed and kept a padlock the icon
// never had (2026-09-26); a change to the icon now fails here until the
// splash follows it.
func TestSplashIsTheIcon(t *testing.T) {
	src, err := os.ReadFile("../../docs/icon.svg")
	if err != nil {
		t.Fatal(err)
	}
	var grid [25][25]byte
	for r := range grid {
		for c := range grid[r] {
			grid[r][c] = 'D'
		}
	}
	comment := regexp.MustCompile(`<!-- (\w) -->`)
	cell := regexp.MustCompile(`x="(\d+)" y="(\d+)" width="10"`)
	var code byte
	for _, line := range strings.Split(string(src), "\n") {
		if m := comment.FindStringSubmatch(line); m != nil {
			code = m[1][0]
			continue
		}
		if m := cell.FindStringSubmatch(line); m != nil && code != 0 {
			x, _ := strconv.Atoi(m[1])
			y, _ := strconv.Atoi(m[2])
			grid[y/10][x/10] = code
		}
	}
	for r, want := range logoPixels {
		if got := string(grid[r+2][:]); got != want {
			t.Errorf("splash row %d = %s, icon row %d = %s", r, want, r+2, got)
		}
	}
	for _, r := range []int{0, 1, 23, 24} {
		if strings.Trim(string(grid[r][:]), "D") != "" {
			t.Errorf("icon row %d has pixels the splash does not show", r)
		}
	}
}

// TestSplashRevealsEveryPixelOnce: the stages together cover the sheet
// once, then every letter and the frame once each, in the icon's order.
func TestSplashRevealsEveryPixelOnce(t *testing.T) {
	var m splashModel
	m.show()
	cells := len(logoPixels) * len(logoPixels[0])
	marks := 0
	for _, row := range logoPixels {
		marks += len(row) - strings.Count(row, "D")
	}
	if got := len(m.pixelOrder); got != cells+marks {
		t.Fatalf("revealed %d pixels, want the sheet %d + the marks %d", got, cells, marks)
	}
	if got := len(m.stageEnds); got != 6 {
		t.Fatalf("%d stages, want sheet, L, O, C, K, U", got)
	}
}
