//go:build darwin || linux

package ui

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/unix"
)

// DetectIconWidth measures how far a Nerd Font icon moves the cursor and
// sets iconCells, so every width counts it right (tdp D6, 2026-09-29;
// filu's iconwidth_unix.go is the reference). Most fonts move it one
// cell; some made for CJK move it two, while lipgloss and x/ansi count
// one. It prints an icon at column 1 and asks the terminal where the
// cursor is (CPR). The icon is nf-fa-folder, as filu's: every icon is as
// wide as the next on such a font, and the board's own square would read
// as a pixel of the board drawn over a custom saver's program.
// LOCKU_ICON_WIDTH, 1 or 2, overrides it. Any failure — not a terminal,
// no answer within 200 ms — leaves iconCells at 1. Call it once, before
// the program starts; on a lock, before termreply.DropPending, which
// takes an answer that came too late.
func DetectIconWidth() {
	if v := os.Getenv("LOCKU_ICON_WIDTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 2 {
			iconCells = n
			return
		}
	}
	in, out := os.Stdin, os.Stdout
	if !term.IsTerminal(in.Fd()) || !term.IsTerminal(out.Fd()) {
		return
	}
	state, err := term.MakeRaw(in.Fd())
	if err != nil {
		return
	}
	defer term.Restore(in.Fd(), state)

	icon := string(rune(0xf07b)) // nf-fa-folder
	if _, err := out.WriteString("\r" + icon + "\x1b[6n"); err != nil {
		return
	}
	col, ok := readCPRColumn(int(in.Fd()))
	out.WriteString("\r\x1b[2K") // the probe goes before the program takes the screen
	if ok && col >= 2 {
		iconCells = col - 1 // from column 1, the icon took col-1 cells
	}
}

// readCPRColumn reads a CPR answer, "\x1b[<row>;<col>R", within 200 ms
// and returns col. poll keeps a terminal that never answers from holding
// up the start.
func readCPRColumn(fd int) (int, bool) {
	deadline := time.Now().Add(200 * time.Millisecond)
	var buf []byte
	one := make([]byte, 1)
	for {
		ms := int(time.Until(deadline).Milliseconds())
		if ms <= 0 {
			return 0, false
		}
		n, err := unix.Poll([]unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}, ms)
		if err == unix.EINTR {
			continue
		}
		if err != nil || n == 0 {
			return 0, false
		}
		if _, err := unix.Read(fd, one); err != nil {
			return 0, false
		}
		buf = append(buf, one[0])
		if one[0] == 'R' {
			return parseCPRColumn(buf)
		}
		if len(buf) > 32 {
			return 0, false
		}
	}
}

// parseCPRColumn takes col out of "…[<row>;<col>R".
func parseCPRColumn(buf []byte) (int, bool) {
	s := string(buf)
	open := strings.IndexByte(s, '[')
	semi := strings.IndexByte(s, ';')
	end := strings.IndexByte(s, 'R')
	if open < 0 || semi < open || end < semi {
		return 0, false
	}
	col, err := strconv.Atoi(s[semi+1 : end])
	if err != nil {
		return 0, false
	}
	return col, true
}
