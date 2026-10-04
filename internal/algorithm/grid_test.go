package algorithm

import (
	"slices"
	"testing"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

func TestClearanceAndEmptyToEdge(t *testing.T) {
	b := board.NewHexagon(5)
	g := grid{b: b, index: b.Index()}
	east, west := board.Axes[0][0], board.Axes[0][1]
	centre := board.Hex{}

	// On an empty board the centre sees past the edge in every direction,
	// and has 4 cells (the board's visibility) between it and the edge.
	for _, dir := range centre.Neighbours() {
		if got := g.clearance(centre, dir, 4); got != 4 {
			t.Errorf("clearance from centre towards %v = %d, want 4", dir, got)
		}
		if got := g.emptyToEdge(centre, dir); got != 4 {
			t.Errorf("emptyToEdge from centre towards %v = %d, want 4", dir, got)
		}
	}

	// An edge cell has nothing between it and the edge outwards.
	if got := g.emptyToEdge(board.Hex{Q: 4}, east); got != 0 {
		t.Errorf("emptyToEdge from (4, 0) east = %d, want 0", got)
	}

	// Obstacles block clearance but are skipped when counting to the edge.
	b.Cells[g.index[board.Hex{Q: 1}]].Obstacle = true
	b.Cells[g.index[board.Hex{Q: -3}]].Obstacle = true
	if got := g.clearance(centre, east, 4); got != 0 {
		t.Errorf("clearance with an adjacent obstacle = %d, want 0", got)
	}
	if got := g.clearance(centre, west, 4); got != 2 {
		t.Errorf("clearance with an obstacle 3 away = %d, want 2", got)
	}
	if got := g.clearance(centre, west, 2); got != 2 {
		t.Errorf("clearance capped at 2 = %d, want 2", got)
	}
	if got := g.emptyToEdge(centre, east); got != 3 {
		t.Errorf("emptyToEdge past one obstacle = %d, want 3", got)
	}
}

func TestBanks(t *testing.T) {
	b := board.NewHexagon(5)
	g := grid{b: b, index: b.Index()}
	for _, h := range []board.Hex{{Q: 0, R: 0}, {Q: 1, R: 0}, {Q: 3, R: 0}, {Q: -2, R: 2}, {Q: -2, R: 3}} {
		b.Cells[g.index[h]].Obstacle = true
	}
	bs := g.banks()
	if len(bs.sizes) != 3 {
		t.Fatalf("found %d banks, want 3", len(bs.sizes))
	}
	tests := []struct {
		h    board.Hex
		want int
	}{
		{board.Hex{Q: 2, R: 0}, 4},  // joins the pair at (0,0)-(1,0) and the single at (3,0)
		{board.Hex{Q: -4, R: 0}, 1}, // touches nothing
		{board.Hex{Q: -1, R: 1}, 5}, // joins the pair at the centre and the pair at (-2,2)-(-2,3)
		{board.Hex{Q: 1, R: -1}, 3}, // touches both cells of the centre pair, counted once
	}
	for _, tt := range tests {
		if got := bs.sizeWith(g, tt.h); got != tt.want {
			t.Errorf("sizeWith(%v) = %d, want %d", tt.h, got, tt.want)
		}
	}
}

func TestSizeWithAll(t *testing.T) {
	b := board.NewHexagon(6)
	g := grid{b: b, index: b.Index()}
	for _, h := range []board.Hex{{Q: 0, R: 0}, {Q: 1, R: 0}, {Q: -3, R: 0}, {Q: 3, R: -3}} {
		b.Cells[g.index[h]].Obstacle = true
	}
	bs := g.banks()
	for _, tt := range []struct {
		hs   []board.Hex
		want int
	}{
		{[]board.Hex{{Q: 4, R: 0}}, 1},                 // touches nothing
		{[]board.Hex{{Q: 2, R: 0}}, 3},                 // joins the centre pair
		{[]board.Hex{{Q: 2, R: 0}, {Q: -2, R: 0}}, 3},  // separate banks: 2 + 1 and 1 + 1
		{[]board.Hex{{Q: -1, R: 0}, {Q: 2, R: 0}}, 4},  // both join the centre pair
		{[]board.Hex{{Q: -2, R: 0}, {Q: -1, R: 0}}, 5}, // adjacent to each other, bridging (-3,0) and the centre pair
		{[]board.Hex{{Q: 3, R: -2}, {Q: -3, R: 1}}, 2}, // each joins a single
	} {
		if got := bs.sizeWithAll(g, tt.hs); got != tt.want {
			t.Errorf("sizeWithAll(%v) = %d, want %d", tt.hs, got, tt.want)
		}
	}
}

func TestFreeCells(t *testing.T) {
	b := board.NewHexagon(3)
	g := grid{b: b, index: b.Index()}
	for _, h := range []board.Hex{{Q: 0, R: 0}, {Q: 1, R: 0}, {Q: -2, R: 2}} {
		b.Cells[g.index[h]].Obstacle = true
	}
	hexes := func(cells []int) []board.Hex {
		var hs []board.Hex
		for _, i := range cells {
			hs = append(hs, b.Cells[i].Hex)
		}
		return hs
	}
	// With no limit, every empty cell is free, in board order.
	free := g.freeCells(0)
	if len(free) != len(b.Cells)-3 || !slices.IsSorted(free) {
		t.Errorf("with no limit, free cells %v, want the %d empty ones in order", hexes(free), len(b.Cells)-3)
	}
	// With a limit of 2, cells that would join the centre pair aren't.
	free = g.freeCells(2)
	for _, h := range hexes(free) {
		if h.DistanceTo(board.Hex{}) == 1 || h.DistanceTo(board.Hex{Q: 1}) == 1 {
			t.Errorf("with a limit of 2, %v is free, but would join the pair at the centre", h)
		}
	}
	if !slices.Contains(hexes(free), board.Hex{Q: -1, R: 2}) {
		t.Errorf("with a limit of 2, free cells %v, want (-1, 2), which makes a pair with (-2, 2)", hexes(free))
	}
}
