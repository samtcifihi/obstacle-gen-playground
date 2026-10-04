// Package algorithm contains obstacle placement algorithms.
package algorithm

import (
	"math/rand/v2"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

var uniformAlgorithm = Algorithm{
	ID:          "uniform",
	Name:        "Uniform random",
	Description: "Places n obstacles, each on an empty cell chosen with equal probability.",
	Params: []Param{
		{Name: "n", Type: Int, Default: 10, Description: "The number of obstacles to place (if possible)"},
	},
	run: func(b *board.Board, v Values, rng *rand.Rand) Trace {
		return Uniform(b, v.Int("n"), rng)
	},
}

// Uniform places n obstacles on b, each on an empty cell chosen uniformly
// at random from the cells still empty. If n is at least the number of
// empty cells, every cell ends up with an obstacle. The trace has a step for
// each obstacle placed.
func Uniform(b *board.Board, n int, rng *rand.Rand) Trace {
	var empty []int
	for i, c := range b.Cells {
		if !c.Obstacle {
			empty = append(empty, i)
		}
	}
	n = max(0, min(n, len(empty)))
	trace := Trace{Metrics: []Metric{chanceMetric}}
	// Partial Fisher-Yates shuffle: empty[:i] holds the cells picked so far.
	for i := 0; i < n; i++ {
		chance := 1 / float64(len(empty)-i)
		candidates := make([]Candidate, 0, len(empty)-i)
		for _, c := range empty[i:] {
			candidates = append(candidates, Candidate{Hex: b.Cells[c].Hex, Values: []float64{chance}})
		}
		j := i + rng.IntN(len(empty)-i)
		empty[i], empty[j] = empty[j], empty[i]
		b.Cells[empty[i]].Obstacle = true
		trace.Steps = append(trace.Steps, Step{Placed: b.Cells[empty[i]].Hex, Candidates: candidates})
	}
	return trace
}
