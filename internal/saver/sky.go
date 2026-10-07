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
// front of it (user, the same day). Time-shifting fades
// each part of the day in from the one before over its first twenty
// seconds — the sky, the runner, the sun and the moon (user, the same
// day: they changed at once, and night to day was sudden).
const (
	BackgroundDay          = "day"
	BackgroundNight        = "night"
	BackgroundTimeShifting = "time-shifting"
)

// Backgrounds are the runner's backgrounds, time-shifting the default.
var Backgrounds = []string{BackgroundDay, BackgroundNight, BackgroundTimeShifting}

// skyFade is how long time-shifting takes from one part of the day to
// the next.
const skyFade = 20 * time.Second

// The inks of the runner's scene past the ground and the runner's: the
// sun, the moon and the setting sun.
const (
	inkSun    uint8 = 2
	inkMoon   uint8 = 3
	inkSunset uint8 = 4
)

// sky is a background: the colours its gradient passes through, top to
// bottom and evenly apart, the colour the runner is drawn in, and
// whether the sun, the moon or the setting sun is in it.
type sky struct {
	stops             []string
	fg                string
	sun, moon, sunset bool
}

var (
	// Day: white, as the user asked, a pale blue at the top; the runner
	// in surface0, the night's ground; the sun.
	daySky = sky{stops: []string{"#cfe8ff", "#ffffff"}, fg: "#313244", sun: true}
	// Dusk: the sunset, catppuccin's mauve, red, peach and yellow; the
	// runner as by day; the setting sun.
	duskSky = sky{stops: []string{"#cba6f7", "#f38ba8", "#fab387", "#f9e2af"}, fg: "#313244", sunset: true}
	// Night: the ground and gold the runner had (user: as now), the
	// ground darker above it and lighter below; the moon.
	nightSky = sky{stops: []string{"#1e1e2e", "#313244", "#45475a"}, fg: "#f2b753", moon: true}
)

// The sun, catppuccin-latte's yellow so it shows on the pale sky, and
// the moon, rosewater, a crescent: each seven pixels square. The setting
// sun is the top of the sun, latte's peach so it shows on the yellow low
// in the dusk.
const (
	sunColour    = "#df8e1d"
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
	moonArt = sprite{
		"..###..",
		".##....",
		"##.....",
		"##.....",
		"##.....",
		".##....",
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

// skies is the runner's sky at this moment: the one it is fading from,
// the one it is fading to, and how far, 1 when it is there. Day and
// night are themselves; time-shifting is the part of the day the clock
// is in, fading in from the one before.
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
	f = min(1, float64(into-time.Duration(part)*time.Minute)/float64(skyFade))
	return parts[(part+2)%3], parts[part], f
}

// skyAt is the sky p of the way down as it fades from one to the other.
func skyAt(from, to sky, f, p float64) string { return mix(from.at(p), to.at(p), f) }

// shows is how much of the sun — or the moon, as in says — shows: all of
// it in its own sky, less and less as the sky fades out of it, more and
// more as it fades in.
func shows(from, to sky, f float64, in func(sky) bool) float64 {
	w := 0.0
	if in(from) {
		w += 1 - f
	}
	if in(to) {
		w += f
	}
	return w
}

func hasSun(s sky) bool    { return s.sun }
func hasMoon(s sky) bool   { return s.moon }
func hasSunset(s sky) bool { return s.sunset }

// sunAt and moonAt are where the sun and the moon sit, their top-left
// pixels: up to the right, the moon a little left of the sun, so as one
// fades out and the other in, both are seen.
func (d *Dino) sunAt() (int, int)  { return d.w - d.w/6 - sunArt.w(), 1 }
func (d *Dino) moonAt() (int, int) { return d.w - d.w/3 - moonArt.w(), 1 }

// sunsetAt is where the setting sun sits: under the sun, on the ground.
func (d *Dino) sunsetAt() (int, int) {
	x, _ := d.sunAt()
	return x, d.groundY() - sunsetArt.h()
}

// down is how far down the scene a row is, 0 the top and 1 the bottom —
// and before the first frame, with no scene yet, the top.
func (d *Dino) down(y int) float64 { return min(1, max(0, float64(y)/float64(max(1, d.h-1)))) }

// Inks are the ground — the top of the sky — the runner's colour, the
// sun's, the moon's and the setting sun's: each of the three in its
// colour as much as it shows, the rest the sky behind it.
func (d *Dino) Inks() []string {
	from, to, f := d.skies()
	_, sy := d.sunAt()
	_, my := d.moonAt()
	_, ty := d.sunsetAt()
	return []string{
		skyAt(from, to, f, 0),
		mix(from.fg, to.fg, f),
		mix(skyAt(from, to, f, d.down(sy+sunArt.h()/2)), sunColour, shows(from, to, f, hasSun)),
		mix(skyAt(from, to, f, d.down(my+moonArt.h()/2)), moonColour, shows(from, to, f, hasMoon)),
		mix(skyAt(from, to, f, d.down(ty+sunsetArt.h()/2)), sunsetColour, shows(from, to, f, hasSunset)),
	}
}

// Ground is the sky a row, rows of them top to bottom.
func (d *Dino) Ground(rows int) []string {
	from, to, f := d.skies()
	out := make([]string, rows)
	for y := range out {
		p := 0.0
		if rows > 1 {
			p = float64(y) / float64(rows-1)
		}
		out[y] = skyAt(from, to, f, p)
	}
	return out
}

// drawLights draws the sun, the moon and the setting sun where they
// show at all.
func (d *Dino) drawLights(sc *Scene) {
	from, to, f := d.skies()
	if shows(from, to, f, hasSun) > 0 {
		x, y := d.sunAt()
		sc.blitIn(sunArt, x, y, inkSun)
	}
	if shows(from, to, f, hasMoon) > 0 {
		x, y := d.moonAt()
		sc.blitIn(moonArt, x, y, inkMoon)
	}
	if shows(from, to, f, hasSunset) > 0 {
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
