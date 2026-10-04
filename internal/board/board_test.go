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
