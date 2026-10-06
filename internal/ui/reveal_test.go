package ui

import "testing"

func TestRevealTouchesOnlyWhatChanged(t *testing.T) {
	from := paint(faceTall, one([]string{"21 05"}, 2), 120, 39)
	to := paint(faceTall, one([]string{"21 06"}, 2), 120, 39)
	r := newReveal(from, to)
	if r == nil {
		t.Fatal("no reveal for a changed board")
	}
	frames := 0
	for {
		frames++
		done := r.advance()
		for i := range to.ink {
			if from.ink[i] == to.ink[i] && r.cur.ink[i] != to.ink[i] {
				t.Fatalf("frame %d changed an unchanged pixel %d", frames, i)
			}
		}
		if done {
			break
		}
		if frames > revealFrames {
			t.Fatalf("still going after %d frames", frames)
		}
	}
	for i := range to.ink {
		if r.cur.ink[i] != to.ink[i] {
			t.Fatalf("pixel %d not revealed", i)
		}
	}
	if frames > revealFrames {
		t.Errorf("%d frames, budget %d", frames, revealFrames)
	}
}

func TestRevealIsNilWhenNothingToDo(t *testing.T) {
	a := paint(faceTall, one([]string{"21 05"}, 2), 120, 39)
	if newReveal(a, a.clone()) != nil {
		t.Error("same board")
	}
	if newReveal(a, paint(faceTall, one([]string{"21 05"}, 1), 80, 23)) != nil {
		t.Error("different size")
	}
}

// A reveal lights and darkens pixels one by one, but a pixel lit on both
// boards wears the new one's ink from the first frame.
func TestRevealRecoloursAtOnce(t *testing.T) {
	from, to := newBoard(3, 1), newBoard(3, 1)
	from.put(0, 0, inkOn)
	from.put(1, 0, inkOn)
	to.put(0, 0, inkAccent)
	to.put(2, 0, inkOn)
	r := newReveal(from, to)
	if r == nil || len(r.order) != 2 {
		t.Fatalf("reveal %+v", r)
	}
	if r.cur.ink[0] != inkAccent || r.cur.ink[1] != inkOn || r.cur.ink[2] != inkOff {
		t.Errorf("first frame %v", r.cur.ink)
	}
}
