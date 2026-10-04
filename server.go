package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"math/rand/v2"
	"net/http"
	"strconv"

	"github.com/samtcifihi/obstacle-gen-playground/internal/algorithm"
	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

// boardEdgeLength is the side length, in hexes, of the generated board.
const boardEdgeLength = 5

//go:embed web
var webFS embed.FS

type generateResponse struct {
	Seed   uint64       `json:"seed,string"`
	Placed int          `json:"placed"`
	Board  *board.Board `json:"board"`
}

func newHandler() http.Handler {
	static, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/generate", handleGenerate)
	mux.Handle("GET /", http.FileServerFS(static))
	return mux
}

// handleGenerate builds a board and places obstacles on it. Query
// parameters:
//
//	n     number of obstacles to place (required, >= 0)
//	seed  random seed (optional; a random one is chosen and returned if omitted)
func handleGenerate(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	n, err := strconv.Atoi(query.Get("n"))
	if err != nil || n < 0 {
		http.Error(w, "n must be a non-negative integer", http.StatusBadRequest)
		return
	}
	// Random seeds are kept to 32 bits so they're easy to note down.
	seed := uint64(rand.Uint32())
	if s := query.Get("seed"); s != "" {
		seed, err = strconv.ParseUint(s, 10, 64)
		if err != nil {
			http.Error(w, "seed must be a non-negative integer", http.StatusBadRequest)
			return
		}
	}

	b := board.NewHexagon(boardEdgeLength)
	placed := algorithm.Uniform(b, n, rand.New(rand.NewPCG(seed, 0)))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(generateResponse{Seed: seed, Placed: placed, Board: b}); err != nil {
		log.Printf("writing response: %v", err)
	}
}
