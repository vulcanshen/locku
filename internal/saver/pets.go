package saver

import (
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"time"
)

// The pets (user, 2026-10-07): cats about a room, after vscode-pets but
// drawn our own. Each goes somewhere at random — along the floor, onto
// a board of the cat tree, a box, a shelf, up a wall — walking there, or
// running when it is far, jumping up and down and climbing as the way
// asks; and there it sits a while, or curls up and sleeps, a z rising
// over it, or, up a wall or a post, holds on; then it goes on. The
// furniture is behind them: a cat goes in front of a box, and of a board
// as it climbs past it.
//
// The cats are five, the first as many as there are of them (user, the
// same day: cats come in a few colours, not any), each of two colours
// (user, the same day): an orange tabby, light orange striped dark; an
// amber, dark yellow striped dark grey; a white striped light grey; a
// grey-blue striped darker; and a black-and-white. The room is a room's
// colours (user, the same day): the ground a dark umber — it was a soft
// sand, until the cats were lost on it — the furniture wood.
//
// The room is laid out anew for each scene: a floor from wall to wall
// and on it, clear of the walls, a cat tree — a post or more, a board on
// each, stepping up — and boxes, one, two side by side, or three in a
// heap, as many as the width holds. Up each wall, the scene's sides, is
// a pole of sisal, floor to top, and on it shelves, one over the other,
// as many as the height holds; a cat climbs the pole, and steps off it
// onto a shelf. A cat jumps up as high as the tree's steps, and down from
// anywhere.
//
// Its one setting is how many cats (user, the same day).

const KindPets = "pets"

// PetCounts are how many cats there may be; PetsDefault is the default.
var PetCounts = []int{1, 2, 3, 4, 5}

const PetsDefault = 3

const (
	petFrame = 80 * time.Millisecond
	petW     = 12 // a cat walking, across; its middle six from its left
	petTall  = 10 // a cat sitting, high: the room it needs over where it sits
	petJump  = 13 // the highest a cat jumps, in pixels
	petReach = 16 // the farthest it jumps across
	petPole  = 2  // a wall's pole, across
	petWall  = 8  // the floor kept clear by each wall: its pole and the climb's
	petGap   = 6  // the least between two things on the floor
	petBoard = 14 // a board of the cat tree, across
	petBox   = 14 // a box, across
	petBoxH  = 9  // and high
	petShelf = 14 // the longest shelf, across from its wall
	petShort = 10 // the shortest: a cat stands on it, in the one place
	petTwist = 3  // the rows from one turn of a pole's rope to the next
	petTop   = 12 // the highest a board or a shelf is: a cat climbing to it fits under the top
)

// The room a cat is to the left of its middle, and to the right.
const (
	petLeft  = petW / 2
	petRight = petW - 1 - petLeft
)

// petsRoom is the least room: the walls, a shelf on each with a cat
// sitting on it and one under it, and the floor between them (user,
// 2026-10-07: a narrow pane too). A larger screen is a larger room — a
// cat tree, more boxes, a shelf — never a larger cat (user, the same
// day: a larger cat than it was, for its eye to be where an eye is).
var petsRoom = Room{W: 2*petWall + petBox, H: 2 + 2*petTall, Most: 1}

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
	// Climbing, the wall at its left: its paws are the left column.
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

// The room's colours (user, 2026-10-07: a room's, the ground earthy and
// dark, the furniture wood — a greyish walnut, the orange tabby's orange
// and the amber's coffee neither of them).
const (
	petGround   = "#2b231e" // a dark umber
	petDarkWood = "#6e6052" // the floor, the posts, the braces
	petWood     = "#a39484" // the boards, the shelves, the boxes
	petRope     = "#d6bc8a" // the poles on the walls: sisal
	petRopeDark = "#9c8158" // and the lay of its rope, across it
	petYellow   = "#ffff00" // a sleeping cat's z (user, the same day)
)

// The inks past the ground: the woods, the rope's two, the z, and each
// coat's five, coat c's from petCoat(c).
const (
	inkPetDark uint8 = 1 + iota
	inkPetWood
	inkPetRope
	inkPetRopeDark
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
// is a wall or a post it holds on to: at is its paws' column, side where
// the wall is, -1 at its left or 1 at its right, and its feet go from
// lo, the top, to hi.
type petPlace struct {
	climb  bool
	wall   bool // a climb up a wall's pole
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
	room   []uint8    // the furniture, in its woods; nil when the scene is too small
	sw     int        // the longest a wall's shelf is, and the floor kept clear under it
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

// Inks are the ground, the woods, the z's, and the coats'.
func (g *Pets) Inks() []string {
	inks := []string{petGround, petDarkWood, petWood, petRope, petRopeDark, petYellow}
	for _, c := range petCoats {
		inks = append(inks, c[:]...)
	}
	return inks
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

// climb adds a climb, the paws in column x and the wall at side, from
// row top to the floor.
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

// mount links climb c's foot to the floor and the spot by it, and the
// spot on ledge l at its row, if there is one, to the ledge.
func (g *Pets) mount(c, l int) {
	p := g.places[c]
	g.both(g.near(0, g.middle(p)), petPoint{c, p.hi})
	if l >= 0 {
		g.both(petPoint{c, g.places[l].at}, g.near(l, g.middle(p)))
	}
}

// A petPiece is something on the floor, before it is put there: a cat
// tree — its boards' heights over the floor, left to right — or boxes,
// heap one of petHeaps; and how wide it is.
type petPiece struct {
	heights []int
	heap    int
	w       int
}

// The heaps of boxes: one, two side by side, and those two and one on
// one of them, a step and a step.
const (
	petOne = iota
	petTwo
	petHeap
	petHeaps
)

// furnish lays the room out: the floor; on it, clear of the walls, a cat
// tree and boxes, as many as the floor holds six pixels apart, in some
// order and spread out; and shelves up both walls.
func (g *Pets) furnish() {
	floor := g.h - 1
	g.hline(0, g.w-1, floor, inkPetDark)
	g.ledge(floor, 0, g.w-1)
	// A pole of sisal up each wall, floor to top (user, 2026-10-07: a
	// wall has something to climb); a cat climbs it. Its rope lies
	// across it aslant, a pixel lower a column on, a turn every three
	// rows (user, the same day).
	for x := range petPole {
		for y := range floor {
			ink := inkPetRope
			if (y-x+petTwist)%petTwist == 0 {
				ink = inkPetRopeDark
			}
			g.light(x, y, ink)
			g.light(g.w-petPole+x, y, ink)
		}
	}
	walls := [2]int{g.climb(petPole, -1, petTop-1), g.climb(g.w-1-petPole, 1, petTop-1)}
	for _, c := range walls {
		g.places[c].wall = true
	}
	// The walls' shelves as long as they are, or shorter for a box on the
	// floor between them, or the shortest (user, 2026-10-07: a narrow pane
	// has them too); the floor under them clear.
	g.sw = min(petShelf, max(petShort, (g.w-petBox)/2))
	room := g.w - 2*g.sw
	var pieces []petPiece
	if t := g.tree(room); t.w > 0 {
		pieces, room = append(pieces, t), room-t.w-petGap
	}
	for {
		b := g.boxes(room)
		if b.w == 0 {
			break
		}
		pieces, room = append(pieces, b), room-b.w-petGap
	}
	g.rng.Shuffle(len(pieces), func(i, j int) { pieces[i], pieces[j] = pieces[j], pieces[i] })
	gaps := g.spread(len(pieces), room+petGap)
	x := g.sw + gaps[0]
	for i, p := range pieces {
		if p.heights != nil {
			g.putTree(x, p.heights)
		} else {
			g.putBoxes(x, p.heap)
		}
		x += p.w + petGap + gaps[i+1]
	}
	// The walls from the floor; onto a shelf, as onto anything by them, a
	// cat jumps from where it holds on (join).
	g.mount(walls[0], -1)
	g.mount(walls[1], -1)
	g.shelves(0)
	g.shelves(1)
}

// spread is the gaps at the ends of n things on the floor and between
// them, spare pixels in all past the least, at random.
func (g *Pets) spread(n, spare int) []int {
	gaps := make([]int, n+1)
	for range spare {
		gaps[g.rng.IntN(n+1)]++
	}
	return gaps
}

// tree is a cat tree no wider than room: two posts to four, as many as
// the height holds — one at the least — a board on each, a step higher
// each to the left or to the right; none when the room is too low or
// too narrow for one.
func (g *Pets) tree(room int) petPiece {
	floor := g.h - 1
	n := 2 + g.rng.IntN(3)
	var hs []int
	for ht := 11 + g.rng.IntN(3); ht <= floor-petTop && len(hs) < n && (len(hs)+1)*petBoard <= room; ht += 6 + g.rng.IntN(4) {
		hs = append(hs, ht)
	}
	if g.rng.IntN(2) == 0 {
		slices.Reverse(hs)
	}
	return petPiece{heights: hs, w: len(hs) * petBoard}
}

// putTree puts the cat tree with its left at x.
func (g *Pets) putTree(x int, heights []int) {
	floor := g.h - 1
	for i, ht := range heights {
		bx, y := x+i*petBoard, floor-ht
		post := bx + petBoard/2 - 1
		g.hline(bx, bx+petBoard-1, y, inkPetWood)
		g.vline(post, y+1, floor-1, inkPetDark)
		g.vline(post+1, y+1, floor-1, inkPetDark)
		g.light(post-1, floor-1, inkPetDark)
		g.light(post+2, floor-1, inkPetDark)
		l := g.ledge(y, bx, bx+petBoard-1)
		// The outer sides of the posts at the ends are clear to climb.
		if i == 0 {
			g.mount(g.climb(post-1, 1, y-1), l)
		}
		if i == len(heights)-1 {
			g.mount(g.climb(post+2, -1, y-1), l)
		}
	}
}

// boxes is a heap of boxes no wider than room, at random; none when none
// fits.
func (g *Pets) boxes(room int) petPiece {
	var heaps []int
	for k := range petHeaps {
		// The heap's top no higher than a board.
		if petHeapW(k) <= room && (k != petHeap || g.h-1-2*petBoxH+1 >= petTop) {
			heaps = append(heaps, k)
		}
	}
	if len(heaps) == 0 {
		return petPiece{}
	}
	k := heaps[g.rng.IntN(len(heaps))]
	return petPiece{heap: k, w: petHeapW(k)}
}

func petHeapW(k int) int {
	if k == petOne {
		return petBox
	}
	return 2 * petBox
}

// putBoxes puts heap k with its left at x.
func (g *Pets) putBoxes(x, k int) {
	top := g.h - 1 - petBoxH
	box := func(bx, by int) {
		g.hline(bx, bx+petBox-1, by, inkPetWood)
		g.hline(bx, bx+petBox-1, by+petBoxH-1, inkPetWood)
		g.vline(bx, by, by+petBoxH-1, inkPetWood)
		g.vline(bx+petBox-1, by, by+petBoxH-1, inkPetWood)
	}
	switch k {
	case petOne:
		box(x, top)
		g.ledge(top, x, x+petBox-1)
	case petTwo:
		box(x, top)
		box(x+petBox, top)
		g.ledge(top, x, x+2*petBox-1)
	default:
		box(x, top)
		box(x+petBox, top)
		lx, hx, hy := x, x+petBox, top-petBoxH+1
		if g.rng.IntN(2) == 0 {
			lx, hx = hx, lx
		}
		box(hx, hy)
		g.ledge(top, lx, lx+petBox-1)
		g.ledge(hy, hx, hx+petBox-1)
	}
}

// shelves puts shelves up the wall at side, 0 the left and 1 the right,
// on its pole, each on a bracket, the triangle of the pole, the shelf and
// a brace from one to the other (user, 2026-10-07: a wall has platforms
// to climb, and a shelf stands on a bracket): as many as there is room
// for, a cat sitting on each under the one over it or the top, and under
// the lowest — sometimes one fewer — at heights at random, high and low
// (user, the same day), and each as long as it is at random, the
// shortest to the longest (user, the same day).
func (g *Pets) shelves(side int) {
	lo, hi := petTall, g.h-2-petTall
	if hi < lo {
		return
	}
	n := max(1, (hi-lo)/(petTall+1)+1-g.rng.IntN(2))
	gaps := g.spread(n, hi-lo-(n-1)*(petTall+1))
	y := lo + gaps[0]
	for i := range n {
		sw := petShort + g.rng.IntN(g.sw-petShort+1)
		x0 := 0
		if side == 1 {
			x0 = g.w - sw
		}
		g.hline(x0, x0+sw-1, y, inkPetWood)
		// The brace, half the shelf past the pole out, as far down.
		brace := (sw - petPole) / 2
		for j := 1; j <= brace; j++ {
			if side == 0 {
				g.light(petPole+brace-j, y+j, inkPetDark)
			} else {
				g.light(g.w-1-petPole-brace+j, y+j, inkPetDark)
			}
		}
		g.ledge(y, x0, x0+sw-1)
		y += petTall + 1 + gaps[i+1]
	}
}

// join links every two ledges a cat can jump between: up onto one no
// more than a jump higher, from beside it, to its nearer end; down — or
// across — off the end of one, onto another clear of it; neither
// farther across than a jump. And a wall to every ledge no farther
// across: from where a cat holds on, level with it — or as high as it
// holds on, a pixel or two under the highest — to its nearer end, and
// back (user, 2026-10-07: a narrow room's box, a wall's shelf).
func (g *Pets) join() {
	for c, C := range g.places {
		if !C.wall {
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
