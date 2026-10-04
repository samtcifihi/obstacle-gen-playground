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

func TestGenerate(t *testing.T) {
	resp := generate(t, "n=10&seed=42")
	if resp.Seed != 42 {
		t.Errorf("seed = %d, want 42", resp.Seed)
	}
	if resp.Placed != 10 {
		t.Errorf("placed = %d, want 10", resp.Placed)
	}
	if resp.Board.EdgeLength != boardEdgeLength {
		t.Errorf("edge length = %d, want %d", resp.Board.EdgeLength, boardEdgeLength)
	}
	obstacles := 0
	for _, c := range resp.Board.Cells {
		if c.Obstacle {
			obstacles++
		}
	}
	if obstacles != 10 {
		t.Errorf("board has %d obstacles, want 10", obstacles)
	}
}

func TestGenerateFillsBoard(t *testing.T) {
	resp := generate(t, "n=1000")
	if resp.Placed != len(resp.Board.Cells) {
		t.Errorf("placed = %d, want every one of the %d cells", resp.Placed, len(resp.Board.Cells))
	}
}

func TestGenerateSeedIsReproducible(t *testing.T) {
	first := generate(t, "n=5")
	again := generate(t, "n=5&seed="+strconv.FormatUint(first.Seed, 10))
	if !reflect.DeepEqual(first, again) {
		t.Errorf("reusing seed %d gave a different board", first.Seed)
	}
}

func TestGenerateRejectsBadParams(t *testing.T) {
	for _, query := range []string{"", "n=", "n=-1", "n=abc", "n=1.5", "n=3&seed=-1", "n=3&seed=x"} {
		if rec := get(t, "/api/generate?"+query); rec.Code != http.StatusBadRequest {
			t.Errorf("GET /api/generate?%s: status %d, want %d", query, rec.Code, http.StatusBadRequest)
		}
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
