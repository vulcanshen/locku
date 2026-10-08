package saver

import (
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"time"
)

// The pets (user, 2026-10-07): cats outdoors, after vscode-pets but drawn
// our own. Each goes somewhere at random — along the grass, onto a stump,
// up a tree and out along a branch — walking there, or running when it
// is far, jumping up and down and climbing as the way asks; and there it
// sits a while, or curls up and sleeps, a z rising over it, or, up a
// trunk, holds on; then it goes on. The trees and the stumps are behind
// them: a cat goes in front of a stump, and of a branch as it climbs past
// it.
//
// The cats are five, the first as many as there are of them (user, the
// same day: cats come in a few colours, not any), each of two colours
// (user, the same day): an orange tabby, light orange striped dark; an
// amber, dark coffee striped yellow; a white striped grey; a grey-blue
// striped darker; and a black-and-white.
//
// Outdoors (user, the same day: they were in a room, with a cat tree,
// boxes, and shelves up the walls): the sky over a green field, a
// gradient down the screen as the runner's dusk is, dark enough for the
// cats to show; the ground a line of grass, and on it trees and stumps,
// laid out anew for each scene — a tree at the least, as many as the
// width holds. A tree is a trunk to climb either side of, branches from
// it either side by turns, a step up each, and a crown of leaves on top.
// A cat jumps up as high as a tree's step, and down from anywhere.
//
// Its settings are what animals, how many (user, the same day) and where
// (2026-10-08): the animals are cats and the scene the outdoors, the one
// choice of each for now, making room for more (user).

const KindPets = "pets"

// PetCounts are how many cats there may be; PetsDefault is the default.
var PetCounts = []int{1, 2, 3, 4, 5}

const PetsDefault = 3

// PetAnimals are what the pets may be, PetScenes where they may be; the
// first of each is the default (user, 2026-10-08: the one choice of each
// for now).
const (
	PetAnimalsCats  = "cats"
	PetSceneOutdoor = "outdoor"
)

var (
	PetAnimals = []string{PetAnimalsCats}
	PetScenes  = []string{PetSceneOutdoor}
)

const (
	petFrame  = 80 * time.Millisecond
	petW      = 12 // a cat walking, across; its middle six from its left
	petTall   = 10 // a cat sitting, high: the room it needs over where it sits
	petJump   = 13 // the highest a cat jumps, in pixels
	petReach  = 16 // the farthest it jumps across
	petGap    = 6  // the least between two things on the ground
	petTrunk  = 3  // a tree's trunk, across
	petShort  = 10 // the shortest branch: a cat stands on it, in the one place
	petLong   = 14 // the longest
	petClimbW = 6  // a cat climbing, across: the room either side of a trunk
	petTop    = 12 // the highest a cat climbs to: its head at the top
)

// The room a cat is to the left of its middle, and to the right.
const (
	petLeft  = petW / 2
	petRight = petW - 1 - petLeft
)

// petsRoom is the least outdoors: a tree, a branch either side, and the
// grass round it, a cat sitting on a branch and one under it (user,
// 2026-10-07: a narrow pane too). A larger screen is more of the
// outdoors — more trees, stumps — never a larger cat (user, the same
// day: a larger cat than it was, for its eye to be where an eye is).
var petsRoom = Room{W: 30, H: 2 + 2*petTall, Most: 1}

// The cat, facing right; a cat facing left is drawn the other way about.
// '#' is its fur, 's' its stripes, 'w' its chest and paws, 'x' the more
// a black-and-white has white — its muzzle, its belly, its legs, the tip
// of its tail — and 'e' its eye: each in its coat's colour.
var (
	petWalk = [2]sprite{
		{
			"x.......#..#",
			"#.......####",
			".s......##e#",
			".##s#s#s##xx",
			".##s#s#s#www",
			"..#xxxxxxww.",
			"..x.x...x.x.",
			"..w.w...w.w.",
		},
		{
			"x.......#..#",
			"#.......####",
			".s......##e#",
			".##s#s#s##xx",
			".##s#s#s#www",
			"..#xxxxxxww.",
			".x..x..x..x.",
			"w...w..w...w",
		},
	}
	petLeap = sprite{
		"........#..#",
		"........####",
		"x.......##e#",
		"#s#s#s#s##xx",
		".##s#s#s#www",
		"wx.......xw.",
		"w.........w.",
	}
	petSit = [2]sprite{
		{
			".....#..#",
			".....####",
			".....##e#",
			".....##xx",
			"....##ww.",
			"...s#sww.",
			"...s#sxx.",
			"..#s#sxx.",
			"x.#s#s#xx",
			".######ww",
		},
		{ // the tail up
			".....#..#",
			".....####",
			".....##e#",
			".....##xx",
			"....##ww.",
			"...s#sww.",
			"x..s#sxx.",
			".#.s#sxx.",
			"..#s#s#xx",
			"..#####ww",
		},
	}
	petSleep = sprite{
		".......#..#.",
		"......####..",
		"..#s#s##xx..",
		".##s#s#s##ww",
		"x#xxxxxxx#ww",
	}
	// Climbing, the trunk at its left: its paws are the left column.
	petClimb = [2]sprite{
		{
			"..#..#",
			"..####",
			"..#e##",
			"..#xx#",
			"ww###.",
			".xsss.",
			".x###.",
			".xsss.",
			"ww###.",
			".x###.",
			"...#..",
			"...x..",
		},
		{
			"..#..#",
			"..####",
			"..#e##",
			"..#xx#",
			".x###.",
			"wwsss.",
			".x###.",
			".xsss.",
			".x###.",
			"ww###.",
			"....#.",
			"....x.",
		},
	}
	petZ = sprite{"###", "..#", ".#.", "###"}
)

// The cats' coats, in order (user, 2026-10-07): the fur, the stripes,
// the chest and paws, the more white and the eyes of each — two colours
// far apart (user, the same day: or the two are lost), the amber mostly
// dark coffee, its stripes yellow, its chest and paws grey (user, the
// same day), the black-and-white white the more (user, the same day).
// The eyes are a colour, not the ground (user, the same day), and not
// the z's yellow (user, the same day): green, the white's blue. The
// black is a charcoal, black being lost on the ground.
var petCoats = [][5]string{
	{"#f6b06a", "#a8460c", "#f6b06a", "#f6b06a", "#7ed957"}, // an orange tabby, green-eyed
	{"#7a4a2a", "#e8b830", "#9a958e", "#7a4a2a", "#7ed957"}, // an amber, green-eyed
	{"#f4f1ea", "#8c8a86", "#f4f1ea", "#f4f1ea", "#5fa8ff"}, // a white, blue-eyed
	{"#9fb2c8", "#4e5d70", "#9fb2c8", "#9fb2c8", "#7ed957"}, // a grey-blue, green-eyed
	{"#55504c", "#55504c", "#f4f1ea", "#f4f1ea", "#7ed957"}, // a black-and-white, green-eyed
}

// The outdoors' colours (user, 2026-10-07: the cats go out). The
// backdrop is a gradient a row at a time, as the runner's dusk is: the
// sky, dark blue high and lighter low, over a green field, lighter far
// and darker near — dark enough for the cats to show on it, a light one
// having lost them. Then the grass of the ground, the trees' bark, their
// leaves and the leaves' shade, a stump's cut top, and the z.
var petBackdrop = []petStop{
	{0, "#1d2745"},
	{0.5, "#3f5a7c"},
	{0.52, "#4e6e46"},
	{1, "#1c2f1a"},
}

// A petStop is a colour of the backdrop's, and how far down the screen.
type petStop struct {
	at     float64
	colour string
}

const (
	petGrass    = "#4f7d36"
	petBark     = "#847260"
	petLeaf     = "#5e9140"
	petLeafDark = "#3f6c2c"
	petCut      = "#b39b78"
	petYellow   = "#ffff00" // a sleeping cat's z (user, the same day)
)

// The inks past the ground: the grass, the bark, the leaves' two, the
// cut, the z, and each coat's five, coat c's from petCoat(c).
const (
	inkPetGrass uint8 = 1 + iota
	inkPetBark
	inkPetLeaf
	inkPetLeafDark
	inkPetCut
	inkPetZ
	inkPetCoats
)

// petCoat is coat c's inks: its fur, stripes, chest and paws, more white
// and eyes.
func petCoat(c int) [5]uint8 {
	k := inkPetCoats + uint8(5*c)
	return [5]uint8{k, k + 1, k + 2, k + 3, k + 4}
}

// A petPlace is somewhere a cat is. A ledge is a line it stands on, from
// x0 to x1: at is the row its feet are in, over the line, and its middle
// goes from lo to hi — a cat may stand two pixels over an edge. A climb
// is a trunk it holds on to: at is its paws' column, side where the
// trunk is, -1 at its left or 1 at its right, and its feet go from lo,
// the top, to hi.
type petPlace struct {
	climb  bool
	trunk  bool // a climb up a tree's trunk
	at     int
	x0, x1 int
	lo, hi int
	side   int
}

// A petPoint is a spot on a place: a ledge's column, a climb's row.
type petPoint struct{ place, pos int }

// A petLink is a jump from a spot on one place to a spot on another.
type petLink struct{ from, to petPoint }

type petDoing int

const (
	petGoing petDoing = iota
	petSitting
	petSleeping
	petHolding
)

type petCat struct {
	coat  int // which of petCoats
	at    petPoint
	face  int // 1 right, -1 left
	goal  petPoint
	plan  []int    // the links on the way to the goal, in order
	arc   [][2]int // a jump under way: where it is, its middle and its feet, a frame a pair
	run   bool
	doing petDoing
	rest  int // frames still to stay where it is
	tick  int // frames at what it is doing: the tail, the z
	steps int // pixels gone: the legs
}

// Pets is the room and its cats.
type Pets struct {
	rng    *rand.Rand
	n      int
	w, h   int
	room   []uint8    // the grass, the trees, the stumps; nil when the scene is too small
	places []petPlace // the floor first
	links  []petLink
	cost   []int // a link's frames
	reach  []int // the places a cat can get to, the floor twice: where it may go
	cats   []petCat
}

// NewPets is a room of count cats — PetsDefault for a count not in
// PetCounts. The same seed is the same room and the same cats.
func NewPets(seed uint64, count int) *Pets {
	if !slices.Contains(PetCounts, count) {
		count = PetsDefault
	}
	return &Pets{rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)), n: count}
}

// Room is the least room.
func (g *Pets) Room() Room { return petsRoom }

// Next is when the next frame is due.
func (g *Pets) Next(now time.Time) time.Time { return now.Add(petFrame) }

// Inks are the sky at the top, the grass, the bark, the leaves', the cut,
// the z's, and the coats'.
func (g *Pets) Inks() []string {
	inks := []string{petBackdrop[0].colour, petGrass, petBark, petLeaf, petLeafDark, petCut, petYellow}
	for _, c := range petCoats {
		inks = append(inks, c[:]...)
	}
	return inks
}

// Shade is the backdrop a row at a time, the sky over the field, under
// the inks: the one look all over.
func (g *Pets) Shade(cols, rows int) Shading {
	l := Look{Ground: make([]string, rows), Inks: g.Inks()}
	for y := range l.Ground {
		l.Ground[y] = petBackdropAt(y, rows)
	}
	return Shading{Looks: [2]Look{l, l}, Look: func(x, y int) int { return 0 }}
}

// petBackdropAt is the backdrop's colour in row y of rows.
func petBackdropAt(y, rows int) string {
	p := 0.0
	if rows > 1 {
		p = float64(y) / float64(rows-1)
	}
	for i := 1; i < len(petBackdrop); i++ {
		if a, b := petBackdrop[i-1], petBackdrop[i]; p <= b.at {
			return mix(a.colour, b.colour, (p-a.at)/(b.at-a.at))
		}
	}
	return petBackdrop[len(petBackdrop)-1].colour
}

// reset is a new room on a w × h scene, and the cats in it.
func (g *Pets) reset(w, h int) {
	g.w, g.h = w, h
	g.room, g.places, g.links, g.cost, g.reach, g.cats = nil, nil, nil, nil, nil, nil
	if w < petsRoom.W || h < petsRoom.H {
		return
	}
	g.room = make([]uint8, w*h)
	g.furnish()
	g.join()
	g.settle()
}

// light lights the furniture at x, y in ink.
func (g *Pets) light(x, y int, ink uint8) {
	if x >= 0 && x < g.w && y >= 0 && y < g.h {
		g.room[y*g.w+x] = ink
	}
}

func (g *Pets) hline(x0, x1, y int, ink uint8) {
	for x := x0; x <= x1; x++ {
		g.light(x, y, ink)
	}
}

func (g *Pets) vline(x, y0, y1 int, ink uint8) {
	for y := y0; y <= y1; y++ {
		g.light(x, y, ink)
	}
}

// ledge adds the line at row y from x0 to x1 as a ledge.
func (g *Pets) ledge(y, x0, x1 int) int {
	g.places = append(g.places, petPlace{at: y - 1, x0: x0, x1: x1, lo: max(x0-2+petLeft, petLeft), hi: min(x1+2-petRight, g.w-1-petRight)})
	return len(g.places) - 1
}

// climb adds a climb, the paws in column x and the trunk at side, from
// row top to the ground.
func (g *Pets) climb(x, side, top int) int {
	g.places = append(g.places, petPlace{climb: true, at: x, side: side, lo: top, hi: g.h - 2})
	return len(g.places) - 1
}

// both links a and b, there and back.
func (g *Pets) both(a, b petPoint) {
	g.links = append(g.links, petLink{a, b}, petLink{b, a})
}

// middle is the column a cat's middle is in, holding on to climb c.
func (g *Pets) middle(c petPlace) int { return c.at - 3*c.side }

// near is the spot on ledge l nearest column x.
func (g *Pets) near(l, x int) petPoint {
	p := g.places[l]
	return petPoint{l, min(max(x, p.lo), p.hi)}
}

// mount links climb c's foot to the ground and the spot by it.
func (g *Pets) mount(c int) {
	p := g.places[c]
	g.both(g.near(0, g.middle(p)), petPoint{c, p.hi})
}

// A petPiece is something on the ground, before it is put there: a tree,
// or a stump — its width and height; and how wide it is.
type petPiece struct {
	tree  *petTree
	stump [2]int
	w     int
}

// A petTree is a tree before it is put there: its crown's top row, how
// high and how wide the crown is, its branches, and how far it reaches to
// the left of its trunk and to the right, branches and crown.
type petTree struct {
	top, crownH, crownW int
	branches            []petBranch
	left, right         int
}

// A petBranch is a branch: its row, the side of the trunk it grows from,
// -1 the left, and how long it is.
type petBranch struct{ y, side, long int }

// furnish lays the outdoors out (user, 2026-10-07: the cats go out): the
// ground, a line of grass with tufts on it; on it a tree at the least
// (user, the same day: a narrow pane too), then trees and stumps, as many
// as the width holds six pixels apart, a tree twice as often as a stump,
// in some order and spread out.
func (g *Pets) furnish() {
	floor := g.h - 1
	g.hline(0, g.w-1, floor, inkPetGrass)
	for x := range g.w {
		if g.rng.IntN(4) == 0 {
			g.light(x, floor-1, inkPetGrass)
		}
	}
	g.ledge(floor, 0, g.w-1)
	room := g.w
	var pieces []petPiece
	for {
		var p petPiece
		if len(pieces) == 0 || g.rng.IntN(3) > 0 {
			p = g.tree(room)
		}
		if p.w == 0 && len(pieces) > 0 {
			p = g.stump(room)
		}
		if p.w == 0 {
			break
		}
		pieces, room = append(pieces, p), room-p.w-petGap
	}
	g.rng.Shuffle(len(pieces), func(i, j int) { pieces[i], pieces[j] = pieces[j], pieces[i] })
	gaps := g.spread(len(pieces), room+petGap)
	x := gaps[0]
	for i, p := range pieces {
		if p.tree != nil {
			g.putTree(x, p.tree)
		} else {
			g.putStump(x, p.stump)
		}
		x += p.w + petGap + gaps[i+1]
	}
}

// spread is the gaps at the ends of n things on the ground and between
// them, spare pixels in all past the least, at random.
func (g *Pets) spread(n, spare int) []int {
	gaps := make([]int, n+1)
	for range spare {
		gaps[g.rng.IntN(n+1)]++
	}
	return gaps
}

// tree is a tree no wider than room, at random: a crown of leaves high
// up, its top as high as a quarter down the sky; under it branches either
// side of the trunk by turns, a step up each — a cat sitting under each,
// on the ground or the branch under it on its side — up to the crown,
// each as long as it is at random; room either side of the trunk for a
// cat to climb it. None when the room is too narrow.
func (g *Pets) tree(room int) petPiece {
	floor := g.h - 1
	half := (room - petTrunk) / 2
	if half < petClimbW {
		return petPiece{}
	}
	t := &petTree{crownH: 7 + g.rng.IntN(4), crownW: min(12+g.rng.IntN(7), room)}
	t.top = 1 + g.rng.IntN(max(1, floor/4))
	side := 1 - 2*g.rng.IntN(2)
	for y := floor - petTall - 1 - g.rng.IntN(3); y >= max(petTall, t.top+t.crownH-3); y -= 6 + g.rng.IntN(3) {
		if long := min(petShort+g.rng.IntN(petLong-petShort+1), half); long >= petShort {
			t.branches = append(t.branches, petBranch{y, side, long})
		}
		side = -side
	}
	t.left = max(petClimbW, (t.crownW-petTrunk+1)/2)
	t.right = t.left
	for _, b := range t.branches {
		if b.side < 0 {
			t.left = max(t.left, b.long)
		} else {
			t.right = max(t.right, b.long)
		}
	}
	w := t.left + petTrunk + t.right
	if w > room {
		return petPiece{}
	}
	return petPiece{tree: t, w: w}
}

// putTree puts tree t with the left of its reach at x: its trunk, from the
// middle of its crown down, its roots spread at the foot; the crown over
// it, an oval of leaves in a speckle of shade; its branches, each a
// ledge; and the trunk's sides to climb, each up to its highest branch,
// or under the crown.
func (g *Pets) putTree(x int, t *petTree) {
	floor := g.h - 1
	tx := x + t.left
	for i := range petTrunk {
		g.vline(tx+i, t.top+t.crownH/2, floor-1, inkPetBark)
	}
	g.light(tx-1, floor-1, inkPetBark)
	g.light(tx+petTrunk, floor-1, inkPetBark)
	cx, cy := float64(tx)+float64(petTrunk-1)/2, float64(t.top)+float64(t.crownH-1)/2
	rx, ry := float64(t.crownW)/2, float64(t.crownH)/2
	for y := t.top; y < t.top+t.crownH; y++ {
		for lx := tx - t.crownW; lx <= tx+t.crownW; lx++ {
			if dx, dy := (float64(lx)-cx)/rx, (float64(y)-cy)/ry; dx*dx+dy*dy > 1 {
				continue
			}
			ink := inkPetLeaf
			if (3*lx+5*y)%7 == 0 {
				ink = inkPetLeafDark
			}
			g.light(lx, y, ink)
		}
	}
	for _, b := range t.branches {
		x0 := tx + petTrunk
		if b.side < 0 {
			x0 = tx - b.long
		}
		g.hline(x0, x0+b.long-1, b.y, inkPetBark)
		g.ledge(b.y, x0, x0+b.long-1)
	}
	for _, side := range []int{-1, 1} {
		top := -1
		for _, b := range t.branches {
			if b.side == side && (top < 0 || b.y < top) {
				top = b.y
			}
		}
		if top < 0 {
			top = t.top + t.crownH
		}
		paws, holds := tx-1, 1
		if side > 0 {
			paws, holds = tx+petTrunk, -1
		}
		c := g.climb(paws, holds, max(petTop-1, top-1))
		g.places[c].trunk = true
		g.mount(c)
	}
}

// stump is a stump no wider than room, at random; none when none fits.
func (g *Pets) stump(room int) petPiece {
	w := 12 + g.rng.IntN(3)
	if w > room {
		return petPiece{}
	}
	return petPiece{stump: [2]int{w, 5 + g.rng.IntN(3)}, w: w}
}

// putStump puts stump s with its left at x: its sides bark, its top the
// wood cut, its roots spread at the foot; its top a ledge.
func (g *Pets) putStump(x int, s [2]int) {
	floor := g.h - 1
	top := floor - s[1]
	for y := top + 1; y < floor; y++ {
		g.hline(x, x+s[0]-1, y, inkPetBark)
	}
	g.hline(x, x+s[0]-1, top, inkPetCut)
	g.light(x-1, floor-1, inkPetBark)
	g.light(x+s[0], floor-1, inkPetBark)
	g.ledge(top, x, x+s[0]-1)
}

// join links every two ledges a cat can jump between: up onto one no
// more than a jump higher, from beside it, to its nearer end; down — or
// across — off the end of one, onto another clear of it; neither
// farther across than a jump. And a trunk to every ledge no farther
// across: from where a cat holds on, level with it — or as high as it
// holds on, a pixel or two under the highest — to its nearer end, and
// back: its branches, and a stump by it.
func (g *Pets) join() {
	for c, C := range g.places {
		if !C.trunk {
			continue
		}
		for l, L := range g.places {
			if l == 0 || L.climb {
				continue
			}
			feet, to := max(L.at, C.lo), g.near(l, g.middle(C))
			if abs(to.pos-g.middle(C)) <= petReach {
				g.both(petPoint{c, feet}, to)
			}
		}
	}
	for a, A := range g.places {
		for b, B := range g.places {
			if a == b || A.climb || B.climb {
				continue
			}
			if rise := A.at - B.at; rise > 0 {
				if rise > petJump {
					continue
				}
				if t := min(A.hi, B.x0-petRight-1); t >= A.lo && B.lo-t <= petReach {
					g.links = append(g.links, petLink{petPoint{a, t}, petPoint{b, B.lo}})
				}
				if t := max(A.lo, B.x1+petLeft+1); t <= A.hi && t-B.hi <= petReach {
					g.links = append(g.links, petLink{petPoint{a, t}, petPoint{b, B.hi}})
				}
				continue
			}
			if l := min(B.hi, A.x0-petRight-1); l >= B.lo && A.lo-l <= petReach {
				g.links = append(g.links, petLink{petPoint{a, A.lo}, petPoint{b, l}})
			}
			if l := max(B.lo, A.x1+petLeft+1); l <= B.hi && l-A.hi <= petReach {
				g.links = append(g.links, petLink{petPoint{a, A.hi}, petPoint{b, l}})
			}
		}
	}
	g.cost = make([]int, len(g.links))
	for i, l := range g.links {
		g.cost[i] = len(g.leap(l))
	}
	// Where a cat can get to from the floor.
	seen := make([]bool, len(g.places))
	seen[0] = true
	for queue := []int{0}; len(queue) > 0; queue = queue[1:] {
		for _, l := range g.links {
			if l.from.place == queue[0] && !seen[l.to.place] {
				seen[l.to.place] = true
				queue = append(queue, l.to.place)
			}
		}
	}
	g.reach = []int{0}
	for p := range g.places {
		if seen[p] {
			g.reach = append(g.reach, p)
		}
	}
}

// settle puts the cats in the room, the first of the coats, sitting
// somewhere a moment before each goes.
func (g *Pets) settle() {
	g.cats = make([]petCat, g.n)
	for i := range g.cats {
		c := &g.cats[i]
		c.coat = i
		c.at = petPoint{0, g.places[0].lo + g.rng.IntN(g.places[0].hi-g.places[0].lo+1)}
		for range 8 {
			if p := g.pick(i, i); !g.places[p.place].climb {
				c.at = p
				break
			}
		}
		c.goal, c.face = c.at, 1-2*g.rng.IntN(2)
		c.doing, c.rest = petSitting, 1+g.rng.IntN(40)
	}
}

// pick is somewhere for cat i to go: a place at random, a spot on it at
// random, not too near where another of the first n cats is or goes.
func (g *Pets) pick(i, n int) petPoint {
	var p petPoint
	for range 8 {
		pl := g.reach[g.rng.IntN(len(g.reach))]
		P := g.places[pl]
		p = petPoint{pl, P.lo + g.rng.IntN(P.hi-P.lo+1)}
		crowded := false
		for j := range n {
			o := g.cats[j]
			if j != i && (o.at.place == p.place && abs(o.at.pos-p.pos) < petW || o.goal.place == p.place && abs(o.goal.pos-p.pos) < petW) {
				crowded = true
			}
		}
		if !crowded {
			break
		}
	}
	return p
}

// way is the links from one spot to another, the quickest: along a
// place two frames a pixel, and a jump its frames.
func (g *Pets) way(from, to petPoint) ([]int, bool) {
	n := 2 + 2*len(g.links)
	pt := func(v int) petPoint {
		switch {
		case v == 0:
			return from
		case v == 1:
			return to
		case v%2 == 0:
			return g.links[(v-2)/2].from
		}
		return g.links[(v-3)/2].to
	}
	const inf = math.MaxInt
	dist, prev, done := make([]int, n), make([]int, n), make([]bool, n)
	for v := range dist {
		dist[v], prev[v] = inf, -1
	}
	dist[0] = 0
	for {
		u := -1
		for v := range n {
			if !done[v] && dist[v] < inf && (u < 0 || dist[v] < dist[u]) {
				u = v
			}
		}
		if u < 0 || u == 1 {
			break
		}
		done[u] = true
		relax := func(v, d int) {
			if !done[v] && dist[u]+d < dist[v] {
				dist[v], prev[v] = dist[u]+d, u
			}
		}
		a := pt(u)
		for v := range n {
			if b := pt(v); b.place == a.place {
				relax(v, 2*abs(a.pos-b.pos))
			}
		}
		if u >= 2 && u%2 == 0 {
			relax(u+1, g.cost[(u-2)/2])
		}
	}
	if dist[1] == inf {
		return nil, false
	}
	var plan []int
	for v := 1; prev[v] >= 0; v = prev[v] {
		if u := prev[v]; u >= 2 && u%2 == 0 && v == u+1 {
			plan = append(plan, (u-2)/2)
		}
	}
	slices.Reverse(plan)
	return plan, true
}

// spot is where a cat at p is: its middle's column and its feet's row.
func (g *Pets) spot(p petPoint) (int, int) {
	pl := g.places[p.place]
	if pl.climb {
		return g.middle(pl), p.pos
	}
	return p.pos, pl.at
}

// leap is the jump along l, frame by frame, to its end: up a pixel or
// two over the higher of its ends, and down; longer the farther.
func (g *Pets) leap(l petLink) [][2]int {
	x0, y0 := g.spot(l.from)
	x1, y1 := g.spot(l.to)
	dx, dy := x1-x0, y1-y0
	n := max(4, 2+(abs(dx)+abs(dy))/3)
	// y0 + dy·t − k·t(1 − t), its top two over the higher end, up or
	// down: k is the same for a drop as for the rise.
	drop := float64(abs(dy))
	k := drop + 4 + math.Sqrt(8*drop+16)
	out := make([][2]int, n)
	for i := range n {
		t := float64(i+1) / float64(n)
		out[i] = [2]int{
			int(math.Round(float64(x0) + float64(dx)*t)),
			int(math.Round(float64(y0) + float64(dy)*t - k*t*(1-t))),
		}
	}
	return out
}

// wander sends cat i somewhere else, walking, or running when the way
// along the ground is long.
func (g *Pets) wander(i int) {
	c := &g.cats[i]
	for range 8 {
		goal := g.pick(i, len(g.cats))
		if plan, ok := g.way(c.at, goal); ok && goal != c.at {
			c.goal, c.plan = goal, plan
			c.doing, c.tick = petGoing, 0
			c.run = g.walk(c.at, plan, goal) > 24 && g.rng.IntN(2) == 0
			return
		}
	}
	c.rest = 10
}

// walk is how far a cat goes along places on its way.
func (g *Pets) walk(at petPoint, plan []int, goal petPoint) int {
	n := 0
	for _, l := range plan {
		n += abs(g.links[l].from.pos - at.pos)
		at = g.links[l].to
	}
	return n + abs(goal.pos-at.pos)
}

// Step is a frame: each cat a frame on.
func (g *Pets) Step() {
	for i := range g.cats {
		g.stepCat(i)
	}
}

func (g *Pets) stepCat(i int) {
	c := &g.cats[i]
	c.tick++
	if c.arc != nil {
		if len(c.arc) > 1 {
			c.arc = c.arc[1:]
			return
		}
		c.at, c.plan, c.arc = g.links[c.plan[0]].to, c.plan[1:], nil
		return
	}
	if c.doing != petGoing {
		if c.rest--; c.rest <= 0 {
			g.wander(i)
		}
		return
	}
	pl := g.places[c.at.place]
	target := c.goal.pos
	if len(c.plan) > 0 {
		target = g.links[c.plan[0]].from.pos
	}
	if c.at.pos != target {
		if c.run && !pl.climb || c.tick%2 == 0 {
			d := 1
			if target < c.at.pos {
				d = -1
			}
			c.at.pos += d
			c.steps++
			if !pl.climb {
				c.face = d
			}
		}
		return
	}
	if len(c.plan) > 0 {
		l := g.links[c.plan[0]]
		c.arc = g.leap(l)
		x0, _ := g.spot(l.from)
		x1, _ := g.spot(l.to)
		switch {
		case x1 > x0:
			c.face = 1
		case x1 < x0:
			c.face = -1
		}
		return
	}
	c.tick = 0
	switch {
	case pl.climb:
		c.doing, c.rest = petHolding, 25+g.rng.IntN(30)
	case g.rng.IntN(3) == 0:
		c.doing, c.rest = petSleeping, 100+g.rng.IntN(100)
	default:
		c.doing, c.rest = petSitting, 40+g.rng.IntN(60)
	}
}

// Draw is the frame at w × h pixels: the furniture, and the cats in
// front of it. A scene of a new size is a new room.
func (g *Pets) Draw(w, h int) Scene {
	if w != g.w || h != g.h {
		g.reset(w, h)
	}
	sc := newScene(w, h)
	if g.room == nil {
		return sc
	}
	copy(sc.Pix, g.room)
	for i := range g.cats {
		g.drawCat(&sc, &g.cats[i])
	}
	return sc
}

func (g *Pets) drawCat(sc *Scene, c *petCat) {
	pl := g.places[c.at.place]
	inks := petCoat(c.coat)
	switch {
	case c.arc != nil:
		petStand(sc, petLeap, c.arc[0][0], c.arc[0][1], c.face, inks)
	case pl.climb:
		sp := petClimb[c.steps%2]
		if pl.side < 0 {
			petBlit(sc, sp, pl.at, c.at.pos-sp.h()+1, false, inks)
		} else {
			petBlit(sc, sp, pl.at-sp.w()+1, c.at.pos-sp.h()+1, true, inks)
		}
	case c.doing == petSitting:
		sp := petSit[0]
		if c.tick%40 >= 34 {
			sp = petSit[1]
		}
		petStand(sc, sp, c.at.pos, pl.at, c.face, inks)
	case c.doing == petSleeping:
		petStand(sc, petSleep, c.at.pos, pl.at, c.face, inks)
		// A z rising over its head, a pixel every six frames, gone a while.
		if ph := c.tick % 36; ph < 24 {
			zx := c.at.pos + 2
			if c.face < 0 {
				zx = c.at.pos - 5
			}
			petBlit(sc, petZ, zx, pl.at-9-ph/6, false, [5]uint8{inkPetZ, inkPetZ, inkPetZ, inkPetZ, inkPetZ})
		}
	default:
		sp := petWalk[c.steps%2]
		if c.run && c.steps%2 == 1 {
			sp = petLeap
		}
		petStand(sc, sp, c.at.pos, pl.at, c.face, inks)
	}
}

// petStand draws sp with its middle in column x and its bottom in row y,
// facing face.
func petStand(sc *Scene, sp sprite, x, y, face int, inks [5]uint8) {
	petBlit(sc, sp, x-sp.w()/2, y-sp.h()+1, face < 0, inks)
}

// petBlit draws sp with its top left at x, y, the other way about if
// flip: '#', 's', 'w', 'x' and 'e' in inks, '.' what is behind.
func petBlit(sc *Scene, sp sprite, x, y int, flip bool, inks [5]uint8) {
	for dy, row := range sp {
		for dx := range len(row) {
			ch := row[dx]
			if flip {
				ch = row[len(row)-1-dx]
			}
			if k := strings.IndexByte("#swxe", ch); k >= 0 {
				sc.put(x+dx, y+dy, inks[k])
			}
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
