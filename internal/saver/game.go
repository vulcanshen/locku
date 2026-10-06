package saver

import "time"

// A Game is a saver that moves rather than spells (2026-10-06): the dino
// run, and every saver after it that draws a picture of its own — the
// canvas steps it a frame at a time and draws each frame whole, scaled.
// Nothing is lettered over it.
type Game interface {
	// Room is the scene the game needs, in its own pixels, and how far
	// the canvas may scale it.
	Room() Room
	// Step moves the game one frame on.
	Step()
	// Draw is the frame at w × h of the game's own pixels: the whole
	// board, at the scale the canvas picked.
	Draw(w, h int) Scene
	// Next is when the next frame is due.
	Next(now time.Time) time.Time
}

// Room is the least scene a game is played in, W × H of its pixels, and
// the most the canvas scales it by. A game has no size setting (user,
// 2026-09-24): it is drawn at the largest scale up to Most that leaves it
// its room, or at 1, clipped, when none does.
type Room struct{ W, H, Most int }

// Inked is a game with colours of its own (user, 2026-10-06: a saver of
// many colours brings them, and has no bg / fg). Inks is them, "#rrggbb",
// the ground first; a scene's ink n wears the nth. A game that is not
// Inked lights in one ink, and the profile's bg / fg are its colours.
type Inked interface {
	Inks() []string
}

// Scene is one frame: a bitmap in the game's own pixels, row by row, each
// pixel an ink — 0 the ground, the others lit.
type Scene struct {
	W, H int
	Pix  []uint8
}

func newScene(w, h int) Scene { return Scene{W: w, H: h, Pix: make([]uint8, w*h)} }

// put inks the pixel at x, y, if it is in the scene.
func (s *Scene) put(x, y int, ink uint8) {
	if x >= 0 && x < s.W && y >= 0 && y < s.H {
		s.Pix[y*s.W+x] = ink
	}
}

func (s *Scene) set(x, y int) { s.put(x, y, 1) }

// blit lights a sprite with its top-left pixel at x, y, clipped.
func (s *Scene) blit(sp sprite, x, y int) {
	for dy, row := range sp {
		for dx := 0; dx < len(row); dx++ {
			if row[dx] == '#' {
				s.set(x+dx, y+dy)
			}
		}
	}
}
