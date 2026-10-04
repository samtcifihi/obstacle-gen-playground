package algorithm

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

var tripleBetaAlgorithm = Algorithm{
	ID:   "triple_beta",
	Name: "Triple Beta",
	Description: "Weights every hex by three beta densities, one for each cube coordinate (q, r, s), each " +
		"scaled so the coordinate's bands across the board evenly split 0 to 1, multiplied together, and by a " +
		"distance weight: a fourth beta density over its distance to the nearest obstacle, or per axis, three more " +
		"over how far its q, r and s are from that obstacle's. Each obstacle goes on a free hex (no obstacle, " +
		"and not making a bank too big) picked with probability proportional to its weight.",
	Params: slices.Concat(
		[]Param{
			{Name: "k_obstacles", Group: "General", Type: Int, Default: 16,
				Description: "The number of obstacles to place (if possible)"},
			{Name: "k_max_bank", Group: "General", Type: Int, Default: 0,
				Description: "Maximum contiguous group of obstacles allowed (0 = no limit)"},
			{Name: "is_alpha_eq_beta", Group: "Distributions", Type: Bool, Default: true,
				Description: "Force α = β for each axis's distribution, so each is symmetric about the centre"},
			{Name: "is_axes_shared", Group: "Distributions", Type: Bool, Default: true,
				Description: "Force all 3 distributions to use axis 1's α and β"},
		},
		axisParams(1, "q", "lower left edge", "upper right edge"),
		axisParams(2, "r", "top edge", "bottom edge"),
		axisParams(3, "s", "lower right edge", "upper left edge"),
		[]Param{
			{Name: "f_obstacles_distance_mode", Group: "Distance to obstacles", Type: Choice, Default: "aggregate",
				Options: []Option{{Value: "aggregate", Label: "aggregate"}, {Value: "per_axis", Label: "per_axis"}},
				Description: "aggregate weights a hex by its hex distance to the nearest obstacle; per_axis weights it by " +
					"how far its q, r and s each are from that obstacle's"},
			{Name: "is_obstacles_alpha_eq_beta", Group: "Distance to obstacles", Type: Bool, Default: false, OnlyIf: aggregateMode,
				Description: "Force α = β for the distance distribution"},
			{Name: "k_alpha_obstacles", Group: "Distance to obstacles", Type: Float, Default: 1.0, MinExclusive: true,
				OnlyIf: aggregateMode,
				Description: "α of the beta distribution over distance to the nearest obstacle: raising it favours " +
					"being far from obstacles, spreading them out"},
			{Name: "k_beta_obstacles", Group: "Distance to obstacles", Type: Float, Default: 1.0, MinExclusive: true,
				OnlyIf: aggregateMode, Follows: []Follow{{When: "is_obstacles_alpha_eq_beta", Param: "k_alpha_obstacles"}},
				Description: "β of the beta distribution over distance to the nearest obstacle: raising it favours " +
					"being close to obstacles, clustering them"},
			{Name: "is_obstacle_axes_alpha_eq_beta", Group: "Distance to obstacles", Type: Bool, Default: false,
				OnlyIf: perAxisMode, Description: "Force α = β for each axis's distance distribution"},
			{Name: "is_obstacle_axes_shared", Group: "Distance to obstacles", Type: Bool, Default: true,
				OnlyIf: perAxisMode, Description: "Force all 3 distance distributions to use axis 1's α and β"},
		},
		obstacleAxisParams(1, "q"),
		obstacleAxisParams(2, "r"),
		obstacleAxisParams(3, "s"),
	),
	run: func(b *board.Board, v Values, rng *rand.Rand) Trace {
		return TripleBeta(b, tripleBetaConfig(v), rng)
	},
}

// The modes of f_obstacles_distance_mode, as conditions for Param.OnlyIf.
const (
	aggregateMode = "f_obstacles_distance_mode=aggregate"
	perAxisMode   = "f_obstacles_distance_mode=per_axis"
)

// axisParams returns the α and β parameters of the distribution over cube
// coordinate coord, whose lowest value is along edge from and highest
// along edge to.
func axisParams(axis int, coord, from, to string) []Param {
	alpha, beta := fmt.Sprintf("k_alpha_%d", axis), fmt.Sprintf("k_beta_%d", axis)
	group := fmt.Sprintf("Axis %d (%s): %s to %s", axis, coord, from, to)
	alphaFollows, betaFollows := tiedFollows("k_alpha_%d", "k_beta_%d", axis, "is_axes_shared", "is_alpha_eq_beta")
	return []Param{
		{Name: alpha, Group: group, Type: Float, Default: 1.0, MinExclusive: true, Follows: alphaFollows,
			Description: fmt.Sprintf("α of the beta distribution over %s: raising it favours the %s", coord, to)},
		{Name: beta, Group: group, Type: Float, Default: 1.0, MinExclusive: true, Follows: betaFollows,
			Description: fmt.Sprintf("β of the beta distribution over %s: raising it favours the %s", coord, from)},
	}
}

// obstacleAxisParams returns the α and β parameters of the distribution
// over how far a hex's cube coordinate coord is from the nearest
// obstacle's, for per-axis distance mode.
func obstacleAxisParams(axis int, coord string) []Param {
	alpha, beta := fmt.Sprintf("k_alpha_obstacles_%d", axis), fmt.Sprintf("k_beta_obstacles_%d", axis)
	group := fmt.Sprintf("Obstacle distance axis %d (%s)", axis, coord)
	alphaFollows, betaFollows := tiedFollows("k_alpha_obstacles_%d", "k_beta_obstacles_%d", axis,
		"is_obstacle_axes_shared", "is_obstacle_axes_alpha_eq_beta")
	return []Param{
		{Name: alpha, Group: group, Type: Float, Default: 1.0, MinExclusive: true, OnlyIf: perAxisMode,
			Follows: alphaFollows,
			Description: fmt.Sprintf("α of the beta distribution over %s distance, how far a hex's %s is from its "+
				"nearest obstacle's: raising it favours more separation in %s", coord, coord, coord)},
		{Name: beta, Group: group, Type: Float, Default: 1.0, MinExclusive: true, OnlyIf: perAxisMode,
			Follows: betaFollows,
			Description: fmt.Sprintf("β of the beta distribution over %s distance: raising it favours less "+
				"separation in %s", coord, coord)},
	}
}

// tiedFollows returns the Follows for the α and β parameters of axis,
// whose names are alpha and beta formatted with the axis number. While
// shared holds, axes 2 and 3 follow axis 1; while alphaEqBeta holds, β
// follows α. With both, all six follow axis 1's α.
func tiedFollows(alpha, beta string, axis int, shared, alphaEqBeta string) (alphaFollows, betaFollows []Follow) {
	if axis > 1 {
		alphaFollows = []Follow{{When: shared, Param: fmt.Sprintf(alpha, 1)}}
		betaFollows = []Follow{{When: shared, Param: fmt.Sprintf(beta, 1)}}
	}
	betaFollows = append(betaFollows, Follow{When: alphaEqBeta, Param: fmt.Sprintf(alpha, axis)})
	return alphaFollows, betaFollows
}

func tripleBetaConfig(v Values) TripleBetaConfig {
	cfg := TripleBetaConfig{
		Obstacles: v.Int("k_obstacles"),
		MaxBank:   v.Int("k_max_bank"),
		Distance: BetaShape{
			Alpha: v.Float("k_alpha_obstacles"),
			Beta:  v.Float("k_beta_obstacles"),
		},
		PerAxisDistance: v.Choice("f_obstacles_distance_mode") == "per_axis",
	}
	for i := range cfg.Axes {
		cfg.Axes[i] = BetaShape{
			Alpha: v.Float(fmt.Sprintf("k_alpha_%d", i+1)),
			Beta:  v.Float(fmt.Sprintf("k_beta_%d", i+1)),
		}
		cfg.AxisDistances[i] = BetaShape{
			Alpha: v.Float(fmt.Sprintf("k_alpha_obstacles_%d", i+1)),
			Beta:  v.Float(fmt.Sprintf("k_beta_obstacles_%d", i+1)),
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
	// PerAxisDistance weights cells by AxisDistances rather than Distance
	// (f_obstacles_distance_mode=per_axis).
	PerAxisDistance bool
	// Distance is the shape of the distribution over distance to the
	// nearest obstacle (k_alpha_obstacles, k_beta_obstacles).
	Distance BetaShape
	// AxisDistances holds the shape of the distribution over how far each
	// cube coordinate is from the nearest obstacle's
	// (k_alpha_obstacles_1 and so on).
	AxisDistances [3]BetaShape
}

// BetaShape holds the parameters of a beta distribution.
type BetaShape struct {
	Alpha, Beta float64
}

// TripleBeta places obstacles on b one at a time. Each cell's weight is its
// position weight (see hexWeights), which is fixed, times its distance
// weight, which is recalculated each round (see hexDistance and
// axisDistance). Each obstacle goes on a free cell, one without an
// obstacle where one wouldn't make a bank bigger than MaxBank, picked with
// probability proportional to its weight.
//
// It stops once it has placed cfg.Obstacles obstacles or no free cell has
// any weight. The trace has a step for each obstacle placed, recording
// each cell that could have taken it with its chance, weights and
// distances.
func TripleBeta(b *board.Board, cfg TripleBetaConfig, rng *rand.Rand) Trace {
	g := grid{b: b, index: b.Index()}
	positions := hexWeights(b, cfg.Axes)
	var distance distanceTerm
	if cfg.PerAxisDistance {
		distance = newAxisDistance(b, cfg.AxisDistances)
	} else {
		distance = newHexDistance(b, cfg.Distance)
	}
	trace := Trace{Metrics: slices.Concat(tripleBetaMetrics, distance.metrics())}
	if slices.ContainsFunc(positions, notFinite) || !distance.finite() {
		// Scaled coordinates and distances are never exactly 0 or 1, so only
		// extreme shapes can get here, by overflowing.
		trace.Note = "some weights are infinite or undefined; α or β is too extreme"
		return trace
	}

	type option struct {
		cell   int
		weight float64
		values []float64 // what distance records
	}
	for len(trace.Steps) < cfg.Obstacles {
		var obstacles []board.Hex
		for _, c := range b.Cells {
			if c.Obstacle {
				obstacles = append(obstacles, c.Hex)
			}
		}
		distance.round(obstacles)

		var options []option
		var weights []float64
		total := 0.0
		free := g.freeCells(cfg.MaxBank)
		for _, i := range free {
			dw, values := distance.weigh(b.Cells[i].Hex)
			if w := positions[i] * dw; w > 0 {
				options = append(options, option{cell: i, weight: w, values: values})
				weights = append(weights, w)
				total += w
			}
		}
		if len(options) == 0 {
			trace.Note = "no free cell left"
			if len(free) > 0 {
				trace.Note = "no free cell has any weight"
			}
			break
		}

		pick := options[weightedIndex(rng, weights, total)].cell
		b.Cells[pick].Obstacle = true

		step := Step{Placed: b.Cells[pick].Hex, Candidates: make([]Candidate, len(options))}
		for j, chance := range weightedChances(weights, total) {
			o := options[j]
			step.Candidates[j] = Candidate{
				Hex:    b.Cells[o.cell].Hex,
				Values: append([]float64{chance, o.weight, positions[o.cell]}, o.values...),
			}
		}
		trace.Steps = append(trace.Steps, step)
	}
	return trace
}

// tripleBetaMetrics are the values TripleBeta records for each cell that
// could take an obstacle, followed by its distance term's.
var tripleBetaMetrics = []Metric{
	{Name: "chance", Description: "Chance of getting this obstacle", Percent: true},
	{Name: "weight", Description: "Weight (position weight × distance weight)"},
	{Name: "position", Description: "Position weight (product of the three axis densities)"},
}

// distanceTerm works out the distance weights of TripleBeta's cells.
type distanceTerm interface {
	// metrics describes the values weigh returns.
	metrics() []Metric
	// finite reports whether every weight the term can give is finite.
	finite() bool
	// round starts a round, with obstacles on the board.
	round(obstacles []board.Hex)
	// weigh returns h's distance weight, and the values to record for it.
	weigh(h board.Hex) (float64, []float64)
}

// hexDistance is the aggregate distance term: the density of one beta
// distribution at a cell's distance to the nearest obstacle, measured up
// to the board's span, so no obstacle is ever out of range (see
// distanceWeight). With no obstacles, every cell is at the span.
type hexDistance struct {
	span      int
	weights   []float64 // by distance
	obstacles []board.Hex
}

func newHexDistance(b *board.Board, shape BetaShape) *hexDistance {
	t := &hexDistance{span: b.Span()}
	t.weights = make([]float64, t.span+1)
	for d := range t.weights {
		t.weights[d] = distanceWeight(d, t.span, shape)
	}
	return t
}

var hexDistanceMetrics = []Metric{
	{Name: "distance", Description: "Hex distance to the nearest obstacle (1 if adjacent)", Integer: true},
	{Name: "distance_weight", Description: "Distance weight (density at the scaled distance)"},
}

func (t *hexDistance) metrics() []Metric           { return hexDistanceMetrics }
func (t *hexDistance) finite() bool                { return !slices.ContainsFunc(t.weights, notFinite) }
func (t *hexDistance) round(obstacles []board.Hex) { t.obstacles = obstacles }

func (t *hexDistance) weigh(h board.Hex) (float64, []float64) {
	d := nearestObstacle(h, t.obstacles, t.span)
	// The weights count the cells between a hex and the nearest obstacle,
	// but the trace shows the plain hex distance.
	return t.weights[d], []float64{float64(d + 1), t.weights[d]}
}

// axisDistance is the per-axis distance term. It measures a cell against
// its nearest obstacle by hex distance, and splits the displacement
// between them into its three cube coordinates: the q distance is
// |q - q'|, so 0 if the obstacle shares the cell's q band, and the same
// goes for r and s. On a board of radius R each runs from 0 to 2R, and
// scales into (0, 1) as the middle of its band, (d + ½)/(2R + 1). The
// obstacle's weight is the product of the three axes' densities there.
//
// When several obstacles are equally near, the distance weight is the mean
// of theirs, summed in a fixed order so it doesn't depend on the order of
// the obstacles, even in rounding. The values recorded are also means
// over them.
//
// With no obstacles, the term has no effect: every weight is 1.
type axisDistance struct {
	radius    int
	weights   [3][]float64 // by axis, then distance
	obstacles []board.Hex
	nearest   [][3]int // scratch space for weigh
}

func newAxisDistance(b *board.Board, shapes [3]BetaShape) *axisDistance {
	t := &axisDistance{radius: b.Radius()}
	for axis, shape := range shapes {
		t.weights[axis] = make([]float64, 2*t.radius+1)
		for d := range t.weights[axis] {
			t.weights[axis][d] = distanceWeight(d, 2*t.radius, shape)
		}
	}
	return t
}

var axisDistanceMetrics = []Metric{
	{Name: "distance_weight", Description: "Distance weight (product of the three axes' densities, for the " +
		"nearest obstacle, or the mean over those equally near)"},
	{Name: "q_distance", Description: "q distance: how far its q is from its nearest obstacle's (the mean, if several are as near)"},
	{Name: "q_distance_weight", Description: "q distance weight (density at the scaled q distance, or the mean)"},
	{Name: "r_distance", Description: "r distance: how far its r is from its nearest obstacle's (the mean, if several are as near)"},
	{Name: "r_distance_weight", Description: "r distance weight (density at the scaled r distance, or the mean)"},
	{Name: "s_distance", Description: "s distance: how far its s is from its nearest obstacle's (the mean, if several are as near)"},
	{Name: "s_distance_weight", Description: "s distance weight (density at the scaled s distance, or the mean)"},
}

func (t *axisDistance) metrics() []Metric { return axisDistanceMetrics }

func (t *axisDistance) finite() bool {
	for _, weights := range t.weights {
		if slices.ContainsFunc(weights, notFinite) {
			return false
		}
	}
	return true
}

func (t *axisDistance) round(obstacles []board.Hex) { t.obstacles = obstacles }

func (t *axisDistance) weigh(h board.Hex) (float64, []float64) {
	values := make([]float64, 7)
	if len(t.obstacles) == 0 {
		// Every coordinate is as far as can be, but the term is left out,
		// as the same factor on every cell would change nothing, unless it
		// rounded down to 0.
		values[0] = 1
		for axis := range 3 {
			values[1+2*axis], values[2+2*axis] = float64(2*t.radius), 1
		}
		return 1, values
	}

	// The q, r and s distances to each of the nearest obstacles.
	t.nearest = t.nearest[:0]
	nearest := math.MaxInt
	for _, o := range t.obstacles {
		d := h.DistanceTo(o)
		if d > nearest {
			continue
		}
		if d < nearest {
			nearest, t.nearest = d, t.nearest[:0]
		}
		c, co := h.Cube(), o.Cube()
		t.nearest = append(t.nearest, [3]int{abs(c[0] - co[0]), abs(c[1] - co[1]), abs(c[2] - co[2])})
	}
	slices.SortFunc(t.nearest, func(a, b [3]int) int { return slices.Compare(a[:], b[:]) })

	weight := 0.0
	for _, ds := range t.nearest {
		w := 1.0
		for axis, d := range ds {
			w *= t.weights[axis][d]
			values[1+2*axis] += float64(d)
			values[2+2*axis] += t.weights[axis][d]
		}
		weight += w
	}
	n := float64(len(t.nearest))
	weight /= n
	values[0] = weight
	for i := 1; i < len(values); i++ {
		values[i] /= n
	}
	return weight, values
}

// nearestObstacle returns the number of cells between h and the nearest of
// obstacles, so 0 if one is adjacent, capped at limit. With no obstacles
// it's limit.
func nearestObstacle(h board.Hex, obstacles []board.Hex, limit int) int {
	d := limit
	for _, o := range obstacles {
		d = min(d, h.DistanceTo(o)-1)
	}
	return d
}

// distanceWeight returns the density of shape at distance d, which runs
// from 0 to limit, scaled into (0, 1) as the middle of its band,
// (d + ½)/(limit + 1), so the density is never infinite.
func distanceWeight(d, limit int, shape BetaShape) float64 {
	return betaPDF((float64(d)+0.5)/float64(limit+1), shape.Alpha, shape.Beta)
}

// hexWeights returns the weight of each cell of b: the product of the beta
// densities in shapes at its cube coordinates q, r and s, each scaled into
// (0, 1). The 2R + 1 values a coordinate takes on a board of radius R split
// [0, 1] into equal bands, and a coordinate x scales to the middle of its
// band, (x + R + ½)/(2R + 1), so it's never exactly 0 or 1, where a beta
// density can be 0 or infinite.
func hexWeights(b *board.Board, shapes [3]BetaShape) []float64 {
	radius := b.Radius()
	scale := func(x int) float64 {
		return (float64(x+radius) + 0.5) / float64(2*radius+1)
	}
	weights := make([]float64, len(b.Cells))
	for i, c := range b.Cells {
		w := 1.0
		for j, x := range c.Cube() {
			w *= betaPDF(scale(x), shapes[j].Alpha, shapes[j].Beta)
		}
		weights[i] = w
	}
	return weights
}
