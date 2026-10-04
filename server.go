package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"

	"github.com/samtcifihi/obstacle-gen-playground/internal/algorithm"
	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

// boardEdgeLength is the side length, in hexes, of the generated board.
const boardEdgeLength = 5

//go:embed web
var webFS embed.FS

type generateResponse struct {
	Seed   uint64          `json:"seed,string"`
	Placed int             `json:"placed"`
	Board  *board.Board    `json:"board"`
	Trace  algorithm.Trace `json:"trace"`
}

func newHandler() http.Handler {
	static, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}
	s := &server{algorithms: algorithm.Algorithms(board.NewHexagon(boardEdgeLength))}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/algorithms", s.handleAlgorithms)
	mux.HandleFunc("GET /api/generate", s.handleGenerate)
	mux.Handle("GET /", http.FileServerFS(static))
	return mux
}

type server struct {
	// algorithms has defaults worked out for the board generate uses.
	algorithms []algorithm.Algorithm
}

// handleAlgorithms lists the algorithms and their parameters.
func (s *server) handleAlgorithms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.algorithms)
}

// handleGenerate builds a board and places obstacles on it. Query
// parameters:
//
//	algorithm  ID of the algorithm to use (required)
//	seed       random seed (optional; a random one is chosen and returned if omitted)
//
// plus the algorithm's own parameters, which take their defaults if
// omitted.
func (s *server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	alg, ok := algorithm.Lookup(s.algorithms, query.Get("algorithm"))
	if !ok {
		ids := make([]string, len(s.algorithms))
		for i, a := range s.algorithms {
			ids[i] = a.ID
		}
		http.Error(w, "algorithm must be one of: "+strings.Join(ids, ", "), http.StatusBadRequest)
		return
	}
	params, err := alg.Parse(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Random seeds are kept to 32 bits so they're easy to note down.
	seed := uint64(rand.Uint32())
	if raw := query.Get("seed"); raw != "" {
		seed, err = strconv.ParseUint(raw, 10, 64)
		if err != nil {
			http.Error(w, "seed must be a non-negative integer", http.StatusBadRequest)
			return
		}
	}

	b := board.NewHexagon(boardEdgeLength)
	trace := alg.Run(b, params, rand.New(rand.NewPCG(seed, 0)))
	writeJSON(w, generateResponse{Seed: seed, Placed: len(trace.Steps), Board: b, Trace: trace})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writing response: %v", err)
	}
}
