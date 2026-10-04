package board

import "testing"

func TestNewHexagonCellCount(t *testing.T) {
	// A hexagon with edge length L has 3L(L-1)+1 hexes.
	for edge, want := range map[int]int{1: 1, 2: 7, 3: 19, 4: 37, 5: 61, 6: 91} {
		if got := len(NewHexagon(edge).Cells); got != want {
			t.Errorf("NewHexagon(%d) has %d cells, want %d", edge, got, want)
		}
	}
}

func TestNewHexagonCells(t *testing.T) {
	const edge = 5
	b := NewHexagon(edge)
	if b.EdgeLength != edge {
		t.Errorf("EdgeLength = %d, want %d", b.EdgeLength, edge)
	}
	seen := make(map[Hex]bool)
	for _, c := range b.Cells {
		if seen[c.Hex] {
			t.Errorf("duplicate cell %v", c.Hex)
		}
		seen[c.Hex] = true
		if c.Obstacle {
			t.Errorf("cell %v starts with an obstacle", c.Hex)
		}
		if d := max(abs(c.Q), abs(c.R), abs(-c.Q-c.R)); d > edge-1 {
			t.Errorf("cell %v is %d from the centre, beyond radius %d", c.Hex, d, edge-1)
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestNeighbours(t *testing.T) {
	b := NewHexagon(2)
	index := b.Index()
	seen := make(map[Hex]bool)
	for _, n := range (Hex{}).Neighbours() {
		if _, ok := index[n]; !ok || n == (Hex{}) || seen[n] {
			t.Errorf("neighbour %v of the centre isn't a distinct one of the ring around it", n)
		}
		seen[n] = true
	}
	// The centre and its six neighbours are the whole edge-length-2 board.
	if len(b.Cells) != 7 {
		t.Fatalf("board has %d cells, want 7", len(b.Cells))
	}
}

func TestVisibility(t *testing.T) {
	for edge := 1; edge <= 6; edge++ {
		if got := NewHexagon(edge).Visibility(); got != edge {
			t.Errorf("NewHexagon(%d).Visibility() = %d, want %d", edge, got, edge)
		}
	}
}
