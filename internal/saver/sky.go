package saver

import (
	"fmt"
	"strconv"
)

// The runner's background (user, 2026-10-07): its own, not the
// profile's bg / fg — day, night, or the two by turns, time-shifting,
// which goes round a day in three minutes by the clock: a minute of day,
// a minute of dusk, a minute of night, day at the hour and every third
// minute after. Each is a sky, a gradient from the top of the board to
// the bottom (user, the same day: never one colour), with a colour for
// the runner and its world that stands out on it; time-shifting changes
// from one to the next at once, with no fade.
const (
	BackgroundDay          = "day"
	BackgroundNight        = "night"
	BackgroundTimeShifting = "time-shifting"
)

// Backgrounds are the runner's backgrounds, time-shifting the default.
var Backgrounds = []string{BackgroundDay, BackgroundNight, BackgroundTimeShifting}

// sky is a background: the colours its gradient passes through, top to
// bottom and evenly apart, and the colour the runner is drawn in.
type sky struct {
	stops []string
	fg    string
}

var (
	// Day: white, as the user asked, a pale blue at the top; the runner
	// in surface0, the night's ground.
	daySky = sky{[]string{"#cfe8ff", "#ffffff"}, "#313244"}
	// Dusk: the sunset, catppuccin's mauve, red, peach and yellow; the
	// runner as by day.
	duskSky = sky{[]string{"#cba6f7", "#f38ba8", "#fab387", "#f9e2af"}, "#313244"}
	// Night: the ground and gold the runner had (user: as now), the
	// ground darker above it and lighter below.
	nightSky = sky{[]string{"#1e1e2e", "#313244", "#45475a"}, "#f2b753"}
)

// sky is the runner's background at this moment.
func (d *Dino) sky() sky {
	switch d.background {
	case BackgroundDay:
		return daySky
	case BackgroundNight:
		return nightSky
	}
	return [3]sky{daySky, duskSky, nightSky}[d.now().Minute()%3]
}

// Inks are the ground — the top of the sky — and the runner's colour.
func (d *Dino) Inks() []string {
	s := d.sky()
	return []string{s.stops[0], s.fg}
}

// Ground is the sky a row, rows of them top to bottom: each row the mix
// of the two stops it falls between.
func (d *Dino) Ground(rows int) []string {
	s := d.sky()
	out := make([]string, rows)
	for y := range out {
		at := 0.0
		if rows > 1 {
			at = float64(y) * float64(len(s.stops)-1) / float64(rows-1)
		}
		i := min(int(at), len(s.stops)-2)
		out[y] = mix(s.stops[i], s.stops[i+1], at-float64(i))
	}
	return out
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
