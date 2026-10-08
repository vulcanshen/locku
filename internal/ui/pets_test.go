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
// 2026-10-07): the outdoors' — the sky over a green field a row at a
// time, the ground's grass lit from the first frame, the trees' — the
// z's, and the cats' coats; a frame every 80 ms, and the cats go about.
func TestPetsLockWearsItsOwnColours(t *testing.T) {
	m := testLock(t, "1234", func(c *config.Config) {
		c.Profiles = []config.Profile{config.NewProfile("p", saver.KindPets)}
		c.Profile = "p"
	})
	if _, ok := m.game.(*saver.Pets); !ok {
		t.Fatalf("the game is %T", m.game)
	}
	if inks := m.inks(); len(inks) != 32 || inks[0] != lipgloss.Color("#7e9bbb") || inks[1] != lipgloss.Color("#4f7d36") || inks[2] != lipgloss.Color("#847260") || inks[7] != lipgloss.Color("#f6b06a") {
		t.Fatalf("inks %v", inks)
	}
	if s := m.shading(); s == nil || s.Looks[0].Ground[0] != "#7e9bbb" || s.Looks[0].Ground[len(s.Looks[0].Ground)-1] != "#1c2f1a" {
		t.Fatalf("no backdrop: %+v", s)
	}
	grass := 0
	for _, k := range m.shown.ink {
		if k == 1 {
			grass++
		}
	}
	if grass < m.shown.w {
		t.Fatalf("%d lit grass at the start: not even the ground", grass)
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

// On the settings screen the pets have what they are, how many and where
// to set, and nothing else (user, 2026-10-07; the animals and the scene
// 2026-10-08, in that order: what, how many, where): the animals only
// cats and the scene only outdoors for now, the count from one to five,
// three to begin with; the cursor on the one it is, written at once.
func TestPetsHaveAnimalsACountAndAScene(t *testing.T) {
	m := newTestApp(t).press("G", "k", "k", "k", "k") // above the custom saver
	if it := m.sideAt(); it.kind != sideSaver || saver.Kinds[it.ref] != saver.KindPets {
		t.Fatalf("the pets sit above the custom saver, not %+v", it)
	}
	v := m.View()
	if !strings.Contains(v, "cats outdoors") || !strings.Contains(v, "count") || !strings.Contains(v, "animals") || !strings.Contains(v, "scene") || strings.Contains(v, " bg ") || strings.Contains(v, "speed") {
		t.Errorf("the pets' [2]:\n%s", v)
	}
	m = m.press("n", "enter")
	it := m.sideAt()
	if it.kind != sideProfile || m.cfg.Profiles[it.ref] != (config.Profile{Name: "pets", Saver: saver.KindPets, Scene: "outdoor", Animals: "cats", Count: 3}) {
		t.Fatalf("the new profile: %+v", m.cfg.Profiles)
	}
	m = m.press("2", "j")
	if r := m.rowAt(); r.kind != rowAnimals || r.label != "animals" || r.value != "cats" {
		t.Fatalf("the row under the name: %+v", r)
	}
	if acts := m.actions(); len(acts) == 0 || acts[0].hint != "what they are" {
		t.Errorf("the animals' actions %+v", acts)
	}
	m = m.press("enter")
	if !m.options.isInteractive() || len(m.options.items) != 1 || m.options.items[0].label != "cats" || m.options.cursor != 0 {
		t.Fatalf("animals %+v", m.options.items)
	}
	m = m.press("enter", "j")
	if r := m.rowAt(); r.kind != rowCount || r.label != "count" || r.value != "3" {
		t.Fatalf("the row under the animals: %+v", r)
	}
	m = m.press("enter")
	if !m.options.isInteractive() || len(m.options.items) != 5 || m.options.items[0].label != "1" || m.options.items[m.options.cursor].label != "3" {
		t.Fatalf("options %+v", m.options.items)
	}
	m = m.press("j", "j", "enter")
	if got := m.cfg.Profiles[it.ref].Count; got != 5 || saved(t).Profiles[it.ref].Count != 5 {
		t.Errorf("chose %d", got)
	}
	m = m.press("j")
	if r := m.rowAt(); r.kind != rowScene || r.label != "scene" || r.value != "outdoor" {
		t.Fatalf("the row under the count: %+v", r)
	}
	if acts := m.actions(); len(acts) == 0 || acts[0].hint != "where they are" {
		t.Errorf("the scene's actions %+v", acts)
	}
	m = m.press("enter")
	if !m.options.isInteractive() || len(m.options.items) != 1 || m.options.items[0].label != "outdoor" {
		t.Fatalf("scenes %+v: the runner's are not the pets'", m.options.items)
	}
	m = m.press("enter")
	if b, _ := os.ReadFile(config.Path()); !strings.Contains(string(b), "count: 5") || !strings.Contains(string(b), "animals: cats") || !strings.Contains(string(b), "scene: outdoor") {
		t.Errorf("the file:\n%s", b)
	}
}
