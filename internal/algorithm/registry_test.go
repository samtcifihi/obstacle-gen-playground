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
		// Conditions and followed parameters must come earlier in the list,
		// so Parse has their values by the time it needs them.
		earlier := make(map[string]Param)
		for _, p := range a.Params {
			if _, dup := earlier[p.Name]; dup {
				t.Errorf("%s: duplicate parameter %s", a.ID, p.Name)
			}

			got, err := p.parse(fmt.Sprint(p.Default))
			if err != nil {
				t.Errorf("%s: default for %s doesn't parse: %v", a.ID, p.Name, err)
			} else if got != p.Default {
				t.Errorf("%s: default for %s is %#v, but parses as %#v", a.ID, p.Name, p.Default, got)
			}

			if name, value, isChoice := strings.Cut(p.OnlyIf, "="); isChoice {
				if on, ok := earlier[name]; !ok || on.Type != Choice {
					t.Errorf("%s: %s is only used if %q, which isn't an earlier choice parameter", a.ID, p.Name, p.OnlyIf)
				} else if _, err := on.parse(value); err != nil {
					t.Errorf("%s: %s is only used if %q, which isn't an option of %s", a.ID, p.Name, p.OnlyIf, name)
				}
			} else if p.OnlyIf != "" {
				if on, ok := earlier[strings.TrimPrefix(p.OnlyIf, "!")]; !ok || on.Type != Bool {
					t.Errorf("%s: %s is only used if %q, which isn't an earlier bool parameter", a.ID, p.Name, p.OnlyIf)
				}
			}
			for _, f := range p.Follows {
				if f.When != "" {
					if on, ok := earlier[strings.TrimPrefix(f.When, "!")]; !ok || on.Type != Bool {
						t.Errorf("%s: %s follows when %q, which isn't an earlier bool parameter", a.ID, p.Name, f.When)
					}
				}
				src, ok := earlier[f.Param]
				switch {
				case !ok:
					t.Errorf("%s: %s follows %q, which isn't an earlier parameter", a.ID, p.Name, f.Param)
				case f.Inverse:
					if src.Type != Choice {
						t.Errorf("%s: %s follows the inverse of %s, which isn't a choice parameter", a.ID, p.Name, src.Name)
						continue
					}
					if want := a.inverse(src.Name, src.Default.(string)); f.When == "" && p.Default != want {
						t.Errorf("%s: default for %s is %v, want %s, the inverse of %s's default", a.ID, p.Name, p.Default, want, src.Name)
					}
					for _, o := range src.Options {
						if _, err := p.parse(o.Inverse); err != nil {
							t.Errorf("%s: %s option %q has inverse %q, which isn't an option of %s", a.ID, src.Name, o.Value, o.Inverse, p.Name)
						}
					}
				case src.Type != p.Type:
					t.Errorf("%s: %s follows %s, which has a different type", a.ID, p.Name, src.Name)
				}
			}
			earlier[p.Name] = p
		}
	}
}

func TestParseFollows(t *testing.T) {
	a, _ := Lookup("triple_beta")
	parse := func(q url.Values) Values {
		t.Helper()
		v, err := a.Parse(q)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	shapes := func(v Values) [6]float64 {
		var s [6]float64
		for i := range 3 {
			s[2*i] = v.Float(fmt.Sprintf("k_alpha_%d", i+1))
			s[2*i+1] = v.Float(fmt.Sprintf("k_beta_%d", i+1))
		}
		return s
	}
	q := url.Values{
		"k_alpha_1": {"1"}, "k_beta_1": {"2"},
		"k_alpha_2": {"3"}, "k_beta_2": {"4"},
		"k_alpha_3": {"5"}, "k_beta_3": {"6"},
	}
	for _, tt := range []struct {
		alphaEqBeta, shared string
		want                [6]float64
	}{
		{"false", "false", [6]float64{1, 2, 3, 4, 5, 6}},
		{"true", "false", [6]float64{1, 1, 3, 3, 5, 5}},
		{"false", "true", [6]float64{1, 2, 1, 2, 1, 2}},
		{"true", "true", [6]float64{1, 1, 1, 1, 1, 1}},
	} {
		q.Set("is_alpha_eq_beta", tt.alphaEqBeta)
		q.Set("is_axes_shared", tt.shared)
		if got := shapes(parse(q)); got != tt.want {
			t.Errorf("is_alpha_eq_beta=%s, is_axes_shared=%s: α, β = %v, want %v", tt.alphaEqBeta, tt.shared, got, tt.want)
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

func TestMaxAdjacentRange(t *testing.T) {
	for _, tt := range []struct {
		value string
		ok    bool
	}{{"0", true}, {"1", true}, {"6", true}, {"7", false}, {"-1", false}, {"3.5", false}, {"7/2", false}} {
		if _, err := splitAlgorithm.Parse(url.Values{"k_max_adjacent": {tt.value}}); (err == nil) != tt.ok {
			t.Errorf("k_max_adjacent=%s: error %v, want ok %v", tt.value, err, tt.ok)
		}
	}
	_, err := splitAlgorithm.Parse(url.Values{"k_max_adjacent": {"7"}})
	if want := "k_max_adjacent must be a whole number from 0 to 6"; err == nil || err.Error() != want {
		t.Errorf("error for 7 is %v, want %q", err, want)
	}
}
