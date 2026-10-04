package algorithm

import (
	"math/rand/v2"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

func newRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, 0))
}

func countObstacles(b *board.Board) int {
	count := 0
	for _, c := range b.Cells {
		if c.Obstacle {
			count++
		}
	}
	return count
}
