package algorithm

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"slices"
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
		func(cfg *TripleBetaConfig) { cfg.PerAxisDistance, cfg.AxisDistances = true, shapes(1e308, 1e308) },
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

func TestTripleBetaAggregateUnchanged(t *testing.T) {
	// The placements and values Triple Beta gave before per-axis distances
	// were added, which aggregate mode, the default, keeps.
	var want []board.Hex
	for _, qr := range [][2]int{{-3, 5}, {0, 1}, {-2, -3}, {-4, 0}, {-1, 0}, {4, 0}, {-3, -1}, {-3, 2}, {-2, 4}, {0, 0},
		{3, -2}, {0, 3}, {-4, 4}, {-5, 3}, {-3, -2}, {-4, 1}} {
		want = append(want, board.Hex{Q: qr[0], R: qr[1]})
	}
	wantValues := []float64{0.04590465251922107, 0.858062297730058, 2.415676945436856, 3, 0.35520573202096123}
	wantMetrics := []string{"chance", "weight", "position", "distance", "distance_weight"}
	a, _ := Lookup("triple_beta")
	q := url.Values{
		"k_alpha_obstacles": {"3"}, "k_beta_obstacles": {"1.5"},
		"is_symmetric": {"false"}, "is_axes_shared": {"false"},
		"k_alpha_1": {"2"}, "k_beta_1": {"5"}, "k_alpha_2": {"3"}, "k_beta_2": {"3"}, "k_alpha_3": {"0.7"}, "k_beta_3": {"1.2"},
	}
	for _, mode := range []string{"", "aggregate"} {
		if mode != "" {
			q.Set("f_obstacles_distance_mode", mode)
		}
		v, err := a.Parse(q)
		if err != nil {
			t.Fatal(err)
		}
		trace := a.Run(board.NewHexagon(6), v, newRNG(7))
		var metrics []string
		for _, m := range trace.Metrics {
			metrics = append(metrics, m.Name)
		}
		if !slices.Equal(metrics, wantMetrics) {
			t.Errorf("mode %q: metrics %v, want %v", mode, metrics, wantMetrics)
		}
		var placed []board.Hex
		for _, step := range trace.Steps {
			placed = append(placed, step.Placed)
		}
		if !slices.Equal(placed, want) {
			t.Fatalf("mode %q: placed %v, want %v", mode, placed, want)
		}
		for _, c := range trace.Steps[3].Candidates {
			if c.Hex == want[3] && !slices.Equal(c.Values, wantValues) {
				t.Errorf("mode %q: step 4 recorded %v for %v, want %v", mode, c.Values, c.Hex, wantValues)
			}
		}
	}
}

func perAxisConfig(t *testing.T, distances [3]BetaShape) TripleBetaConfig {
	t.Helper()
	cfg := defaultTripleBetaConfig(t)
	cfg.PerAxisDistance = true
	cfg.AxisDistances = distances
	return cfg
}

func TestTripleBetaRecordsAxisDistances(t *testing.T) {
	b := board.NewHexagon(5)
	cfg := perAxisConfig(t, [3]BetaShape{{5, 1}, {1, 5}, {2, 3}})
	cfg.Axes = [3]BetaShape{{2, 5}, {3, 3}, {1, 2}}
	trace := TripleBeta(b, cfg, newRNG(3))
	checkTrace(t, "triple beta per axis", b, trace)
	if len(trace.Steps) != cfg.Obstacles {
		t.Fatalf("placed %d obstacles, want %d; note %q", len(trace.Steps), cfg.Obstacles, trace.Note)
	}
	positions := hexWeights(b, cfg.Axes)
	index := b.Index()
	weight, position, dw := metricIndex(t, trace, "weight"), metricIndex(t, trace, "position"), metricIndex(t, trace, "distance_weight")
	coords := func(h board.Hex) [3]int { return [3]int{h.Q, h.R, -h.Q - h.R} }
	var placed []board.Hex
	for i, step := range trace.Steps {
		for _, c := range step.Candidates {
			product := 1.0
			for axis, name := range []string{"q", "r", "s"} {
				// With no obstacles, every coordinate is as far as can be, 2R,
				// but the term has no effect.
				wantD, wantW := 8, 1.0
				if len(placed) > 0 {
					wantD = 8
					for _, o := range placed {
						wantD = min(wantD, abs(coords(c.Hex)[axis]-coords(o)[axis]))
					}
					wantW = distanceWeight(wantD, 8, cfg.AxisDistances[axis])
				}
				d, w := c.Values[metricIndex(t, trace, name+"_distance")], c.Values[metricIndex(t, trace, name+"_distance_weight")]
				if d != float64(wantD) || w != wantW {
					t.Fatalf("step %d: %v has %s distance %v and weight %v, want %d and %v", i, c.Hex, name, d, w, wantD, wantW)
				}
				product *= w
			}
			pos := positions[index[c.Hex]]
			if c.Values[dw] != product || c.Values[position] != pos || c.Values[weight] != pos*product {
				t.Fatalf("step %d: %v has weight %v = %v × %v, want %v × %v", i, c.Hex, c.Values[weight], c.Values[position], c.Values[dw], pos, product)
			}
		}
		placed = append(placed, step.Placed)
	}
}

func TestTripleBetaAxisDistancesEmptyBoard(t *testing.T) {
	// With no obstacles, the distance term has no effect, so the first
	// obstacle's chances follow the position weights alone. Even shapes
	// whose densities at the far end would round the product down to 0.
	b := board.NewHexagon(5)
	cfg := perAxisConfig(t, shapes(1, 80))
	cfg.Axes = [3]BetaShape{{2, 5}, {3, 3}, {1, 2}}
	trace := TripleBeta(b, cfg, newRNG(1))
	if len(trace.Steps) != cfg.Obstacles {
		t.Fatalf("placed %d obstacles, want %d; note %q", len(trace.Steps), cfg.Obstacles, trace.Note)
	}
	positions := hexWeights(b, cfg.Axes)
	total := 0.0
	for _, w := range positions {
		total += w
	}
	index := b.Index()
	for _, c := range trace.Steps[0].Candidates {
		if want := positions[index[c.Hex]] / total; math.Abs(c.Values[0]-want) > 1e-12 {
			t.Errorf("%v has chance %v, want %v", c.Hex, c.Values[0], want)
		}
	}
}

func TestTripleBetaPerAxisIsNotAggregate(t *testing.T) {
	// From an obstacle at the centre, (2, -2) is two steps along a line,
	// sharing its s band, and (2, -1) two steps between lines. They're the
	// same hex distance away, but different q, r and s distances, so even
	// with every axis tied, per-axis mode weighs them differently.
	b := board.NewHexagon(5)
	b.Cells[b.Index()[board.Hex{}]].Obstacle = true
	cfg := perAxisConfig(t, shapes(1, 3))
	cfg.Obstacles = 1
	trace := TripleBeta(b, cfg, newRNG(1))
	got := make(map[board.Hex][]float64)
	for _, c := range trace.Steps[0].Candidates {
		got[c.Hex] = c.Values
	}
	values := func(h board.Hex) (distances [3]float64, weight float64) {
		for axis, name := range []string{"q", "r", "s"} {
			distances[axis] = got[h][metricIndex(t, trace, name+"_distance")]
		}
		return distances, got[h][metricIndex(t, trace, "distance_weight")]
	}
	line, lineWeight := values(board.Hex{Q: 2, R: -2})
	between, betweenWeight := values(board.Hex{Q: 2, R: -1})
	if line != [3]float64{2, 2, 0} || between != [3]float64{2, 1, 1} {
		t.Errorf("q, r, s distances are %v and %v, want [2 2 0] and [2 1 1]", line, between)
	}
	if lineWeight == betweenWeight {
		t.Errorf("both have distance weight %v, want different", lineWeight)
	}
}

func TestTripleBetaAxisDistanceShapes(t *testing.T) {
	// Beta(5, 1) on q spreads obstacles over q bands, and Beta(1, 10) on r
	// gathers them into a few r bands, compared with neutral shapes.
	bands := func(distances [3]BetaShape) (q, r float64) {
		const runs = 100
		for seed := range uint64(runs) {
			b := board.NewHexagon(6)
			TripleBeta(b, perAxisConfig(t, distances), newRNG(seed))
			qs, rs := make(map[int]bool), make(map[int]bool)
			for _, c := range b.Cells {
				if c.Obstacle {
					qs[c.Q], rs[c.R] = true, true
				}
			}
			q += float64(len(qs)) / runs
			r += float64(len(rs)) / runs
		}
		return q, r
	}
	spreadQ, _ := bands([3]BetaShape{{5, 1}, {1, 1}, {1, 1}})
	_, clusteredR := bands([3]BetaShape{{1, 1}, {1, 10}, {1, 1}})
	neutralQ, neutralR := bands(shapes(1, 1))
	t.Logf("q bands taken: %.1f (Beta(5, 1)), %.1f (neutral); r bands: %.1f (Beta(1, 10)), %.1f (neutral)", spreadQ, neutralQ, clusteredR, neutralR)
	if !(spreadQ > neutralQ+1 && clusteredR < neutralR-1) {
		t.Errorf("q bands taken: %.1f with Beta(5, 1), %.1f neutral; r bands: %.1f with Beta(1, 10), %.1f neutral", spreadQ, neutralQ, clusteredR, neutralR)
	}
}

func TestParseDistanceMode(t *testing.T) {
	a, _ := Lookup("triple_beta")
	for _, tt := range []struct {
		mode    string
		perAxis bool
	}{{"", false}, {"aggregate", false}, {"per_axis", true}} {
		q := url.Values{}
		if tt.mode != "" {
			q.Set("f_obstacles_distance_mode", tt.mode)
		}
		v, err := a.Parse(q)
		if err != nil {
			t.Fatal(err)
		}
		if got := tripleBetaConfig(v).PerAxisDistance; got != tt.perAxis {
			t.Errorf("mode %q: per axis %v, want %v", tt.mode, got, tt.perAxis)
		}
	}
	if _, err := a.Parse(url.Values{"f_obstacles_distance_mode": {"both"}}); err == nil {
		t.Error("mode both parsed, want an error")
	}
}

func TestParseObstacleAxesFollow(t *testing.T) {
	a, _ := Lookup("triple_beta")
	q := url.Values{"f_obstacles_distance_mode": {"per_axis"}}
	for i := range 6 {
		q.Set(fmt.Sprintf("k_%s_obstacles_%d", [2]string{"alpha", "beta"}[i%2], i/2+1), fmt.Sprint(i+1))
	}
	for _, tt := range []struct {
		symmetric, shared string
		want              [3]BetaShape
	}{
		{"false", "false", [3]BetaShape{{1, 2}, {3, 4}, {5, 6}}},
		{"true", "false", [3]BetaShape{{1, 1}, {3, 3}, {5, 5}}},
		{"false", "true", [3]BetaShape{{1, 2}, {1, 2}, {1, 2}}},
		{"true", "true", [3]BetaShape{{1, 1}, {1, 1}, {1, 1}}},
	} {
		q.Set("is_obstacle_axes_symmetric", tt.symmetric)
		q.Set("is_obstacle_axes_shared", tt.shared)
		v, err := a.Parse(q)
		if err != nil {
			t.Fatal(err)
		}
		if got := tripleBetaConfig(v).AxisDistances; got != tt.want {
			t.Errorf("is_obstacle_axes_symmetric=%s, is_obstacle_axes_shared=%s: shapes %v, want %v", tt.symmetric, tt.shared, got, tt.want)
		}
	}
}
