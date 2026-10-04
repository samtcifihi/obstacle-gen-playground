package algorithm

import (
	"math"
	"math/rand/v2"
	"slices"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

var splitAlgorithm = Algorithm{
	ID:   "split",
	Name: "Split",
	Description: "Places obstacles greedily where they best break up long clear lines across the board. Each " +
		"round, every eligible hex scores how evenly it splits the clear line along each axis, less a penalty " +
		"for touching obstacles, plus a little noise. Scores are rounded into buckets, and the obstacle " +
		"goes on a random hex among the best.",
	Params: []Param{
		{Name: "k_obstacles", Group: "General", Type: Int, Default: 16,
			Description: "The number of obstacles to place (if possible)"},
		{Name: "k_max_bank", Group: "General", Type: Int, Default: 0,
			Description: "Maximum contiguous group of obstacles allowed (0 = no limit)"},
		{Name: "is_symmetric", Group: "General", Type: Bool, Default: true,
			Description: "Place obstacles in pairs, each the other rotated 180° about the centre, except the " +
				"centre itself, which is placed on its own"},
		{Name: "k_edge_margin", Group: "Eligibility", Type: Int, Default: 1,
			Description: "Fewest cells allowed between an obstacle and the edge (0 allows the perimeter)"},
		{Name: "k_max_adjacent", Group: "Eligibility", Type: Int, Default: 1,
			Description: "Most obstacles a new one may touch (0 = none, 6 = no limit)"},
		{Name: "f_split_axes", Group: "Score", Type: Choice, Default: "max", Options: aggregateOptions,
			Description: "Combines the three axes' splits: max rewards one good line-break, min rewards breaking " +
				"lines in every direction"},
		{Name: "k_adjacent_penalty", Group: "Score", Type: Float, Default: 1.0,
			Description: "Score penalty per obstacle a hex touches"},
		{Name: "k_noise", Group: "Score", Type: Int, Default: 1,
			Description: "Adds a random whole number from 0 to this to each score"},
		{Name: "k_score_bucket", Group: "Score", Type: Int, Default: 2, Min: 1,
			Description: "Scores are divided by this and rounded towards 0, so close scores tie and are picked between " +
				"at random"},
	},
	run: func(b *board.Board, v Values, rng *rand.Rand) Trace {
		return Split(b, SplitConfig{
			Obstacles:       v.Int("k_obstacles"),
			MaxBank:         v.Int("k_max_bank"),
			Symmetric:       v.Bool("is_symmetric"),
			EdgeMargin:      v.Int("k_edge_margin"),
			MaxAdjacent:     v.Int("k_max_adjacent"),
			SplitAxes:       aggregateFuncs[v.Choice("f_split_axes")],
			AdjacentPenalty: v.Float("k_adjacent_penalty"),
			Noise:           v.Int("k_noise"),
			ScoreBucket:     v.Int("k_score_bucket"),
		}, rng)
	},
}

// SplitConfig holds the parameters of Split.
type SplitConfig struct {
	Obstacles       int                     // k_obstacles
	MaxBank         int                     // k_max_bank; 0 means no limit
	Symmetric       bool                    // is_symmetric
	EdgeMargin      int                     // k_edge_margin
	MaxAdjacent     int                     // k_max_adjacent
	SplitAxes       func([]float64) float64 // f_split_axes
	AdjacentPenalty float64                 // k_adjacent_penalty
	Noise           int                     // k_noise
	ScoreBucket     int                     // k_score_bucket
}

// Split places obstacles on b greedily. Each round it looks at every
// placement: a single empty cell, or with Symmetric, an empty cell and its
// 180° rotation about the centre (or the centre on its own). A placement
// is eligible if none of its cells has fewer than EdgeMargin cells between
// it and the edge, and with its obstacles added, none touches more than
// MaxAdjacent obstacles and no bank is bigger than MaxBank (if set). A
// pair is only eligible while two or more obstacles are left to place, so
// it never places more than cfg.Obstacles, but the centre can leave an odd
// number for the pairs, in which case it stops one short.
//
// An eligible placement is scored from one of its cells (on a symmetric
// board, both score the same):
//
//	split = SplitAxes(s₁, s₂, s₃), where sᵢ is the shorter of the clear runs
//	        either way along axis i: the empty cells before an obstacle or
//	        the edge
//	raw   = split − AdjacentPenalty × (obstacles it touches) + U{0, …, Noise}
//	score = raw / ScoreBucket, rounded towards 0 like Java's integer division
//
// and the obstacles go on a random placement among those with the top
// score. It stops once it has placed cfg.Obstacles obstacles or nothing is
// eligible. The trace has a step for each placement.
func Split(b *board.Board, cfg SplitConfig, rng *rand.Rand) Trace {
	g := grid{b: b, index: b.Index()}
	trace := Trace{Metrics: splitMetrics}
	type option struct {
		cells  []int
		values []float64 // in the order of splitMetrics, chance left at 0
	}
	for placed := 0; placed < cfg.Obstacles; {
		bs := g.banks()
		var options []option
		left := cfg.Obstacles - placed
		for i, c := range b.Cells {
			cells := []int{i}
			if cfg.Symmetric {
				j, ok := g.index[board.Hex{Q: -c.Q, R: -c.R}]
				if !ok || j < i {
					continue // not symmetric, or already seen with its partner
				}
				if j != i {
					cells = append(cells, j)
				}
			}
			if len(cells) > left {
				continue
			}
			hexes := make([]board.Hex, len(cells))
			eligible := true
			for k, cell := range cells {
				hexes[k] = b.Cells[cell].Hex
				eligible = eligible && !b.Cells[cell].Obstacle && g.edgeDistance(hexes[k]) >= cfg.EdgeMargin
			}
			if !eligible {
				continue
			}
			adjacent := make([]int, len(hexes))
			for k, h := range hexes {
				for _, n := range h.Neighbours() {
					if _, obstacle := g.at(n); obstacle || slices.Contains(hexes, n) {
						adjacent[k]++
					}
				}
				eligible = eligible && adjacent[k] <= cfg.MaxAdjacent
			}
			if !eligible || (cfg.MaxBank > 0 && bs.sizeWithAll(g, hexes) > cfg.MaxBank) {
				continue
			}

			h := hexes[0]
			var splits []float64
			for _, axis := range board.Axes {
				splits = append(splits, float64(min(g.run(h, axis[0]), g.run(h, axis[1]))))
			}
			split := cfg.SplitAxes(splits)
			noise := rng.Uint64N(uint64(cfg.Noise) + 1)
			raw := split - cfg.AdjacentPenalty*float64(adjacent[0]) + float64(noise)
			score := math.Trunc(raw / float64(cfg.ScoreBucket))
			if score == 0 {
				score = 0 // not -0, which encodes as "-0"
			}
			options = append(options, option{
				cells:  cells,
				values: []float64{score, 0, raw, split, float64(adjacent[0]), float64(noise), float64(g.edgeDistance(h))},
			})
		}
		if len(options) == 0 {
			trace.Note = "no eligible placement left"
			if cfg.Symmetric && left == 1 {
				trace.Note = "only the centre could take the last obstacle, and it isn't eligible"
			}
			break
		}

		best := math.Inf(-1)
		for _, o := range options {
			best = max(best, o.values[0])
		}
		var top []int
		for k, o := range options {
			if o.values[0] == best {
				top = append(top, k)
			}
		}
		pick := options[top[rng.IntN(len(top))]]
		for _, cell := range pick.cells {
			b.Cells[cell].Obstacle = true
		}
		placed += len(pick.cells)

		step := Step{Placed: b.Cells[pick.cells[0]].Hex}
		for _, cell := range pick.cells[1:] {
			step.Also = append(step.Also, b.Cells[cell].Hex)
		}
		for _, o := range options {
			values := slices.Clone(o.values)
			if values[0] == best {
				values[1] = 1 / float64(len(top))
			}
			for _, cell := range o.cells {
				step.Candidates = append(step.Candidates, Candidate{Hex: b.Cells[cell].Hex, Values: values})
			}
		}
		trace.Steps = append(trace.Steps, step)
	}
	return trace
}

// splitMetrics are the values Split records for each cell of each eligible
// placement.
var splitMetrics = []Metric{
	{Name: "score", Description: "Score (raw score ÷ k_score_bucket, rounded towards 0)", Integer: true},
	{Name: "chance", Description: "Chance of being chosen", Percent: true},
	{Name: "raw_score", Description: "Raw score (split − penalty + noise)"},
	{Name: "split", Description: "Split (f_split_axes of each axis's shorter clear run)"},
	{Name: "adjacent", Description: "Obstacles it would touch", Integer: true},
	{Name: "noise", Description: "Noise", Integer: true},
	{Name: "edge_distance", Description: "Cells between it and the edge", Integer: true},
}
