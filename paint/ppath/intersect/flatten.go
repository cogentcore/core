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

// Flatten flattens all Bézier and arc curves into linear segments
// and returns a new path. It uses tolerance as the maximum deviation.
func Flatten(p ppath.Path, tolerance float32) ppath.Path {
	quad := func(p0, p1, p2 math32.Vector2) ppath.Path {
		return FlattenQuadraticBezier(p0, p1, p2, tolerance)
	}
	cube := func(p0, p1, p2, p3 math32.Vector2) ppath.Path {
		return FlattenCubicBezier(p0, p1, p2, p3, 0.0, tolerance)
	}
	arc := func(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2) ppath.Path {
		return FlattenEllipticArc(start, rx, ry, phi, large, sweep, end, tolerance)
	}
	return p.Replace(nil, quad, cube, arc)
}

func FlattenEllipticArc(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2, tolerance float32) ppath.Path {
	if ppath.Equal(rx, ry) {
		// circle
		r := rx
		cx, cy, theta0, theta1 := ppath.EllipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
		theta0 += phi
		theta1 += phi

		// draw line segments from arc+tolerance to arc+tolerance, touching arc-tolerance in between
		// we start and end at the arc itself
		dtheta := math32.Abs(theta1 - theta0)
		thetaEnd := math32.Acos((r - tolerance) / r)               // half angle of first/last segment
		thetaMid := math32.Acos((r - tolerance) / (r + tolerance)) // half angle of middle segments
		n := math32.Ceil((dtheta - thetaEnd*2.0) / (thetaMid * 2.0))

		// evenly space out points along arc
		ratio := dtheta / (thetaEnd*2.0 + thetaMid*2.0*n)
		thetaEnd *= ratio
		thetaMid *= ratio

		// adjust distance from arc to lower total deviation area, add points on the outer circle
		// of the tolerance since the middle of the line segment touches the inner circle and thus
		// even out. Ratio < 1 is when the line segments are shorter (and thus not touch the inner
		// tolerance circle).
		r += ratio * tolerance

		p := ppath.Path{}
		p.MoveTo(start.X, start.Y)
		theta := thetaEnd + thetaMid
		for i := 0; i < int(n); i++ {
			t := theta0 + math32.Copysign(theta, theta1-theta0)
			pos := math32.Vector2Polar(t, r).Add(math32.Vector2{cx, cy})
			p.LineTo(pos.X, pos.Y)
			theta += 2.0 * thetaMid
		}
		p.LineTo(end.X, end.Y)
		return p
	}
	// TODO: (flatten ellipse) use direct algorithm
	return Flatten(ppath.ArcToCube(start, rx, ry, phi, large, sweep, end), tolerance)
}

func FlattenQuadraticBezier(p0, p1, p2 math32.Vector2, tolerance float32) ppath.Path {
	// see Flat, precise flattening of cubic Bézier path and offset curves, by T.F. Hain et al., 2005,  https://www.sciencedirect.com/science/article/pii/S0097849305001287
	t := float32(0.0)
	p := ppath.Path{}
	p.MoveTo(p0.X, p0.Y)
	for t < 1.0 {
		D := p1.Sub(p0)
		if ppath.EqualPoint(p0, p1) {
			// p0 == p1, curve is a straight line from p0 to p2
			// should not occur directly from paths as this is prevented in QuadTo, but may appear in other subroutines
			break
		}
		denom := math32.Hypot(D.X, D.Y) // equal to r1
		s2nom := D.Cross(p2.Sub(p0))
		//effFlatness := tolerance / (1.0 - d*s2nom/(denom*denom*denom)/2.0)
		t = 2.0 * math32.Sqrt(tolerance*math32.Abs(denom/s2nom))
		if t >= 1.0 {
			break
		}

		_, _, _, p0, p1, p2 = quadraticBezierSplit(p0, p1, p2, t)
		p.LineTo(p0.X, p0.Y)
	}
	p.LineTo(p2.X, p2.Y)
	return p
}

// see Flat, precise flattening of cubic Bézier path and offset curves, by T.F. Hain et al., 2005,  https://www.sciencedirect.com/science/article/pii/S0097849305001287
// see https://github.com/Manishearth/stylo-flat/blob/master/gfx/2d/Path.cpp for an example implementation
// or https://docs.rs/crate/lyon_bezier/0.4.1/source/src/flatten_cubic.rs
// p0, p1, p2, p3 are the start points, two control points and the end points respectively. With flatness defined as the maximum error from the orinal curve, and d the half width of the curve used for stroking (positive is to the right).
func FlattenCubicBezier(p0, p1, p2, p3 math32.Vector2, d, tolerance float32) ppath.Path {
	tolerance = math32.Max(tolerance, ppath.Epsilon) // prevent infinite loop if user sets tolerance to zero

	p := ppath.Path{}
	start := p0.Add(CubicBezierNormal(p0, p1, p2, p3, 0.0, d))
	p.MoveTo(start.X, start.Y)

	// 0 <= t1 <= 1 if t1 exists
	// 0 <= t1 <= t2 <= 1 if t1 and t2 both exist
	t1, t2 := findInflectionPointCubicBezier(p0, p1, p2, p3)
	if math32.IsNaN(t1) && math32.IsNaN(t2) {
		// There are no inflection points or cusps, approximate linearly by subdivision.
		FlattenSmoothCubicBezier(&p, p0, p1, p2, p3, d, tolerance)
		return p
	}

	// t1min <= t1max; with 0 <= t1max and t1min <= 1
	// t2min <= t2max; with 0 <= t2max and t2min <= 1
	t1min, t1max := findInflectionPointRangeCubicBezier(p0, p1, p2, p3, t1, tolerance)
	t2min, t2max := findInflectionPointRangeCubicBezier(p0, p1, p2, p3, t2, tolerance)

	if math32.IsNaN(t2) && t1min <= 0.0 && 1.0 <= t1max {
		// There is no second inflection point, and the first inflection point can be entirely approximated linearly.
		addCubicBezierLine(&p, p0, p1, p2, p3, 1.0, d)
		return p
	}

	if 0.0 < t1min {
		// Flatten up to t1min
		q0, q1, q2, q3, _, _, _, _ := cubicBezierSplit(p0, p1, p2, p3, t1min)
		FlattenSmoothCubicBezier(&p, q0, q1, q2, q3, d, tolerance)
	}

	if 0.0 < t1max && t1max < 1.0 && t1max < t2min {
		// t1 and t2 ranges do not overlap, approximate t1 linearly
		_, _, _, _, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, t1max)
		addCubicBezierLine(&p, q0, q1, q2, q3, 0.0, d)
		if 1.0 <= t2min {
			// No t2 present, approximate the rest linearly by subdivision
			FlattenSmoothCubicBezier(&p, q0, q1, q2, q3, d, tolerance)
			return p
		}
	} else if 1.0 <= t2min {
		// No t2 present and t1max is past the end of the curve, approximate linearly
		addCubicBezierLine(&p, p0, p1, p2, p3, 1.0, d)
		return p
	}

	// t1 and t2 exist and ranges might overlap
	if 0.0 < t2min {
		if t2min < t1max {
			// t2 range starts inside t1 range, approximate t1 range linearly
			_, _, _, _, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, t1max)
			addCubicBezierLine(&p, q0, q1, q2, q3, 0.0, d)
		} else {
			// no overlap
			_, _, _, _, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, t1max)
			t2minq := (t2min - t1max) / (1 - t1max)
			q0, q1, q2, q3, _, _, _, _ = cubicBezierSplit(q0, q1, q2, q3, t2minq)
			FlattenSmoothCubicBezier(&p, q0, q1, q2, q3, d, tolerance)
		}
	}

	// handle (the rest of) t2
	if t2max < 1.0 {
		_, _, _, _, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, t2max)
		addCubicBezierLine(&p, q0, q1, q2, q3, 0.0, d)
		FlattenSmoothCubicBezier(&p, q0, q1, q2, q3, d, tolerance)
	} else {
		// t2max extends beyond 1
		addCubicBezierLine(&p, p0, p1, p2, p3, 1.0, d)
	}
	return p
}

// split the curve and replace it by lines as long as (maximum deviation <= tolerance) is maintained
func FlattenSmoothCubicBezier(p *ppath.Path, p0, p1, p2, p3 math32.Vector2, d, tolerance float32) {
	t := float32(0.0)
	for t < 1.0 {
		D := p1.Sub(p0)
		if ppath.EqualPoint(p0, p1) {
			// p0 == p1, base on p2
			D = p2.Sub(p0)
			if ppath.EqualPoint(p0, p2) {
				// p0 == p1 == p2, curve is a straight line from p0 to p3
				p.LineTo(p3.X, p3.Y)
				return
			}
		}
		denom := D.Length() // equal to r1

		// effective flatness distorts the stroke width as both sides have different cuts
		//effFlatness := flatness / (1.0 - d*s2nom/(denom*denom*denom)*2.0/3.0)
		s2nom := D.Cross(p2.Sub(p0))
		s2inv := denom / s2nom
		t2 := 2.0 * math32.Sqrt(tolerance*math32.Abs(s2inv)/3.0)

		// if s2 is small, s3 may represent the curvature more accurately
		// we cannot calculate the effective flatness here
		s3nom := D.Cross(p3.Sub(p0))
		s3inv := denom / s3nom
		t3 := 2.0 * math32.Cbrt(tolerance*math32.Abs(s3inv))

		// choose whichever is most curved, P2-P0 or P3-P0
		t = math32.Min(t2, t3)
		if 1.0 <= t {
			break
		}
		_, _, _, _, p0, p1, p2, p3 = cubicBezierSplit(p0, p1, p2, p3, t)
		addCubicBezierLine(p, p0, p1, p2, p3, 0.0, d)
	}
	addCubicBezierLine(p, p0, p1, p2, p3, 1.0, d)
}
