package algorithm

import "github.com/samtcifihi/obstacle-gen-playground/internal/board"

// Trace records how an algorithm placed each obstacle, so a UI can step
// through the placements and show what each choice was based on.
type Trace struct {
	// Metrics describes the values recorded for each candidate cell.
	Metrics []Metric `json:"metrics"`
	// Steps has one entry per obstacle placed, in order.
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
}

// Step records the placement of one obstacle.
type Step struct {
	Placed board.Hex `json:"placed"`
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
