// Package algorithm contains obstacle placement algorithms.
package algorithm

import (
	"math/rand/v2"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

// Uniform places n obstacles on b, each on an empty cell chosen uniformly
// at random from the cells still empty. If n is at least the number of
// empty cells, every cell ends up with an obstacle. It returns the number
// of obstacles placed.
func Uniform(b *board.Board, n int, rng *rand.Rand) int {
	var empty []int
	for i, c := range b.Cells {
		if !c.Obstacle {
			empty = append(empty, i)
		}
	}
	n = max(0, min(n, len(empty)))
	// Partial Fisher-Yates shuffle: empty[:i] holds the cells picked so far.
	for i := 0; i < n; i++ {
		j := i + rng.IntN(len(empty)-i)
		empty[i], empty[j] = empty[j], empty[i]
		b.Cells[empty[i]].Obstacle = true
	}
	return n
}
