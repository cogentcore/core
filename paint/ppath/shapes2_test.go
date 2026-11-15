// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ppath

import (
	"testing"

	"cogentcore.org/core/base/goscript"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/sides"
	"github.com/stretchr/testify/assert"
)

// TestShapes tests the coordinates and bounds of the basic shapes.
func TestShapes(t *testing.T) {
	assert.Equal(t, "M1 2L3 4", New().Line(1, 2, 3, 4).String())
	assert.True(t, New().Line(1, 2, 1, 2).Empty(), "a zero-length line")

	pts := []math32.Vector2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}}
	assert.Equal(t, "M0 0L1 0L1 1", New().Polyline(pts...).String())
	assert.True(t, New().Polyline(pts[0]).Empty())
	assert.Equal(t, "M0 0L1 0L1 1z", New().Polygon(pts...).String())

	r := New().Rectangle(1, 2, 3, 4)
	assert.Equal(t, "M1 2L4 2L4 6L1 6z", r.String())
	assert.True(t, r.Closed())
	assert.True(t, New().Rectangle(0, 0, 0, 4).Empty())

	tolEqualBox2(t, New().RoundedRectangle(0, 0, 4, 2, 0.5).FastBounds(), math32.B2(0, 0, 4, 2))
	assert.Equal(t, r.String(), New().RoundedRectangle(1, 2, 3, 4, 0).String(), "no radius is a rectangle")
	assert.True(t, New().RoundedRectangle(0, 0, 0, 4, 1).Empty())
	concave := New().RoundedRectangle(0, 0, 4, 2, -5) // radius is clamped
	var maxx float32
	for _, c := range concave.Coords() {
		maxx = max(maxx, c.X)
	}
	assert.InDelta(t, 4, maxx, 1e-5)

	rs := New().RoundedRectangleSides(0, 0, 4, 2, sides.NewFloats(0.5, 0, 3, 1))
	tolEqualBox2(t, rs.FastBounds(), math32.B2(0, 0, 4, 2))
	assert.True(t, rs.Closed())

	bev := New().BeveledRectangle(0, 0, 4, 2, -0.5)
	assert.Equal(t, 9, bev.Len(), "a MoveTo, 7 LineTos, and a Close")
	tolEqualBox2(t, bev.FastBounds(), math32.B2(0, 0, 4, 2))
	assert.Equal(t, r.String(), New().BeveledRectangle(1, 2, 3, 4, 0).String())
	assert.True(t, New().BeveledRectangle(0, 0, 4, 0, 1).Empty())

	c := New().Circle(1, 1, 2)
	assert.True(t, c.Closed())
	tolEqualBox2(t, c.FastBounds(), math32.B2(-1, -1, 3, 3), 0.01)
	assert.True(t, New().Ellipse(0, 0, 0, 1).Empty())

	arc := New().CircularArc(0, 0, 1, 0, math32.Pi/2)
	tolEqualVec2(t, arc.StartPos(), math32.Vec2(1, 0))
	tolEqualVec2(t, arc.Pos(), math32.Vec2(0, 1))
	earc := New().EllipticalArc(0, 0, 2, 1, 0, 0, math32.Pi)
	tolEqualVec2(t, earc.Pos(), math32.Vec2(-2, 0))
}

// TestPolygons tests regular and star polygons.
func TestPolygons(t *testing.T) {
	tri := New().Triangle(1)
	assert.Len(t, tri.Coords(), 4, "three points and the close")
	tolEqualVec2(t, tri.StartPos(), math32.Vec2(0, 1), 1e-5)

	down := New().RegularPolygon(4, 1, false)
	tolEqualVec2(t, down.StartPos(), math32.Vec2(-math32.Sqrt(2)/2, math32.Sqrt(2)/2), 1e-5)
	assert.True(t, New().RegularPolygon(2, 1, true).Empty(), "too few points")
	assert.True(t, New().RegularStarPolygon(6, 3, 1, true).Empty(), "n = 2d")

	star := New().RegularStarPolygon(5, 2, 1, true)
	assert.Equal(t, 6, star.Len(), "a pentagram has a MoveTo, 4 LineTos, and a close")

	sp := New().StarPolygon(5, 2, 1, true)
	coords := sp.Coords()
	assert.Len(t, coords, 11)
	for i, c := range coords[:10] {
		want := float32(2)
		if i%2 == 1 {
			want = 1
		}
		assert.InDelta(t, want, c.Length(), 1e-5, "alternating radii")
	}
	tolEqualVec2(t, New().StarPolygon(5, 2, 1, false).StartPos().Normal(), math32.Vec2(math32.Cos(math32.Pi/2+math32.Pi/5), math32.Sin(math32.Pi/2+math32.Pi/5)), 1e-5)
	assert.True(t, New().StarPolygon(2, 2, 1, true).Empty())
}

// TestGrid tests a grid, which has a hole for each cell.
func TestGrid(t *testing.T) {
	g := New().Grid(10, 7, 3, 2, 1)
	assert.Len(t, g.Split(), 1+3*2, "the outline and the cells")
	assert.True(t, New().Grid(1, 1, 3, 3, 1).Empty(), "too small for the lines")
	tolEqualBox2(t, g.FastBounds(), math32.B2(0, 0, 10, 7))
	tolEqualBox2(t, g.Split()[2].FastBounds(), math32.B2(4, 1, 6, 3)) // the second cell
}

// TestFastBoundsCube tests that the fast bounds of a cubic Bézier
// include its end point.
func TestFastBoundsCube(t *testing.T) {
	c := MustParseSVGPath("M0 0C1 1 2 1 3 0")
	b := c.FastBounds()
	assert.Equal(t, float32(0), b.Min.Y)
	assert.Equal(t, float32(3), b.Max.X, "the end point")
}

// TestPathAccessors tests the basic queries and copies of paths.
func TestPathAccessors(t *testing.T) {
	goscript.Skip(t, "the identity of the memory of slices")
	defer withArcs()()
	p := MustParseSVGPath("M0 0Q1 2 2 0C3 -1 4 1 5 0A1 1 0 0 1 7 0L7 3z")
	assert.Equal(t, 6, p.Len())
	assert.True(t, p.Sane())
	assert.False(t, p.HasSubpaths())
	assert.False(t, p.PointClosed())
	assert.True(t, Paths{nil, *New()}.Empty())
	assert.False(t, Paths{p}.Empty())

	cp, end := p.QuadToPoints(4)
	tolEqualVec2(t, cp, math32.Vec2(1, 2))
	tolEqualVec2(t, end, math32.Vec2(2, 0))
	rx, ry, phi, large, sweep, aend := p.ArcToPoints(4 + CmdLen(QuadTo) + CmdLen(CubeTo))
	assert.Equal(t, []float32{1, 1, 0}, []float32{rx, ry, phi})
	assert.False(t, large)
	assert.True(t, sweep)
	tolEqualVec2(t, aend, math32.Vec2(7, 0))

	q := p.CopyTo(nil)
	assert.True(t, q.Equals(p))
	big := make(Path, 100)
	q = p.CopyTo(big)
	assert.True(t, q.Equals(p))
	assert.Equal(t, &big[0], &q[0], "uses the given memory")

	q.Reset()
	assert.True(t, q.Empty())

	bad := MustParseSVGPath("M0 0L1 1")
	bad[5] = math32.NaN() // the x of the LineTo
	assert.False(t, bad.Sane())
	inf := MustParseSVGPath("M0 0Q1 1 2 2")
	inf[5] = math32.Inf(1)
	assert.False(t, inf.Sane())

	pc := Path{MoveTo, 0, 0, MoveTo, LineTo, 1, 0, LineTo, LineTo, 0, 0, LineTo, Close, 0, 0, Close}
	assert.True(t, pc.PointClosed())
	assert.True(t, MustParseSVGPath("M0 0L1 0M2 2L3 3").HasSubpaths())

	for _, fl := range [][2]bool{{false, false}, {false, true}, {true, false}, {true, true}} {
		l, s := ToArcFlags(fromArcFlags(fl[0], fl[1]))
		assert.Equal(t, fl, [2]bool{l, s})
	}
}

// TestScanners tests scanning the commands of a path forward and backward.
func TestScanners(t *testing.T) {
	defer withArcs()()
	p := MustParseSVGPath("M1 1L2 1Q3 2 4 1C5 0 6 2 7 1A1 1 0 0 1 9 1z")
	var cmds []float32
	s := p.Scanner()
	for s.Scan() {
		cmds = append(cmds, s.Cmd())
		switch s.Cmd() {
		case MoveTo:
			assert.Equal(t, math32.Vec2(0, 0), s.Start())
			assert.Equal(t, 0, s.Index())
		case QuadTo:
			assert.Equal(t, math32.Vec2(3, 2), s.CP1())
			assert.Equal(t, math32.Vec2(2, 1), s.Start())
			assert.Equal(t, []float32{3, 2, 4, 1}, s.Values())
			assert.True(t, s.Path().Equals(MustParseSVGPath("M2 1Q3 2 4 1")))
			assert.Panics(t, func() { s.CP2() })
			assert.Panics(t, func() { s.Arc() })
		case CubeTo:
			assert.Equal(t, math32.Vec2(5, 0), s.CP1())
			assert.Equal(t, math32.Vec2(6, 2), s.CP2())
			assert.True(t, s.Path().Equals(MustParseSVGPath("M4 1C5 0 6 2 7 1")))
		case ArcTo:
			rx, ry, _, large, sweep := s.Arc()
			assert.Equal(t, []any{float32(1), float32(1), false, true}, []any{rx, ry, large, sweep})
			assert.Equal(t, math32.Vec2(9, 1), s.End())
			assert.Equal(t, CmdLen(ArcTo), len(s.Path())-CmdLen(MoveTo))
		case LineTo:
			assert.True(t, s.Path().Equals(MustParseSVGPath("M1 1L2 1")))
			assert.Panics(t, func() { s.CP1() })
		}
	}
	assert.Equal(t, []float32{MoveTo, LineTo, QuadTo, CubeTo, ArcTo, Close}, cmds)

	var rcmds []float32
	rs := p.ReverseScanner()
	for rs.Scan() {
		rcmds = append(rcmds, rs.Cmd())
		switch rs.Cmd() {
		case QuadTo:
			assert.Equal(t, math32.Vec2(3, 2), rs.CP1())
			assert.Equal(t, math32.Vec2(2, 1), rs.Start())
			assert.Equal(t, math32.Vec2(4, 1), rs.End())
			assert.Equal(t, []float32{3, 2, 4, 1}, rs.Values())
			assert.True(t, rs.Path().Equals(MustParseSVGPath("M2 1Q3 2 4 1")))
		case CubeTo:
			assert.Equal(t, math32.Vec2(6, 2), rs.CP2())
			assert.True(t, rs.Path().Equals(MustParseSVGPath("M4 1C5 0 6 2 7 1")))
		case ArcTo:
			rx, _, _, _, sweep := rs.Arc()
			assert.Equal(t, float32(1), rx)
			assert.True(t, sweep)
			assert.NotEmpty(t, rs.Path())
		case LineTo:
			assert.True(t, rs.Path().Equals(MustParseSVGPath("M1 1L2 1")))
		case MoveTo:
			assert.Equal(t, math32.Vec2(0, 0), rs.Start())
		}
	}
	assert.Equal(t, []float32{Close, ArcTo, CubeTo, QuadTo, LineTo, MoveTo}, rcmds)
}

// withArcs makes [Path.ArcTo] add arcs, until the returned function is
// called.
func withArcs() func() {
	imm := ArcToCubeImmediate
	ArcToCubeImmediate = false
	return func() { ArcToCubeImmediate = imm }
}

// TestTranslateScale tests translating and scaling paths, including
// arcs, whose radii scale.
func TestTranslateScale(t *testing.T) {
	defer withArcs()()
	r := New().Rectangle(0, 0, 1, 2)
	tolEqualBox2(t, r.Clone().Translate(3, 4).FastBounds(), math32.B2(3, 4, 4, 6))
	tolEqualBox2(t, r.Clone().Scale(2, 3).FastBounds(), math32.B2(0, 0, 2, 6))

	a := New()
	a.MoveTo(0, 0)
	a.ArcTo(1, 1, 0, false, true, 2, 0)
	s := a.Clone().Scale(2, 2)
	rx, ry, _, _, _, end := s.ArcToPoints(CmdLen(MoveTo))
	assert.InDelta(t, 2, rx, 1e-5)
	assert.InDelta(t, 2, ry, 1e-5)
	tolEqualVec2(t, end, math32.Vec2(4, 0))

	// non-uniform scale makes an ellipse
	e := a.Clone().Scale(3, 1)
	rx, ry, _, _, _, end = e.ArcToPoints(CmdLen(MoveTo))
	assert.InDelta(t, 3, rx, 1e-4)
	assert.InDelta(t, 1, ry, 1e-4)
	tolEqualVec2(t, end, math32.Vec2(6, 0))

	// a flip reverses the sweep
	f := a.Clone().Scale(1, -1)
	_, _, _, _, sweep, _ := f.ArcToPoints(CmdLen(MoveTo))
	assert.False(t, sweep)

	// a rotation rotates the arc
	rot := a.Clone().Transform(math32.Rotate2D(math32.Pi / 2))
	_, _, _, _, _, end = rot.ArcToPoints(CmdLen(MoveTo))
	tolEqualVec2(t, end, math32.Vec2(0, 2))

	// quads and cubes transform their control points
	qc := MustParseSVGPath("M0 0Q1 1 2 0C3 1 4 1 5 0").Translate(1, 1)
	assert.True(t, qc.Equals(MustParseSVGPath("M1 1Q2 2 3 1C4 2 5 2 6 1")))
}

// TestDirections tests the directions along paths.
func TestDirections(t *testing.T) {
	sq := MustParseSVGPath("M0 0L1 0L1 1L0 1z")
	dirs := sq.CoordDirections()
	assert.Len(t, dirs, 5)
	tolEqualVec2(t, dirs[1], math32.Vec2(1, 1).Normal(), 1e-5)
	tolEqualVec2(t, dirs[0], dirs[4])

	open := MustParseSVGPath("M0 0L1 0L1 1")
	dirs = open.CoordDirections()
	assert.Len(t, dirs, 3)
	tolEqualVec2(t, dirs[2], math32.Vec2(0, 1))
	assert.Len(t, New().CoordDirections(), 1)

	tolEqualVec2(t, DirectionIndex(open, 4, 0.5), math32.Vec2(1, 0))
	tolEqualVec2(t, DirectionIndex(open, 0, 0), math32.Vec2(1, 0), 1e-5)
	curves := MustParseSVGPath("M0 0Q1 1 2 0C2 1 3 1 3 0")
	tolEqualVec2(t, DirectionIndex(curves, 4, 0), math32.Vec2(1, 1).Normal(), 1e-5)
	tolEqualVec2(t, DirectionIndex(curves, 4+CmdLen(QuadTo), 0), math32.Vec2(0, 1), 1e-5)

	assert.Equal(t, math32.Vector2{}, New().Direction(0, 0))
	// segments are the commands, including the MoveTo
	tolEqualVec2(t, open.Direction(0, 0), math32.Vec2(1, 0), 1e-5)
	tolEqualVec2(t, open.Direction(1, 0.5), math32.Vec2(1, 0))
	tolEqualVec2(t, open.Direction(2, 0.5), math32.Vec2(0, 1))
	two := MustParseSVGPath("M0 0L1 0M5 5L5 6")
	tolEqualVec2(t, two.Direction(3, 0.5), math32.Vec2(0, 1), 1e-5) // in the second subpath
	assert.Equal(t, math32.Vector2{}, open.Direction(3, 0.5), "no such segment")
}

// TestDashesMath tests scaling dashes, and the angle helpers.
func TestDashesMath(t *testing.T) {
	off, d := ScaleDash(2, 1, Dashdotted)
	assert.Equal(t, float32(2), off)
	assert.Equal(t, []float32{6, 4, 2, 4}, d)
	assert.Equal(t, []float32{3, 2, 1, 2}, Dashdotted, "not changed")

	assert.Equal(t, float32(2), Slope(math32.Vec2(1, 2)))
	assert.InDelta(t, 3*math32.Pi/2, Angle(math32.Vec2(0, -1)), 1e-6)
	assert.InDelta(t, math32.Pi/2, AngleBetween(math32.Vec2(1, 0), math32.Vec2(0, 1)), 1e-6)

	assert.Equal(t, "evenodd", EvenOdd.String())
	assert.Equal(t, "round", CapRound.String())
	assert.Equal(t, "miter-clip", JoinMiterClip.String())
	assert.Equal(t, "non-scaling-stroke", VectorEffectNonScalingStroke.String())
}

// TestPathString tests the string form of each kind of command.
func TestPathString(t *testing.T) {
	defer withArcs()()
	s := "M0 0L1 0Q2 1 3 0C4 1 5 1 6 0A2 1 90 1 0 8 0z"
	assert.Equal(t, s, MustParseSVGPath(s).String())
	// the larger radius is first, rotated by 90 degrees
	assert.Equal(t, "M0 0A2 1 90 0 0 0 4", MustParseSVGPath("M0 0A1 2 0 0 0 0 4").String())
	s = "M0 0A1 1 0 0 1 2 0"
	assert.Equal(t, s, MustParseSVGPath(s).String())
}

// TestMinifyNumber tests minifying numbers to a precision.
func TestMinifyNumber(t *testing.T) {
	tests := []struct {
		in   string
		prec int
		want string
	}{
		{"0", 0, "0"},
		{"1.000", 0, "1"},
		{"+0.500", 0, ".5"},
		{"-0.500", 0, "-.5"},
		{"0012.3400", 0, "12.34"},
		{"1000000", 0, "1e6"},
		{"1.5e3", 0, "1500"},
		{"1.5E+3", 0, "1500"},
		{"1e", 0, "1e"},
		{"3.14159", 3, "3.14"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, string(MinifyNumber([]byte(tt.in), tt.prec)), "%s %d", tt.in, tt.prec)
	}
	assert.Equal(t, "1e-6", string(MinifyNumber([]byte("0.000001"), 0)))
}

// TestFills tests which windings each fill rule fills.
func TestFills(t *testing.T) {
	assert.True(t, NonZero.Fills(-1))
	assert.False(t, NonZero.Fills(0))
	assert.True(t, EvenOdd.Fills(3))
	assert.False(t, EvenOdd.Fills(2))
	assert.True(t, Positive.Fills(1))
	assert.False(t, Positive.Fills(-1))
	assert.True(t, Negative.Fills(-1))
	assert.False(t, Negative.Fills(1))
	assert.False(t, FillRules(9).Fills(1))
}
