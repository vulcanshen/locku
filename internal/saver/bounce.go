package saver

import (
	"math/rand/v2"
	"time"
)

// The bouncing box (user, 2026-09-27, made 2026-10-06): the old video
// recorder's screensaver — a box drifts across the screen, and every
// edge it meets sends it back the other way in another colour. It has
// the time in it, so it is a clock too (user, 2026-10-06), and it
// moves, so nothing stays lit in one place. Run into a corner, edge and
// edge at once, it flashes through every colour it has: the thing
// everyone watching it waits for.
//
// Its colours are its own (user, 2026-10-06: a saver of many colours
// brings them, and has no bg / fg). Its settings are its speed, picked
// by name as the snake's is, and its time, HH:MM or HH:MM:SS, with
// colons — it has no size to keep small (user, 2026-10-07; it had none,
// and HH MM).

const KindBounce = "bounce"

// The bouncing box's times (user, 2026-10-07): hours and minutes, or the
// seconds too.
const (
	BounceTimeHM  = "HH:MM"
	BounceTimeHMS = "HH:MM:SS"
)

// BounceTimes are the bouncing box's times, HH:MM the default.
var BounceTimes = []string{BounceTimeHM, BounceTimeHMS}

// bounceSpeeds are its speeds in pixels a second, a pixel each way a
// frame: normal ten, as it always went, and each about 1.4 times the one
// before, as the snake's; twenty-eight at most, short of the thirty
// frames a second the lock draws at most.
var bounceSpeeds = map[string]int{SpeedSlow: 7, SpeedNormal: 10, SpeedFast: 14, SpeedVeryFast: 20, SpeedSuperFast: 28}

const (
	bounceBorder = 2 // the frame's width (user, 2026-10-07: thicker; it was a pixel)
	bouncePad    = 2 // dark pixels between the frame and the time
	bounceRooms  = 3 // the box is about a third of the screen each way
	bounceMost   = 4 // and drawn at most four times over
)

// Bounce is one box on its way.
type Bounce struct {
	rng    *rand.Rand
	now    func() time.Time
	spell  func(string) []string
	bw, bh int // the box, frame and all
	w, h   int // the scene as last drawn; nothing moves before the first draw
	x, y   int // the box's top-left
	dx, dy int // a pixel a frame, each way
	ink    int // the box's colour, in ownColours
	flash  int // frames of a corner's flash still to come
	frame  time.Duration
	layout string // the time, as Go lays it out
}

// NewBounce is a box from its first frame, at a speed out of Speeds and
// with a time out of BounceTimes — normal and HH:MM for any other. now
// is the time it shows; spell sets a line in the board's pixel font, as
// rows of '#' and '.'. The same seed is the same box.
func NewBounce(seed uint64, now func() time.Time, spell func(string) []string, speed, times string) *Bounce {
	n, ok := bounceSpeeds[speed]
	if !ok {
		n = bounceSpeeds[SpeedNormal]
	}
	b := &Bounce{
		rng:    rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
		now:    now,
		spell:  spell,
		frame:  time.Second / time.Duration(n),
		layout: "15:04",
	}
	if times == BounceTimeHMS {
		b.layout = "15:04:05"
	}
	t := spell(time.Time{}.Format(b.layout))
	b.bw, b.bh = len(t[0])+2*(bouncePad+bounceBorder), len(t)+2*(bouncePad+bounceBorder)
	b.ink = b.rng.IntN(len(ownColours))
	b.dx, b.dy = 1-2*b.rng.IntN(2), 1-2*b.rng.IntN(2)
	return b
}

// Room is a scene three boxes wide and high.
func (b *Bounce) Room() Room {
	return Room{W: bounceRooms * b.bw, H: bounceRooms * b.bh, Most: bounceMost}
}

// Next is when the next frame is due.
func (b *Bounce) Next(now time.Time) time.Time { return now.Add(b.frame) }

// Inks are the ground and the box's colours.
func (b *Bounce) Inks() []string { return ownInks() }

// Step moves the box a pixel each way. An edge it reaches turns it back
// and changes its colour; two at once, a corner, start the flash.
func (b *Bounce) Step() {
	if b.w == 0 {
		return
	}
	if b.flash > 0 {
		b.flash--
		b.ink = (b.ink + 1) % len(ownColours)
	}
	var hitX, hitY bool
	b.x, b.dx, hitX = bounceAxis(b.x, b.dx, b.w-b.bw)
	b.y, b.dy, hitY = bounceAxis(b.y, b.dy, b.h-b.bh)
	switch {
	case hitX && hitY:
		b.flash = 2 * len(ownColours)
	case hitX || hitY:
		b.ink = otherColour(b.rng, b.ink)
	}
}

// bounceAxis moves p a pixel along d inside 0 … span, and says whether
// it reached an end, where it turns. No span — the box as wide as the
// scene or wider — is no movement that way.
func bounceAxis(p, d, span int) (int, int, bool) {
	if span <= 0 {
		return 0, d, false
	}
	switch p += d; {
	case p <= 0:
		return 0, 1, true
	case p >= span:
		return span, -1, true
	}
	return p, d, false
}

// place puts the box anywhere on a new scene, or, on a scene of a new
// size, where it was as near as it fits.
func (b *Bounce) place(w, h int) {
	if b.w == 0 {
		b.x, b.y = b.rng.IntN(max(1, w-b.bw+1)), b.rng.IntN(max(1, h-b.bh+1))
	}
	b.w, b.h = w, h
	b.x, b.y = min(b.x, max(0, w-b.bw)), min(b.y, max(0, h-b.bh))
}

// Draw is the frame at w × h pixels: the box's frame, two pixels
// thick, and the time in it, in the box's colour.
func (b *Bounce) Draw(w, h int) Scene {
	if w != b.w || h != b.h {
		b.place(w, h)
	}
	sc := newScene(w, h)
	ink := uint8(1 + b.ink)
	for k := 0; k < bounceBorder; k++ {
		for i := 0; i < b.bw; i++ {
			sc.put(b.x+i, b.y+k, ink)
			sc.put(b.x+i, b.y+b.bh-1-k, ink)
		}
		for j := 0; j < b.bh; j++ {
			sc.put(b.x+k, b.y+j, ink)
			sc.put(b.x+b.bw-1-k, b.y+j, ink)
		}
	}
	t := b.spell(b.now().Format(b.layout))
	ox, oy := b.x+(b.bw-len(t[0]))/2, b.y+bouncePad+bounceBorder
	for dy, row := range t {
		for dx := 0; dx < len(row); dx++ {
			if row[dx] == '#' {
				sc.put(ox+dx, oy+dy, ink)
			}
		}
	}
	return sc
}
