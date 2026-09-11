package main

import (
	"strings"

	"codeberg.org/anaseto/gruid"
)

// splashLetters holds 5x5 block-glyphs for the letters in "NULLRUN". Built
// from a table instead of hand-typed lines so the shapes stay easy to
// proofread and tweak.
var splashLetters = map[rune][5]string{
	'N': {
		"█   █",
		"██  █",
		"█ █ █",
		"█  ██",
		"█   █",
	},
	'U': {
		"█   █",
		"█   █",
		"█   █",
		"█   █",
		" ███ ",
	},
	'L': {
		"█    ",
		"█    ",
		"█    ",
		"█    ",
		"█████",
	},
	'R': {
		"████ ",
		"█   █",
		"████ ",
		"█  █ ",
		"█   █",
	},
}

// splashLogo renders word as a 5-line block-letter banner, one space
// between letters.
func splashLogo(word string) []string {
	lines := make([]string, 5)
	for row := 0; row < 5; row++ {
		var b strings.Builder
		for i, r := range word {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(splashLetters[r][row])
		}
		lines[row] = b.String()
	}
	return lines
}

const (
	splashSubtitle = "a cyberpunk hacking roguelike for your terminal"
	splashPrompt   = "press any key to jack in..."
)

// drawSplash renders the title screen: the NULLRUN logo, a subtitle, and a
// prompt, all centered on their own grid sized to fit them.
func (g *game) drawSplash() gruid.Grid {
	lines := append([]string{""}, splashLogo("NULLRUN")...)
	lines = append(lines, "", splashSubtitle, "", splashPrompt)

	width := borderWidth
	g.grid = g.grid.Resize(width, len(lines))
	g.grid.Fill(gruid.Cell{Rune: ' '})

	for y, line := range lines {
		col := ColorPlayer
		switch line {
		case splashSubtitle:
			col = ColorEdge
		case splashPrompt:
			col = ColorData
		}
		runes := []rune(line)
		start := (width - len(runes)) / 2
		for i, r := range runes {
			x := start + i
			if x < 0 || x >= width {
				continue
			}
			g.grid.Set(gruid.Point{X: x, Y: y}, gruid.Cell{Rune: r, Style: gruid.Style{Fg: col}})
		}
	}
	return g.grid
}
