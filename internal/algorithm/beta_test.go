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
