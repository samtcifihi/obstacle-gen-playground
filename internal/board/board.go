// Package board models game boards made of hexes.
package board

// Hex is a hex position in axial coordinates. The implicit third cube
// coordinate is s = -q - r.
type Hex struct {
	Q int `json:"q"`
	R int `json:"r"`
}

// Axes lists the six directions from a hex as three axes, each a pair of
// opposite directions.
var Axes = [3][2]Hex{
	{{Q: 1, R: 0}, {Q: -1, R: 0}},
	{{Q: 0, R: 1}, {Q: 0, R: -1}},
	{{Q: 1, R: -1}, {Q: -1, R: 1}},
}

// Add returns h moved by offset o.
func (h Hex) Add(o Hex) Hex {
	return Hex{Q: h.Q + o.Q, R: h.R + o.R}
}

// DistanceTo returns the number of steps from h to o.
func (h Hex) DistanceTo(o Hex) int {
	dq, dr := h.Q-o.Q, h.R-o.R
	return max(abs(dq), abs(dr), abs(dq+dr))
}

// Cube returns h's cube coordinates: q, r and s = -q - r.
func (h Hex) Cube() [3]int {
	return [3]int{h.Q, h.R, -h.Q - h.R}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Neighbours returns the six hexes adjacent to h.
func (h Hex) Neighbours() [6]Hex {
	var ns [6]Hex
	for a, axis := range Axes {
		for s, dir := range axis {
			ns[2*a+s] = h.Add(dir)
		}
	}
	return ns
}

// Cell is a hex on a board together with what occupies it.
type Cell struct {
	Hex
	Obstacle bool `json:"obstacle"`
}

// Board is a set of cells centred on the origin. Cells are kept in a fixed
// order so that seeded generation is reproducible.
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

// Index maps each hex on b to its position in b.Cells.
func (b *Board) Index() map[Hex]int {
	index := make(map[Hex]int, len(b.Cells))
	for i, c := range b.Cells {
		index[c.Hex] = i
	}
	return index
}

// Radius is the furthest any cell is from the centre: the largest of its
// cube coordinates' absolute values, each running from -Radius to Radius.
// On a hexagon that's its edge length - 1.
func (b *Board) Radius() int {
	r := 0
	for _, c := range b.Cells {
		r = max(r, abs(c.Q), abs(c.R), abs(c.Q+c.R))
	}
	return r
}

// Visibility is the furthest distance from the board's centre cell to the
// edge of the board, ignoring obstacles, counted in the cells between them:
// a cell next to the edge is at distance 0. On a hexagon that's its edge
// length - 1, the number of cells from the centre out to the rim.
func (b *Board) Visibility() int {
	index := b.Index()
	visibility := 0
	for _, dir := range (Hex{}).Neighbours() {
		cells := 0
		for h := dir; ; h = h.Add(dir) {
			if _, ok := index[h]; !ok {
				break
			}
			cells++
		}
		visibility = max(visibility, cells)
	}
	return visibility
}

// Span is the longest straight line across the board, counted in the cells
// between its two ends, the same way Visibility counts. No two cells are
// further apart than that on a hexagon, where it runs corner to corner
// through the centre: 2 × edge length - 3 cells.
func (b *Board) Span() int {
	index := b.Index()
	span := 0
	for _, c := range b.Cells {
		for _, dir := range (Hex{}).Neighbours() {
			steps := 0
			for h := c.Add(dir); ; h = h.Add(dir) {
				if _, ok := index[h]; !ok {
					break
				}
				steps++
			}
			span = max(span, steps-1)
		}
	}
	return span
}
