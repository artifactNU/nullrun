package main

import (
	"math/rand"
	"time"

	"codeberg.org/anaseto/gruid"
	"codeberg.org/anaseto/gruid/rl"
)

const fovRadius = 8

// traceMax is when the system finishes tracing you and the run ends. Each
// move fills the trace meter by traceStep. Simulating a near-optimal
// frontier-exploring agent (using the game's real FOV) over entry ->
// datastore -> entry round trips: avg ~88 tiles, worst observed ~171 out of
// 300 runs. Playtested and tuned from there: a clean run clears it, but the
// meter is a real constraint rather than a formality — backtracking or a
// bad layout can genuinely cost you the run.
const (
	traceMax      = 185
	traceStep     = 1
	traceBarWidth = 20
)

// game is the application's Model. It implements gruid.Model.
type game struct {
	net    *network
	player gruid.Point

	fov        *rl.FOV
	discovered map[gruid.Point]bool // ever been in view; fog hides the rest

	trace int

	hasData bool
	won     bool
	traced  bool
	message string

	grid gruid.Grid
}

func newGame() *game {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	net := generateNetwork(rng)
	g := &game{
		net:        net,
		player:     net.entry,
		fov:        rl.NewFOV(gruid.NewRange(0, 0, netWidth, netHeight)),
		discovered: map[gruid.Point]bool{},
		grid:       gruid.NewGrid(netWidth, netHeight+2), // +1 stats bar, +1 message line
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
	if msg.Key == gruid.KeyEscape || msg.Key == "q" {
		return gruid.End()
	}
	if g.won || g.traced {
		return nil
	}

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
	default:
		return nil
	}
	next := g.player.Shift(dx, dy)
	if !g.net.walkable(next) {
		return nil
	}
	g.player = next
	g.scan()
	g.addTrace(traceStep)
	if g.traced {
		return nil
	}
	g.checkObjective()
	return nil
}

// addTrace fills the trace meter, ending the run if it maxes out.
func (g *game) addTrace(amount int) {
	g.trace += amount
	if g.trace >= traceMax {
		g.trace = traceMax
		g.traced = true
		g.message = "TRACE COMPLETE. The system finds you. Press q to quit."
	}
}

// traceBar renders the TRACE meter as a label plus a block of filled and
// empty cells, e.g. "TRACE ███░░░░░░░".
func (g *game) traceBar() string {
	filled := g.trace * traceBarWidth / traceMax
	bar := make([]rune, traceBarWidth)
	for i := range bar {
		if i < filled {
			bar[i] = '█'
		} else {
			bar[i] = '░'
		}
	}
	return "TRACE " + string(bar)
}

// checkObjective handles reaching the datastore and getting back to the
// entry point with the data, which is the run's win condition.
func (g *game) checkObjective() {
	switch {
	case !g.hasData && g.player == g.net.datastore:
		g.hasData = true
		g.message = "Data acquired. Get back to the entry to jack out."
	case g.hasData && g.player == g.net.entry:
		g.won = true
		g.message = "Data extracted — run complete. Press q to quit."
	}
}

// Draw implements gruid.Model.Draw.
func (g *game) Draw() gruid.Grid {
	msg := []rune(g.message)
	stats := []rune(g.traceBar())
	g.grid.Map(func(p gruid.Point, _ gruid.Cell) gruid.Cell {
		if p.Y == netHeight+1 {
			if p.X < len(msg) {
				return gruid.Cell{Rune: msg[p.X]}
			}
			return gruid.Cell{Rune: ' '}
		}
		if p.Y == netHeight {
			if p.X < len(stats) {
				return gruid.Cell{Rune: stats[p.X], Style: gruid.Style{Fg: ColorTrace}}
			}
			return gruid.Cell{Rune: ' '}
		}
		switch {
		case p == g.player:
			return gruid.Cell{Rune: '@', Style: gruid.Style{Fg: ColorPlayer}}
		case !g.discovered[p]:
			return gruid.Cell{Rune: ' '}
		case p == g.net.datastore && !g.hasData:
			return gruid.Cell{Rune: '$', Style: gruid.Style{Fg: ColorData}}
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
