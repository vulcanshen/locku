package saver

import (
	"fmt"
	"strconv"
	"time"
)

// The runner's background (user, 2026-10-07): its own, not the
// profile's bg / fg — day, night, or the two by turns, time-shifting,
// which goes round a day in three minutes by the clock: a minute of day,
// a minute of dusk, a minute of night, day at the hour and every third
// minute after. Each is a sky, a gradient from the top of the board to
// the bottom (user, the same day: never one colour), with a colour for
// the runner and its world that stands out on it. By day the sun is in
// it, by night the moon (user, the same day), up to the right where the
// clouds go by in front of them; at dusk the sun is setting, half of it
// on the horizon under where it was by day, the obstacles going by in
// front of it (user, the same day). Time-shifting turns from one part of
// the day to the next square by square, from right to left, the squares
// of a stretch in an order at random, over ten seconds (user, the same
// day: twenty was long): each square
// is the one sky or the other, the runner and the sun or the moon in it
// too, never a mix of the two (user, the same day: it changed all at
// once and night to day was sudden; then a fade of the whole board was
// smooth, but not what they had in mind).
//
// By night (user, the same day) the world — the ground, the obstacles,
// the clouds — is in the moon's grey light rather than gold, which was
// a lamp's; and the runner, dark in every sky, is a silhouette, a
// darker dark than the sky behind it but not the crust a terminal's own
// ground is often in, which runs into the gaps between its squares; an
// outline round it, a cooler white than the moon's — none under its
// feet on the ground, all round it in the air — and its eyes in gold.
// It is not gold all over, which came in all at once from dusk; the
// outline is not the moon's own white, which the runner at the top of a
// jump ran into; and the runner is not the dusk's dark, the night sky's
// own half way down, against which only its outline was seen.
const (
	BackgroundDay          = "day"
	BackgroundNight        = "night"
	BackgroundTimeShifting = "time-shifting"
)

// Backgrounds are the runner's backgrounds, time-shifting the default.
var Backgrounds = []string{BackgroundDay, BackgroundNight, BackgroundTimeShifting}

// skyTurn is how long time-shifting takes from one part of the day to
// the next; skyScatter is how much of it the squares of one column
// spread over, at random.
const (
	skyTurn    = 10 * time.Second
	skyScatter = 0.3
)

// The inks of the runner's scene past the ground and the world's: the
// sun, the moon and the setting sun; the runner, its outline and its
// eyes; the clouds.
const (
	inkSun     uint8 = 2
	inkMoon    uint8 = 3
	inkSunset  uint8 = 4
	inkRunner  uint8 = 5
	inkOutline uint8 = 6
	inkEye     uint8 = 7
	inkCloud   uint8 = 8
)

// sky is a background: the colours its gradient passes through, top to
// bottom and evenly apart, the colour the world is drawn in — the
// ground line, the obstacles — the runner's, its outline's and its
// eyes' ("" for none), the clouds', and whether the sun, the moon or the
// setting sun is in it.
type sky struct {
	stops                []string
	fg                   string
	runner, outline, eye string
	cloud                string
	sun, moon, sunset    bool
}

var (
	// Day: white, as the user asked, a blue at the top; the world and the
	// runner in surface0, the night's ground; the sun. The clouds white
	// (user, 2026-10-08: they were the world's dark; a grey edge round
	// them, as the night outlines the runner, was a mistake), and the
	// blue at the top deep enough for them to show on (user, the same
	// day: it was a pale #cfe8ff, as light as they are).
	daySky = sky{stops: []string{"#9ccfff", "#ffffff"}, fg: "#313244", runner: "#313244", cloud: "#ffffff", sun: true}
	// Dusk: the sunset, catppuccin's mauve, red, peach and yellow; the
	// world and the runner as by day; the setting sun. The clouds white
	// (user, 2026-10-08).
	duskSky = sky{stops: []string{"#cba6f7", "#f38ba8", "#fab387", "#f9e2af"}, fg: "#313244", runner: "#313244", cloud: "#ffffff", sunset: true}
	// Night: the ground the runner had (user: as now), darker above it
	// and lighter below; the world in moonlight, catppuccin's overlay1,
	// a cool grey — darker than the moon and the runner's outline, which
	// stand out from it, lighter than all the sky (user, 2026-10-07: it
	// was the gold the runner had, a lamp's); the runner in base, darker
	// than the sky it runs against, a step lighter than crust — in
	// crust, the terminal's ground, its squares ran into one blot (user,
	// the same day) — with its outline white (user, the same day: it was
	// the gold) — catppuccin's text, a cool white, the moon's warm — and
	// its eyes in the gold, the one warm light; the moon. The clouds in
	// the world's moonlight.
	nightSky = sky{stops: []string{"#1e1e2e", "#313244", "#45475a"}, fg: "#7f849c", runner: "#1e1e2e", outline: "#cdd6f4", eye: "#f2b753", cloud: "#7f849c", moon: true}
)

// The sun, a yellow deep enough to show on the pale sky — catppuccin's
// are pale, and latte's was orange (user, 2026-10-07: a little more
// yellow) — and the moon, rosewater, a crescent: each seven pixels
// square. The setting sun is the top of the sun, latte's peach so it
// shows on the yellow low in the dusk.
const (
	sunColour    = "#f5c211"
	moonColour   = "#f5e0dc"
	sunsetColour = "#fe640b"
)

var (
	sunArt = sprite{
		"..###..",
		".#####.",
		"#######",
		"#######",
		"#######",
		".#####.",
		"..###..",
	}
	// The moon, as the user drew it (2026-10-07): the round of the sun
	// along its left and its bottom, its inner edge a diagonal, its
	// horns reaching up and to the right (a C as thick all the way round
	// looked an oval; the sun with a disc taken out had blunt horns).
	moonArt = sprite{
		"..##...",
		".##....",
		"##.....",
		"###...#",
		"####.##",
		".#####.",
		"..###..",
	}
	sunsetArt = sunArt[:4]
)

// at is the sky's colour p of the way down, 0 the top and 1 the bottom:
// the mix of the two stops it falls between.
func (s sky) at(p float64) string {
	at := p * float64(len(s.stops)-1)
	i := min(int(at), len(s.stops)-2)
	return mix(s.stops[i], s.stops[i+1], at-float64(i))
}

// look is the sky on a board rows tall: its ground a row, and its inks —
// the runner's, the sun's, the moon's and the setting sun's, and the
// clouds', each "" when it is not in this sky, the ground there.
func (s sky) look(rows int) Look {
	l := Look{Ground: make([]string, rows), Inks: []string{s.stops[0], s.fg, "", "", "", s.runner, s.outline, s.eye, s.cloud}}
	for y := range l.Ground {
		p := 0.0
		if rows > 1 {
			p = float64(y) / float64(rows-1)
		}
		l.Ground[y] = s.at(p)
	}
	if s.sun {
		l.Inks[inkSun] = sunColour
	}
	if s.moon {
		l.Inks[inkMoon] = moonColour
	}
	if s.sunset {
		l.Inks[inkSunset] = sunsetColour
	}
	return l
}

// skies is the runner's sky at this moment: the one it is turning from,
// the one it is turning to, and how far, 1 when it is there. Day and
// night are themselves; time-shifting is the part of the day the clock
// is in, turning from the one before.
func (d *Dino) skies() (from, to sky, f float64) {
	switch d.background {
	case BackgroundDay:
		return daySky, daySky, 1
	case BackgroundNight:
		return nightSky, nightSky, 1
	}
	now := d.now()
	into := time.Duration(now.Minute()%3)*time.Minute + time.Duration(now.Second())*time.Second + time.Duration(now.Nanosecond())
	part := int(into / time.Minute)
	parts := [3]sky{daySky, duskSky, nightSky}
	f = min(1, float64(into-time.Duration(part)*time.Minute)/float64(skyTurn))
	return parts[(part+2)%3], parts[part], f
}

// turnsAt is how far into a turn of the sky square x, y of a board cols
// wide turns: the rightmost column first, the leftmost last, the squares
// of a column scattered at random — the same each time, so a square
// turns once and stays.
func turnsAt(x, y, cols int) float64 {
	h := uint32(x)*0x9e3779b1 ^ uint32(y)*0x85ebca6b
	h ^= h >> 15
	h *= 0x2c1b3c6d
	h ^= h >> 12
	scatter := float64(h&0xffff) / 0x10000
	along := float64(cols-1-x) / float64(max(1, cols-1))
	return along*(1-skyScatter) + scatter*skyScatter
}

// Shade is the runner's board, cols × rows, square by square: the sky
// it is turning from and the one it is turning to, and which a square is
// in.
func (d *Dino) Shade(cols, rows int) Shading {
	from, to, f := d.skies()
	return Shading{
		Looks: [2]Look{from.look(rows), to.look(rows)},
		Look: func(x, y int) int {
			if f > turnsAt(x, y, cols) {
				return 1
			}
			return 0
		},
	}
}

// Inks are the sky it is in, or turning to: the ground at the top, the
// runner's colour, and the sun's, the moon's and the setting sun's —
// the ground at the top for any not in it.
func (d *Dino) Inks() []string {
	_, to, _ := d.skies()
	inks := to.look(1).Inks
	for k, c := range inks {
		if c == "" {
			inks[k] = inks[0]
		}
	}
	return inks
}

// shows says whether the sun — or the moon, or the setting sun, as in
// says — is in the sky turning or turned to, or still in some of the
// one it is turning from.
func shows(from, to sky, f float64, in func(sky) bool) bool {
	return in(from) && f < 1 || in(to) && f > 0
}

func hasSun(s sky) bool     { return s.sun }
func hasMoon(s sky) bool    { return s.moon }
func hasSunset(s sky) bool  { return s.sunset }
func hasOutline(s sky) bool { return s.outline != "" }

// rim is the outline round a sprite — every pixel about it, corners
// too, that is not of it, under its feet as well: on the ground they are
// on the ground line, and the outline goes on the sky alone (user,
// 2026-10-07: none under the feet on the ground, but under them in the
// air) — and its eyes, the pixels marked 'e'. (A dark pixel the body
// closes round is not always an eye: between the T-Rex's legs, mid
// stride, is one; and the ghost's eyes are open at a corner.)
func rim(sp sprite) (edge, eyes [][2]int) {
	at := func(x, y int) byte {
		if y >= 0 && y < len(sp) && x >= 0 && x < len(sp[y]) {
			return sp[y][x]
		}
		return '.'
	}
	for y := -1; y <= len(sp); y++ {
		for x := -1; x <= sp.w(); x++ {
			switch at(x, y) {
			case '#':
				continue
			case 'e':
				eyes = append(eyes, [2]int{x, y})
				continue
			}
			for _, d := range [8][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}} {
				if at(x+d[0], y+d[1]) == '#' {
					edge = append(edge, [2]int{x, y})
					break
				}
			}
		}
	}
	return edge, eyes
}

// sunAt and moonAt are where the sun and the moon sit, their top-left
// pixels: up to the right, the sun a little left of the moon, so as one
// goes and the other comes, both are seen. The moon is the further from
// the runner — on the narrowest scene eight pixels from its outline at
// the top of a jump — so the two do not run together (user,
// 2026-10-07: it was a pixel off); the sun, by day, has no outline to
// run into.
func (d *Dino) sunAt() (int, int)  { return d.w - d.w/3 - sunArt.w(), 1 }
func (d *Dino) moonAt() (int, int) { return d.w - d.w/6 - moonArt.w(), 1 }

// sunsetAt is where the setting sun sits: under the sun, on the ground.
func (d *Dino) sunsetAt() (int, int) {
	x, _ := d.sunAt()
	return x, d.groundY() - sunsetArt.h()
}

// drawLights draws the sun, the moon and the setting sun where they are
// in a sky at all — in the squares of a sky without one, its ink is the
// ground.
func (d *Dino) drawLights(sc *Scene) {
	from, to, f := d.skies()
	if shows(from, to, f, hasSun) {
		x, y := d.sunAt()
		sc.blitIn(sunArt, x, y, inkSun)
	}
	if shows(from, to, f, hasMoon) {
		x, y := d.moonAt()
		sc.blitIn(moonArt, x, y, inkMoon)
	}
	if shows(from, to, f, hasSunset) {
		x, y := d.sunsetAt()
		sc.blitIn(sunsetArt, x, y, inkSunset)
	}
}

// mix is the colour f of the way from "#rrggbb" a to b, channel by
// channel.
func mix(a, b string, f float64) string {
	ca, cb := channels(a), channels(b)
	var c [3]int
	for k := range c {
		c[k] = int(float64(ca[k]) + (float64(cb[k])-float64(ca[k]))*f + 0.5)
	}
	return fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2])
}

// channels is "#rrggbb" as its three channels.
func channels(hex string) [3]int {
	n, _ := strconv.ParseUint(hex[1:], 16, 32)
	return [3]int{int(n >> 16 & 0xff), int(n >> 8 & 0xff), int(n & 0xff)}
}
