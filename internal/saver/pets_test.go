package saver

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// emptyRoom is the outdoors w × h with nothing in it yet, for the trees
// and the stumps — and the ground's ledge, if a test wants it — to be
// put in by hand.
func emptyRoom(w, h int) *Pets {
	g := NewPets(1, 1)
	g.w, g.h, g.room = w, h, make([]uint8, w*h)
	return g
}

// roomArt is what is put in, from x0 to x1, rows y0 to y1: 'g' the
// grass, 'B' the bark, 'L' the leaves, 'l' their shade, 'c' a stump's
// cut top.
func roomArt(g *Pets, x0, x1, y0, y1 int) []string {
	var out []string
	for y := y0; y <= y1; y++ {
		var b strings.Builder
		for x := x0; x <= x1; x++ {
			b.WriteByte(".gBLlc?"[min(int(g.room[y*g.w+x]), 6)])
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

// A stump: its sides bark, filled, its top the wood cut, its roots spread
// a pixel either side at the foot; its top a ledge, two pixels over each
// end for a cat's middle.
func TestPetsStump(t *testing.T) {
	g := emptyRoom(40, 35)
	g.putStump(3, [2]int{12, 6})
	sameArt(t, "the stump", roomArt(g, 1, 16, 27, 34), []string{
		"................",
		"..cccccccccccc..",
		"..BBBBBBBBBBBB..",
		"..BBBBBBBBBBBB..",
		"..BBBBBBBBBBBB..",
		"..BBBBBBBBBBBB..",
		".BBBBBBBBBBBBBB.",
		"................",
	})
	if want := []petPlace{{at: 27, x0: 3, x1: 14, lo: 7, hi: 11}}; !slices.Equal(g.places, want) {
		t.Errorf("ledges %+v", g.places)
	}
}

// A tree: its trunk three across, from the middle of its crown to the
// ground, its roots spread at the foot; the crown an oval of leaves in a
// speckle of shade, over the trunk's top; its branches either side, each
// a ledge; the trunk's sides to climb, from the ground and the spot by it
// up to the highest branch that side — or, with none, to under the crown.
func TestPetsTree(t *testing.T) {
	g := emptyRoom(40, 35)
	g.ledge(34, 0, 39)
	g.putTree(2, &petTree{top: 2, crownH: 7, crownW: 12, branches: []petBranch{{23, -1, 10}, {17, 1, 12}}, left: 10, right: 12})
	sameArt(t, "the tree", roomArt(g, 2, 28, 2, 34), []string{
		"........LLLlLLL............",
		".......lLLLLLLlL...........",
		"......LLLLlLLLLLL..........",
		".....LlLLLLLLlLLLL.........",
		"......LLLlLLLLLLl..........",
		".......LLLLLlLLL...........",
		"........lLLLLLL............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBBBBBBBBBBBBBB..",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"BBBBBBBBBBBBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		"..........BBB..............",
		".........BBBBB.............",
		"...........................",
	})
	want := []petPlace{
		{at: 33, x0: 0, x1: 39, lo: 6, hi: 34},
		{at: 22, x0: 2, x1: 11, lo: 6, hi: 8},
		{at: 16, x0: 15, x1: 26, lo: 19, hi: 23},
		{climb: true, trunk: true, at: 11, lo: 22, hi: 33, side: 1},
		{climb: true, trunk: true, at: 15, lo: 16, hi: 33, side: -1},
	}
	if !slices.Equal(g.places, want) {
		t.Errorf("places %+v", g.places)
	}
	if want := []petLink{{petPoint{0, 8}, petPoint{3, 33}}, {petPoint{3, 33}, petPoint{0, 8}}, {petPoint{0, 18}, petPoint{4, 33}}, {petPoint{4, 33}, petPoint{0, 18}}}; !slices.Equal(g.links, want) {
		t.Errorf("links %+v", g.links)
	}
	g = emptyRoom(40, 35)
	g.ledge(34, 0, 39)
	g.putTree(2, &petTree{top: 8, crownH: 7, crownW: 12, branches: []petBranch{{23, -1, 10}}, left: 10, right: 6})
	if c := g.places[3]; !c.climb || c.side != -1 || c.lo != 14 {
		t.Errorf("the side with no branch: %+v", c)
	}
}

// A cat drawn facing left is the one facing right the other way about,
// its middle where it was, over what is behind it: its fur, its
// stripes, its chest and paws, its more white and its eye each in its
// coat's ink.
func TestPetsCatFacesBothWays(t *testing.T) {
	right, left := newScene(14, 11), newScene(14, 11)
	for i := range right.Pix {
		right.Pix[i], left.Pix[i] = inkPetBark, inkPetBark
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

// Every outdoors, whatever its size, narrow or wide, low or high, has its
// ground a line of grass from side to side, and a tree at the least (user,
// 2026-10-07: a narrow pane too); every place in it is one a cat can get
// to from the ground. A cat's middle has room on each ledge, and a cat
// sitting there fits under the top; a ledge over another has room for a
// cat sitting between them, but for a stump on the ground; a trunk is
// climbed either side, by the bark, from the ground up to a cat's height
// under the top at the most.
func TestPetsRoomsAreWhole(t *testing.T) {
	for _, size := range [][2]int{{30, 22}, {30, 26}, {30, 60}, {31, 24}, {37, 32}, {40, 23}, {60, 35}, {65, 30}, {80, 22}, {80, 26}, {100, 49}} {
		for seed := range uint64(30) {
			g := NewPets(seed, 3)
			g.Draw(size[0], size[1])
			w, h := size[0], size[1]
			floor := h - 1
			for x := range w {
				if g.room[floor*w+x] != inkPetGrass {
					t.Fatalf("%v seed %d: the ground is not whole at %d", size, seed, x)
				}
			}
			trunks := 0
			for i, p := range g.places {
				if p.climb {
					trunks++
					if !p.trunk || p.lo > p.hi || p.lo < petTop-1 || p.hi != floor-1 || g.room[(floor-2)*w+p.at+p.side] != inkPetBark {
						t.Fatalf("%v seed %d: a climb %+v", size, seed, p)
					}
					continue
				}
				if p.lo > p.hi || p.at-(petTall-1) < 0 {
					t.Fatalf("%v seed %d: a ledge %+v", size, seed, p)
				}
				for j, q := range g.places {
					if j == i || q.climb || q.at <= p.at || q.x1 < p.x0 || q.x0 > p.x1 || j == 0 && g.room[(p.at+1)*w+p.x0] == inkPetCut {
						continue
					}
					if q.at-p.at < petTall+1 {
						t.Fatalf("%v seed %d: %+v over %+v", size, seed, p, q)
					}
				}
			}
			if trunks < 2 {
				t.Errorf("%v seed %d: no tree", size, seed)
			}
			if len(g.reach) != len(g.places)+1 {
				t.Errorf("%v seed %d: %d places, %d reached", size, seed, len(g.places), len(g.reach)-1)
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
// another is three. Too small a scene is dark, the least outdoors has its
// ground, and a scene of a new size is laid out anew.
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
	if sc := g.Draw(30, 22); sc.Pix[21*30] != inkPetGrass {
		t.Error("the least outdoors has no ground")
	}
	g.Draw(70, 30)
	if g.places[0].at != 28 || g.places[0].hi != 64 {
		t.Errorf("the floor of a new size: %+v", g.places[0])
	}
}

// Its room, its pace and its colours (user, 2026-10-07): the outdoors' —
// the sky's at the top, the grass, the bark, the leaves and their shade,
// a stump's cut — the z yellow, then the coats.
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
	if len(inks) != 32 || inks[0] != "#1d2745" || inks[inkPetGrass] != "#4f7d36" || inks[inkPetBark] != "#847260" || inks[inkPetLeaf] != "#5e9140" || inks[inkPetLeafDark] != "#3f6c2c" || inks[inkPetCut] != "#b39b78" || inks[inkPetZ] != "#ffff00" || inks[inkPetCoats] != "#f6b06a" {
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
					// A trunk's: as far across as a jump, and as high.
					if (A.trunk || B.trunk) && (abs(x1-x0) > 16 || abs(y1-y0) > 13) {
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

// The way to somewhere is the quickest: from the ground by a tree to its
// high branch, up the near side of the trunk to the low branch's height
// and across onto the high one — not round to climb the far side.
func TestPetsWayIsTheQuickest(t *testing.T) {
	g := emptyRoom(40, 35)
	g.ledge(34, 0, 39)
	g.putTree(2, &petTree{top: 2, crownH: 7, crownW: 12, branches: []petBranch{{23, -1, 10}, {17, 1, 12}}, left: 10, right: 12})
	g.join()
	plan, ok := g.way(petPoint{0, 6}, petPoint{2, 23})
	var hops []petLink
	for _, l := range plan {
		hops = append(hops, g.links[l])
	}
	want := []petLink{
		{petPoint{0, 8}, petPoint{3, 33}},  // onto the trunk, by the near side
		{petPoint{3, 22}, petPoint{2, 19}}, // across onto the high branch
	}
	if !ok || !slices.Equal(hops, want) {
		t.Errorf("the way: %+v", hops)
	}
	if plan, ok := g.way(petPoint{0, 30}, petPoint{0, 10}); !ok || len(plan) != 0 {
		t.Errorf("along the ground: %v", plan)
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

// The outdoors is laid out anew, at random (user, 2026-10-07): from one
// to the next the trees are more or fewer, their tops as high as they
// are, their branches as long as they are, and a stump is there some of
// the time.
func TestPetsTreesVary(t *testing.T) {
	trees, tops, lengths, stumps := map[int]bool{}, map[int]bool{}, map[int]bool{}, 0
	for seed := range uint64(30) {
		g := NewPets(seed, 1)
		g.Draw(100, 49)
		n := 0
		for _, p := range g.places {
			switch {
			case p.climb && p.side == 1:
				// A tree: its top, the highest leaf over its trunk.
				n++
				y := 0
				for g.room[y*100+p.at+1] == 0 {
					y++
				}
				tops[y] = true
			case p.climb || p.at == 47:
			case g.room[(p.at+1)*100+p.x0] == inkPetCut:
				stumps++
			default:
				lengths[p.x1-p.x0+1] = true
			}
		}
		trees[n] = true
	}
	if len(trees) < 2 || len(tops) < 4 || len(lengths) < 4 || stumps == 0 {
		t.Errorf("trees %v, tops %v, branches %v, %d stumps", trees, tops, lengths, stumps)
	}
}

// The backdrop is a gradient a row at a time, the way the runner's dusk
// goes down the screen (user, 2026-10-07): the sky dark blue high and
// lighter low, over a green field lighter far and darker near; the one
// look all over, its inks the game's.
func TestPetsBackdrop(t *testing.T) {
	g := NewPets(1, 1)
	var _ Graded = g
	s := g.Shade(10, 51)
	l := s.Looks[0]
	if len(l.Ground) != 51 || l.Ground[0] != "#1d2745" || l.Ground[13] != "#2f4262" || l.Ground[25] != "#3f5a7c" || l.Ground[26] != "#4e6e46" || l.Ground[50] != "#1c2f1a" {
		t.Errorf("the backdrop %v", l.Ground)
	}
	if !slices.Equal(l.Inks, g.Inks()) || !slices.Equal(s.Looks[1].Ground, l.Ground) || s.Look(3, 7) != 0 {
		t.Errorf("the looks %+v", s.Looks)
	}
}
