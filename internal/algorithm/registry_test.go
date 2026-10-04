package algorithm

import (
	"fmt"
	"math"
	"net/url"
	"strings"
	"testing"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

func TestDefaultsAreValid(t *testing.T) {
	for _, a := range All {
		params := make(map[string]Param)
		for _, p := range a.Params {
			if _, dup := params[p.Name]; dup {
				t.Errorf("%s: duplicate parameter %s", a.ID, p.Name)
			}
			params[p.Name] = p

			got, err := p.parse(fmt.Sprint(p.Default))
			if err != nil {
				t.Errorf("%s: default for %s doesn't parse: %v", a.ID, p.Name, err)
			} else if got != p.Default {
				t.Errorf("%s: default for %s is %#v, but parses as %#v", a.ID, p.Name, p.Default, got)
			}
		}
		for _, p := range a.Params {
			if p.OnlyIf != "" {
				if on, ok := params[strings.TrimPrefix(p.OnlyIf, "!")]; !ok || on.Type != Bool {
					t.Errorf("%s: %s is only used if %q, which isn't a bool parameter", a.ID, p.Name, p.OnlyIf)
				}
			}
			if p.InverseOf != "" {
				src, ok := params[p.InverseOf]
				if !ok || src.Type != Choice {
					t.Errorf("%s: %s is the inverse of %q, which isn't a choice parameter", a.ID, p.Name, p.InverseOf)
					continue
				}
				if want := a.inverse(src.Name, src.Default.(string)); p.Default != want {
					t.Errorf("%s: default for %s is %v, want %s, the inverse of %s's default", a.ID, p.Name, p.Default, want, src.Name)
				}
				for _, o := range src.Options {
					if _, err := p.parse(o.Inverse); err != nil {
						t.Errorf("%s: %s option %q has inverse %q, which isn't an option of %s", a.ID, src.Name, o.Value, o.Inverse, p.Name)
					}
				}
			}
		}
	}
}

func TestOptionsHaveFuncs(t *testing.T) {
	for _, o := range scalarOptions {
		f, inv := scalarFuncs[o.Value], scalarFuncs[o.Inverse]
		if f == nil || inv == nil {
			t.Errorf("scalar option %q or its inverse %q has no function", o.Value, o.Inverse)
			continue
		}
		for _, x := range []float64{0, 0.5, 1, 2.5, 8} {
			if got := inv(f(x)); math.Abs(got-x) > 1e-9 {
				t.Errorf("%s then %s maps %v to %v", o.Value, o.Inverse, x, got)
			}
		}
	}
	for _, o := range aggregateOptions {
		if aggregateFuncs[o.Value] == nil {
			t.Errorf("aggregate option %q has no function", o.Value)
		}
	}
}

func TestRunWithDefaults(t *testing.T) {
	for _, a := range All {
		v, err := a.Parse(url.Values{})
		if err != nil {
			t.Fatalf("%s: %v", a.ID, err)
		}
		b := board.NewHexagon(5)
		if trace := a.Run(b, v, newRNG(1)); len(trace.Steps) == 0 {
			t.Errorf("%s placed no obstacles with its defaults", a.ID)
		}
	}
}

func TestParse(t *testing.T) {
	v, err := evolvedAlgorithm.Parse(url.Values{
		"k_obstacles":            {"3"},
		"k_beta":                 {"0.5"},
		"is_weighted":            {"false"},
		"f_edge_a":               {"ln"},
		"f_edge_a'":              {"identity"},
		"not_a_parameter":        {"ignored"},
		"is_obstacles_flattened": {"0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if v.Int("k_obstacles") != 3 || v.Float("k_beta") != 0.5 || v.Bool("is_weighted") ||
		v.Choice("f_edge_a") != "ln" || v.Bool("is_obstacles_flattened") {
		t.Errorf("parsed values don't match the query: %v", v)
	}
	if v.Int("k_max_bank") != 0 || v.Float("k_stretch") != 8 {
		t.Errorf("missing parameters didn't get their defaults: %v", v)
	}
	// f_*_a' always mirrors f_*_a, whatever the query says.
	if got := v.Choice("f_edge_a'"); got != "exp" {
		t.Errorf("f_edge_a' = %q, want exp, the inverse of f_edge_a=ln", got)
	}
	if got := v.Choice("f_obstacles_a'"); got != "square" {
		t.Errorf("f_obstacles_a' = %q, want square, the inverse of the default f_obstacles_a=sqrt", got)
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	for _, q := range []url.Values{
		{"k_obstacles": {"-1"}},
		{"k_obstacles": {"1.5"}},
		{"k_obstacles": {""}},
		{"k_max_bank": {"x"}},
		{"k_beta": {"0"}},
		{"k_beta": {"-2"}},
		{"k_beta": {"NaN"}},
		{"k_beta_coef": {"-0.1"}},
		{"k_beta_coef": {"Inf"}},
		{"k_stretch": {"0"}},
		{"is_weighted": {"maybe"}},
		{"f_obstacles_b": {"mode"}},
		{"f_obstacles_a": {"cube"}},
	} {
		if _, err := evolvedAlgorithm.Parse(q); err == nil {
			t.Errorf("Parse(%v) succeeded, want an error", q)
		}
	}
}
