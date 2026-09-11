package main

import (
	"fmt"
	"math/rand"

	"codeberg.org/anaseto/gruid"
)

// The network is a graph of nodes connected by corridors: nodes are
// scattered one per cell of a coarse grid so they can't overlap, connected
// by a minimum spanning tree (so the whole thing is reachable) plus a
// handful of extra edges for loops. Edges are carved into the tile grid as
// single-bend corridors. See DESIGN.md's node graph.

const (
	netWidth  = 64
	netHeight = 22

	netCols = 4
	netRows = 3

	netMargin = 3
)

type tileKind uint8

const (
	tileVoid tileKind = iota
	tileNode
	tileEdge
)

// network holds the generated map: which tiles are walkable (node or edge)
// and where the nodes are. entry is where the player jacks in and must
// return to; datastore is the objective node. sentry and spiker are nodes
// held by stationary ICE; patrolRoute is the tile path a moving ICE paces
// back and forth along. name is cosmetic flavor shown in the UI border.
type network struct {
	name        string
	tiles       [][]tileKind // [y][x]
	nodes       []gruid.Point
	entry       gruid.Point
	datastore   gruid.Point
	sentry      gruid.Point
	spiker      gruid.Point
	patrolRoute []gruid.Point
}

// networkCorps and networkSuffixes are combined with a random number to
// generate a flavorful network name, e.g. "OBSIDIAN LOGISTICS-GRID-14".
var networkCorps = []string{
	"OBSIDIAN LOGISTICS", "HELIX BIOTECH", "IRONVEIL DYNAMICS", "SABLECORE",
	"ZENITH ORBITAL", "COBALT MERIDIAN", "PALEFIRE INDUSTRIES", "VANTAPOINT",
	"ECHELON-9", "BLACKWELL HOLDINGS", "NEONBANK", "GRIDLOCK SYSTEMS",
	"OSAKA DYNAMICS", "VESPER CORP", "MERIDIAN ARMS", "DEEPWELL BIOTECH",
}

var networkSuffixes = []string{
	"SUBNET", "NODE", "GRID", "CLUSTER", "MESH", "CORE", "VAULT", "ARRAY",
	"RELAY", "NEXUS", "SECTOR",
}

func randomNetworkName(rng *rand.Rand) string {
	corp := networkCorps[rng.Intn(len(networkCorps))]
	suffix := networkSuffixes[rng.Intn(len(networkSuffixes))]
	num := rng.Intn(99) + 1
	return fmt.Sprintf("%s-%s-%d", corp, suffix, num)
}

func (nw *network) walkable(p gruid.Point) bool {
	if p.X < 0 || p.X >= netWidth || p.Y < 0 || p.Y >= netHeight {
		return false
	}
	return nw.tiles[p.Y][p.X] != tileVoid
}

func generateNetwork(rng *rand.Rand) *network {
	nw := &network{name: randomNetworkName(rng), tiles: make([][]tileKind, netHeight)}
	for y := range nw.tiles {
		nw.tiles[y] = make([]tileKind, netWidth)
	}

	nw.nodes = placeNodes(rng)
	edges := connectNodes(nw.nodes, rng)
	for _, e := range edges {
		nw.carveEdge(nw.nodes[e[0]], nw.nodes[e[1]], rng)
	}
	for _, p := range nw.nodes {
		nw.tiles[p.Y][p.X] = tileNode
	}

	nw.entry = nw.nodes[0]
	nw.datastore = farthestNode(nw.nodes, nw.entry)
	nw.sentry = otherNode(nw.nodes, rng, nw.entry, nw.datastore)
	nw.spiker = otherNode(nw.nodes, rng, nw.entry, nw.datastore, nw.sentry)
	nw.patrolRoute = nw.choosePatrolRoute(edges, rng, nw.entry, nw.datastore, nw.sentry, nw.spiker)

	return nw
}

// choosePatrolRoute picks a random edge whose endpoints are clear of the
// other fixed points, and returns the tile path between them for a patrol
// ICE to pace along. Falls back to any edge if every one is contested,
// which only happens on a very small or crowded network.
func (nw *network) choosePatrolRoute(edges [][2]int, rng *rand.Rand, exclude ...gruid.Point) []gruid.Point {
	excluded := func(p gruid.Point) bool {
		for _, e := range exclude {
			if p == e {
				return true
			}
		}
		return false
	}
	var candidates [][2]int
	for _, e := range edges {
		if !excluded(nw.nodes[e[0]]) && !excluded(nw.nodes[e[1]]) {
			candidates = append(candidates, e)
		}
	}
	if len(candidates) == 0 {
		candidates = edges
	}
	e := candidates[rng.Intn(len(candidates))]
	return nw.tilePath(nw.nodes[e[0]], nw.nodes[e[1]])
}

// tilePath finds the shortest walkable-tile path between a and b via BFS.
// Used for the patrol's beat, where a and b are two nodes joined by a
// carved edge, so a path is guaranteed to exist.
func (nw *network) tilePath(a, b gruid.Point) []gruid.Point {
	prev := map[gruid.Point]gruid.Point{a: a}
	queue := []gruid.Point{a}
	dirs := []gruid.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if p == b {
			var path []gruid.Point
			for p != a {
				path = append([]gruid.Point{p}, path...)
				p = prev[p]
			}
			return append([]gruid.Point{a}, path...)
		}
		for _, d := range dirs {
			n := gruid.Point{X: p.X + d.X, Y: p.Y + d.Y}
			if !nw.walkable(n) {
				continue
			}
			if _, ok := prev[n]; ok {
				continue
			}
			prev[n] = p
			queue = append(queue, n)
		}
	}
	return nil
}

// otherNode picks a random node that isn't any of the excluded points.
func otherNode(nodes []gruid.Point, rng *rand.Rand, exclude ...gruid.Point) gruid.Point {
	excluded := func(p gruid.Point) bool {
		for _, e := range exclude {
			if p == e {
				return true
			}
		}
		return false
	}
	for {
		p := nodes[rng.Intn(len(nodes))]
		if !excluded(p) {
			return p
		}
	}
}

// farthestNode returns the node farthest (by straight-line distance) from a
// given point, used to place the datastore away from the entry.
func farthestNode(nodes []gruid.Point, from gruid.Point) gruid.Point {
	best := nodes[0]
	bestDist := -1
	for _, p := range nodes {
		if p == from {
			continue
		}
		if d := sqDist(p, from); d > bestDist {
			best, bestDist = p, d
		}
	}
	return best
}

// placeNodes scatters one node per cell of a netCols x netRows grid, with a
// little jitter so they don't look mechanically aligned.
func placeNodes(rng *rand.Rand) []gruid.Point {
	cellW := (netWidth - 2*netMargin) / netCols
	cellH := (netHeight - 2*netMargin) / netRows

	var nodes []gruid.Point
	for r := 0; r < netRows; r++ {
		for c := 0; c < netCols; c++ {
			cx := netMargin + c*cellW + cellW/2
			cy := netMargin + r*cellH + cellH/2
			cx += rng.Intn(cellW/2+1) - cellW/4
			cy += rng.Intn(cellH/2+1) - cellH/4
			nodes = append(nodes, gruid.Point{X: cx, Y: cy})
		}
	}
	rng.Shuffle(len(nodes), func(i, j int) {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	})
	return nodes
}

func sqDist(a, b gruid.Point) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}

// connectNodes returns node index pairs to connect: a minimum spanning tree
// (Prim's algorithm), so the network is fully reachable, plus a few extra
// edges between nodes that aren't already connected, for loops.
func connectNodes(nodes []gruid.Point, rng *rand.Rand) [][2]int {
	n := len(nodes)
	inTree := make([]bool, n)
	inTree[0] = true
	var edges [][2]int

	for len(edges) < n-1 {
		from, to, best := -1, -1, 0
		for i := 0; i < n; i++ {
			if !inTree[i] {
				continue
			}
			for j := 0; j < n; j++ {
				if inTree[j] {
					continue
				}
				if d := sqDist(nodes[i], nodes[j]); from == -1 || d < best {
					from, to, best = i, j, d
				}
			}
		}
		inTree[to] = true
		edges = append(edges, [2]int{from, to})
	}

	connected := func(a, b int) bool {
		for _, e := range edges {
			if (e[0] == a && e[1] == b) || (e[0] == b && e[1] == a) {
				return true
			}
		}
		return false
	}
	const extraEdgeChance = 0.1
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if !connected(i, j) && rng.Float64() < extraEdgeChance {
				edges = append(edges, [2]int{i, j})
			}
		}
	}
	return edges
}

// carveEdge marks an L-shaped corridor of tileEdge cells between a and b,
// bending at a randomly chosen corner.
func (nw *network) carveEdge(a, b gruid.Point, rng *rand.Rand) {
	corner := gruid.Point{X: b.X, Y: a.Y}
	if rng.Intn(2) == 0 {
		corner = gruid.Point{X: a.X, Y: b.Y}
	}
	nw.carveLine(a, corner)
	nw.carveLine(corner, b)
}

// carveLine carves a straight (horizontal or vertical) line of tileEdge
// cells between a and b, which must share one coordinate.
func (nw *network) carveLine(a, b gruid.Point) {
	x, y := a.X, a.Y
	for {
		if nw.tiles[y][x] == tileVoid {
			nw.tiles[y][x] = tileEdge
		}
		if x == b.X && y == b.Y {
			break
		}
		if x != b.X {
			x += sign(b.X - x)
		} else {
			y += sign(b.Y - y)
		}
	}
}

func sign(x int) int {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	default:
		return 0
	}
}
