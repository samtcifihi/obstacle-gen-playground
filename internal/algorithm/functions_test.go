package algorithm

import (
	"math"
	"testing"
)

func TestAggregates(t *testing.T) {
	tests := []struct {
		name string
		xs   []float64
		want float64
	}{
		{"mean", []float64{1, 2, 6}, 3},
		{"geom_mean", []float64{1, 2, 4}, 2},
		{"geom_mean", []float64{0, 2, 4}, 0},
		{"harm_mean", []float64{1, 4, 4}, 2},
		{"harm_mean", []float64{0, 4, 4}, 0},
		{"median", []float64{5, 1, 3}, 3},
		{"median", []float64{5, 1, 4, 2}, 3},
		{"median", []float64{math.Inf(-1), 2}, math.Inf(-1)},
		{"max", []float64{1, 5, 3}, 5},
		{"min", []float64{4, 1, 3}, 1},
	}
	for _, tt := range tests {
		if got := aggregateFuncs[tt.name](tt.xs); got != tt.want && math.Abs(got-tt.want) > 1e-12 {
			t.Errorf("%s(%v) = %v, want %v", tt.name, tt.xs, got, tt.want)
		}
	}
	if got := aggregateFuncs["geom_mean"]([]float64{math.Inf(-1), 0, 1}); !math.IsNaN(got) {
		t.Errorf("geom_mean(-Inf, 0, 1) = %v, want NaN", got)
	}
}

func TestOptionsHaveFuncs(t *testing.T) {
	for _, o := range scalarOptions {
		f, inv := scalarFuncs[o.Value], scalarFuncs[o.Inverse]
		if f == nil || inv == nil {
			t.Errorf("scalar option %q or its inverse %q has no function", o.Value, o.Inverse)
			continue
		}
		for _, x := range []float64{0, 0.5, 1, 2.5, 8} {
			if got := inv(f(x)); math.Abs(got-x) > 1e-9 {
				t.Errorf("%s then %s maps %v to %v", o.Value, o.Inverse, x, got)
			}
		}
	}
	for _, o := range aggregateOptions {
		if aggregateFuncs[o.Value] == nil {
			t.Errorf("aggregate option %q has no function", o.Value)
		}
	}
}
