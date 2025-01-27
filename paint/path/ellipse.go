// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package path

import (
	"cogentcore.org/core/math32"
)

func ellipseDeriv(rx, ry, phi float32, sweep bool, theta float32) math32.Vector2 {
	sintheta, costheta := math32.Sincos(theta)
	sinphi, cosphi := math32.Sincos(phi)
	dx := -rx*sintheta*cosphi - ry*costheta*sinphi
	dy := -rx*sintheta*sinphi + ry*costheta*cosphi
	if !sweep {
		return math32.Vector2{-dx, -dy}
	}
	return math32.Vector2{dx, dy}
}

func ellipseDeriv2(rx, ry, phi float32, theta float32) math32.Vector2 {
	sintheta, costheta := math32.Sincos(theta)
	sinphi, cosphi := math32.Sincos(phi)
	ddx := -rx*costheta*cosphi + ry*sintheta*sinphi
	ddy := -rx*costheta*sinphi - ry*sintheta*cosphi
	return math32.Vector2{ddx, ddy}
}

func ellipseCurvatureRadius(rx, ry float32, sweep bool, theta float32) float32 {
	// positive for ccw / sweep
	// phi has no influence on the curvature
	dp := ellipseDeriv(rx, ry, 0.0, sweep, theta)
	ddp := ellipseDeriv2(rx, ry, 0.0, theta)
	a := dp.Cross(ddp)
	if Equal(a, 0.0) {
		return math32.NaN()
	}
	return math32.Pow(dp.X*dp.X+dp.Y*dp.Y, 1.5) / a
}

// ellipseNormal returns the normal to the right at angle theta of the ellipse, given rotation phi.
func ellipseNormal(rx, ry, phi float32, sweep bool, theta, d float32) math32.Vector2 {
	return ellipseDeriv(rx, ry, phi, sweep, theta).Rot90CW().Normal().MulScalar(d)
}

// ellipseLength calculates the length of the elliptical arc
// it uses Gauss-Legendre (n=5) and has an error of ~1% or less (empirical)
func ellipseLength(rx, ry, theta1, theta2 float32) float32 {
	if theta2 < theta1 {
		theta1, theta2 = theta2, theta1
	}
	speed := func(theta float32) float32 {
		return ellipseDeriv(rx, ry, 0.0, true, theta).Length()
	}
	return gaussLegendre5(speed, theta1, theta2)
}

// ellipseToCenter converts to the center arc format and returns
// (centerX, centerY, angleFrom, angleTo) with angles in radians.
// When angleFrom with range [0, 2*PI) is bigger than angleTo with range
// (-2*PI, 4*PI), the ellipse runs clockwise.
// The angles are from before the ellipse has been stretched and rotated.
// See https://www.w3.org/TR/SVG/implnote.html#ArcImplementationNotes
func ellipseToCenter(x1, y1, rx, ry, phi float32, large, sweep bool, x2, y2 float32) (float32, float32, float32, float32) {
	if Equal(x1, x2) && Equal(y1, y2) {
		return x1, y1, 0.0, 0.0
	} else if Equal(math32.Abs(x2-x1), rx) && Equal(y1, y2) && Equal(phi, 0.0) {
		// common case since circles are defined as two arcs from (+dx,0) to (-dx,0) and back
		cx, cy := x1+(x2-x1)/2.0, y1
		theta := float32(0.0)
		if x1 < x2 {
			theta = math32.Pi
		}
		delta := float32(math32.Pi)
		if !sweep {
			delta = -delta
		}
		return cx, cy, theta, theta + delta
	}

	// compute the half distance between start and end point for the unrotated ellipse
	sinphi, cosphi := math32.Sincos(phi)
	x1p := cosphi*(x1-x2)/2.0 + sinphi*(y1-y2)/2.0
	y1p := -sinphi*(x1-x2)/2.0 + cosphi*(y1-y2)/2.0

	// check that radii are large enough to reduce rounding errors
	radiiCheck := x1p*x1p/rx/rx + y1p*y1p/ry/ry
	if 1.0 < radiiCheck {
		radiiScale := math32.Sqrt(radiiCheck)
		rx *= radiiScale
		ry *= radiiScale
	}

	// calculate the center point (cx,cy)
	sq := (rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p) / (rx*rx*y1p*y1p + ry*ry*x1p*x1p)
	if sq <= Epsilon {
		// Epsilon instead of 0.0 improves numerical stability for coef near zero
		// this happens when start and end points are at two opposites of the ellipse and
		// the line between them passes through the center, a common case
		sq = 0.0
	}
	coef := math32.Sqrt(sq)
	if large == sweep {
		coef = -coef
	}
	cxp := coef * rx * y1p / ry
	cyp := coef * -ry * x1p / rx
	cx := cosphi*cxp - sinphi*cyp + (x1+x2)/2.0
	cy := sinphi*cxp + cosphi*cyp + (y1+y2)/2.0

	// specify U and V vectors; theta = arccos(U*V / sqrt(U*U + V*V))
	ux := (x1p - cxp) / rx
	uy := (y1p - cyp) / ry
	vx := -(x1p + cxp) / rx
	vy := -(y1p + cyp) / ry

	// calculate the start angle (theta) and extent angle (delta)
	theta := math32.Acos(ux / math32.Sqrt(ux*ux+uy*uy))
	if uy < 0.0 {
		theta = -theta
	}
	theta = angleNorm(theta)

	deltaAcos := (ux*vx + uy*vy) / math32.Sqrt((ux*ux+uy*uy)*(vx*vx+vy*vy))
	deltaAcos = math32.Min(1.0, math32.Max(-1.0, deltaAcos))
	delta := math32.Acos(deltaAcos)
	if ux*vy-uy*vx < 0.0 {
		delta = -delta
	}
	if !sweep && 0.0 < delta { // clockwise in Cartesian
		delta -= 2.0 * math32.Pi
	} else if sweep && delta < 0.0 { // counter clockwise in Cartesian
		delta += 2.0 * math32.Pi
	}
	return cx, cy, theta, theta + delta
}

// scale ellipse if rx and ry are too small, see https://www.w3.org/TR/SVG/implnote.html#ArcCorrectionOutOfRangeRadii
func ellipseRadiiCorrection(start math32.Vector2, rx, ry, phi float32, end math32.Vector2) float32 {
	diff := start.Sub(end)
	sinphi, cosphi := math32.Sincos(phi)
	x1p := (cosphi*diff.X + sinphi*diff.Y) / 2.0
	y1p := (-sinphi*diff.X + cosphi*diff.Y) / 2.0
	return math32.Sqrt(x1p*x1p/rx/rx + y1p*y1p/ry/ry)
}

// ellipseSplit returns the new mid point, the two large parameters and the ok bool, the rest stays the same
func ellipseSplit(rx, ry, phi, cx, cy, theta0, theta1, theta float32) (math32.Vector2, bool, bool, bool) {
	if !angleBetween(theta, theta0, theta1) {
		return math32.Vector2{}, false, false, false
	}

	mid := EllipsePos(rx, ry, phi, cx, cy, theta)
	large0, large1 := false, false
	if math32.Abs(theta-theta0) > math32.Pi {
		large0 = true
	} else if math32.Abs(theta-theta1) > math32.Pi {
		large1 = true
	}
	return mid, large0, large1, true
}

// see Drawing and elliptical arc using polylines, quadratic or cubic Bézier curves (2003), L. Maisonobe, https://spaceroots.org/documents/ellipse/elliptical-arc.pdf
func ellipseToCubicBeziers(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2) [][4]math32.Vector2 {
	cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

	dtheta := float32(math32.Pi / 2.0) // TODO: use error measure to determine dtheta?
	n := int(math32.Ceil(math32.Abs(theta1-theta0) / dtheta))
	dtheta = math32.Abs(theta1-theta0) / float32(n) // evenly spread the n points, dalpha will get smaller
	kappa := math32.Sin(dtheta) * (math32.Sqrt(4.0+3.0*math32.Pow(math32.Tan(dtheta/2.0), 2.0)) - 1.0) / 3.0
	if !sweep {
		dtheta = -dtheta
	}

	beziers := [][4]math32.Vector2{}
	startDeriv := ellipseDeriv(rx, ry, phi, sweep, theta0)
	for i := 1; i < n+1; i++ {
		theta := theta0 + float32(i)*dtheta
		end := EllipsePos(rx, ry, phi, cx, cy, theta)
		endDeriv := ellipseDeriv(rx, ry, phi, sweep, theta)

		cp1 := start.Add(startDeriv.MulScalar(kappa))
		cp2 := end.Sub(endDeriv.MulScalar(kappa))
		beziers = append(beziers, [4]math32.Vector2{start, cp1, cp2, end})

		startDeriv = endDeriv
		start = end
	}
	return beziers
}

// see Drawing and elliptical arc using polylines, quadratic or cubic Bézier curves (2003), L. Maisonobe, https://spaceroots.org/documents/ellipse/elliptical-arc.pdf
func ellipseToQuadraticBeziers(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2) [][3]math32.Vector2 {
	cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

	dtheta := float32(math32.Pi / 2.0) // TODO: use error measure to determine dtheta?
	n := int(math32.Ceil(math32.Abs(theta1-theta0) / dtheta))
	dtheta = math32.Abs(theta1-theta0) / float32(n) // evenly spread the n points, dalpha will get smaller
	kappa := math32.Tan(dtheta / 2.0)
	if !sweep {
		dtheta = -dtheta
	}

	beziers := [][3]math32.Vector2{}
	startDeriv := ellipseDeriv(rx, ry, phi, sweep, theta0)
	for i := 1; i < n+1; i++ {
		theta := theta0 + float32(i)*dtheta
		end := EllipsePos(rx, ry, phi, cx, cy, theta)
		endDeriv := ellipseDeriv(rx, ry, phi, sweep, theta)

		cp := start.Add(startDeriv.MulScalar(kappa))
		beziers = append(beziers, [3]math32.Vector2{start, cp, end})

		startDeriv = endDeriv
		start = end
	}
	return beziers
}

func xmonotoneEllipticArc(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2) Path {
	sign := float32(1.0)
	if !sweep {
		sign = -1.0
	}

	cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
	sinphi, cosphi := math32.Sincos(phi)
	thetaRight := math32.Atan2(-ry*sinphi, rx*cosphi)
	thetaLeft := thetaRight + math32.Pi

	p := Path{}
	p.MoveTo(start.X, start.Y)
	left := !angleEqual(thetaLeft, theta0) && angleNorm(sign*(thetaLeft-theta0)) < angleNorm(sign*(thetaRight-theta0))
	for t := theta0; !angleEqual(t, theta1); {
		dt := angleNorm(sign * (theta1 - t))
		if left {
			dt = math32.Min(dt, angleNorm(sign*(thetaLeft-t)))
		} else {
			dt = math32.Min(dt, angleNorm(sign*(thetaRight-t)))
		}
		t += sign * dt

		pos := EllipsePos(rx, ry, phi, cx, cy, t)
		p.ArcTo(rx, ry, phi, false, sweep, pos.X, pos.Y)
		left = !left
	}
	return p
}

func FlattenEllipticArc(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2, tolerance float32) Path {
	if Equal(rx, ry) {
		// circle
		r := rx
		cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
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

		p := Path{}
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
	return arcToCube(start, rx, ry, phi, large, sweep, end).Flatten(tolerance)
}
