package algorithm

import (
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"testing"
)

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
