package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"

	"github.com/samtcifihi/obstacle-gen-playground/internal/algorithm"
	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

// The board is a hexagon whose sides are edge hexes long. index.html's edge
// field has the same default and maximum.
const (
	defaultEdgeLength = 6
	// maxEdgeLength keeps boards, and the traces of filling them, a
	// manageable size.
	maxEdgeLength = 15
)

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
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/algorithms", handleAlgorithms)
	mux.HandleFunc("GET /api/generate", handleGenerate)
	mux.Handle("GET /", http.FileServerFS(static))
	return mux
}

// handleAlgorithms lists the algorithms and their parameters.
func handleAlgorithms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, algorithm.All)
}

// handleGenerate builds a board and places obstacles on it. Query
// parameters:
//
//	algorithm  ID of the algorithm to use (required)
//	edge       edge length of the board (optional; defaultEdgeLength if omitted)
//	seed       random seed (optional; a random one is chosen and returned if omitted)
//
// plus the algorithm's own parameters, which take their defaults if
// omitted.
func handleGenerate(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	alg, ok := algorithm.Lookup(query.Get("algorithm"))
	if !ok {
		ids := make([]string, len(algorithm.All))
		for i, a := range algorithm.All {
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
	edge := defaultEdgeLength
	if raw := query.Get("edge"); raw != "" {
		edge, err = strconv.Atoi(raw)
		if err != nil || edge < 1 || edge > maxEdgeLength {
			http.Error(w, fmt.Sprintf("edge must be a whole number from 1 to %d", maxEdgeLength), http.StatusBadRequest)
			return
		}
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

	b := board.NewHexagon(edge)
	trace := alg.Run(b, params, rand.New(rand.NewPCG(seed, 0)))
	writeJSON(w, generateResponse{Seed: seed, Placed: trace.Placed(), Board: b, Trace: trace})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writing response: %v", err)
	}
}
