// Package algorithm contains obstacle placement algorithms.
package algorithm

import (
	"math/rand/v2"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

var uniformAlgorithm = Algorithm{
	ID:   "uniform",
	Name: "Uniform",
	Description: "Places obstacles one at a time, each on a free hex (no obstacle, and not making a bank too big) " +
		"chosen with equal probability.",
	Params: []Param{
		{Name: "k_obstacles", Type: Int, Default: 16,
			Description: "The number of obstacles to place (if possible)"},
		{Name: "k_max_bank", Type: Int, Default: 0,
			Description: "Maximum contiguous group of obstacles allowed (0 = no limit)"},
	},
	run: func(b *board.Board, v Values, rng *rand.Rand) Trace {
		return Uniform(b, UniformConfig{Obstacles: v.Int("k_obstacles"), MaxBank: v.Int("k_max_bank")}, rng)
	},
}

// UniformConfig holds the parameters of Uniform.
type UniformConfig struct {
	Obstacles int // k_obstacles
	MaxBank   int // k_max_bank; 0 means no limit
}

// Uniform places obstacles on b one at a time, each on a free cell, one
// without an obstacle where one wouldn't make a bank bigger than MaxBank,
// chosen uniformly at random. It stops once it has placed cfg.Obstacles
// obstacles or no cell is free. The trace has a step for each obstacle
// placed.
func Uniform(b *board.Board, cfg UniformConfig, rng *rand.Rand) Trace {
	trace := Trace{Metrics: []Metric{chanceMetric}}
	// Banks only matter with a limit, so don't work them out otherwise.
	var g grid
	if cfg.MaxBank > 0 {
		g = grid{b: b, index: b.Index()}
	}
	for len(trace.Steps) < cfg.Obstacles {
		var bs banks
		if cfg.MaxBank > 0 {
			bs = g.banks()
		}
		var free []int
		for i, c := range b.Cells {
			if !c.Obstacle && (cfg.MaxBank == 0 || bs.sizeWith(g, c.Hex) <= cfg.MaxBank) {
				free = append(free, i)
			}
		}
		if len(free) == 0 {
			trace.Note = "no free cell left"
			break
		}

		pick := free[rng.IntN(len(free))]
		b.Cells[pick].Obstacle = true

		chance := 1 / float64(len(free))
		step := Step{Placed: b.Cells[pick].Hex, Candidates: make([]Candidate, len(free))}
		for j, i := range free {
			step.Candidates[j] = Candidate{Hex: b.Cells[i].Hex, Values: []float64{chance}}
		}
		trace.Steps = append(trace.Steps, step)
	}
	return trace
}
