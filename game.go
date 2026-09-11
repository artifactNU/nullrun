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
	traceBarWidth = 12
)

// integrityMax is starting health; zero means you flatline. The sentry ICE
// deals sentryDamage per hit — enough that one contact is a real cost but
// not an instant kill.
const (
	integrityMax      = 100
	integrityBarWidth = 12
	sentryDamage      = 25
)

// RAM is a fixed budget for the whole run — it does not recharge, so
// running a program is spending down a resource you can't get back, not
// topping off a meter. Costs are sized so you can afford one coherent plan
// (say, scan then break) but not everything: break+cloak alone already
// blows the budget, since a run only ever has the one sentry to deal with.
// Each program also taxes the trace meter via its own traceStep multiplier.
const (
	ramMax      = 8
	ramBarWidth = 10

	breakRAMCost   = 4
	breakTraceCost = 3

	scanRAMCost   = 3
	scanTraceCost = 2

	cloakRAMCost   = 5
	cloakTraceCost = 2
	cloakDuration  = 6 // turns of sentry immunity after casting
)

// The game view is framed in a box: a title bar naming the network, the
// map, a separator, then the stats and message lines inside the border.
const (
	borderWidth  = netWidth + 2
	borderHeight = netHeight + 5

	rowSeparator = netHeight + 1
	rowStats     = netHeight + 2
	rowMessage   = netHeight + 3
	rowBottom    = netHeight + 4
)

// screen selects which of the game's two views Update and Draw operate on.
type screen int

const (
	screenSplash screen = iota
	screenPlaying
)

// game is the application's Model. It implements gruid.Model.
type game struct {
	screen screen

	net    *network
	player gruid.Point

	fov        *rl.FOV
	discovered map[gruid.Point]bool // ever been in view; fog hides the rest

	trace      int
	integrity  int
	ram        int
	cloakTurns int // remaining turns the sentry can't see or block you

	iceAlive   bool
	iceSpotted bool // for the one-time "hasn't seen you yet" flavor message

	hasData   bool
	won       bool
	traced    bool
	flatlined bool
	message   string

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
		integrity:  integrityMax,
		ram:        ramMax,
		iceAlive:   true,
		grid:       gruid.NewGrid(1, 1), // resized on first Draw to fit the current screen
	}
	g.scan()
	return g
}

// scan recomputes what's visible from the player's position and adds it to
// the discovered set, which is permanent for the rest of the run.
func (g *game) scan() {
	for _, p := range g.fov.SSCVisionMap(g.player, fovRadius, g.net.walkable, true) {
		if !g.discovered[p] && p == g.net.sentry && g.iceAlive && !g.iceSpotted {
			g.iceSpotted = true
			g.message = "A sentry ICE blocks the way ahead. It hasn't noticed you yet."
		}
		g.discovered[p] = true
	}
}

// adjacent reports whether a and b are one orthogonal step apart.
func adjacent(a, b gruid.Point) bool {
	dx, dy := a.X-b.X, a.Y-b.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx+dy == 1
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
	if g.screen == screenSplash {
		g.screen = screenPlaying
		return nil
	}
	if msg.Key == gruid.KeyEscape || msg.Key == "q" {
		return gruid.End()
	}
	if g.won || g.traced || g.flatlined {
		return nil
	}

	switch msg.Key {
	case "b":
		return g.updateBreak()
	case "s":
		return g.updateScan()
	case "c":
		return g.updateCloak()
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
	if next == g.net.sentry && g.iceAlive && g.cloakTurns == 0 {
		g.message = "Sentry ICE blocks the way. Break (b) or cloak past (c)."
		return nil
	}
	g.player = next
	g.scan()
	g.addTrace(traceStep)
	if g.traced {
		return nil
	}
	g.afterAction()
	if g.flatlined {
		return nil
	}
	g.checkObjective()
	return nil
}

// updateBreak handles the "b" command: destroy the sentry ICE if the
// player is standing next to it.
func (g *game) updateBreak() gruid.Effect {
	if !g.iceAlive || !adjacent(g.player, g.net.sentry) {
		g.message = "Nothing to break here."
		return nil
	}
	if !g.spendRAM(breakRAMCost) {
		return nil
	}
	g.iceAlive = false
	g.message = "Sentry ICE broken. The way is clear."
	g.addTrace(traceStep * breakTraceCost)
	if g.traced {
		return nil
	}
	g.afterAction()
	return nil
}

// updateScan handles the "s" command: a full network ping that reveals
// every corridor and node on the map, ignoring fog and line of sight.
func (g *game) updateScan() gruid.Effect {
	if !g.spendRAM(scanRAMCost) {
		return nil
	}
	for y := range g.net.tiles {
		for x := range g.net.tiles[y] {
			if g.net.tiles[y][x] != tileVoid {
				g.discovered[gruid.Point{X: x, Y: y}] = true
			}
		}
	}
	g.message = "Scanner ping complete. Network topology revealed."
	g.addTrace(traceStep * scanTraceCost)
	if g.traced {
		return nil
	}
	g.afterAction()
	return nil
}

// updateCloak handles the "c" command: go dark for cloakDuration turns,
// during which the sentry ICE neither blocks you nor hits you.
func (g *game) updateCloak() gruid.Effect {
	if !g.spendRAM(cloakRAMCost) {
		return nil
	}
	g.cloakTurns = cloakDuration
	g.message = "Cloak engaged. ICE won't see you for a while."
	g.addTrace(traceStep * cloakTraceCost)
	if g.traced {
		return nil
	}
	g.afterAction()
	return nil
}

// spendRAM attempts to deduct cost from the RAM pool, returning false (and
// setting a message) if there isn't enough to cover it.
func (g *game) spendRAM(cost int) bool {
	if g.ram < cost {
		g.message = "Not enough RAM."
		return false
	}
	g.ram -= cost
	return true
}

// afterAction runs the bookkeeping shared by every turn-consuming action:
// an active cloak counts down, and the sentry ICE (if still alive,
// adjacent, and not cloaked against) gets its counter-attack in. RAM does
// not recharge — what you spend is gone for the run.
func (g *game) afterAction() {
	if g.cloakTurns > 0 {
		g.cloakTurns--
		if g.cloakTurns == 0 {
			g.message = "Your cloak fades."
		}
	}
	g.resolveSentry()
}

// resolveSentry deals damage if the player is next to a still-alive,
// uncloaked sentry ICE, ending the run if it drops integrity to zero.
func (g *game) resolveSentry() {
	if !g.iceAlive || g.cloakTurns > 0 || !adjacent(g.player, g.net.sentry) {
		return
	}
	g.integrity -= sentryDamage
	if g.integrity <= 0 {
		g.integrity = 0
		g.flatlined = true
		g.message = "FLATLINED. The sentry ICE fries your brain. Press q to quit."
		return
	}
	g.message = "The sentry ICE hits you. Integrity dropping."
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
	return "TRACE " + meterBar(g.trace, traceMax, traceBarWidth, '█')
}

// integrityBar renders the INTEGRITY meter the same way.
func (g *game) integrityBar() string {
	return "INTEGRITY " + meterBar(g.integrity, integrityMax, integrityBarWidth, '█')
}

// ramBar renders the RAM meter with its own fill glyph to set it apart from
// the other two meters at a glance.
func (g *game) ramBar() string {
	return "RAM " + meterBar(g.ram, ramMax, ramBarWidth, '▓')
}

func meterBar(value, max, width int, fill rune) string {
	filled := value * width / max
	bar := make([]rune, width)
	for i := range bar {
		if i < filled {
			bar[i] = fill
		} else {
			bar[i] = '░'
		}
	}
	return string(bar)
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
	if g.screen == screenSplash {
		return g.drawSplash()
	}
	return g.drawGame()
}

// drawGame renders the network view inside a box-drawn frame: a title bar
// naming the network, the map, a separator, then stats and message lines.
func (g *game) drawGame() gruid.Grid {
	g.grid = g.grid.Resize(borderWidth, borderHeight)

	integrityStr := g.integrityBar()
	traceStr := g.traceBar()
	ramStr := g.ramBar()
	const sep = "  "
	statsLine := " " + integrityStr + sep + traceStr + sep + ramStr
	stats := []rune(padTrunc(statsLine, netWidth))
	integrityEnd := 1 + len([]rune(integrityStr))
	traceEnd := integrityEnd + len(sep) + len([]rune(traceStr))

	msg := []rune(padTrunc(" > "+g.message, netWidth))

	top := []rune(topBorder(g.net.name))
	middle := []rune(separatorBorder())
	bottom := []rune(bottomBorder())

	g.grid.Map(func(p gruid.Point, _ gruid.Cell) gruid.Cell {
		switch p.Y {
		case 0:
			return frameCell(top, p.X)
		case rowBottom:
			return frameCell(bottom, p.X)
		case rowSeparator:
			return frameCell(middle, p.X)
		case rowStats, rowMessage:
			if p.X == 0 || p.X == borderWidth-1 {
				return gruid.Cell{Rune: '│', Style: gruid.Style{Fg: ColorNode}}
			}
			ix := p.X - 1
			if p.Y == rowStats {
				col := ColorRAM
				switch {
				case ix < integrityEnd:
					col = ColorIntegrity
				case ix < traceEnd:
					col = ColorTrace
				}
				return gruid.Cell{Rune: stats[ix], Style: gruid.Style{Fg: col}}
			}
			return gruid.Cell{Rune: msg[ix]}
		default:
			if p.X == 0 || p.X == borderWidth-1 {
				return gruid.Cell{Rune: '│', Style: gruid.Style{Fg: ColorNode}}
			}
			return g.mapCell(gruid.Point{X: p.X - 1, Y: p.Y - 1})
		}
	})
	return g.grid
}

// mapCell returns the cell to draw for a position on the network map
// itself (not the border or stats/message lines).
func (g *game) mapCell(p gruid.Point) gruid.Cell {
	switch {
	case p == g.player:
		return gruid.Cell{Rune: '@', Style: gruid.Style{Fg: ColorPlayer}}
	case !g.discovered[p]:
		return gruid.Cell{Rune: ' '}
	case p == g.net.sentry && g.iceAlive:
		return gruid.Cell{Rune: '▲', Style: gruid.Style{Fg: ColorIce}}
	case p == g.net.datastore && !g.hasData:
		return gruid.Cell{Rune: '$', Style: gruid.Style{Fg: ColorData}}
	case p == g.net.entry:
		return gruid.Cell{Rune: '⌂', Style: gruid.Style{Fg: ColorEntry}}
	case g.net.tiles[p.Y][p.X] == tileNode:
		return gruid.Cell{Rune: '◊', Style: gruid.Style{Fg: ColorNode}}
	case g.net.tiles[p.Y][p.X] == tileEdge:
		return gruid.Cell{Rune: '·', Style: gruid.Style{Fg: ColorEdge}}
	default:
		return gruid.Cell{Rune: ' '}
	}
}
