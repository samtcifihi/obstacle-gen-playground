package algorithm

import (
	"math/rand/v2"
	"testing"

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

func TestUniformPlacesN(t *testing.T) {
	total := len(board.NewHexagon(5).Cells)
	tests := []struct {
		n, want int
	}{
		{n: -1, want: 0},
		{n: 0, want: 0},
		{n: 1, want: 1},
		{n: 10, want: 10},
		{n: total - 1, want: total - 1},
		{n: total, want: total},
		{n: total + 100, want: total},
	}
	for _, tt := range tests {
		b := board.NewHexagon(5)
		placed := Uniform(b, tt.n, newRNG(1))
		if placed != tt.want {
			t.Errorf("Uniform(n=%d) returned %d, want %d", tt.n, placed, tt.want)
		}
		if got := countObstacles(b); got != tt.want {
			t.Errorf("Uniform(n=%d) left %d obstacles on the board, want %d", tt.n, got, tt.want)
		}
	}
}

func TestUniformOnlyUsesEmptyCells(t *testing.T) {
	b := board.NewHexagon(5)
	total := len(b.Cells)
	for i := 0; i < total; i += 2 {
		b.Cells[i].Obstacle = true
	}
	before := countObstacles(b)

	placed := Uniform(b, total, newRNG(1))
	if placed != total-before {
		t.Errorf("placed %d obstacles, want %d (the number of empty cells)", placed, total-before)
	}
	if got := countObstacles(b); got != total {
		t.Errorf("board has %d obstacles, want it full with %d", got, total)
	}
}

func TestUniformIsReproducible(t *testing.T) {
	a, b := board.NewHexagon(5), board.NewHexagon(5)
	Uniform(a, 20, newRNG(42))
	Uniform(b, 20, newRNG(42))
	for i := range a.Cells {
		if a.Cells[i] != b.Cells[i] {
			t.Fatalf("same seed gave different boards at cell %d: %v vs %v", i, a.Cells[i], b.Cells[i])
		}
	}
}

func TestUniformIsUniform(t *testing.T) {
	const trials = 100_000
	rng := newRNG(7)
	var counts []int
	for range trials {
		b := board.NewHexagon(5)
		Uniform(b, 3, rng)
		if counts == nil {
			counts = make([]int, len(b.Cells))
		}
		for i, c := range b.Cells {
			if c.Obstacle {
				counts[i]++
			}
		}
	}
	// Each cell should hold an obstacle in about 3/61 of trials.
	want := float64(trials) * 3 / float64(len(counts))
	for i, got := range counts {
		if diff := float64(got) - want; diff > want*0.1 || diff < -want*0.1 {
			t.Errorf("cell %d got an obstacle %d times, want about %.0f", i, got, want)
		}
	}
}
