package algorithm

import "math"

func isFinite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

func notFinite(x float64) bool { return !isFinite(x) }

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
