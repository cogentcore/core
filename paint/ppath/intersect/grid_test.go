// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intersect_test

import (
	"math/rand"
	"testing"

	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/ppath/stroke"
	"github.com/stretchr/testify/assert"
)

// polygonArea returns the shoelace area of a flat path (all subpaths).
func polygonArea(p ppath.Path) float32 {
	a := float32(0)
	for _, sp := range p.Split() {
		pts := sp.Coords()
		for i := range pts {
			j := (i + 1) % len(pts)
			a += pts[i].X*pts[j].Y - pts[j].X*pts[i].Y
		}
	}
	return a / 2
}

// noisyGrid returns n×n adjacent square cells of size 0.1 whose corners
// carry random noise of the given magnitude, as the floating point drift of
// computed geometry such as Voronoi cells.
func noisyGrid(n int, eps float32, rng *rand.Rand) ppath.Path {
	var p ppath.Path
	nz := func() float32 { return (rng.Float32()*2 - 1) * eps }
	for i := range n {
		for j := range n {
			x, y := 0.05+0.1*float32(i), 0.05+0.1*float32(j)
			p.MoveTo(x+nz(), y+nz())
			p.LineTo(x+0.1+nz(), y+nz())
			p.LineTo(x+0.1+nz(), y+0.1+nz())
			p.LineTo(x+nz(), y+0.1+nz())
			p.Close()
		}
	}
	return p
}

// TestStrokeNoisyGrid strokes a merged grid of nearly exact cells, which
// settles the outline: the stroke outlines of adjacent cells overlap along
// the shared edges, as stacks of (nearly) identical segments starting in the
// same point, and the noise leaves some segments with both ends in the same
// grid column. The settled outline of the 10×10 grid is the single outer
// square of side 1.2. (after https://github.com/tdewolff/canvas/pull/394)
//
// Noise up to 1e-7 is merged by the snapping of [intersect.BentleyOttmannEpsilon]
// (1e-6), which is as large as it is because of the precision of float32.
func TestStrokeNoisyGrid(t *testing.T) {
	for _, eps := range []float32{0, 1e-9, 1e-8, 1e-7} {
		for seed := range int64(6) {
			rng := rand.New(rand.NewSource(seed))
			grid := noisyGrid(10, eps, rng)
			var res ppath.Path
			assert.NotPanics(t, func() {
				res = stroke.Stroke(grid, 0.2, stroke.ButtCap, stroke.MiterJoin, 0.01)
			}, "eps %g seed %d", eps, seed)
			n, a := len(res.Split()), polygonArea(res)
			assert.Equal(t, 1, n, "eps %g seed %d: subpaths", eps, seed)
			assert.InDelta(t, 1.44, a, 1e-4, "eps %g seed %d: area", eps, seed)
		}
	}
}
