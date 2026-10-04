package algorithm

import (
	"math"
	"slices"
	"testing"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

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

func TestWeightedIndexSkipsZeroWeights(t *testing.T) {
	weights := []float64{1, 3, 0}
	rng := newRNG(3)
	counts := make([]int, 3)
	const trials = 40_000
	for range trials {
		counts[weightedIndex(rng, weights, 4)]++
	}
	if counts[2] != 0 {
		t.Errorf("weight 0 was chosen %d times", counts[2])
	}
	if got := float64(counts[1]) / trials; math.Abs(got-0.75) > 0.01 {
		t.Errorf("3/4 of the weight was chosen %.3f of the time", got)
	}
}

func TestWeightedChances(t *testing.T) {
	if got, want := weightedChances([]float64{1, 3, 0, 3}, 7), []float64{1.0 / 7, 3.0 / 7, 0, 3.0 / 7}; !slices.Equal(got, want) {
		t.Errorf("chances = %v, want %v", got, want)
	}
}
