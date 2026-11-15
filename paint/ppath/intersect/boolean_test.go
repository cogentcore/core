// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intersect

import (
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"github.com/stretchr/testify/assert"
)

// area returns the net area of the given path, flattening it first,
// as the sum of the signed areas of its subpaths (which are oriented
// CCW for fills and CW for holes after a boolean operation).
func area(p ppath.Path) float32 {
	p = Flatten(p, 0.001)
	var sum float32
	for _, sp := range p.Split() {
		pts := sp.Coords()
		n := len(pts)
		for i := range n {
			a, b := pts[i], pts[(i+1)%n]
			sum += a.X*b.Y - b.X*a.Y
		}
	}
	return math32.Abs(sum / 2)
}

// square returns a square path of the given size at the given position.
func square(x, y, size float32) ppath.Path {
	return *ppath.New().Rectangle(x, y, size, size)
}

// TestBooleanOps tests the boolean operations of two overlapping
// squares, by the areas and bounds of their results.
func TestBooleanOps(t *testing.T) {
	p := square(0, 0, 2)
	q := square(1, 1, 2)
	tests := []struct {
		name   string
		result ppath.Path
		area   float32
		bounds math32.Box2
	}{
		{"and", And(p, q), 1, math32.B2(1, 1, 2, 2)},
		{"or", Or(p, q), 7, math32.B2(0, 0, 3, 3)},
		{"xor", Xor(p, q), 6, math32.B2(0, 0, 3, 3)},
		{"not", Not(p, q), 3, math32.B2(0, 0, 2, 2)},
		{"div", DivideBy(p, q), 4, math32.B2(0, 0, 2, 2)},
		{"and-paths", AndPaths(p.Split(), q.Split()), 1, math32.B2(1, 1, 2, 2)},
		{"or-paths", OrPaths(p.Split(), q.Split()), 7, math32.B2(0, 0, 3, 3)},
		{"xor-paths", XorPaths(p.Split(), q.Split()), 6, math32.B2(0, 0, 3, 3)},
		{"not-paths", NotPaths(p.Split(), q.Split()), 3, math32.B2(0, 0, 2, 2)},
		{"div-paths", DivideByPaths(p.Split(), q.Split()), 4, math32.B2(0, 0, 2, 2)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.area, area(tt.result), 1e-4)
			tolEqualBox2(t, tt.result.FastBounds(), tt.bounds)
		})
	}
	assert.Len(t, DivideBy(p, q).Split(), 2, "divided into the part inside and the part outside")
	assert.Len(t, Xor(p, q).Split(), 2)
}

// TestBooleanDisjoint tests boolean operations of shapes that do not
// overlap, and of a shape that contains another, making a hole.
func TestBooleanDisjoint(t *testing.T) {
	p := square(0, 0, 1)
	q := square(5, 5, 1)
	assert.True(t, And(p, q).Empty())
	assert.InDelta(t, 2, area(Or(p, q)), 1e-4)
	assert.Len(t, Or(p, q).Split(), 2)
	assert.InDelta(t, 1, area(Not(p, q)), 1e-4)

	outer := square(0, 0, 4)
	inner := square(1, 1, 2)
	hole := Not(outer, inner)
	assert.InDelta(t, 12, area(hole), 1e-4, "the hole is subtracted")
	assert.True(t, Contains(hole, 0.5, 0.5, ppath.NonZero))
	assert.False(t, Contains(hole, 2, 2, ppath.NonZero), "in the hole")
	assert.InDelta(t, 4, area(And(outer, inner)), 1e-4)
	assert.InDelta(t, 16, area(Or(outer, inner)), 1e-4)
}

// TestBooleanCurves tests boolean operations of curved shapes, which
// are flattened.
func TestBooleanCurves(t *testing.T) {
	c := *ppath.New().Circle(0, 0, 1)
	sq := square(0, 0, 2)
	a := And(c, sq)
	assert.True(t, IsFlat(a), "the result is flattened")
	assert.False(t, IsFlat(c))
	assert.InDelta(t, math32.Pi/4, area(a), 0.03, "a quarter of the circle, flattened coarsely")
	assert.InDelta(t, math32.Pi+4-math32.Pi/4, area(Or(c, sq)), 0.05)
}

// TestSettle tests settling a self-intersecting path, a figure eight,
// into separately oriented parts, and the fill rules.
func TestSettle(t *testing.T) {
	// a bowtie: two triangles meeting at (1,1)
	bow := *ppath.New().Polygon(math32.Vec2(0, 0), math32.Vec2(2, 2), math32.Vec2(2, 0), math32.Vec2(0, 2))
	s := Settle(bow, ppath.NonZero)
	assert.InDelta(t, 2, area(s), 1e-4)
	assert.Len(t, s.Split(), 2)
	for _, sp := range s.Split() {
		assert.True(t, CCW(sp), "fills are counter clockwise")
	}
	assert.InDelta(t, 2, area(SettlePaths(bow.Split(), ppath.EvenOdd)), 1e-4)

	// two overlapping squares in one path: the overlap counts twice for
	// non-zero, and is a hole for even-odd
	two := square(0, 0, 2).Append(square(1, 1, 2))
	assert.InDelta(t, 7, area(Settle(two, ppath.NonZero)), 1e-4)
	assert.InDelta(t, 6, area(Settle(two, ppath.EvenOdd)), 1e-4)
}

// TestFillRules tests which windings each fill rule fills.
func TestFillRules(t *testing.T) {
	tests := []struct {
		rule ppath.FillRules
		want [5]bool // windings -2..2
	}{
		{ppath.NonZero, [5]bool{true, true, false, true, true}},
		{ppath.EvenOdd, [5]bool{false, true, false, true, false}},
		{ppath.Positive, [5]bool{false, false, false, true, true}},
		{ppath.Negative, [5]bool{true, true, false, false, false}},
		{ppath.FillRules(10), [5]bool{}},
	}
	for _, tt := range tests {
		for i, w := range []int{-2, -1, 0, 1, 2} {
			assert.Equal(t, tt.want[i], tt.rule.Fills(w), "%v %d", tt.rule, w)
		}
	}
}

// TestContainsFlatten tests containment, flatness, and flattening of
// paths with curves.
func TestContainsFlatten(t *testing.T) {
	c := *ppath.New().Circle(0, 0, 1)
	assert.True(t, Contains(c, 0, 0, ppath.NonZero))
	assert.True(t, Contains(c, 0.7, 0, ppath.NonZero))
	assert.False(t, Contains(c, 0.8, 0.8, ppath.NonZero))
	assert.True(t, Contains(c, 1, 0, ppath.NonZero), "the boundary is contained")

	f := Flatten(c, 0.01)
	assert.True(t, IsFlat(f))
	assert.InDelta(t, math32.Pi, area(f), 0.05)
	assert.InDelta(t, 2*math32.Pi, Length(f), 0.05)
	tolEqualBox2(t, Bounds(f), math32.B2(-1, -1, 1, 1), 0.01)

	q := ppath.New()
	q.MoveTo(0, 0)
	q.QuadTo(1, 2, 2, 0)
	q.CubeTo(3, -2, 4, 2, 5, 0)
	fq := Flatten(*q, 0.01)
	assert.True(t, IsFlat(fq))
	assert.False(t, IsFlat(*q))
	tolEqualVec2(t, fq.Pos(), math32.Vec2(5, 0))
}

// TestCurvature tests the curvature of lines and curves.
func TestCurvature(t *testing.T) {
	line := ppath.New()
	line.MoveTo(0, 0)
	line.LineTo(1, 0)
	assert.Equal(t, float32(0), Curvature(*line, 0, 0.5))
	assert.Equal(t, float32(0), Curvature(*line, 5, 0.5), "no such segment")

	immediate := ppath.ArcToCubeImmediate
	ppath.ArcToCubeImmediate = false // a real arc, not cubics
	defer func() { ppath.ArcToCubeImmediate = immediate }()
	arc := ppath.New()
	arc.MoveTo(2, 0)
	arc.ArcTo(2, 2, 0, false, true, -2, 0)
	assert.InDelta(t, 0.5, math32.Abs(Curvature(*arc, 1, 0.5)), 1e-3, "1/r")
}
