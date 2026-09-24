package saver

import (
	"reflect"
	"testing"
	"time"
)

var at = time.Date(2026, time.September, 24, 21, 5, 9, 0, time.UTC)

func TestLines(t *testing.T) {
	cases := []struct {
		c    Clock
		want []string
	}{
		{Clock{TimeHM, DateOff}, []string{"21:05"}},
		{Clock{TimeHM12, DateOff}, []string{"09:05 PM"}},
		{Clock{TimeHMS, DateOff}, []string{"21:05:09"}},
		{Clock{TimeHMS12, DateOff}, []string{"09:05:09 PM"}},
		{Clock{TimeHM, DateYMD}, []string{"21:05", "2026-09-24"}},
		{Clock{TimeHM, DateYMonD}, []string{"21:05", "2026-SEP-24"}},
		{Clock{TimeHM, DateMD}, []string{"21:05", "09-24"}},
		{Clock{TimeHM, DateMonD}, []string{"21:05", "SEP-24"}},
		{Clock{"bogus", "bogus"}, []string{"21:05"}},
	}
	for _, c := range cases {
		if got := c.c.Lines(at); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%+v: got %q want %q", c.c, got, c.want)
		}
	}
}

func TestDegradeLadder(t *testing.T) {
	var steps []Clock
	var s Saver = Clock{TimeHMS12, DateYMonD}
	for {
		steps = append(steps, s.(Clock))
		next, ok := s.Degrade()
		if !ok {
			break
		}
		s = next
	}
	want := []Clock{
		{TimeHMS12, DateYMonD},
		{TimeHMS12, DateMonD}, // the year first
		{TimeHM12, DateMonD},  // then the seconds, the 12-hour shape kept
		{TimeHM12, DateOff},   // then the date
	}
	if !reflect.DeepEqual(steps, want) {
		t.Errorf("ladder %v, want %v", steps, want)
	}
	if _, ok := (Clock{TimeHM, DateOff}).Degrade(); ok {
		t.Error("a bare HH:MM has nothing to drop")
	}
}

func TestNext(t *testing.T) {
	if got := (Clock{TimeHMS, DateOff}).Next(at); !got.Equal(at.Add(time.Second).Truncate(time.Second)) {
		t.Errorf("seconds: %v", got)
	}
	if got := (Clock{TimeHM, DateOff}).Next(at); !got.Equal(time.Date(2026, 9, 24, 21, 6, 0, 0, time.UTC)) {
		t.Errorf("minutes: %v", got)
	}
}
