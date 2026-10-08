package saver

import (
	"math/rand/v2"
	"slices"
	"time"
)

// The dino run (function.md §5.2): the offline game as a screensaver. The
// runner runs, the ground and the obstacles scroll past, and it jumps
// them by itself, for ever (user, 2026-10-07: the runner, as a kind of
// saver; it was the dino). Nobody plays it: a jump is timed to clear
// what is coming, at a random moment inside the window that clears it,
// and now and then there is a jump for nothing when the way is clear
// (user, 2026-09-24: endless, random obstacles, random jumps). Three
// of its settings pick the art — the participants, the character and
// the scene. Participants: a big one, a small one, or two of either size
// one behind the other, each jumping on its own (user, 2026-09-25: the
// six ways; they were the runner till 2026-10-07). Characters: the
// T-Rex, a cat, a rabbit, a giraffe, Pac-Man's ghost (user, 2026-10-06;
// the giraffe was a horse till the day after), any of them in any of
// the six. Scenes:
// grassland, with cacti; the desert, with pyramids; the city (user,
// 2026-10-08), with bungalows, blocks and skyscrapers — each's in three
// sizes, small, medium and large (user, 2026-09-25: there had been
// only small and medium), and the jump is as high as the size asks
// (user, the same day: it had been one height over everything).
//
// Everything here is in the scene's own pixels; the canvas scales them.

// The kinds of saver there are — the classes, in the user's word (2026-09-24);
// a profile is one of them set up under a name — and the runner's
// participants and scenes.
const (
	KindClock  = "clock"
	KindRunner = "runner"

	// Participants' names read their figures back to front — left to right
	// on the screen (user, 2026-09-25): small-big is the small one
	// behind and the big one in front. Before that day there were two,
	// trex and two-trex; config reads them as big and big-small.
	RunnerBig        = "big"
	RunnerSmall      = "small"
	RunnerBigBig     = "big-big"
	RunnerSmallSmall = "small-small"
	RunnerSmallBig   = "small-big"
	RunnerBigSmall   = "big-small"
	SceneGrass       = "grassland"
	SceneDesert      = "desert"
	SceneCity        = "city"

	// Who runs (user, 2026-10-06): the T-Rex, as it always was, or
	// another, in either size.
	CharacterTRex    = "t-rex"
	CharacterCat     = "cat"
	CharacterRabbit  = "rabbit"
	CharacterGiraffe = "giraffe"
	CharacterGhost   = "ghost"
)

var (
	Kinds        = []string{KindClock, KindRunner, KindBounce, KindSnake, KindTetromino, KindPets, KindCustom}
	Participants = []string{RunnerBig, RunnerSmall, RunnerBigBig, RunnerSmallSmall, RunnerSmallBig, RunnerBigSmall}
	Characters   = []string{CharacterTRex, CharacterCat, CharacterRabbit, CharacterGiraffe, CharacterGhost}
	Scenes       = []string{SceneGrass, SceneDesert, SceneCity}
)

// DinoFrame is the time between two frames: fourteen a second.
const DinoFrame = 70 * time.Millisecond

// dinoRoom is the scene the run needs: the runner, a jump over the
// tallest cactus, the ground, and a runway — the rows are the T-Rex,
// fourteen, its jump, eleven, the ground, two, and one of sky
// (2026-09-25: twenty-eight; twenty-five while the jump was eight,
// before the large obstacles). Drawn at most three times over.
var dinoRoom = Room{W: 40, H: 28, Most: 3}

const (
	speed     = 2   // pixels the world moves a frame
	groundH   = 2   // the ground line, and the row of tufts under it
	minGap    = 44  // the least between two obstacles: a jump, and a landing
	maxGap    = 100 // the most
	firstGap  = 60  // before the first
	runnerGap = 4   // between two runners, nose to tail
)

// tier is an obstacle's size — small, medium or large — and with it the
// jump over it: the arc of the same tier. The jump goes by the tier and
// not by the height, because the desert's pyramids are lower than the
// cacti: by height alone the great pyramid would take the small jump
// and the huge one the middling, and the user asked for the jump to go
// with the obstacle's size.
type tier int

const (
	small tier = iota
	medium
	large
)

// arcs are the jumps, one a tier: how far the runner is above the ground,
// frame by frame. Each is as high as its tier asks — a pixel over the
// tallest obstacle of that tier in either scene: six over the small
// cactus, eight over the tall one, eleven over the big one; the pyramids
// are lower, and the jumps stand off them by more (user, 2026-09-25: the
// height must go with the obstacle's size; there had been the one jump,
// eleven high, over everything — eight, before the large tier). Every
// jump is the same slow sixteen frames, because a small obstacle can be
// as wide as a large one — three cacti in a row are eleven pixels, the
// pair of pyramids thirteen, as wide as anything gets — and at two
// pixels a frame the runner, twelve wide, needs twelve frames above the
// widest: a lower jump is lower, not shorter.
var arcs = [...][]int{
	small:  {2, 4, 5, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 5, 4, 2},
	medium: {3, 5, 7, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 7, 5, 3},
	large:  {3, 6, 8, 10, 11, 11, 11, 11, 11, 11, 11, 11, 10, 8, 6, 3},
}

// arcTop is the highest jump's height: the sky starts above the runner
// at it.
var arcTop = max(slices.Max(arcs[small]), slices.Max(arcs[medium]), slices.Max(arcs[large]))

// A sprite is rows of '#' lit and '.' dark, top to bottom, all one width.
// An 'e' is dark too: an eye, which the night lights (user, 2026-10-07);
// and a 'w', a window, which the night lights too (user, 2026-10-08).
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

// runnerArt is a runner: the figures that run, back to front, each
// jumping on its own (user, 2026-09-24: two of them, one behind the
// other; 2026-09-25: either size, in either order, or one alone).
type runnerArt struct {
	figures []figure
}

// figure is one runner's art: two running poses and the one in the air.
type figure struct {
	run [2]sprite
	air sprite
}

func (a runnerArt) count() int { return len(a.figures) }

// tallest is the tallest figure's height: the sky starts above it.
func (a runnerArt) tallest() int {
	h := 0
	for _, f := range a.figures {
		h = max(h, f.air.h())
	}
	return h
}

// sceneArt is a scene: what stands in the way, what drifts by, and how
// often the ground has a tuft.
type sceneArt struct {
	obstacles []obstacleArt
	cloud     sprite
	tuftEvery uint32
}

// obstacleArt is one thing in the way: its sprite, and its tier — which
// is the jump over it.
type obstacleArt struct {
	sprite
	tier tier
}

// The T-Rex, facing the way it runs, twelve wide and fourteen tall.
var trexBody = sprite{
	".......#####",
	".......#e###",
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

var trex = figure{
	run: [2]sprite{
		append(append(sprite{}, trexBody...), "....##.#....", "....#..##..."),
		append(append(sprite{}, trexBody...), "....#.##....", "....##..#..."),
	},
	air: append(append(sprite{}, trexBody...), "....##.##...", "....#...#..."),
}

// A small T-Rex, eight wide and ten tall.
var smallTRexBody = sprite{
	"....####",
	"....#e##",
	"....####",
	"....###.",
	"#..#####",
	"##.####.",
	".######.",
	"..#####.",
}

var smallTRex = figure{
	run: [2]sprite{
		append(append(sprite{}, smallTRexBody...), "...#.#..", "...#..#."),
		append(append(sprite{}, smallTRexBody...), "...#.#..", "..#..#.."),
	},
	air: append(append(sprite{}, smallTRexBody...), "...#.#..", "...#.#.."),
}

// The other characters (user, 2026-10-06), each as wide as the T-Rex
// of its size — twelve, or eight — which the jumps are timed to, and
// no taller: they are lower, and the run has the room it always had.

// A cat at a gallop, its tail up behind, its ears pricked: twelve wide
// and nine tall.
var catBody = sprite{
	"#.......#..#",
	"#.......####",
	".#......#e##",
	".#......####",
	"..##########",
	"..#########.",
	"...########.",
}

var cat = figure{
	run: [2]sprite{
		append(append(sprite{}, catBody...), "..#......#..", ".#........#."),
		append(append(sprite{}, catBody...), "....#...#...", "....#...#..."),
	},
	air: append(append(sprite{}, catBody...), ".##......##.", "#..........#"),
}

// A small cat, the cat made smaller: eight wide and seven tall (user,
// 2026-10-08: it had no eye; six tall till then, its head one row under
// the ears, where an eye would have opened onto the gap between them).
var smallCatBody = sprite{
	"#...#..#",
	"#...####",
	".#..#e##",
	"..######",
	"..#####.",
}

var smallCat = figure{
	run: [2]sprite{
		append(append(sprite{}, smallCatBody...), "..#...#.", ".#.....#"),
		append(append(sprite{}, smallCatBody...), "...#.#..", "...#.#.."),
	},
	air: append(append(sprite{}, smallCatBody...), ".#....#.", "#......#"),
}

// A rabbit, ears up, hopping: twelve wide and thirteen tall.
var rabbitBody = sprite{
	".......#.#..",
	".......#.#..",
	".......#.#..",
	"......#####.",
	"......##e###",
	"......######",
	"..#########.",
	".##########.",
	"##########..",
	".#########..",
	"..#######...",
}

var rabbit = figure{
	run: [2]sprite{
		append(append(sprite{}, rabbitBody...), "..#.....#...", ".##.....##.."),
		append(append(sprite{}, rabbitBody...), "...#...#....", "..##...##..."),
	},
	air: append(append(sprite{}, rabbitBody...), ".##......#..", "#.........#."),
}

// A small rabbit, eight wide and ten tall.
var smallRabbitBody = sprite{
	".....#.#",
	".....#.#",
	"....####",
	"....#e##",
	"..######",
	"#######.",
	".######.",
	"..####..",
}

var smallRabbit = figure{
	run: [2]sprite{
		append(append(sprite{}, smallRabbitBody...), "..#..#..", ".##..##."),
		append(append(sprite{}, smallRabbitBody...), "...##...", "..#..#.."),
	},
	air: append(append(sprite{}, smallRabbitBody...), ".#....#.", "#......#"),
}

// A giraffe at a gallop, its neck reaching forward, its tail streaming
// (drawn as a horse; user, 2026-10-07: it is more like a giraffe):
// twelve wide and twelve tall.
var giraffeBody = sprite{
	"..........#.",
	".........###",
	"........##e#",
	".......#####",
	"......###...",
	".....###....",
	"#.#######...",
	"##########..",
	".#########..",
}

var giraffe = figure{
	run: [2]sprite{
		append(append(sprite{}, giraffeBody...), "..#.....#...", ".#.......#..", "#.........#."),
		append(append(sprite{}, giraffeBody...), "..#.....#...", "...#...#....", "...#...#...."),
	},
	air: append(append(sprite{}, giraffeBody...), ".#.......#..", "#.........#.", "............"),
}

// A small giraffe, eight wide and nine tall.
var smallGiraffeBody = sprite{
	"......#.",
	".....###",
	"....#e##",
	"...###..",
	"#.####..",
	"######..",
	".#####..",
}

var smallGiraffe = figure{
	run: [2]sprite{
		append(append(sprite{}, smallGiraffeBody...), ".#...#..", "#.....#."),
		append(append(sprite{}, smallGiraffeBody...), "..#.#...", "..#.#..."),
	},
	air: append(append(sprite{}, smallGiraffeBody...), "#.....#.", "........"),
}

// Pac-Man's ghost, looking the way it goes: no legs, its skirt swaying
// from one frame to the next; twelve wide and twelve tall.
var ghostBody = sprite{
	"....####....",
	"..########..",
	".##########.",
	".####ee##ee#",
	"#####ee##ee#",
	"############",
	"############",
	"############",
	"############",
}

var ghost = figure{
	run: [2]sprite{
		append(append(sprite{}, ghostBody...), "############", "##.###.###.#", "#...#...#..."),
		append(append(sprite{}, ghostBody...), "############", "#.###.###.##", "...#...#...#"),
	},
	air: append(append(sprite{}, ghostBody...), "############", "##.###.###.#", "#...#...#..."),
}

// A small ghost, eight wide and eight tall.
var smallGhostBody = sprite{
	"..####..",
	".######.",
	"###e##e#",
	"###e##e#",
	"########",
	"########",
}

var smallGhost = figure{
	run: [2]sprite{
		append(append(sprite{}, smallGhostBody...), "########", "#.##.##."),
		append(append(sprite{}, smallGhostBody...), "########", ".##.##.#"),
	},
	air: append(append(sprite{}, smallGhostBody...), "########", "#.##.##."),
}

// formations is what each runner name puts on the ground, back to
// front: true a big one, false a small one.
var formations = map[string][]bool{
	RunnerBig:        {true},
	RunnerSmall:      {false},
	RunnerBigBig:     {true, true},
	RunnerSmallSmall: {false, false},
	RunnerSmallBig:   {false, true},
	RunnerBigSmall:   {true, false},
}

// cast is each character, big and small.
var cast = map[string][2]figure{
	CharacterTRex:    {trex, smallTRex},
	CharacterCat:     {cat, smallCat},
	CharacterRabbit:  {rabbit, smallRabbit},
	CharacterGiraffe: {giraffe, smallGiraffe},
	CharacterGhost:   {ghost, smallGhost},
}

var (
	// The grassland's cacti, in three sizes (2026-09-25 — the user saw
	// only small and medium): small, five high, one, two or three in a
	// row; tall, seven; big, ten, its trunk two wide.
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
	bigCactus = sprite{
		"..##..",
		"#.##..",
		"#.##..",
		"#.##.#",
		"#.##.#",
		"####.#",
		"..####",
		"..##..",
		"..##..",
		"..##..",
	}
	cloudArt = sprite{
		"..##..###.",
		"##########",
	}
	grassland = sceneArt{
		obstacles: []obstacleArt{
			{cactus, small},
			{beside(1, cactus, cactus), small},
			{beside(1, cactus, cactus, cactus), small},
			{tallCactus, medium},
			{bigCactus, large},
		},
		cloud:     cloudArt,
		tuftEvery: 5,
	}

	// The desert's pyramids: stepped, three to five high — and seven, the
	// large one (2026-09-25), as wide as anything gets at thirteen.
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
	hugePyramid = sprite{
		"......#......",
		".....###.....",
		"....#####....",
		"...#######...",
		"..#########..",
		".###########.",
		"#############",
	}
	desert = sceneArt{
		obstacles: []obstacleArt{
			{pyramid, small},
			{bigPyramid, small},
			{greatPyramid, medium},
			{hugePyramid, large},
			{beside(1, pyramid, bigPyramid), small},
		},
		cloud:     cloudArt,
		tuftEvery: 11, // sand: fewer specks
	}

	// The city's (user, 2026-10-08): a bungalow, small, five high, its
	// roof three rows over two windows and a door; a block, medium,
	// seven, four windows a floor; a skyscraper, large, ten, stepping in
	// twice up to its mast. The windows are dark, the sky through them,
	// and at night some are lit, the 'w's (user, the same day).
	bungalow = sprite{
		"...#####...",
		".#########.",
		"###########",
		".#.w#.#w.#.",
		".####.####.",
	}
	block = sprite{
		"#########",
		"#w#.#w#w#",
		"#########",
		"#.#w#.#w#",
		"#########",
		"#w#.#w#.#",
		"####.####",
	}
	skyscraper = sprite{
		"....#....",
		"....#....",
		"...###...",
		"...#w#...",
		"..#####..",
		"..#w#.#..",
		".#######.",
		".#w#.#w#.",
		".#w#.#w#.",
		".###.###.",
	}
	city = sceneArt{
		obstacles: []obstacleArt{
			{bungalow, small},
			{block, medium},
			{skyscraper, large},
		},
		cloud:     cloudArt,
		tuftEvery: 9,
	}
)

// runnerOf and sceneOf are the art a name picks: the runner's figures,
// back to front, drawn as the character; an unknown name is the first
// choice, as a saver with no such setting would be.
func runnerOf(name, character string) runnerArt {
	sizes, ok := formations[name]
	if !ok {
		sizes = formations[Participants[0]]
	}
	who, ok := cast[character]
	if !ok {
		who = cast[Characters[0]]
	}
	var a runnerArt
	for _, big := range sizes {
		if big {
			a.figures = append(a.figures, who[0])
		} else {
			a.figures = append(a.figures, who[1])
		}
	}
	return a
}

func sceneOf(name string) sceneArt {
	switch name {
	case SceneDesert:
		return desert
	case SceneCity:
		return city
	}
	return grassland
}

// Dino is one run in progress.
type Dino struct {
	rng    *rand.Rand
	runner runnerArt
	scene  sceneArt
	w, h   int    // the scene as last drawn; nothing runs before the first draw
	t      int    // frames run
	dist   int    // pixels the world has moved: the ground's tufts scroll by it
	air    []int  // one a runner: -1 on the ground, else how far into its arc
	jump   []tier // one a runner: the tier of its jump — which arc it is on
	obs    []obstacle
	gap    int // pixels until the next obstacle
	clouds []cloud

	background string           // one of Backgrounds
	now        func() time.Time // the clock time-shifting goes by
}

type obstacle struct{ x, kind int }
type cloud struct{ x, y int }

// NewDino is a run from its first frame, with the art participants,
// character and scene name, on a background out of Backgrounds — any
// other is time-shifting, by now. The same seed is the same run.
func NewDino(seed uint64, participants, character, scene, background string, now func() time.Time) *Dino {
	d := &Dino{
		rng:        rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
		runner:     runnerOf(participants, character),
		scene:      sceneOf(scene),
		gap:        firstGap,
		background: background,
		now:        now,
	}
	d.air = make([]int, d.runner.count())
	d.jump = make([]tier, d.runner.count())
	for i := range d.air {
		d.air[i] = -1
	}
	return d
}

// Next is when the next frame is due.
func (d *Dino) Next(now time.Time) time.Time { return now.Add(DinoFrame) }

// Room is the scene the run needs.
func (d *Dino) Room() Room { return dinoRoom }

func (d *Dino) groundY() int { return d.h - groundH }

// runnerX is where runner i stands: the first a sixth of the way in,
// each next one a gap ahead of the one behind it.
func (d *Dino) runnerX(i int) int {
	x := d.w / 6
	for j := 0; j < i; j++ {
		x += d.runner.figures[j].air.w() + runnerGap
	}
	return x
}

// runnerW is runner i's width.
func (d *Dino) runnerW(i int) int { return d.runner.figures[i].air.w() }

// lift is how far above the ground runner i is this frame.
func (d *Dino) lift(i int) int {
	if d.air[i] >= 0 {
		return arcs[d.jump[i]][d.air[i]]
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
		if d.air[i]++; d.air[i] >= len(arcs[d.jump[i]]) {
			d.air[i] = -1
		}
		return
	}
	if o, ok := d.ahead(i); ok {
		// Jump it with its tier's arc, inside the window that clears it:
		// at its last frame, or earlier by chance — each runner's own
		// window, its own chance.
		if d.clears(i, o, 0) && (!d.clears(i, o, 1) || d.rng.IntN(6) == 0) {
			d.air[i], d.jump[i] = 0, d.scene.obstacles[o.kind].tier
		}
		return
	}
	// Nothing coming, and nothing due before the longest jump would land:
	// a jump for the fun of it, any height.
	if d.gap > len(arcs[large])*speed+8 && d.rng.IntN(50) == 0 {
		d.air[i], d.jump[i] = 0, tier(d.rng.IntN(len(arcs)))
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

// clears reports whether a jump runner i begins delay frames from now,
// on the arc of o's tier, takes it over o and lands it past — with the
// runner's whole box, which is more careful than its shape.
func (d *Dino) clears(i int, o obstacle, delay int) bool {
	s := d.scene.obstacles[o.kind]
	arc := arcs[s.tier]
	dx, dw := d.runnerX(i), d.runnerW(i)
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
	return cloud{x: x, y: 1 + d.rng.IntN(max(1, d.groundY()-d.runner.tallest()-arcTop-d.scene.cloud.h()))}
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

// windows puts a sprite's windows, its 'w's, in the eyes' ink: gold at
// night, and by day and at dusk, which have no eyes to light, the sky.
func windows(sc *Scene, sp sprite, x, y int) {
	for dy, row := range sp {
		for dx := 0; dx < len(row); dx++ {
			if row[dx] == 'w' {
				sc.put(x+dx, y+dy, inkEye)
			}
		}
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
// bottom, the sun, the moon or the setting sun, the clouds in front of
// them, the obstacles, and the runners where their jumps
// have them, each in the pose its stride is at — the one behind half a
// stride off the one in front. Nothing else — no score, no clock (user,
// 2026-09-24): it is a screensaver, not a game being played.
func (d *Dino) Draw(w, h int) Scene {
	if w != d.w || h != d.h {
		d.resize(w, h)
	}
	sc := newScene(w, h)
	gy := d.groundY()
	for x := 0; x < w; x++ {
		sc.set(x, gy)
		if tuft(x+d.dist, d.scene.tuftEvery) {
			sc.set(x, gy+1)
		}
	}
	d.drawLights(&sc)
	// The clouds in their own colour (user, 2026-10-08).
	for _, c := range d.clouds {
		sc.blitIn(d.scene.cloud, c.x, c.y, inkCloud)
	}
	for _, o := range d.obs {
		s := d.scene.obstacles[o.kind]
		sc.blit(s.sprite, o.x, gy-s.h())
		windows(&sc, s.sprite, o.x, gy-s.h())
	}
	d.drawRunners(&sc, gy)
	return sc
}

// drawRunners draws the runners where their jumps have them, each in the
// pose its stride is at; and while the night is in the sky, the outline
// round each and its eyes — on the sky alone, so nothing it passes is
// drawn over — first, so the runners are on top of one another's.
func (d *Dino) drawRunners(sc *Scene, gy int) {
	type placed struct {
		pose sprite
		x, y int
	}
	var on []placed
	for i, f := range d.runner.figures {
		pose := f.air
		if d.air[i] < 0 {
			pose = f.run[(d.t/3+i)%2]
		}
		on = append(on, placed{pose, d.runnerX(i), gy - pose.h() - d.lift(i)})
	}
	if from, to, f := d.skies(); shows(from, to, f, hasOutline) {
		for _, p := range on {
			edge, eyes := rim(p.pose)
			for ink, qs := range map[uint8][][2]int{inkOutline: edge, inkEye: eyes} {
				for _, q := range qs {
					if x, y := p.x+q[0], p.y+q[1]; x >= 0 && x < sc.W && y >= 0 && y < sc.H && sc.Pix[y*sc.W+x] == 0 {
						sc.put(x, y, ink)
					}
				}
			}
		}
	}
	for _, p := range on {
		sc.blitIn(p.pose, p.x, p.y, inkRunner)
	}
}
