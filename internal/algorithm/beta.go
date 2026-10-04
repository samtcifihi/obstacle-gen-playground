package algorithm

import (
	"math"
	"math/rand/v2"
)

// beta returns a sample from the Beta(a, b) distribution, as X/(X+Y) for
// X ~ Gamma(a) and Y ~ Gamma(b). It works with logs so that small shapes,
// whose gamma samples can underflow to 0, still give a valid result.
func beta(rng *rand.Rand, a, b float64) float64 {
	logX, logY := logGamma(rng, a), logGamma(rng, b)
	return 1 / (1 + math.Exp(logY-logX))
}

// logGamma returns the log of a sample from the Gamma(shape, 1)
// distribution, using Marsaglia and Tsang's method.
func logGamma(rng *rand.Rand, shape float64) float64 {
	if shape < 1 {
		// If X ~ Gamma(shape+1) and U ~ Uniform(0, 1], X·U^(1/shape) ~ Gamma(shape).
		return logGamma(rng, shape+1) + math.Log(1-rng.Float64())/shape
	}
	d := shape - 1.0/3
	c := 1 / math.Sqrt(9*d)
	for {
		x := rng.NormFloat64()
		v := 1 + c*x
		if v <= 0 {
			continue
		}
		v = v * v * v
		u := 1 - rng.Float64()
		if math.Log(u) < x*x/2+d*(1-v+math.Log(v)) {
			return math.Log(d * v)
		}
	}
}

// betaCDF returns P(X <= x) for X ~ Beta(a, b): the regularized incomplete
// beta function, evaluated with a continued fraction (Numerical Recipes,
// section 6.4).
func betaCDF(x, a, b float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	lgab, _ := math.Lgamma(a + b)
	lga, _ := math.Lgamma(a)
	lgb, _ := math.Lgamma(b)
	front := math.Exp(lgab - lga - lgb + a*math.Log(x) + b*math.Log1p(-x))
	// The continued fraction converges fastest on this side of the mean.
	if x < (a+1)/(a+b+2) {
		return front * betaContinuedFraction(x, a, b) / a
	}
	return 1 - front*betaContinuedFraction(1-x, b, a)/b
}

// betaContinuedFraction evaluates the continued fraction for the
// incomplete beta function by the modified Lentz method.
func betaContinuedFraction(x, a, b float64) float64 {
	const (
		maxIterations = 10_000
		epsilon       = 1e-15
		tiny          = 1e-300
	)
	clamp := func(v float64) float64 {
		if math.Abs(v) < tiny {
			return tiny
		}
		return v
	}
	c, d := 1.0, 1/clamp(1-(a+b)*x/(a+1))
	h := d
	for m := 1.0; m <= maxIterations; m++ {
		// Even step.
		aa := m * (b - m) * x / ((a + 2*m - 1) * (a + 2*m))
		d = 1 / clamp(1+aa*d)
		c = clamp(1 + aa/c)
		h *= d * c
		// Odd step.
		aa = -(a + m) * (a + b + m) * x / ((a + 2*m) * (a + 2*m + 1))
		d = 1 / clamp(1+aa*d)
		c = clamp(1 + aa/c)
		delta := d * c
		h *= delta
		if math.Abs(delta-1) < epsilon {
			break
		}
	}
	return h
}

// betaQuantile returns the x with betaCDF(x, a, b) = p, found by bisection.
func betaQuantile(p, a, b float64) float64 {
	lo, hi := 0.0, 1.0
	for range 64 {
		mid := (lo + hi) / 2
		if betaCDF(mid, a, b) < p {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}
