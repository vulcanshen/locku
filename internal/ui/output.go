package ui

import (
	"bytes"
	"os"
)

// eraseRight is what bubbletea's renderer appends to a line it measures
// shorter than the terminal: erase to the end of the line.
var eraseRight = []byte("\x1b[K")

// ScreenOutput is the terminal a lock program draws on, minus every
// eraseRight (tdp E5). bubbletea v1 measures a line with x/ansi, an icon
// one cell; where the terminal gives an icon two, the board's full rows
// measure short and the renderer sends eraseRight after them, the cursor on
// the last column, and a terminal that follows xterm erases that last
// cell. Every row the lock draws is the full width (tdp L4), so the erase
// has nothing to clear.
//
// The lib's App.Run does this for its own program (screenOutput); the lock
// runs the App in a program of its own, on its own reader of keys
// (LockInput), so it brings the same.
type ScreenOutput struct{ *os.File }

func (o ScreenOutput) Write(p []byte) (int, error) {
	if !bytes.Contains(p, eraseRight) {
		return o.File.Write(p)
	}
	// The renderer writes a whole frame at once: eraseRight is never split
	// across two writes.
	_, err := o.File.Write(bytes.ReplaceAll(p, eraseRight, nil))
	return len(p), err
}
