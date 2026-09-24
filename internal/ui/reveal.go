package ui

import (
	"math/rand/v2"
	"time"
)

// A reveal is how the board changes after the first frame (function.md
// §5.3 / §10.18): only the pixels that differ between the board on screen
// and the one it is becoming are touched, in a shuffled order, over a few
// frames — the family splash's reveal, applied to a diff. A pixel that is
// the same in both never flickers, and a whole change lands inside 400 ms.
const (
	revealFrames = 24
	revealStep   = 16 * time.Millisecond // 24 × 16 ms = 384 ms
)

type reveal struct {
	cur   board // what is on screen; mutated frame by frame
	to    board
	order []int // the differing pixels, shuffled
	done  int
	step  int // pixels flipped per frame
}

// newReveal plans the change from one board to another. nil when there is
// nothing to animate: the boards are different sizes (a resize redraws
// whole) or already the same.
func newReveal(from, to board) *reveal {
	if !from.same(to) {
		return nil
	}
	var order []int
	for i := range to.lit {
		if from.lit[i] != to.lit[i] {
			order = append(order, i)
		}
	}
	if len(order) == 0 {
		return nil
	}
	rand.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	return &reveal{
		cur:   from.clone(),
		to:    to,
		order: order,
		step:  (len(order) + revealFrames - 1) / revealFrames,
	}
}

// advance flips the next batch of pixels and reports whether the reveal
// is complete.
func (r *reveal) advance() bool {
	n := min(len(r.order), r.done+r.step)
	for _, i := range r.order[r.done:n] {
		r.cur.lit[i] = r.to.lit[i]
	}
	r.done = n
	return r.done >= len(r.order)
}
