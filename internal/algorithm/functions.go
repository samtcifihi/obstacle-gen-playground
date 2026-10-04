package algorithm

import (
	"math"
	"slices"
)

// scalarOptions are the functions of one number a Choice parameter can
// pick from, each with its inverse, and scalarFuncs implements them.
var scalarOptions = []Option{
	{Value: "sqrt", Label: "sqrt(x)", Inverse: "square"},
	{Value: "square", Label: "x^2", Inverse: "sqrt"},
	{Value: "identity", Label: "x", Inverse: "identity"},
	{Value: "ln", Label: "ln(x)", Inverse: "exp"},
	{Value: "exp", Label: "e^x", Inverse: "ln"},
}

var scalarFuncs = map[string]func(float64) float64{
	"sqrt":     math.Sqrt,
	"square":   func(x float64) float64 { return x * x },
	"identity": func(x float64) float64 { return x },
	"ln":       math.Log, // ln 0 is -Inf, which e^x maps back to 0
	"exp":      math.Exp,
}

// aggregateOptions are the functions combining several numbers into one a
// Choice parameter can pick from, and aggregateFuncs implements them.
var aggregateOptions = []Option{
	{Value: "mean", Label: "mean"},
	{Value: "geom_mean", Label: "geom_mean"},
	{Value: "harm_mean", Label: "harm_mean"},
	{Value: "median", Label: "median"},
	{Value: "max", Label: "max"},
	{Value: "min", Label: "min"},
}

var aggregateFuncs = map[string]func([]float64) float64{
	"mean": func(xs []float64) float64 {
		sum := 0.0
		for _, x := range xs {
			sum += x
		}
		return sum / float64(len(xs))
	},
	"geom_mean": func(xs []float64) float64 {
		product := 1.0
		for _, x := range xs {
			product *= x
		}
		if product < 0 {
			return math.NaN() // no real root
		}
		return math.Pow(product, 1/float64(len(xs)))
	},
	"harm_mean": func(xs []float64) float64 {
		// A 0 makes its reciprocal +Inf and the result 0, as in the limit.
		sum := 0.0
		for _, x := range xs {
			sum += 1 / x
		}
		return float64(len(xs)) / sum
	},
	"median": func(xs []float64) float64 {
		sorted := slices.Clone(xs)
		slices.Sort(sorted)
		mid := len(sorted) / 2
		if len(sorted)%2 == 1 {
			return sorted[mid]
		}
		return (sorted[mid-1] + sorted[mid]) / 2
	},
	"max": slices.Max[[]float64],
	"min": slices.Min[[]float64],
}
