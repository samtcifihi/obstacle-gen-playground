package algorithm

import (
	"math"
	"math/rand/v2"
	"slices"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

var evolvedAlgorithm = Algorithm{
	ID:   "evolved",
	Name: "Evolved",
	Description: "Places obstacles one at a time. Each round, every empty cell is scored by how far it can see " +
		"before an obstacle, how far it is from the edge, and some noise, then the next obstacle goes on a cell " +
		"chosen by score. Distances are divided by visibility: the number of cells between the centre cell " +
		"and the edge of an empty board, which on a hexagon is its edge length minus 1.",
	Params: slices.Concat(
		[]Param{
			{Name: "k_obstacles", Group: "General", Type: Int, Default: 16,
				Description: "The number of obstacles to place (if possible)"},
			{Name: "k_max_bank", Group: "General", Type: Int, Default: 0,
				Description: "Maximum contiguous group of obstacles allowed (0 = no limit)"},
		},
		conjugateParams("obstacles", "Distance to obstacles"),
		conjugateParams("edge", "Distance to edge"),
		[]Param{
			{Name: "k_beta", Group: "Noise", Type: Float, Default: 2.0, MinExclusive: true,
				Description: "Both parameters of the beta distribution the noise is drawn from"},
			{Name: "k_beta_coef", Group: "Noise", Type: Float, Default: 1.0,
				Description: "The coefficient the shifted beta sample is multiplied by"},
			{Name: "is_stretching", Group: "Selection", Type: Bool, Default: true,
				Description: "Stretch scores exponentially before choosing, otherwise shift them so the lowest is 1"},
			{Name: "k_stretch", Group: "Selection", Type: Float, Default: 8.0, MinExclusive: true, OnlyIf: "is_stretching",
				Description: "The strength of the stretching"},
			{Name: "is_weighted", Group: "Selection", Type: Bool, Default: true,
				Description: "Choose a random cell weighted by score, otherwise the highest-scoring cell"},
		},
	),
	run: func(b *board.Board, v Values, rng *rand.Rand) Trace {
		return Evolved(b, evolvedConfig(v), rng)
	},
}

// conjugateParams returns the parameters of the Conjugate that combines
// distances for term ("obstacles" or "edge").
func conjugateParams(term, group string) []Param {
	flattened := "is_" + term + "_flattened"
	return []Param{
		{Name: flattened, Group: group, Type: Bool, Default: true,
			Description: "Treat the 6 directions the same rather than grouping them by axis"},
		{Name: "f_" + term + "_a", Group: group, Type: Choice, Default: "sqrt", Options: scalarOptions,
			Description: "The first function of the conjugate"},
		{Name: "f_" + term + "_b", Group: group, Type: Choice, Default: "mean", Options: aggregateOptions,
			Description: "The second function of the conjugate"},
		{Name: "f_" + term + "_a'", Group: group, Type: Choice, Default: "square", Options: scalarOptions,
			Follows:     []Follow{{Param: "f_" + term + "_a", Inverse: true}},
			Description: "The third function of the conjugate: the inverse of the first, set automatically"},
		{Name: "f_" + term + "_dir", Group: group, Type: Choice, Default: "mean", Options: aggregateOptions, OnlyIf: "!" + flattened,
			Description: "Converts the 3 axes' conjugates to a single number"},
	}
}

func conjugateFrom(v Values, term string) Conjugate {
	return Conjugate{
		Flattened: v.Bool("is_" + term + "_flattened"),
		A:         scalarFuncs[v.Choice("f_"+term+"_a")],
		B:         aggregateFuncs[v.Choice("f_"+term+"_b")],
		Dir:       aggregateFuncs[v.Choice("f_"+term+"_dir")],
		AInv:      scalarFuncs[v.Choice("f_"+term+"_a'")],
	}
}

func evolvedConfig(v Values) EvolvedConfig {
	return EvolvedConfig{
		Obstacles:    v.Int("k_obstacles"),
		MaxBank:      v.Int("k_max_bank"),
		ObstacleTerm: conjugateFrom(v, "obstacles"),
		EdgeTerm:     conjugateFrom(v, "edge"),
		Beta:         v.Float("k_beta"),
		BetaCoef:     v.Float("k_beta_coef"),
		Stretching:   v.Bool("is_stretching"),
		Stretch:      v.Float("k_stretch"),
		Weighted:     v.Bool("is_weighted"),
	}
}

// EvolvedConfig holds the parameters of Evolved.
type EvolvedConfig struct {
	Obstacles    int       // k_obstacles
	MaxBank      int       // k_max_bank; 0 means no limit
	ObstacleTerm Conjugate // is_obstacles_flattened, f_obstacles_*
	EdgeTerm     Conjugate // is_edge_flattened, f_edge_*
	Beta         float64   // k_beta
	BetaCoef     float64   // k_beta_coef
	Stretching   bool      // is_stretching
	Stretch      float64   // k_stretch
	Weighted     bool      // is_weighted
}

// Conjugate reduces a cell's distances in each direction, grouped by axis
// as [[a, b], [c, d], [e, f]], to one number. Flattened, it is
//
//	AInv(B(A(a), A(b), A(c), A(d), A(e), A(f)))
//
// and otherwise each axis gets its own conjugate, which Dir combines:
//
//	Dir(AInv(B(A(a), A(b))), AInv(B(A(c), A(d))), AInv(B(A(e), A(f))))
//
// With AInv the inverse of A, each conjugate is a generalised mean: A =
// sqrt, B = mean, AInv = square gives the power mean with exponent 1/2.
type Conjugate struct {
	Flattened bool
	A, AInv   func(float64) float64
	B, Dir    func([]float64) float64
}

// Apply reduces distances s to one number.
func (c Conjugate) Apply(s [3][2]float64) float64 {
	if c.Flattened {
		flat := make([]float64, 0, 6)
		for _, axis := range s {
			flat = append(flat, c.A(axis[0]), c.A(axis[1]))
		}
		return c.AInv(c.B(flat))
	}
	perAxis := make([]float64, 0, 3)
	for _, axis := range s {
		perAxis = append(perAxis, c.AInv(c.B([]float64{c.A(axis[0]), c.A(axis[1])})))
	}
	return c.Dir(perAxis)
}

// Evolved places obstacles on b one at a time. Each round it scores the
// cells, then places an obstacle on one chosen by score:
//
//  1. Empty cells start with a score of 0. Obstacles have no score, so are
//     ignored by the rest of the round and can't be chosen.
//
//  2. Cells where an obstacle would make a bank (contiguous group of
//     obstacles) bigger than MaxBank have no score.
//
//  3. Add ObstacleTerm of the number of empty cells in each direction
//     before an obstacle, capped at the board's visibility, divided by
//     visibility. The edge of the board doesn't block: a direction with no
//     obstacle within visibility counts as visibility.
//
//  4. Add EdgeTerm of the number of empty cells in each direction before
//     the edge of the board, divided by visibility.
//
//     If either term is undefined or infinite (NaN or ±Inf), which some
//     combinations of functions give for some distances, the cell has no
//     score.
//
//  5. Add (s - 0.5) * BetaCoef, where s ~ Beta(Beta, Beta).
//
//  6. Stretch the scores; see stretch.
//
//  7. Place an obstacle on a random cell weighted by score if Weighted,
//     otherwise on the highest-scoring cell, breaking ties randomly.
//
// It stops once it has placed cfg.Obstacles obstacles or no cell has a
// score. The trace has a step for each obstacle placed, recording every
// scored cell's values for evolvedMetrics.
func Evolved(b *board.Board, cfg EvolvedConfig, rng *rand.Rand) Trace {
	g := grid{b: b, index: b.Index()}
	// A one-cell board has visibility 0, but then every distance is 0 too.
	visibility := max(b.Visibility(), 1)
	trace := Trace{Metrics: evolvedMetrics.metrics()}
	for len(trace.Steps) < cfg.Obstacles {
		var cells []scoredCell
		for _, i := range g.freeCells(cfg.MaxBank) {
			c := b.Cells[i]
			var clearance, toEdge [3][2]float64
			for a, axis := range board.Axes {
				for s, dir := range axis {
					clearance[a][s] = float64(g.clearance(c.Hex, dir, visibility))
					toEdge[a][s] = float64(g.emptyToEdge(c.Hex, dir))
				}
			}
			sc := scoredCell{
				cell:      i,
				obstacles: cfg.ObstacleTerm.Apply(clearance) / float64(visibility),
				edge:      cfg.EdgeTerm.Apply(toEdge) / float64(visibility),
				noise:     (beta(rng, cfg.Beta, cfg.Beta) - 0.5) * cfg.BetaCoef,
			}
			if !isFinite(sc.obstacles) || !isFinite(sc.edge) {
				continue
			}
			sc.score = sc.obstacles + sc.edge + sc.noise
			cells = append(cells, sc)
		}
		if len(cells) == 0 {
			trace.Note = "no cell had a score"
			break
		}

		stretch(cells, cfg.Stretching, cfg.Stretch)
		var pick int
		var chances []float64
		if cfg.Weighted {
			weights, total := make([]float64, len(cells)), 0.0
			for i, c := range cells {
				weights[i] = c.weight
				total += c.weight
			}
			pick = cells[weightedIndex(rng, weights, total)].cell
			chances = weightedChances(weights, total)
		} else {
			pick = bestChoice(cells, rng)
			chances = bestChances(cells)
		}
		b.Cells[pick].Obstacle = true

		step := Step{Placed: b.Cells[pick].Hex, Candidates: make([]Candidate, len(cells))}
		for i, chance := range chances {
			cells[i].chance = chance
			step.Candidates[i] = Candidate{Hex: b.Cells[cells[i].cell].Hex, Values: evolvedMetrics.values(cells[i])}
		}
		trace.Steps = append(trace.Steps, step)
	}
	return trace
}

// evolvedMetrics are the values Evolved records for each scored cell.
var evolvedMetrics = metricList[scoredCell]{
	{Metric{Name: "score", Description: "Score (steps 1–5)"}, func(c scoredCell) float64 { return c.score }},
	{chanceMetric, func(c scoredCell) float64 { return c.chance }},
	{Metric{Name: "weight", Description: "Score after stretching or shifting (step 6)"},
		func(c scoredCell) float64 { return c.weight }},
	{Metric{Name: "obstacles", Description: "Distance to obstacles term (step 3)"},
		func(c scoredCell) float64 { return c.obstacles }},
	{Metric{Name: "edge", Description: "Distance to edge term (step 4)"}, func(c scoredCell) float64 { return c.edge }},
	{Metric{Name: "noise", Description: "Noise term (step 5)"}, func(c scoredCell) float64 { return c.noise }},
}

// scoredCell is a cell that has a score in the current round.
type scoredCell struct {
	cell                   int     // index into Board.Cells
	obstacles, edge, noise float64 // terms from steps 3, 4 and 5
	score                  float64 // their sum
	weight                 float64 // score after step 6
	chance                 float64 // of being picked
}

// stretch sets each cell's weight from its score, making every weight
// positive. When stretching it maps each score s to e^(k(s - max)), where
// max is the highest score, so the best cell gets 1 and larger k favours it
// more. Otherwise it maps s to s + 1 - min, where min is the lowest score.
func stretch(cells []scoredCell, stretching bool, k float64) {
	lo, hi := cells[0].score, cells[0].score
	for _, c := range cells[1:] {
		lo, hi = min(lo, c.score), max(hi, c.score)
	}
	for i := range cells {
		if stretching {
			cells[i].weight = math.Exp(k * (cells[i].score - hi))
		} else {
			cells[i].weight = cells[i].score + 1 - lo
		}
	}
}

// bestChoice returns the cell with the highest weight, choosing uniformly at
// random between ties.
func bestChoice(cells []scoredCell, rng *rand.Rand) int {
	best, ties := cells[0], 1
	for _, c := range cells[1:] {
		switch {
		case c.weight > best.weight:
			best, ties = c, 1
		case c.weight == best.weight:
			ties++
			if rng.IntN(ties) == 0 {
				best = c
			}
		}
	}
	return best.cell
}

// bestChances returns each cell's chance of being picked by bestChoice.
func bestChances(cells []scoredCell) []float64 {
	p := make([]float64, len(cells))
	best, ties := cells[0].weight, 0
	for _, c := range cells {
		best = max(best, c.weight)
	}
	for _, c := range cells {
		if c.weight == best {
			ties++
		}
	}
	for i, c := range cells {
		if c.weight == best {
			p[i] = 1 / float64(ties)
		}
	}
	return p
}
