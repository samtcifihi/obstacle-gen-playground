package algorithm

import (
	"slices"

	"github.com/samtcifihi/obstacle-gen-playground/internal/board"
)

// grid looks up cells on a board by position.
type grid struct {
	b     *board.Board
	index map[board.Hex]int
}

// at reports whether h is on the board and, if so, whether it holds an
// obstacle.
func (g grid) at(h board.Hex) (onBoard, obstacle bool) {
	i, ok := g.index[h]
	return ok, ok && g.b.Cells[i].Obstacle
}

// clearance returns the number of empty cells from h in direction dir
// before an obstacle, up to limit. The edge of the board doesn't block.
func (g grid) clearance(h, dir board.Hex, limit int) int {
	for k := 0; k < limit; k++ {
		h = h.Add(dir)
		onBoard, obstacle := g.at(h)
		if !onBoard {
			break
		}
		if obstacle {
			return k
		}
	}
	return limit
}

// emptyToEdge returns the number of empty cells from h in direction dir
// to the edge of the board.
func (g grid) emptyToEdge(h, dir board.Hex) int {
	n := 0
	for {
		h = h.Add(dir)
		onBoard, obstacle := g.at(h)
		if !onBoard {
			return n
		}
		if !obstacle {
			n++
		}
	}
}

// banks records which bank (contiguous group of obstacles) each obstacle
// belongs to.
type banks struct {
	of    []int // bank of each cell, or -1 if it's empty
	sizes []int // number of obstacles in each bank
}

func (g grid) banks() banks {
	bs := banks{of: make([]int, len(g.b.Cells))}
	for i := range bs.of {
		bs.of[i] = -1
	}
	for i, c := range g.b.Cells {
		if !c.Obstacle || bs.of[i] >= 0 {
			continue
		}
		bank := len(bs.sizes)
		bs.sizes = append(bs.sizes, 0)
		stack := []int{i}
		bs.of[i] = bank
		for len(stack) > 0 {
			j := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			bs.sizes[bank]++
			for _, n := range g.b.Cells[j].Neighbours() {
				if k, ok := g.index[n]; ok && g.b.Cells[k].Obstacle && bs.of[k] < 0 {
					bs.of[k] = bank
					stack = append(stack, k)
				}
			}
		}
	}
	return bs
}

// sizeWith returns the size of the bank an obstacle on empty hex h would
// be part of.
func (bs banks) sizeWith(g grid, h board.Hex) int {
	size := 1
	var joined []int
	for _, n := range h.Neighbours() {
		i, ok := g.index[n]
		if !ok || bs.of[i] < 0 || slices.Contains(joined, bs.of[i]) {
			continue
		}
		joined = append(joined, bs.of[i])
		size += bs.sizes[bs.of[i]]
	}
	return size
}

// sizeWithAll returns the size of the largest bank there would be with
// obstacles on all of hs, which must be empty: obstacles on hs that touch
// each other, or the same bank, end up in one bank together.
func (bs banks) sizeWithAll(g grid, hs []board.Hex) int {
	// The banks each new obstacle would touch.
	touches := make([][]int, len(hs))
	for i, h := range hs {
		for _, n := range h.Neighbours() {
			if j, ok := g.index[n]; ok && bs.of[j] >= 0 && !slices.Contains(touches[i], bs.of[j]) {
				touches[i] = append(touches[i], bs.of[j])
			}
		}
	}
	// Group the new obstacles that end up in the same bank.
	group := make([]int, len(hs))
	for i := range group {
		group[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if group[i] != i {
			group[i] = find(group[i])
		}
		return group[i]
	}
	for i := range hs {
		for j := i + 1; j < len(hs); j++ {
			shared := slices.ContainsFunc(touches[i], func(bank int) bool { return slices.Contains(touches[j], bank) })
			if hs[i].DistanceTo(hs[j]) == 1 || shared {
				group[find(i)] = find(j)
			}
		}
	}
	largest := 0
	for root := range hs {
		if find(root) != root {
			continue
		}
		size := 0
		var joined []int
		for i := range hs {
			if find(i) != root {
				continue
			}
			size++
			for _, bank := range touches[i] {
				if !slices.Contains(joined, bank) {
					joined = append(joined, bank)
					size += bs.sizes[bank]
				}
			}
		}
		largest = max(largest, size)
	}
	return largest
}

// run returns the number of empty cells from h in direction dir before an
// obstacle or the edge of the board.
func (g grid) run(h, dir board.Hex) int {
	n := 0
	for {
		h = h.Add(dir)
		if onBoard, obstacle := g.at(h); !onBoard || obstacle {
			return n
		}
		n++
	}
}

// edgeDistance returns the number of cells between h and the nearest edge
// of the board, ignoring obstacles: 0 for a cell on the edge.
func (g grid) edgeDistance(h board.Hex) int {
	d := -1
	for _, dir := range (board.Hex{}).Neighbours() {
		n := 0
		for p := h.Add(dir); ; p = p.Add(dir) {
			if onBoard, _ := g.at(p); !onBoard {
				break
			}
			n++
		}
		if d < 0 || n < d {
			d = n
		}
	}
	return d
}
