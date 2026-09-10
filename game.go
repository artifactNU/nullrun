package main

import (
	"math/rand"
	"time"

	"codeberg.org/anaseto/gruid"
	"codeberg.org/anaseto/gruid/rl"
)

const fovRadius = 8

// game is the application's Model. It implements gruid.Model.
type game struct {
	net    *network
	player gruid.Point

	fov        *rl.FOV
	discovered map[gruid.Point]bool // ever been in view; fog hides the rest

	grid gruid.Grid
}

func newGame() *game {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	net := generateNetwork(rng)
	g := &game{
		net:        net,
		player:     net.nodes[0],
		fov:        rl.NewFOV(gruid.NewRange(0, 0, netWidth, netHeight)),
		discovered: map[gruid.Point]bool{},
		grid:       gruid.NewGrid(netWidth, netHeight),
	}
	g.scan()
	return g
}

// scan recomputes what's visible from the player's position and adds it to
// the discovered set, which is permanent for the rest of the run.
func (g *game) scan() {
	for _, p := range g.fov.SSCVisionMap(g.player, fovRadius, g.net.walkable, true) {
		g.discovered[p] = true
	}
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
	if g.net.walkable(next) {
		g.player = next
		g.scan()
	}
	return nil
}

// Draw implements gruid.Model.Draw.
func (g *game) Draw() gruid.Grid {
	g.grid.Map(func(p gruid.Point, _ gruid.Cell) gruid.Cell {
		switch {
		case p == g.player:
			return gruid.Cell{Rune: '@', Style: gruid.Style{Fg: ColorPlayer}}
		case !g.discovered[p]:
			return gruid.Cell{Rune: ' '}
		case g.net.tiles[p.Y][p.X] == tileNode:
			return gruid.Cell{Rune: '◊', Style: gruid.Style{Fg: ColorNode}}
		case g.net.tiles[p.Y][p.X] == tileEdge:
			return gruid.Cell{Rune: '·', Style: gruid.Style{Fg: ColorEdge}}
		default:
			return gruid.Cell{Rune: ' '}
		}
	})
	return g.grid
}
