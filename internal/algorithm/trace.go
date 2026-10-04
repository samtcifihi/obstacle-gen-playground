package algorithm

import "github.com/samtcifihi/obstacle-gen-playground/internal/board"

// Trace records how an algorithm placed each obstacle, so a UI can step
// through the placements and show what each choice was based on.
type Trace struct {
	// Metrics describes the values recorded for each candidate cell.
	Metrics []Metric `json:"metrics"`
	// Steps has one entry per choice of where to place obstacles, in order.
	// Most algorithms place one obstacle per step.
	Steps []Step `json:"steps"`
	// Note says why the algorithm stopped early, if it did.
	Note string `json:"note,omitempty"`
}

// Metric describes a value recorded for candidate cells.
type Metric struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// Percent marks a value between 0 and 1 that reads best as a percentage.
	Percent bool `json:"percent,omitempty"`
	// Integer marks a value that's always a whole number.
	Integer bool `json:"integer,omitempty"`
}

// Placed returns the number of obstacles placed.
func (t Trace) Placed() int {
	n := 0
	for _, s := range t.Steps {
		n += 1 + len(s.Also)
	}
	return n
}

// Step records the placement of one obstacle, or a few at once.
type Step struct {
	Placed board.Hex `json:"placed"`
	// Also lists any other hexes that got obstacles in the same step, such
	// as a symmetric partner.
	Also []board.Hex `json:"also,omitempty"`
	// Candidates are the cells the obstacle could have been placed on.
	Candidates []Candidate `json:"candidates"`
}

// Candidate is a cell that could have been chosen, with a value for each of
// the trace's metrics, in the same order.
type Candidate struct {
	board.Hex
	Values []float64 `json:"values"`
}

var chanceMetric = Metric{Name: "chance", Description: "Chance of being chosen", Percent: true}

// metricList describes the values an algorithm records for each candidate,
// a T, pairing each metric with how to read its value from one, so the
// metrics and the values can't get out of step.
type metricList[T any] []metric[T]

type metric[T any] struct {
	Metric
	value func(T) float64
}

// metrics returns the descriptions of the metrics, for Trace.Metrics.
func (ms metricList[T]) metrics() []Metric {
	described := make([]Metric, len(ms))
	for i, m := range ms {
		described[i] = m.Metric
	}
	return described
}

// values returns x's value for each metric, for Candidate.Values.
func (ms metricList[T]) values(x T) []float64 {
	values := make([]float64, len(ms))
	for i, m := range ms {
		values[i] = m.value(x)
	}
	return values
}
