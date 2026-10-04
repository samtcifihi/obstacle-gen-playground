package algorithm

import (
	"testing"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

func uniform(n int) UniformConfig {
	return UniformConfig{Obstacles: n}
}

func TestUniformPlacesObstacles(t *testing.T) {
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
		placed := len(Uniform(b, uniform(tt.n), newRNG(1)).Steps)
		if placed != tt.want {
			t.Errorf("Uniform(k_obstacles=%d) placed %d, want %d", tt.n, placed, tt.want)
		}
		if got := countObstacles(b); got != tt.want {
			t.Errorf("Uniform(k_obstacles=%d) left %d obstacles on the board, want %d", tt.n, got, tt.want)
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

	placed := len(Uniform(b, uniform(total), newRNG(1)).Steps)
	if placed != total-before {
		t.Errorf("placed %d obstacles, want %d (the number of empty cells)", placed, total-before)
	}
	if got := countObstacles(b); got != total {
		t.Errorf("board has %d obstacles, want it full with %d", got, total)
	}
}

func TestUniformIsReproducible(t *testing.T) {
	a, b := board.NewHexagon(5), board.NewHexagon(5)
	Uniform(a, uniform(20), newRNG(42))
	Uniform(b, uniform(20), newRNG(42))
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
		Uniform(b, uniform(3), rng)
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

func TestUniformMaxBank(t *testing.T) {
	for _, maxBank := range []int{1, 2, 3} {
		b := board.NewHexagon(5)
		trace := Uniform(b, UniformConfig{Obstacles: 1000, MaxBank: maxBank}, newRNG(uint64(maxBank)))
		g := grid{b: b, index: b.Index()}
		bs := g.banks()
		for bank, size := range bs.sizes {
			if size > maxBank {
				t.Errorf("k_max_bank=%d: bank %d has %d obstacles", maxBank, bank, size)
			}
		}
		// It stops once no cell can take an obstacle, before filling the board.
		if trace.Note != "no free cell left" || countObstacles(b) >= len(b.Cells) {
			t.Errorf("k_max_bank=%d: placed %d with note %q", maxBank, countObstacles(b), trace.Note)
		}
		for _, c := range b.Cells {
			if !c.Obstacle && bs.sizeWith(g, c.Hex) <= maxBank {
				t.Errorf("k_max_bank=%d: stopped with %v still free", maxBank, c.Hex)
			}
		}
		checkTrace(t, "uniform", b, trace)
	}
}
