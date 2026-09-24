package ui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// TestDump prints the screens at a few sizes, for eyes rather than
// assertions: LOCKU_DUMP=1 go test ./internal/ui -run TestDump -v
// (the family's `make dump`). Without the variable it is a no-op.
func TestDump(t *testing.T) {
	if os.Getenv("LOCKU_DUMP") == "" {
		t.Skip("set LOCKU_DUMP=1 to print the screens")
	}
	t.Setenv("LOCKU_CONFIG", t.TempDir())
	cfg := config.Default()
	second := config.DefaultProfile()
	second.Name, second.Time, second.Date = "clock2", "HH MM SS", "YYYY-MMM-DD"
	cfg.Profiles = append(cfg.Profiles, second)
	show := func(label string, v string) {
		fmt.Printf("===== %s =====\n%s\n", label, v)
	}
	m := NewApp(cfg, "").size(100, 30)
	show("settings 100x30, [1] on clock", m.View())
	show("settings, [2] preference", m.press("G", "2").View())
	show("settings, space menu on clock2", m.press("j", " ").View())
	show("settings, [2] clock with a colour draft", m.press("2", "G", "enter", "G", "enter").View())
	show("settings, [2] clock menu with regions", m.press("2", " ").View())
	show("settings, options on time", m.press("2", "j", "enter").View())
	show("settings 50x18 narrow", m.size(50, 18).View())

	// The board's pixels are Nerd Font glyphs, invisible in a plain dump:
	// the same boards as # and . show the font.
	ascii := func(b board) string {
		var sb strings.Builder
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				if b.at(x, y) {
					sb.WriteString("#")
				} else {
					sb.WriteString(".")
				}
			}
			sb.WriteString("\n")
		}
		return sb.String()
	}
	show("board 120x40 k=2 21:05 as text", ascii(paint(faceTall, one([]string{"21 05"}, 2), 120, 39)))
	show("board 200x60 k=1 the whole font", ascii(paint(faceTall, one([]string{"0123456789:-", "ABCDEFGHIJKLM", "NOPQRSTUVWXYZ"}, 1), 200, 59)))
	d := saver.NewDino(5, saver.RunnerTRex, saver.SceneGrass)
	d.Draw(76, 31)
	for i := 0; i < 60; i++ {
		d.Step()
	}
	show("dino 152x32 k=1 after 60 frames", ascii(paintScene(d.Draw(76, 31), 1, 152, 31)))
	for i := 0; i < 8; i++ {
		d.Step()
	}
	show("dino 152x32 k=1 after 68 frames", ascii(paintScene(d.Draw(76, 31), 1, 152, 31)))

	lk := testLock(t, "1234", nil)
	lk.now = func() time.Time { return at }
	show("lock 80x24", lk.View())
	lk2, _ := lk.step(tea.WindowSizeMsg{Width: 120, Height: 40})
	show("lock 120x40 with prompt", openPrompt(t, lk2).View())
	full := testLock(t, "1234", func(c *config.Config) { c.Profiles[0].Time = "HH MM SS"; c.Profiles[0].Date = "YYYY-MM-DD" })
	show("lock 80x24 HH MM SS + YYYY-MM-DD (degrades)", full.View())
	small, _ := testLock(t, "", nil).step(tea.WindowSizeMsg{Width: 40, Height: 12})
	show("lock 40x12 no PIN (plain text)", small.View())
}
