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

// seg returns the path segment of the given command and values.
func seg(cmd float32, vals ...float32) ppath.Path {
	p := ppath.Path{cmd}
	p = append(p, vals...)
	return append(p, cmd)
}

// arcSeg returns an arc segment from the origin to the given end point,
// as an arc rather than the cubic Béziers of [ppath.ArcToCubeImmediate].
func arcSeg(rx, ry float32, large, sweep bool, x, y float32) ppath.Path {
	defer withArcs()()
	p := ppath.New()
	p.MoveTo(0, 0)
	p.ArcTo(rx, ry, 0, large, sweep, x, y)
	return (*p)[ppath.CmdLen(ppath.MoveTo):]
}

// withArcs makes [ppath.Path.ArcTo] add arcs, until the returned
// function is called.
func withArcs() func() {
	imm := ppath.ArcToCubeImmediate
	ppath.ArcToCubeImmediate = false
	return func() { ppath.ArcToCubeImmediate = imm }
}

// TestIntersectionSegment tests the intersections of the segments of
// each kind with a line, and of two arcs.
func TestIntersectionSegment(t *testing.T) {
	origin := math32.Vec2(0, 0)
	vline0 := math32.Vec2(1, -2)
	vline := seg(ppath.LineTo, 1, 2) // x = 1 from y = -2 to 2

	zs := IntersectionSegment(nil, origin, seg(ppath.LineTo, 2, 0), vline0, vline)
	require.Len(t, zs, 1, "line-line")
	tolEqualVec2(t, zs[0].Vector2, math32.Vec2(1, 0))
	assert.InDelta(t, 0.5, zs[0].T[0], 1e-5)
	assert.InDelta(t, 0.5, zs[0].T[1], 1e-5)
	assert.True(t, zs.Has())
	assert.True(t, zs.HasSecant())
	assert.False(t, zs.HasTangent())
	assert.True(t, zs[0].Equals(zs[0]))
	assert.NotEmpty(t, zs[0].String())

	quad := seg(ppath.QuadTo, 1, 2, 2, 0) // from the origin
	zs = IntersectionSegment(nil, vline0, vline, origin, quad)
	require.Len(t, zs, 1, "line-quad")
	tolEqualVec2(t, zs[0].Vector2, math32.Vec2(1, 1))
	swapped := IntersectionSegment(nil, origin, quad, vline0, vline)
	require.Len(t, swapped, 1, "quad-line")
	assert.Equal(t, zs[0].T[0], swapped[0].T[1], "the times are swapped")

	cube := seg(ppath.CubeTo, 0, 2, 2, 2, 2, 0)
	zs = IntersectionSegment(nil, vline0, vline, origin, cube)
	require.Len(t, zs, 1, "line-cube")
	tolEqualVec2(t, zs[0].Vector2, math32.Vec2(1, 1.5))
	assert.Len(t, IntersectionSegment(nil, origin, cube, vline0, vline), 1, "cube-line")

	// a half circle from (0,0) to (2,0) through (1,-1) or (1,1)
	arc := arcSeg(1, 1, false, true, 2, 0)
	zs = IntersectionSegment(nil, vline0, vline, origin, arc)
	require.Len(t, zs, 1, "line-arc")
	assert.InDelta(t, 1, math32.Abs(zs[0].Y), 1e-4)
	assert.Len(t, IntersectionSegment(nil, origin, arc, vline0, vline), 1, "arc-line")

	// a horizontal line through the half circle crosses it twice
	zs = IntersectionSegment(nil, math32.Vec2(-1, zs[0].Y/2), seg(ppath.LineTo, 3, zs[0].Y/2), origin, arc)
	assert.Len(t, zs, 2, "a line crossing an arc twice")

	// two half circles of radius 1, centered at (1,0) and (2,0)
	arc2 := arcSeg(1, 1, false, true, 2, 0)
	arc2[5] = 3 // the end x, for the start at (1,0)
	zs = IntersectionSegment(nil, origin, arc, math32.Vec2(1, 0), arc2)
	require.Len(t, zs, 1, "arc-arc")
	tolEqualVec2(t, zs[0].Vector2, math32.Vec2(1.5, -math32.Sqrt(3)/2))
	zs = IntersectionSegment(nil, origin, arc, origin, arc)
	assert.Len(t, zs, 2, "the same arc overlaps from start to end")
	assert.True(t, zs[0].Same)

	assert.Empty(t, IntersectionSegment(nil, origin, seg(ppath.LineTo, 0.5, 0), vline0, vline), "too short")
	assert.Panics(t, func() { IntersectionSegment(nil, origin, quad, origin, quad) }, "quad-quad is not supported")
}

// TestIntersectionTangent tests that an intersection where a line
// touches another at its end is tangent.
func TestIntersectionTangent(t *testing.T) {
	zs := IntersectionSegment(nil, math32.Vec2(0, 0), seg(ppath.LineTo, 1, 0), math32.Vec2(1, -1), seg(ppath.LineTo, 1, 1))
	require.Len(t, zs, 1)
	assert.True(t, zs[0].Tangent, "at the end of the first line")
	assert.True(t, zs.HasTangent())
	assert.False(t, zs.HasSecant())
	assert.False(t, Intersections(nil).Has())
}

// TestIntersectionRays tests the intersections of rays with lines and
// circles, and of two circles.
func TestIntersectionRays(t *testing.T) {
	p, ok := IntersectionRayLine(math32.Vec2(0, 0), math32.Vec2(1, 0), math32.Vec2(3, -1), math32.Vec2(3, 1))
	assert.True(t, ok)
	tolEqualVec2(t, p, math32.Vec2(3, 0))
	_, ok = IntersectionRayLine(math32.Vec2(0, 0), math32.Vec2(1, 0), math32.Vec2(0, 1), math32.Vec2(1, 1))
	assert.False(t, ok, "parallel")
	_, ok = IntersectionRayLine(math32.Vec2(0, 0), math32.Vec2(1, 0), math32.Vec2(3, 1), math32.Vec2(3, 2))
	assert.False(t, ok, "beyond the line")

	a, b, ok := IntersectionRayCircle(math32.Vec2(-5, 0), math32.Vec2(5, 0), math32.Vec2(0, 0), 2)
	assert.True(t, ok)
	assert.InDelta(t, 2, math32.Abs(a.X), 1e-5)
	assert.InDelta(t, 2, math32.Abs(b.X), 1e-5)
	assert.InDelta(t, 0, a.X+b.X, 1e-5)
	_, _, ok = IntersectionRayCircle(math32.Vec2(-5, 3), math32.Vec2(5, 3), math32.Vec2(0, 0), 2)
	assert.False(t, ok, "misses")

	a, b, ok = IntersectionCircleCircle(math32.Vec2(0, 0), 1, math32.Vec2(1, 0), 1)
	assert.True(t, ok)
	assert.InDelta(t, 0.5, a.X, 1e-5)
	assert.InDelta(t, 0.5, b.X, 1e-5)
	assert.InDelta(t, math32.Sqrt(3)/2, math32.Abs(a.Y), 1e-5)
	_, _, ok = IntersectionCircleCircle(math32.Vec2(0, 0), 1, math32.Vec2(5, 0), 1)
	assert.False(t, ok, "too far apart")
	_, _, ok = IntersectionCircleCircle(math32.Vec2(0, 0), 1, math32.Vec2(0, 0), 2)
	assert.False(t, ok, "concentric")
}

// TestRayIntersectionsCurves tests the intersections of a ray with a
// path of curves.
func TestRayIntersectionsCurves(t *testing.T) {
	defer withArcs()()
	p := ppath.New()
	p.MoveTo(0, 0)
	p.QuadTo(2, 4, 4, 0)
	p.CubeTo(5, -3, 7, -3, 8, 0)
	p.ArcTo(2, 2, 0, false, false, 4, 0)
	p.Close()
	zs := RayIntersections(*p, -1, 1)
	assert.Len(t, zs, 2, "through the quad")
	for i := 1; i < len(zs); i++ {
		assert.LessOrEqual(t, zs[i-1].X, zs[i].X, "sorted along the ray")
	}
	assert.Len(t, RayIntersections(*p, -1, -1), 4, "through the cube and the arc")
	assert.Len(t, RayIntersections(*p, -1, -2.1), 2, "through the cube, below the arc")
	assert.Empty(t, RayIntersections(*p, -1, 10))
}
