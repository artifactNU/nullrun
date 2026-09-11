package main

import (
	"strings"

	"codeberg.org/anaseto/gruid"
)

// padTrunc pads s with spaces to width, or truncates it (by rune) if it's
// longer, so fixed-width UI rows never spill past the frame.
func padTrunc(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		return string(r[:width])
	}
	return s + strings.Repeat(" ", width-len(r))
}

// topBorder renders the frame's title bar, e.g.
// "┌─ NETWORK: OBSIDIAN LOGISTICS-GRID-14 ──────────────┐".
func topBorder(name string) string {
	prefix := "┌─ NETWORK: " + name + " "
	fill := borderWidth - len([]rune(prefix)) - 1 // room for the closing ┐
	if fill < 0 {
		fill = 0
	}
	return prefix + strings.Repeat("─", fill) + "┐"
}

func separatorBorder() string {
	return "├" + strings.Repeat("─", borderWidth-2) + "┤"
}

func bottomBorder() string {
	return "└" + strings.Repeat("─", borderWidth-2) + "┘"
}

// frameCell returns the cell for column x of a full-width border line
// (top/separator/bottom), styled uniformly.
func frameCell(line []rune, x int) gruid.Cell {
	if x < 0 || x >= len(line) {
		return gruid.Cell{Rune: ' '}
	}
	return gruid.Cell{Rune: line[x], Style: gruid.Style{Fg: ColorNode}}
}
