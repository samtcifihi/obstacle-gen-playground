package algorithm

import (
	"fmt"
	"math"
	"testing"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

func metricIndex(t *testing.T, trace Trace, name string) int {
	t.Helper()
	for i, m := range trace.Metrics {
		if m.Name == name {
			return i
		}
	}
	t.Fatalf("trace has no %q metric", name)
	return -1
}

// checkTrace checks that trace is consistent with the final board b, which
// started empty.
func checkTrace(t *testing.T, name string, b *board.Board, trace Trace) {
	t.Helper()
	index := b.Index()
	placed := make(map[board.Hex]bool)
	chance := metricIndex(t, trace, "chance")
	for i, step := range trace.Steps {
		if !b.Cells[index[step.Placed]].Obstacle {
			t.Errorf("%s: step %d placed %v, which has no obstacle on the final board", name, i, step.Placed)
		}
		total, chosen := 0.0, false
		for _, c := range step.Candidates {
			if len(c.Values) != len(trace.Metrics) {
				t.Fatalf("%s: step %d candidate %v has %d values for %d metrics", name, i, c.Hex, len(c.Values), len(trace.Metrics))
			}
			if placed[c.Hex] {
				t.Errorf("%s: step %d offers %v, which already has an obstacle", name, i, c.Hex)
			}
			total += c.Values[chance]
			chosen = chosen || c.Hex == step.Placed
		}
		if !chosen {
			t.Errorf("%s: step %d placed %v, which wasn't a candidate", name, i, step.Placed)
		}
		if math.Abs(total-1) > 1e-9 {
			t.Errorf("%s: step %d chances add up to %v, want 1", name, i, total)
		}
		placed[step.Placed] = true
	}
	if len(placed) != countObstacles(b) {
		t.Errorf("%s: trace placed %d distinct cells, but the board has %d obstacles", name, len(placed), countObstacles(b))
	}
}

func TestTraces(t *testing.T) {
	b := board.NewHexagon(5)
	checkTrace(t, "uniform", b, Uniform(b, UniformConfig{Obstacles: 20}, newRNG(1)))

	for _, weighted := range []bool{true, false} {
		cfg := defaultEvolvedConfig(t)
		cfg.Weighted = weighted
		cfg.MaxBank = 2
		b := board.NewHexagon(5)
		checkTrace(t, fmt.Sprintf("evolved, weighted=%v", weighted), b, Evolved(b, cfg, newRNG(1)))
	}
}
