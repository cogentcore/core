// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package intersect

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
)

func cubicBezierDeriv2(p0, p1, p2, p3 math32.Vector2, t float32) math32.Vector2 {
	p0 = p0.MulScalar(6.0 - 6.0*t)
	p1 = p1.MulScalar(18.0*t - 12.0)
	p2 = p2.MulScalar(6.0 - 18.0*t)
	p3 = p3.MulScalar(6.0 * t)
	return p0.Add(p1).Add(p2).Add(p3)
}

func cubicBezierDeriv3(p0, p1, p2, p3 math32.Vector2, t float32) math32.Vector2 {
	p0 = p0.MulScalar(-6.0)
	p1 = p1.MulScalar(18.0)
	p2 = p2.MulScalar(-18.0)
	p3 = p3.MulScalar(6.0)
	return p0.Add(p1).Add(p2).Add(p3)
}

func cubicBezierPos(p0, p1, p2, p3 math32.Vector2, t float32) math32.Vector2 {
	p0 = p0.MulScalar(1.0 - 3.0*t + 3.0*t*t - t*t*t)
	p1 = p1.MulScalar(3.0*t - 6.0*t*t + 3.0*t*t*t)
	p2 = p2.MulScalar(3.0*t*t - 3.0*t*t*t)
	p3 = p3.MulScalar(t * t * t)
	return p0.Add(p1).Add(p2).Add(p3)
}

func quadraticBezierDeriv2(p0, p1, p2 math32.Vector2) math32.Vector2 {
	p0 = p0.MulScalar(2.0)
	p1 = p1.MulScalar(-4.0)
	p2 = p2.MulScalar(2.0)
	return p0.Add(p1).Add(p2)
}

func quadraticBezierPos(p0, p1, p2 math32.Vector2, t float32) math32.Vector2 {
	p0 = p0.MulScalar(1.0 - 2.0*t + t*t)
	p1 = p1.MulScalar(2.0*t - 2.0*t*t)
	p2 = p2.MulScalar(t * t)
	return p0.Add(p1).Add(p2)
}

// negative when curve bends CW while following t
func CubicBezierCurvatureRadius(p0, p1, p2, p3 math32.Vector2, t float32) float32 {
	dp := ppath.CubicBezierDeriv(p0, p1, p2, p3, t)
	ddp := cubicBezierDeriv2(p0, p1, p2, p3, t)
	a := dp.Cross(ddp) // negative when bending right ie. curve is CW at this point
	if ppath.Equal(a, 0.0) {
		return math32.NaN()
	}
	return math32.Pow(dp.X*dp.X+dp.Y*dp.Y, 1.5) / a
}

// negative when curve bends CW while following t
func quadraticBezierCurvatureRadius(p0, p1, p2 math32.Vector2, t float32) float32 {
	dp := ppath.QuadraticBezierDeriv(p0, p1, p2, t)
	ddp := quadraticBezierDeriv2(p0, p1, p2)
	a := dp.Cross(ddp) // negative when bending right ie. curve is CW at this point
	if ppath.Equal(a, 0.0) {
		return math32.NaN()
	}
	return math32.Pow(dp.X*dp.X+dp.Y*dp.Y, 1.5) / a
}

// see https://malczak.linuxpl.com/blog/quadratic-bezier-curve-length/
func quadraticBezierLength(p0, p1, p2 math32.Vector2) float32 {
	a := p0.Sub(p1.MulScalar(2.0)).Add(p2)
	b := p1.MulScalar(2.0).Sub(p0.MulScalar(2.0))
	A := 4.0 * a.Dot(a)
	B := 4.0 * a.Dot(b)
	C := b.Dot(b)
	if ppath.Equal(A, 0.0) {
		// p1 is in the middle between p0 and p2, so it is a straight line from p0 to p2
		return p2.Sub(p0).Length()
	}

	Sabc := 2.0 * math32.Sqrt(A+B+C)
	A2 := math32.Sqrt(A)
	A32 := 2.0 * A * A2
	C2 := 2.0 * math32.Sqrt(C)
	BA := B / A2
	return (A32*Sabc + A2*B*(Sabc-C2) + (4.0*C*A-B*B)*math32.Log((2.0*A2+BA+Sabc)/(BA+C2))) / (4.0 * A32)
}

func findInflectionPointCubicBezier(p0, p1, p2, p3 math32.Vector2) (float32, float32) {
	// see www.faculty.idc.ac.il/arik/quality/appendixa.html
	// we omit multiplying bx,by,cx,cy with 3.0, so there is no need for divisions when calculating a,b,c
	ax := -p0.X + 3.0*p1.X - 3.0*p2.X + p3.X
	ay := -p0.Y + 3.0*p1.Y - 3.0*p2.Y + p3.Y
	bx := p0.X - 2.0*p1.X + p2.X
	by := p0.Y - 2.0*p1.Y + p2.Y
	cx := -p0.X + p1.X
	cy := -p0.Y + p1.Y

	a := (ay*bx - ax*by)
	b := (ay*cx - ax*cy)
	c := (by*cx - bx*cy)
	x1, x2 := solveQuadraticFormula(a, b, c)
	if x1 < ppath.Epsilon/2.0 || 1.0-ppath.Epsilon/2.0 < x1 {
		x1 = math32.NaN()
	}
	if x2 < ppath.Epsilon/2.0 || 1.0-ppath.Epsilon/2.0 < x2 {
		x2 = math32.NaN()
	} else if math32.IsNaN(x1) {
		x1, x2 = x2, x1
	}
	return x1, x2
}

func findInflectionPointRangeCubicBezier(p0, p1, p2, p3 math32.Vector2, t, tolerance float32) (float32, float32) {
	// find the range around an inflection point that we consider flat within the flatness criterion
	if math32.IsNaN(t) {
		return math32.Inf(1), math32.Inf(1)
	} else if t < 0.0 || t > 1.0 {
		panic("t outside 0.0--1.0 range")
	}

	// we state that s(t) = 3*s2*t^2 + (s3 - 3*s2)*t^3 (see paper on the r-s coordinate system)
	// with s(t) aligned perpendicular to the curve at t = 0
	// then we impose that s(tf) = flatness and find tf
	// at inflection points however, s2 = 0, so that s(t) = s3*t^3

	if !ppath.Equal(t, 0.0) {
		_, _, _, _, p0, p1, p2, p3 = cubicBezierSplit(p0, p1, p2, p3, t)
	}
	nr := p1.Sub(p0)
	ns := p3.Sub(p0)
	if ppath.Equal(nr.X, 0.0) && ppath.Equal(nr.Y, 0.0) {
		// if p0=p1, then rn (the velocity at t=0) needs adjustment
		// nr = lim[t->0](B'(t)) = 3*(p1-p0) + 6*t*((p1-p0)+(p2-p1)) + second order terms of t
		// if (p1-p0)->0, we use (p2-p1)=(p2-p0)
		nr = p2.Sub(p0)
	}

	if ppath.Equal(nr.X, 0.0) && ppath.Equal(nr.Y, 0.0) {
		// if rn is still zero, this curve has p0=p1=p2, so it is straight
		return math32.Max(0.0, 2.0*t-1.0), 1.0
	}

	s3 := math32.Abs(ns.X*nr.Y-ns.Y*nr.X) / math32.Hypot(nr.X, nr.Y)
	if ppath.Equal(s3, 0.0) {
		return math32.Max(0.0, 2.0*t-1.0), 1.0 // can approximate whole curve linearly
	}

	tf := math32.Cbrt(tolerance / s3)
	return t - tf*(1.0-t), t + tf*(1.0-t)
}

// cubicBezierLength calculates the length of the Bézier, taking care of inflection points. It uses Gauss-Legendre (n=5) and has an error of ~1% or less (empirical).
func cubicBezierLength(p0, p1, p2, p3 math32.Vector2) float32 {
	t1, t2 := findInflectionPointCubicBezier(p0, p1, p2, p3)
	var beziers [][4]math32.Vector2
	if t1 > 0.0 && t1 < 1.0 && t2 > 0.0 && t2 < 1.0 {
		p0, p1, p2, p3, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, t1)
		t2 = (t2 - t1) / (1.0 - t1)
		q0, q1, q2, q3, r0, r1, r2, r3 := cubicBezierSplit(q0, q1, q2, q3, t2)
		beziers = append(beziers, [4]math32.Vector2{p0, p1, p2, p3})
		beziers = append(beziers, [4]math32.Vector2{q0, q1, q2, q3})
		beziers = append(beziers, [4]math32.Vector2{r0, r1, r2, r3})
	} else if t1 > 0.0 && t1 < 1.0 {
		p0, p1, p2, p3, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, t1)
		beziers = append(beziers, [4]math32.Vector2{p0, p1, p2, p3})
		beziers = append(beziers, [4]math32.Vector2{q0, q1, q2, q3})
	} else {
		beziers = append(beziers, [4]math32.Vector2{p0, p1, p2, p3})
	}

	length := float32(0.0)
	for _, bezier := range beziers {
		speed := func(t float32) float32 {
			return ppath.CubicBezierDeriv(bezier[0], bezier[1], bezier[2], bezier[3], t).Length()
		}
		length += gaussLegendre7(speed, 0.0, 1.0)
	}
	return length
}

func quadraticBezierSplit(p0, p1, p2 math32.Vector2, t float32) (math32.Vector2, math32.Vector2, math32.Vector2, math32.Vector2, math32.Vector2, math32.Vector2) {
	q0 := p0
	q1 := p0.Lerp(p1, t)

	r2 := p2
	r1 := p1.Lerp(p2, t)

	r0 := q1.Lerp(r1, t)
	q2 := r0
	return q0, q1, q2, r0, r1, r2
}

func cubicBezierSplit(p0, p1, p2, p3 math32.Vector2, t float32) (math32.Vector2, math32.Vector2, math32.Vector2, math32.Vector2, math32.Vector2, math32.Vector2, math32.Vector2, math32.Vector2) {
	pm := p1.Lerp(p2, t)

	q0 := p0
	q1 := p0.Lerp(p1, t)
	q2 := q1.Lerp(pm, t)

	r3 := p3
	r2 := p2.Lerp(p3, t)
	r1 := pm.Lerp(r2, t)

	r0 := q2.Lerp(r1, t)
	q3 := r0
	return q0, q1, q2, q3, r0, r1, r2, r3
}

func cubicBezierNumInflections(p0, p1, p2, p3 math32.Vector2) int {
	t1, t2 := findInflectionPointCubicBezier(p0, p1, p2, p3)
	if !math32.IsNaN(t2) {
		return 2
	} else if !math32.IsNaN(t1) {
		return 1
	}
	return 0
}

func xmonotoneCubicBezier(p0, p1, p2, p3 math32.Vector2) ppath.Path {
	a := -p0.X + 3*p1.X - 3*p2.X + p3.X
	b := 2*p0.X - 4*p1.X + 2*p2.X
	c := -p0.X + p1.X

	p := ppath.Path{}
	p.MoveTo(p0.X, p0.Y)

	split := false
	t1, t2 := solveQuadraticFormula(a, b, c)
	if !math32.IsNaN(t1) && inIntervalExclusive(t1, 0.0, 1.0) {
		_, q1, q2, q3, r0, r1, r2, r3 := cubicBezierSplit(p0, p1, p2, p3, t1)
		p.CubeTo(q1.X, q1.Y, q2.X, q2.Y, q3.X, q3.Y)
		p0, p1, p2, p3 = r0, r1, r2, r3
		split = true
	}
	if !math32.IsNaN(t2) && inIntervalExclusive(t2, 0.0, 1.0) {
		if split {
			t2 = (t2 - t1) / (1.0 - t1)
		}
		_, q1, q2, q3, _, r1, r2, r3 := cubicBezierSplit(p0, p1, p2, p3, t2)
		p.CubeTo(q1.X, q1.Y, q2.X, q2.Y, q3.X, q3.Y)
		p1, p2, p3 = r1, r2, r3
	}
	p.CubeTo(p1.X, p1.Y, p2.X, p2.Y, p3.X, p3.Y)
	return p
}

func quadraticBezierDistance(p0, p1, p2, q math32.Vector2) float32 {
	f := p0.Sub(p1.MulScalar(2.0)).Add(p2)
	g := p1.MulScalar(2.0).Sub(p0.MulScalar(2.0))
	h := p0.Sub(q)

	a := 4.0 * (f.X*f.X + f.Y*f.Y)
	b := 6.0 * (f.X*g.X + f.Y*g.Y)
	c := 2.0 * (2.0*(f.X*h.X+f.Y*h.Y) + g.X*g.X + g.Y*g.Y)
	d := 2.0 * (g.X*h.X + g.Y*h.Y)

	dist := math32.Inf(1.0)
	t0, t1, t2 := solveCubicFormula(a, b, c, d)
	ts := []float32{t0, t1, t2, 0.0, 1.0}
	for _, t := range ts {
		if !math32.IsNaN(t) {
			if t < 0.0 {
				t = 0.0
			} else if 1.0 < t {
				t = 1.0
			}
			if tmpDist := quadraticBezierPos(p0, p1, p2, t).Sub(q).Length(); tmpDist < dist {
				dist = tmpDist
			}
		}
	}
	return dist
}

func xmonotoneQuadraticBezier(p0, p1, p2 math32.Vector2) ppath.Path {
	p := ppath.Path{}
	p.MoveTo(p0.X, p0.Y)
	if tdenom := (p0.X - 2*p1.X + p2.X); !ppath.Equal(tdenom, 0.0) {
		if t := (p0.X - p1.X) / tdenom; 0.0 < t && t < 1.0 {
			_, q1, q2, _, r1, r2 := quadraticBezierSplit(p0, p1, p2, t)
			p.QuadTo(q1.X, q1.Y, q2.X, q2.Y)
			p1, p2 = r1, r2
		}
	}
	p.QuadTo(p1.X, p1.Y, p2.X, p2.Y)
	return p
}

// return the normal at the right-side of the curve (when increasing t)
func CubicBezierNormal(p0, p1, p2, p3 math32.Vector2, t, d float32) math32.Vector2 {
	// TODO: remove and use CubicBezierDeriv + Rot90CW?
	if t == 0.0 {
		n := p1.Sub(p0)
		if n.X == 0 && n.Y == 0 {
			n = p2.Sub(p0)
		}
		if n.X == 0 && n.Y == 0 {
			n = p3.Sub(p0)
		}
		if n.X == 0 && n.Y == 0 {
			return math32.Vector2{}
		}
		return n.Rot90CW().Normal().MulScalar(d)
	} else if t == 1.0 {
		n := p3.Sub(p2)
		if n.X == 0 && n.Y == 0 {
			n = p3.Sub(p1)
		}
		if n.X == 0 && n.Y == 0 {
			n = p3.Sub(p0)
		}
		if n.X == 0 && n.Y == 0 {
			return math32.Vector2{}
		}
		return n.Rot90CW().Normal().MulScalar(d)
	}
	panic("not implemented") // not needed
}

func addCubicBezierLine(p *ppath.Path, p0, p1, p2, p3 math32.Vector2, t, d float32) {
	if ppath.EqualPoint(p0, p3) && (ppath.EqualPoint(p0, p1) || ppath.EqualPoint(p0, p2)) {
		// Bézier has p0=p1=p3 or p0=p2=p3 and thus has no surface or length
		return
	}

	pos := math32.Vector2{}
	if t == 0.0 {
		// line to beginning of path
		pos = p0
		if d != 0.0 {
			n := CubicBezierNormal(p0, p1, p2, p3, t, d)
			pos = pos.Add(n)
		}
	} else if t == 1.0 {
		// line to the end of the path
		pos = p3
		if d != 0.0 {
			n := CubicBezierNormal(p0, p1, p2, p3, t, d)
			pos = pos.Add(n)
		}
	} else {
		panic("not implemented")
	}
	p.LineTo(pos.X, pos.Y)
}
