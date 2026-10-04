package algorithm

import (
	"encoding/json"
	"math"
	"net/url"
	"testing"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

func shapes(alpha, beta float64) [3]BetaShape {
	return [3]BetaShape{{alpha, beta}, {alpha, beta}, {alpha, beta}}
}

func ring(h board.Hex) int {
	return max(abs(h.Q), abs(h.R), abs(h.Q+h.R))
}

func TestHexWeightsUniform(t *testing.T) {
	// Beta(1, 1) is flat, so every hex gets the same weight.
	for i, w := range hexWeights(board.NewHexagon(5), shapes(1, 1)) {
		if w != 1 {
			t.Errorf("cell %d has weight %v, want 1", i, w)
		}
	}
}

func TestHexWeightsEdges(t *testing.T) {
	// Coordinates scale to the middles of 9 equal bands, so they're never
	// exactly 0 or 1, where Beta(2, 2) is 0 and Beta(0.5, 0.5) infinite.
	b := board.NewHexagon(5)
	for _, shape := range []BetaShape{{2, 2}, {0.5, 0.5}, {3, 0.2}} {
		for i, w := range hexWeights(b, [3]BetaShape{shape, shape, shape}) {
			if w <= 0 || !isFinite(w) {
				t.Errorf("Beta(%v, %v): %v has weight %v", shape.Alpha, shape.Beta, b.Cells[i].Hex, w)
			}
		}
	}
	// The corner (4, -4, 0) is at 8.5/9, 0.5/9 and 0.5.
	pdf := func(x float64) float64 { return 6 * x * (1 - x) }
	want := pdf(8.5/9) * pdf(0.5/9) * pdf(0.5)
	if w := hexWeights(b, shapes(2, 2))[b.Index()[board.Hex{Q: 4, R: -4}]]; math.Abs(w-want) > 1e-12 {
		t.Errorf("corner has weight %v, want %v", w, want)
	}
	// The centre is at 0.5 on every axis, where Beta(2, 2) is 1.5.
	if w := hexWeights(b, shapes(2, 2))[b.Index()[board.Hex{}]]; math.Abs(w-1.5*1.5*1.5) > 1e-12 {
		t.Errorf("centre has weight %v, want 1.5³", w)
	}
}

func TestHexWeightsSymmetric(t *testing.T) {
	// The same symmetric shape on every axis has sixfold symmetry.
	b := board.NewHexagon(5)
	index := b.Index()
	weights := hexWeights(b, shapes(2.5, 2.5))
	for i, c := range b.Cells {
		rotated := board.Hex{Q: -c.R, R: c.Q + c.R}
		if got := weights[index[rotated]]; math.Abs(got-weights[i]) > 1e-12*weights[i] {
			t.Errorf("%v has weight %v, but rotated 60° to %v has %v", c.Hex, weights[i], rotated, got)
		}
	}
}

func TestHexWeightsAsymmetric(t *testing.T) {
	// Beta(2, 5) on q favours low q: the lower left edge.
	b := board.NewHexagon(5)
	weights := hexWeights(b, [3]BetaShape{{2, 5}, {1, 1}, {1, 1}})
	sum, total := 0.0, 0.0
	for i, c := range b.Cells {
		sum += weights[i] * float64(c.Q)
		total += weights[i]
	}
	if mean := sum / total; mean > -1 {
		t.Errorf("weighted mean q is %v, want well below 0", mean)
	}
}

func TestWeightedIndexMatchesWeights(t *testing.T) {
	const samples = 200_000
	b := board.NewHexagon(5)
	weights := hexWeights(b, [3]BetaShape{{2, 5}, {3, 3}, {0.7, 1.2}})
	total := 0.0
	for _, w := range weights {
		total += w
	}
	counts := make([]float64, len(weights))
	rng := newRNG(9)
	for range samples {
		counts[weightedIndex(rng, weights, total)]++
	}
	for i, w := range weights {
		p := w / total
		if got := counts[i] / samples; math.Abs(got-p) > 4*math.Sqrt(p*(1-p)/samples)+1e-4 {
			t.Errorf("cell %v picked %.4f of the time, want %.4f", b.Cells[i].Hex, got, p)
		}
	}
}

func defaultTripleBetaConfig(t *testing.T) TripleBetaConfig {
	t.Helper()
	a, _ := Lookup("triple_beta")
	v, err := a.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	return tripleBetaConfig(v)
}

func TestTripleBetaPlacesObstacles(t *testing.T) {
	for _, n := range []int{0, 1, 16, 40} {
		b := board.NewHexagon(5)
		cfg := defaultTripleBetaConfig(t)
		cfg.Obstacles = n
		trace := TripleBeta(b, cfg, newRNG(1))
		if len(trace.Steps) != n || countObstacles(b) != n {
			t.Errorf("k_obstacles=%d: placed %d (%d on the board); note %q", n, len(trace.Steps), countObstacles(b), trace.Note)
		}
		checkTrace(t, "triple beta", b, trace)
	}
}

func TestTripleBetaChancesAreExact(t *testing.T) {
	// With flat distributions, every free cell has the same chance.
	b := board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t)
	cfg.Obstacles = 5
	trace := TripleBeta(b, cfg, newRNG(1))
	for i, step := range trace.Steps {
		for _, c := range step.Candidates {
			if chance, weight := c.Values[0], c.Values[1]; math.Abs(chance-1/float64(61-i)) > 1e-12 || weight != 1 {
				t.Fatalf("step %d: %v has chance %v and weight %v, want 1/%d and 1", i, c.Hex, chance, weight, 61-i)
			}
		}
	}
}

func TestTripleBetaFillsBoard(t *testing.T) {
	b := board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t)
	cfg.Obstacles = 100
	trace := TripleBeta(b, cfg, newRNG(1))
	if len(trace.Steps) != 61 || trace.Note != "no free cell left" {
		t.Errorf("placed %d obstacles with note %q, want all 61 and no free cell left", len(trace.Steps), trace.Note)
	}
	checkTrace(t, "triple beta", b, trace)
}

func TestTripleBetaFollowsWeights(t *testing.T) {
	// Over many boards, each cell gets the first obstacle in proportion to
	// its weight.
	const runs = 20_000
	cfg := defaultTripleBetaConfig(t)
	cfg.Axes = [3]BetaShape{{2, 5}, {3, 3}, {1, 2}}
	cfg.Obstacles = 1
	b := board.NewHexagon(5)
	index := b.Index()
	weights := hexWeights(b, cfg.Axes)
	total := 0.0
	for _, w := range weights {
		total += w
	}
	counts := make([]float64, len(weights))
	rng := newRNG(4)
	for range runs {
		b := board.NewHexagon(5)
		counts[index[TripleBeta(b, cfg, rng).Steps[0].Placed]]++
	}
	for i, w := range weights {
		p := w / total
		if got := counts[i] / runs; math.Abs(got-p) > 4*math.Sqrt(p*(1-p)/runs)+1e-4 {
			t.Errorf("cell %v got the first obstacle %.4f of the time, want %.4f", b.Cells[i].Hex, got, p)
		}
	}
}

func TestTripleBetaZeroWeightCells(t *testing.T) {
	// Beta(2000, 1) is so steep that it underflows to 0 below the middle,
	// and every hex has a coordinate at or below the middle, so no hex has
	// any weight.
	b := board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t)
	cfg.Axes = shapes(2000, 1)
	trace := TripleBeta(b, cfg, newRNG(1))
	if len(trace.Steps) != 0 || trace.Note != "no free cell has any weight" {
		t.Errorf("placed %d obstacles with note %q, want none and no weight", len(trace.Steps), trace.Note)
	}
}

func TestTripleBetaUndefinedWeights(t *testing.T) {
	// Shapes this extreme overflow the density's arithmetic.
	for _, cfg := range []func(*TripleBetaConfig){
		func(cfg *TripleBetaConfig) { cfg.Axes = shapes(1e308, 1e308) },
		func(cfg *TripleBetaConfig) { cfg.Distance = BetaShape{1e308, 1e308} },
	} {
		b := board.NewHexagon(5)
		c := defaultTripleBetaConfig(t)
		cfg(&c)
		trace := TripleBeta(b, c, newRNG(1))
		if len(trace.Steps) != 0 || trace.Note == "" {
			t.Errorf("placed %d obstacles with note %q, want none and a note", len(trace.Steps), trace.Note)
		}
		if _, err := json.Marshal(trace); err != nil {
			t.Errorf("trace doesn't encode as JSON: %v", err)
		}
	}
}

func TestTripleBetaMaxBank(t *testing.T) {
	b := board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t)
	cfg.Obstacles = 61
	cfg.MaxBank = 1
	trace := TripleBeta(b, cfg, newRNG(2))
	g := grid{b: b, index: b.Index()}
	for bank, size := range g.banks().sizes {
		if size > 1 {
			t.Errorf("bank %d has %d obstacles", bank, size)
		}
	}
	if trace.Note != "no free cell left" {
		t.Errorf("stopped with note %q, want no free cell left", trace.Note)
	}
	checkTrace(t, "triple beta", b, trace)
}

func TestTripleBetaIsReproducible(t *testing.T) {
	a, b := board.NewHexagon(5), board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t)
	cfg.Axes = [3]BetaShape{{2, 5}, {3, 3}, {1, 2}}
	TripleBeta(a, cfg, newRNG(42))
	TripleBeta(b, cfg, newRNG(42))
	for i := range a.Cells {
		if a.Cells[i] != b.Cells[i] {
			t.Fatalf("same seed gave different boards at cell %d", i)
		}
	}
}

func TestNearestObstacle(t *testing.T) {
	obstacles := []board.Hex{{Q: 0, R: 0}, {Q: 3, R: -1}}
	for _, tt := range []struct {
		h    board.Hex
		want int
	}{
		{board.Hex{Q: 1}, 0},        // adjacent to the centre
		{board.Hex{Q: 2, R: -1}, 0}, // adjacent to (3, -1)
		{board.Hex{Q: -2, R: 0}, 1}, // one cell between it and the centre
		{board.Hex{Q: -4, R: 4}, 3}, // a corner, 4 steps from the centre
		{board.Hex{Q: -4, R: 0}, 3},
		{board.Hex{Q: -9, R: 0}, 4}, // capped at the limit
	} {
		if got := nearestObstacle(tt.h, obstacles, 4); got != tt.want {
			t.Errorf("nearestObstacle(%v) = %d, want %d", tt.h, got, tt.want)
		}
	}
	if got := nearestObstacle(board.Hex{}, nil, 4); got != 4 {
		t.Errorf("with no obstacles, distance = %d, want the limit 4", got)
	}
}

func TestDistanceWeight(t *testing.T) {
	// Distances 0 to 4 sit at 0.1, 0.3, 0.5, 0.7 and 0.9.
	for d := 0; d <= 4; d++ {
		x := (float64(d) + 0.5) / 5
		if got := distanceWeight(d, 4, BetaShape{1, 1}); got != 1 {
			t.Errorf("Beta(1, 1) at distance %d = %v, want 1", d, got)
		}
		if got, want := distanceWeight(d, 4, BetaShape{2, 2}), 6*x*(1-x); math.Abs(got-want) > 1e-12 {
			t.Errorf("Beta(2, 2) at distance %d = %v, want %v", d, got, want)
		}
		if got := distanceWeight(d, 4, BetaShape{0.3, 0.3}); !isFinite(got) {
			t.Errorf("Beta(0.3, 0.3) at distance %d = %v, want finite", d, got)
		}
	}
}

func TestTripleBetaRecordsDistances(t *testing.T) {
	b := board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t)
	cfg.Distance = BetaShape{3, 1.5}
	trace := TripleBeta(b, cfg, newRNG(5))
	distance := metricIndex(t, trace, "distance")
	var placed []board.Hex
	for i, step := range trace.Steps {
		for _, c := range step.Candidates {
			// The trace shows hex distance: one more than the cells between.
			if want := float64(nearestObstacle(c.Hex, placed, 7) + 1); c.Values[distance] != want {
				t.Fatalf("step %d: %v recorded distance %v, want %v", i, c.Hex, c.Values[distance], want)
			}
		}
		placed = append(placed, step.Placed)
	}
	checkTrace(t, "triple beta", b, trace)
}

func TestTripleBetaDistanceShapes(t *testing.T) {
	// Favouring large distances spreads obstacles out, favouring small ones
	// clusters them, compared with the neutral Beta(1, 1).
	adjacentPairs := func(shape BetaShape) float64 {
		pairs := 0
		const runs = 100
		for seed := range uint64(runs) {
			b := board.NewHexagon(5)
			cfg := defaultTripleBetaConfig(t)
			cfg.Distance = shape
			TripleBeta(b, cfg, newRNG(seed))
			g := grid{b: b, index: b.Index()}
			for _, c := range b.Cells {
				for _, n := range c.Neighbours() {
					if _, obstacle := g.at(n); c.Obstacle && obstacle {
						pairs++
					}
				}
			}
		}
		return float64(pairs) / 2 / runs
	}
	spread, neutral, clustered := adjacentPairs(BetaShape{6, 1}), adjacentPairs(BetaShape{1, 1}), adjacentPairs(BetaShape{1, 6})
	if !(spread < neutral && neutral < clustered) {
		t.Errorf("adjacent pairs per board: Beta(6, 1) %.1f, Beta(1, 1) %.1f, Beta(1, 6) %.1f; want increasing", spread, neutral, clustered)
	}
}

func TestParseObstaclesSymmetric(t *testing.T) {
	a, _ := Lookup("triple_beta")
	for _, tt := range []struct {
		symmetric string
		want      float64
	}{{"false", 4}, {"true", 2}} {
		v, err := a.Parse(url.Values{"is_obstacles_symmetric": {tt.symmetric}, "k_alpha_obstacles": {"2"}, "k_beta_obstacles": {"4"}})
		if err != nil {
			t.Fatal(err)
		}
		if got := v.Float("k_beta_obstacles"); got != tt.want {
			t.Errorf("is_obstacles_symmetric=%s: k_beta_obstacles = %v, want %v", tt.symmetric, got, tt.want)
		}
	}
}

func TestTripleBetaSeesAcrossTheBoard(t *testing.T) {
	// With one obstacle in the left corner, distances run all the way to
	// the right corner, 8 steps away, rather than stopping at 5.
	b := board.NewHexagon(5)
	b.Cells[b.Index()[board.Hex{Q: -4}]].Obstacle = true
	cfg := defaultTripleBetaConfig(t)
	cfg.Obstacles = 1
	trace := TripleBeta(b, cfg, newRNG(1))
	distance := metricIndex(t, trace, "distance")
	got := make(map[board.Hex]float64)
	for _, c := range trace.Steps[0].Candidates {
		got[c.Hex] = c.Values[distance]
	}
	for h, want := range map[board.Hex]float64{{Q: -3}: 1, {Q: 0}: 4, {Q: 2}: 6, {Q: 4}: 8, {Q: 4, R: -4}: 8} {
		if got[h] != want {
			t.Errorf("%v has distance %v, want %v", h, got[h], want)
		}
	}
}
