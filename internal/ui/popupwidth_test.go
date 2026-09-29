package ui

import (
	"strings"
	"testing"
	"time"
)

// Every popup is min(terminal width - 2, 120) wide, whatever it holds
// (tdp F7, 2026-09-28): 78 on 80 columns, 98 on 100, 120 on 200 — the PIN
// boxes too, which were 48 — and a box does not grow as it is typed in.
func TestEveryPopupIsF7Wide(t *testing.T) {
	open := popupAnimator{phase: animOpen}
	long := strings.Repeat("a very long description that goes on ", 8)
	for _, tw := range []struct{ screen, want int }{{80, 78}, {100, 98}, {200, 120}} {
		W, H := tw.screen, 40
		menu := newSpaceMenu()
		menu.setSize(W, H)
		menu.setItems([]menuItem{{label: "Edit", key: "enter", hint: "its rows"}}, "[1] locku", 1)
		menu.anim = open
		help := helpPopup{anim: open, entries: []helpEntry{{key: "x", desc: "short"}, {key: "y", desc: long}}, screenW: W, screenH: H}
		views := map[string]string{
			"Space menu": menu.view(),
			"confirm":    confirmPopup{anim: open, title: "Delete", accept: "delete", lines: []string{"Delete clock?"}, screenW: W, screenH: H}.view(),
			"input":      inputPopup{anim: open, title: "name", prompt: "name", screenW: W, screenH: H}.view(),
			"input, 90 typed": inputPopup{anim: open, title: "name", prompt: "name", value: strings.Repeat("n", 90),
				screenW: W, screenH: H}.view(),
			"PIN input":  inputPopup{anim: open, title: "new PIN", masked: true, value: "1234", screenW: W, screenH: H}.view(),
			"help":       help.view(),
			"toast":      toastModel{anim: open, msg: "PIN set", screenW: W, screenH: H}.view(),
			"PIN prompt": pinPrompt{anim: open, value: []rune("1234"), screenW: W, screenH: H}.view(time.Now()),
		}
		for name, v := range views {
			for i, line := range strings.Split(v, "\n") {
				if got := dispW(line); got != tw.want {
					t.Errorf("%d columns, %s, line %d: %d wide, want %d:\n%s", W, name, i, got, tw.want, v)
					break
				}
			}
		}
	}
}
