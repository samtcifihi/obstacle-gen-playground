package algorithm

import (
	"fmt"
	"math/rand/v2"
	"net/url"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

// All lists the available algorithms, in the order a UI should offer them.
var All = []Algorithm{uniformAlgorithm, evolvedAlgorithm, tripleBetaAlgorithm, splitAlgorithm}

// Lookup returns the algorithm with the given ID.
func Lookup(id string) (Algorithm, bool) {
	for _, a := range All {
		if a.ID == id {
			return a, true
		}
	}
	return Algorithm{}, false
}

// Algorithm is a placement algorithm together with a description of its
// parameters, from which a UI can build a form.
type Algorithm struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Params      []Param `json:"params"`
	run         func(b *board.Board, v Values, rng *rand.Rand) Trace
}

// Run places obstacles on b using parameters v from a.Parse. The trace
// has a step for each obstacle placed.
func (a Algorithm) Run(b *board.Board, v Values, rng *rand.Rand) Trace {
	trace := a.run(b, v, rng)
	if trace.Steps == nil {
		trace.Steps = []Step{} // so it encodes as [], not null
	}
	return trace
}

// Parse reads a's parameters from q, using the default for any that are
// missing. A parameter that's following another (see Param.Follows) takes
// that parameter's value, whatever q says.
func (a Algorithm) Parse(q url.Values) (Values, error) {
	v := make(Values, len(a.Params))
	for _, p := range a.Params {
		if f, ok := p.following(v); ok {
			if f.Inverse {
				v[p.Name] = a.inverse(f.Param, v.Choice(f.Param))
			} else {
				v[p.Name] = v[f.Param]
			}
			continue
		}
		if !q.Has(p.Name) {
			v[p.Name] = p.Default
			continue
		}
		value, err := p.parse(q.Get(p.Name))
		if err != nil {
			return nil, err
		}
		v[p.Name] = value
	}
	return v, nil
}

// inverse returns the inverse of option value of the Choice parameter
// named param.
func (a Algorithm) inverse(param, value string) string {
	for _, p := range a.Params {
		if p.Name != param {
			continue
		}
		for _, o := range p.Options {
			if o.Value == value {
				return o.Inverse
			}
		}
	}
	panic(fmt.Sprintf("%s: parameter %s has no option %q", a.ID, param, value))
}

// Active returns the values in v of a's parameters that have any effect,
// those whose OnlyIf condition holds, in the order a lists them.
func (a Algorithm) Active(v Values) ParamValues {
	active := ParamValues{}
	for _, p := range a.Params {
		if v.holds(p.OnlyIf) {
			active = append(active, ParamValue{Name: p.Name, Value: v[p.Name]})
		}
	}
	return active
}
