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

### Evolved

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
`f_a'` is always the inverse of `f_a`, so it's set automatically and only
shown for reference. That makes the conjugate a generalised mean: the defaults
(`2 root`, `mean`, `2 ^`) give the power mean with exponent ½, and `ln`, `mean`,
`e x ^` gives the geometric mean.

- `f_a` (and so `f_a'`): `2 root` ↔ `2 ^`, `T f -> T :: x` (identity),
  `ln` ↔ `e x ^`
- `f_b` and `f_dir`: `mean`, `geom_mean`, `harm_mean`, `median`, `max`, `min`

`ln 0` is −∞, so with `ln` a distance of 0 usually pulls the result to 0. Some
combinations, like `ln` then `geom_mean` (which multiplies −∞ by 0), are
undefined for some distances. A cell whose distance term comes out undefined
or infinite has no score for that round, and shows as "can't be chosen".

**Visibility** is the furthest distance from the board's centre cell to the
edge of an empty board, counted in the cells between them, the same way steps 3
and 4 count (a cell next to an obstacle or the edge is at distance 0). On a
hexagon that's its edge length minus 1, so 4 here. On an empty board the centre
therefore scores 1 for both distance terms. Step 4 counts aren't capped, so
cells near a corner can score more than 1 in some directions (up to 8/4 here).

### Triple Beta

Uses three beta distributions as weights over the board's hexes, one for each
cube coordinate (`q`, `r` and `s`, with `q + r + s = 0`, each from −R to R on a
board of radius R):

1. Scale each coordinate into [0, 1]: `x → (x/R + 1)/2`.
2. Weight each hex by the product of the three beta densities at its scaled
   coordinates: `w = f_q(Q) · f_r(R') · f_s(S)`, using `k_alpha_1`/`k_beta_1`
   for `q`, `_2` for `r` and `_3` for `s`.
3. Each try picks a hex of the board with probability `w / Σw`. It fails if the
   hex already has an obstacle or would make a bank bigger than `k_max_bank`.

Equal coordinate values form parallel bands across the board, so each
distribution sets how much each band of one family is favoured:

- axis 1 (`q`): lower left edge (0) to upper right edge (1)
- axis 2 (`r`): top edge (0) to bottom edge (1)
- axis 3 (`s`): lower right edge (0) to upper left edge (1)

Beta(1, 1) is flat, so the default (all three flat) is exactly uniform.
Symmetric shapes like Beta(2, 2) favour the middle, and asymmetric ones like
Beta(2, 5) favour one side. The same symmetric shape on every axis gives sixfold
symmetry. `is_symmetric` forces α = β for each axis, and `is_axes_shared` makes
axes 2 and 3 use axis 1's α and β. Forced values show locked, the same way as
Evolved's `f_a'`.

It stops after placing `k_obstacles` obstacles, after `k_max_tries` tries in
all (successful or not; default twice the number of cells), or when no free
cell has any weight. Failed tries don't get their own steps, but each step says
how many tries it took. The heatmap's chances are exact.

**Edges.** A density of 0 at 0 or 1 (α or β > 1) gives the matching edge hexes
no weight: with Beta(2, 2) on every axis, no perimeter hex can be chosen. A
density with α or β < 1 is infinite at 0 or 1, which gives edge hexes infinite
weight, so it won't place anything. `is_inset` fixes both: it scales each
coordinate to the middle of its band, `(x + R + ½)/(2R + 1)`, so the edges sit
just inside (0, 1).

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
could have been chosen, with one value per metric, and, for Triple Beta, how
many tries it took. If the algorithm stopped early, `trace.note` says why.

Cells use [axial coordinates](https://www.redblobgames.com/grids/hexagons/#coordinates-axial).
The seed is a string because it can exceed JavaScript's safe integer range.
