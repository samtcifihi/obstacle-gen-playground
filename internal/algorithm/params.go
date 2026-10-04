package algorithm

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

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
	// Max, if set, is the highest value an Int or Float parameter may take.
	Max *float64 `json:"max,omitempty"`
	// Options are the values a Choice parameter may take.
	Options []Option `json:"options,omitempty"`
	// OnlyIf is a condition for this parameter to have any effect: the name
	// of a Bool parameter that must be true, or false if prefixed with "!",
	// or name=value for a Choice parameter that must have that value. A
	// choice picks between sets of parameters, so a UI should hide the ones
	// it doesn't pick, but only grey out those a bool switches off.
	OnlyIf string `json:"onlyIf,omitempty"`
	// Follows lists other parameters this one takes its value from. The
	// first whose condition holds applies, and while one does, this
	// parameter can't be set directly.
	Follows []Follow `json:"follows,omitempty"`
}

// Follow makes a parameter take another's value while a condition holds.
type Follow struct {
	// When is a condition, written like OnlyIf, that must hold. Empty means
	// always.
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
		if v.holds(f.When) {
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
		x, err := parseReal(s)
		if err != nil || !p.inRange(x) {
			return nil, fmt.Errorf("%s must be a number, like 0.5 or 1/2, %s", p.Name, p.rangeText())
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

// parseReal parses a finite real number written as a decimal, like 0.5,
// or as a fraction of two, like 1/2.
func parseReal(s string) (float64, error) {
	parse := func(s string) (float64, error) {
		x, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err == nil && (math.IsInf(x, 0) || math.IsNaN(x)) {
			err = fmt.Errorf("%q isn't finite", s)
		}
		return x, err
	}
	num, den, isFraction := strings.Cut(s, "/")
	x, err := parse(num)
	if err != nil || !isFraction {
		return x, err
	}
	d, err := parse(den)
	if err != nil {
		return 0, err
	}
	if d == 0 {
		return 0, fmt.Errorf("%q divides by 0", s)
	}
	if x /= d; math.IsInf(x, 0) {
		return 0, fmt.Errorf("%q is too big", s)
	}
	return x, nil
}

func (p Param) inRange(x float64) bool {
	if p.Max != nil && x > *p.Max {
		return false
	}
	if p.MinExclusive {
		return x > p.Min
	}
	return x >= p.Min
}

func (p Param) rangeText() string {
	switch {
	case p.Max != nil && !p.MinExclusive:
		return fmt.Sprintf("from %g to %g", p.Min, *p.Max)
	case p.Max != nil:
		return fmt.Sprintf("greater than %g and at most %g", p.Min, *p.Max)
	case p.MinExclusive:
		return fmt.Sprintf("greater than %g", p.Min)
	}
	return fmt.Sprintf("of at least %g", p.Min)
}
