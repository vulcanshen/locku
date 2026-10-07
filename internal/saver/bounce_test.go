package saver

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

var bounceAt = time.Date(2026, time.October, 6, 21, 5, 9, 0, time.UTC)

// fakeSpell sets a line as the board's 3x5 face would be wide — three
// pixels a glyph, one between glyphs, one for a space or a colon —
// lighting every column of a glyph, so a test can find the line in a
// frame.
func fakeSpell(said *string) func(string) []string {
	return func(line string) []string {
		*said = line
		var row strings.Builder
		for i, r := range line {
			if i > 0 {
				row.WriteByte('.')
			}
			switch r {
			case ' ':
				row.WriteByte('.')
			case ':':
				row.WriteByte('#')
			default:
				row.WriteString("###")
			}
		}
		rows := make([]string, 5)
		for i := range rows {
			rows[i] = row.String()
		}
		return rows
	}
}

func newTestBounce(seed uint64) (*Bounce, *string) {
	said := new(string)
	return NewBounce(seed, func() time.Time { return bounceAt }, fakeSpell(said), SpeedNormal, BounceTimeHM), said
}

// litBox is the lit pixels' bounding box and their one ink; ok is false
// when the frame has two inks.
func litBox(sc Scene) (x0, y0, x1, y1 int, ink uint8, ok bool) {
	x0, y0, x1, y1, ok = sc.W, sc.H, -1, -1, true
	for y := 0; y < sc.H; y++ {
		for x := 0; x < sc.W; x++ {
			k := sc.Pix[y*sc.W+x]
			if k == 0 {
				continue
			}
			if ink != 0 && k != ink {
				ok = false
			}
			ink = k
			x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
		}
	}
	return
}

// The box is the frame, 25 × 13 around the time, in one ink; it never
// leaves the scene, and an edge reached turns it back in another colour.
func TestBounceTurnsAtEveryEdgeInAnotherColour(t *testing.T) {
	for _, sz := range [][2]int{{76, 31}, {40, 23}, {50, 29}, {100, 59}, {26, 14}} {
		b, _ := newTestBounce(3)
		if b.bw != 25 || b.bh != 13 {
			t.Fatalf("box %dx%d", b.bw, b.bh)
		}
		turns := 0
		for i := 0; i < 3000; i++ {
			sc := b.Draw(sz[0], sz[1])
			x0, y0, x1, y1, ink, ok := litBox(sc)
			if !ok || x1-x0+1 != b.bw || y1-y0+1 != b.bh || x0 != b.x || y0 != b.y {
				t.Fatalf("%v frame %d: lit %d,%d–%d,%d one ink %v, box at %d,%d", sz, i, x0, y0, x1, y1, ok, b.x, b.y)
			}
			if int(ink) != 1+b.ink {
				t.Fatalf("%v frame %d: ink %d, colour %d", sz, i, ink, b.ink)
			}
			dx, dy, colour, flash := b.dx, b.dy, b.ink, b.flash
			b.Step()
			if b.dx != dx || b.dy != dy {
				turns++
				if b.ink == colour && b.flash == flash {
					t.Fatalf("%v frame %d: turned in the same colour", sz, i)
				}
			} else if b.ink != colour && flash == 0 {
				t.Fatalf("%v frame %d: changed colour without an edge", sz, i)
			}
		}
		if turns < 20 {
			t.Errorf("%v: %d turns in 3000 frames", sz, turns)
		}
	}
}

// A corner — two edges at once — flashes every colour twice over, a
// frame each.
func TestBounceCornerFlashes(t *testing.T) {
	b, _ := newTestBounce(1)
	w, h := b.bw+40, b.bh+41 // no other edge for the length of the flash
	b.Draw(w, h)
	b.x, b.y, b.dx, b.dy = 39, 40, 1, 1
	b.Step()
	if b.x != 40 || b.y != 41 || b.dx != -1 || b.dy != -1 || b.flash != 2*len(ownColours) {
		t.Fatalf("at %d,%d going %d,%d, flash %d", b.x, b.y, b.dx, b.dy, b.flash)
	}
	for i := 0; i < 2*len(ownColours); i++ {
		was := b.ink
		b.Draw(w, h)
		b.Step()
		if b.ink != (was+1)%len(ownColours) {
			t.Fatalf("flash frame %d: colour %d after %d", i, b.ink, was)
		}
	}
	if b.flash != 0 {
		t.Errorf("flash still %d", b.flash)
	}
}

// The time in the box is HH:MM, centred across, under the two pixels of
// the frame and two dark ones.
func TestBounceShowsTheTime(t *testing.T) {
	b, said := newTestBounce(5)
	sc := b.Draw(76, 31)
	if *said != "21:05" {
		t.Fatalf("spelled %q", *said)
	}
	ox, oy := b.x+4, b.y+4
	for y := 0; y < 5; y++ {
		var row strings.Builder
		for x := 0; x < 17; x++ {
			if sc.Pix[(oy+y)*sc.W+ox+x] != 0 {
				row.WriteByte('#')
			} else {
				row.WriteByte('.')
			}
		}
		if got := row.String(); got != "###.###.#.###.###" {
			t.Fatalf("time row %d: %s", y, got)
		}
	}
}

// The same seed is the same box; a resize keeps it where it was, as near
// as it fits.
func TestBounceIsTheSeedsAndKeepsItsPlace(t *testing.T) {
	a, _ := newTestBounce(9)
	c, _ := newTestBounce(9)
	for i := 0; i < 200; i++ {
		if !reflect.DeepEqual(a.Draw(76, 31), c.Draw(76, 31)) {
			t.Fatalf("frame %d differs", i)
		}
		a.Step()
		c.Step()
	}
	a.x, a.y = 50, 18
	a.Draw(40, 23)
	if a.x != 40-25 || a.y != 23-13 {
		t.Errorf("after a resize the box is at %d,%d", a.x, a.y)
	}
}

// A scene smaller than the box clips it and holds it still that way:
// no edge every frame, so no colour every frame.
func TestBounceInATinyScene(t *testing.T) {
	b, _ := newTestBounce(2)
	b.Draw(20, 30)
	colour := b.ink
	for i := 0; i < 100; i++ {
		b.Draw(20, 30)
		if b.Step(); b.x != 0 {
			t.Fatalf("x %d on a scene narrower than the box", b.x)
		}
	}
	if b.ink == colour {
		t.Log("colour unchanged after 100 frames: the box turned only up and down, if at all")
	}
	turns := 0
	for i := 0; i < 100; i++ {
		was := b.ink
		b.Step()
		if b.ink != was {
			turns++
		}
	}
	if turns > 20 {
		t.Errorf("%d colours in 100 frames: the narrow way is turning it", turns)
	}
}

// The box's speed is picked by name as the snake's is (user, 2026-10-07):
// seven, ten, fourteen, twenty and twenty-eight pixels a second, normal
// the ten it always went, any other normal; its time is HH:MM or
// HH:MM:SS, with colons, any other HH:MM, the box as wide as the time.
func TestBounceSpeedsAndTimes(t *testing.T) {
	now := time.Date(2026, time.October, 7, 11, 5, 0, 0, time.UTC)
	for _, c := range []struct {
		speed string
		moves int
	}{
		{SpeedSlow, 7}, {SpeedNormal, 10}, {SpeedFast, 14}, {SpeedVeryFast, 20}, {SpeedSuperFast, 28}, {"", 10}, {"12", 10},
	} {
		b := NewBounce(1, func() time.Time { return bounceAt }, fakeSpell(new(string)), c.speed, BounceTimeHM)
		if got := b.Next(now).Sub(now); got != time.Second/time.Duration(c.moves) {
			t.Errorf("speed %q: a move every %v", c.speed, got)
		}
	}
	if !reflect.DeepEqual(BounceTimes, []string{"HH:MM", "HH:MM:SS"}) {
		t.Errorf("times %v", BounceTimes)
	}
	for _, c := range []struct {
		times, said string
		w           int
	}{
		{BounceTimeHM, "21:05", 25}, {BounceTimeHMS, "21:05:09", 35}, {"HH MM", "21:05", 25}, {"", "21:05", 25},
	} {
		said := new(string)
		b := NewBounce(1, func() time.Time { return bounceAt }, fakeSpell(said), SpeedNormal, c.times)
		b.Draw(76, 31)
		if *said != c.said || b.bw != c.w || b.bh != 13 {
			t.Errorf("time %q: spelled %q, the box %dx%d", c.times, *said, b.bw, b.bh)
		}
	}
}

// The frame is two pixels thick (user, 2026-10-07: it was one), the two
// dark ones inside it all round the time.
func TestBounceFrameIsTwoThick(t *testing.T) {
	b, _ := newTestBounce(4)
	sc := b.Draw(76, 31)
	at := func(x, y int) bool { return sc.Pix[(b.y+y)*sc.W+b.x+x] != 0 }
	for k := 0; k < 4; k++ {
		for i := k; i < b.bw-k; i++ {
			for _, y := range []int{k, b.bh - 1 - k} {
				if at(i, y) != (k < 2) {
					t.Fatalf("ring %d, %d,%d lit %v", k, i, y, at(i, y))
				}
			}
		}
		for j := k; j < b.bh-k; j++ {
			for _, x := range []int{k, b.bw - 1 - k} {
				if at(x, j) != (k < 2) {
					t.Fatalf("ring %d, %d,%d lit %v", k, x, j, at(x, j))
				}
			}
		}
	}
}
