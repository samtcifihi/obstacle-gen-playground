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
		"scaled so the coordinate's range on the board runs from 0 to 1, multiplied together. Each try picks a " +
		"hex with probability proportional to its weight. If it already has an obstacle or would make a bank " +
		"too big, it tries again, up to k_max_tries tries in all.",
	Params: slices.Concat(
		[]Param{
			{Name: "k_obstacles", Group: "General", Type: Int, Default: 16,
				Description: "The number of obstacles to place (if possible)"},
			{Name: "k_max_bank", Group: "General", Type: Int, Default: 0,
				Description: "Maximum contiguous group of obstacles allowed (0 = no limit)"},
			{Name: "k_max_tries", Group: "General", Type: Int, Default: 0,
				boardDefault: func(b *board.Board) any { return 2 * len(b.Cells) },
				Description:  "Tries, successful or not, before giving up (default twice the number of cells on the board)"},
			{Name: "is_symmetric", Group: "Distributions", Type: Bool, Default: true,
				Description: "Force α = β for each distribution, so each is symmetric about the centre"},
			{Name: "is_axes_shared", Group: "Distributions", Type: Bool, Default: true,
				Description: "Force all 3 distributions to use axis 1's α and β"},
			{Name: "is_inset", Group: "Distributions", Type: Bool, Default: false,
				Description: "Scale each coordinate to the middle of its band, (x + R + ½)/(2R + 1), instead of " +
					"(x/R + 1)/2, so the board's edges aren't at exactly 0 and 1. Keeps edge hexes possible when " +
					"α or β > 1, and is needed when α or β < 1, whose density is infinite at 0 or 1"},
		},
		axisParams(1, "q", "lower left edge", "upper right edge"),
		axisParams(2, "r", "top edge", "bottom edge"),
		axisParams(3, "s", "lower right edge", "upper left edge"),
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
		MaxTries:  v.Int("k_max_tries"),
		Inset:     v.Bool("is_inset"),
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
	Obstacles int  // k_obstacles
	MaxBank   int  // k_max_bank; 0 means no limit
	MaxTries  int  // k_max_tries
	Inset     bool // is_inset
	// Axes holds the shape of the distribution over each cube coordinate:
	// q, r and s.
	Axes [3]BetaShape
}

// BetaShape holds the parameters of a beta distribution.
type BetaShape struct {
	Alpha, Beta float64
}

// TripleBeta places obstacles on b one at a time. Each cell has a fixed
// weight (see hexWeights), and each try picks a cell of the board with
// probability proportional to its weight. The try fails if that cell
// already has an obstacle or one there would make a bank bigger than
// MaxBank.
//
// It stops once it has placed cfg.Obstacles obstacles, used cfg.MaxTries
// tries (successful or not), or no free cell has any weight. The trace has
// a step for each obstacle placed, recording each cell that could have
// taken it with its chances and weight.
func TripleBeta(b *board.Board, cfg TripleBetaConfig, rng *rand.Rand) Trace {
	g := grid{b: b, index: b.Index()}
	weights := hexWeights(b, cfg.Axes, cfg.Inset)
	trace := Trace{Metrics: tripleBetaMetrics}

	total := 0.0
	for _, w := range weights {
		if !isFinite(w) {
			trace.Note = "some hexes have infinite weight (a beta density with α or β below 1 is infinite at the " +
				"board's edges); turn on is_inset"
			return trace
		}
		total += w
	}

	tries := 0
	for len(trace.Steps) < cfg.Obstacles {
		banks := g.banks()
		free := func(i int) bool {
			return !b.Cells[i].Obstacle && (cfg.MaxBank == 0 || banks.sizeWith(g, b.Cells[i].Hex) <= cfg.MaxBank)
		}
		var open []int
		openWeight := 0.0
		for i := range b.Cells {
			if free(i) && weights[i] > 0 {
				open = append(open, i)
				openWeight += weights[i]
			}
		}
		if len(open) == 0 {
			trace.Note = "no free cell has any weight"
			break
		}

		pick, stepTries := -1, 0
		for pick < 0 && tries < cfg.MaxTries {
			tries++
			stepTries++
			if i := weightedIndex(rng, weights, total); free(i) {
				pick = i
			}
		}
		if pick < 0 {
			trace.Note = fmt.Sprintf("gave up after %d tries", cfg.MaxTries)
			break
		}
		b.Cells[pick].Obstacle = true

		step := Step{Placed: b.Cells[pick].Hex, Tries: stepTries, Candidates: make([]Candidate, len(open))}
		for j, i := range open {
			step.Candidates[j] = Candidate{
				Hex:    b.Cells[i].Hex,
				Values: []float64{weights[i] / openWeight, weights[i] / total, weights[i]},
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
	{Name: "landing", Description: "Chance a single try lands here", Percent: true},
	{Name: "weight", Description: "Weight (product of the three densities)"},
}

// hexWeights returns the weight of each cell of b: the product of the beta
// densities in shapes at its cube coordinates q, r and s, each scaled into
// [0, 1]. Normally a coordinate x in [-R, R], where R is the board's
// radius, scales to (x/R + 1)/2, so the board's extremes are at 0 and 1.
// With inset it scales to the middle of its band, (x + R + ½)/(2R + 1).
func hexWeights(b *board.Board, shapes [3]BetaShape, inset bool) []float64 {
	radius := 0
	for _, c := range b.Cells {
		radius = max(radius, abs(c.Q), abs(c.R), abs(c.Q+c.R))
	}
	scale := func(x int) float64 {
		switch {
		case inset:
			return (float64(x+radius) + 0.5) / float64(2*radius+1)
		case radius == 0:
			return 0.5
		}
		return (float64(x)/float64(radius) + 1) / 2
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
