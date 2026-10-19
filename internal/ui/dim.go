package ui

import "fmt"

// hexRGB is a #rrggbb colour's three channels.
func hexRGB(hex string) [3]int {
	var c [3]int
	fmt.Sscanf(hex, "#%02x%02x%02x", &c[0], &c[1], &c[2])
	return c
}
