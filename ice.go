package main

import "codeberg.org/anaseto/gruid"

// iceKind distinguishes the three ICE behaviors. Sentry and spiker are
// stationary; patrol paces back and forth along a corridor.
type iceKind int

const (
	iceSentry iceKind = iota
	icePatrol
	iceSpiker
)

// ice is one hostile program occupying the network. pos is its current
// tile; for a patrol this changes every turn as it walks route back and
// forth. spotted gates the one-time "you've noticed it" flavor message.
type ice struct {
	kind    iceKind
	pos     gruid.Point
	alive   bool
	spotted bool

	route     []gruid.Point // patrol only: the corridor it paces
	routeStep int
	forward   bool
}

func (ic *ice) glyph() rune {
	switch ic.kind {
	case icePatrol:
		return '◆'
	case iceSpiker:
		return '✷'
	default:
		return '▲'
	}
}

func (ic *ice) color() gruid.Color {
	switch ic.kind {
	case icePatrol:
		return ColorPatrol
	case iceSpiker:
		return ColorTrace
	default:
		return ColorIce
	}
}

func (ic *ice) spottedMessage() string {
	switch ic.kind {
	case icePatrol:
		return "A patrol ICE sweeps the corridor ahead."
	case iceSpiker:
		return "A trace spiker hums nearby. Don't linger."
	default:
		return "A sentry ICE blocks the way ahead. It hasn't noticed you yet."
	}
}

func (ic *ice) blockedMessage() string {
	switch ic.kind {
	case icePatrol:
		return "Patrol ICE blocks the way. Break (b) or cloak past (c)."
	case iceSpiker:
		return "Trace spiker blocks the way. Break (b) or cloak past (c)."
	default:
		return "Sentry ICE blocks the way. Break (b) or cloak past (c)."
	}
}

func (ic *ice) brokenMessage() string {
	switch ic.kind {
	case icePatrol:
		return "Patrol ICE broken. The way is clear."
	case iceSpiker:
		return "Trace spiker broken. The way is clear."
	default:
		return "Sentry ICE broken. The way is clear."
	}
}

func (ic *ice) hitMessage() string {
	switch ic.kind {
	case icePatrol:
		return "The patrol ICE catches you. Integrity dropping."
	case iceSpiker:
		return "The trace spiker pings you. Trace spikes!"
	default:
		return "The sentry ICE hits you. Integrity dropping."
	}
}

func (ic *ice) flatlineMessage() string {
	switch ic.kind {
	case icePatrol:
		return "FLATLINED. The patrol ICE fries your brain. Press q to quit."
	default:
		return "FLATLINED. The sentry ICE fries your brain. Press q to quit."
	}
}

// advance steps a patrol one tile along its route, ping-ponging between the
// two ends. It refuses to step onto the player's tile, waiting a turn
// instead. No-op for stationary kinds.
func (ic *ice) advance(playerPos gruid.Point) {
	if ic.kind != icePatrol || !ic.alive || len(ic.route) < 2 {
		return
	}
	step := ic.routeStep
	if ic.forward {
		step++
		if step >= len(ic.route) {
			step = len(ic.route) - 2
			ic.forward = false
		}
	} else {
		step--
		if step < 0 {
			step = 1
			ic.forward = true
		}
	}
	next := ic.route[step]
	if next == playerPos {
		return
	}
	ic.routeStep = step
	ic.pos = next
}
