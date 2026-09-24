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

func TestBlocks(t *testing.T) {
	cases := []struct {
		c    Clock
		want []Block
	}{
		{Clock{TimeHMS, DateYMonD, LayoutRow}, []Block{
			{Variants: [][]string{{"21 05 09"}, {"21 05"}}},
			{Variants: [][]string{{"2026-SEP-24"}, {"SEP-24"}}},
		}},
		{Clock{TimeHM, DateOff, LayoutRow}, []Block{{Variants: [][]string{{"21 05"}}}}},
		{Clock{TimeHMS, DateOff, LayoutColumn}, []Block{{Variants: [][]string{{"21", "05", "09"}, {"21", "05"}}}}},
		{Clock{TimeHM, DateMD, LayoutColumn}, []Block{
			{Variants: [][]string{{"21", "05"}}},
			{Variants: [][]string{{"09", "24"}}},
		}},
	}
	for _, c := range cases {
		if got := c.c.Blocks(at); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%+v: got %v want %v", c.c, got, c.want)
		}
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
