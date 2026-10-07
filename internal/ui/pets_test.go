package ui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
)

// The pets are a game on the board in colours of their own (user,
// 2026-10-07): a room's — a dark umber, the woods, the floor lit from
// the first frame in the dark one — the z's, and the cats' coats; a frame
// every 80 ms, and the cats go about.
func TestPetsLockWearsItsOwnColours(t *testing.T) {
	m := testLock(t, "1234", func(c *config.Config) {
		c.Profiles = []config.Profile{config.NewProfile("p", saver.KindPets)}
		c.Profile = "p"
	})
	if _, ok := m.game.(*saver.Pets); !ok {
		t.Fatalf("the game is %T", m.game)
	}
	if inks := m.inks(); len(inks) != 31 || inks[0] != lipgloss.Color("#2b231e") || inks[1] != lipgloss.Color("#6e6052") || inks[2] != lipgloss.Color("#a39484") || inks[6] != lipgloss.Color("#f6b06a") {
		t.Fatalf("inks %v", inks)
	}
	wood := 0
	for _, k := range m.shown.ink {
		if k == 1 {
			wood++
		}
	}
	if wood < m.shown.w {
		t.Fatalf("%d lit dark wood at the start: not even the floor", wood)
	}
	if d := m.game.Next(at).Sub(at); d != 80*time.Millisecond {
		t.Errorf("a frame every %v", d)
	}
	before := m.shown.clone()
	same := true
	for i := 0; i < 60 && same; i++ {
		next, cmd := m.step(clockTickMsg{gen: m.tickGen})
		if cmd == nil || next.rev != nil {
			t.Fatal("a frame must schedule the next and never reveal")
		}
		m = next
		for j := range before.ink {
			if before.ink[j] != m.shown.ink[j] {
				same = false
			}
		}
	}
	if same {
		t.Error("no cat moved")
	}
}

// On the settings screen the pets have how many cats to set and nothing
// else (user, 2026-10-07): from one to five, three to begin with, the
// cursor on the one it is, written at once.
func TestPetsHaveTheirCount(t *testing.T) {
	m := newTestApp(t).press("G", "k", "k", "k", "k") // above the custom saver
	if it := m.sideAt(); it.kind != sideSaver || saver.Kinds[it.ref] != saver.KindPets {
		t.Fatalf("the pets sit above the custom saver, not %+v", it)
	}
	v := m.View()
	if !strings.Contains(v, "cats about a room") || !strings.Contains(v, "count") || strings.Contains(v, " bg ") || strings.Contains(v, "speed") {
		t.Errorf("the pets' [2]:\n%s", v)
	}
	m = m.press("n", "enter")
	it := m.sideAt()
	if it.kind != sideProfile || m.cfg.Profiles[it.ref] != (config.Profile{Name: "pets", Saver: saver.KindPets, Count: 3}) {
		t.Fatalf("the new profile: %+v", m.cfg.Profiles)
	}
	m = m.press("2", "j")
	if r := m.rowAt(); r.kind != rowCount || r.label != "count" || r.value != "3" {
		t.Fatalf("the row under the name: %+v", r)
	}
	m = m.press("enter")
	if !m.options.isInteractive() || len(m.options.items) != 5 || m.options.items[0].label != "1" || m.options.items[m.options.cursor].label != "3" {
		t.Fatalf("options %+v", m.options.items)
	}
	m = m.press("j", "j", "enter")
	if got := m.cfg.Profiles[it.ref].Count; got != 5 || saved(t).Profiles[it.ref].Count != 5 {
		t.Errorf("chose %d", got)
	}
	if b, _ := os.ReadFile(config.Path()); !strings.Contains(string(b), "count: 5") {
		t.Errorf("the file:\n%s", b)
	}
}
