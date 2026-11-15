// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stroke

import (
	"fmt"
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFromStyle tests the cappers and joiners of the styles, and their names.
func TestFromStyle(t *testing.T) {
	caps := map[ppath.Caps]string{ppath.CapButt: "Butt", ppath.CapRound: "Round", ppath.CapSquare: "Square", ppath.Caps(99): "Butt"}
	for c, name := range caps {
		assert.Equal(t, name, CapFromStyle(c).(fmt.Stringer).String())
	}
	joins := map[ppath.Joins]string{ppath.JoinMiter: "Miter", ppath.JoinMiterClip: "MiterClip", ppath.JoinRound: "Round",
		ppath.JoinBevel: "Bevel", ppath.JoinArcs: "Arcs", ppath.JoinArcsClip: "ArcsClip", ppath.Joins(99): "Miter"}
	for j, name := range joins {
		assert.Equal(t, name, JoinFromStyle(j).(fmt.Stringer).String())
	}
}

// strokeBounds returns the bounds of the stroke of the given path, after
// checking that it has no NaN values.
func strokeBounds(t *testing.T, path string, w float32, cr Capper, jr Joiner) math32.Box2 {
	t.Helper()
	s := Stroke(ppath.MustParseSVGPath(path), w, cr, jr, 0.1)
	require.False(t, s.Empty(), path)
	for _, v := range s {
		require.False(t, math32.IsNaN(v), "%s: %v", path, s)
	}
	return s.FastBounds()
}

// assertBox asserts the bounds within the given tolerance.
func assertBox(t *testing.T, want, got math32.Box2, tol float32, msg string) {
	t.Helper()
	assert.InDelta(t, want.Min.X, got.Min.X, float64(tol), "%s min x: %v", msg, got)
	assert.InDelta(t, want.Min.Y, got.Min.Y, float64(tol), "%s min y: %v", msg, got)
	assert.InDelta(t, want.Max.X, got.Max.X, float64(tol), "%s max x: %v", msg, got)
	assert.InDelta(t, want.Max.Y, got.Max.Y, float64(tol), "%s max y: %v", msg, got)
}

// TestRoundMiterJoins tests the extent of round and miter joins, bending
// each way, and the clipping of miters at their limit.
func TestRoundMiterJoins(t *testing.T) {
	box := func(x0, y0, x1, y1 float32) math32.Box2 { return math32.B2(x0, y0, x1, y1) }
	// a round join reaches out by the half width around the corner
	assertBox(t, box(0, -1, 11, 10), strokeBounds(t, "L10 0L10 10", 2, ButtCap, RoundJoin), 0.01, "round cw")
	assertBox(t, box(0, -10, 11, 1), strokeBounds(t, "L10 0L10 -10", 2, ButtCap, RoundJoin), 0.01, "round ccw")
	assertBox(t, box(-1, -1, 11, 11), strokeBounds(t, "L10 0L10 10", 2, RoundCap, RoundJoin), 0.01, "round caps")

	// a sharp turn: the miter is beyond the limit, so it is a bevel, or clipped
	sharp := "L10 0L0 2"
	bevel := strokeBounds(t, sharp, 2, ButtCap, MiterJoin)
	clip := strokeBounds(t, sharp, 2, ButtCap, MiterClipJoin)
	assert.Less(t, bevel.Max.X, float32(11.1), "bevel")
	assert.Greater(t, clip.Max.X, bevel.Max.X, "the clipped miter extends further")
	assert.LessOrEqual(t, clip.Max.X, float32(10+4+0.2), "but only to about the limit (of the miter direction)")
	clipCCW := strokeBounds(t, "L10 0L0 -2", 2, ButtCap, MiterClipJoin)
	assert.InDelta(t, clip.Max.X, clipCCW.Max.X, 0.01, "the same the other way")
	// a 90 degree miter is within the limit
	assertBox(t, box(0, -1, 11, 10), strokeBounds(t, "L10 0L10 10", 2, ButtCap, MiterJoin), 0.01, "miter")
	// a turn back is a bevel
	assertBox(t, box(0, -1, 10, 1), strokeBounds(t, "L10 0L5 0", 2, ButtCap, MiterClipJoin), 0.01, "turn back")
}

// TestArcsJoins tests the arcs joins of curves and lines, and their
// clipping at the limit, which are within the extent of the miter.
func TestArcsJoins(t *testing.T) {
	paths := []string{
		"C5 0 10 5 10 10C15 10 20 15 20 20", // curve to curve
		"C5 0 10 5 10 10C5 10 0 15 0 20",    // the other way
		"L10 0C10 5 15 10 20 10",            // line to curve
		"L10 0C10 -5 15 -10 20 -10",         // the other way
		"C5 0 10 5 10 10L20 10",             // curve to line
		"C5 0 10 -5 10 -10L20 -10",          // the other way
		"C0 10 10 10 10 0C10 10 20 10 20 0", // a sharp turn
		"Q10 0 10 10L10 0",                  // a turn back
	}
	joiners := []Joiner{ArcsJoin, ArcsClipJoin, ArcsJoiner{BevelJoin, 1}, ArcsJoiner{nil, 1}, ArcsJoiner{nil, 1.5}}
	for _, path := range paths {
		pb := ppath.MustParseSVGPath(path).FastBounds()
		for _, jr := range joiners {
			b := strokeBounds(t, path, 2, ButtCap, jr)
			msg := fmt.Sprintf("%s %v", path, jr)
			// the stroke contains the path and is within the miter limit
			assert.LessOrEqual(t, b.Min.X, pb.Min.X+0.01, msg)
			assert.GreaterOrEqual(t, b.Max.X, pb.Max.X-0.01, msg)
			assert.GreaterOrEqual(t, b.Min.X, pb.Min.X-5, msg)
			assert.LessOrEqual(t, b.Max.Y, pb.Max.Y+5, msg)
		}
	}
	// a closed curved path joins its end to its start
	b := strokeBounds(t, "C10 0 10 10 0 10C-10 10 -10 0 0 0z", 2, ButtCap, ArcsJoin)
	assert.Less(t, b.Min.X, float32(-7))
}

// TestClosestArcIntersection tests choosing the intersection that is
// reached first along an arc, in either direction.
func TestClosestArcIntersection(t *testing.T) {
	c := math32.Vector2{}
	pivot := math32.Vec2(1, 0)
	i0 := math32.Vec2(0, 1)  // 90 degrees ccw
	i1 := math32.Vec2(0, -1) // 90 degrees cw
	assert.Equal(t, i0, closestArcIntersection(c, false, pivot, i0, math32.Vec2(-1, 0.01)))
	assert.Equal(t, i1, closestArcIntersection(c, true, pivot, math32.Vec2(-1, 0.01), i1))
}

// TestCheckDash tests whether a dash pattern dashes or strokes a path.
func TestCheckDash(t *testing.T) {
	p := ppath.MustParseSVGPath("L10 0")
	d, stroke := checkDash(p, 0, nil)
	assert.Empty(t, d)
	assert.True(t, stroke, "no dashes")
	d, stroke = checkDash(p, 0, []float32{0})
	assert.Empty(t, d)
	assert.False(t, stroke, "dashes of no length")
	d, stroke = checkDash(p, 0, []float32{20, 5})
	assert.Empty(t, d)
	assert.True(t, stroke, "the first dash covers the path")
	d, stroke = checkDash(p, 20, []float32{20, 15})
	assert.Empty(t, d)
	assert.False(t, stroke, "the first space covers the path")
	d, stroke = checkDash(p, 0, []float32{2, 1})
	assert.Equal(t, []float32{2, 1}, d)
	assert.True(t, stroke)
}
