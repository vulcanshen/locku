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
		{Clock{TimeHM, DateOff, LayoutRow}, []string{"21 05"}},
		{Clock{TimeHMS, DateOff, LayoutRow}, []string{"21 05 09"}},
		{Clock{TimeHM, DateYMD, LayoutRow}, []string{"21 05", "2026-09-24"}},
		{Clock{TimeHM, DateYMonD, LayoutRow}, []string{"21 05", "2026-SEP-24"}},
		{Clock{TimeHM, DateMD, LayoutRow}, []string{"21 05", "09-24"}},
		{Clock{TimeHM, DateMonD, LayoutRow}, []string{"21 05", "SEP-24"}},
		{Clock{"bogus", "bogus", "bogus"}, []string{"21 05"}},
		// The spellings a file may still carry.
		{Clock{"HH:MM:SS", DateOff, LayoutRow}, []string{"21 05 09"}},
		{Clock{"HH:MM AM/PM", DateOff, LayoutRow}, []string{"21 05"}},
		{Clock{"HH:MM:SS AM/PM", DateOff, LayoutRow}, []string{"21 05 09"}},
		{Clock{TimeHMS, DateOff, LayoutColumn}, []string{"21", "05", "09"}},
		{Clock{TimeHM, DateYMonD, LayoutColumn}, []string{"21", "05", "2026", "SEP", "24"}},
		{Clock{TimeHM, DateMD, LayoutColumn}, []string{"21", "05", "09", "24"}},
	}
	for _, c := range cases {
		if got := c.c.Lines(at); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%+v: got %q want %q", c.c, got, c.want)
		}
	}
}

func TestSteps(t *testing.T) {
	steps := func(c Clock) []Clock {
		var out []Clock
		for _, s := range c.Steps() {
			out = append(out, s.(Clock))
		}
		return out
	}
	want := []Clock{
		{TimeHMS, DateYMonD, LayoutRow},
		{TimeHMS, DateMonD, LayoutRow}, // the year goes
		{TimeHM, DateMonD, LayoutRow},  // the seconds go, the date stays
		{TimeHMS, DateOff, LayoutRow},  // the date goes, the seconds are back
		{TimeHM, DateOff, LayoutRow},   // the seconds go too
	}
	if got := steps(Clock{TimeHMS, DateYMonD, LayoutRow}); !reflect.DeepEqual(got, want) {
		t.Errorf("steps %v, want %v", got, want)
	}
	if got := steps(Clock{TimeHM, DateOff, LayoutRow}); !reflect.DeepEqual(got, []Clock{{TimeHM, DateOff, LayoutRow}}) {
		t.Errorf("a bare HH MM has nothing to drop: %v", got)
	}
	if got := steps(Clock{TimeHMS, DateOff, LayoutColumn}); !reflect.DeepEqual(got, []Clock{{TimeHMS, DateOff, LayoutColumn}, {TimeHM, DateOff, LayoutColumn}}) {
		t.Errorf("seconds alone: %v", got)
	}
	if got := steps(Clock{TimeHM, DateMD, LayoutRow}); !reflect.DeepEqual(got, []Clock{{TimeHM, DateMD, LayoutRow}, {TimeHM, DateOff, LayoutRow}}) {
		t.Errorf("a short date alone: %v", got)
	}
}

func TestNext(t *testing.T) {
	if got := (Clock{TimeHMS, DateOff, LayoutRow}).Next(at); !got.Equal(at.Add(time.Second).Truncate(time.Second)) {
		t.Errorf("seconds: %v", got)
	}
	if got := (Clock{TimeHM, DateOff, LayoutRow}).Next(at); !got.Equal(time.Date(2026, 9, 24, 21, 6, 0, 0, time.UTC)) {
		t.Errorf("minutes: %v", got)
	}
	if got := (Clock{"HH:MM:SS", DateOff, LayoutRow}).Next(at); !got.Equal(at.Add(time.Second).Truncate(time.Second)) {
		t.Errorf("an old spelling with seconds must tick by the second: %v", got)
	}
}
