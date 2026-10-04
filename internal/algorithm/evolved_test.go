package algorithm

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
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

func TestAggregates(t *testing.T) {
	tests := []struct {
		name string
		xs   []float64
		want float64
	}{
		{"mean", []float64{1, 2, 6}, 3},
		{"geom_mean", []float64{1, 2, 4}, 2},
		{"geom_mean", []float64{0, 2, 4}, 0},
		{"harm_mean", []float64{1, 4, 4}, 2},
		{"harm_mean", []float64{0, 4, 4}, 0},
		{"median", []float64{5, 1, 3}, 3},
		{"median", []float64{5, 1, 4, 2}, 3},
		{"median", []float64{math.Inf(-1), 2}, math.Inf(-1)},
		{"max", []float64{1, 5, 3}, 5},
		{"min", []float64{4, 1, 3}, 1},
	}
	for _, tt := range tests {
		if got := aggregateFuncs[tt.name](tt.xs); got != tt.want && math.Abs(got-tt.want) > 1e-12 {
			t.Errorf("%s(%v) = %v, want %v", tt.name, tt.xs, got, tt.want)
		}
	}
	if got := aggregateFuncs["geom_mean"]([]float64{math.Inf(-1), 0, 1}); !math.IsNaN(got) {
		t.Errorf("geom_mean(-Inf, 0, 1) = %v, want NaN", got)
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

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestDistances(t *testing.T) {
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

func TestWeightedChoice(t *testing.T) {
	cs := []scoredCell{{cell: 0, weight: 1}, {cell: 1, weight: 3}, {cell: 2, weight: 0}}
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

func TestChances(t *testing.T) {
	cells := []scoredCell{{weight: 1}, {weight: 3}, {weight: 0}, {weight: 3}}
	for _, tt := range []struct {
		weighted bool
		want     []float64
	}{
		{weighted: true, want: []float64{1.0 / 7, 3.0 / 7, 0, 3.0 / 7}},
		{weighted: false, want: []float64{0, 0.5, 0, 0.5}},
	} {
		got := chances(cells, tt.weighted)
		for i := range got {
			if math.Abs(got[i]-tt.want[i]) > 1e-12 {
				t.Errorf("weighted=%v: chances = %v, want %v", tt.weighted, got, tt.want)
				break
			}
		}
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

func metricIndex(t *testing.T, trace Trace, name string) int {
	t.Helper()
	for i, m := range trace.Metrics {
		if m.Name == name {
			return i
		}
	}
	t.Fatalf("trace has no %q metric", name)
	return -1
}

// checkTrace checks that trace is consistent with the final board b, which
// started empty.
func checkTrace(t *testing.T, name string, b *board.Board, trace Trace) {
	t.Helper()
	index := b.Index()
	placed := make(map[board.Hex]bool)
	chance := metricIndex(t, trace, "chance")
	for i, step := range trace.Steps {
		if !b.Cells[index[step.Placed]].Obstacle {
			t.Errorf("%s: step %d placed %v, which has no obstacle on the final board", name, i, step.Placed)
		}
		total, chosen := 0.0, false
		for _, c := range step.Candidates {
			if len(c.Values) != len(trace.Metrics) {
				t.Fatalf("%s: step %d candidate %v has %d values for %d metrics", name, i, c.Hex, len(c.Values), len(trace.Metrics))
			}
			if placed[c.Hex] {
				t.Errorf("%s: step %d offers %v, which already has an obstacle", name, i, c.Hex)
			}
			total += c.Values[chance]
			chosen = chosen || c.Hex == step.Placed
		}
		if !chosen {
			t.Errorf("%s: step %d placed %v, which wasn't a candidate", name, i, step.Placed)
		}
		if math.Abs(total-1) > 1e-9 {
			t.Errorf("%s: step %d chances add up to %v, want 1", name, i, total)
		}
		placed[step.Placed] = true
	}
	if len(placed) != countObstacles(b) {
		t.Errorf("%s: trace placed %d distinct cells, but the board has %d obstacles", name, len(placed), countObstacles(b))
	}
}

func TestTraces(t *testing.T) {
	b := board.NewHexagon(5)
	checkTrace(t, "uniform", b, Uniform(b, 20, newRNG(1)))

	for _, weighted := range []bool{true, false} {
		cfg := defaultEvolvedConfig(t)
		cfg.Weighted = weighted
		cfg.MaxBank = 2
		b := board.NewHexagon(5)
		checkTrace(t, fmt.Sprintf("evolved, weighted=%v", weighted), b, Evolved(b, cfg, newRNG(1)))
	}
}
