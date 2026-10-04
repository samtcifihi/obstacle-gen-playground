package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

func get(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	newHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func generate(t *testing.T, query string) generateResponse {
	t.Helper()
	rec := get(t, "/api/generate?"+query)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/generate?%s: status %d, body %q", query, rec.Code, rec.Body)
	}
	var resp generateResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func countObstacles(b *board.Board) int {
	count := 0
	for _, c := range b.Cells {
		if c.Obstacle {
			count++
		}
	}
	return count
}

func TestGenerate(t *testing.T) {
	resp := generate(t, "algorithm=uniform&n=10&seed=42")
	if resp.Seed != 42 {
		t.Errorf("seed = %d, want 42", resp.Seed)
	}
	if resp.Placed != 10 {
		t.Errorf("placed = %d, want 10", resp.Placed)
	}
	if resp.Board.EdgeLength != boardEdgeLength {
		t.Errorf("edge length = %d, want %d", resp.Board.EdgeLength, boardEdgeLength)
	}
	if got := countObstacles(resp.Board); got != 10 {
		t.Errorf("board has %d obstacles, want 10", got)
	}
}

func TestGenerateEvolved(t *testing.T) {
	resp := generate(t, "algorithm=evolved&k_obstacles=12&k_max_bank=1&f_obstacles_a=ln&is_weighted=false")
	if resp.Placed != 12 {
		t.Errorf("placed = %d, want 12", resp.Placed)
	}
	if got := countObstacles(resp.Board); got != 12 {
		t.Errorf("board has %d obstacles, want 12", got)
	}
	if len(resp.Trace.Steps) != 12 || len(resp.Trace.Metrics) == 0 {
		t.Errorf("trace has %d steps and %d metrics, want 12 steps and some metrics", len(resp.Trace.Steps), len(resp.Trace.Metrics))
	}

	// With no parameters, it uses the defaults.
	if resp := generate(t, "algorithm=evolved"); resp.Placed != 16 {
		t.Errorf("placed = %d with default parameters, want 16", resp.Placed)
	}
}

func TestGenerateTripleBeta(t *testing.T) {
	resp := generate(t, "algorithm=triple_beta&k_obstacles=10&is_axes_shared=false&k_alpha_2=3")
	if resp.Placed != 10 || countObstacles(resp.Board) != 10 {
		t.Errorf("placed = %d with %d on the board, want 10", resp.Placed, countObstacles(resp.Board))
	}
	for _, step := range resp.Trace.Steps {
		if step.Tries < 1 {
			t.Errorf("step placing %v records %d tries", step.Placed, step.Tries)
		}
	}
}

func TestGenerateNothingEncodesEmptySteps(t *testing.T) {
	for _, query := range []string{
		"algorithm=uniform&n=0",
		"algorithm=evolved&k_obstacles=0",
		"algorithm=triple_beta&k_alpha_1=0.5",
	} {
		rec := get(t, "/api/generate?"+query)
		if body := rec.Body.String(); !strings.Contains(body, `"steps":[]`) {
			t.Errorf("GET /api/generate?%s: trace steps aren't an empty list: %s", query, body)
		}
	}
}

func TestGenerateFillsBoard(t *testing.T) {
	resp := generate(t, "algorithm=uniform&n=1000")
	if resp.Placed != len(resp.Board.Cells) {
		t.Errorf("placed = %d, want every one of the %d cells", resp.Placed, len(resp.Board.Cells))
	}
}

func TestGenerateSeedIsReproducible(t *testing.T) {
	for _, query := range []string{"algorithm=uniform&n=5", "algorithm=evolved"} {
		first := generate(t, query)
		again := generate(t, query+"&seed="+strconv.FormatUint(first.Seed, 10))
		if !reflect.DeepEqual(first, again) {
			t.Errorf("%s: reusing seed %d gave a different board", query, first.Seed)
		}
	}
}

func TestGenerateRejectsBadParams(t *testing.T) {
	for _, query := range []string{
		"",
		"n=3",
		"algorithm=nope",
		"algorithm=uniform&n=",
		"algorithm=uniform&n=-1",
		"algorithm=uniform&n=abc",
		"algorithm=uniform&n=1.5",
		"algorithm=uniform&n=3&seed=-1",
		"algorithm=uniform&n=3&seed=x",
		"algorithm=evolved&k_beta=0",
		"algorithm=evolved&f_edge_b=mode",
		"algorithm=triple_beta&is_symmetric=false&k_beta_1=0",
		"algorithm=triple_beta&k_max_tries=-1",
	} {
		if rec := get(t, "/api/generate?"+query); rec.Code != http.StatusBadRequest {
			t.Errorf("GET /api/generate?%s: status %d, want %d", query, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestAlgorithms(t *testing.T) {
	rec := get(t, "/api/algorithms")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/algorithms: status %d", rec.Code)
	}
	var algs []struct {
		ID     string
		Params []struct{ Name string }
	}
	if err := json.NewDecoder(rec.Body).Decode(&algs); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(algs) != 3 || algs[0].ID != "uniform" || algs[1].ID != "evolved" || algs[2].ID != "triple_beta" {
		t.Fatalf("algorithms = %+v, want uniform, evolved and triple_beta", algs)
	}
	if n := len(algs[1].Params); n != 17 {
		t.Errorf("evolved has %d parameters, want 17", n)
	}
}

func TestServesApp(t *testing.T) {
	rec := get(t, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<svg") {
		t.Errorf("GET / doesn't look like the app: %q", rec.Body)
	}
}

func TestBrowserURL(t *testing.T) {
	tests := map[string]string{
		"127.0.0.1:8080":   "http://localhost:8080/",
		"[::]:8080":        "http://localhost:8080/",
		"0.0.0.0:9000":     "http://localhost:9000/",
		"192.168.1.5:8080": "http://192.168.1.5:8080/",
	}
	for addr, want := range tests {
		tcp, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		if got := browserURL(tcp); got != want {
			t.Errorf("browserURL(%s) = %s, want %s", addr, got, want)
		}
	}
}
