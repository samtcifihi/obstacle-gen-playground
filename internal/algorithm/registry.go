package algorithm

import (
	"fmt"
	"math"
	"math/rand/v2"
	"net/url"
	"strconv"
	"strings"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

// All lists the available algorithms, in the order a UI should offer them.
var All = []Algorithm{uniformAlgorithm, evolvedAlgorithm, tripleBetaAlgorithm}

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

// ParamType is the kind of value a parameter takes.
type ParamType string

const (
	Int    ParamType = "int"
	Float  ParamType = "float"
	Bool   ParamType = "bool"
	Choice ParamType = "choice"
)

// Param describes one parameter of an algorithm.
type Param struct {
	Name        string    `json:"name"`
	Group       string    `json:"group,omitempty"`
	Description string    `json:"description"`
	Type        ParamType `json:"type"`
	// Default must be an int, float64, bool or string to match Type.
	Default any `json:"default"`
	// Min is the lowest value an Int or Float parameter may take. With
	// MinExclusive, values must be strictly greater.
	Min          float64 `json:"min"`
	MinExclusive bool    `json:"minExclusive,omitempty"`
	// Options are the values a Choice parameter may take.
	Options []Option `json:"options,omitempty"`
	// OnlyIf names a Bool parameter that must be true for this one to have
	// any effect, or false if the name is prefixed with "!".
	OnlyIf string `json:"onlyIf,omitempty"`
	// Follows lists other parameters this one takes its value from. The
	// first whose condition holds applies, and while one does, this
	// parameter can't be set directly.
	Follows []Follow `json:"follows,omitempty"`
}

// Follow makes a parameter take another's value while a condition holds.
type Follow struct {
	// When names a Bool parameter that must be true, or false if prefixed
	// with "!". Empty means always.
	When string `json:"when,omitempty"`
	// Param names the parameter to take the value of. It and When's
	// parameter must come earlier in the list.
	Param string `json:"param"`
	// Inverse takes the Inverse of Param's chosen option instead.
	Inverse bool `json:"inverse,omitempty"`
}

// following returns the Follow that applies to p given the values parsed
// so far, if any.
func (p Param) following(v Values) (Follow, bool) {
	for _, f := range p.Follows {
		if f.When == "" || v.Bool(strings.TrimPrefix(f.When, "!")) != strings.HasPrefix(f.When, "!") {
			return f, true
		}
	}
	return Follow{}, false
}

// Option is one value of a Choice parameter.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
	// Inverse is the value of the option that undoes this one, for
	// parameters that follow this one with Follow.Inverse.
	Inverse string `json:"inverse,omitempty"`
}

func (p Param) parse(s string) (any, error) {
	switch p.Type {
	case Int:
		n, err := strconv.Atoi(s)
		if err != nil || !p.inRange(float64(n)) {
			return nil, fmt.Errorf("%s must be a whole number %s", p.Name, p.rangeText())
		}
		return n, nil
	case Float:
		x, err := strconv.ParseFloat(s, 64)
		if err != nil || math.IsInf(x, 0) || !p.inRange(x) {
			return nil, fmt.Errorf("%s must be a number %s", p.Name, p.rangeText())
		}
		return x, nil
	case Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return nil, fmt.Errorf("%s must be true or false", p.Name)
		}
		return b, nil
	case Choice:
		values := make([]string, len(p.Options))
		for i, o := range p.Options {
			if o.Value == s {
				return s, nil
			}
			values[i] = o.Value
		}
		return nil, fmt.Errorf("%s must be one of: %s", p.Name, strings.Join(values, ", "))
	}
	panic(fmt.Sprintf("parameter %s has unknown type %q", p.Name, p.Type))
}

func (p Param) inRange(x float64) bool {
	if p.MinExclusive {
		return x > p.Min
	}
	return x >= p.Min
}

func (p Param) rangeText() string {
	if p.MinExclusive {
		return fmt.Sprintf("greater than %g", p.Min)
	}
	return fmt.Sprintf("of at least %g", p.Min)
}

// Values holds parsed parameter values by name.
type Values map[string]any

func (v Values) Int(name string) int       { return v[name].(int) }
func (v Values) Float(name string) float64 { return v[name].(float64) }
func (v Values) Bool(name string) bool     { return v[name].(bool) }
func (v Values) Choice(name string) string { return v[name].(string) }
