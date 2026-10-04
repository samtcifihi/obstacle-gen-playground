package algorithm

import (
	"math"
	"net/url"
	"testing"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

// defaultScoredConfig returns the config Scored runs with when no
// parameters are given.
func defaultScoredConfig(t *testing.T) ScoredConfig {
	t.Helper()
	v, err := scoredAlgorithm.Parse(url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	return scoredConfig(v)
}

func conjugate(flattened bool, a, b, dir, aInv string) Conjugate {
	return Conjugate{
		Flattened: flattened,
		A:         scalarFuncs[a],
		B:         aggregateFuncs[b],
		Dir:       aggregateFuncs[dir],
		AInv:      scalarFuncs[aInv],
	}
}

func TestConjugateApply(t *testing.T) {
	s := [3][2]float64{{0, 4}, {1, 9}, {4, 4}}
	tests := []struct {
		name string
		c    Conjugate
		want float64
	}{
		// sqrt: [0 2 1 3 2 2], mean 10/6, squared.
		{"flattened power mean", conjugate(true, "sqrt", "mean", "min", "square"), 100.0 / 36},
		{"flattened max", conjugate(true, "identity", "max", "min", "identity"), 9},
		{"flattened min", conjugate(true, "identity", "min", "max", "identity"), 0},
		// Per axis: sqrt means [1 2 2], then max 2, squared.
		{"by axis max", conjugate(false, "sqrt", "mean", "max", "square"), 4},
		// Per axis: means [2 5 4], then min.
		{"by axis min", conjugate(false, "identity", "mean", "min", "identity"), 2},
		// Per axis: geometric means [0 3 4], then mean.
		{"by axis geom_mean", conjugate(false, "identity", "geom_mean", "mean", "identity"), 7.0 / 3},
		{"flattened geom_mean", conjugate(true, "identity", "geom_mean", "mean", "identity"), 0},
	}
	for _, tt := range tests {
		if got := tt.c.Apply(s); math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestDistances(t *testing.T) {
	b := board.NewHexagon(5)
	g := grid{b: b, index: b.Index()}
	east, west := board.Axes[0][0], board.Axes[0][1]
	centre := board.Hex{}

	// On an empty board the centre sees past the edge in every direction,
	// and has 4 cells between it and the edge.
	for _, dir := range centre.Neighbours() {
		if got := g.clearance(centre, dir, 5); got != 5 {
			t.Errorf("clearance from centre towards %v = %d, want 5", dir, got)
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
	if got := g.clearance(centre, east, 5); got != 0 {
		t.Errorf("clearance with an adjacent obstacle = %d, want 0", got)
	}
	if got := g.clearance(centre, west, 5); got != 2 {
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

func TestStretch(t *testing.T) {
	cs := []candidate{{score: -0.5}, {score: 0.25}, {score: 1}}
	stretch(cs, true, 2)
	for i, want := range []float64{math.Exp(-3), math.Exp(-1.5), 1} {
		if math.Abs(cs[i].score-want) > 1e-12 {
			t.Errorf("stretched score %d = %v, want %v", i, cs[i].score, want)
		}
	}

	cs = []candidate{{score: -0.5}, {score: 0.25}, {score: 1}}
	stretch(cs, false, 2)
	for i, want := range []float64{1, 1.75, 2.5} {
		if math.Abs(cs[i].score-want) > 1e-12 {
			t.Errorf("shifted score %d = %v, want %v", i, cs[i].score, want)
		}
	}
}

func TestWeightedChoice(t *testing.T) {
	cs := []candidate{{cell: 0, score: 1}, {cell: 1, score: 3}, {cell: 2, score: 0}}
	rng := newRNG(3)
	counts := make([]int, 3)
	const trials = 40_000
	for range trials {
		counts[weightedChoice(cs, rng)]++
	}
	if counts[2] != 0 {
		t.Errorf("cell with weight 0 was chosen %d times", counts[2])
	}
	if got := float64(counts[1]) / trials; math.Abs(got-0.75) > 0.01 {
		t.Errorf("cell with 3/4 of the weight was chosen %.3f of the time", got)
	}
}

func TestBestChoice(t *testing.T) {
	rng := newRNG(3)
	if got := bestChoice([]candidate{{cell: 0, score: 1}, {cell: 1, score: 3}, {cell: 2, score: 2}}, rng); got != 1 {
		t.Errorf("bestChoice picked cell %d, want 1", got)
	}

	ties := []candidate{{cell: 0, score: 1}, {cell: 1, score: 2}, {cell: 2, score: 2}, {cell: 3, score: 2}}
	counts := make([]int, 4)
	for range 30_000 {
		counts[bestChoice(ties, rng)]++
	}
	if counts[0] != 0 {
		t.Errorf("lower-scoring cell was chosen %d times", counts[0])
	}
	for cell := 1; cell <= 3; cell++ {
		if counts[cell] < 9_000 || counts[cell] > 11_000 {
			t.Errorf("tied cell %d was chosen %d of 30000 times, want about 10000", cell, counts[cell])
		}
	}
}

func TestScoredPlacesObstacles(t *testing.T) {
	for _, n := range []int{0, 1, 16, 61, 100} {
		cfg := defaultScoredConfig(t)
		cfg.Obstacles = n
		b := board.NewHexagon(5)
		want := min(n, len(b.Cells))
		if got := Scored(b, cfg, newRNG(1)); got != want {
			t.Errorf("Scored(k_obstacles=%d) returned %d, want %d", n, got, want)
		}
		if got := countObstacles(b); got != want {
			t.Errorf("Scored(k_obstacles=%d) left %d obstacles, want %d", n, got, want)
		}
	}
}

func TestScoredMaxBank(t *testing.T) {
	for _, maxBank := range []int{1, 2, 3} {
		for _, weighted := range []bool{true, false} {
			cfg := defaultScoredConfig(t)
			cfg.Obstacles = 1000
			cfg.MaxBank = maxBank
			cfg.Weighted = weighted
			b := board.NewHexagon(5)
			placed := Scored(b, cfg, newRNG(uint64(maxBank)))
			g := grid{b: b, index: b.Index()}
			bs := g.banks()
			for bank, size := range bs.sizes {
				if size > maxBank {
					t.Errorf("k_max_bank=%d, weighted=%v: bank %d has %d obstacles", maxBank, weighted, bank, size)
				}
			}
			// It stops once no cell can take an obstacle, before filling the board.
			if placed >= len(b.Cells) {
				t.Errorf("k_max_bank=%d, weighted=%v: placed %d, filling the board", maxBank, weighted, placed)
			}
			for _, c := range b.Cells {
				if !c.Obstacle && bs.sizeWith(g, c.Hex) <= maxBank {
					t.Errorf("k_max_bank=%d, weighted=%v: stopped with %v still free", maxBank, weighted, c.Hex)
				}
			}
		}
	}
}

func TestScoredFirstPickWithoutNoise(t *testing.T) {
	// With no noise, on an empty board every cell sees as far as it can
	// in every direction, so the edge term decides: the centre wins.
	cfg := defaultScoredConfig(t)
	cfg.Obstacles = 1
	cfg.BetaCoef = 0
	cfg.Weighted = false
	b := board.NewHexagon(5)
	Scored(b, cfg, newRNG(1))
	if !b.Cells[b.Index()[board.Hex{}]].Obstacle {
		t.Errorf("first obstacle wasn't placed on the centre")
	}
}

func TestScoredIsReproducible(t *testing.T) {
	cfg := defaultScoredConfig(t)
	a, b := board.NewHexagon(5), board.NewHexagon(5)
	Scored(a, cfg, newRNG(42))
	Scored(b, cfg, newRNG(42))
	for i := range a.Cells {
		if a.Cells[i] != b.Cells[i] {
			t.Fatalf("same seed gave different boards at cell %d: %v vs %v", i, a.Cells[i], b.Cells[i])
		}
	}
}
