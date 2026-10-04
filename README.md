# obstacle-gen-playground

A small browser app for trying out algorithms that place random obstacles on
game boards.

The board is currently a hexagon of hexes with edge length 5 (61 hexes), with
one obstacle type and one algorithm:

- **Uniform random** — places `n` obstacles, each on an empty hex chosen with
  equal probability. If `n` is at least the number of empty hexes, the board
  ends up full.

Each generation uses a seed, shown under the controls. Enter it in the seed box
to reproduce a board; leave the box empty for a new random one.

## Running

Requires [Go](https://go.dev/) 1.22+ and [just](https://github.com/casey/just).

```sh
just          # start the server on http://localhost:8080/
just open     # start the server and open it in your default browser
just test     # run the tests
```

To listen somewhere else, override `addr`, e.g. `just addr=:9000` or
`just addr=0.0.0.0:8080 open`.

Without just: `go run .` (flags: `-addr`, `-open`).

## API

`GET /api/generate?n=<count>[&seed=<seed>]` returns the generated board:

```json
{
  "seed": "42",
  "placed": 10,
  "board": {
    "edgeLength": 5,
    "cells": [{ "q": 0, "r": -4, "obstacle": false }, ...]
  }
}
```

Cells use [axial coordinates](https://www.redblobgames.com/grids/hexagons/#coordinates-axial).
The seed is a string because it can exceed JavaScript's safe integer range.
