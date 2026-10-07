package saver

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// emptyRoom is a room w × h with nothing in it yet, for the furniture to
// be put in by hand.
func emptyRoom(w, h int) *Pets {
	g := NewPets(1, 1)
	g.w, g.h, g.room = w, h, make([]uint8, w*h)
	return g
}

// roomArt is the furniture from x0 to x1, rows y0 to y1: '#' the wood,
// 'D' the dark wood.
func roomArt(g *Pets, x0, x1, y0, y1 int) []string {
	var out []string
	for y := y0; y <= y1; y++ {
		var b strings.Builder
		for x := x0; x <= x1; x++ {
			switch g.room[y*g.w+x] {
			case 0:
				b.WriteByte('.')
			case inkPetDark:
				b.WriteByte('D')
			case inkPetWood:
				b.WriteByte('#')
			default:
				b.WriteByte('?')
			}
		}
		out = append(out, b.String())
	}
	return out
}

func sameArt(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s:\n%s\nwant:\n%s", what, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// The boxes: two side by side and one on one of them, a step and a
// step, nine pixels high and fourteen across, outlined, the top one's
// foot the other's top; a ledge on each top a cat can stand on, two
// pixels over each end for a cat's middle.
func TestPetsHeapOfBoxes(t *testing.T) {
	g := emptyRoom(60, 35)
	g.putBoxes(2, petHeap)
	sameArt(t, "the heap", roomArt(g, 2, 29, 17, 33), []string{
		"..............##############",
		"..............#............#",
		"..............#............#",
		"..............#............#",
		"..............#............#",
		"..............#............#",
		"..............#............#",
		"..............#............#",
		"############################",
		"#............##............#",
		"#............##............#",
		"#............##............#",
		"#............##............#",
		"#............##............#",
		"#............##............#",
		"#............##............#",
		"############################",
	})
	want := []petPlace{
		{at: 24, x0: 2, x1: 15, lo: 6, hi: 12},
		{at: 16, x0: 16, x1: 29, lo: 20, hi: 26},
	}
	if !slices.Equal(g.places, want) {
		t.Errorf("ledges %+v", g.places)
	}
}

// The cat tree: a board fourteen across in wood on a post two thick,
// with its feet, in the dark wood, a step higher each; the outer side of
// the posts at the ends a climb, from the floor to the board's height.
func TestPetsCatTree(t *testing.T) {
	g := emptyRoom(60, 35)
	g.ledge(34, 0, 59)
	g.putTree(2, []int{12, 20})
	sameArt(t, "the tree", roomArt(g, 2, 29, 14, 33), []string{
		"..............##############",
		"....................DD......",
		"....................DD......",
		"....................DD......",
		"....................DD......",
		"....................DD......",
		"....................DD......",
		"....................DD......",
		"##############......DD......",
		"......DD............DD......",
		"......DD............DD......",
		"......DD............DD......",
		"......DD............DD......",
		"......DD............DD......",
		"......DD............DD......",
		"......DD............DD......",
		"......DD............DD......",
		"......DD............DD......",
		"......DD............DD......",
		".....DDDD..........DDDD.....",
	})
	var climbs []petPlace
	for _, p := range g.places {
		if p.climb {
			climbs = append(climbs, p)
		}
	}
	if want := []petPlace{{climb: true, at: 7, side: 1, lo: 21, hi: 33}, {climb: true, at: 24, side: -1, lo: 13, hi: 33}}; !slices.Equal(climbs, want) {
		t.Errorf("climbs %+v", climbs)
	}
	// Each climb's foot to the floor, its top to its board, there and
	// back: the first climb's from its middle, three over from its paws.
	if len(g.links) != 8 || g.links[0] != (petLink{petPoint{0, 6}, petPoint{2, 33}}) || g.links[2] != (petLink{petPoint{2, 21}, petPoint{1, 6}}) ||
		g.links[4] != (petLink{petPoint{0, 27}, petPoint{4, 33}}) || g.links[6] != (petLink{petPoint{4, 13}, petPoint{3, 26}}) {
		t.Errorf("links %+v", g.links)
	}
}

// A cat drawn facing left is the one facing right the other way about,
// its middle where it was, over what is behind it: its fur, its
// stripes, its chest and paws, its more white and its eye each in its
// coat's ink.
func TestPetsCatFacesBothWays(t *testing.T) {
	right, left := newScene(14, 11), newScene(14, 11)
	for i := range right.Pix {
		right.Pix[i], left.Pix[i] = inkPetWood, inkPetWood
	}
	inks := [5]uint8{20, 21, 22, 23, 24}
	petStand(&right, petSit[0], 7, 10, 1, inks)
	petStand(&left, petSit[0], 7, 10, -1, inks)
	show := func(sc Scene) []string {
		var out []string
		for y := 0; y < sc.H; y++ {
			var b strings.Builder
			for x := 0; x < sc.W; x++ {
				if k := sc.Pix[y*sc.W+x]; k >= 20 {
					b.WriteByte("fswxe"[k-20])
				} else {
					b.WriteByte('#')
				}
			}
			out = append(out, b.String())
		}
		return out
	}
	sameArt(t, "facing right", show(right), []string{
		"##############",
		"########f##f##",
		"########ffff##",
		"########ffef##",
		"########ffxx##",
		"#######ffww###",
		"######sfsww###",
		"######sfsxx###",
		"#####fsfsxx###",
		"###x#fsfsfxx##",
		"####ffffffww##",
	})
	sameArt(t, "facing left", show(left), []string{
		"##############",
		"###f##f#######",
		"###ffff#######",
		"###feff#######",
		"###xxff#######",
		"####wwff######",
		"####wwsfs#####",
		"####xxsfs#####",
		"####xxsfsf####",
		"###xxfsfsf#x##",
		"###wwffffff###",
	})
}

// The coats: the cats' own colours, no other, two each far apart (user,
// 2026-10-07) — light orange and burnt, white and grey, light grey-blue
// and slate, charcoal and white, the black-and-white the more white; the
// amber dark coffee, a little yellow, a little grey; the eyes a colour,
// not the ground nor the z's yellow.
func TestPetsCoats(t *testing.T) {
	want := [][5]string{
		{"#f6b06a", "#a8460c", "#f6b06a", "#f6b06a", "#7ed957"},
		{"#7a4a2a", "#e8b830", "#9a958e", "#7a4a2a", "#7ed957"},
		{"#f4f1ea", "#8c8a86", "#f4f1ea", "#f4f1ea", "#5fa8ff"},
		{"#9fb2c8", "#4e5d70", "#9fb2c8", "#9fb2c8", "#7ed957"},
		{"#55504c", "#55504c", "#f4f1ea", "#f4f1ea", "#7ed957"},
	}
	inks := NewPets(1, 5).Inks()
	for c, coat := range want {
		for k, ink := range petCoat(c) {
			if inks[ink] != coat[k] {
				t.Errorf("coat %d's %c is %s", c, "fswxe"[k], inks[ink])
			}
		}
	}
}

// Every room, whatever its size, narrow or wide, low or high, has its
// floor from wall to wall, a pole up each wall and shelves on it, a cat
// tree where it is high and wide enough, the floor by the poles clear,
// and every place in it is one a cat can get to from the floor; a cat's
// middle has room on each ledge, a cat sitting there fits under the top,
// and a shelf has its bracket, and room for a cat sitting under it.
func TestPetsRoomsAreWhole(t *testing.T) {
	for _, size := range [][2]int{{30, 22}, {30, 26}, {30, 60}, {31, 24}, {37, 32}, {40, 23}, {60, 35}, {65, 30}, {80, 22}, {80, 26}, {100, 49}} {
		for seed := range uint64(30) {
			g := NewPets(seed, 3)
			g.Draw(size[0], size[1])
			w, h := size[0], size[1]
			floor := h - 1
			for x := range w {
				if g.room[floor*w+x] != inkPetDark {
					t.Fatalf("%v seed %d: the floor is not whole at %d", size, seed, x)
				}
			}
			climbs := 0
			var shelves []petPlace
			brace := map[[2]int]bool{}
			for _, p := range g.places {
				if p.climb {
					climbs++
				} else if p.lo > p.hi || p.at-(petTall-1) < 0 {
					t.Fatalf("%v seed %d: a ledge %+v", size, seed, p)
				}
				if p.climb || p.at == floor-1 || p.x0 != 0 && p.x1 != w-1 {
					continue
				}
				// A shelf: as long as the shortest to the longest; its
				// bracket under it, a brace from its pole out to under it,
				// aslant, half the shelf past the pole; past that, dark under
				// it for a cat sitting.
				shelves = append(shelves, p)
				long := p.x1 - p.x0 + 1
				if long < petShort || long > g.sw {
					t.Fatalf("%v seed %d: a shelf %+v of %d", size, seed, p, long)
				}
				n := (long - petPole) / 2
				for i := 1; i <= n; i++ {
					bx := petPole + n - i
					if p.x0 != 0 {
						bx = w - 1 - bx
					}
					brace[[2]int{bx, p.at + 1 + i}] = true
					if g.room[(p.at+1+i)*w+bx] != inkPetDark {
						t.Fatalf("%v seed %d: no brace under %+v at %d, %d", size, seed, p, bx, p.at+1+i)
					}
				}
				for y := p.at + 2; y <= p.at+1+petTall; y++ {
					for x := max(p.x0, petPole); x <= min(p.x1, w-1-petPole); x++ {
						if g.room[y*w+x] != 0 && !brace[[2]int{x, y}] {
							t.Fatalf("%v seed %d: %d, %d under the shelf %+v is lit", size, seed, x, y, p)
						}
					}
				}
			}
			if climbs < 3 && h >= 26 && w-2*g.sw >= petBoard {
				t.Errorf("%v seed %d: no cat tree", size, seed)
			}
			// Shelves on both walls, one at the least, two on a high
			// room; and the floor under them clear but for them.
			for side, x0 := range []int{petPole, w - g.sw} {
				n := 0
				for _, p := range shelves {
					if p.x0 == 0 == (side == 0) {
						n++
					}
				}
				if n == 0 || n < 2 && h >= 46 {
					t.Errorf("%v seed %d: %d shelves on wall %d", size, seed, n, side)
				}
				for y := range floor {
					for x := x0; x < x0+g.sw-petPole; x++ {
						k := g.room[y*w+x]
						if k != 0 && !brace[[2]int{x, y}] && !slices.ContainsFunc(shelves, func(p petPlace) bool { return p.at+1 == y && p.x0 <= x && x <= p.x1 }) {
							t.Fatalf("%v seed %d: %d, %d under the shelves is lit", size, seed, x, y)
						}
					}
				}
			}
			if len(g.reach) != len(g.places)+1 {
				t.Errorf("%v seed %d: %d places, %d reached", size, seed, len(g.places), len(g.reach)-1)
			}
			// The poles: sisal, its rope aslant, a dark pixel a column on
			// down, every third row, but where its wall's shelves go
			// across; a cat climbs each by it.
			for _, x := range []int{0, 1, w - 2, w - 1} {
				rope, across := 0, 0
				for y := range floor {
					k := g.room[y*w+x]
					if k == inkPetRope || k == inkPetRopeDark {
						rope++
						lx := x
						if x >= petPole {
							lx = x - (w - petPole)
						}
						if dark := (y-lx+3)%3 == 0; dark != (k == inkPetRopeDark) {
							t.Fatalf("%v seed %d: the rope at %d, %d", size, seed, x, y)
						}
					}
				}
				for _, p := range g.places {
					if !p.climb && p.at != floor-1 && p.x0 <= x && x <= p.x1 {
						across++
					}
				}
				if rope != floor-across {
					t.Fatalf("%v seed %d: the pole at %d is %d of sisal, %d shelves across", size, seed, x, rope, across)
				}
			}
			if l, r := g.places[1], g.places[2]; !l.wall || l.at != petPole || l.side != -1 || !r.wall || r.at != w-1-petPole || r.side != 1 {
				t.Fatalf("%v seed %d: the walls' climbs %+v %+v", size, seed, l, r)
			}
			for y := range floor {
				for x := range w {
					pole := x < petPole || x >= w-petPole
					if !pole && !brace[[2]int{x, y}] && (x < petWall || x >= w-petWall) && y >= floor-6 && g.room[y*w+x] != 0 {
						t.Fatalf("%v seed %d: %d, %d by a wall is lit", size, seed, x, y)
					}
				}
			}
		}
	}
}

// The cats go about the room: a pixel a frame along a place at most,
// a jump from where its link starts to where it ends, inside the scene;
// each of them goes to a good few places, and between them they walk,
// run, climb, jump, sit, sleep with a z over them and hold on.
func TestPetsGoAbout(t *testing.T) {
	seen := map[string]int{}
	for seed := range uint64(4) {
		g := NewPets(seed, 3)
		w, h := 60, 35
		g.Draw(w, h)
		places := make([]map[int]bool, len(g.cats))
		for i := range places {
			places[i] = map[int]bool{}
		}
		for f := 0; f < 6000; f++ {
			before := slices.Clone(g.cats)
			g.Step()
			sc := g.Draw(w, h)
			if slices.Contains(sc.Pix, inkPetZ) {
				seen["z"]++
			}
			for i, c := range g.cats {
				b := before[i]
				pl := g.places[c.at.place]
				places[i][c.at.place] = true
				switch {
				case c.arc != nil:
					seen["jump"]++
					if x, y := c.arc[0][0], c.arc[0][1]; x < 0 || x >= w || y-(petLeap.h()-1) < 0 || y > h-2 {
						t.Fatalf("seed %d frame %d: cat %d jumps off the scene: %d, %d", seed, f, i, x, y)
					}
					if b.arc == nil {
						l := g.links[c.plan[0]]
						if c.at != l.from || len(c.plan) != len(b.plan) {
							t.Fatalf("seed %d frame %d: cat %d jumps from %v, not %v", seed, f, i, c.at, l.from)
						}
						// It faces the way it jumps, or as it did.
						x0, _ := g.spot(l.from)
						x1, _ := g.spot(l.to)
						if face := map[bool]int{true: 1, false: -1}[x1 > x0]; x1 != x0 && c.face != face || x1 == x0 && c.face != b.face {
							t.Fatalf("seed %d frame %d: cat %d jumps from %d to %d facing %d", seed, f, i, x0, x1, c.face)
						}
					}
					continue
				case b.arc != nil:
					if l := g.links[b.plan[0]]; c.at != l.to || !slices.Equal(c.plan, b.plan[1:]) {
						t.Fatalf("seed %d frame %d: cat %d lands at %v, not %v", seed, f, i, c.at, l.to)
					}
				case c.at.place != b.at.place || abs(c.at.pos-b.at.pos) > 1:
					t.Fatalf("seed %d frame %d: cat %d went from %v to %v", seed, f, i, b.at, c.at)
				}
				if c.at.pos < pl.lo || c.at.pos > pl.hi {
					t.Fatalf("seed %d frame %d: cat %d at %v, off %+v", seed, f, i, c.at, pl)
				}
				// A way goes on from where the cat is, a link from where the
				// one before ends, to the goal.
				at := c.at
				for _, l := range c.plan {
					if g.links[l].from.place != at.place {
						t.Fatalf("seed %d frame %d: cat %d's way %v from %v", seed, f, i, c.plan, c.at)
					}
					at = g.links[l].to
				}
				if at.place != c.goal.place {
					t.Fatalf("seed %d frame %d: cat %d's way %v does not get to %v", seed, f, i, c.plan, c.goal)
				}
				moved := c.at != b.at
				switch {
				case c.doing == petSitting:
					seen["sit"]++
				case c.doing == petSleeping:
					seen["sleep"]++
				case c.doing == petHolding:
					seen["hold"]++
				case moved && pl.climb:
					seen["climb"]++
				case moved && c.run:
					seen["run"]++
				case moved:
					seen["walk"]++
				}
			}
		}
		for i, p := range places {
			if len(p) < 4 {
				t.Errorf("seed %d: cat %d went to %d places", seed, i, len(p))
			}
		}
	}
	for _, k := range []string{"walk", "run", "climb", "jump", "sit", "sleep", "hold", "z"} {
		if seen[k] == 0 {
			t.Errorf("no cat did %s: %v", k, seen)
		}
	}
}

// A sleeping cat has a z over its head for two seconds of three, rising
// a pixel every six frames, on the side its head is.
func TestPetsSleepUnderAZ(t *testing.T) {
	g := NewPets(1, 1)
	g.Draw(60, 35)
	c := &g.cats[0]
	c.at, c.face, c.doing, c.rest, c.arc, c.plan = petPoint{0, 30}, 1, petSleeping, 1000, nil, nil
	zs := func() (n, top, left int) {
		sc := g.Draw(60, 35)
		top, left = sc.H, sc.W
		for i, k := range sc.Pix {
			if k == inkPetZ {
				n++
				top, left = min(top, i/sc.W), min(left, i%sc.W)
			}
		}
		return
	}
	for tick, want := range map[int][2]int{0: {24, 32}, 5: {24, 32}, 6: {23, 32}, 23: {21, 32}} {
		c.tick = tick
		if n, top, left := zs(); n != 8 || top != want[0] || left != want[1] {
			t.Errorf("tick %d: %d z pixels, top %d, left %d", tick, n, top, left)
		}
	}
	c.tick = 24
	if n, _, _ := zs(); n != 0 {
		t.Errorf("a z between two: %d", n)
	}
	c.tick, c.face = 0, -1
	if _, _, left := zs(); left != 25 {
		t.Errorf("facing left, the z at %d", left)
	}
}

// The same seed is the same room and the same cats, frame for frame.
func TestPetsIsTheSeeds(t *testing.T) {
	a, b := NewPets(9, 4), NewPets(9, 4)
	for f := range 300 {
		if !slices.Equal(a.Draw(60, 35).Pix, b.Draw(60, 35).Pix) {
			t.Fatalf("frame %d differs", f)
		}
		a.Step()
		b.Step()
	}
}

// A count it has is as many cats, the first of the coats in order;
// another is three. Too small a scene is dark, the least room has its
// floor, and a scene of a new size is a new room.
func TestPetsCountAndSize(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3, 4, 5, 6} {
		g := NewPets(2, n)
		g.Draw(60, 35)
		want := n
		if n < 1 || n > 5 {
			want = PetsDefault
		}
		var coats []int
		for _, c := range g.cats {
			coats = append(coats, c.coat)
		}
		if len(g.cats) != want || !slices.Equal(coats, []int{0, 1, 2, 3, 4}[:want]) {
			t.Errorf("count %d: coats %v", n, coats)
		}
	}
	g := NewPets(3, 3)
	for _, size := range [][2]int{{29, 22}, {30, 21}} {
		if sc := g.Draw(size[0], size[1]); slices.ContainsFunc(sc.Pix, func(k uint8) bool { return k != 0 }) {
			t.Errorf("%v is lit", size)
		}
		g.Step()
	}
	if sc := g.Draw(30, 22); sc.Pix[21*30] != inkPetDark {
		t.Error("the least room has no floor")
	}
	g.Draw(70, 30)
	if g.places[0].at != 28 || g.places[0].hi != 64 {
		t.Errorf("the floor of a new size: %+v", g.places[0])
	}
}

// Its room, its pace and its colours (user, 2026-10-07): a room's, the
// ground a dark umber, the furniture two woods, the poles sisal and its
// rope's lay, the z yellow, then the coats.
func TestPetsRoomPaceAndInks(t *testing.T) {
	g := NewPets(1, 3)
	if g.Room() != (Room{W: 30, H: 22, Most: 1}) {
		t.Errorf("room %+v", g.Room())
	}
	at := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	if d := g.Next(at).Sub(at); d != 80*time.Millisecond {
		t.Errorf("a frame every %v", d)
	}
	inks := g.Inks()
	if len(inks) != 31 || inks[0] != "#2b231e" || inks[inkPetDark] != "#6e6052" || inks[inkPetWood] != "#a39484" || inks[inkPetRope] != "#d6bc8a" || inks[inkPetRopeDark] != "#9c8158" || inks[inkPetZ] != "#ffff00" || inks[inkPetCoats] != "#f6b06a" {
		t.Errorf("inks %v", inks)
	}
}

// A cat jumps up no higher than a step of the tree, from beside what it
// jumps onto, and down off the end of what it is on, clear of it; never
// farther across than its reach. A jump goes over the higher of its ends
// and comes down where it is going, longer the farther.
func TestPetsJumps(t *testing.T) {
	for _, size := range [][2]int{{40, 23}, {60, 35}, {100, 49}} {
		for seed := range uint64(20) {
			g := NewPets(seed, 1)
			g.Draw(size[0], size[1])
			for _, l := range g.links {
				A, B := g.places[l.from.place], g.places[l.to.place]
				x0, y0 := g.spot(l.from)
				x1, y1 := g.spot(l.to)
				arc := g.leap(l)
				top := y0
				for _, p := range arc {
					top = min(top, p[1])
				}
				if arc[len(arc)-1] != [2]int{x1, y1} || top > min(y0, y1)-1 || len(arc) != max(4, 2+(abs(x1-x0)+abs(y1-y0))/3) {
					t.Fatalf("%v seed %d: the jump %+v is %v", size, seed, l, arc)
				}
				if A.climb || B.climb {
					// A wall's: as far across as a jump, and as high.
					if (A.wall || B.wall) && (abs(x1-x0) > 16 || abs(y1-y0) > 13) {
						t.Fatalf("%v seed %d: %+v is too far a jump", size, seed, l)
					}
					continue
				}
				if abs(x1-x0) > 16 || y0-y1 > 13 {
					t.Fatalf("%v seed %d: %+v is too far a jump", size, seed, l)
				}
				if y1 < y0 && x0+5 >= B.x0 && x0-6 <= B.x1 {
					t.Fatalf("%v seed %d: %+v jumps up from under %+v", size, seed, l, B)
				}
				if y1 >= y0 && x1+5 >= A.x0 && x1-6 <= A.x1 {
					t.Fatalf("%v seed %d: %+v comes down under %+v", size, seed, l, A)
				}
			}
		}
	}
}

// The way to somewhere is the quickest: from the floor by the cat tree
// to the top of it, up the post by the floor, onto the low board, along
// it and up onto the high one — not along the floor to jump up from it.
func TestPetsWayIsTheQuickest(t *testing.T) {
	g := emptyRoom(60, 35)
	g.ledge(34, 0, 59)
	g.putTree(2, []int{12, 20})
	g.join()
	plan, ok := g.way(petPoint{0, 6}, petPoint{3, 26})
	var hops []petLink
	for _, l := range plan {
		hops = append(hops, g.links[l])
	}
	want := []petLink{
		{petPoint{0, 6}, petPoint{2, 33}},  // onto the post
		{petPoint{2, 21}, petPoint{1, 6}},  // off it onto the low board
		{petPoint{1, 10}, petPoint{3, 20}}, // up onto the high one
	}
	if !ok || !slices.Equal(hops, want) {
		t.Errorf("the way: %+v", hops)
	}
	if plan, ok := g.way(petPoint{0, 30}, petPoint{0, 10}); !ok || len(plan) != 0 {
		t.Errorf("along the floor: %v", plan)
	}
}

// Where a cat goes is not by where another is, or is going.
func TestPetsKeepApart(t *testing.T) {
	g := NewPets(4, 2)
	g.Draw(60, 35)
	g.cats[0].at, g.cats[0].goal = petPoint{0, 30}, petPoint{3, g.places[3].lo}
	for range 300 {
		p := g.pick(1, 2)
		if p.place == 0 && abs(p.pos-30) < petW || p.place == 3 && abs(p.pos-g.places[3].lo) < petW {
			t.Fatalf("by the other cat: %v", p)
		}
	}
}

// A cat walks a pixel every other frame, runs one every frame, and
// climbs as it walks.
func TestPetsPace(t *testing.T) {
	g := emptyRoom(40, 23)
	g.ledge(22, 0, 39)
	g.climb(0, -1, 8)
	g.cats = make([]petCat, 1)
	c := &g.cats[0]
	for _, k := range []struct {
		at, goal petPoint
		run      bool
		frames   int
	}{
		{petPoint{0, 10}, petPoint{0, 20}, false, 20},
		{petPoint{0, 20}, petPoint{0, 10}, true, 10},
		{petPoint{1, 21}, petPoint{1, 11}, true, 20},
	} {
		*c = petCat{at: k.at, goal: k.goal, run: k.run, face: 1}
		f := 0
		for ; c.at != k.goal; f++ {
			g.stepCat(0)
		}
		if f != k.frames {
			t.Errorf("%v to %v run %v: %d frames", k.at, k.goal, k.run, f)
		}
	}
}

// A cat sitting lifts its tail for six frames of forty.
func TestPetsTail(t *testing.T) {
	g := NewPets(1, 1)
	g.Draw(60, 35)
	c := &g.cats[0]
	c.at, c.face, c.doing, c.rest, c.arc, c.plan = petPoint{0, 30}, 1, petSitting, 1000, nil, nil
	for tick, sp := range map[int]sprite{0: petSit[0], 33: petSit[0], 34: petSit[1], 39: petSit[1], 40: petSit[0]} {
		c.tick = tick
		want := newScene(60, 35)
		copy(want.Pix, g.room)
		petStand(&want, sp, 30, 33, 1, petCoat(c.coat))
		if !slices.Equal(g.Draw(60, 35).Pix, want.Pix) {
			t.Errorf("tick %d: not the sprite", tick)
		}
	}
}

// What the floor has over goes into the gaps and at the ends, all of it.
func TestPetsSpread(t *testing.T) {
	g := NewPets(1, 1)
	for n := 1; n <= 3; n++ {
		for spare := range 31 {
			gaps := g.spread(n, spare)
			sum := 0
			for _, k := range gaps {
				sum += k
			}
			if len(gaps) != n+1 || sum != spare {
				t.Fatalf("%d things, %d over: %v", n, spare, gaps)
			}
		}
	}
}

// A wall's shelves are high and low, long and short, at random (user,
// 2026-10-07): from room to room the top one is at more than a few
// heights, the shelves of more than a few lengths, and the two walls are
// not the one wall twice.
func TestPetsShelvesVary(t *testing.T) {
	tops, lengths := map[int]bool{}, map[int]bool{}
	alike := 0
	for seed := range uint64(30) {
		g := NewPets(seed, 1)
		g.Draw(100, 49)
		var walls [2][]int
		for _, p := range g.places {
			if p.climb || p.at == 47 || p.x0 != 0 && p.x1 != 99 {
				continue
			}
			side := 0
			if p.x0 != 0 {
				side = 1
			}
			walls[side] = append(walls[side], p.at)
			lengths[p.x1-p.x0+1] = true
		}
		tops[slices.Min(walls[0])] = true
		if slices.Equal(walls[0], walls[1]) {
			alike++
		}
	}
	if len(tops) < 4 || len(lengths) < 4 || alike > 5 {
		t.Errorf("tops %v, lengths %v, %d of 30 the walls alike", tops, lengths, alike)
	}
}
