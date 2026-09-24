package saver

import (
	"math/rand/v2"
	"time"
)

// The dino run (function.md §5.2): the offline game as a screensaver. The
// runner runs, the ground and the obstacles scroll past, and it jumps
// them by itself, for ever. Nobody plays it: a jump is timed to clear
// what is coming, at a random moment inside the window that clears it,
// and now and then there is a jump for nothing when the way is clear
// (user, 2026-09-24: endless, random obstacles, random jumps). Two of
// its settings pick the art — the runner and the scene. Runners: the
// T-Rex, or two of them one behind the other, each jumping on its own.
// Scenes: grassland, with cacti; the desert, with pyramids.
//
// Everything here is in the scene's own pixels; the canvas scales them.

// The kinds of saver there are — the classes, in the user's word (2026-09-24);
// a profile is one of them set up under a name — and the dino's runners
// and scenes.
const (
	KindClock = "clock"
	KindDino  = "dino"

	RunnerTRex    = "trex"
	RunnerTwoTRex = "two-trex"
	SceneGrass    = "grassland"
	SceneDesert   = "desert"
)

var (
	Kinds   = []string{KindClock, KindDino}
	Runners = []string{RunnerTRex, RunnerTwoTRex}
	Scenes  = []string{SceneGrass, SceneDesert}
)

// DinoFrame is the time between two frames: fourteen a second.
const DinoFrame = 70 * time.Millisecond

// Scene is one frame: a bitmap in the game's own pixels, row by row.
// Nothing else — no score, no clock (user, 2026-09-24): it is a
// screensaver, not a game being played.
type Scene struct {
	W, H int
	Pix  []bool
}

func (s *Scene) set(x, y int) {
	if x >= 0 && x < s.W && y >= 0 && y < s.H {
		s.Pix[y*s.W+x] = true
	}
}

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

const (
	speed     = 2   // pixels the world moves a frame
	groundH   = 2   // the ground line, and the row of tufts under it
	minGap    = 44  // the least between two obstacles: a jump, and a landing
	maxGap    = 100 // the most
	firstGap  = 60  // before the first
	runnerGap = 4   // between two runners, nose to tail
)

// arc is a jump: how far the runner is above the ground, frame by frame.
// It is a slow, high jump — the widest obstacle is thirteen pixels and
// the runner twelve, and at two pixels a frame the two need ten frames
// or so above the tallest cactus.
var arc = []int{2, 4, 6, 7, 8, 8, 8, 8, 8, 8, 8, 8, 7, 6, 4, 2}

// A sprite is rows of '#' lit and '.' dark, top to bottom, all one width.
type sprite []string

func (s sprite) w() int { return len(s[0]) }
func (s sprite) h() int { return len(s) }

// beside puts sprites in a row, bottoms aligned, gap dark pixels apart.
func beside(gap int, parts ...sprite) sprite {
	w, h := -gap, 0
	for _, p := range parts {
		w += p.w() + gap
		h = max(h, p.h())
	}
	out := make(sprite, h)
	for y := range out {
		row := make([]byte, w)
		for i := range row {
			row[i] = '.'
		}
		x := 0
		for _, p := range parts {
			if py := y - (h - p.h()); py >= 0 {
				copy(row[x:], p[py])
			}
			x += p.w() + gap
		}
		out[y] = string(row)
	}
	return out
}

// runnerArt is a runner: two running poses and the one in the air, and
// how many of them run — one behind the other, each jumping on its own
// (user, 2026-09-24).
type runnerArt struct {
	run   [2]sprite
	air   sprite
	count int
}

// sceneArt is a scene: what stands in the way, what drifts by, and how
// often the ground has a tuft.
type sceneArt struct {
	obstacles []sprite
	cloud     sprite
	tuftEvery uint32
}

// The T-Rex, facing the way it runs, twelve wide and fourteen tall.
var trexBody = sprite{
	".......#####",
	".......#.###",
	".......#####",
	".......####.",
	".......###..",
	"#.....######",
	"##...######.",
	"###.#######.",
	".##########.",
	"..#########.",
	"...#######..",
	"....#####...",
}

var trexArt = runnerArt{
	run: [2]sprite{
		append(append(sprite{}, trexBody...), "....##.#....", "....#..##..."),
		append(append(sprite{}, trexBody...), "....#.##....", "....##..#..."),
	},
	air:   append(append(sprite{}, trexBody...), "....##.##...", "....#...#..."),
	count: 1,
}

var twoTRexArt = runnerArt{run: trexArt.run, air: trexArt.air, count: 2}

var (
	cactus = sprite{
		".#.",
		"#.#",
		"###",
		".#.",
		".#.",
	}
	tallCactus = sprite{
		"..#..",
		"#.#..",
		"#.#.#",
		"###.#",
		"..###",
		"..#..",
		"..#..",
	}
	cloudArt = sprite{
		"..##..###.",
		"##########",
	}
	grassland = sceneArt{
		obstacles: []sprite{
			cactus,
			beside(1, cactus, cactus),
			beside(1, cactus, cactus, cactus),
			tallCactus,
		},
		cloud:     cloudArt,
		tuftEvery: 5,
	}

	// The desert's pyramids: stepped, three to five high.
	pyramid = sprite{
		"..#..",
		".###.",
		"#####",
	}
	bigPyramid = sprite{
		"...#...",
		"..###..",
		".#####.",
		"#######",
	}
	greatPyramid = sprite{
		"....#....",
		"...###...",
		"..#####..",
		".#######.",
		"#########",
	}
	desert = sceneArt{
		obstacles: []sprite{
			pyramid,
			bigPyramid,
			greatPyramid,
			beside(1, pyramid, bigPyramid),
		},
		cloud:     cloudArt,
		tuftEvery: 11, // sand: fewer specks
	}
)

// runnerOf and sceneOf are the art a name picks; an unknown name is the
// first choice, as a saver with no such setting would be.
func runnerOf(name string) runnerArt {
	if name == RunnerTwoTRex {
		return twoTRexArt
	}
	return trexArt
}

func sceneOf(name string) sceneArt {
	if name == SceneDesert {
		return desert
	}
	return grassland
}

// Dino is one run in progress.
type Dino struct {
	rng    *rand.Rand
	runner runnerArt
	scene  sceneArt
	w, h   int   // the scene as last drawn; nothing runs before the first draw
	t      int   // frames run
	dist   int   // pixels the world has moved: the ground's tufts scroll by it
	air    []int // one a runner: -1 on the ground, else how far into the arc
	obs    []obstacle
	gap    int // pixels until the next obstacle
	clouds []cloud
}

type obstacle struct{ x, kind int }
type cloud struct{ x, y int }

// NewDino is a run from its first frame, with the art runner and scene
// name. The same seed is the same run.
func NewDino(seed uint64, runner, scene string) *Dino {
	d := &Dino{
		rng:    rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
		runner: runnerOf(runner),
		scene:  sceneOf(scene),
		gap:    firstGap,
	}
	d.air = make([]int, d.runner.count)
	for i := range d.air {
		d.air[i] = -1
	}
	return d
}

// Next is when the next frame is due.
func (d *Dino) Next(now time.Time) time.Time { return now.Add(DinoFrame) }

func (d *Dino) groundY() int { return d.h - groundH }

// runnerX is where runner i stands: the first a sixth of the way in,
// each next one a runner's width and a gap ahead of it.
func (d *Dino) runnerX(i int) int { return d.w/6 + i*(d.runner.air.w()+runnerGap) }

// lift is how far above the ground runner i is this frame.
func (d *Dino) lift(i int) int {
	if d.air[i] >= 0 {
		return arc[d.air[i]]
	}
	return 0
}

// Step moves the world one frame on: the ground and the obstacles scroll,
// one may come into view, the clouds drift, and each runner goes on
// with its jump, lands, or decides to jump.
func (d *Dino) Step() {
	if d.w == 0 {
		return
	}
	d.t++
	d.dist += speed
	keep := d.obs[:0]
	for _, o := range d.obs {
		o.x -= speed
		if o.x+d.scene.obstacles[o.kind].w() > 0 {
			keep = append(keep, o)
		}
	}
	d.obs = keep
	if d.gap -= speed; d.gap <= 0 {
		d.obs = append(d.obs, obstacle{x: d.w, kind: d.rng.IntN(len(d.scene.obstacles))})
		d.gap = minGap + d.rng.IntN(maxGap-minGap+1)
	}
	if d.t%3 == 0 {
		for i := range d.clouds {
			if d.clouds[i].x--; d.clouds[i].x+d.scene.cloud.w() < 0 {
				d.clouds[i] = d.newCloud(d.w + d.rng.IntN(d.w))
			}
		}
	}
	for i := range d.air {
		d.step(i)
	}
}

// step is runner i's frame: on with the jump, or the decision to jump.
func (d *Dino) step(i int) {
	if d.air[i] >= 0 {
		if d.air[i]++; d.air[i] >= len(arc) {
			d.air[i] = -1
		}
		return
	}
	if o, ok := d.ahead(i); ok {
		// Jump inside the window that clears it: at its last frame, or
		// earlier by chance — each runner's own window, its own chance.
		if d.clears(i, o, 0) && (!d.clears(i, o, 1) || d.rng.IntN(6) == 0) {
			d.air[i] = 0
		}
		return
	}
	// Nothing coming, and nothing due before this jump would land: a jump
	// for the fun of it.
	if d.gap > len(arc)*speed+8 && d.rng.IntN(50) == 0 {
		d.air[i] = 0
	}
}

// ahead is the nearest obstacle runner i has not passed.
func (d *Dino) ahead(i int) (obstacle, bool) {
	var best obstacle
	found := false
	for _, o := range d.obs {
		if o.x+d.scene.obstacles[o.kind].w() > d.runnerX(i) && (!found || o.x < best.x) {
			best, found = o, true
		}
	}
	return best, found
}

// clears reports whether a jump runner i begins delay frames from now
// takes it over o and lands it past — with the runner's whole box, which
// is more careful than its shape.
func (d *Dino) clears(i int, o obstacle, delay int) bool {
	s := d.scene.obstacles[o.kind]
	dx, dw := d.runnerX(i), d.runner.air.w()
	for t := 0; ; t++ {
		ox := o.x - speed*t
		if ox+s.w() <= dx {
			return true
		}
		lift := 0
		if t >= delay && t-delay < len(arc) {
			lift = arc[t-delay]
		}
		if ox < dx+dw && lift < s.h() {
			return false
		}
	}
}

func (d *Dino) newCloud(x int) cloud {
	return cloud{x: x, y: 1 + d.rng.IntN(max(1, d.groundY()-d.runner.air.h()-arc[6]-d.scene.cloud.h()))}
}

// resize is a new scene size: the clouds find their sky again; the
// obstacles keep their places, off the edge or not.
func (d *Dino) resize(w, h int) {
	d.w, d.h = w, h
	if len(d.clouds) == 0 {
		d.clouds = []cloud{d.newCloud(d.rng.IntN(max(1, w))), d.newCloud(w/2 + d.rng.IntN(max(1, w)))}
		return
	}
	for i, c := range d.clouds {
		d.clouds[i] = d.newCloud(c.x)
	}
}

// tuft says whether the ground has a tuft at world position x, one in
// every so many: a hash, so the tufts scroll with the ground and cost
// nothing to keep.
func tuft(x int, every uint32) bool {
	h := uint32(x) * 2654435761
	return h>>24%every == 0
}

// Draw is the current frame at w × h pixels: the ground along the
// bottom, the clouds, the obstacles, and the runners where their jumps
// have them, each in the pose its stride is at — the one behind half a
// stride off the one in front.
func (d *Dino) Draw(w, h int) Scene {
	if w != d.w || h != d.h {
		d.resize(w, h)
	}
	sc := Scene{W: w, H: h, Pix: make([]bool, w*h)}
	gy := d.groundY()
	for x := 0; x < w; x++ {
		sc.set(x, gy)
		if tuft(x+d.dist, d.scene.tuftEvery) {
			sc.set(x, gy+1)
		}
	}
	for _, c := range d.clouds {
		sc.blit(d.scene.cloud, c.x, c.y)
	}
	for _, o := range d.obs {
		s := d.scene.obstacles[o.kind]
		sc.blit(s, o.x, gy-s.h())
	}
	for i := range d.air {
		pose := d.runner.air
		if d.air[i] < 0 {
			pose = d.runner.run[(d.t/3+i)%2]
		}
		sc.blit(pose, d.runnerX(i), gy-pose.h()-d.lift(i))
	}
	return sc
}
