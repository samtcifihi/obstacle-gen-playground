package algorithm

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"slices"
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

func TestParseReal(t *testing.T) {
	for _, tt := range []struct {
		s    string
		want float64
	}{
		{"0.5", 0.5},
		{"1/2", 0.5},
		{" 3 / 4 ", 0.75},
		{"-1/2", -0.5},
		{"1/-4", -0.25},
		{"2.5/0.5", 5},
		{"1e-1/2", 0.05},
		{"7", 7},
		{"2/3", 2.0 / 3},
	} {
		if got, err := parseReal(tt.s); err != nil || math.Abs(got-tt.want) > 1e-15 {
			t.Errorf("parseReal(%q) = %v, %v; want %v", tt.s, got, err, tt.want)
		}
	}
	for _, s := range []string{"", "/", "1/", "/2", "1/0", "0/0", "1/2/3", "a/2", "1/b", "Inf", "1/Inf", "NaN/2", "1e308/1e-308", "½"} {
		if got, err := parseReal(s); err == nil {
			t.Errorf("parseReal(%q) = %v, want an error", s, got)
		}
	}
}

func TestParseFractionsRespectLimits(t *testing.T) {
	for _, tt := range []struct {
		alg   Algorithm
		param string
		value string
		ok    bool
	}{
		{evolvedAlgorithm, "k_beta", "1/2", true},
		{evolvedAlgorithm, "k_beta", "0/2", false}, // must be greater than 0
		{evolvedAlgorithm, "k_beta", "-1/2", false},
		{evolvedAlgorithm, "k_beta_coef", "0/2", true}, // may be 0
		{evolvedAlgorithm, "k_beta_coef", "-1/3", false},
		{evolvedAlgorithm, "k_stretch", "17/2", true},
		{tripleBetaAlgorithm, "k_alpha_1", "2/3", true},
		{tripleBetaAlgorithm, "k_alpha_obstacles", "1/1000", true},
		{splitAlgorithm, "k_adjacent_penalty", "3/2", true},
		{splitAlgorithm, "k_adjacent_penalty", "-3/2", false},
		{splitAlgorithm, "k_obstacles", "1/2", false}, // whole numbers stay whole
	} {
		v, err := tt.alg.Parse(url.Values{tt.param: {tt.value}})
		if (err == nil) != tt.ok {
			t.Errorf("%s=%s: error %v, want ok %v", tt.param, tt.value, err, tt.ok)
		}
		if err == nil {
			want, _ := parseReal(tt.value)
			if got := v.Float(tt.param); got != want {
				t.Errorf("%s=%s parsed as %v, want %v", tt.param, tt.value, got, want)
			}
		}
	}
	// 1/2 and 0.5 are the same.
	half, _ := evolvedAlgorithm.Parse(url.Values{"k_beta": {"1/2"}})
	point5, _ := evolvedAlgorithm.Parse(url.Values{"k_beta": {"0.5"}})
	if half.Float("k_beta") != point5.Float("k_beta") {
		t.Errorf("1/2 parsed as %v, 0.5 as %v", half.Float("k_beta"), point5.Float("k_beta"))
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

func TestActive(t *testing.T) {
	names := func(alg Algorithm, q url.Values) []string {
		t.Helper()
		v, err := alg.Parse(q)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, p := range alg.Active(v) {
			if p.Value != v[p.Name] {
				t.Errorf("%s: active %s = %v, want %v", alg.ID, p.Name, p.Value, v[p.Name])
			}
			names = append(names, p.Name)
		}
		// In the algorithm's order.
		var order []string
		for _, p := range alg.Params {
			if slices.Contains(names, p.Name) {
				order = append(order, p.Name)
			}
		}
		if !slices.Equal(names, order) {
			t.Errorf("%s: active parameters %v, want them in the order %v", alg.ID, names, order)
		}
		return names
	}
	for _, tt := range []struct {
		alg        Algorithm
		q          url.Values
		has, hasnt []string
	}{
		{tripleBetaAlgorithm, url.Values{},
			[]string{"k_obstacles", "k_beta_3", "f_obstacles_distance_mode", "is_obstacles_alpha_eq_beta", "k_beta_obstacles"},
			[]string{"is_obstacle_axes_shared", "k_alpha_obstacles_1", "k_beta_obstacles_3"}},
		{tripleBetaAlgorithm, url.Values{"f_obstacles_distance_mode": {"per_axis"}},
			[]string{"f_obstacles_distance_mode", "is_obstacle_axes_shared", "k_alpha_obstacles_1", "k_beta_obstacles_3"},
			[]string{"is_obstacles_alpha_eq_beta", "k_alpha_obstacles", "k_beta_obstacles"}},
		{evolvedAlgorithm, url.Values{"is_obstacles_flattened": {"true"}, "is_edge_flattened": {"false"}, "is_stretching": {"false"}},
			[]string{"f_obstacles_a'", "f_edge_dir", "is_stretching"},
			[]string{"f_obstacles_dir", "k_stretch"}},
	} {
		got := names(tt.alg, tt.q)
		for _, name := range tt.has {
			if !slices.Contains(got, name) {
				t.Errorf("%s %v: active parameters %v, want %s among them", tt.alg.ID, tt.q, got, name)
			}
		}
		for _, name := range tt.hasnt {
			if slices.Contains(got, name) {
				t.Errorf("%s %v: active parameters %v, don't want %s", tt.alg.ID, tt.q, got, name)
			}
		}
	}
	// Parameters following another have its value.
	v, _ := tripleBetaAlgorithm.Parse(url.Values{"k_alpha_1": {"3"}})
	for _, p := range tripleBetaAlgorithm.Active(v) {
		if strings.HasPrefix(p.Name, "k_alpha_") || strings.HasPrefix(p.Name, "k_beta_") {
			if !strings.Contains(p.Name, "obstacles") && p.Value != 3.0 {
				t.Errorf("%s = %v, want 3, following k_alpha_1", p.Name, p.Value)
			}
		}
	}
}

func TestParamValuesJSON(t *testing.T) {
	pv := ParamValues{{"k_b", 16}, {"is_a", true}, {"f_c", "max"}, {"k_d", 0.5}}
	data, err := json.Marshal(pv)
	if want := `{"k_b":16,"is_a":true,"f_c":"max","k_d":0.5}`; err != nil || string(data) != want {
		t.Errorf("encodes as %s, %v; want %s", data, err, want)
	}
	var back ParamValues
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	want := ParamValues{{"k_b", 16.0}, {"is_a", true}, {"f_c", "max"}, {"k_d", 0.5}}
	if !slices.Equal(back, want) {
		t.Errorf("decodes as %v, want %v", back, want)
	}
	if data, _ := json.Marshal(ParamValues{}); string(data) != "{}" {
		t.Errorf("no values encode as %s, want {}", data)
	}
	if err := json.Unmarshal([]byte(`[1, 2]`), &back); err == nil {
		t.Error("decoding an array succeeded, want an error")
	}
}
