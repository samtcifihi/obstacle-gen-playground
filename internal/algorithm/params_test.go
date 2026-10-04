package algorithm

import (
	"math"
	"net/url"
	"testing"
)

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
