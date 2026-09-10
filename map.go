package main

// Static placeholder map for M0. Later milestones replace this with a
// generated node graph (see DESIGN.md).

const (
	mapWidth  = 60
	mapHeight = 20
)

// buildMap returns a [y][x] grid of walls: true where a wall blocks
// movement, false where it's open floor. The layout is a bordered room
// split by two interior walls, each with a single doorway gap.
func buildMap() [][]bool {
	walls := make([][]bool, mapHeight)
	for y := range walls {
		walls[y] = make([]bool, mapWidth)
	}

	// Outer border.
	for x := 0; x < mapWidth; x++ {
		walls[0][x] = true
		walls[mapHeight-1][x] = true
	}
	for y := 0; y < mapHeight; y++ {
		walls[y][0] = true
		walls[y][mapWidth-1] = true
	}

	// Horizontal divider with a doorway.
	const hWallY = 7
	for x := 10; x < mapWidth-10; x++ {
		if x == 25 || x == 26 {
			continue // doorway
		}
		walls[hWallY][x] = true
	}

	// Vertical divider with a doorway.
	const vWallX = 40
	for y := 8; y < mapHeight-2; y++ {
		if y == 11 {
			continue // doorway
		}
		walls[y][vWallX] = true
	}

	return walls
}
