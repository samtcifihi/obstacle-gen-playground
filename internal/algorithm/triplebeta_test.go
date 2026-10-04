package algorithm

import (
	"math"
	"testing"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

func shapes(alpha, beta float64) [3]BetaShape {
	return [3]BetaShape{{alpha, beta}, {alpha, beta}, {alpha, beta}}
}

func TestFitHexIsExactForConsistentPositions(t *testing.T) {
	// A hex's position along each axis is its projection onto that axis,
	// in cells: with s = -q - r, (q - s)/2, (r - s)/2 and (q - r)/2.
	for _, c := range board.NewHexagon(5).Cells {
		q, r := float64(c.Q), float64(c.R)
		s := -q - r
		if got := fitHex([3]float64{(q - s) / 2, (r - s) / 2, (q - r) / 2}); got != c.Hex {
			t.Errorf("positions of %v fit %v", c.Hex, got)
		}
	}
	// The corners are visibility cells along each axis.
	for i, dir := range axes {
		var pos [3]float64
		pos[i] = 4
		if got := fitHex(pos); got == (board.Hex{Q: 4 * dir.Q, R: 4 * dir.R}) {
			t.Errorf("a position of 4 along axis %d alone reached the corner; the other axes should pull it in", i+1)
		}
	}
}

func TestLandingChancesMatchSampling(t *testing.T) {
	const samples = 300_000
	b := board.NewHexagon(5)
	index := b.Index()
	halfLength := float64(b.Visibility()) + 0.5
	for _, s := range [][3]BetaShape{
		shapes(1, 1),
		shapes(2, 2),
		shapes(0.5, 0.5),
		{{3, 1}, {1, 1}, {0.7, 2}},
	} {
		want := landingChances(b, s, halfLength)
		got := make([]float64, len(b.Cells))
		rng := newRNG(9)
		for range samples {
			if i, ok := index[sampleHex(rng, s, halfLength)]; ok {
				got[i]++
			}
		}
		for i := range got {
			got[i] /= samples
			// Allow for sampling noise (about 4 standard errors) plus a
			// little for the estimate's own error.
			tolerance := 4*math.Sqrt(want[i]*(1-want[i])/samples) + 0.001
			if math.Abs(got[i]-want[i]) > tolerance {
				t.Errorf("shapes %v: cell %v estimated %.4f, sampled %.4f", s, b.Cells[i].Hex, want[i], got[i])
			}
		}
	}
}

func TestLandingChancesAreSymmetric(t *testing.T) {
	// With the same symmetric distribution on every axis, rotating the board
	// by 60° shouldn't change anything.
	b := board.NewHexagon(5)
	index := b.Index()
	chances := landingChances(b, shapes(1.5, 1.5), float64(b.Visibility())+0.5)
	total := 0.0
	for i, c := range b.Cells {
		total += chances[i]
		rotated := board.Hex{Q: -c.R, R: c.Q + c.R}
		if got := chances[index[rotated]]; math.Abs(got-chances[i]) > 1e-3 {
			t.Errorf("cell %v has chance %.4f, but %v rotated 60° has %.4f", c.Hex, chances[i], rotated, got)
		}
	}
	if total > 1+1e-9 {
		t.Errorf("chances add up to %v, more than 1", total)
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

func TestTripleBetaGivesUp(t *testing.T) {
	b := board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t, b)
	cfg.Obstacles = 61
	cfg.MaxTries = 30
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
	if len(trace.Steps) >= 30 || trace.Note != "gave up after 30 tries" {
		t.Errorf("placed %d obstacles with note %q, want fewer than 30 and to give up", len(trace.Steps), trace.Note)
	}
	checkTrace(t, "triple beta", b, trace)
}

func TestTripleBetaMaxBank(t *testing.T) {
	b := board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t, b)
	cfg.Obstacles = 61
	cfg.MaxBank = 1
	cfg.MaxTries = 100_000
	trace := TripleBeta(b, cfg, newRNG(2))
	g := grid{b: b, index: b.Index()}
	bs := g.banks()
	for bank, size := range bs.sizes {
		if size > 1 {
			t.Errorf("bank %d has %d obstacles", bank, size)
		}
	}
	if trace.Note != "no free cell could take an obstacle" {
		t.Errorf("stopped with note %q, want no free cell", trace.Note)
	}
	checkTrace(t, "triple beta", b, trace)
}

func TestTripleBetaFollowsDistribution(t *testing.T) {
	// Skewing every axis towards its "to" corner moves obstacles that way:
	// right (axis 1), down right (axis 2) and up right (axis 3), so towards
	// higher q overall.
	b := board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t, b)
	cfg.Axes = shapes(6, 1.5)
	cfg.Obstacles = 10
	TripleBeta(b, cfg, newRNG(3))
	sum := 0
	for _, c := range b.Cells {
		if c.Obstacle {
			sum += c.Q
		}
	}
	if sum <= 10 {
		t.Errorf("obstacles' q adds up to %d, want them well to the right", sum)
	}
}

func TestTripleBetaIsReproducible(t *testing.T) {
	a, b := board.NewHexagon(5), board.NewHexagon(5)
	cfg := defaultTripleBetaConfig(t, a)
	TripleBeta(a, cfg, newRNG(42))
	TripleBeta(b, cfg, newRNG(42))
	for i := range a.Cells {
		if a.Cells[i] != b.Cells[i] {
			t.Fatalf("same seed gave different boards at cell %d", i)
		}
	}
}

func BenchmarkLandingChances(bm *testing.B) {
	b := board.NewHexagon(5)
	for range bm.N {
		landingChances(b, shapes(2, 2), 4.5)
	}
}
