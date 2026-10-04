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

func TestBetaCDF(t *testing.T) {
	tests := []struct {
		a, b float64
		cdf  func(x float64) float64
	}{
		{1, 1, func(x float64) float64 { return x }},
		{2, 2, func(x float64) float64 { return 3*x*x - 2*x*x*x }},
		{0.5, 0.5, func(x float64) float64 { return 2 / math.Pi * math.Asin(math.Sqrt(x)) }},
		// For whole a and b, I_x(a, b) = P(Binomial(a+b-1, x) >= a).
		{2, 5, func(x float64) float64 { return 1 - math.Pow(1-x, 6) - 6*x*math.Pow(1-x, 5) }},
		{5, 2, func(x float64) float64 { return math.Pow(x, 6) + 6*math.Pow(x, 5)*(1-x) }},
	}
	for _, tt := range tests {
		for _, x := range []float64{0, 0.01, 0.2, 0.5, 0.7, 0.99, 1} {
			if got, want := betaCDF(x, tt.a, tt.b), tt.cdf(x); math.Abs(got-want) > 1e-10 {
				t.Errorf("betaCDF(%v, %v, %v) = %v, want %v", x, tt.a, tt.b, got, want)
			}
		}
	}
}

func TestBetaQuantile(t *testing.T) {
	for _, ab := range [][2]float64{{1, 1}, {2, 2}, {0.3, 0.7}, {5, 2}, {50, 80}} {
		for _, p := range []float64{0.001, 0.1, 0.5, 0.9, 0.999} {
			x := betaQuantile(p, ab[0], ab[1])
			if got := betaCDF(x, ab[0], ab[1]); math.Abs(got-p) > 1e-9 {
				t.Errorf("Beta(%v, %v): quantile %v is %v, whose CDF is %v", ab[0], ab[1], p, x, got)
			}
		}
	}
}
