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
	Description: "Places each obstacle by sampling three beta distributions, one along each axis through the " +
		"centre, each running from corner to corner. The obstacle goes on the hex whose position along each axis " +
		"best matches the samples. If that hex is off the board, already has an obstacle, or would make a bank " +
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
		},
		axisParams(1, "left corner", "right corner"),
		axisParams(2, "top left corner", "bottom right corner"),
		axisParams(3, "bottom left corner", "top right corner"),
	),
	run: func(b *board.Board, v Values, rng *rand.Rand) Trace {
		return TripleBeta(b, tripleBetaConfig(v), rng)
	},
}

// axisParams returns the α and β parameters of the distribution along
// axis, which runs from corner from to corner to.
func axisParams(axis int, from, to string) []Param {
	alpha, beta := fmt.Sprintf("k_alpha_%d", axis), fmt.Sprintf("k_beta_%d", axis)
	group := fmt.Sprintf("Axis %d: %s to %s", axis, from, to)
	var alphaFollows, betaFollows []Follow
	if axis > 1 {
		alphaFollows = []Follow{{When: "is_axes_shared", Param: "k_alpha_1"}}
		betaFollows = []Follow{{When: "is_axes_shared", Param: "k_beta_1"}}
	}
	betaFollows = append(betaFollows, Follow{When: "is_symmetric", Param: alpha})
	return []Param{
		{Name: alpha, Group: group, Type: Float, Default: 1.0, MinExclusive: true, Follows: alphaFollows,
			Description: "α of the beta distribution: raising it moves samples towards the " + to},
		{Name: beta, Group: group, Type: Float, Default: 1.0, MinExclusive: true, Follows: betaFollows,
			Description: "β of the beta distribution: raising it moves samples towards the " + from},
	}
}

func tripleBetaConfig(v Values) TripleBetaConfig {
	cfg := TripleBetaConfig{
		Obstacles: v.Int("k_obstacles"),
		MaxBank:   v.Int("k_max_bank"),
		MaxTries:  v.Int("k_max_tries"),
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
	MaxTries  int // k_max_tries
	// Axes holds the shape of the distribution along each axis: left to
	// right, top left to bottom right, and bottom left to top right.
	Axes [3]BetaShape
}

// BetaShape holds the parameters of a beta distribution.
type BetaShape struct {
	Alpha, Beta float64
}

// axes are the directions of the three axes through the centre, matching
// TripleBetaConfig.Axes. Each runs from corner to corner.
var axes = [3]board.Hex{{Q: 1, R: 0}, {Q: 0, R: 1}, {Q: 1, R: -1}}

// TripleBeta places obstacles on b one at a time. For each try it samples
// a position along each axis from that axis's beta distribution, scaled
// so that 0.5 is the centre cell and 0 and 1 are the outer edges of the
// corner cells. The try lands on the hex whose positions along the axes
// best match the three samples (see fitHex). If that hex is off the
// board, already has an obstacle, or would make a bank bigger than
// MaxBank, the try fails.
//
// It stops once it has placed cfg.Obstacles obstacles, used cfg.MaxTries
// tries (successful or not), or no cell could take an obstacle. The trace
// has a step for each obstacle placed, recording each cell that could
// have taken it and the estimated chances of a try landing there.
func TripleBeta(b *board.Board, cfg TripleBetaConfig, rng *rand.Rand) Trace {
	g := grid{b: b, index: b.Index()}
	// Each axis has visibility cells either side of the centre cell, and
	// the outer edge of a corner cell is half a cell beyond its centre.
	halfLength := float64(b.Visibility()) + 0.5
	landing := landingChances(b, cfg.Axes, halfLength)
	trace := Trace{Metrics: tripleBetaMetrics}

	tries := 0
	for len(trace.Steps) < cfg.Obstacles {
		banks := g.banks()
		free := func(i int) bool {
			return !b.Cells[i].Obstacle && (cfg.MaxBank == 0 || banks.sizeWith(g, b.Cells[i].Hex) <= cfg.MaxBank)
		}
		var open []int
		openLanding := 0.0
		for i := range b.Cells {
			if free(i) {
				open = append(open, i)
				openLanding += landing[i]
			}
		}
		if len(open) == 0 {
			trace.Note = "no free cell could take an obstacle"
			break
		}

		pick, stepTries := -1, 0
		for pick < 0 && tries < cfg.MaxTries {
			tries++
			stepTries++
			if i, ok := g.index[sampleHex(rng, cfg.Axes, halfLength)]; ok && free(i) {
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
			chance := 0.0
			if openLanding > 0 {
				chance = landing[i] / openLanding
			}
			step.Candidates[j] = Candidate{Hex: b.Cells[i].Hex, Values: []float64{chance, landing[i]}}
		}
		trace.Steps = append(trace.Steps, step)
	}
	return trace
}

// tripleBetaMetrics are the values TripleBeta records for each cell that
// could take an obstacle.
var tripleBetaMetrics = []Metric{
	{Name: "chance", Description: "Chance of getting this obstacle (estimated)", Percent: true},
	{Name: "landing", Description: "Chance a single try lands here (estimated)", Percent: true},
}

// sampleHex returns where one try lands.
func sampleHex(rng *rand.Rand, shapes [3]BetaShape, halfLength float64) board.Hex {
	var t [3]float64
	for i, s := range shapes {
		t[i] = (2*beta(rng, s.Alpha, s.Beta) - 1) * halfLength
	}
	return fitHex(t)
}

// fitHex returns the hex nearest the point whose positions along the
// three axes best match t, in cells from the centre, in the least-squares
// sense.
//
// A point's position along an axis is its projection onto that axis. The
// axes are 60° apart, so the three projections of a point p onto unit
// vectors u₁, u₂, u₃ satisfy Σ (p·uᵢ) uᵢ = 3/2 p. The least-squares point
// for positions t is therefore 2/3 Σ tᵢ uᵢ: exactly the point itself if
// the positions are consistent, and the closest compromise otherwise.
func fitHex(t [3]float64) board.Hex {
	var q, r float64
	for i, dir := range axes {
		q += 2.0 / 3 * t[i] * float64(dir.Q)
		r += 2.0 / 3 * t[i] * float64(dir.R)
	}
	return board.Round(q, r)
}

// quantilePoints is how many points per axis landingChances evaluates.
const quantilePoints = 96

// landingChances estimates the chance that a try lands on each cell of b.
// It evaluates fitHex at every combination of quantilePoints evenly
// spaced quantiles of each axis's distribution, each combination being
// equally likely, so cells that a try reaches very rarely can come out
// as 0.
func landingChances(b *board.Board, shapes [3]BetaShape, halfLength float64) []float64 {
	var t [3][quantilePoints]float64
	for i, s := range shapes {
		for k := range quantilePoints {
			x := betaQuantile((float64(k)+0.5)/quantilePoints, s.Alpha, s.Beta)
			t[i][k] = (2*x - 1) * halfLength
		}
	}

	// A dense lookup from position to cell, which is much faster than the
	// board's index map in the loop below.
	lo, hi := 0, 0
	for _, c := range b.Cells {
		lo, hi = min(lo, c.Q, c.R), max(hi, c.Q, c.R)
	}
	width := hi - lo + 1
	cellAt := make([]int, width*width)
	for i := range cellAt {
		cellAt[i] = -1
	}
	for i, c := range b.Cells {
		cellAt[(c.Q-lo)*width+c.R-lo] = i
	}

	chances := make([]float64, len(b.Cells))
	weight := 1.0 / (quantilePoints * quantilePoints * quantilePoints)
	for _, t1 := range t[0] {
		for _, t2 := range t[1] {
			for _, t3 := range t[2] {
				h := fitHex([3]float64{t1, t2, t3})
				if h.Q < lo || h.Q > hi || h.R < lo || h.R > hi {
					continue
				}
				if i := cellAt[(h.Q-lo)*width+h.R-lo]; i >= 0 {
					chances[i] += weight
				}
			}
		}
	}
	return chances
}
