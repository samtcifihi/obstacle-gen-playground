# obstacle-gen-playground

A small browser app for trying out algorithms that place random obstacles on
game boards.

The board is currently a hexagon of hexes with edge length 5 (61 hexes), with
one obstacle type. Pick an algorithm, tweak its parameters, and the board
regenerates as you go. The settings are kept in the address bar, so reloading
or sharing the page keeps them.

Each generation uses a seed, shown above the board. Click "reuse" (or type a
seed) to keep it fixed while you tweak parameters; leave the seed box empty for
a new random board each time.

Use the slider, the arrow buttons or the ← → keys to step through the
placements, from the empty board to the final one. Each step shows the board
before an obstacle is placed, with a heatmap of the choice (score, chance of
being chosen, and so on) and the cell that was chosen outlined. Hover over a
cell to see all its values. The heatmap's colours run from the step's lowest
value to its highest. Changing a parameter keeps you on the same step.

## Algorithms

### Uniform random

Places `n` obstacles, each on an empty hex chosen with equal probability. If
`n` is at least the number of empty hexes, the board ends up full.

### Scored

Places obstacles one at a time. Each round:

1. Empty cells start with a score of 0. Obstacles have no score, so they're
   ignored and can't be chosen.
2. Cells where an obstacle would make a bank (contiguous group of obstacles)
   bigger than `k_max_bank` have no score (`0` = no limit).
3. For each of the six directions, count the empty cells before an obstacle,
   up to `visibility`. The edge of the board doesn't block, so a direction with
   no obstacle within `visibility` counts as `visibility`. Combine the counts
   with the obstacle conjugate (below), divide by `visibility`, and add to the
   score.
4. For each of the six directions, count the empty cells before the edge of
   the board (obstacles on the way are skipped, not counted). Combine with the
   edge conjugate, divide by `visibility`, and add.
5. Add `(s - 0.5) * k_beta_coef`, with `s` drawn from Beta(`k_beta`,
   `k_beta`).
6. If `is_stretching`, map each score `s` to `e^(k_stretch * (s - max))`;
   otherwise to `s + 1 - min`. Either way every score is positive, and with
   stretching the best cell gets 1.
7. If `is_weighted`, place an obstacle on a random cell weighted by score;
   otherwise on the highest-scoring cell, breaking ties randomly.

It stops after placing `k_obstacles` obstacles or when no cell has a score.

**Conjugates.** Steps 3 and 4 reduce the six counts, `[[a, b], [c, d], [e, f]]`
grouped by axis, to one number. Flattened, that's
`f_a'(f_b(f_a(a), …, f_a(f)))`. Otherwise it's
`f_a'(f_dir(f_b(f_a(a), f_a(b)), f_b(f_a(c), f_a(d)), f_b(f_a(e), f_a(f))))`.
With `f_a'` the inverse of `f_a`, this is a generalised mean: the defaults
(`2 root`, `mean`, `2 ^`) give the power mean with exponent ½.

**Visibility** is the furthest distance from the board's centre cell to the
edge of an empty board, counted in the cells between them, the same way steps 3
and 4 count (a cell next to an obstacle or the edge is at distance 0). On a
hexagon that's its edge length minus 1, so 4 here. On an empty board the centre
therefore scores 1 for both distance terms. Step 4 counts aren't capped, so
cells near a corner can score more than 1 in some directions (up to 8/4 here).

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

`GET /api/algorithms` lists the algorithms with their parameters: name, type
(`int`, `float`, `bool` or `choice`), default, minimum and options.

`GET /api/generate?algorithm=<id>[&seed=<seed>][&<param>=<value>...]` returns a
generated board. Parameters that are left out take their defaults.

```json
{
  "seed": "42",
  "placed": 10,
  "board": {
    "edgeLength": 5,
    "cells": [{ "q": 0, "r": -4, "obstacle": false }, ...]
  },
  "trace": {
    "metrics": [{ "name": "score", "description": "Score (steps 1–5)" }, ...],
    "steps": [
      {
        "placed": { "q": 1, "r": -2 },
        "candidates": [{ "q": 0, "r": -4, "values": [1.23, ...] }, ...]
      },
      ...
    ]
  }
}
```

The trace has one step per obstacle, in order. Each step lists the cells that
could have been chosen, with one value per metric.

Cells use [axial coordinates](https://www.redblobgames.com/grids/hexagons/#coordinates-axial).
The seed is a string because it can exceed JavaScript's safe integer range.
