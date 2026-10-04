package algorithm

import (
	"encoding/json"
	"math"
	"net/url"
	"slices"
	"testing"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

// defaultEvolvedConfig returns the config Evolved runs with when no
// parameters are given.
func defaultEvolvedConfig(t *testing.T) EvolvedConfig {
	t.Helper()
	v, err := evolvedAlgorithm.Parse(url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	return evolvedConfig(v)
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
		// Per axis: sqrt means [1 2 2], squared [1 4 4], then max.
		{"by axis max", conjugate(false, "sqrt", "mean", "max", "square"), 4},
		// Per axis: squared [1 4 4], then their mean. (Taking the mean
		// before squaring would give (5/3)² instead.)
		{"by axis mean", conjugate(false, "sqrt", "mean", "mean", "square"), 3},
		// Per axis: ln means [-Inf, ln 3, ln 4], exponentiated [0 3 4], then
		// their mean.
		{"by axis geometric means", conjugate(false, "ln", "mean", "mean", "exp"), 7.0 / 3},
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

func TestLnConjugateIsGeometricMean(t *testing.T) {
	// ln, mean, e^x is the geometric mean; a 0 distance makes it 0.
	c := conjugate(true, "ln", "mean", "mean", "exp")
	if got := c.Apply([3][2]float64{{1, 2}, {4, 8}, {2, 4}}); math.Abs(got-math.Pow(512, 1.0/6)) > 1e-9 {
		t.Errorf("got %v, want the geometric mean %v", got, math.Pow(512, 1.0/6))
	}
	if got := c.Apply([3][2]float64{{0, 2}, {4, 8}, {2, 4}}); got != 0 {
		t.Errorf("with a 0 distance got %v, want 0", got)
	}
}

func TestEvolvedSkipsUndefinedScores(t *testing.T) {
	// ln then geom_mean is undefined for edge cells, whose outward distance
	// is 0 (ln 0 = -Inf), so they get no score and the trace stays valid.
	cfg := defaultEvolvedConfig(t)
	cfg.EdgeTerm = conjugate(true, "ln", "geom_mean", "mean", "exp")
	b := board.NewHexagon(5)
	trace := Evolved(b, cfg, newRNG(1))
	if len(trace.Steps) == 0 {
		t.Fatal("placed no obstacles")
	}
	if _, err := json.Marshal(trace); err != nil {
		t.Fatalf("trace doesn't encode as JSON: %v", err)
	}
	for _, c := range trace.Steps[0].Candidates {
		if max(abs(c.Q), abs(c.R), abs(-c.Q-c.R)) == 4 {
			t.Errorf("edge cell %v was scored", c.Hex)
		}
		for i, v := range c.Values {
			if !isFinite(v) {
				t.Errorf("cell %v has %s %v", c.Hex, trace.Metrics[i].Name, v)
			}
		}
	}
}

func TestStretch(t *testing.T) {
	cs := []scoredCell{{score: -0.5}, {score: 0.25}, {score: 1}}
	stretch(cs, true, 2)
	for i, want := range []float64{math.Exp(-3), math.Exp(-1.5), 1} {
		if math.Abs(cs[i].weight-want) > 1e-12 {
			t.Errorf("stretched score %d = %v, want %v", i, cs[i].weight, want)
		}
	}

	cs = []scoredCell{{score: -0.5}, {score: 0.25}, {score: 1}}
	stretch(cs, false, 2)
	for i, want := range []float64{1, 1.75, 2.5} {
		if math.Abs(cs[i].weight-want) > 1e-12 {
			t.Errorf("shifted score %d = %v, want %v", i, cs[i].weight, want)
		}
	}
}

func TestBestChoice(t *testing.T) {
	rng := newRNG(3)
	if got := bestChoice([]scoredCell{{cell: 0, weight: 1}, {cell: 1, weight: 3}, {cell: 2, weight: 2}}, rng); got != 1 {
		t.Errorf("bestChoice picked cell %d, want 1", got)
	}

	ties := []scoredCell{{cell: 0, weight: 1}, {cell: 1, weight: 2}, {cell: 2, weight: 2}, {cell: 3, weight: 2}}
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

func TestEvolvedPlacesObstacles(t *testing.T) {
	for _, n := range []int{0, 1, 16, 61, 100} {
		cfg := defaultEvolvedConfig(t)
		cfg.Obstacles = n
		b := board.NewHexagon(5)
		want := min(n, len(b.Cells))
		if got := len(Evolved(b, cfg, newRNG(1)).Steps); got != want {
			t.Errorf("Evolved(k_obstacles=%d) returned %d, want %d", n, got, want)
		}
		if got := countObstacles(b); got != want {
			t.Errorf("Evolved(k_obstacles=%d) left %d obstacles, want %d", n, got, want)
		}
	}
}

func TestEvolvedMaxBank(t *testing.T) {
	for _, maxBank := range []int{1, 2, 3} {
		for _, weighted := range []bool{true, false} {
			cfg := defaultEvolvedConfig(t)
			cfg.Obstacles = 1000
			cfg.MaxBank = maxBank
			cfg.Weighted = weighted
			b := board.NewHexagon(5)
			placed := len(Evolved(b, cfg, newRNG(uint64(maxBank))).Steps)
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

func TestEvolvedFirstPickWithoutNoise(t *testing.T) {
	// With no noise, on an empty board every cell sees as far as it can
	// in every direction, so the edge term decides: the centre wins.
	cfg := defaultEvolvedConfig(t)
	cfg.Obstacles = 1
	cfg.BetaCoef = 0
	cfg.Weighted = false
	b := board.NewHexagon(5)
	Evolved(b, cfg, newRNG(1))
	if !b.Cells[b.Index()[board.Hex{}]].Obstacle {
		t.Errorf("first obstacle wasn't placed on the centre")
	}
}

func TestEvolvedIsReproducible(t *testing.T) {
	cfg := defaultEvolvedConfig(t)
	a, b := board.NewHexagon(5), board.NewHexagon(5)
	Evolved(a, cfg, newRNG(42))
	Evolved(b, cfg, newRNG(42))
	for i := range a.Cells {
		if a.Cells[i] != b.Cells[i] {
			t.Fatalf("same seed gave different boards at cell %d: %v vs %v", i, a.Cells[i], b.Cells[i])
		}
	}
}

func TestBestChances(t *testing.T) {
	cells := []scoredCell{{weight: 1}, {weight: 3}, {weight: 0}, {weight: 3}}
	if got, want := bestChances(cells), []float64{0, 0.5, 0, 0.5}; !slices.Equal(got, want) {
		t.Errorf("chances = %v, want %v", got, want)
	}
}

func TestEvolvedCentreTermsOnEmptyBoard(t *testing.T) {
	// The centre is visibility cells from the edge and sees no obstacles,
	// so both distance terms are 1 for any conjugate that averages.
	cfg := defaultEvolvedConfig(t)
	cfg.Obstacles = 1
	trace := Evolved(board.NewHexagon(5), cfg, newRNG(1))
	for _, c := range trace.Steps[0].Candidates {
		if c.Hex != (board.Hex{}) {
			continue
		}
		obstacles, edge := c.Values[metricIndex(t, trace, "obstacles")], c.Values[metricIndex(t, trace, "edge")]
		if math.Abs(obstacles-1) > 1e-12 || math.Abs(edge-1) > 1e-12 {
			t.Errorf("centre has obstacles term %v and edge term %v, want 1 and 1", obstacles, edge)
		}
		return
	}
	t.Fatal("centre wasn't a candidate on the empty board")
}
