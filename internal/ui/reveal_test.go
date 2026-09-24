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
		for i := range to.lit {
			if from.lit[i] == to.lit[i] && r.cur.lit[i] != to.lit[i] {
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
	for i := range to.lit {
		if r.cur.lit[i] != to.lit[i] {
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
