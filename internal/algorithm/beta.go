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
