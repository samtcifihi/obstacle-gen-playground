package algorithm

import (
	"math"
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
	// Beta(1, 1) is flat, so every hex gets the same weight either way.
	for _, inset := range []bool{false, true} {
		for i, w := range hexWeights(board.NewHexagon(5), shapes(1, 1), inset) {
			if w != 1 {
				t.Errorf("inset=%v: cell %d has weight %v, want 1", inset, i, w)
			}
		}
	}
}

func TestHexWeightsPerimeter(t *testing.T) {
	// Beta(2, 2) is 0 at 0 and 1, so perimeter hexes get no weight unless
	// the coordinates are inset.
	b := board.NewHexagon(5)
	for _, inset := range []bool{false, true} {
		weights := hexWeights(b, shapes(2, 2), inset)
		for i, c := range b.Cells {
			if perimeter := ring(c.Hex) == 4; (weights[i] == 0) != (perimeter && !inset) {
				t.Errorf("inset=%v: %v has weight %v", inset, c.Hex, weights[i])
			}
		}
	}
	// The centre is at 0.5 on every axis, where Beta(2, 2) is 1.5.
	if w := hexWeights(b, shapes(2, 2), false)[b.Index()[board.Hex{}]]; math.Abs(w-1.5*1.5*1.5) > 1e-12 {
		t.Errorf("centre has weight %v, want 1.5³", w)
	}
}

func TestHexWeightsSymmetric(t *testing.T) {
	// The same symmetric shape on every axis has sixfold symmetry.
	b := board.NewHexagon(5)
	index := b.Index()
	for _, inset := range []bool{false, true} {
		weights := hexWeights(b, shapes(2.5, 2.5), inset)
		for i, c := range b.Cells {
			rotated := board.Hex{Q: -c.R, R: c.Q + c.R}
			if got := weights[index[rotated]]; math.Abs(got-weights[i]) > 1e-12*weights[i] {
				t.Errorf("inset=%v: %v has weight %v, but rotated 60° to %v has %v", inset, c.Hex, weights[i], rotated, got)
			}
		}
	}
}

func TestHexWeightsAsymmetric(t *testing.T) {
	// Beta(2, 5) on q favours low q: the lower left edge.
	b := board.NewHexagon(5)
	weights := hexWeights(b, [3]BetaShape{{2, 5}, {1, 1}, {1, 1}}, false)
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
	weights := hexWeights(b, [3]BetaShape{{2, 5}, {3, 3}, {0.7, 1.2}}, true)
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

func defaultTripleBetaConfig(t *testing.T, b *board.Board) TripleBetaConfig {
	t.Helper()
	a, _ := Lookup(Algorithms(b), "triple_beta")
	v, err := a.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	return tripleBetaConfig(v)
}

func TestTripleBetaPlacesObstacles(t *testing.T) {
	for _, n := range []int{0, 1, 16, 40} {
		b := board.NewHexagon(5)
		cfg := defaultTripleBetaConfig(t, b)
		cfg.Obstacles = n
		trace := TripleBeta(b, cfg, newRNG(1))
		if len(trace.Steps) != n || countObstacles(b) != n {
			t.Errorf("k_obstacles=%d: placed %d (%d on the board); note %q", n, len(trace.Steps), countObstacles(b), trace.Note)
		}
		checkTrace(t, "triple beta", b, trace)
	}
}

func TestTripleBetaChancesAreExact(t *testing.T) {
	// With flat distributions, every free cell has the same chance, and
	// every cell the same chance per try.
	b := board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t, b)
	cfg.Obstacles = 5
	trace := TripleBeta(b, cfg, newRNG(1))
	for i, step := range trace.Steps {
		for _, c := range step.Candidates {
			if chance, landing := c.Values[0], c.Values[1]; math.Abs(chance-1/float64(61-i)) > 1e-12 || math.Abs(landing-1.0/61) > 1e-12 {
				t.Fatalf("step %d: %v has chance %v and landing %v, want 1/%d and 1/61", i, c.Hex, chance, landing, 61-i)
			}
		}
	}
}

func TestTripleBetaGivesUp(t *testing.T) {
	b := board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t, b)
	cfg.Obstacles = 61
	cfg.MaxTries = 50
	trace := TripleBeta(b, cfg, newRNG(1))
	tries := 0
	for _, s := range trace.Steps {
		if s.Tries < 1 {
			t.Errorf("step placing %v took %d tries", s.Placed, s.Tries)
		}
		tries += s.Tries
	}
	if tries > cfg.MaxTries {
		t.Errorf("steps took %d tries in all, more than k_max_tries=%d", tries, cfg.MaxTries)
	}
	if len(trace.Steps) >= 50 || trace.Note != "gave up after 50 tries" {
		t.Errorf("placed %d obstacles with note %q, want fewer than 50 and to give up", len(trace.Steps), trace.Note)
	}
	checkTrace(t, "triple beta", b, trace)
}

func TestTripleBetaZeroWeightCells(t *testing.T) {
	// With Beta(2, 2) the perimeter has no weight, so only the 37 inner
	// cells can be filled, and they're offered as the only candidates.
	b := board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t, b)
	cfg.Axes = shapes(2, 2)
	cfg.Obstacles = 61
	cfg.MaxTries = 1_000_000
	trace := TripleBeta(b, cfg, newRNG(1))
	if len(trace.Steps) != 37 || trace.Note != "no free cell has any weight" {
		t.Errorf("placed %d obstacles with note %q, want 37 and no weight left", len(trace.Steps), trace.Note)
	}
	for _, step := range trace.Steps {
		for _, c := range step.Candidates {
			if ring(c.Hex) == 4 {
				t.Fatalf("perimeter hex %v offered as a candidate", c.Hex)
			}
		}
	}
	checkTrace(t, "triple beta", b, trace)
}

func TestTripleBetaInfiniteWeights(t *testing.T) {
	// α < 1 makes the density infinite at 0, so it needs is_inset.
	for _, inset := range []bool{false, true} {
		b := board.NewHexagon(5)
		cfg := defaultTripleBetaConfig(t, b)
		cfg.Axes = shapes(0.5, 0.5)
		cfg.Inset = inset
		trace := TripleBeta(b, cfg, newRNG(1))
		if placed := len(trace.Steps); inset != (placed == cfg.Obstacles) || inset == (trace.Note != "") {
			t.Errorf("inset=%v: placed %d with note %q", inset, placed, trace.Note)
		}
	}
}

func TestTripleBetaMaxBank(t *testing.T) {
	b := board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t, b)
	cfg.Obstacles = 61
	cfg.MaxBank = 1
	cfg.MaxTries = 1_000_000
	trace := TripleBeta(b, cfg, newRNG(2))
	g := grid{b: b, index: b.Index()}
	for bank, size := range g.banks().sizes {
		if size > 1 {
			t.Errorf("bank %d has %d obstacles", bank, size)
		}
	}
	if trace.Note != "no free cell has any weight" {
		t.Errorf("stopped with note %q, want no free cell left", trace.Note)
	}
	checkTrace(t, "triple beta", b, trace)
}

func TestTripleBetaIsReproducible(t *testing.T) {
	a, b := board.NewHexagon(5), board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t, a)
	cfg.Axes = [3]BetaShape{{2, 5}, {3, 3}, {1, 2}}
	TripleBeta(a, cfg, newRNG(42))
	TripleBeta(b, cfg, newRNG(42))
	for i := range a.Cells {
		if a.Cells[i] != b.Cells[i] {
			t.Fatalf("same seed gave different boards at cell %d", i)
		}
	}
}
