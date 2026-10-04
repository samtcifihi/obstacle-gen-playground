package algorithm

import (
	"math"
	"math/rand/v2"
	"slices"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

var scoredAlgorithm = Algorithm{
	ID:   "scored",
	Name: "Scored",
	Description: "Places obstacles one at a time. Each round, every empty cell is scored by how far it can see " +
		"before an obstacle, how far it is from the edge, and some noise, then the next obstacle goes on a cell " +
		"chosen by score. Distances are divided by visibility: the most steps it takes to walk off an empty " +
		"board from its centre cell, which on a hexagon is its edge length.",
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
	run: func(b *board.Board, v Values, rng *rand.Rand) int {
		return Scored(b, scoredConfig(v), rng)
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
		{Name: "f_" + term + "_dir", Group: group, Type: Choice, Default: "mean", Options: aggregateOptions, OnlyIf: "!" + flattened,
			Description: "Converts the 3 axes to a single number"},
		{Name: "f_" + term + "_a'", Group: group, Type: Choice, Default: "square", Options: scalarOptions,
			Description: "The third function of the conjugate (should be the inverse of the first)"},
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

func scoredConfig(v Values) ScoredConfig {
	return ScoredConfig{
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

var scalarOptions = []Option{
	{Value: "sqrt", Label: "2 root"},
	{Value: "square", Label: "2 ^"},
	{Value: "identity", Label: "T f -> T :: x"},
}

var scalarFuncs = map[string]func(float64) float64{
	"sqrt":     math.Sqrt,
	"square":   func(x float64) float64 { return x * x },
	"identity": func(x float64) float64 { return x },
}

var aggregateOptions = []Option{
	{Value: "mean", Label: "mean"},
	{Value: "geom_mean", Label: "geom_mean"},
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
		return math.Pow(product, 1/float64(len(xs)))
	},
	"max": slices.Max[[]float64],
	"min": slices.Min[[]float64],
}

// ScoredConfig holds the parameters of Scored.
type ScoredConfig struct {
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
// and otherwise
//
//	AInv(Dir(B(A(a), A(b)), B(A(c), A(d)), B(A(e), A(f))))
//
// With AInv the inverse of A, this is a generalised mean: A = sqrt,
// B = mean, AInv = square gives the power mean with exponent 1/2.
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
		perAxis = append(perAxis, c.B([]float64{c.A(axis[0]), c.A(axis[1])}))
	}
	return c.AInv(c.Dir(perAxis))
}

// Scored places obstacles on b one at a time. Each round it scores the
// cells, then places an obstacle on one chosen by score:
//
//  1. Empty cells start with a score of 0. Obstacles have no score, so
//     are ignored by the rest of the round and can't be chosen.
//  2. Cells where an obstacle would make a bank (contiguous group of
//     obstacles) bigger than MaxBank have no score.
//  3. Add ObstacleTerm of the number of empty cells in each direction
//     before an obstacle, capped at the board's visibility, divided by
//     visibility. The edge of the board doesn't block: a direction with
//     no obstacle within visibility counts as visibility.
//  4. Add EdgeTerm of the number of empty cells in each direction before
//     the edge of the board, divided by visibility.
//  5. Add (s - 0.5) * BetaCoef, where s ~ Beta(Beta, Beta).
//  6. Stretch the scores; see stretch.
//  7. Place an obstacle on a random cell weighted by score if Weighted,
//     otherwise on the highest-scoring cell, breaking ties randomly.
//
// It stops once it has placed cfg.Obstacles obstacles or no cell has a
// score, and returns the number placed.
func Scored(b *board.Board, cfg ScoredConfig, rng *rand.Rand) int {
	g := grid{b: b, index: b.Index()}
	visibility := b.Visibility()
	placed := 0
	for ; placed < cfg.Obstacles; placed++ {
		banks := g.banks()
		var candidates []candidate
		for i, c := range b.Cells {
			if c.Obstacle || (cfg.MaxBank > 0 && banks.sizeWith(g, c.Hex) > cfg.MaxBank) {
				continue
			}
			var clearance, toEdge [3][2]float64
			for a, axis := range board.Axes {
				for s, dir := range axis {
					clearance[a][s] = float64(g.clearance(c.Hex, dir, visibility))
					toEdge[a][s] = float64(g.emptyToEdge(c.Hex, dir))
				}
			}
			score := cfg.ObstacleTerm.Apply(clearance) / float64(visibility)
			score += cfg.EdgeTerm.Apply(toEdge) / float64(visibility)
			score += (beta(rng, cfg.Beta, cfg.Beta) - 0.5) * cfg.BetaCoef
			candidates = append(candidates, candidate{cell: i, score: score})
		}
		if len(candidates) == 0 {
			break
		}

		stretch(candidates, cfg.Stretching, cfg.Stretch)
		var pick int
		if cfg.Weighted {
			pick = weightedChoice(candidates, rng)
		} else {
			pick = bestChoice(candidates, rng)
		}
		b.Cells[pick].Obstacle = true
	}
	return placed
}

type candidate struct {
	cell  int // index into Board.Cells
	score float64
}

// stretch makes every score positive so they can be used as weights. When
// stretching it maps each score s to e^(k(s - max)), where max is the
// highest score, so the best cell gets 1 and larger k favours it more.
// Otherwise it maps s to s + 1 - min, where min is the lowest score.
func stretch(cs []candidate, stretching bool, k float64) {
	lo, hi := cs[0].score, cs[0].score
	for _, c := range cs[1:] {
		lo, hi = min(lo, c.score), max(hi, c.score)
	}
	for i := range cs {
		if stretching {
			cs[i].score = math.Exp(k * (cs[i].score - hi))
		} else {
			cs[i].score += 1 - lo
		}
	}
}

// weightedChoice returns the cell of a random candidate, chosen with
// probability proportional to its score.
func weightedChoice(cs []candidate, rng *rand.Rand) int {
	total := 0.0
	for _, c := range cs {
		total += c.score
	}
	r := rng.Float64() * total
	fallback := cs[0].cell
	for _, c := range cs {
		if r < c.score {
			return c.cell
		}
		r -= c.score
		if c.score > 0 {
			fallback = c.cell
		}
	}
	// Rounding left r just past the end: take the last candidate that
	// had a chance.
	return fallback
}

// bestChoice returns the cell of the highest-scoring candidate, choosing
// uniformly at random between ties.
func bestChoice(cs []candidate, rng *rand.Rand) int {
	best, ties := cs[0], 1
	for _, c := range cs[1:] {
		switch {
		case c.score > best.score:
			best, ties = c, 1
		case c.score == best.score:
			ties++
			if rng.IntN(ties) == 0 {
				best = c
			}
		}
	}
	return best.cell
}

// grid looks up cells on a board by position.
type grid struct {
	b     *board.Board
	index map[board.Hex]int
}

// at reports whether h is on the board and, if so, whether it holds an
// obstacle.
func (g grid) at(h board.Hex) (onBoard, obstacle bool) {
	i, ok := g.index[h]
	return ok, ok && g.b.Cells[i].Obstacle
}

// clearance returns the number of empty cells from h in direction dir
// before an obstacle, up to limit. The edge of the board doesn't block.
func (g grid) clearance(h, dir board.Hex, limit int) int {
	for k := 0; k < limit; k++ {
		h = h.Add(dir)
		onBoard, obstacle := g.at(h)
		if !onBoard {
			break
		}
		if obstacle {
			return k
		}
	}
	return limit
}

// emptyToEdge returns the number of empty cells from h in direction dir
// to the edge of the board.
func (g grid) emptyToEdge(h, dir board.Hex) int {
	n := 0
	for {
		h = h.Add(dir)
		onBoard, obstacle := g.at(h)
		if !onBoard {
			return n
		}
		if !obstacle {
			n++
		}
	}
}

// banks records which bank (contiguous group of obstacles) each obstacle
// belongs to.
type banks struct {
	of    []int // bank of each cell, or -1 if it's empty
	sizes []int // number of obstacles in each bank
}

func (g grid) banks() banks {
	bs := banks{of: make([]int, len(g.b.Cells))}
	for i := range bs.of {
		bs.of[i] = -1
	}
	for i, c := range g.b.Cells {
		if !c.Obstacle || bs.of[i] >= 0 {
			continue
		}
		bank := len(bs.sizes)
		bs.sizes = append(bs.sizes, 0)
		stack := []int{i}
		bs.of[i] = bank
		for len(stack) > 0 {
			j := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			bs.sizes[bank]++
			for _, n := range g.b.Cells[j].Neighbours() {
				if k, ok := g.index[n]; ok && g.b.Cells[k].Obstacle && bs.of[k] < 0 {
					bs.of[k] = bank
					stack = append(stack, k)
				}
			}
		}
	}
	return bs
}

// sizeWith returns the size of the bank an obstacle on empty hex h would
// be part of.
func (bs banks) sizeWith(g grid, h board.Hex) int {
	size := 1
	var joined []int
	for _, n := range h.Neighbours() {
		i, ok := g.index[n]
		if !ok || bs.of[i] < 0 || slices.Contains(joined, bs.of[i]) {
			continue
		}
		joined = append(joined, bs.of[i])
		size += bs.sizes[bs.of[i]]
	}
	return size
}
