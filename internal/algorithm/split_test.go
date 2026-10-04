package algorithm

import (
	"math"
	"testing"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

func defaultSplitConfig(t *testing.T) SplitConfig {
	t.Helper()
	return SplitConfig{
		Obstacles:       16,
		Symmetric:       true,
		EdgeMargin:      1,
		MaxAdjacent:     1,
		SplitAxes:       aggregateFuncs["max"],
		AdjacentPenalty: 1,
		Noise:           1,
		ScoreBucket:     2,
	}
}

// candidateValues returns the values a step recorded for each cell.
func candidateValues(t *testing.T, trace Trace, step int) map[board.Hex]map[string]float64 {
	t.Helper()
	values := make(map[board.Hex]map[string]float64)
	for _, c := range trace.Steps[step].Candidates {
		values[c.Hex] = make(map[string]float64)
		for i, m := range trace.Metrics {
			values[c.Hex][m.Name] = c.Values[i]
		}
	}
	return values
}

func TestSplitScores(t *testing.T) {
	// No noise and no bucketing, on an empty edge-length-5 board, with the
	// perimeter allowed.
	cfg := defaultSplitConfig(t)
	cfg.Obstacles, cfg.Symmetric, cfg.EdgeMargin, cfg.Noise, cfg.ScoreBucket = 1, false, 0, 0, 1
	b := board.NewHexagon(5)
	trace := Split(b, cfg, newRNG(1))
	values := candidateValues(t, trace, 0)
	for _, tt := range []struct {
		h                  board.Hex
		split, edge, score float64
	}{
		{board.Hex{}, 4, 4, 4},            // 4 clear cells every way
		{board.Hex{Q: 1}, 3, 3, 3},        // 3 east, 5 west: the shorter is 3, and likewise on the other axes
		{board.Hex{Q: 4}, 0, 0, 0},        // on the edge
		{board.Hex{Q: 2, R: -1}, 3, 2, 3}, // 2 to the upper right edge on two axes, but 3 either way along r
	} {
		v := values[tt.h]
		if v["split"] != tt.split || v["edge_distance"] != tt.edge || v["score"] != tt.score || v["noise"] != 0 || v["adjacent"] != 0 {
			t.Errorf("%v: got %v, want split %v, edge distance %v, score %v", tt.h, v, tt.split, tt.edge, tt.score)
		}
	}
	// The centre is the unique best, so it's chosen.
	if trace.Steps[0].Placed != (board.Hex{}) || values[board.Hex{}]["chance"] != 1 {
		t.Errorf("placed %v with the centre's chance %v, want the centre for certain", trace.Steps[0].Placed, values[board.Hex{}]["chance"])
	}
}

func TestSplitScoreFormula(t *testing.T) {
	// raw = split − penalty × adjacent + noise; score = raw / bucket, rounded
	// towards 0.
	cfg := defaultSplitConfig(t)
	cfg.Obstacles, cfg.Symmetric, cfg.AdjacentPenalty, cfg.Noise, cfg.ScoreBucket = 1, false, 1.5, 3, 3
	cfg.SplitAxes = aggregateFuncs["mean"]
	b := board.NewHexagon(6)
	b.Cells[b.Index()[board.Hex{Q: 1}]].Obstacle = true
	trace := Split(b, cfg, newRNG(2))
	for h, v := range candidateValues(t, trace, 0) {
		raw := v["split"] - 1.5*v["adjacent"] + v["noise"]
		if math.Abs(v["raw_score"]-raw) > 1e-12 || v["score"] != math.Trunc(raw/3) || math.Signbit(v["score"]) && v["score"] == 0 {
			t.Errorf("%v: %v doesn't follow the formula", h, v)
		}
		if v["noise"] < 0 || v["noise"] > 3 || v["noise"] != math.Trunc(v["noise"]) {
			t.Errorf("%v: noise %v isn't a whole number from 0 to 3", h, v["noise"])
		}
	}
}

func isSymmetric(b *board.Board) bool {
	index := b.Index()
	for _, c := range b.Cells {
		if c.Obstacle != b.Cells[index[board.Hex{Q: -c.Q, R: -c.R}]].Obstacle {
			return false
		}
	}
	return true
}

func TestSplitSymmetric(t *testing.T) {
	for _, n := range []int{1, 2, 15, 16, 17, 21} {
		for seed := range uint64(5) {
			b := board.NewHexagon(6)
			cfg := defaultSplitConfig(t)
			cfg.Obstacles = n
			trace := Split(b, cfg, newRNG(seed))
			placed := trace.Placed()
			if placed > n || placed != countObstacles(b) || !isSymmetric(b) || (placed < n) == (trace.Note == "") {
				t.Errorf("k_obstacles=%d, seed %d: placed %d (%d on the board), symmetric %v; note %q",
					n, seed, placed, countObstacles(b), isSymmetric(b), trace.Note)
			}
			// Each step places a pair, or the centre on its own.
			for i, step := range trace.Steps {
				centre := step.Placed == (board.Hex{})
				if centre != (len(step.Also) == 0) || (!centre && step.Also[0] != (board.Hex{Q: -step.Placed.Q, R: -step.Placed.R})) {
					t.Errorf("k_obstacles=%d, seed %d: step %d placed %v and %v", n, seed, i, step.Placed, step.Also)
				}
			}
		}
	}
}

func TestSplitEvenCountCanUseCentre(t *testing.T) {
	// With no noise or bucketing, the centre has the best split on an empty
	// board, so it goes first, on its own. That leaves an odd number for
	// the pairs, so an even count ends one short.
	b := board.NewHexagon(6)
	cfg := defaultSplitConfig(t)
	cfg.Noise, cfg.ScoreBucket = 0, 1
	trace := Split(b, cfg, newRNG(1))
	if first := trace.Steps[0]; first.Placed != (board.Hex{}) || len(first.Also) != 0 {
		t.Errorf("first step placed %v and %v, want the centre alone", first.Placed, first.Also)
	}
	if trace.Placed() != 15 || trace.Note != "only the centre could take the last obstacle, and it isn't eligible" {
		t.Errorf("placed %d with note %q, want 15 and the centre note", trace.Placed(), trace.Note)
	}
}

func TestSplitCentreUnavailable(t *testing.T) {
	// With the centre taken, an odd count can't be finished symmetrically.
	b := board.NewHexagon(6)
	b.Cells[b.Index()[board.Hex{}]].Obstacle = true
	cfg := defaultSplitConfig(t)
	cfg.Obstacles = 3
	trace := Split(b, cfg, newRNG(1))
	if trace.Placed() != 2 || trace.Note != "only the centre could take the last obstacle, and it isn't eligible" {
		t.Errorf("placed %d with note %q, want 2 and the centre note", trace.Placed(), trace.Note)
	}
}

func TestSplitEdgeMargin(t *testing.T) {
	for _, margin := range []int{0, 1, 2} {
		b := board.NewHexagon(6)
		cfg := defaultSplitConfig(t)
		cfg.Obstacles, cfg.EdgeMargin, cfg.MaxAdjacent, cfg.Symmetric = 1000, margin, 6, false
		trace := Split(b, cfg, newRNG(1))
		for _, c := range b.Cells {
			if c.Obstacle && c.Hex.DistanceTo(board.Hex{}) > 5-margin {
				t.Errorf("k_edge_margin=%d: obstacle at %v, %d cells from the edge", margin, c.Hex, 5-c.Hex.DistanceTo(board.Hex{}))
			}
		}
		// It fills everything it's allowed to.
		allowed := 3*(6-margin)*(5-margin) + 1
		if countObstacles(b) != allowed {
			t.Errorf("k_edge_margin=%d: placed %d, want all %d allowed cells; note %q", margin, countObstacles(b), allowed, trace.Note)
		}
	}
}

func TestSplitMaxAdjacent(t *testing.T) {
	for _, maxAdjacent := range []int{0, 1, 2} {
		for _, symmetric := range []bool{false, true} {
			b := board.NewHexagon(6)
			cfg := defaultSplitConfig(t)
			cfg.Obstacles, cfg.MaxAdjacent, cfg.Symmetric, cfg.EdgeMargin = 1000, maxAdjacent, symmetric, 0
			trace := Split(b, cfg, newRNG(uint64(maxAdjacent)))
			// Replay the placements: each new obstacle touches at most
			// maxAdjacent obstacles once its step is done.
			placed := make(map[board.Hex]bool)
			for i, step := range trace.Steps {
				hexes := append([]board.Hex{step.Placed}, step.Also...)
				for _, h := range hexes {
					placed[h] = true
				}
				for _, h := range hexes {
					touching := 0
					for _, n := range h.Neighbours() {
						if placed[n] {
							touching++
						}
					}
					if touching > maxAdjacent {
						t.Errorf("k_max_adjacent=%d, symmetric %v: step %d put %v touching %d obstacles", maxAdjacent, symmetric, i, h, touching)
					}
				}
			}
			if trace.Note == "" {
				t.Errorf("k_max_adjacent=%d, symmetric %v: filled the board", maxAdjacent, symmetric)
			}
		}
	}
}

func TestSplitMaxBank(t *testing.T) {
	for _, maxBank := range []int{1, 2, 3} {
		for _, symmetric := range []bool{false, true} {
			b := board.NewHexagon(6)
			cfg := defaultSplitConfig(t)
			cfg.Obstacles, cfg.MaxBank, cfg.Symmetric, cfg.MaxAdjacent, cfg.EdgeMargin = 1000, maxBank, symmetric, 6, 0
			Split(b, cfg, newRNG(uint64(maxBank)))
			g := grid{b: b, index: b.Index()}
			for bank, size := range g.banks().sizes {
				if size > maxBank {
					t.Errorf("k_max_bank=%d, symmetric %v: bank %d has %d obstacles", maxBank, symmetric, bank, size)
				}
			}
		}
	}
}

func TestSplitChances(t *testing.T) {
	for _, symmetric := range []bool{false, true} {
		b := board.NewHexagon(6)
		cfg := defaultSplitConfig(t)
		cfg.Symmetric = symmetric
		trace := Split(b, cfg, newRNG(7))
		chance := metricIndex(t, trace, "chance")
		score := metricIndex(t, trace, "score")
		for i, step := range trace.Steps {
			// Count each placement once: a pair's cells share its chance.
			total, best := 0.0, math.Inf(-1)
			seen := make(map[board.Hex]bool)
			var chosen float64
			for _, c := range step.Candidates {
				best = max(best, c.Values[score])
				if c.Hex == step.Placed {
					chosen = c.Values[chance]
				}
				if !seen[board.Hex{Q: -c.Q, R: -c.R}] || !symmetric {
					total += c.Values[chance]
				}
				seen[c.Hex] = true
			}
			if math.Abs(total-1) > 1e-9 {
				t.Errorf("symmetric %v, step %d: chances add up to %v", symmetric, i, total)
			}
			if chosen == 0 {
				t.Errorf("symmetric %v, step %d: placed %v, which had no chance", symmetric, i, step.Placed)
			}
			for _, c := range step.Candidates {
				if (c.Values[chance] > 0) != (c.Values[score] == best) {
					t.Errorf("symmetric %v, step %d: %v has score %v and chance %v, with the best score %v",
						symmetric, i, c.Hex, c.Values[score], c.Values[chance], best)
				}
			}
		}
	}
	// One huge bucket makes every eligible placement tie.
	b := board.NewHexagon(6)
	cfg := defaultSplitConfig(t)
	cfg.Obstacles, cfg.Symmetric, cfg.ScoreBucket = 1, false, 1000
	trace := Split(b, cfg, newRNG(1))
	for _, c := range trace.Steps[0].Candidates {
		if want := 1 / float64(len(trace.Steps[0].Candidates)); c.Values[metricIndex(t, trace, "chance")] != want {
			t.Fatalf("%v has chance %v, want %v", c.Hex, c.Values[1], want)
		}
	}
	checkTrace(t, "split", b, trace)
}

func TestSplitIsReproducible(t *testing.T) {
	a, b := board.NewHexagon(6), board.NewHexagon(6)
	cfg := defaultSplitConfig(t)
	Split(a, cfg, newRNG(42))
	Split(b, cfg, newRNG(42))
	for i := range a.Cells {
		if a.Cells[i] != b.Cells[i] {
			t.Fatalf("same seed gave different boards at cell %d", i)
		}
	}
}

func TestSplitRoundsTowardsZero(t *testing.T) {
	// (2, 0) touches an obstacle at (1, 0), so its shortest run, and with
	// min its split, is 0, and its raw score is minus the penalty.
	for _, tt := range []struct{ penalty, score float64 }{
		{1, 0},  // -1/2 rounds to 0, not -1, and isn't -0
		{3, -1}, // -3/2 rounds to -1, not -2
	} {
		b := board.NewHexagon(6)
		b.Cells[b.Index()[board.Hex{Q: 1}]].Obstacle = true
		cfg := defaultSplitConfig(t)
		cfg.Obstacles, cfg.Symmetric, cfg.EdgeMargin, cfg.Noise = 1, false, 0, 0
		cfg.SplitAxes, cfg.AdjacentPenalty = aggregateFuncs["min"], tt.penalty
		v := candidateValues(t, Split(b, cfg, newRNG(1)), 0)[board.Hex{Q: 2}]
		if v["raw_score"] != -tt.penalty || v["score"] != tt.score || math.Signbit(v["score"]) && tt.score == 0 {
			t.Errorf("penalty %v: raw %v, score %v, want raw %v and score %v", tt.penalty, v["raw_score"], v["score"], -tt.penalty, tt.score)
		}
	}
}
