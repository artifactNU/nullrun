package main

import "codeberg.org/anaseto/gruid"

// game is the application's Model. It implements gruid.Model.
type game struct {
	walls  [][]bool // [y][x]; true means blocked
	player gruid.Point
	grid   gruid.Grid
}

func newGame() *game {
	return &game{
		walls:  buildMap(),
		player: gruid.Point{X: 5, Y: 5},
		grid:   gruid.NewGrid(mapWidth, mapHeight),
	}
}

// blocked reports whether p is outside the map or a wall.
func (g *game) blocked(p gruid.Point) bool {
	if p.X < 0 || p.X >= mapWidth || p.Y < 0 || p.Y >= mapHeight {
		return true
	}
	return g.walls[p.Y][p.X]
}

// Update implements gruid.Model.Update.
func (g *game) Update(msg gruid.Msg) gruid.Effect {
	switch msg := msg.(type) {
	case gruid.MsgInit:
		return nil
	case gruid.MsgKeyDown:
		return g.updateKeyDown(msg)
	case gruid.MsgQuit:
		return gruid.End()
	}
	return nil
}

func (g *game) updateKeyDown(msg gruid.MsgKeyDown) gruid.Effect {
	var dx, dy int
	switch msg.Key {
	case gruid.KeyArrowLeft, "h":
		dx = -1
	case gruid.KeyArrowRight, "l":
		dx = 1
	case gruid.KeyArrowUp, "k":
		dy = -1
	case gruid.KeyArrowDown, "j":
		dy = 1
	case gruid.KeyEscape, "q":
		return gruid.End()
	default:
		return nil
	}
	next := g.player.Shift(dx, dy)
	if !g.blocked(next) {
		g.player = next
	}
	return nil
}

// Draw implements gruid.Model.Draw.
func (g *game) Draw() gruid.Grid {
	g.grid.Map(func(p gruid.Point, _ gruid.Cell) gruid.Cell {
		switch {
		case p == g.player:
			return gruid.Cell{Rune: '@', Style: gruid.Style{Fg: ColorPlayer}}
		case g.walls[p.Y][p.X]:
			return gruid.Cell{Rune: '#', Style: gruid.Style{Fg: ColorWall}}
		default:
			return gruid.Cell{Rune: '.', Style: gruid.Style{Fg: ColorFloor}}
		}
	})
	return g.grid
}
