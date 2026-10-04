package algorithm

import (
	"math"
	"testing"
)

func TestBetaMoments(t *testing.T) {
	const samples = 200_000
	for _, k := range []float64{0.1, 0.5, 1, 2, 10} {
		rng := newRNG(5)
		sum, sumSq := 0.0, 0.0
		for range samples {
			x := beta(rng, k, k)
			if x < 0 || x > 1 || math.IsNaN(x) {
				t.Fatalf("Beta(%v, %v) gave %v, outside [0, 1]", k, k, x)
			}
			sum += x
			sumSq += x * x
		}
		mean := sum / samples
		variance := sumSq/samples - mean*mean
		// Beta(k, k) has mean 1/2 and variance 1/(4(2k+1)).
		wantVariance := 1 / (4 * (2*k + 1))
		if math.Abs(mean-0.5) > 0.005 {
			t.Errorf("Beta(%v, %v) mean = %.4f, want 0.5", k, k, mean)
		}
		if math.Abs(variance-wantVariance) > 0.02*wantVariance {
			t.Errorf("Beta(%v, %v) variance = %.5f, want %.5f", k, k, variance, wantVariance)
		}
	}
}

func TestBetaAsymmetric(t *testing.T) {
	const samples = 200_000
	rng := newRNG(6)
	sum := 0.0
	for range samples {
		sum += beta(rng, 2, 6)
	}
	// Beta(a, b) has mean a/(a+b).
	if mean := sum / samples; math.Abs(mean-0.25) > 0.005 {
		t.Errorf("Beta(2, 6) mean = %.4f, want 0.25", mean)
	}
}

func TestBetaPDF(t *testing.T) {
	tests := []struct {
		a, b float64
		pdf  func(x float64) float64
	}{
		{1, 1, func(x float64) float64 { return 1 }},
		{2, 2, func(x float64) float64 { return 6 * x * (1 - x) }},
		{2, 5, func(x float64) float64 { return 30 * x * math.Pow(1-x, 4) }},
		{0.5, 0.5, func(x float64) float64 { return 1 / (math.Pi * math.Sqrt(x*(1-x))) }},
	}
	for _, tt := range tests {
		for _, x := range []float64{0.01, 0.2, 0.5, 0.7, 0.99} {
			if got, want := betaPDF(x, tt.a, tt.b), tt.pdf(x); math.Abs(got-want) > 1e-9*max(1, want) {
				t.Errorf("betaPDF(%v, %v, %v) = %v, want %v", x, tt.a, tt.b, got, want)
			}
		}
	}
	for _, tt := range []struct{ x, a, b, want float64 }{
		{0, 1, 1, 1}, {1, 1, 1, 1},
		{0, 2, 2, 0}, {1, 2, 2, 0},
		{0, 1, 3, 3}, {1, 3, 1, 3},
		{0, 0.5, 2, math.Inf(1)}, {1, 2, 0.5, math.Inf(1)},
	} {
		if got := betaPDF(tt.x, tt.a, tt.b); got != tt.want && math.Abs(got-tt.want) > 1e-12 {
			t.Errorf("betaPDF(%v, %v, %v) = %v, want %v", tt.x, tt.a, tt.b, got, tt.want)
		}
	}
}
