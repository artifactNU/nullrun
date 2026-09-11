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

// integrityMax is starting health; zero means you flatline. Sentry and
// patrol ICE deal iceDamage per hit — enough that one contact is a real
// cost but not an instant kill. A trace spiker instead dumps
// spikeTraceAmount straight into the trace meter, a comparable proportion
// of its budget.
const (
	integrityMax      = 100
	integrityBarWidth = 12
	iceDamage         = 25
	spikeTraceAmount  = 45
)

// RAM is a fixed budget for the whole run — it does not recharge, so
// running a program is spending down a resource you can't get back, not
// topping off a meter. Costs are sized so you can afford one coherent plan
// (say, scan then break one threat) but not everything: with three ICE now
// on the map, breaking every one of them isn't a viable default strategy —
// cloaking past some of them is the budget-friendly play.
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
	cloakDuration  = 6 // turns of ICE immunity after casting
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
	visible    map[gruid.Point]bool // in view this turn; gates live ICE positions

	trace      int
	integrity  int
	ram        int
	cloakTurns int // remaining turns ICE can't see or block you

	ices []ice

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
		ices:       newIces(net),
		grid:       gruid.NewGrid(1, 1), // resized on first Draw to fit the current screen
	}
	g.scan()
	return g
}

// newIces builds the run's ICE from the network's fixed placements: a
// stationary sentry and spiker, plus a patrol if the network generated a
// usable beat for one to pace along.
func newIces(net *network) []ice {
	ices := []ice{
		{kind: iceSentry, pos: net.sentry, alive: true},
		{kind: iceSpiker, pos: net.spiker, alive: true},
	}
	if len(net.patrolRoute) >= 2 {
		ices = append(ices, ice{
			kind:    icePatrol,
			pos:     net.patrolRoute[0],
			alive:   true,
			route:   net.patrolRoute,
			forward: true,
		})
	}
	return ices
}

// scan recomputes what's visible from the player's position. visible is
// this turn's line of sight only, used to gate live ICE positions (so a
// patrol's current spot isn't given away by stale map memory); discovered
// is permanent terrain memory for the rest of the run.
func (g *game) scan() {
	pts := g.fov.SSCVisionMap(g.player, fovRadius, g.net.walkable, true)
	g.visible = make(map[gruid.Point]bool, len(pts))
	for _, p := range pts {
		g.visible[p] = true
		if !g.discovered[p] {
			g.spotIce(p)
		}
		g.discovered[p] = true
	}
}

// spotIce fires an ICE's one-time "you've noticed it" flavor message if it
// currently occupies the newly discovered tile p.
func (g *game) spotIce(p gruid.Point) {
	for i := range g.ices {
		ic := &g.ices[i]
		if ic.alive && !ic.spotted && ic.pos == p {
			ic.spotted = true
			g.message = ic.spottedMessage()
			return
		}
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
	if g.cloakTurns == 0 {
		if ic := g.iceOccupying(next); ic != nil {
			g.message = ic.blockedMessage()
			return nil
		}
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

// updateBreak handles the "b" command: destroy whichever ICE the player is
// standing next to.
func (g *game) updateBreak() gruid.Effect {
	ic := g.adjacentIce()
	if ic == nil {
		g.message = "Nothing to break here."
		return nil
	}
	if !g.spendRAM(breakRAMCost) {
		return nil
	}
	ic.alive = false
	g.message = ic.brokenMessage()
	g.addTrace(traceStep * breakTraceCost)
	if g.traced {
		return nil
	}
	g.afterAction()
	return nil
}

// adjacentIce returns the first alive ICE next to the player, if any.
func (g *game) adjacentIce() *ice {
	for i := range g.ices {
		if g.ices[i].alive && adjacent(g.player, g.ices[i].pos) {
			return &g.ices[i]
		}
	}
	return nil
}

// iceOccupying returns the alive ICE standing exactly at p, if any —
// physical occupancy, independent of whether the player can currently see
// it (you can still bump into something in the fog).
func (g *game) iceOccupying(p gruid.Point) *ice {
	for i := range g.ices {
		if g.ices[i].alive && g.ices[i].pos == p {
			return &g.ices[i]
		}
	}
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
// an active cloak counts down, any patrol takes its step, and any ICE
// still alive, adjacent, and not cloaked against gets its effect in. RAM
// does not recharge — what you spend is gone for the run.
func (g *game) afterAction() {
	if g.cloakTurns > 0 {
		g.cloakTurns--
		if g.cloakTurns == 0 {
			g.message = "Your cloak fades."
		}
	}
	for i := range g.ices {
		g.ices[i].advance(g.player)
	}
	g.resolveIce()
}

// resolveIce applies each still-alive, adjacent, uncloaked ICE's effect:
// sentry and patrol damage integrity (ending the run at zero); a spiker
// dumps trace instead (ending the run if that maxes it out).
func (g *game) resolveIce() {
	if g.cloakTurns > 0 {
		return
	}
	for i := range g.ices {
		ic := &g.ices[i]
		if !ic.alive || !adjacent(g.player, ic.pos) {
			continue
		}
		if ic.kind == iceSpiker {
			g.addTrace(spikeTraceAmount)
			if g.traced {
				return
			}
			g.message = ic.hitMessage()
			continue
		}
		g.integrity -= iceDamage
		if g.integrity <= 0 {
			g.integrity = 0
			g.flatlined = true
			g.message = ic.flatlineMessage()
			return
		}
		g.message = ic.hitMessage()
	}
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
	if p == g.player {
		return gruid.Cell{Rune: '@', Style: gruid.Style{Fg: ColorPlayer}}
	}
	if !g.discovered[p] {
		return gruid.Cell{Rune: ' '}
	}
	if ic := g.iceAt(p); ic != nil {
		return gruid.Cell{Rune: ic.glyph(), Style: gruid.Style{Fg: ic.color()}}
	}
	switch {
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

// iceAt returns the alive ICE to render at p, if any. A patrol only shows
// up while p is within the player's current line of sight (visible), so
// its live position isn't given away by permanent map memory; stationary
// ICE render as soon as their tile is discovered, same as before —
// including via a full-map scanner ping, which reveals topology but not a
// patrol's real-time position.
func (g *game) iceAt(p gruid.Point) *ice {
	for i := range g.ices {
		ic := &g.ices[i]
		if !ic.alive || ic.pos != p {
			continue
		}
		if ic.kind == icePatrol && !g.visible[p] {
			return nil
		}
		return ic
	}
	return nil
}
