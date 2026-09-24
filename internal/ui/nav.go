package ui

// List navigation, in one place because it is one vocabulary (ux.md §3):
// every list in the app answers to the same keys, and a letter hotkey may
// not quietly take one of them away.

// navKeys is that vocabulary as a set, so a test can check that no action
// declares one of them (TestNoActionClaimsANavigationKey). `u` and `d`
// move half a page, which is why delete is on `x` and not `d` (ux.md §A.1).
var navKeys = map[string]bool{
	"j": true, "k": true, "down": true, "up": true,
	"u": true, "d": true, "ctrl+u": true, "ctrl+d": true,
	"g": true, "gg": true, "G": true,
}

// moveCursor resolves one navigation key against a list of n items.
//
// page is how many rows are on screen; `u`/`d` move by HALF of it.
//
// j and k WRAP when wrap is set — a menu is a ring — and stop at the ends
// when it is not: a panel's list stops (ux.md §1, §3). u, d, gg and G never
// wrap: a half-page that silently teleports to the other end is worse than
// one that stops.
func moveCursor(cur, n int, k string, page int, wrap bool) int {
	if n == 0 {
		return 0
	}
	half := max(1, page/2)
	switch k {
	case "j", "down":
		if wrap {
			return (cur + 1) % n
		}
		return min(n-1, cur+1)
	case "k", "up":
		if wrap {
			return (cur - 1 + n) % n
		}
		return max(0, cur-1)
	case "d", "ctrl+d":
		return min(n-1, cur+half)
	case "u", "ctrl+u":
		return max(0, cur-half)
	case "gg":
		return 0
	case "G":
		return n - 1
	}
	return cur
}

// moveScroll is the same vocabulary against a VIEWPORT — a thing with a top
// and no cursor, like the help text. It never wraps.
//
// last is the furthest the top may go, so the final line stays on screen.
func moveScroll(top, last int, k string, page int) int {
	if last <= 0 {
		return 0
	}
	half := max(1, page/2)
	switch k {
	case "j", "down":
		top++
	case "k", "up":
		top--
	case "d", "ctrl+d":
		top += half
	case "u", "ctrl+u":
		top -= half
	case "gg":
		return 0
	case "G":
		return last
	}
	return clamp(top, 0, last)
}
