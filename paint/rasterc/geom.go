// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package raster

import (
	"fmt"

	"cogentcore.org/core/math32"
	"golang.org/x/image/math/fixed"
)

// Invert returns the point inverted around the origin
func Invert(v fixed.Point26_6) fixed.Point26_6 {
	return fixed.Point26_6{X: -v.X, Y: -v.Y}
}

// TurnStarboard90 returns the vector 90 degrees starboard (right in direction heading)
func TurnStarboard90(v fixed.Point26_6) fixed.Point26_6 {
	return fixed.Point26_6{X: -v.Y, Y: v.X}
}

// TurnPort90 returns the vector 90 degrees port (left in direction heading)
func TurnPort90(v fixed.Point26_6) fixed.Point26_6 {
	return fixed.Point26_6{X: v.Y, Y: -v.X}
}

// DotProd returns the inner product of p and q
func DotProd(p fixed.Point26_6, q fixed.Point26_6) fixed.Int52_12 {
	return fixed.Int52_12(int64(p.X)*int64(q.X) + int64(p.Y)*int64(q.Y))
}

// Length is the distance from the origin of the point
func Length(v fixed.Point26_6) fixed.Int26_6 {
	vx, vy := float32(v.X), float32(v.Y)
	return fixed.Int26_6(math32.Sqrt(vx*vx + vy*vy))
}

// PathCommand is the type for the path command token
type PathCommand fixed.Int26_6 //enums:enum -no-extend

// Human readable path command constants
const (
	PathMoveTo PathCommand = iota
	PathLineTo
	PathQuadTo
	PathCubicTo
	PathClose
)

// A Path starts with a PathCommand value followed by zero to three fixed
// int points.
type Path []fixed.Int26_6

// ToSVGPath returns a string representation of the path
func (p Path) ToSVGPath() string {
	s := ""
	for i := 0; i < len(p); {
		if i != 0 {
			s += " "
		}
		switch PathCommand(p[i]) {
		case PathMoveTo:
			s += fmt.Sprintf("M%4.3f,%4.3f", float32(p[i+1])/64, float32(p[i+2])/64)
			i += 3
		case PathLineTo:
			s += fmt.Sprintf("L%4.3f,%4.3f", float32(p[i+1])/64, float32(p[i+2])/64)
			i += 3
		case PathQuadTo:
			s += fmt.Sprintf("Q%4.3f,%4.3f,%4.3f,%4.3f", float32(p[i+1])/64, float32(p[i+2])/64,
				float32(p[i+3])/64, float32(p[i+4])/64)
			i += 5
		case PathCubicTo:
			s += "C" + fmt.Sprintf("C%4.3f,%4.3f,%4.3f,%4.3f,%4.3f,%4.3f", float32(p[i+1])/64, float32(p[i+2])/64,
				float32(p[i+3])/64, float32(p[i+4])/64, float32(p[i+5])/64, float32(p[i+6])/64)
			i += 7
		case PathClose:
			s += "Z"
			i++
		default:
			panic("freetype/rasterx: bad pather")
		}
	}
	return s
}

// String returns a readable representation of a Path.
func (p Path) String() string {
	return p.ToSVGPath()
}

// Clear zeros the path slice
func (p *Path) Clear() {
	*p = (*p)[:0]
}

// Start starts a new curve at the given point.
func (p *Path) Start(a fixed.Point26_6) {
	*p = append(*p, fixed.Int26_6(PathMoveTo), a.X, a.Y)
}

// Line adds a linear segment to the current curve.
func (p *Path) Line(b fixed.Point26_6) {
	*p = append(*p, fixed.Int26_6(PathLineTo), b.X, b.Y)
}

// QuadBezier adds a quadratic segment to the current curve.
func (p *Path) QuadBezier(b, c fixed.Point26_6) {
	*p = append(*p, fixed.Int26_6(PathQuadTo), b.X, b.Y, c.X, c.Y)
}

// CubeBezier adds a cubic segment to the current curve.
func (p *Path) CubeBezier(b, c, d fixed.Point26_6) {
	*p = append(*p, fixed.Int26_6(PathCubicTo), b.X, b.Y, c.X, c.Y, d.X, d.Y)
}

// Stop joins the ends of the path
func (p *Path) Stop(closeLoop bool) {
	if closeLoop {
		*p = append(*p, fixed.Int26_6(PathClose))
	}
}

// AddTo adds the Path p to q.
func (p Path) AddTo(q Adder) {
	for i := 0; i < len(p); {
		switch PathCommand(p[i]) {
		case PathMoveTo:
			q.Stop(false) // Fixes issues #1 by described by Djadala; implicit close if currently in path.
			q.Start(fixed.Point26_6{X: p[i+1], Y: p[i+2]})
			i += 3
		case PathLineTo:
			q.Line(fixed.Point26_6{X: p[i+1], Y: p[i+2]})
			i += 3
		case PathQuadTo:
			q.QuadBezier(fixed.Point26_6{X: p[i+1], Y: p[i+2]}, fixed.Point26_6{X: p[i+3], Y: p[i+4]})
			i += 5
		case PathCubicTo:
			q.CubeBezier(fixed.Point26_6{X: p[i+1], Y: p[i+2]},
				fixed.Point26_6{X: p[i+3], Y: p[i+4]}, fixed.Point26_6{X: p[i+5], Y: p[i+6]})
			i += 7
		case PathClose:
			q.Stop(true)
			i++
		default:
			panic("AddTo: bad path")
		}
	}
	q.Stop(false)
}

// ToLength scales the point to the length indicated by ln
func ToLength(p fixed.Point26_6, ln fixed.Int26_6) (q fixed.Point26_6) {
	if ln == 0 || (p.X == 0 && p.Y == 0) {
		return
	}

	pX, pY := float32(p.X), float32(p.Y)
	lnF := float32(ln)
	pLen := math32.Sqrt(pX*pX + pY*pY)

	qX, qY := pX*lnF/pLen, pY*lnF/pLen
	q.X, q.Y = fixed.Int26_6(qX), fixed.Int26_6(qY)
	return
}

// ClosestPortside returns the closest of p1 or p2 on the port side of the
// line from the bow to the stern. (port means left side of the direction you are heading)
// isIntersecting is just convienice to reduce code, and if false returns false, because p1 and p2 are not valid
func ClosestPortside(bow, stern, p1, p2 fixed.Point26_6, isIntersecting bool) (xt fixed.Point26_6, intersects bool) {
	if !isIntersecting {
		return
	}
	dir := bow.Sub(stern)
	dp1 := p1.Sub(stern)
	dp2 := p2.Sub(stern)
	cp1 := dir.X*dp1.Y - dp1.X*dir.Y
	cp2 := dir.X*dp2.Y - dp2.X*dir.Y
	switch {
	case cp1 < 0 && cp2 < 0:
		return
	case cp1 < 0 && cp2 >= 0:
		return p2, true
	case cp1 >= 0 && cp2 < 0:
		return p1, true
	default: // both points on port side
		dirdot := DotProd(dir, dir)
		// calculate vector rejections of dp1 and dp2 onto dir
		h1 := dp1.Sub(dir.Mul(fixed.Int26_6((DotProd(dp1, dir) << 6) / dirdot)))
		h2 := dp2.Sub(dir.Mul(fixed.Int26_6((DotProd(dp2, dir) << 6) / dirdot)))
		// return point with smallest vector rejection; i.e. closest to dir line
		if (h1.X*h1.X + h1.Y*h1.Y) > (h2.X*h2.X + h2.Y*h2.Y) {
			return p2, true
		}
		return p1, true
	}
}

// RadCurvature returns the curvature of a Bezier curve end point,
// given an end point, the two adjacent control points and the degree.
// The sign of the value indicates if the center of the osculating circle
// is left or right (port or starboard) of the curve in the forward direction.
func RadCurvature(p0, p1, p2 fixed.Point26_6, dm fixed.Int52_12) fixed.Int26_6 {
	a, b := p2.Sub(p1), p1.Sub(p0)
	abdot, bbdot := DotProd(a, b), DotProd(b, b)
	h := a.Sub(b.Mul(fixed.Int26_6((abdot << 6) / bbdot))) // h is the vector rejection of a onto b
	if h.X == 0 && h.Y == 0 {                              // points are co-linear
		return 0
	}
	radCurve := fixed.Int26_6((fixed.Int52_12(a.X*a.X+a.Y*a.Y) * dm / fixed.Int52_12(Length(h)<<6)) >> 6)
	if a.X*b.Y > b.X*a.Y { // xprod sign
		return radCurve
	}
	return -radCurve
}

// CircleCircleIntersection calculates the points of intersection of
// two circles or returns with intersects == false if no such points exist.
func CircleCircleIntersection(ct, cl fixed.Point26_6, rt, rl fixed.Int26_6) (xt1, xt2 fixed.Point26_6, intersects bool) {
	dc := cl.Sub(ct)
	d := Length(dc)

	// Check for solvability.
	if d > (rt + rl) {
		return // No solution. Circles do not intersect.
	}
	// check if  d < abs(rt-rl)
	if da := rt - rl; (da > 0 && d < da) || (da < 0 && d < -da) {
		return // No solution. One circle is contained by the other.
	}

	rlf, rtf, df := float32(rl), float32(rt), float32(d)
	af := (rtf*rtf - rlf*rlf + df*df) / df / 2.0
	hfd := math32.Sqrt(rtf*rtf-af*af) / df
	afd := af / df

	rOffx, rOffy := float32(-dc.Y)*hfd, float32(dc.X)*hfd
	p2x := float32(ct.X) + float32(dc.X)*afd
	p2y := float32(ct.Y) + float32(dc.Y)*afd
	xt1x, xt1y := p2x+rOffx, p2y+rOffy
	xt2x, xt2y := p2x-rOffx, p2y-rOffy
	return fixed.Point26_6{X: fixed.Int26_6(xt1x), Y: fixed.Int26_6(xt1y)},
		fixed.Point26_6{X: fixed.Int26_6(xt2x), Y: fixed.Int26_6(xt2y)}, true
}

// CalcIntersect calculates the points of intersection of two fixed point lines
// and panics if the determinate is zero. You have been warned.
func CalcIntersect(a1, a2, b1, b2 fixed.Point26_6) (x fixed.Point26_6) {
	da, db, ds := a2.Sub(a1), b2.Sub(b1), a1.Sub(b1)
	det := float32(da.X*db.Y - db.X*da.Y) // Determinate
	t := float32(ds.Y*db.X-ds.X*db.Y) / det
	x = a1.Add(fixed.Point26_6{X: fixed.Int26_6(float32(da.X) * t), Y: fixed.Int26_6(float32(da.Y) * t)})
	return
}

// RayCircleIntersection calculates the points of intersection of
// a ray starting at s2 passing through s1 and a circle in fixed point.
// Returns intersects == false if no solution is possible. If two
// solutions are possible, the point closest to s2 is returned
func RayCircleIntersection(s1, s2, c fixed.Point26_6, r fixed.Int26_6) (x fixed.Point26_6, intersects bool) {
	fx, fy, intersects := RayCircleIntersectionF(float32(s1.X), float32(s1.Y),
		float32(s2.X), float32(s2.Y), float32(c.X), float32(c.Y), float32(r))
	return fixed.Point26_6{X: fixed.Int26_6(fx),
		Y: fixed.Int26_6(fy)}, intersects

}

// RayCircleIntersectionF calculates in floating point the points of intersection of
// a ray starting at s2 passing through s1 and a circle in fixed point.
// Returns intersects == false if no solution is possible. If two
// solutions are possible, the point closest to s2 is returned
func RayCircleIntersectionF(s1X, s1Y, s2X, s2Y, cX, cY, r float32) (x, y float32, intersects bool) {
	n := s2X - cX // Calculating using 64* rather than divide
	m := s2Y - cY

	e := s2X - s1X
	d := s2Y - s1Y

	// Quadratic normal form coefficients
	A, B, C := e*e+d*d, -2*(e*n+m*d), n*n+m*m-r*r

	D := B*B - 4*A*C

	if D <= 0 {
		return // No intersection or is tangent
	}

	D = math32.Sqrt(D)
	t1, t2 := (-B+D)/(2*A), (-B-D)/(2*A)
	p1OnSide := t1 > 0
	p2OnSide := t2 > 0

	switch {
	case p1OnSide && p2OnSide:
		if t2 < t1 { // both on ray, use closest to s2
			t1 = t2
		}
	case p2OnSide: // Only p2 on ray
		t1 = t2
	case p1OnSide: // only p1 on ray
	default: // Neither solution is on the ray
		return
	}
	return (n - e*t1) + cX, (m - d*t1) + cY, true
}

// MatrixAdder is an adder that applies matrix M to all points
type MatrixAdder struct {
	Adder
	M math32.Matrix2
}

// Reset sets the matrix M to identity
func (t *MatrixAdder) Reset() {
	t.M = math32.Identity2()
}

// Start starts a new path
func (t *MatrixAdder) Start(a fixed.Point26_6) {
	t.Adder.Start(t.M.MulFixedAsPoint(a))
}

// Line adds a linear segment to the current curve.
func (t *MatrixAdder) Line(b fixed.Point26_6) {
	t.Adder.Line(t.M.MulFixedAsPoint(b))
}

// QuadBezier adds a quadratic segment to the current curve.
func (t *MatrixAdder) QuadBezier(b, c fixed.Point26_6) {
	t.Adder.QuadBezier(t.M.MulFixedAsPoint(b), t.M.MulFixedAsPoint(c))
}

// CubeBezier adds a cubic segment to the current curve.
func (t *MatrixAdder) CubeBezier(b, c, d fixed.Point26_6) {
	t.Adder.CubeBezier(t.M.MulFixedAsPoint(b), t.M.MulFixedAsPoint(c), t.M.MulFixedAsPoint(d))
}
