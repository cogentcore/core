// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intersect

import (
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBooleanSharedEdges tests boolean operations of squares whose edges
// overlap (coinciding wholly or partly), and of a square with itself.
func TestBooleanSharedEdges(t *testing.T) {
	tests := []struct {
		name         string
		p, q         ppath.Path
		and, or, xor float32
		not          float32
		orSubpaths   int
	}{
		{"same", square(0, 0, 2), square(0, 0, 2), 4, 4, 0, 0, 1},
		{"adjacent", square(0, 0, 2), square(2, 0, 2), 0, 8, 8, 4, 1},
		{"partly adjacent", square(0, 0, 2), square(2, 1, 2), 0, 8, 8, 4, 1},
		{"overlapping edges", square(0, 0, 2), square(1, 0, 2), 2, 6, 4, 2, 1},
		{"inside sharing a corner", square(0, 0, 4), square(0, 0, 2), 4, 16, 12, 12, 1},
		{"corner to corner", square(0, 0, 2), square(2, 2, 2), 0, 8, 8, 4, 2},
		{"reversed", square(0, 0, 2), square(1, 0, 2).Reverse(), 2, 6, 4, 2, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.and, area(And(tt.p, tt.q)), 1e-4, "and")
			or := Or(tt.p, tt.q)
			assert.InDelta(t, tt.or, area(or), 1e-4, "or")
			assert.InDelta(t, tt.xor, area(Xor(tt.p, tt.q)), 1e-4, "xor")
			assert.InDelta(t, tt.not, area(Not(tt.p, tt.q)), 1e-4, "not")
			assert.Len(t, or.Split(), tt.orSubpaths, "or: %v", or)
		})
	}
}

// TestBooleanMany tests boolean operations of a grid of overlapping
// squares, with many intersections, and of triangles touching at points.
func TestBooleanMany(t *testing.T) {
	var ps, qs ppath.Paths
	for i := range 4 {
		ps = append(ps, square(float32(i)*1.5, 0, 2))
		qs = append(qs, square(0, float32(i)*1.5, 2))
	}
	or := OrPaths(ps, qs)
	tolEqualBox2(t, or.FastBounds(), math32.B2(0, 0, 6.5, 6.5))
	// each row of squares covers 6.5x2, and they overlap in a 2x2 square
	assert.InDelta(t, 2*6.5*2-4, area(or), 1e-3)
	assert.InDelta(t, 4, area(AndPaths(ps, qs)), 1e-3)

	// triangles touching at a vertex, and a vertex touching an edge
	a := *ppath.New().Polygon(math32.Vec2(0, 0), math32.Vec2(2, 0), math32.Vec2(1, 1))
	b := *ppath.New().Polygon(math32.Vec2(1, 1), math32.Vec2(2, 2), math32.Vec2(0, 2))
	c := *ppath.New().Polygon(math32.Vec2(1, 0), math32.Vec2(2, -1), math32.Vec2(0, -1))
	assert.InDelta(t, 2, area(Or(a, b)), 1e-4)
	assert.InDelta(t, 0, area(And(a, b)), 1e-4)
	assert.InDelta(t, 2, area(Or(a, c)), 1e-4)
	assert.InDelta(t, 1, area(Not(a, c)), 1e-4)
}

// TestIntersectionLineEllipse tests the intersections of lines with an
// arc of an ellipse (not a circle): crossing it, and touching it.
func TestIntersectionLineEllipse(t *testing.T) {
	origin := math32.Vec2(0, 0)
	// half of an ellipse centered at (2,0), with radii 2 and 1
	arc := arcSeg(2, 1, false, true, 4, 0)
	// which half it is
	top := float32(1)
	if zs := IntersectionSegment(nil, math32.Vec2(2, -2), seg(ppath.LineTo, 2, 2), origin, arc); len(zs) == 1 && zs[0].Y < 0 {
		top = -1
	}
	// a vertical line crosses it at its middle
	zs := IntersectionSegment(nil, math32.Vec2(2, -2), seg(ppath.LineTo, 2, 2), origin, arc)
	require.Len(t, zs, 1)
	tolEqualVec2(t, zs[0].Vector2, math32.Vec2(2, top))
	assert.False(t, zs.HasTangent())

	// a horizontal line touches it there
	zs = IntersectionSegment(nil, math32.Vec2(0, top), seg(ppath.LineTo, 4, top), origin, arc)
	require.Len(t, zs, 1, "touching")
	tolEqualVec2(t, zs[0].Vector2, math32.Vec2(2, top))
	assert.True(t, zs.HasTangent())

	// a slanted line crosses it at the same point, which is not touching
	zs = IntersectionSegment(nil, math32.Vec2(1, 1.5*top), seg(ppath.LineTo, 3, 0.5*top), origin, arc)
	require.Len(t, zs, 1, "slanted")
	tolEqualVec2(t, zs[0].Vector2, math32.Vec2(2, top))
	assert.False(t, zs.HasTangent(), "a crossing, not a tangent")

	// a horizontal line crosses it twice, and a line beyond it not at all
	assert.Len(t, IntersectionSegment(nil, math32.Vec2(-1, top/2), seg(ppath.LineTo, 5, top/2), origin, arc), 2)
	assert.Empty(t, IntersectionSegment(nil, math32.Vec2(-1, 3*top), seg(ppath.LineTo, 5, 3*top), origin, arc))
	assert.Empty(t, IntersectionSegment(nil, math32.Vec2(1, 1), seg(ppath.LineTo, 1, 1), origin, arc), "a zero length line")
}

// TestBooleanSnapping tests boolean operations of shapes with vertices
// within the tolerance of edges of the other shape, which are snapped to
// them, breaking up those edges, and a third shape crossing at the same point.
func TestBooleanSnapping(t *testing.T) {
	sq := square(0, 0, 2)
	eps := ppath.Tolerance / 10
	// triangles with a vertex just off the bottom and the left edges
	below := *ppath.New().Polygon(math32.Vec2(1, eps), math32.Vec2(3, -1), math32.Vec2(3, 1))
	left := *ppath.New().Polygon(math32.Vec2(eps, 1), math32.Vec2(-1, 3), math32.Vec2(1, 3))
	for _, tri := range []ppath.Path{below, left} {
		and := And(sq, tri)
		or := Or(sq, tri)
		assert.InDelta(t, area(sq)+area(tri), area(and)+area(or), 1e-3, "and + or = p + q")
		assert.InDelta(t, area(or)-area(and), area(Xor(sq, tri)), 1e-3)
		assert.InDelta(t, area(sq)-area(and), area(Not(sq, tri)), 1e-3)
	}
	// three shapes crossing at one point
	x := *ppath.New().Polygon(math32.Vec2(0, 0), math32.Vec2(2, 2), math32.Vec2(2, 0))
	y := *ppath.New().Polygon(math32.Vec2(0, 2), math32.Vec2(2, 0), math32.Vec2(2+eps, 2))
	or := OrPaths(ppath.Paths{x, y}, ppath.Paths{sq})
	assert.InDelta(t, 4, area(or), 1e-3)
	assert.InDelta(t, area(x), area(AndPaths(ppath.Paths{x}, ppath.Paths{sq})), 1e-3)
}
