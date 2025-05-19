// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package ppath

import (
	"cogentcore.org/core/math32"
)

func EllipseDeriv(rx, ry, phi float32, sweep bool, theta float32) math32.Vector2 {
	sintheta, costheta := math32.Sincos(theta)
	sinphi, cosphi := math32.Sincos(phi)
	dx := -rx*sintheta*cosphi - ry*costheta*sinphi
	dy := -rx*sintheta*sinphi + ry*costheta*cosphi
	if !sweep {
		return math32.Vector2{-dx, -dy}
	}
	return math32.Vector2{dx, dy}
}

// EllipsePos adds the position on the ellipse at angle theta.
func EllipsePos(rx, ry, phi, cx, cy, theta float32) math32.Vector2 {
	sintheta, costheta := math32.Sincos(theta)
	sinphi, cosphi := math32.Sincos(phi)
	x := cx + rx*costheta*cosphi - ry*sintheta*sinphi
	y := cy + rx*costheta*sinphi + ry*sintheta*cosphi
	return math32.Vector2{x, y}
}

// EllipseToCenter converts to the center arc format and returns
// (centerX, centerY, angleFrom, angleTo) with angles in radians.
// When angleFrom with range [0, 2*PI) is bigger than angleTo with range
// (-2*PI, 4*PI), the ellipse runs clockwise.
// The angles are from before the ellipse has been stretched and rotated.
// See https://www.w3.org/TR/SVG/implnote.html#ArcImplementationNotes
func EllipseToCenter(x1, y1, rx, ry, phi float32, large, sweep bool, x2, y2 float32) (float32, float32, float32, float32) {
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
	theta = AngleNorm(theta)

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
func EllipseRadiiCorrection(start math32.Vector2, rx, ry, phi float32, end math32.Vector2) float32 {
	diff := start.Sub(end)
	sinphi, cosphi := math32.Sincos(phi)
	x1p := (cosphi*diff.X + sinphi*diff.Y) / 2.0
	y1p := (-sinphi*diff.X + cosphi*diff.Y) / 2.0
	return math32.Sqrt(x1p*x1p/rx/rx + y1p*y1p/ry/ry)
}

// see Drawing and elliptical arc using polylines, quadratic or cubic Bézier curves (2003), L. Maisonobe, https://spaceroots.org/documents/ellipse/elliptical-arc.pdf
func ellipseToCubicBeziers(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2) [][4]math32.Vector2 {
	cx, cy, theta0, theta1 := EllipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

	dtheta := float32(math32.Pi / 2.0) // TODO: use error measure to determine dtheta?
	n := int(math32.Ceil(math32.Abs(theta1-theta0) / dtheta))
	dtheta = math32.Abs(theta1-theta0) / float32(n) // evenly spread the n points, dalpha will get smaller
	kappa := math32.Sin(dtheta) * (math32.Sqrt(4.0+3.0*math32.Pow(math32.Tan(dtheta/2.0), 2.0)) - 1.0) / 3.0
	if !sweep {
		dtheta = -dtheta
	}

	beziers := [][4]math32.Vector2{}
	startDeriv := EllipseDeriv(rx, ry, phi, sweep, theta0)
	for i := 1; i < n+1; i++ {
		theta := theta0 + float32(i)*dtheta
		end := EllipsePos(rx, ry, phi, cx, cy, theta)
		endDeriv := EllipseDeriv(rx, ry, phi, sweep, theta)

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
	cx, cy, theta0, theta1 := EllipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

	dtheta := float32(math32.Pi / 2.0) // TODO: use error measure to determine dtheta?
	n := int(math32.Ceil(math32.Abs(theta1-theta0) / dtheta))
	dtheta = math32.Abs(theta1-theta0) / float32(n) // evenly spread the n points, dalpha will get smaller
	kappa := math32.Tan(dtheta / 2.0)
	if !sweep {
		dtheta = -dtheta
	}

	beziers := [][3]math32.Vector2{}
	startDeriv := EllipseDeriv(rx, ry, phi, sweep, theta0)
	for i := 1; i < n+1; i++ {
		theta := theta0 + float32(i)*dtheta
		end := EllipsePos(rx, ry, phi, cx, cy, theta)
		endDeriv := EllipseDeriv(rx, ry, phi, sweep, theta)

		cp := start.Add(startDeriv.MulScalar(kappa))
		beziers = append(beziers, [3]math32.Vector2{start, cp, end})

		startDeriv = endDeriv
		start = end
	}
	return beziers
}
