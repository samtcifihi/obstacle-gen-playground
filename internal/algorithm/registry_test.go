package algorithm

import (
	"fmt"
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
			if p.OnlyIf == "" {
				continue
			}
			if on, ok := params[strings.TrimPrefix(p.OnlyIf, "!")]; !ok || on.Type != Bool {
				t.Errorf("%s: %s is only used if %q, which isn't a bool parameter", a.ID, p.Name, p.OnlyIf)
			}
		}
	}
}

func TestOptionsHaveFuncs(t *testing.T) {
	for _, o := range scalarOptions {
		if scalarFuncs[o.Value] == nil {
			t.Errorf("scalar option %q has no function", o.Value)
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
		"f_edge_a'":              {"identity"},
		"not_a_parameter":        {"ignored"},
		"is_obstacles_flattened": {"0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if v.Int("k_obstacles") != 3 || v.Float("k_beta") != 0.5 || v.Bool("is_weighted") ||
		v.Choice("f_edge_a'") != "identity" || v.Bool("is_obstacles_flattened") {
		t.Errorf("parsed values don't match the query: %v", v)
	}
	if v.Int("k_max_bank") != 0 || v.Float("k_stretch") != 8 {
		t.Errorf("missing parameters didn't get their defaults: %v", v)
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
		{"f_obstacles_b": {"median"}},
	} {
		if _, err := evolvedAlgorithm.Parse(q); err == nil {
			t.Errorf("Parse(%v) succeeded, want an error", q)
		}
	}
}
