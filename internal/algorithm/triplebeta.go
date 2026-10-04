package algorithm

import (
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

var tripleBetaAlgorithm = Algorithm{
	ID:   "triple_beta",
	Name: "Triple Beta",
	Description: "Weights every hex by three beta densities, one for each cube coordinate (q, r, s), each " +
		"scaled so the coordinate's bands across the board evenly split 0 to 1, multiplied together, and by a fourth " +
		"beta density over its distance to the nearest obstacle. Each obstacle goes on a free hex (no obstacle, " +
		"and not making a bank too big) picked with probability proportional to its weight.",
	Params: slices.Concat(
		[]Param{
			{Name: "k_obstacles", Group: "General", Type: Int, Default: 16,
				Description: "The number of obstacles to place (if possible)"},
			{Name: "k_max_bank", Group: "General", Type: Int, Default: 0,
				Description: "Maximum contiguous group of obstacles allowed (0 = no limit)"},
			{Name: "is_symmetric", Group: "Distributions", Type: Bool, Default: true,
				Description: "Force α = β for each axis's distribution, so each is symmetric about the centre"},
			{Name: "is_axes_shared", Group: "Distributions", Type: Bool, Default: true,
				Description: "Force all 3 distributions to use axis 1's α and β"},
		},
		axisParams(1, "q", "lower left edge", "upper right edge"),
		axisParams(2, "r", "top edge", "bottom edge"),
		axisParams(3, "s", "lower right edge", "upper left edge"),
		[]Param{
			{Name: "is_obstacles_symmetric", Group: "Distance to obstacles", Type: Bool, Default: false,
				Description: "Force α = β for the distance distribution"},
			{Name: "k_alpha_obstacles", Group: "Distance to obstacles", Type: Float, Default: 1.0, MinExclusive: true,
				Description: "α of the beta distribution over distance to the nearest obstacle: raising it favours " +
					"being far from obstacles, spreading them out"},
			{Name: "k_beta_obstacles", Group: "Distance to obstacles", Type: Float, Default: 1.0, MinExclusive: true,
				Follows: []Follow{{When: "is_obstacles_symmetric", Param: "k_alpha_obstacles"}},
				Description: "β of the beta distribution over distance to the nearest obstacle: raising it favours " +
					"being close to obstacles, clustering them"},
		},
	),
	run: func(b *board.Board, v Values, rng *rand.Rand) Trace {
		return TripleBeta(b, tripleBetaConfig(v), rng)
	},
}

// axisParams returns the α and β parameters of the distribution over cube
// coordinate coord, whose lowest value is along edge from and highest
// along edge to.
func axisParams(axis int, coord, from, to string) []Param {
	alpha, beta := fmt.Sprintf("k_alpha_%d", axis), fmt.Sprintf("k_beta_%d", axis)
	group := fmt.Sprintf("Axis %d (%s): %s to %s", axis, coord, from, to)
	var alphaFollows, betaFollows []Follow
	if axis > 1 {
		alphaFollows = []Follow{{When: "is_axes_shared", Param: "k_alpha_1"}}
		betaFollows = []Follow{{When: "is_axes_shared", Param: "k_beta_1"}}
	}
	betaFollows = append(betaFollows, Follow{When: "is_symmetric", Param: alpha})
	return []Param{
		{Name: alpha, Group: group, Type: Float, Default: 1.0, MinExclusive: true, Follows: alphaFollows,
			Description: fmt.Sprintf("α of the beta distribution over %s: raising it favours the %s", coord, to)},
		{Name: beta, Group: group, Type: Float, Default: 1.0, MinExclusive: true, Follows: betaFollows,
			Description: fmt.Sprintf("β of the beta distribution over %s: raising it favours the %s", coord, from)},
	}
}

func tripleBetaConfig(v Values) TripleBetaConfig {
	cfg := TripleBetaConfig{
		Obstacles: v.Int("k_obstacles"),
		MaxBank:   v.Int("k_max_bank"),
		Distance: BetaShape{
			Alpha: v.Float("k_alpha_obstacles"),
			Beta:  v.Float("k_beta_obstacles"),
		},
	}
	for i := range cfg.Axes {
		cfg.Axes[i] = BetaShape{
			Alpha: v.Float(fmt.Sprintf("k_alpha_%d", i+1)),
			Beta:  v.Float(fmt.Sprintf("k_beta_%d", i+1)),
		}
	}
	return cfg
}

// TripleBetaConfig holds the parameters of TripleBeta.
type TripleBetaConfig struct {
	Obstacles int // k_obstacles
	MaxBank   int // k_max_bank; 0 means no limit
	// Axes holds the shape of the distribution over each cube coordinate:
	// q, r and s.
	Axes [3]BetaShape
	// Distance is the shape of the distribution over distance to the
	// nearest obstacle (k_alpha_obstacles, k_beta_obstacles).
	Distance BetaShape
}

// BetaShape holds the parameters of a beta distribution.
type BetaShape struct {
	Alpha, Beta float64
}

// TripleBeta places obstacles on b one at a time. Each cell's weight is its
// position weight (see hexWeights), which is fixed, times its distance
// weight, which is recalculated each round: the density of cfg.Distance at
// the cell's distance to the nearest obstacle (see distanceWeight). Each
// obstacle goes on a free cell, one without an obstacle where one wouldn't
// make a bank bigger than MaxBank, picked with probability proportional to
// its weight.
//
// It stops once it has placed cfg.Obstacles obstacles or no free cell has
// any weight. The trace has a step for each obstacle placed, recording
// each cell that could have taken it with its chance, weights and distance.
func TripleBeta(b *board.Board, cfg TripleBetaConfig, rng *rand.Rand) Trace {
	g := grid{b: b, index: b.Index()}
	visibility := b.Visibility()
	positions := hexWeights(b, cfg.Axes)
	distances := make([]float64, visibility+1) // distance weight by distance
	for d := range distances {
		distances[d] = distanceWeight(d, visibility, cfg.Distance)
	}
	trace := Trace{Metrics: tripleBetaMetrics}
	notFinite := func(w float64) bool { return !isFinite(w) }
	if slices.ContainsFunc(positions, notFinite) || slices.ContainsFunc(distances, notFinite) {
		// Scaled coordinates and distances are never exactly 0 or 1, so only
		// extreme shapes can get here, by overflowing.
		trace.Note = "some weights are infinite or undefined; α or β is too extreme"
		return trace
	}

	type option struct {
		cell                   int
		distance               int
		distanceWeight, weight float64
	}
	for len(trace.Steps) < cfg.Obstacles {
		banks := g.banks()
		var obstacles []board.Hex
		for _, c := range b.Cells {
			if c.Obstacle {
				obstacles = append(obstacles, c.Hex)
			}
		}

		var options []option
		var weights []float64
		total, anyFree := 0.0, false
		for i, c := range b.Cells {
			if c.Obstacle || (cfg.MaxBank > 0 && banks.sizeWith(g, c.Hex) > cfg.MaxBank) {
				continue
			}
			anyFree = true
			d := nearestObstacle(c.Hex, obstacles, visibility)
			dw := distances[d]
			if w := positions[i] * dw; w > 0 {
				options = append(options, option{cell: i, distance: d, distanceWeight: dw, weight: w})
				weights = append(weights, w)
				total += w
			}
		}
		if len(options) == 0 {
			trace.Note = "no free cell left"
			if anyFree {
				trace.Note = "no free cell has any weight"
			}
			break
		}

		pick := options[weightedIndex(rng, weights, total)].cell
		b.Cells[pick].Obstacle = true

		step := Step{Placed: b.Cells[pick].Hex, Candidates: make([]Candidate, len(options))}
		for j, o := range options {
			step.Candidates[j] = Candidate{
				Hex:    b.Cells[o.cell].Hex,
				Values: []float64{o.weight / total, o.weight, positions[o.cell], float64(o.distance), o.distanceWeight},
			}
		}
		trace.Steps = append(trace.Steps, step)
	}
	return trace
}

// tripleBetaMetrics are the values TripleBeta records for each cell that
// could take an obstacle.
var tripleBetaMetrics = []Metric{
	{Name: "chance", Description: "Chance of getting this obstacle", Percent: true},
	{Name: "weight", Description: "Weight (position weight × distance weight)"},
	{Name: "position", Description: "Position weight (product of the three axis densities)"},
	{Name: "distance", Description: "Distance to the nearest obstacle (capped at visibility)", Integer: true},
	{Name: "distance_weight", Description: "Distance weight (density at the scaled distance)"},
}

// nearestObstacle returns the number of cells between h and the nearest of
// obstacles, so 0 if one is adjacent, capped at visibility. With no
// obstacles it's visibility.
func nearestObstacle(h board.Hex, obstacles []board.Hex, visibility int) int {
	d := visibility
	for _, o := range obstacles {
		d = min(d, h.DistanceTo(o)-1)
	}
	return d
}

// distanceWeight returns the density of shape at distance d, which runs
// from 0 to visibility, scaled into (0, 1) as the middle of its band,
// (d + ½)/(visibility + 1), so the density is never infinite.
func distanceWeight(d, visibility int, shape BetaShape) float64 {
	return betaPDF((float64(d)+0.5)/float64(visibility+1), shape.Alpha, shape.Beta)
}

// hexWeights returns the weight of each cell of b: the product of the beta
// densities in shapes at its cube coordinates q, r and s, each scaled into
// (0, 1). The 2R + 1 values a coordinate takes on a board of radius R split
// [0, 1] into equal bands, and a coordinate x scales to the middle of its
// band, (x + R + ½)/(2R + 1), so it's never exactly 0 or 1, where a beta
// density can be 0 or infinite.
func hexWeights(b *board.Board, shapes [3]BetaShape) []float64 {
	radius := 0
	for _, c := range b.Cells {
		radius = max(radius, abs(c.Q), abs(c.R), abs(c.Q+c.R))
	}
	scale := func(x int) float64 {
		return (float64(x+radius) + 0.5) / float64(2*radius+1)
	}
	weights := make([]float64, len(b.Cells))
	for i, c := range b.Cells {
		w := 1.0
		for j, x := range [3]int{c.Q, c.R, -c.Q - c.R} {
			w *= betaPDF(scale(x), shapes[j].Alpha, shapes[j].Beta)
		}
		weights[i] = w
	}
	return weights
}

// weightedIndex returns a random index into weights, chosen with
// probability proportional to its weight. total must be their sum.
func weightedIndex(rng *rand.Rand, weights []float64, total float64) int {
	r := rng.Float64() * total
	fallback := 0
	for i, w := range weights {
		if r < w {
			return i
		}
		r -= w
		if w > 0 {
			fallback = i
		}
	}
	// Rounding left r just past the end: take the last index that had a
	// chance.
	return fallback
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
