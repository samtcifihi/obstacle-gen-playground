package algorithm

import "math/rand/v2"

// weightedIndex returns a random index into weights, chosen with
// probability proportional to its weight. total must be their sum.
func weightedIndex(rng *rand.Rand, weights []float64, total float64) int {
	r := rng.Float64() * total
	fallback := 0
	for i, w := range weights {
		if r < w {
			return i
		}
		r -= w
		if w > 0 {
			fallback = i
		}
	}
	// Rounding left r just past the end: take the last index that had a
	// chance.
	return fallback
}

// weightedChances returns each weight's chance of being picked by
// weightedIndex: its share of total, which must be their sum.
func weightedChances(weights []float64, total float64) []float64 {
	chances := make([]float64, len(weights))
	for i, w := range weights {
		chances[i] = w / total
	}
	return chances
}
