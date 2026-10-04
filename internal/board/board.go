// Package board models game boards made of hexes.
package board

// Hex is a hex position in axial coordinates. The implicit third cube
// coordinate is s = -q - r.
type Hex struct {
	Q int `json:"q"`
	R int `json:"r"`
}

// Cell is a hex on a board together with what occupies it.
type Cell struct {
	Hex
	Obstacle bool `json:"obstacle"`
}

// Board is a set of cells. Cells are kept in a fixed order so that
// seeded generation is reproducible.
type Board struct {
	EdgeLength int    `json:"edgeLength"`
	Cells      []Cell `json:"cells"`
}

// NewHexagon returns an empty hexagon-shaped board whose sides are each
// edgeLength hexes long, centred on the origin. Cells are ordered by row
// (r), then by column (q).
func NewHexagon(edgeLength int) *Board {
	radius := edgeLength - 1
	b := &Board{EdgeLength: edgeLength}
	for r := -radius; r <= radius; r++ {
		for q := max(-radius, -r-radius); q <= min(radius, -r+radius); q++ {
			b.Cells = append(b.Cells, Cell{Hex: Hex{Q: q, R: r}})
		}
	}
	return b
}
