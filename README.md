# obstacle-gen-playground

A small browser app for trying out algorithms that place random obstacles on
game boards.

The board is a hexagon of hexes, with one obstacle type. Its edge length is
set separately from the algorithm: 6 by default (91 hexes), from 1 to 15. Pick
an algorithm, tweak its parameters, and the board regenerates as you go. The
settings are kept in the address bar, so reloading or sharing the page keeps
them.

Each generation uses a seed, shown above the board. Click "reuse" (or type a
seed) to keep it fixed while you tweak parameters; leave the seed box empty for
a new random board each time.

Use the slider, the arrow buttons or the ← → keys to step through the
placements, from the empty board to the final one. Each step shows the board
before an obstacle is placed, with a heatmap of the choice (score, chance of
being chosen, and so on) and the cell that was chosen outlined. Hover over a
cell to see all its values. The heatmap's colours run from the step's lowest
value to its highest. Changing a parameter keeps you on the same step.

"Copy image" copies the board as shown, heatmap and all, to the clipboard as a
PNG, in the page's current light or dark colours. Browsers only allow that on
secure pages, which includes `localhost` but not plain HTTP from another
machine (say, with `addr=0.0.0.0:8080`), so there it downloads the PNG instead.

Parameters that take real numbers (rather than whole numbers) accept fractions
as well as decimals: `1/2` and `0.5` are the same.

## Algorithms

### Uniform

Places `k_obstacles` (default 16) obstacles one at a time, each on a free hex (no obstacle,
and not making a bank bigger than `k_max_bank`, where 0 means no limit) chosen
with equal probability. It stops early if no hex is free, so with no bank limit
and `k_obstacles` at least the number of hexes, the board ends up full.

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
`f_a'(f_b(f_a(a), …, f_a(f)))`. Otherwise each axis gets its own conjugate and
`f_dir` combines them:
`f_dir(f_a'(f_b(f_a(a), f_a(b))), f_a'(f_b(f_a(c), f_a(d))), f_a'(f_b(f_a(e), f_a(f))))`.
`f_a'` is always the inverse of `f_a`, so it's set automatically and only
shown for reference. That makes each conjugate a generalised mean: the defaults
(`sqrt(x)`, `mean`, `x^2`) give the power mean with exponent ½, and `ln(x)`,
`mean`, `e^x` gives the geometric mean.

- `f_a` (and so `f_a'`): `sqrt(x)` ↔ `x^2`, `x` (identity), `ln(x)` ↔ `e^x`
- `f_b` and `f_dir`: `mean`, `geom_mean`, `harm_mean`, `median`, `max`, `min`

`ln(0)` is −∞, so with `ln(x)` a distance of 0 usually pulls the result to 0.
Some combinations, like `ln(x)` then `geom_mean` (which multiplies −∞ by 0), are
undefined for some distances. A cell whose distance term comes out undefined
or infinite has no score for that round, and shows as "can't be chosen".

**Visibility** is the furthest distance from the board's centre cell to the
edge of an empty board, counted in the cells between them, the same way steps 3
and 4 count (a cell next to an obstacle or the edge is at distance 0). On a
hexagon that's its edge length minus 1, so 5 on the default board. On an empty
board the centre therefore scores 1 for both distance terms. Step 4 counts
aren't capped, so cells near a corner can score more than 1 in some directions
(up to 10/5 on the default board).

### Triple Beta

Uses three beta distributions as weights over the board's hexes, one for each
cube coordinate (`q`, `r` and `s`, with `q + r + s = 0`, each from −R to R on a
board of radius R):

1. Scale each coordinate into (0, 1): the 2R + 1 values it takes split [0, 1]
   into equal bands, and `x → (x + R + ½)/(2R + 1)`, the middle of its band.
2. Weight each hex by the product of the three beta densities at its scaled
   coordinates: `w = f_q(Q) · f_r(R') · f_s(S)`, using `k_alpha_1`/`k_beta_1`
   for `q`, `_2` for `r` and `_3` for `s`.
3. Each round, multiply that by a distance weight (see Distance to obstacles
   below), then put the obstacle on a free hex (no obstacle, and not making a
   bank bigger than `k_max_bank`), picked with probability proportional to its
   weight over the free hexes.

Equal coordinate values form parallel bands across the board, so each
distribution sets how much each band of one family is favoured:

- axis 1 (`q`): lower left edge (0) to upper right edge (1)
- axis 2 (`r`): top edge (0) to bottom edge (1)
- axis 3 (`s`): lower right edge (0) to upper left edge (1)

Beta(1, 1) is flat, so the default (all three flat) is exactly uniform.
Symmetric shapes like Beta(2, 2) favour the middle, and asymmetric ones like
Beta(2, 5) favour one side. The same symmetric shape on every axis gives sixfold
symmetry. `is_alpha_eq_beta` forces α = β for each axis, and `is_axes_shared`
makes axes 2 and 3 use axis 1's α and β. Forced values show locked, the same
way as Evolved's `f_a'`.

**Distance to obstacles.** The distance weight sets how new obstacles relate
to existing ones. `f_obstacles_distance_mode` picks how it's measured:
`aggregate` (the default) by hex distance to the nearest obstacle, or
`per_axis` by each cube coordinate separately. Only the chosen mode's
parameters are shown.

*Aggregate.* A fourth beta distribution (`k_alpha_obstacles`,
`k_beta_obstacles`) sets how far from existing obstacles new ones like to be.
Each round, each free hex's distance `d` is the number of cells between it and
the nearest obstacle (0 if adjacent, the same convention as Evolved). It's
measured up to the board's span S: the longest straight line across the board,
corner to corner through the centre, counted the same way. That's twice the
edge length minus 3, so 9 on the default board. No two hexes are further apart
than that, so no obstacle is ever out of range. With no obstacles yet, every
hex is at S. Like the coordinates, `d` is scaled to the middle of its band,
`(d + ½)/(S + 1)`, and the hex's weight is multiplied by the density there.
The heatmap and hover text show the plain hex distance instead, `d + 1`: 1 if
adjacent, up to 10 on the default board.

- Beta(1, 1) is neutral.
- α > β favours larger distances, spreading obstacles out.
- β > α favours smaller distances, clustering them.
- α = β > 1 favours middling distances, and α = β < 1 favours both extremes.

On an edge-length-5 board (61 hexes), with 16 obstacles and flat positions,
Beta(1, 1) leaves about 10 pairs of adjacent obstacles per board, Beta(6, 1)
about 1 and Beta(1, 6) about 15. With that many obstacles, most distances are 0
to 2, low on that board's 0 to 7 scale, so clustering shapes need a larger β to
pull hard. `is_obstacles_alpha_eq_beta` (off by default) forces α = β.

*Per axis.* Three more beta distributions, one for each cube coordinate, weigh
the shape of the gap between a hex and its nearest obstacle. Each round, each
free hex's nearest obstacle is found by hex distance, and the displacement
between them is split into its coordinates: the q distance is
`d_q = |q − q'|`, and the same for r and s. `d_q = 0` means the obstacle
shares the hex's q band. The largest of the three is the hex distance, and
the other two add up to it. Each runs from 0 to 2R (10 on the default board),
and like the coordinates scales to the middle of its band, `(d + ½)/(2R + 1)`.
The distance weight is the product of the three densities there:

```text
w = position weight × f_q(d_q) × f_r(d_r) × f_s(d_s)
```

using `k_alpha_obstacles_1`/`k_beta_obstacles_1` for q, `_2` for r and `_3`
for s. If several obstacles are equally near, the distance weight is the mean
of their products, so it doesn't depend on which comes first. The heatmap and
hover text then show the means of their q, r and s distances and weights,
which can be fractions (and the distance weight shown is the mean of the
products, not the product of the means).

So the position distributions say where on the board obstacles go, and these
say how each sits relative to its nearest neighbour. For each axis, raising α
favours more separation along it, raising β less, and Beta(1, 1) makes the
axis neutral. With every axis tied, far-favouring shapes spread obstacles out
much like aggregate mode does. Untied, they set which way neighbours line up:
two neighbouring hexes share exactly one band. On the default board with 40
obstacles, r Beta(5, 1) on its own cuts the neighbouring pairs that share an
r band to about 9 per board, against about 15 sharing q or s (and 15 each
with neutral shapes), while r Beta(1, 10) raises them to about 21. For
example, q Beta(5, 1), r Beta(1, 5) and s Beta(1, 1) favour placing an
obstacle in its nearest neighbour's r band but well apart in q: spaced rows.

`is_obstacle_axes_alpha_eq_beta` forces α = β for each axis, and
`is_obstacle_axes_shared` (on by default) makes axes 2 and 3 use axis 1's α
and β, the same way as the position distributions' controls. With no
obstacles yet the term is left out: every axis weight is 1, and the heatmap
shows each distance as 2R.

Like aggregate mode, it can only spread obstacles while there's room. On the
default board, far-favouring shapes leave every free hex next to an obstacle
by about obstacle 26. From then on tied shapes can't tell the free hexes
apart, as every neighbour is (1, 1, 0) away in some order, but untied ones
still set which way neighbours line up.

Tying the three axes isn't the same as aggregate mode. Seen from its nearest
obstacle, a hex 2 steps away along a line, like (2, −2), is (2, 2, 0) away in
q, r and s, sharing its s band, while one 2 steps away between lines, like
(2, −1), is (2, 1, 1) away. Aggregate mode sees both at the same distance, but
per-axis mode generally weighs them differently.

It stops after placing `k_obstacles` obstacles or when no free hex has any
weight. The heatmap's chances are exact, and it can also show each hex's
position weight, distance weight and distance, or per axis, the q, r and s
distances and their weights.

**Edges.** Because coordinates and distances scale to the middles of their
bands, they're never exactly 0 or 1, where a beta density with α or β > 1 is 0
and one with α or β < 1 is infinite. So short of extreme shapes (say α in the
thousands, whose density rounds down to 0), every hex keeps a finite, non-zero
weight: with Beta(2, 2) on every axis, perimeter hexes are unlikely but
possible.

### Split (mrraow)

A greedy line-breaker, based on an existing Java generator. Each round it looks
at every placement: one empty hex, or with `is_symmetric` (on by default) a
hex and its 180° rotation about the centre, placed together.

**Eligibility.** A placement is skipped if, once its obstacles are added:

- any of its hexes has fewer than `k_edge_margin` cells between it and the edge
  (0 allows the perimeter, 1 is the original, 2 keeps another ring clear);
- any of its hexes touches more than `k_max_adjacent` obstacles, a whole
  number from 0 to 6 (0 means none may touch, 1 is the original, 6 is no
  limit). Unlike `k_max_bank`, which caps
  a bank's size, this limits local shape: long chains are fine, but not a hex
  touching two obstacles;
- a bank would be bigger than `k_max_bank` (0 = no limit).

**Score.** For each axis, `sᵢ` is the shorter of the clear runs either way
along it (empty cells before an obstacle or the edge), so a hex scores well when
it splits a long clear line evenly. Then:

```text
split = f_split_axes(s₁, s₂, s₃)           max is the original
raw   = split − k_adjacent_penalty × (obstacles touched) + U{0, …, k_noise}
score = raw / k_score_bucket, rounded towards 0
```

The obstacle goes on a random placement among those with the top score.
`f_split_axes` takes the same choices as Evolved: `max` is happy with one good
line-break, `mean` rewards being useful along several axes, `median` wants at
least two good axes, and `min` wants every axis broken up. `k_score_bucket`
sets how picky it is: 1 makes every point count, 2 is the original, and larger
buckets make more hexes tie, so more is left to chance. Scores round towards 0,
like Java's integer division, so with the default bucket of 2 a raw score of −1
ties with 0 and 1. The defaults reproduce the original.

**Symmetry.** With `is_symmetric`, each step places a pair, both scored the
same, as the board stays symmetric. The centre is its own rotation, so taking
it places a single obstacle. A pair isn't placed when only one obstacle is left,
so it never places more than `k_obstacles`, and it stops early, with a note, if
nothing is eligible. In particular, if the centre is taken (it often is, having
the best split on an empty board), an even count ends one short, and otherwise
an odd count does.

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
(`int`, `float`, `bool` or `choice`), default, minimum, maximum (if any) and
options. `float` parameters accept fractions like `1/2`.

`GET /api/generate?algorithm=<id>[&edge=<1–15>][&seed=<seed>][&<param>=<value>...]`
returns a generated board. `edge` is the board's edge length (default 6).
Parameters that are left out take their defaults.

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

The trace has one step per choice, in order. Most steps place one obstacle at
`placed`; Split (mrraow) with `is_symmetric` also lists the mirrored one in
`also`. Each step lists the cells that could have been chosen, with one value
per metric. If the algorithm stopped early, `trace.note` says why. `placed` at
the top level counts obstacles, not steps.

Cells use [axial coordinates](https://www.redblobgames.com/grids/hexagons/#coordinates-axial).
The seed is a string because it can exceed JavaScript's safe integer range.
