package ui

import (
	"fmt"
	"strconv"
	"strings"
)

// Dimming is fading what was drawn, not drawing it again (tdp F8, D2,
// v0.1.11–v0.1.12; 2026-09-28): every foreground and background colour
// in the SGR sequences of an already-styled screen goes toward the base,
// and the text, the layout, bold, the backgrounds that make a capsule, a
// cursor bar or a swatch all stay. A popup's border fades with the rest,
// and so is its own layer colour dimmed. filu's dim.go is the family's
// reference; locku's own first try stripped the colours and drew all of
// it in Overlay0, which took the capsules, the cursor bars and the
// swatches away.

// dimKeep is how much of a colour survives: dim(c) = c × 0.45 + base × 0.55
// (tdp D2).
const dimKeep = 0.45

// dimFloor is what a colour fades toward: the canvas.
var dimFloor = hexRGB(baseHex)

// dimText is text that sets no colour of its own, dimmed: the terminal's
// foreground, taken as the theme's text.
var dimText = dimRGB(hexRGB(string(textColor)))

func hexRGB(hex string) [3]int {
	var c [3]int
	fmt.Sscanf(hex, "#%02x%02x%02x", &c[0], &c[1], &c[2])
	return c
}

// dimRGB fades c toward the canvas, and never lightens it: a colour darker
// than the canvas, a black board's ground, stays as it is (tdp D2,
// v0.1.12).
func dimRGB(c [3]int) [3]int {
	var out [3]int
	for i := range c {
		out[i] = min(c[i], int(float64(c[i])*dimKeep+float64(dimFloor[i])*(1-dimKeep)+0.5))
	}
	return out
}

func sgrRGB(fg bool, c [3]int) string {
	lead := "48"
	if fg {
		lead = "38"
	}
	return fmt.Sprintf("%s;2;%d;%d;%d", lead, c[0], c[1], c[2])
}

// ansi16 is the xterm palette for the 16 basic colours.
var ansi16 = [16][3]int{
	{0, 0, 0}, {205, 0, 0}, {0, 205, 0}, {205, 205, 0}, {0, 0, 238}, {205, 0, 205}, {0, 205, 205}, {229, 229, 229},
	{127, 127, 127}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0}, {92, 92, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
}

// xterm256 is a 256-colour index as RGB: the output is 24-bit whatever
// came in (tdp D2, D6: the family needs a truecolor terminal).
func xterm256(n int) [3]int {
	switch {
	case n < 16:
		return ansi16[n]
	case n < 232:
		n -= 16
		step := func(v int) int {
			if v == 0 {
				return 0
			}
			return 55 + v*40
		}
		return [3]int{step(n / 36), step(n / 6 % 6), step(n % 6)}
	default:
		v := 8 + (n-232)*10
		return [3]int{v, v, v}
	}
}

// dimANSI is s, already styled, dimmed: every colour in its SGR sequences
// faded, and each line begun in the dimmed text colour for text that
// sets none. Everything else — the text, cursor movement, bold, reverse —
// is as it was.
func dimANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s) + len(s)/4)
	lineStart := "\x1b[" + sgrRGB(true, dimText) + "m"
	b.WriteString(lineStart)
	for i := 0; i < len(s); {
		switch {
		case s[i] == '\n':
			b.WriteByte('\n')
			b.WriteString(lineStart)
			i++
		case strings.HasPrefix(s[i:], "\x1b["):
			end := i + 2
			for end < len(s) && (s[end] < 0x40 || s[end] > 0x7e) {
				end++
			}
			if end >= len(s) {
				b.WriteString(s[i:])
				return b.String()
			}
			if s[end] == 'm' {
				b.WriteString("\x1b[" + dimSGR(s[i+2:end]) + "m")
			} else {
				b.WriteString(s[i : end+1])
			}
			i = end + 1
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// dimSGR is one SGR parameter list with its colours dimmed. A reset, or
// the default foreground, is followed by the dimmed text colour, so what
// comes after it stays dim.
func dimSGR(params string) string {
	ps := strings.Split(params, ";")
	var out []string
	needText := false
	for i := 0; i < len(ps); i++ {
		p := ps[i]
		n, err := strconv.Atoi(p)
		if p == "" {
			n, err = 0, nil
		}
		if err != nil {
			out = append(out, p)
			continue
		}
		switch {
		case n == 0:
			out = append(out, "0")
			needText = true
		case n == 39:
			needText = true
		case (n == 38 || n == 48) && i+1 < len(ps):
			fg := n == 38
			var c [3]int
			switch ps[i+1] {
			case "2":
				if i+4 >= len(ps) {
					out = append(out, ps[i:]...)
					i = len(ps)
					continue
				}
				for k := 0; k < 3; k++ {
					c[k], _ = strconv.Atoi(ps[i+2+k])
				}
				i += 4
			case "5":
				if i+2 >= len(ps) {
					out = append(out, ps[i:]...)
					i = len(ps)
					continue
				}
				idx, _ := strconv.Atoi(ps[i+2])
				c = xterm256(idx)
				i += 2
			default:
				out = append(out, p)
				continue
			}
			out = append(out, sgrRGB(fg, dimRGB(c)))
			if fg {
				needText = false
			}
		case n >= 30 && n <= 37, n >= 90 && n <= 97:
			idx := n - 30
			if n >= 90 {
				idx = n - 90 + 8
			}
			out = append(out, sgrRGB(true, dimRGB(ansi16[idx])))
			needText = false
		case n >= 40 && n <= 47, n >= 100 && n <= 107:
			idx := n - 40
			if n >= 100 {
				idx = n - 100 + 8
			}
			out = append(out, sgrRGB(false, dimRGB(ansi16[idx])))
		default:
			out = append(out, p)
		}
	}
	if needText {
		out = append(out, sgrRGB(true, dimText))
	}
	return strings.Join(out, ";")
}
